package biu

import (
	"fmt"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/emicklei/go-restful"
	"github.com/gin-gonic/gin/binding"
	"github.com/json-iterator/go"
	"github.com/mpvl/errc"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

var anonymousFuncCount int32

// nameOfFunction returns the short name of the function f for documentation.
// It uses a runtime feature for debugging ; its value may change for later Go versions.
func nameOfFunction(f interface{}) string {
	fun := runtime.FuncForPC(reflect.ValueOf(f).Pointer())
	tokenized := strings.Split(fun.Name(), ".")
	last := tokenized[len(tokenized)-1]
	last = strings.TrimSuffix(last, ")·fm") // < Go 1.5
	last = strings.TrimSuffix(last, ")-fm") // Go 1.5
	last = strings.TrimSuffix(last, "·fm")  // < Go 1.5
	last = strings.TrimSuffix(last, "-fm")  // Go 1.5
	if last == "func1" {                    // this could mean conflicts in API docs
		val := atomic.AddInt32(&anonymousFuncCount, 1)
		last = "func" + fmt.Sprintf("%d", val)
		atomic.StoreInt32(&anonymousFuncCount, val)
	}
	return last
}

// DisableErrHandler disables error handler using errc.
var DisableErrHandler bool

// Handle transform a biu handler to a restful.RouteFunction.
func Handle(f func(ctx Ctx)) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		ctx := Ctx{
			Request:  request,
			Response: response,
		}
		if !DisableErrHandler {
			e := errc.Catch(new(error))
			defer e.Handle()
			ctx.ErrCatcher = e
		}
		f(ctx)
	}
}

// Filter transform a biu handler to a restful.FilterFunction
func Filter(f func(ctx Ctx)) restful.FilterFunction {
	return func(request *restful.Request, response *restful.Response,
		chain *restful.FilterChain) {
		f(Ctx{
			Request:     request,
			Response:    response,
			FilterChain: chain,
		})
	}
}

// WrapHandler wraps a biu handler to http.HandlerFunc
func WrapHandler(f func(ctx Ctx)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(Ctx{
			Request:  restful.NewRequest(r),
			Response: restful.NewResponse(w),
		})
	}
}

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

// Ctx wrap *restful.Request and *restful.Response in one struct.
type Ctx struct {
	*restful.Request
	*restful.Response
	*restful.FilterChain
	ErrCatcher errc.Catcher
}

// ResponseJSON is a convenience method
// for writing a value wrap in CommonResp as JSON.
// It uses jsoniter for marshalling the value.
func (ctx *Ctx) ResponseJSON(v interface{}) {
	CommonResponse(ctx.Response, ctx.RouteID(), 0, "", v)
}

// ResponseError is a convenience method to response an error code and message.
// It uses jsoniter for marshalling the value.
func (ctx *Ctx) ResponseError(msg string, code int) {
	CommonResponse(ctx.Response, ctx.RouteID(), code, msg, nil)
}

// RouteID returns the RouteID of current route.
func (ctx *Ctx) RouteID() string {
	return routeIDMap[ctx.RouteSignature()]
}

// RouteSignature returns the signature of current route.
// Example: /v1/user/login POST
func (ctx *Ctx) RouteSignature() string {
	return ctx.SelectedRoutePath() + " " + ctx.Request.Request.Method
}

// ErrMsg returns the message of a error code in current route.
func (ctx *Ctx) ErrMsg(code int) string {
	msg, ok := routeErrMap[ctx.RouteSignature()][code]
	if ok {
		return msg
	}
	return globalErrMap[code]
}

// Redirect replies to the request with a redirect to url.
func (ctx *Ctx) Redirect(url string, code int) {
	http.Redirect(ctx.ResponseWriter, ctx.Request.Request, url, code)
}

// ContainsError is a convenience method to check error is nil.
// If error is nil, it will return false,
// else it will log the error, make a CommonResp response and return true.
// if code is 0, it will use err.Error() as CommonResp.message.
func (ctx *Ctx) ContainsError(err error, code int, v ...interface{}) bool {
	msg := ctx.ErrMsg(code)
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	if CheckError(err, Info().
		Str("routeID", ctx.RouteID()).
		Str("routeSig", ctx.RouteSignature()).
		Int("code", code).
		Str("msg", msg)) {
		return false
	}
	if code == 0 {
		msg = err.Error()
	}
	ResponseError(ctx.Response, ctx.RouteID(), msg, code)
	return true
}

type errHandler struct {
	ctx  *Ctx
	code int
	v    []interface{}
}

// Handle implements errc.Handle
func (e errHandler) Handle(s errc.State, err error) error {
	msg := e.ctx.ErrMsg(e.code)
	vLen := len(e.v)
	if vLen > 0 {
		if f, ok := e.v[vLen-1].(func()); ok {
			f()
			e.v = e.v[:vLen-1]
		}
		msg = fmt.Sprintf(msg, e.v...)
	}
	Info().
		Str("routeID", e.ctx.RouteID()).
		Str("routeSig", e.ctx.RouteSignature()).
		Int("code", e.code).
		Str("msg", msg).Str(zerolog.ErrorFieldName,
		fmt.Sprintf("%+v\n", errors.WithStack(err))).
		Msg("verify error")
	if e.code == 0 {
		msg = err.Error()
	}
	ResponseError(e.ctx.Response, e.ctx.RouteID(), msg, e.code)
	return err
}

// Must causes a return from a function if err is not nil.
func (ctx *Ctx) Must(err error, code int, v ...interface{}) {
	if !DisableErrHandler {
		ctx.ErrCatcher.Must(err, errHandler{ctx: ctx, code: code, v: v})
	}
}

// ResponseStdErrCode is a convenience method response a code
// with msg in Code Desc.
func (ctx *Ctx) ResponseStdErrCode(code int, v ...interface{}) {
	msg := ctx.ErrMsg(code)
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	ResponseError(ctx.Response, ctx.RouteID(), msg, code)
}

// UserID returns UserID stored in attribute.
func (ctx *Ctx) UserID() string {
	userID, ok := ctx.Attribute("UserID").(string)
	if !ok {
		return ""
	}
	return userID
}

// IP returns the IP address of request.
func (ctx *Ctx) IP() string {
	req := ctx.Request.Request
	ra := req.RemoteAddr
	if ip := ctx.HeaderParameter("X-Forwarded-For"); ip != "" {
		ra = strings.Split(ip, ", ")[0]
	} else if ip := ctx.HeaderParameter("X-Real-IP"); ip != "" {
		ra = ip
	} else {
		ra, _, _ = net.SplitHostPort(ra)
	}
	return ra
}

// Proxy returns the proxy endpoints behind a request.
func (ctx *Ctx) Proxy() []string {
	if ipArr := ctx.HeaderParameter("X-Forwarded-For"); ipArr != "" {
		return strings.Split(ipArr, ",")
	}
	return []string{}
}

// BodyParameterValues returns the array of parameter in a POST form body.
func (ctx *Ctx) BodyParameterValues(name string) ([]string, error) {
	err := ctx.Request.Request.ParseForm()
	if err != nil {
		return []string{}, err
	}
	if ctx.Request.Request.PostForm == nil {
		err = ctx.Request.Request.ParseMultipartForm(defaultMaxMemory)
		if err != nil {
			return []string{}, err
		}
	}
	if vs := ctx.Request.Request.PostForm[name]; len(vs) > 0 {
		return vs, nil
	}
	return []string{}, nil
}

// Query reads query parameter with name.
func (ctx *Ctx) Query(name string) Parameter {
	return Parameter{Value: []string{ctx.QueryParameter(name)}}
}

// Form reads form parameter with name.
func (ctx *Ctx) Form(name string) Parameter {
	val, err := ctx.BodyParameterValues(name)
	return Parameter{Value: val, error: err}
}

// Path reads path parameter with name.
func (ctx *Ctx) Path(name string) Parameter {
	return Parameter{Value: []string{ctx.PathParameter(name)}}
}

// Header reads header parameter with name.
func (ctx *Ctx) Header(name string) Parameter {
	return Parameter{Value: []string{ctx.HeaderParameter(name)}}
}

// Bind checks the Content-Type to select a binding engine automatically,
// Depending the "Content-Type" header different bindings are used:
//     "application/json" --> JSON binding
//     "application/xml"  --> XML binding
// otherwise --> returns an error.
// It parses the request's body as JSON if Content-Type == "application/json" using JSON or XML as a JSON input.
// It decodes the json payload into the struct specified as a pointer.
// It writes a 400 error and sets Content-Type header "text/plain" in the response if input is not valid.
func (ctx *Ctx) Bind(obj interface{}) error {
	b := binding.Default(ctx.Request.Request.Method, ctx.Request.HeaderParameter("Content-Type"))
	return ctx.BindWith(obj, b)
}

// MustBind is a shortcur for ctx.Must(ctx.Bind(obj), code, v...)
func (ctx *Ctx) MustBind(obj interface{}, code int, v ...interface{}) {
	ctx.Must(ctx.Bind(obj), code, v...)
}

// BindWith binds the passed struct pointer using the specified binding engine.
// See the binding package.
func (ctx *Ctx) BindWith(obj interface{}, b binding.Binding) error {
	return b.Bind(ctx.Request.Request, obj)
}

// MustBindWith is a shortcur for ctx.Must(ctx.BindWith(obj, b), code, v...)
func (ctx *Ctx) MustBindWith(obj interface{}, b binding.Binding, code int, v ...interface{}) {
	ctx.Must(ctx.BindWith(obj, b), code, v...)
}

// BindJSON is a shortcut for ctx.BindWith(obj, binding.JSON).
func (ctx *Ctx) BindJSON(obj interface{}) error {
	return ctx.BindWith(obj, binding.JSON)
}

// MustBindJSON is a shortcur for ctx.Must(ctx.BindJSON(obj), code, v...)
func (ctx *Ctx) MustBindJSON(obj interface{}, code int, v ...interface{}) {
	ctx.Must(ctx.BindJSON(obj), code, v...)
}

// BindQuery is a shortcut for ctx.BindWith(obj, binding.Query).
func (ctx *Ctx) BindQuery(obj interface{}) error {
	return ctx.BindWith(obj, binding.Query)
}

// MustBindQuery is a shortcur for ctx.Must(ctx.BindQuery(obj), code, v...)
func (ctx *Ctx) MustBindQuery(obj interface{}, code int, v ...interface{}) {
	ctx.Must(ctx.BindQuery(obj), code, v...)
}

// ResponseJSON is a convenience method
// for writing a value wrap in CommonResp as JSON.
// It uses jsoniter for marshalling the value.
func ResponseJSON(w http.ResponseWriter, routeID string, v interface{}) {
	CommonResponse(w, routeID, 0, "", v)
}

// ResponseError is a convenience method to response an error code and message.
// It uses jsoniter for marshalling the value.
func ResponseError(w http.ResponseWriter, routeID string, msg string, code int) {
	CommonResponse(w, routeID, code, msg, nil)
}

// ContainsError is a convenience method to check error is nil.
// If error is nil, it will return false,
// else it will log the error, make a CommonResp response and return true.
// if code is 0, it will use err.Error() as CommonResp.message.
func ContainsError(w http.ResponseWriter, RouteSignature string, err error, code int) bool {
	msg, ok := routeErrMap[RouteSignature][code]
	if !ok {
		msg = globalErrMap[code]
	}
	if CheckError(err, Info().Int("code", code).Str("msg", msg)) {
		return false
	}
	if code == 0 {
		msg = err.Error()
	}
	ResponseError(w, routeIDMap[RouteSignature], msg, code)
	return true
}

// CheckError is a convenience method to check error is nil.
// If error is nil, it will return true,
// else it will log the error and return false
func CheckError(err error, log *LogWrap) bool {
	if err == nil {
		return true
	}
	if log != nil {
		log.Str(zerolog.ErrorFieldName,
			fmt.Sprintf("%+v", errors.WithStack(err))).
			Msg("verify error")
	}
	return false
}

// CommonResponse is a response func.
// just replace it if you'd like to custom response.
var CommonResponse = func(w http.ResponseWriter,
	routeID string, code int, message string, data interface{}) {
	if err := writeJSON(w, http.StatusOK, CommonResp{
		Code:    code,
		Message: message,
		Data:    data,
		RouteID: routeID,
	}); err != nil {
		Warn().Err(err).Msg("json encode")
	}
}

func writeJSON(resp http.ResponseWriter, status int, v interface{}) error {
	if v == nil {
		resp.WriteHeader(status)
		return nil
	}
	resp.Header().Set(restful.HEADER_ContentType, restful.MIME_JSON)
	resp.WriteHeader(status)
	return jsoniter.NewEncoder(resp).Encode(v)
}

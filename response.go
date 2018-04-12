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

// Handle transform a biu handler to a restful.RouteFunction.
func Handle(f func(ctx Ctx)) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		f(Ctx{
			Request:  request,
			Response: response,
		})
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

func (ctx *Ctx) RouteID() string {
	return routeIDMap[ctx.RouteSignature()]
}

func (ctx *Ctx) RouteSignature() string {
	return ctx.SelectedRoutePath() + " " + ctx.Request.Request.Method
}

func (ctx *Ctx) ErrMsg(code int) string {
	return routeErrMap[ctx.RouteSignature()][code]
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
	if CheckError(err, Log().Int("code", code).Str("msg", msg)) {
		return false
	}
	if code == 0 {
		msg = err.Error()
	}
	ResponseError(ctx.Response, ctx.RouteID(), msg, code)
	return true
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
	ipArr := ctx.Proxy()
	if len(ipArr) > 0 && ipArr[0] != "" {
		ip, _, err := net.SplitHostPort(ipArr[0])
		if err != nil {
			ip = ipArr[0]
		}
		return ip
	}
	ip, _, err := net.SplitHostPort(ctx.Request.Request.RemoteAddr)
	if err != nil {
		return ctx.Request.Request.RemoteAddr
	}
	return ip
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

func (ctx *Ctx) Query(name string) Parameter {
	return Parameter{Value: []string{ctx.QueryParameter(name)}}
}

func (ctx *Ctx) Form(name string) Parameter {
	val, err := ctx.BodyParameterValues(name)
	return Parameter{Value: val, error: err}
}

func (ctx *Ctx) Path(name string) Parameter {
	return Parameter{Value: []string{ctx.PathParameter(name)}}
}

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

// BindWith binds the passed struct pointer using the specified binding engine.
// See the binding package.
func (ctx *Ctx) BindWith(obj interface{}, b binding.Binding) error {
	return b.Bind(ctx.Request.Request, obj)
}

// BindJSON is a shortcut for c.BindWith(obj, binding.JSON).
func (ctx *Ctx) BindJSON(obj interface{}) error {
	return ctx.BindWith(obj, binding.JSON)
}

// BindQuery is a shortcut for c.BindWith(obj, binding.Query).
func (ctx *Ctx) BindQuery(obj interface{}) error {
	return ctx.BindWith(obj, binding.Query)
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
	msg := routeErrMap[RouteSignature][code]
	if CheckError(err, Log().Int("code", code).Str("msg", msg)) {
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
func CheckError(err error, log *LogEvt) bool {
	if err == nil {
		return true
	}
	if log != nil {
		Info("verify error", log.Err(err))
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
		Warn("json encode", Log().Err(err))
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

package box

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	_ "unsafe"

	"github.com/dgrijalva/jwt-go/request"
	"github.com/emicklei/go-restful"
	"github.com/gin-gonic/gin/binding"
	"github.com/json-iterator/go"
	"github.com/mpvl/errc"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/tuotoo/biu/auth"
	"github.com/tuotoo/biu/log"
	"github.com/tuotoo/biu/param"
)

var (
	RouteIDMap   = make(map[string]string)
	RouteErrMap  = make(map[string]map[int]string)
	GlobalErrMap = make(map[int]string)
)

// DisableErrHandler disables error handler using errc.
var DisableErrHandler bool

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
	return RouteIDMap[ctx.RouteSignature()]
}

// RouteSignature returns the signature of current route.
// Example: /v1/user/login POST
func (ctx *Ctx) RouteSignature() string {
	return ctx.SelectedRoutePath() + " " + ctx.Request.Request.Method
}

// ErrMsg returns the message of a error code in current route.
func (ctx *Ctx) ErrMsg(code int) string {
	msg, ok := RouteErrMap[ctx.RouteSignature()][code]
	if ok {
		return msg
	}
	return GlobalErrMap[code]
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
	if CheckError(err, log.Info().
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
	log.Info().
		Str("routeID", e.ctx.RouteID()).
		Str("routeSig", e.ctx.RouteSignature()).
		Int("code", e.code).
		Str("msg", msg).
		Str(zerolog.ErrorFieldName, fmt.Sprintf("%+v\n", err)).
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

// Host returns the host of request.
func (ctx *Ctx) Host() string {
	if ctx.Request.Request.Host != "" {
		if hostPart, _, err := net.SplitHostPort(ctx.Request.Request.Host); err == nil {
			return hostPart
		}
		return ctx.Request.Request.Host
	}
	return "localhost"
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
func (ctx *Ctx) Query(name string) param.Parameter {
	return param.NewParameter([]string{ctx.QueryParameter(name)}, nil)
}

// Form reads form parameter with name.
func (ctx *Ctx) Form(name string) param.Parameter {
	return param.NewParameter(ctx.BodyParameterValues(name))
}

// Path reads path parameter with name.
func (ctx *Ctx) Path(name string) param.Parameter {
	return param.NewParameter([]string{ctx.PathParameter(name)}, nil)
}

// Header reads header parameter with name.
func (ctx *Ctx) Header(name string) param.Parameter {
	return param.NewParameter([]string{ctx.HeaderParameter(name)}, nil)
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
	msg, ok := RouteErrMap[RouteSignature][code]
	if !ok {
		msg = GlobalErrMap[code]
	}
	if CheckError(err, log.Info().Int("code", code).Str("msg", msg)) {
		return false
	}
	if code == 0 {
		msg = err.Error()
	}
	ResponseError(w, RouteIDMap[RouteSignature], msg, code)
	return true
}

// CheckError is a convenience method to check error is nil.
// If error is nil, it will return true,
// else it will log the error and return false
func CheckError(err error, log *log.Wrap) bool {
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
		log.Warn().Err(err).Msg("json encode")
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

// IsLogin gets JWT token in request by OAuth2Extractor,
// and parse it with CheckToken.
func (ctx *Ctx) IsLogin() (userID string, err error) {
	tokenString, err := request.OAuth2Extractor.ExtractToken(ctx.Request.Request)
	if err != nil {
		log.Info().Err(err).Msg("no auth header")
		return "", err
	}
	return auth.CheckToken(tokenString)
}

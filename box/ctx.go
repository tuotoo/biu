package box

import (
	"net"
	"net/http"
	"strings"
	_ "unsafe"

	"github.com/dgrijalva/jwt-go/v4/request"
	"github.com/emicklei/go-restful/v3"
	"github.com/gin-gonic/gin/binding"
	"github.com/mpvl/errc"
	"github.com/tuotoo/biu/auth"
	"github.com/tuotoo/biu/log"
	"github.com/tuotoo/biu/param"
	"golang.org/x/xerrors"
)

const (
	defaultMaxMemory  = 32 << 20 // 32 MB
	BiuAttrErr        = "__BIU_ERROR__"
	BiuAttrErrCode    = "__BIU_ERROR_CODE__"
	BiuAttrErrMsg     = "__BIU_ERROR_MESSAGE__"
	BiuAttrErrArgs    = "__BIU_ERROR_ARGS__"
	BiuAttrRouteID    = "__BIU_ROUTE_ID__"
	BiuAttrAuthUserID = "__BIU_AUTH_USER_ID__"
	BiuAttrEntities   = "__BIU_ENTITIES__"
)

const CtxSignature = "github.com/tuotoo/biu/box.Ctx"

// Ctx wrap *restful.Request and *restful.Response in one struct.
type Ctx struct {
	*restful.Request
	*restful.Response
	*restful.FilterChain
	ErrCatcher errc.Catcher
	Logger     log.ILogger
}

// Req returns http.Request of ctx.
func (ctx Ctx) Req() *http.Request {
	return ctx.Request.Request
}

// Resp returns http.ResponseWriter of ctx.
func (ctx Ctx) Resp() http.ResponseWriter {
	return ctx.Response.ResponseWriter
}

// ResponseJSON is a convenience method
// for writing a value wrap in CommonResp as JSON.
func (ctx *Ctx) ResponseJSON(v ...interface{}) {
	ctx.SetAttribute(BiuAttrEntities, v)
}

func (ctx *Ctx) Transform(f func(...interface{}) []interface{}) {
	if entities, ok := ctx.Attribute(BiuAttrEntities).([]interface{}); ok {
		ctx.SetAttribute(BiuAttrEntities, f(entities...))
	}
}

// ResponseError is a convenience method to response an error code and message.
func (ctx *Ctx) ResponseError(code int, msg string) {
	ctx.SetAttribute(BiuAttrErrCode, code)
	ctx.SetAttribute(BiuAttrErrMsg, msg)
}

// RouteID returns the RouteID of current route.
func (ctx *Ctx) RouteID() string {
	return ctx.Attribute(BiuAttrRouteID).(string)
}

// RouteSignature returns the signature of current route.
// Example: /v1/user/login POST
func (ctx *Ctx) RouteSignature() string {
	return ctx.SelectedRoutePath() + " " + ctx.Req().Method
}

// Redirect replies to the request with a redirect to url.
func (ctx *Ctx) Redirect(url string, code int) {
	http.Redirect(ctx.Resp(), ctx.Req(), url, code)
}

// ContainsError is a convenience method to check error is nil.
// If error is nil, it will return false,
// else it will log the error, make a CommonResp response and return true.
// if code is 0, it will use err.Error() as CommonResp.message.
func (ctx *Ctx) ContainsError(err error, code int, v ...interface{}) bool {
	if err == nil {
		return false
	}
	ctx.ResponseStdErrCode(code, v...)
	return true
}

type errHandler struct {
	ctx  *Ctx
	code int
	v    []interface{}
}

// Handle implements errc.Handle
func (e errHandler) Handle(s errc.State, err error) error {
	vLen := len(e.v)
	if vLen > 0 {
		if f, ok := e.v[vLen-1].(func()); ok {
			f()
			e.v = e.v[:vLen-1]
		}
	}
	e.ctx.ResponseStdErrCode(e.code, e.v...)
	e.ctx.SetAttribute(BiuAttrErr, err)
	return err
}

// Must causes a return from a function if err is not nil.
func (ctx *Ctx) Must(err error, code int, v ...interface{}) {
	ctx.ErrCatcher.Must(err, errHandler{ctx: ctx, code: code, v: v})
}

// ResponseStdErrCode is a convenience method response a code
// with msg in Code Desc.
func (ctx *Ctx) ResponseStdErrCode(code int, v ...interface{}) {
	ctx.SetAttribute(BiuAttrErrCode, code)
	ctx.SetAttribute(BiuAttrErrArgs, v)
}

// UserID returns UserID stored in attribute.
func (ctx *Ctx) UserID() string {
	userID, ok := ctx.Attribute(BiuAttrAuthUserID).(string)
	if !ok {
		return ""
	}
	return userID
}

// IP returns the IP address of request.
func (ctx *Ctx) IP() string {
	ra := ctx.Req().RemoteAddr
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
	if ctx.Req().Host != "" {
		if hostPart, _, err := net.SplitHostPort(ctx.Req().Host); err == nil {
			return hostPart
		}
		return ctx.Req().Host
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
	if strings.HasPrefix(ctx.Req().Header.Get(restful.HEADER_ContentType), "multipart/form-data") {
		err := ctx.Req().ParseMultipartForm(defaultMaxMemory)
		if err != nil {
			return []string{}, err
		}
	} else {
		err := ctx.Req().ParseForm()
		if err != nil {
			return []string{}, err
		}
	}
	if vs := ctx.Req().PostForm[name]; len(vs) > 0 {
		return vs, nil
	}
	return []string{}, nil
}

// Query reads query parameter with name.
func (ctx *Ctx) Query(name string) param.Parameter {
	return param.NewParameter(ctx.QueryParameters(name), nil)
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

func filterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}

// Bind checks the Content-Type to select a binding engine automatically,
// Depending the "Content-Type" header different bindings are used:
//
//	"application/json" --> JSON binding
//	"application/xml"  --> XML binding
//
// otherwise --> returns an error.
// It parses the request's body as JSON if Content-Type == "application/json" using JSON or XML as a JSON input.
// It decodes the json payload into the struct specified as a pointer.
// It writes a 400 error and sets Content-Type header "text/plain" in the response if input is not valid.
func (ctx *Ctx) Bind(obj interface{}) error {
	b := binding.Default(ctx.Req().Method, filterFlags(ctx.Request.HeaderParameter("Content-Type")))
	return ctx.BindWith(obj, b)
}

// MustBind is a shortcur for ctx.Must(ctx.Bind(obj), code, v...)
func (ctx *Ctx) MustBind(obj interface{}, code int, v ...interface{}) {
	ctx.Must(ctx.Bind(obj), code, v...)
}

// BindWith binds the passed struct pointer using the specified binding engine.
// See the binding package.
func (ctx *Ctx) BindWith(obj interface{}, b binding.Binding) error {
	return b.Bind(ctx.Req(), obj)
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

// IsLogin gets JWT token in request by OAuth2Extractor,
// and parse it with CheckToken.
func (ctx *Ctx) IsLogin(i *auth.Instance) (userID string, err error) {
	tokenString, err := request.OAuth2Extractor.ExtractToken(ctx.Req())
	if err != nil {
		return "", xerrors.Errorf("no auth header: %w", err)
	}
	return i.CheckToken(tokenString)
}

func (ctx *Ctx) Next() {
	ctx.ProcessFilter(ctx.Request, ctx.Response)
}

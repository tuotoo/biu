package biu

import (
	"fmt"
	"github.com/go-openapi/spec"
	"github.com/tuotoo/biu/box"
	"net/http"
	"net/http/httptest"

	"github.com/emicklei/go-restful"
	"github.com/tuotoo/biu/opt"
)

var DefaultContainer = New()

// Container of restful
type Container struct {
	*restful.Container
	swaggerTags map[*http.ServeMux][]spec.Tag
	errors      map[int]string
}

// New creates a new restful container.
func New() *Container {
	container := restful.NewContainer()
	errors := make(map[int]string)
	container.Filter(Filter(func(ctx box.Ctx) {
		ctx.Next()
		code, ok := ctx.Attribute(box.BiuAttrErrCode).(int)
		if !ok || code == 0 {
			return
		}
		msg := errors[code]
		args, ok := ctx.Attribute(box.BiuAttrErrArgs).([]interface{})
		if ok && len(args) > 0 {
			msg = fmt.Sprintf(msg, args...)
		}
		routeID := ctx.Attribute(box.BiuAttrRouteID).(string)
		box.ResponseError(ctx.Resp(), routeID, msg, code)
	}))
	return &Container{
		Container:   container,
		swaggerTags: make(map[*http.ServeMux][]spec.Tag),
		errors:      errors,
	}
}

// AddServices adds services with namespace for container.
func (c *Container) AddServices(prefix string, opts opt.ServicesFuncArr, wss ...NS) {
	addService(prefix, opts, c, wss...)
}

// AddServices adds services with namespace.
func AddServices(prefix string, opts opt.ServicesFuncArr, wss ...NS) {
	DefaultContainer.AddServices(prefix, opts, wss...)
}

// Run starts up a web server for container.
func (c *Container) Run(addr string, opts ...opt.RunFunc) {
	run(addr, c, opts...)
}

// Run starts up a web server with default container.
func Run(addr string, opts ...opt.RunFunc) {
	DefaultContainer.Run(addr, opts...)
}

// NewTestServer returns a Test Server.
func (c *Container) NewTestServer() *TestServer {
	return &TestServer{
		Server: httptest.NewServer(c),
	}
}

// NewTestServer returns a Test Server.
func NewTestServer() *TestServer {
	return DefaultContainer.NewTestServer()
}

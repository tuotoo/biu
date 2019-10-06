package biu

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/emicklei/go-restful"
	"github.com/go-openapi/spec"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/log"
	"github.com/tuotoo/biu/opt"
)

var DefaultContainer = New(restful.DefaultContainer)

// Container of restful
type Container struct {
	*restful.Container
	swaggerTags map[*http.ServeMux][]spec.Tag
	errors      map[int]string
	routeID     map[string]string
	logger      log.ILogger
}

// New creates a new restful container.
func New(container ...*restful.Container) *Container {
	var rc *restful.Container
	if len(container) > 0 {
		rc = container[0]
	} else {
		rc = restful.NewContainer()
	}

	errors := make(map[int]string)
	routeMap := make(map[string]string)
	c := &Container{
		Container:   rc,
		swaggerTags: make(map[*http.ServeMux][]spec.Tag),
		routeID:     routeMap,
		errors:      errors,
		logger:      log.DefaultLogger{},
	}
	c.Filter(c.FilterFunc(func(ctx box.Ctx) {
		ctx.Next()

		code, ok := ctx.Attribute(box.BiuAttrErrCode).(int)
		if ok && code != 0 {
			return
		}

		err := ctx.WriteAsJson(box.CommonResp{
			Data:    ctx.Attribute(box.BiuAttrEntity),
			RouteID: ctx.RouteID(),
		})
		if err != nil {
			ctx.Logger.Info(log.BiuInternalInfo{
				Err: err,
				Extras: map[string]interface{}{
					"Data":    ctx.Attribute(box.BiuAttrEntity),
					"RouteID": ctx.RouteID(),
				},
			})
		}
	}))
	c.Filter(c.FilterFunc(func(ctx box.Ctx) {
		routeID := routeMap[ctx.RouteSignature()]
		ctx.SetAttribute(box.BiuAttrRouteID, routeID)
		ctx.Next()
		code, ok := ctx.Attribute(box.BiuAttrErrCode).(int)
		if !ok || code == 0 {
			return
		}
		msg, ok := ctx.Attribute(box.BiuAttrErrMsg).(string)
		if !ok {
			msg = errors[code]
		}
		args, ok := ctx.Attribute(box.BiuAttrErrArgs).([]interface{})
		if ok && len(args) > 0 {
			msg = fmt.Sprintf(msg, args...)
		}
		logInfo := log.BiuInternalInfo{
			Extras: map[string]interface{}{
				"routeID":  ctx.RouteID(),
				"routeSig": ctx.RouteSignature(),
				"code":     code,
				"msg":      msg,
			},
		}
		if err, ok := ctx.Attribute(box.BiuAttrErr).(error); ok && err != nil {
			logInfo.Err = err
		}
		ctx.Logger.Info(logInfo)
		ctx.ResponseError(code, msg)
	}))
	return c
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

func (c *Container) Handle(f func(ctx box.Ctx)) restful.RouteFunction {
	return HandleWithLogger(f, c.logger)
}

// Filter transform a biu handler to a restful.FilterFunction
func (c *Container) FilterFunc(f func(ctx box.Ctx)) restful.FilterFunction {
	return FilterWithLogger(f, c.logger)
}

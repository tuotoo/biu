package biu

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"

	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/log"
	"github.com/tuotoo/biu/opt"
)

var DefaultContainer = New(restful.DefaultContainer)

func init() {
	// TrimRightSlashEnabled controls whether
	// - path on route building is using path.Join
	// - the path of the incoming request is trimmed of its slash suffix.
	// Value of false matches the behavior of go-restful > 3.9.0
	restful.TrimRightSlashEnabled = false
}

// Container of restful
type Container struct {
	*restful.Container
	*http.Server
	swaggerTags map[*http.ServeMux][]spec.Tag
	errors      map[int]string
	routeID     map[string]string
	logger      log.ILogger
}

func DefaultResponseTransformer(ctx box.Ctx) {
	ctx.Next()

	code, ok := ctx.Attribute(box.BiuAttrErrCode).(int)
	if ok && code != 0 {
		return
	}

	entities, ok := ctx.Attribute(box.BiuAttrEntities).([]interface{})
	if !ok {
		return
	}
	if len(entities) < 1 {
		return
	}

	err := ctx.WriteAsJson(box.CommonResp{
		Data:    entities[0],
		RouteID: ctx.RouteID(),
	})
	if err != nil {
		ctx.Logger.Info(log.BiuInternalInfo{
			Err: err,
			Extras: map[string]interface{}{
				"Data":    entities[0],
				"RouteID": ctx.RouteID(),
			},
		})
	}
}

func DefaultErrorTransformer(c *Container) func(ctx box.Ctx) {
	return func(ctx box.Ctx) {
		routeID := c.RouteIDMap()[ctx.RouteSignature()]
		ctx.SetAttribute(box.BiuAttrRouteID, routeID)
		ctx.Next()
		code, ok := ctx.Attribute(box.BiuAttrErrCode).(int)
		if !ok || code == 0 {
			return
		}
		msg, ok := ctx.Attribute(box.BiuAttrErrMsg).(string)
		if !ok {
			msg = c.ErrorMap()[code]
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
		err := ctx.WriteAsJson(box.CommonResp{
			Code:    code,
			Message: msg,
			RouteID: ctx.RouteID(),
		})
		if err != nil {
			ctx.Logger.Info(log.BiuInternalInfo{
				Err: err,
				Extras: map[string]interface{}{
					"Code":    code,
					"Msg":     msg,
					"RouteID": ctx.RouteID(),
				},
			})
		}
	}
}

// New creates a new restful container.
func New(container ...*restful.Container) *Container {
	c := NewContainer(container...)
	c.Filter(c.FilterFunc(DefaultResponseTransformer))
	c.Filter(c.FilterFunc(DefaultErrorTransformer(c)))
	return c
}

func NewContainer(container ...*restful.Container) *Container {
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

// FilterFunc transform a biu handler to a restful.FilterFunction
func (c *Container) FilterFunc(f func(ctx box.Ctx)) restful.FilterFunction {
	return FilterWithLogger(f, c.logger)
}

func (c *Container) RouteIDMap() map[string]string {
	return c.routeID
}

func (c *Container) ErrorMap() map[int]string {
	return c.errors
}

func (c *Container) NewWS() WS {
	return WS{
		WebService: new(restful.WebService),
		Container:  c,
		errors:     make(map[string]map[int]string),
	}
}

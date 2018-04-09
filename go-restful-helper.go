package biu

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/json-iterator/go"
	"github.com/rs/zerolog"
)

// nolint
// MIME_HTML_FORM is application/x-www-form-urlencoded header
const MIME_HTML_FORM = "application/x-www-form-urlencoded"

var swaggerTags []spec.Tag

// Container of restful
type Container struct{ *restful.Container }

// New creates a new restful container.
func New() Container {
	return Container{Container: restful.NewContainer()}
}

// AddServices adds services with namespace for container.
func (c *Container) AddServices(prefix string,
	filters []restful.FilterFunction, wss ...NS,
) {
	addService(prefix, filters, c.Container, wss...)
}

// AddServices adds services with namespace.
func AddServices(prefix string, filters []restful.FilterFunction, wss ...NS) {
	addService(prefix, filters, restful.DefaultContainer, wss...)
}

// RunConfig is the running config of container.
type RunConfig struct {
	BeforeShutDown func()
	AfterShutDown  func()
}

// Run starts up a web server for container.
func (c *Container) Run(addr string, cfg *RunConfig) {
	run(addr, c.Container, cfg)
}

// Run starts up a web server with default container.
func Run(addr string, cfg *RunConfig) {
	run(addr, nil, cfg)
}

var (
	routeIDMap  = make(map[string]string)
	routeErrMap = make(map[string]map[int]string)
)

// RouteOpt contains some options of route.
type RouteOpt struct {
	ID              string
	To              func(ctx Ctx)
	Auth            bool
	NeedPermissions []string
	Errors          map[int]string
}

// Route creates a new Route using the RouteBuilder
// and add to the ordered list of Routes.
func (ws WS) Route(builder *restful.RouteBuilder, opt *RouteOpt) {
	if opt != nil {
		if opt.To != nil {
			builder = builder.To(Handle(opt.To))
			if opt.ID != "" {
				builder = builder.Operation(opt.ID)
			} else {
				builder = builder.Operation(nameOfFunction(opt.To))
			}
		}
		elm := reflect.ValueOf(builder).Elem()
		p1 := elm.FieldByName("rootPath").String()
		p2 := elm.FieldByName("currentPath").String()
		path := strings.TrimRight(p1, "/") + "/" + strings.TrimLeft(p2, "/")
		method := elm.FieldByName("httpMethod").String()
		mapKey := path + " " + method
		if opt.ID != "" {
			routeIDMap[mapKey] = opt.ID
		}
		if _, ok := routeErrMap[mapKey]; !ok {
			routeErrMap[mapKey] = make(map[int]string)
		}
		for k, v := range opt.Errors {
			routeErrMap[mapKey][k] = v
			builder = builder.Returns(k, v, nil)
		}
		if opt.Auth {
			builder = builder.Metadata("jwt", true)
		}
		if len(opt.NeedPermissions) != 0 {
			builder = builder.Notes("Need Permission: " +
				strings.Join(opt.NeedPermissions, " "))
		}
	}
	ws.WebService.Route(builder)
}

func addService(
	prefix string,
	filters []restful.FilterFunction,
	container *restful.Container,
	wss ...NS,
) {
	for _, v := range wss {
		// build web service
		ws := new(restful.WebService)
		path := prefix + "/" + v.NameSpace
		ws.Path(path).
			Consumes(restful.MIME_JSON).
			Produces(restful.MIME_JSON)
		for _, f := range filters {
			ws.Filter(f)
		}
		v.Controller.WebService(WS{WebService: ws})
		container.Add(ws)

		// add swagger tags to routes of webservice
		tagProps := spec.TagProps{
			Name:        v.NameSpace,
			Description: v.Desc,
		}
		if v.ExternalDesc != "" {
			tagProps.ExternalDocs = &spec.ExternalDocumentation{
				Description: v.ExternalDesc,
				URL:         v.ExternalURL,
			}
		}
		swaggerTags = append(swaggerTags, spec.Tag{
			TagProps: tagProps,
		})
		routes := ws.Routes()
		for ri, r := range routes {
			Info("routers", Log().
				Str("path", r.Path).
				Str("method", r.Method))
			if routes[ri].Metadata == nil {
				routes[ri].Metadata = make(map[string]interface{})
			}
			routes[ri].Metadata[restfulspec.KeyOpenAPITags] = []string{v.NameSpace}
		}

	}
}

func run(addr string, handler http.Handler, cfg *RunConfig) {
	address := addr
	hostAndPort := strings.Split(addr, ":")
	if len(hostAndPort) == 0 || (len(hostAndPort) > 1 && hostAndPort[1] == "") {
		address = ":8080"
	}
	server := http.Server{
		Addr:    address,
		Handler: handler,
	}
	go func() {
		Info("listening", Log().Str("addr", address))
		Fatal("listening", Log().Err(server.ListenAndServe()))
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	Info("signal receive", Log().Interface("ch", <-ch))
	if cfg != nil && cfg.BeforeShutDown != nil {
		cfg.BeforeShutDown()
	}
	Info("shut down", Log().Err(server.Shutdown(context.TODO())))
	if cfg != nil && cfg.AfterShutDown != nil {
		cfg.AfterShutDown()
	}
	Info("server is down gracefully", Log())
}

type ctlCtx struct {
	filters  []restful.FilterFunction
	function restful.RouteFunction
	method   string
	path     string
}

// CtlFuncs is a map contains all handler of a controller.
// the key of CtlFuncs is "Method Path" of handler.
type CtlFuncs map[string]ctlCtx

// GetCtlFuncs returns the handler map of a controller.
func GetCtlFuncs(ctlInterface CtlInterface) CtlFuncs {
	ws := new(restful.WebService)
	ws.Path("/").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	ctlInterface.WebService(WS{ws})
	m := make(map[string]ctlCtx)
	for _, v := range ws.Routes() {
		m[v.Method+" "+v.Path] = ctlCtx{
			filters:  v.Filters,
			function: v.Function,
			method:   v.Method,
			path:     v.Path,
		}
	}
	return m
}

func (m CtlFuncs) httpHandler(n string) http.Handler {
	c := restful.NewContainer()
	ws := new(restful.WebService)
	for _, f := range m[n].filters {
		ws = ws.Filter(f)
	}
	ws.Route(ws.Method(m[n].method).Path(m[n].path).To(func(
		request *restful.Request,
		response *restful.Response,
	) {
		m[n].function(request, response)
	}))
	c.Add(ws)
	return c
}

// NewTestServer returns a Test Server.
func (m CtlFuncs) NewTestServer(method, path string) *httptest.Server {
	return httptest.NewServer(m.httpHandler(method + " " + path))
}

// LogFilter logs
//	{
//		remote_addr,
//		method,
// 		uri,
//		proto,
//		status_code,
//		content_length,
//	}
// for each request
func LogFilter() restful.FilterFunction {
	return func(
		req *restful.Request,
		resp *restful.Response,
		chain *restful.FilterChain,
	) {
		chain.ProcessFilter(req, resp)
		logger.Info().Dict("fields", zerolog.Dict().
			Str("remote_addr", strings.Split(req.Request.RemoteAddr, ":")[0]).
			Str("method", req.Request.Method).
			Str("uri", req.Request.URL.RequestURI()).
			Str("proto", req.Request.Proto).
			Int("status_code", resp.StatusCode()).
			Int("content_length", resp.ContentLength())).Msg("req")
	}
}

func init() {
	restful.RegisterEntityAccessor(restful.MIME_JSON,
		newJsoniterEntityAccessor())
}

func newJsoniterEntityAccessor() restful.EntityReaderWriter {
	return jsoniterEntityAccess{}
}

type jsoniterEntityAccess struct{}

// Read unmarshalls the value from JSON using jsoniter.
func (jsoniterEntityAccess) Read(req *restful.Request, v interface{}) error {
	decoder := jsoniter.NewDecoder(req.Request.Body)
	decoder.UseNumber()
	return decoder.Decode(v)
}

// Write marshalls the value to JSON using jsoniter
// and set the Content-Type Header.
func (j jsoniterEntityAccess) Write(
	resp *restful.Response,
	status int,
	v interface{},
) error {
	return writeJSON(resp, status, v)
}

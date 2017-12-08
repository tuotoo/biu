package biu

import (
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/rs/zerolog"
)

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

// RouteOpt contains some options of route.
type RouteOpt struct {
	Auth   bool
	Errors map[int]string
}

// Route creates a new Route using the RouteBuilder
// and add to the ordered list of Routes.
func (ws WS) Route(builder *restful.RouteBuilder, opt *RouteOpt) {
	if opt != nil {
		for k, v := range opt.Errors {
			codeDesc.m[k] = v
			builder = builder.Returns(k, v, nil)
		}
		if opt.Auth {
			builder = builder.Param(ws.HeaderParameter("Authorization", "JWT Token").
				DefaultValue("bearer ").DataType("string").Required(true))
		}
	}
	ws.WebService.Route(builder)
}

func addService(
	prefix string, filters []restful.FilterFunction,
	container *restful.Container, wss ...NS,
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
			Info("router", Log().
				Str("path", r.Path).
				Str("method", r.Method))
			routes[ri].Metadata = map[string]interface{}{
				restfulspec.KeyOpenAPITags: []string{v.NameSpace},
			}
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

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	Info("signal receive", Log().Interface("ch", <-ch))
	if cfg != nil {
		cfg.BeforeShutDown()
	}
	server.Shutdown(nil)
	if cfg != nil {
		cfg.AfterShutDown()
	}
	Info("shut down", Log())
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

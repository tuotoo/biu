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
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/gavv/httpexpect"
	"github.com/go-openapi/spec"
	"github.com/json-iterator/go"
	"github.com/rs/zerolog"
)

const (
	// MIME_HTML_FORM is application/x-www-form-urlencoded header
	MIME_HTML_FORM = "application/x-www-form-urlencoded"
	// MIME_FILE_FORM is multipart/form-data
	MIME_FILE_FORM = "multipart/form-data"
)

var swaggerTags = make(map[*http.ServeMux][]spec.Tag)

type GlobalServiceOpt struct {
	Filters []restful.FilterFunction
	Errors  map[int]string
}

// Container of restful
type Container struct{ *restful.Container }

// New creates a new restful container.
func New() Container {
	return Container{Container: restful.NewContainer()}
}

// AddServices adds services with namespace for container.
func (c *Container) AddServices(prefix string, opt *GlobalServiceOpt, wss ...NS) {
	addService(prefix, opt, c.Container, wss...)
}

// AddServices adds services with namespace.
func AddServices(prefix string, opt *GlobalServiceOpt, wss ...NS) {
	addService(prefix, opt, restful.DefaultContainer, wss...)
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
	routeIDMap   = make(map[string]string)
	routeErrMap  = make(map[string]map[int]string)
	globalErrMap = make(map[int]string)
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
	opt *GlobalServiceOpt,
	container *restful.Container,
	wss ...NS,
) {
	for _, v := range wss {
		// build web service
		ws := new(restful.WebService)
		path := prefix + "/" + v.NameSpace
		ws.Path(path).Produces(restful.MIME_JSON)
		if opt != nil {
			for _, f := range opt.Filters {
				ws.Filter(f)
			}
			for k, v := range opt.Errors {
				globalErrMap[k] = v
			}
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
		swaggerTags[container.ServeMux] = append(swaggerTags[container.ServeMux], spec.Tag{
			TagProps: tagProps,
		})
		routes := ws.Routes()
		for ri, r := range routes {
			Info().
				Str("path", r.Path).
				Str("method", r.Method).
				Msg("routers")
			if routes[ri].Metadata == nil {
				routes[ri].Metadata = make(map[string]interface{})
			}
			if len(routes[ri].Consumes) == 0 {
				if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
					r.Consumes = []string{MIME_HTML_FORM}
				} else {
					r.Consumes = []string{restful.MIME_JSON}
				}
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
		Info().Str("addr", address).Msg("listening")
		Fatal().Err(server.ListenAndServe()).Msg("listening")
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	Info().Interface("ch", <-ch).Msg("signal receive")
	if cfg != nil && cfg.BeforeShutDown != nil {
		cfg.BeforeShutDown()
	}
	Info().Err(server.Shutdown(context.TODO())).Msg("shut down")
	if cfg != nil && cfg.AfterShutDown != nil {
		cfg.AfterShutDown()
	}
	Info().Msg("server is down gracefully")
}

// TestServer wraps a httptest.Server
type TestServer struct {
	*httptest.Server
}

// NewTestServer returns a Test Server.
func (c *Container) NewTestServer() *TestServer {
	return &TestServer{
		Server: httptest.NewServer(c),
	}
}

// NewTestServer returns a Test Server.
func NewTestServer() *TestServer {
	return &TestServer{
		Server: httptest.NewServer(restful.DefaultContainer),
	}
}

// WithT accept testing.T and returns httpexpect.Expect
func (s *TestServer) WithT(t *testing.T) *httpexpect.Expect {
	return httpexpect.New(t, s.URL)
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
	restful.RegisterEntityAccessor(restful.MIME_JSON, newJsoniterEntityAccessor())
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

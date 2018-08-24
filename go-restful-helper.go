package biu

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
	_ "unsafe"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/gavv/httpexpect"
	"github.com/go-openapi/spec"
	"github.com/rs/zerolog"
	"github.com/tuotoo/biu/opt"
)

const (
	// MIME_HTML_FORM is application/x-www-form-urlencoded header
	MIME_HTML_FORM = "application/x-www-form-urlencoded"
	// MIME_FILE_FORM is multipart/form-data
	MIME_FILE_FORM = "multipart/form-data"
)

type pathExpression struct {
	LiteralCount int      // the number of literal characters (means those not resulting from template variable substitution)
	VarNames     []string // the names of parameters (enclosed by {}) in the path
	VarCount     int      // the number of named parameters (enclosed by {}) in the path
	Matcher      *regexp.Regexp
	Source       string // Path as defined by the RouteBuilder
	tokens       []string
}

//go:linkname newPathExpression github.com/emicklei/go-restful.newPathExpression
func newPathExpression(path string) (*pathExpression, error)

var swaggerTags = make(map[*http.ServeMux][]spec.Tag)

// GlobalServiceOpt is the options of global service.
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

// Run starts up a web server for container.
func (c *Container) Run(addr string, opts ...opt.RunFunc) {
	run(addr, c.Container, opts...)
}

// Run starts up a web server with default container.
func Run(addr string, opts ...opt.RunFunc) {
	run(addr, nil, opts...)
}

var (
	routeIDMap   = make(map[string]string)
	routeErrMap  = make(map[string]map[int]string)
	globalErrMap = make(map[int]string)
)

// RouteErrors is a errors map for routes
type RouteErrors map[int]string

// RouteOpt contains some options of route.
type RouteOpt struct {
	ID                 string
	To                 func(ctx Ctx)
	Auth               bool
	NeedPermissions    []string
	Errors             RouteErrors
	DisableAutoPathDoc bool
	ExtraPathDocs      []string
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
		if elm.FieldByName("function").IsNil() {
			builder = builder.To(func(_ *restful.Request, _ *restful.Response) {})
		}
		p1 := elm.FieldByName("rootPath").String()
		p2 := elm.FieldByName("currentPath").String()
		path := strings.TrimRight(p1, "/") + "/" + strings.TrimLeft(p2, "/")
		method := elm.FieldByName("httpMethod").String()
		mapKey := path + " " + method

		if globalOptions.autoGenPathDoc && !opt.DisableAutoPathDoc {
			exp, err := newPathExpression(p2)
			if err != nil {
				Fatal().Err(err).Str("path", p2).Msg("invalid path")
			}
			for i, v := range exp.VarNames {
				desc := v
				if len(opt.ExtraPathDocs) > i {
					desc = opt.ExtraPathDocs[i]
				}
				builder = builder.Param(ws.PathParameter(v, desc))
			}
		}

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
			Info().Str("path", r.Path).
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

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// ListenAndServe listens on the TCP network address srv.Addr and then
// calls Serve to handle requests on incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
// If srv.Addr is blank, ":http" is used.
// ListenAndServe always returns a non-nil error.
func ListenAndServe(srv *http.Server, addrChan chan<- string) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":0"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	tcpListener := ln.(*net.TCPListener)
	{
		addr := tcpListener.Addr()
		addrChan <- addr.String()
	}
	return srv.Serve(tcpKeepAliveListener{TCPListener: tcpListener})
}

func run(addr string, handler http.Handler, opts ...opt.RunFunc) {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	addrChan := make(chan string)
	go func() {
		Fatal().Err(ListenAndServe(server, addrChan)).Msg("listening")
	}()
	select {
	case addr := <-addrChan:
		Info().Str("addr", addr).Msg("listening")
	case <-time.After(time.Second):
		Fatal().Msg("something went wrong when starting the server")
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	Info().Interface("ch", <-ch).Msg("signal receive")

	cfg := &opt.Run{
		BeforeShutDown: func() {},
		AfterShutDown:  func() {},
	}
	for _, f := range opts {
		if f != nil {
			f(cfg)
		}
	}

	cfg.BeforeShutDown()
	Info().Err(server.Shutdown(context.TODO())).Msg("shutting down")
	cfg.AfterShutDown()
	Info().Msg("server shuts down gracefully")
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
		start := time.Now()
		chain.ProcessFilter(req, resp)
		logger.Info().Dict("fields", zerolog.Dict().
			Str("remote_addr", strings.Split(req.Request.RemoteAddr, ":")[0]).
			Str("method", req.Request.Method).
			Str("uri", req.Request.URL.RequestURI()).
			Str("proto", req.Request.Proto).
			Int("status_code", resp.StatusCode()).
			Dur("dur", time.Since(start)).
			Int("content_length", resp.ContentLength())).Msg("req")
	}
}

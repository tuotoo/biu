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
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/log"
	"github.com/tuotoo/biu/opt"
	"golang.org/x/xerrors"
)

const (
	// MIME_HTML_FORM is application/x-www-form-urlencoded header
	MIME_HTML_FORM = "application/x-www-form-urlencoded"
	// MIME_FILE_FORM is multipart/form-data
	MIME_FILE_FORM = "multipart/form-data"
)

var AutoGenPathDoc = false

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

//go:linkname nameOfFunction github.com/emicklei/go-restful.nameOfFunction
func nameOfFunction(f interface{}) string

// Route creates a new Route using the RouteBuilder
// and add to the ordered list of Routes.
func (ws WS) Route(builder *restful.RouteBuilder, opts ...opt.RouteFunc) {
	cfg := &opt.Route{
		EnableAutoPathDoc: true,
		To:                func(ctx box.Ctx) {},
	}
	for _, f := range opts {
		if f != nil {
			f(cfg)
		}
	}
	builder = builder.To(ws.Container.Handle(cfg.To))
	if cfg.ID != "" {
		builder = builder.Operation(cfg.ID)
	} else {
		builder = builder.Operation(nameOfFunction(cfg.To))
	}

	elm := reflect.ValueOf(builder).Elem()

	p1 := elm.FieldByName("rootPath").String()
	p2 := elm.FieldByName("currentPath").String()
	path := strings.TrimRight(p1, "/") + "/" + strings.TrimLeft(p2, "/")
	method := elm.FieldByName("httpMethod").String()
	mapKey := path + " " + method

	for _, v := range cfg.Params {
		switch v.FieldType {
		case opt.FieldQuery:
			param := ws.QueryParameter(v.Name, v.Desc).DataType(v.Type).DataFormat(v.Format)
			if v.IsMulti {
				param = param.AllowMultiple(true).CollectionFormat("multi")
			}
			builder = builder.Param(param)
		case opt.FieldForm:
			param := ws.FormParameter(v.Name, v.Desc).DataType(v.Type).DataFormat(v.Format)
			if v.IsMulti {
				param = param.AllowMultiple(true).CollectionFormat("multi")
			}
			builder = builder.Param(param)
			if v.Type == "file" {
				builder = builder.Consumes(MIME_FILE_FORM)
			}
		case opt.FieldBody:
			builder = builder.Reads(v.Body, v.Desc)
		case opt.FieldPath:
			param := ws.PathParameter(v.Name, v.Desc).DataType(v.Type).DataFormat(v.Format)
			builder = builder.Param(param)
		case opt.FieldHeader:
			param := ws.HeaderParameter(v.Name, v.Desc).DataType(v.Type).DataFormat(v.Format)
			if v.IsMulti {
				param = param.AllowMultiple(true).CollectionFormat("multi")
			}
			builder = builder.Param(param)
		case opt.FieldReturn:
			builder = builder.DefaultReturns(v.Desc, v.Body)
		case opt.FieldUnknown:
			var param *restful.Parameter
			switch method {
			case http.MethodGet, http.MethodDelete:
				param = ws.QueryParameter(v.Name, v.Desc)
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				param = ws.FormParameter(v.Name, v.Desc)
				if v.Type == "file" {
					builder = builder.Consumes(MIME_FILE_FORM)
				}
			default:
				continue
			}
			param = param.DataType(v.Type).DataFormat(v.Format)
			if v.IsMulti {
				param = param.AllowMultiple(true).CollectionFormat("multi")
			}
			builder = builder.Param(param)
		}
	}

	if AutoGenPathDoc && cfg.EnableAutoPathDoc {
		exp, err := newPathExpression(p2)
		if err != nil {
			ws.Container.logger.Fatal(log.BiuInternalInfo{
				Err:    xerrors.Errorf("invalid path: %s", err),
				Extras: map[string]interface{}{"path": p2},
			})
		}
		for i, v := range exp.VarNames {
			desc := v
			if len(cfg.ExtraPathDocs) > i {
				desc = cfg.ExtraPathDocs[i]
			}
			builder = builder.Param(ws.PathParameter(v, desc))
		}
	}

	if cfg.ID != "" {
		ws.Container.routeID[mapKey] = cfg.ID
	}

	if _, ok := ws.errors[mapKey]; !ok {
		ws.errors[mapKey] = make(map[int]string)
	}
	for k, v := range cfg.Errors {
		ws.errors[mapKey][k] = v
		builder = builder.Returns(k, v, nil)
	}

	if cfg.Auth {
		builder = builder.Metadata("jwt", true)
	}

	builder.Filter(Filter(func(ctx box.Ctx) {
		ctx.Next()
		code, ok := ctx.Attribute(box.BiuAttrErrCode).(int)
		if !ok || code == 0 {
			return
		}
		msg, ok := ws.errors[ctx.RouteSignature()][code]
		if !ok {
			return
		}
		ctx.SetAttribute(box.BiuAttrErrMsg, msg)
	}))

	ws.WebService.Route(builder)
}

func addService(
	prefix string,
	opts opt.ServicesFuncArr,
	container *Container,
	wss ...NS,
) {
	for _, v := range wss {
		// build web service
		ws := new(restful.WebService)
		path := prefix + "/" + v.NameSpace
		ws.Path(path).Produces(restful.MIME_JSON)

		cfg := &opt.Services{}
		for _, f := range opts {
			f(cfg)
		}
		for _, f := range cfg.Filters {
			ws.Filter(f)
		}
		for k, v := range cfg.Errors {
			container.errors[k] = v
		}

		v.Controller.WebService(WS{
			WebService: ws,
			Container:  container,
			errors:     make(map[string]map[int]string),
		})
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
		container.swaggerTags[container.ServeMux] = append(container.swaggerTags[container.ServeMux], spec.Tag{
			TagProps: tagProps,
		})
		routes := ws.Routes()
		for ri, r := range routes {
			container.logger.Info(log.BiuInternalInfo{Extras: map[string]interface{}{
				"PATH":   r.Path,
				"METHOD": r.Method,
			}})
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
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(3 * time.Minute)
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

func run(addr string, c *Container, opts ...opt.RunFunc) {
	nCtx, nCancel := context.WithCancel(context.Background())
	cfg := &opt.Run{
		BeforeShutDown: func() {},
		AfterShutDown:  func() {},
		Ctx:            nCtx,
		Cancel:         nCancel,
	}
	for _, f := range opts {
		if f != nil {
			f(cfg)
		}
	}

	server := &http.Server{
		Addr:    addr,
		Handler: c,
	}
	addrChan := make(chan string)

	go func() {
		c.logger.Info(log.BiuInternalInfo{
			Err: xerrors.Errorf("listen and serve: %w", ListenAndServe(server, addrChan)),
		})
		if cfg.Cancel != nil {
			cfg.Cancel()
		}
	}()
	select {
	case addr := <-addrChan:
		c.logger.Info(log.BiuInternalInfo{
			Extras: map[string]interface{}{
				"Listening Addr": addr,
			},
		})
	case <-time.After(time.Second):
		c.logger.Fatal(log.BiuInternalInfo{
			Err: xerrors.New("start server timeout"),
		})
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	c.logger.Info(log.BiuInternalInfo{
		Extras: map[string]interface{}{
			"Received Signal": <-ch,
		},
	})

	cfg.BeforeShutDown()
	c.logger.Info(server.Shutdown(cfg.Ctx))
	<-cfg.Ctx.Done()
	cfg.AfterShutDown()
}

// TestServer wraps a httptest.Server
type TestServer struct {
	*httptest.Server
}

// WithT accept testing.T and returns httpexpect.Expect
func (s *TestServer) WithT(t *testing.T) *httpexpect.Expect {
	return httpexpect.New(t, s.URL)
}

// LogFilter logs
// 	{
// 		remote_addr,
// 		method,
// 		uri,
// 		proto,
// 		status_code,
// 		content_length,
// 	}
// for each request
func LogFilter() restful.FilterFunction {
	return DefaultContainer.FilterFunc(func(ctx box.Ctx) {
		start := time.Now()
		ctx.Next()
		ctx.Logger.Info(log.BiuInternalInfo{
			Extras: map[string]interface{}{
				"remote_addr":    ctx.IP(),
				"method":         ctx.Req().Method,
				"uri":            ctx.Req().URL.RequestURI(),
				"proto":          ctx.Req().Proto,
				"status_code":    ctx.Response.StatusCode(),
				"dur":            time.Since(start),
				"content_length": ctx.Response.ContentLength(),
			},
		})
	})
}

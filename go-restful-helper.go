package biu

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"

	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/internal"
	"github.com/tuotoo/biu/internal/cert"
	"github.com/tuotoo/biu/opt"
)

const (
	// MIME_HTML_FORM is application/x-www-form-urlencoded header
	MIME_HTML_FORM = "application/x-www-form-urlencoded"
	// MIME_FILE_FORM is multipart/form-data
	MIME_FILE_FORM = "multipart/form-data"
)

var AutoGenPathDoc = true

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
		builder = builder.Operation(internal.NameOfFunction(cfg.To))
	}

	elm := reflect.ValueOf(builder).Elem()

	p1 := elm.FieldByName("rootPath").String()
	p2 := elm.FieldByName("currentPath").String()
	routePath := path.Join(p1, "/")
	if ws.namespace != "" {
		routePath = path.Join(routePath, ws.namespace)
		builder.Path(ws.namespace + p2)
	}
	routePath = path.Join(routePath, p2)
	method := elm.FieldByName("httpMethod").String()
	mapKey := routePath + " " + method

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
			builder = builder.Returns(200, v.Desc, v.Return)
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
		exp, err := internal.NewPathExpression(p2)
		if err != nil {
			ws.Container.logger.Error("invalid routePath",
				slog.Any("err", err),
				slog.String("routePath", p2),
			)
			os.Exit(1)
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

	builder.Filter(FilterWithLogger(func(ctx box.Ctx) {
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
	}, ws.Container.logger))

	ws.WebService.Route(builder)
}

func addService(
	prefix string,
	opts opt.ServicesFuncArr,
	container *Container,
	wss ...NS,
) {
	expr, err := internal.NewPathExpression(prefix)
	if err != nil {
		panic(err)
	}
	var inCommonNS bool
	if expr.VarCount > 0 {
		inCommonNS = true
	}
	cfg := &opt.Services{}
	for _, f := range opts {
		f(cfg)
	}
	maps.Copy(container.errors, cfg.Errors)
	commonWS := container.NewWS()
	commonWS.Path(prefix).Produces(restful.MIME_JSON)
	var filterAdded bool
	for _, v := range wss {
		// build web service
		ws := container.NewWS()
		wsPath := path.Join("/", prefix, v.NameSpace)
		ws.Path(wsPath).Produces(restful.MIME_JSON)
		if inCommonNS {
			ws = commonWS
			ws.namespace = v.NameSpace
		}

		if (inCommonNS && !filterAdded) || !inCommonNS {
			for _, f := range cfg.Filters {
				ws.Filter(f)
			}
			filterAdded = true
		}

		v.Controller.WebService(ws)
		if !inCommonNS {
			container.Add(ws.WebService)
		}

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
			if routes[ri].Metadata == nil {
				routes[ri].Metadata = make(map[string]any)
			}
			if len(routes[ri].Consumes) == 0 {
				if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
					r.Consumes = []string{MIME_HTML_FORM}
				} else {
					r.Consumes = []string{restful.MIME_JSON}
				}
			}
			if strings.HasPrefix(path.Join(r.Path, "/"), path.Join(wsPath, "/")) {
				container.logger.Debug("route",
					slog.String("PATH", r.Path),
					slog.String("METHOD", r.Method),
				)
				routes[ri].Metadata[restfulspec.KeyOpenAPITags] = []string{v.NameSpace}
			}
		}
	}
	if inCommonNS {
		container.Add(commonWS.WebService)
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
	srv.Addr = tcpListener.Addr().String()
	return srv.Serve(tcpKeepAliveListener{TCPListener: tcpListener})
}

func run(addr string, c *Container, opts ...opt.RunFunc) {
	nCtx, nCancel := context.WithCancel(context.Background())
	cfg := &opt.Run{
		AfterStart:     func() {},
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

	// TLS mode: delegate to runTLS
	if cfg.TLS != nil {
		runTLS(addr, c, cfg)
		return
	}

	// HTTP mode (existing behavior, unchanged)
	c.Server = &http.Server{
		Addr:    addr,
		Handler: c,
	}
	addrChan := make(chan string)

	go func() {
		c.logger.Debug("listen and serve", slog.Any("err", ListenAndServe(c.Server, addrChan)))
		if cfg.Cancel != nil {
			cfg.Cancel()
		}
	}()
	select {
	case addr := <-addrChan:
		c.logger.Debug("listen", slog.String("addr", addr))
		cfg.AfterStart()
	case <-time.After(time.Second):
		c.logger.Error("start server timeout")
		os.Exit(1)
	}
	if cfg.Pprof != nil {
		pprofSrv, pprofAddrChan := startPprof(cfg.Pprof, c.logger)
		select {
		case paddr := <-pprofAddrChan:
			c.logger.Debug("pprof server", slog.String("addr", paddr))
		case <-time.After(time.Second):
			c.logger.Error("start pprof server timeout")
			os.Exit(1)
		}
		userBeforeShutDown := cfg.BeforeShutDown
		cfg.BeforeShutDown = func() {
			userBeforeShutDown()
			c.logger.Debug("pprof shutdown", slog.Any("err", pprofSrv.Shutdown(cfg.Ctx)))
		}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	c.logger.Debug("Received Signal", slog.Any("signal", <-ch))

	cfg.BeforeShutDown()
	c.logger.Debug("Server Shutdown", slog.Any("err", c.Server.Shutdown(cfg.Ctx)))
	<-cfg.Ctx.Done()
	cfg.AfterShutDown()
}

// runTLS starts the container with automatic TLS via Let's Encrypt / ZeroSSL.
func runTLS(addr string, c *Container, cfg *opt.Run) {
	// Detect IP if auto-detection requested
	if cfg.TLS.Mode == opt.TLSModeIP && cfg.TLS.IP == "" {
		ip, err := cert.DetectPublicIP("")
		if err != nil {
			c.logger.Error("auto-detect public IP failed", slog.Any("err", err))
			os.Exit(1)
		}
		cfg.TLS.IP = ip
		c.logger.Info("auto-detected public IP", slog.String("ip", ip))
	}

	// Pass logger to TLS config
	cfg.TLS.Logger = c.logger

	// Create cert manager (loads from cache or obtains new)
	cm, err := cert.NewCertManager(*cfg.TLS)
	if err != nil {
		c.logger.Error("create cert manager", slog.Any("err", err))
		os.Exit(1)
	}

	// Start auto-renewal
	cm.StartAutoRenew(cfg.Ctx)
	defer func() {
		if err := cm.Shutdown(cfg.Ctx); err != nil {
			c.logger.Error("cert manager shutdown", slog.Any("err", err))
		}
	}()

	// Set up TLS server
	c.Server = &http.Server{
		Addr:      addr,
		Handler:   c,
		TLSConfig: cm.TLSConfig(),
	}

	addrChan := make(chan string)

	go func() {
		c.logger.Debug("listen and serve TLS", slog.Any("err", ListenAndServeTLS(c.Server, addrChan)))
		if cfg.Cancel != nil {
			cfg.Cancel()
		}
	}()

	select {
	case listenAddr := <-addrChan:
		c.logger.Debug("listening", slog.String("addr", listenAddr))
		cfg.AfterStart()
	case <-time.After(time.Second):
		c.logger.Error("start TLS server timeout")
		os.Exit(1)
	}
	if cfg.Pprof != nil {
		pprofSrv, pprofAddrChan := startPprof(cfg.Pprof, c.logger)
		select {
		case paddr := <-pprofAddrChan:
			c.logger.Debug("pprof server", slog.String("addr", paddr))
		case <-time.After(time.Second):
			c.logger.Error("start pprof server timeout")
			os.Exit(1)
		}
		userBeforeShutDown := cfg.BeforeShutDown
		cfg.BeforeShutDown = func() {
			userBeforeShutDown()
			c.logger.Debug("pprof shutdown", slog.Any("err", pprofSrv.Shutdown(cfg.Ctx)))
		}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	c.logger.Debug("Received Signal", slog.Any("signal", <-ch))

	cfg.BeforeShutDown()
	c.logger.Debug("TLS Server Shutdown", slog.Any("err", c.Server.Shutdown(cfg.Ctx)))
	<-cfg.Ctx.Done()
	cfg.AfterShutDown()
}

// ListenAndServeTLS listens on TCP and serves TLS with the server's TLSConfig.
func ListenAndServeTLS(srv *http.Server, addrChan chan<- string) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen TLS: %w", err)
	}
	tcpListener := ln.(*net.TCPListener)
	addrChan <- tcpListener.Addr().String()
	srv.Addr = tcpListener.Addr().String()

	return srv.ServeTLS(tcpKeepAliveListener{TCPListener: tcpListener}, "", "")
}

// TestServer wraps a httptest.Server
type TestServer struct {
	*httptest.Server
}

// LogFilter logs
//
//	{
//		remote_addr,
//		method,
//		uri,
//		proto,
//		status_code,
//		content_length,
//	}
//
// for each request
func LogFilter(c ...*Container) restful.FilterFunction {
	if len(c) == 0 {
		c = []*Container{DefaultContainer}
	}
	return c[0].FilterFunc(func(ctx box.Ctx) {
		start := time.Now()
		ctx.Next()
		ctx.Logger.Info("request",
			slog.String("remote_addr", ctx.IP()),
			slog.String("method", ctx.Req().Method),
			slog.String("uri", ctx.Req().URL.RequestURI()),
			slog.String("proto", ctx.Req().Proto),
			slog.Int("status_code", ctx.Response.StatusCode()),
			slog.Duration("dur", time.Since(start)),
			slog.Int("content_length", ctx.Response.ContentLength()),
		)
	})
}

package biu

import "github.com/levigross/grequests"

// CT is a controller testing object.
type CT struct {
	ctlFuncs CtlFuncs
	ctl      CtlInterface
}

// NewCT creates a controller testing object.
func NewCT(ctl CtlInterface) *CT {
	return &CT{ctl: ctl, ctlFuncs: GetCtlFuncs(ctl)}
}

// Handler creates a request for testing handler.
func (t CT) Handler(method, path string,
	option *grequests.RequestOptions) (*grequests.Response, error) {
	return reqPathHandler(t.ctlFuncs, method, path, path, option)
}

// PathHandler creates a request for testing handler with path parameter.
func (t CT) PathHandler(method, path string, reqPath string,
	option *grequests.RequestOptions) (*grequests.Response, error) {
	return reqPathHandler(t.ctlFuncs, method, path, reqPath, option)
}

func reqPathHandler(ctlFuncs CtlFuncs, method, path, reqPath string,
	option *grequests.RequestOptions) (*grequests.Response, error) {
	testServer := ctlFuncs.NewTestServer(method, path)
	defer testServer.Close()
	return grequests.Req(method, testServer.URL+reqPath, option)
}

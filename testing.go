package biu

import "github.com/levigross/grequests"

type CT struct {
	ctl CtlInterface
}

func NewCT(ctl CtlInterface) *CT {
	return &CT{ctl: ctl}
}

func (t CT) Handler(method, path string,
	option *grequests.RequestOptions) (*grequests.Response, error) {
	return reqPathHandler(t.ctl, method, path, path, option)
}

func (t CT) PathHandler(method, path string, reqPath string,
	option *grequests.RequestOptions) (*grequests.Response, error) {
	return reqPathHandler(t.ctl, method, path, reqPath, option)
}

func reqPathHandler(ctl CtlInterface, method, path, reqPath string,
	option *grequests.RequestOptions) (*grequests.Response, error) {
	testServer := GetCtlFuncs(ctl).NewTestServer(method, path)
	defer testServer.Close()
	return grequests.Req(method, testServer.URL+reqPath, option)
}

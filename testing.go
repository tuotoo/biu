package biu

import (
	"encoding/json"
	"testing"

	"github.com/json-iterator/go"
	"github.com/levigross/grequests"
	"github.com/stretchr/testify/assert"
)

// CT is a controller testing object.
type CT struct {
	ctlFuncs CtlFuncs
	ctl      CtlInterface
}

// CommonRespRawData is a common response with json.RawMessage
type CommonRespRawData struct {
	Code    int
	Message string
	Data    json.RawMessage
}

// NewCT creates a controller testing object.
func NewCT(ctl CtlInterface) *CT {
	return &CT{ctl: ctl, ctlFuncs: GetCtlFuncs(ctl)}
}

// Handler creates a request for testing handler.
func (ct CT) Handler(method, path string,
	option *grequests.RequestOptions) (*grequests.Response, error) {
	return reqPathHandler(ct.ctlFuncs, method, path, path, option)
}

// AssertHandler requests a handler with error assert
// and returns a CommonRespRawData.
func (ct CT) AssertHandler(t *testing.T, method, path string,
	option *grequests.RequestOptions) *CommonRespRawData {
	resp, err := reqPathHandler(ct.ctlFuncs, method, path, path, option)
	assert.Nil(t, err)
	crt := &CommonRespRawData{}
	err = resp.JSON(crt)
	assert.Nil(t, err)
	return crt
}

// PathHandler creates a request for testing handler with path parameter.
func (ct CT) PathHandler(method, path string, reqPath string,
	option *grequests.RequestOptions) (*grequests.Response, error) {
	return reqPathHandler(ct.ctlFuncs, method, path, reqPath, option)
}

// AssertPathHandler requests a path handler with error assert
// and returns a CommonRespRawData.
func (ct CT) AssertPathHandler(t *testing.T, method, path string, reqPath string,
	option *grequests.RequestOptions) *CommonRespRawData {
	resp, err := reqPathHandler(ct.ctlFuncs, method, path, reqPath, option)
	assert.Nil(t, err)
	crt := &CommonRespRawData{}
	err = resp.JSON(crt)
	assert.Nil(t, err)
	return crt
}

func AssertJSON(t *testing.T, data []byte, v interface{}) {
	err := jsoniter.Unmarshal(data, v)
	assert.Nil(t, err)
}

func reqPathHandler(ctlFuncs CtlFuncs, method, path, reqPath string,
	option *grequests.RequestOptions) (*grequests.Response, error) {
	testServer := ctlFuncs.NewTestServer(method, path)
	defer testServer.Close()
	return grequests.Req(method, testServer.URL+reqPath, option)
}

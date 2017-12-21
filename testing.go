package biu

import (
	"encoding/json"
	"errors"
	"strings"
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
	option *grequests.RequestOptions, v ...string) (*grequests.Response, error) {
	return reqHandler(ct.ctlFuncs, method, path, option, v...)
}

// AssertHandler requests a handler with error assert
// and returns a CommonRespRawData.
func (ct CT) AssertHandler(t *testing.T, method, path string,
	option *grequests.RequestOptions, v ...string) *CommonRespRawData {
	resp, err := reqHandler(ct.ctlFuncs, method, path, option, v...)
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

func reqHandler(ctlFuncs CtlFuncs, method, path string,
	option *grequests.RequestOptions, v ...string) (*grequests.Response, error) {
	testServer := ctlFuncs.NewTestServer(method, path)
	defer testServer.Close()
	i := 0
	for {
		li := strings.Index(path, "{")
		if li < 0 {
			break
		}
		ri := strings.Index(path, "}")
		if ri < 0 {
			break
		}
		if len(v) < i+1 {
			return nil, errors.New("not enough arguments")
		}
		path = path[:li] + v[i] + path[ri+1:]
		i++
	}
	return grequests.Req(method, testServer.URL+path, option)
}

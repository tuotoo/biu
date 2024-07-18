package expect

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/stretchr/testify/assert"
)

type Req struct {
	t       assert.TestingT
	baseURL string
	method  string
	headers map[string]string
	path    string
}

type Exp struct {
	t    assert.TestingT
	resp *http.Response
	json json.RawMessage
}

func Default(t assert.TestingT, baseURL string) *Req {
	return &Req{
		t:       t,
		baseURL: baseURL,
		headers: make(map[string]string),
	}
}

func (r *Req) GET(path string) *Req {
	r.method = http.MethodGet
	r.path = path
	return r
}

func (r *Req) POST(path string) *Req {
	r.method = http.MethodPost
	r.path = path
	return r
}

func (r *Req) WithHeader(k, v string) *Req {
	r.headers[k] = v
	return r
}

func (r *Req) Expect() *Exp {
	path, err := url.JoinPath(r.baseURL, r.path)
	assert.NoError(r.t, err)
	req, err := http.NewRequest(r.method, path, nil)
	assert.NoError(r.t, err)
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(r.t, err)
	return &Exp{
		t:    r.t,
		resp: resp,
	}
}

func (e *Exp) JSON() *Exp {
	var body json.RawMessage
	assert.NoError(e.t, json.NewDecoder(e.resp.Body).Decode(&body))
	e.json = body
	return e
}

func (e *Exp) Object() *Exp {
	var obj map[string]json.RawMessage
	assert.NoError(e.t, json.Unmarshal(e.json, &obj))
	assert.Greater(e.t, len(obj), 0)
	return e
}

func (e *Exp) HasValue(key string, value any) *Exp {
	var obj map[string]any
	assert.NoError(e.t, json.Unmarshal(e.json, &obj))
	actual, err := json.Marshal(obj[key])
	assert.NoError(e.t, err)
	expect, err := json.Marshal(value)
	assert.NoError(e.t, err)
	assert.Equal(e.t, expect, actual)
	return e
}

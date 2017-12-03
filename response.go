package biu

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/emicklei/go-restful"
)

var codeDesc struct {
	sync.Once
	m map[int]string
}

func init() {
	codeDesc.Do(func() {
		codeDesc.m = make(map[int]string)
	})
}

func Handle(f func(ctl *Ctx)) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		f(&Ctx{
			Request:  request,
			Response: response,
		})
	}
}

// AddErrDesc adds map of code-message to stdCodeDesc
func AddErrDesc(m map[int]string) {
	for k, v := range m {
		codeDesc.m[k] = v
	}
}

type Ctx struct {
	*restful.Request
	*restful.Response
}

// ResponseJSON is a convenience method
// for writing a value wrap in CommonResp as JSON.
// It uses jsoniter for marshalling the value.
func (ctx *Ctx) ResponseJSON(v interface{}) {
	commonResponse(ctx.Response, CommonResp{Data: v})
}

// ResponseError is a convenience method to response an error code and message.
// It uses jsoniter for marshalling the value.
func (ctx *Ctx) ResponseError(msg string, code int) {
	commonResponse(ctx.Response, CommonResp{Code: code, Message: msg})
}

// ContainsError is a convenience method to check error is nil.
// If error is nil, it will return false,
// else it will log the error, make a CommonResp response and return true.
// if code is 0, it will use err.Error() as CommonResp.message.
func (ctx *Ctx) ContainsError(err error, code int) bool {
	msg := codeDesc.m[code]
	if CheckError(err, Log().Int("code", code).Str("msg", msg)) {
		return false
	}
	if code == 0 {
		msg = err.Error()
	}
	ResponseError(ctx.Response, msg, code)
	return true
}

// ResponseStdErrCode is a convenience method response a code with msg in Code Desc.
func (ctx *Ctx) ResponseStdErrCode(code int) {
	msg := codeDesc.m[code]
	ResponseError(ctx.Response, msg, code)
}

// ResponseJSON is a convenience method
// for writing a value wrap in CommonResp as JSON.
// It uses jsoniter for marshalling the value.
func ResponseJSON(w http.ResponseWriter, v interface{}) {
	commonResponse(w, CommonResp{Data: v})
}

// ResponseError is a convenience method to response an error code and message.
// It uses jsoniter for marshalling the value.
func ResponseError(w http.ResponseWriter, msg string, code int) {
	commonResponse(w, CommonResp{Code: code, Message: msg})
}

// ContainsError is a convenience method to check error is nil.
// If error is nil, it will return false,
// else it will log the error, make a CommonResp response and return true.
// if code is 0, it will use err.Error() as CommonResp.message.
func ContainsError(w http.ResponseWriter, err error, code int) bool {
	msg := codeDesc.m[code]
	if CheckError(err, Log().Int("code", code).Str("msg", msg)) {
		return false
	}
	if code == 0 {
		msg = err.Error()
	}
	ResponseError(w, msg, code)
	return true
}

// CheckError is a convenience method to check error is nil.
// If error is nil, it will return true,
// else it will log the error and return false
func CheckError(err error, log *LogEvt) bool {
	if err == nil {
		return true
	}
	if log != nil {
		Info("verify error", log.Err(err))
	}
	return false
}

func commonResponse(w http.ResponseWriter, resp CommonResp) {
	w.Header().Set(restful.HEADER_ContentType, restful.MIME_JSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		Warn("json encode", Log().Err(err))
	}
}

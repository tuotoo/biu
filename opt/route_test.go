package opt_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"

	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/opt"
)

func TestRouteTo(t *testing.T) {
	cfg := &opt.Route{}
	a := 1
	opt.RouteTo(func(ctx box.Ctx) {
		a = 2
	})(cfg)
	cfg.To(box.Ctx{})
	assert.Equal(t, 2, a)
}

func TestDisableAuthPathDoc(t *testing.T) {
	cfg := &opt.Route{EnableAutoPathDoc: true}
	opt.DisableAuthPathDoc()(cfg)
	assert.Equal(t, false, cfg.EnableAutoPathDoc)
}

func TestEnableAuth(t *testing.T) {
	cfg := &opt.Route{}
	opt.EnableAuth()(cfg)
	assert.Equal(t, true, cfg.Auth)
}

func TestExtraPathDocs(t *testing.T) {
	cfg := &opt.Route{}
	opt.ExtraPathDocs("1", "2")(cfg)
	assert.Equal(t, []string{"1", "2"}, cfg.ExtraPathDocs)
}

func TestRouteErrors(t *testing.T) {
	cfg := &opt.Route{}
	opt.RouteErrors(map[int]string{1: "2"})(cfg)
	assert.Contains(t, cfg.Errors, 1)
	assert.Equal(t, "2", cfg.Errors[1])
}

func TestRouteID(t *testing.T) {
	cfg := &opt.Route{}
	opt.RouteID("routeID")(cfg)
	assert.Equal(t, "routeID", cfg.ID)
}

func TestRouteAPI(t *testing.T) {
	for _, v := range []struct {
		req     func() *restful.Request
		routAPI any
	}{
		{
			req: func() *restful.Request {
				return restful.NewRequest(httptest.NewRequest(http.MethodPost, "/", strings.NewReader("1")))
			},
			routAPI: func(ctx box.Ctx, api struct {
				Body string
			}) {
				assert.Equal(t, "1", api.Body)
			},
		},
		{
			req: func() *restful.Request {
				return restful.NewRequest(httptest.NewRequest(http.MethodPost, "/?a=1", nil))
			},
			routAPI: func(ctx box.Ctx) {
				assert.Equal(t, 1, ctx.Query("a").IntDefault(0))
			},
		},
		{
			req: func() *restful.Request {
				return restful.NewRequest(httptest.NewRequest(http.MethodPost, "/", strings.NewReader("1")))
			},
			routAPI: func(ctx box.Ctx, api struct {
				Body *string
			}) {
				assert.Equal(t, "1", *api.Body)
			},
		},
		{
			req: func() *restful.Request {
				req := restful.NewRequest(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":1}`)))
				req.Request.Header.Add("Content-Type", "application/json")
				return req
			},
			routAPI: func(ctx box.Ctx, api struct {
				Body struct {
					A int
				}
			}) {
				assert.Equal(t, 1, api.Body.A)
			},
		},
		{
			req: func() *restful.Request {
				req := restful.NewRequest(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":["1","2"]}`)))
				req.Request.Header.Add("Content-Type", "application/json")
				return req
			},
			routAPI: func(ctx box.Ctx, api struct {
				Body struct {
					A []string
				}
			}) {
				assert.EqualValues(t, []string{"1", "2"}, api.Body.A)
			},
		},
		{
			req: func() *restful.Request {
				return restful.NewRequest(httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body")))
			},
			routAPI: func(ctx box.Ctx, api struct {
				Body io.Reader
			}) {
				bs, err := io.ReadAll(api.Body)
				assert.NoError(t, err)
				assert.Equal(t, "body", string(bs))
			},
		},
		{
			req: func() *restful.Request {
				return restful.NewRequest(httptest.NewRequest(http.MethodPost, "/?a=1", nil))
			},
			routAPI: func(ctx box.Ctx, api struct {
				Query struct {
					A int
				}
			}) {
				assert.Equal(t, 1, api.Query.A)
			},
		},
		{
			req: func() *restful.Request {
				req := restful.NewRequest(httptest.NewRequest(http.MethodPost, "/", nil))
				req.Request.Header.Set("A", "1")
				return req
			},
			routAPI: func(ctx box.Ctx, api struct {
				Header struct {
					A int
				}
			}) {
				assert.Equal(t, 1, api.Header.A)
			},
		},
		{
			req: func() *restful.Request {
				req := restful.NewRequest(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`a=1`)))
				req.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			routAPI: func(ctx box.Ctx, api struct {
				Form struct {
					A int
				}
			}) {
				assert.Equal(t, 1, api.Form.A)
			},
		},
		{
			req: func() *restful.Request {
				form := url.Values{
					"time": []string{
						time.Now().Format(time.RFC3339),
					},
					"arr_time": []string{
						time.Now().Format(time.RFC3339),
						time.Now().AddDate(0, 0, 1).Format(time.RFC3339),
					},
					"bool": []string{
						"true",
					},
					"f64": []string{
						"1.23",
					},
					"f32": []string{
						"1.23",
					},
					"int": []string{
						"1",
					},
					"i8": []string{
						"1",
					},
					"i16": []string{
						"1",
					},
					"i32": []string{
						"1",
					},
					"i64": []string{
						"1",
					},
					"uint": []string{
						"1",
					},
					"u32": []string{
						"1",
					},
					"u64": []string{
						"1",
					},
					"string": []string{
						"1",
					},
					"arr_int": []string{
						"1", "2", "3",
					},
					"arr_string": []string{
						"1", "2", "3",
					},
					"arr_bool": []string{
						"true", "false", "true",
					},
					"arr_f64": []string{
						"1.23", "1.23", "1.23",
					},
					"arr_f32": []string{
						"1.23", "1.23", "1.23",
					},
					"arr_i8": []string{
						"1", "2", "3",
					},
					"arr_i16": []string{
						"1", "2", "3",
					},
					"arr_i32": []string{
						"1", "2", "3",
					},
					"arr_i64": []string{
						"1", "2", "3",
					},
					"arr_u16": []string{
						"1", "2", "3",
					},
					"arr_u32": []string{
						"1", "2", "3",
					},
					"arr_u64": []string{
						"1", "2", "3",
					},
					"arr_uint": []string{
						"1", "2", "3",
					},
					"p_bool": []string{
						"true",
					},
					"p_f64": []string{
						"1.23",
					},
					"p_f32": []string{
						"1.23",
					},
					"p_i8": []string{
						"1",
					},
					"p_i16": []string{
						"1",
					},
					"p_i32": []string{
						"1",
					},
					"p_i64": []string{
						"1",
					},
					"p_u8": []string{
						"1",
					},
					"p_u16": []string{
						"1",
					},
					"p_u32": []string{
						"1",
					},
					"p_u64": []string{
						"1",
					},
					"p_int": []string{
						"1",
					},
					"p_uint": []string{
						"1",
					},
					"bytes": []string{
						"hey",
					},
					"ignore": []string{
						"1",
					},
				}
				req := restful.NewRequest(httptest.NewRequest(http.MethodPost, "/",
					strings.NewReader(form.Encode()),
				))
				req.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			routAPI: func(ctx box.Ctx, api struct {
				Form struct {
					Time      time.Time
					ArrTime   []time.Time
					Bool      bool
					F64       float64
					F32       float32
					Int       int
					I8        int8
					I16       int16
					I32       int32
					I64       int64
					Uint      uint
					U32       uint32
					U64       uint64
					String    string
					ArrInt    []int
					ArrString []string
					ArrBool   []bool
					ArrF64    []float64
					ArrF32    []float32
					ArrI8     []int8
					ArrI16    []int16
					ArrI32    []int32
					ArrI64    []int64
					ArrU16    []uint16
					ArrU32    []uint32
					ArrU64    []uint64
					ArrUint   []uint
					PBool     *bool
					PInt      *int
					PUint     *uint
					PF64      *float64
					PF32      *float32
					PI8       *int8   `biu:"name:p_i8"`
					PI16      *int16  `biu:"name:p_i16"`
					PI32      *int32  `biu:"name:p_i32"`
					PI64      *int64  `biu:"name:p_i64"`
					PU8       *uint8  `biu:"name:p_u8"`
					PU16      *uint16 `biu:"name:p_u16"`
					PU32      *uint32 `biu:"name:p_u32"`
					PU64      *uint64 `biu:"name:p_u64"`
					Bytes     []byte
					Ignore    string `biu:"-"`
				}
				Return func(string)
			}) {
				assert.Equal(t, time.Now().Format(time.DateOnly), api.Form.Time.Format(time.DateOnly))
				arrDate := make([]string, len(api.Form.ArrTime))
				for i, v := range api.Form.ArrTime {
					arrDate[i] = v.Format(time.DateOnly)
				}
				assert.Len(t, arrDate, 2)
				assert.Equal(t, []string{time.Now().Format(time.DateOnly), time.Now().AddDate(0, 0, 1).Format(time.DateOnly)}, arrDate)
				assert.Equal(t, true, api.Form.Bool)
				assert.Equal(t, 1.23, api.Form.F64)
				assert.Equal(t, float32(1.23), api.Form.F32)
				assert.Equal(t, 1, api.Form.Int)
				assert.Equal(t, int8(1), api.Form.I8)
				assert.Equal(t, int16(1), api.Form.I16)
				assert.Equal(t, int32(1), api.Form.I32)
				assert.Equal(t, int64(1), api.Form.I64)
				assert.Equal(t, uint(1), api.Form.Uint)
				assert.Equal(t, uint32(1), api.Form.U32)
				assert.Equal(t, uint64(1), api.Form.U64)
				assert.Equal(t, "1", api.Form.String)
				assert.Equal(t, []int{1, 2, 3}, api.Form.ArrInt)
				assert.Equal(t, []string{"1", "2", "3"}, api.Form.ArrString)
				assert.Equal(t, []bool{true, false, true}, api.Form.ArrBool)
				assert.Equal(t, []float64{1.23, 1.23, 1.23}, api.Form.ArrF64)
				assert.Equal(t, []float32{1.23, 1.23, 1.23}, api.Form.ArrF32)
				assert.Equal(t, []int8{1, 2, 3}, api.Form.ArrI8)
				assert.Equal(t, []int16{1, 2, 3}, api.Form.ArrI16)
				assert.Equal(t, []int32{1, 2, 3}, api.Form.ArrI32)
				assert.Equal(t, []int64{1, 2, 3}, api.Form.ArrI64)
				assert.Equal(t, []uint16{1, 2, 3}, api.Form.ArrU16)
				assert.Equal(t, []uint32{1, 2, 3}, api.Form.ArrU32)
				assert.Equal(t, []uint64{1, 2, 3}, api.Form.ArrU64)
				assert.Equal(t, []uint{1, 2, 3}, api.Form.ArrUint)
				assert.Equal(t, true, *api.Form.PBool)
				assert.Equal(t, int8(1), *api.Form.PI8)
				assert.Equal(t, int16(1), *api.Form.PI16)
				assert.Equal(t, int32(1), *api.Form.PI32)
				assert.Equal(t, int64(1), *api.Form.PI64)
				assert.Equal(t, uint8(1), *api.Form.PU8)
				assert.Equal(t, uint16(1), *api.Form.PU16)
				assert.Equal(t, uint32(1), *api.Form.PU32)
				assert.Equal(t, uint64(1), *api.Form.PU64)
				assert.Equal(t, int(1), *api.Form.PInt)
				assert.Equal(t, uint(1), *api.Form.PUint)
				assert.Equal(t, "hey", string(api.Form.Bytes))
				assert.Equal(t, "", api.Form.Ignore)
				api.Return("1")
			},
		},
		{
			req: func() *restful.Request {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				fw, err := w.CreateFormFile("a", "filename")
				assert.NoError(t, err)
				_, err = fw.Write([]byte("123"))
				assert.NoError(t, err)
				assert.NoError(t, w.Close())

				req := restful.NewRequest(httptest.NewRequest(http.MethodPost, "/", &b))
				req.Request.Header.Set("Content-Type", w.FormDataContentType())
				return req
			},
			routAPI: func(ctx box.Ctx, api struct {
				Form struct {
					A box.File
				}
			}) {
				assert.Equal(t, "filename", api.Form.A.Header.Filename)
				bs, err := io.ReadAll(api.Form.A.File)
				assert.NoError(t, err)
				assert.Equal(t, "123", string(bs))
			},
		},
	} {
		cfg := &opt.Route{}
		opt.RouteAPI(v.routAPI)(cfg)
		cfg.To(box.Ctx{
			Request: v.req(),
		})
	}
}

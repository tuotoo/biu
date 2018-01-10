package biu_test

import (
	"testing"

	"github.com/levigross/grequests"
	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/biu"
)

var fooCT = biu.NewCT(Foo{})

var _ func(t *testing.T)

func ExampleNewCT() {
	_ = func(t *testing.T) {
		req := func(num string, code int) (rst Bar) {
			resp := fooCT.AssertHandler(t, "GET", "/",
				&grequests.RequestOptions{
					QueryStruct: struct {
						Num string `url:"num"`
					}{
						Num: num,
					},
				})
			assert.Equal(t, code, resp.Code)
			biu.AssertJSON(t, resp.Data, &rst)
			return rst
		}
		req("abc", 100)
		resp := req("123", 0)
		assert.Equal(t, "bar", resp.Msg)
		assert.Equal(t, 123, resp.Num)
	}
}

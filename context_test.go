package biu_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/gavv/httpexpect"
	"github.com/tuotoo/biu"
)

func TestCtx_Must(t *testing.T) {
	var funcOK bool
	ws := biu.WS{WebService: &restful.WebService{}}
	ws.Route(ws.GET("/{id}"), &biu.RouteOpt{
		To: func(ctx biu.Ctx) {
			i := ctx.Path("id").IntDefault(1)
			switch i {
			case 1:
				ctx.Must(errors.New("1"), 1)
			case 2:
				ctx.Must(errors.New("2"), 2, "OK")
			case 3:
				ctx.Must(errors.New("3"), 3, func() {
					funcOK = true
				})
			case 4:
				ctx.Must(errors.New("4"), 4, func() {
					funcOK = true
				}, "OK")
			}
		},
		Errors: map[int]string{
			1: "normal err",
			2: "with arg %s",
			3: "with func",
			4: "with func and arg %s",
		},
	})
	c := restful.NewContainer()
	c.Add(ws.WebService)
	s := httptest.NewServer(c)
	defer s.Close()
	biu.UseConsoleLogger()
	httpexpect.New(t, s.URL).GET("/1").Expect().JSON().Object().
		ValueEqual("code", 1).ValueEqual("message", "normal err")
	httpexpect.New(t, s.URL).GET("/2").Expect().JSON().Object().
		ValueEqual("code", 2).ValueEqual("message", "with arg OK")
	if funcOK {
		t.Errorf("expect func ok %t but got %t", false, funcOK)
	}
	httpexpect.New(t, s.URL).GET("/3").Expect().JSON().Object().
		ValueEqual("code", 3).ValueEqual("message", "with func")
	if !funcOK {
		t.Errorf("expect func ok %t but got %t", true, funcOK)
	}
	funcOK = false
	httpexpect.New(t, s.URL).GET("/4").Expect().JSON().Object().
		ValueEqual("code", 4).ValueEqual("message", "with func and arg OK")
	if !funcOK {
		t.Errorf("expect func ok %t but got %t", true, funcOK)
	}
}

package biu_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/gavv/httpexpect"
	"github.com/tuotoo/biu"
	"github.com/tuotoo/biu/ctx"
	"github.com/tuotoo/biu/log"
	"github.com/tuotoo/biu/opt"
)

func TestCtx_Must(t *testing.T) {
	var tmpValue int
	ws := biu.WS{WebService: &restful.WebService{}}
	ws.Route(ws.GET("/must/{id}"),
		opt.RouteID("test.must"),
		opt.RouteTo(func(ctx ctx.Ctx) {
			i := ctx.Path("id").IntDefault(1)
			switch i {
			case 1:
				ctx.Must(errors.New("1"), 1)
			case 2:
				ctx.Must(errors.New("2"), 2, "OK")
			case 3:
				ctx.Must(errors.New("3"), 3, func() {
					tmpValue = 1
				})
			case 4:
				ctx.Must(errors.New("4"), 4, "OK", func() {
					tmpValue = 2
				})
			}
		}),
		opt.RouteErrors(map[int]string{
			1: "normal err",
			2: "with arg %s",
			3: "with func",
			4: "with func and arg %s",
		}),
	)
	c := biu.New()
	c.Add(ws.WebService)
	s := httptest.NewServer(c)
	defer s.Close()
	log.UseConsoleLogger()
	httpexpect.New(t, s.URL).GET("/must/1").Expect().JSON().Object().
		ValueEqual("code", 1).ValueEqual("message", "normal err")
	httpexpect.New(t, s.URL).GET("/must/2").Expect().JSON().Object().
		ValueEqual("code", 2).ValueEqual("message", "with arg OK")
	if tmpValue != 0 {
		t.Errorf("expect func ok %d but got %d", 0, tmpValue)
	}
	httpexpect.New(t, s.URL).GET("/must/3").Expect().JSON().Object().
		ValueEqual("code", 3).ValueEqual("message", "with func")
	if tmpValue != 1 {
		t.Errorf("expect func ok %d but got %d", 1, tmpValue)
	}
	httpexpect.New(t, s.URL).GET("/must/4").Expect().JSON().Object().
		ValueEqual("code", 4).ValueEqual("message", "with func and arg OK")
	if tmpValue != 2 {
		t.Errorf("expect func ok %d but got %d", 2, tmpValue)
	}
}

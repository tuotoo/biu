package box_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/tuotoo/biu"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/opt"
)

type mustCtl struct {
	tmpValue int
}

func (ctl *mustCtl) WebService(ws biu.WS) {
	ws.Route(ws.GET("/{id}"),
		opt.RouteID("test.must"),
		opt.RouteTo(func(ctx box.Ctx) {
			i := ctx.Path("id").IntDefault(1)
			switch i {
			case 1:
				ctx.Must(errors.New("1"), 1)
			case 2:
				ctx.Must(errors.New("2"), 2, "OK")
			case 3:
				ctx.Must(errors.New("3"), 3, func() {
					ctl.tmpValue = 1
				})
			case 4:
				ctx.Must(errors.New("4"), 4, "OK", func() {
					ctl.tmpValue = 2
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
}

func TestCtx_Must(t *testing.T) {
	c := biu.New()
	ctl := &mustCtl{}
	c.AddServices("", nil, biu.NS{
		NameSpace:  "must",
		Controller: ctl,
	})
	s := httptest.NewServer(c)
	defer s.Close()
	httpexpect.Default(t, s.URL).GET("/must/1").Expect().JSON().Object().
		HasValue("code", 1).HasValue("message", "normal err")
	httpexpect.Default(t, s.URL).GET("/must/2").Expect().JSON().Object().
		HasValue("code", 2).HasValue("message", "with arg OK")
	if ctl.tmpValue != 0 {
		t.Errorf("expect func ok %d but got %d", 0, ctl.tmpValue)
	}
	httpexpect.Default(t, s.URL).GET("/must/3").Expect().JSON().Object().
		HasValue("code", 3).HasValue("message", "with func")
	if ctl.tmpValue != 1 {
		t.Errorf("expect func ok %d but got %d", 1, ctl.tmpValue)
	}
	httpexpect.Default(t, s.URL).GET("/must/4").Expect().JSON().Object().
		HasValue("code", 4).HasValue("message", "with func and arg OK")
	if ctl.tmpValue != 2 {
		t.Errorf("expect func ok %d but got %d", 2, ctl.tmpValue)
	}
}

package biu_test

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gavv/httpexpect"
	"github.com/tuotoo/biu"
)

type test struct{}

func (ctl test) WebService(ws biu.WS) {
	ws.Route(ws.GET("/{id}"), &biu.RouteOpt{
		ID: "A47C8CD0-7528-4283-BDCA-1CD0C9E22B07",
		To: ctl.get,
		Errors: map[int]string{
			100: "err msg in route",
		},
	})
}

func (ctl test) get(ctx biu.Ctx) {
	i := ctx.Path("id").IntDefault(1)
	switch i {
	case 1:
		ctx.Must(errors.New("1"), 101)
	case 100:
		ctx.Must(errors.New("100"), 100)
	}
}

func TestContainer_AddServices(t *testing.T) {
	biu.UseConsoleLogger()
	c := biu.New()
	for _, v := range c.RegisteredWebServices() {
		for _, j := range v.Routes() {
			fmt.Println(j.Path, j.Method)
		}
	}
	c.AddServices("", &biu.GlobalServiceOpt{
		Errors: map[int]string{
			101: "err msg global",
		},
	}, biu.NS{
		Controller: test{},
	})

	s := httptest.NewServer(c)
	defer s.Close()

	httpexpect.New(t, s.URL).GET("/1").Expect().JSON().Object().
		ValueEqual("code", 101).ValueEqual("message", "err msg global")
	httpexpect.New(t, s.URL).GET("/100").Expect().JSON().Object().
		ValueEqual("code", 100).ValueEqual("message", "err msg in route")
}

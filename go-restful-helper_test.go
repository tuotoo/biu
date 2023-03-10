package biu_test

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/biu"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/opt"
)

type test struct{}

func (ctl test) WebService(ws biu.WS) {
	ws.Route(ws.GET("/{id}").Filter(biu.Filter(func(ctx box.Ctx) {
		ctx.Next()
		ctx.Transform(func(i ...interface{}) []interface{} {
			return []interface{}{i[0].(string) + " TRANSFORM " + i[1].(string)}
		})
	})),
		opt.RouteID("test.addService"),
		opt.RouteTo(ctl.get),
		opt.RouteErrors(map[int]string{
			1: "err msg in route",
		}),
	)
}

func (ctl test) get(ctx box.Ctx) {
	i := ctx.Path("id").IntDefault(1)
	switch i {
	case 1:
		ctx.Must(errors.New("1"), 1)
	case 2:
		ctx.Must(errors.New("2"), 2)
	}
	ctx.ResponseJSON("COOL", "COMPLETED")
}

func TestContainer_AddServices(t *testing.T) {
	biu.AutoGenPathDoc = true
	c := biu.New()
	for _, v := range c.RegisteredWebServices() {
		for _, j := range v.Routes() {
			fmt.Println(j.Path, j.Method)
		}
	}
	c.AddServices("", opt.ServicesFuncArr{
		opt.Filters(biu.LogFilter()),
		opt.ServiceErrors(map[int]string{2: "err msg global"}),
	}, biu.NS{
		NameSpace:  "add-service",
		Controller: test{},
	})

	s := httptest.NewServer(c)
	defer s.Close()

	httpexpect.New(t, s.URL).GET("/add-service/1").Expect().JSON().Object().
		ValueEqual("code", 1).ValueEqual("message", "err msg in route")
	httpexpect.New(t, s.URL).GET("/add-service/2").Expect().JSON().Object().
		ValueEqual("code", 2).ValueEqual("message", "err msg global")
	httpexpect.New(t, s.URL).GET("/add-service/3").Expect().JSON().Object().
		ValueEqual("code", 0).ValueEqual("data", "COOL TRANSFORM COMPLETED")
}

type addSrvCtrl struct {
	subPath string
}

func (a addSrvCtrl) WebService(ws biu.WS) {
	ws.Route(ws.GET(a.subPath))
}

func TestAddServices(t *testing.T) {
	table := []struct {
		prefix      string
		namespace   string
		subPath     string
		expectRoute string
	}{
		{
			prefix:      "",
			namespace:   "",
			subPath:     "",
			expectRoute: "/",
		},
		{
			prefix:      "/",
			namespace:   "/",
			subPath:     "/",
			expectRoute: "/",
		},
		{
			prefix:      "",
			namespace:   "/",
			subPath:     "/",
			expectRoute: "/",
		},
		{
			prefix:      "/v",
			namespace:   "/",
			subPath:     "/p",
			expectRoute: "/v/p",
		},
	}
	for _, v := range table {
		c := biu.New()
		c.AddServices(v.prefix, nil, biu.NS{
			NameSpace: v.namespace,
			Controller: addSrvCtrl{
				subPath: v.subPath,
			},
		})
		assert.Equal(t, v.expectRoute, c.RegisteredWebServices()[0].Routes()[0].Path)
	}
}

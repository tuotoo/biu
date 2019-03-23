package main

import (
	"github.com/tuotoo/biu"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/opt"
)

// Foo controller
type Foo struct{}

// WebService implements CtlInterface
func (ctl Foo) WebService(ws biu.WS) {
	ws.Route(ws.GET("/").Doc("Get Bar").
		Param(ws.QueryParameter("num", "number")).
		DefaultReturns("Bar", Bar{}),
		opt.RouteID("example.foo"),
		opt.RouteTo(ctl.getBar),
		opt.RouteErrors(map[int]string{
			200: "%s is not a Number",
		}),
	)

	// add more routes as you like:
	// ws.Route(ws.POST("/foo"),nil)
	// ...
}

// Bar is the response of getBar
type Bar struct {
	Msg string `json:"msg"`
	Num int    `json:"num"`
}

func (ctl Foo) getBar(ctx box.Ctx) {
	num, err := ctx.Query("num").Int()
	if ctx.ContainsError(err, 200, ctx.QueryParameter("num")) {
		return
	}

	ctx.ResponseJSON(Bar{Msg: "bar", Num: num})
}

func main() {
	c := biu.New()
	c.AddServices("/v1", opt.ServicesFuncArr{
		opt.ServiceErrors(map[int]string{
			100: "something goes wrong",
		}),
	},
		biu.NS{
			NameSpace:  "foo",
			Controller: Foo{},
			Desc:       "Foo Controller",
		},
	)
	// Note: you should add swagger service after adding services.
	// swagger document will be available at http://localhost:8080/v1/swagger
	swaggerService := c.NewSwaggerService(biu.SwaggerInfo{
		Title:        "Foo Bar",
		Description:  "Foo Bar Service",
		ContactName:  "tuotoo",
		ContactEmail: "jqs7@tuotoo.com",
		ContactURL:   "https://tuotoo.com",
		Version:      "1.0.0",
		RoutePrefix:  "/v1",
	})
	c.Add(swaggerService)
	c.Run(":8080", nil)
}

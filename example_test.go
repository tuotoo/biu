package biu_test

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
		Param(ws.QueryParameter("num", "number").
			DataType("integer")).
		DefaultReturns("Bar", Bar{}),
		opt.RouteID("example.foo"),
		opt.RouteTo(ctl.getBar),
		opt.RouteErrors(map[int]string{
			100: "num not Number",
		}),
	)

	ws.Route(ws.POST("/").
		Doc("Post Req"),
		opt.RouteID("example.post"),
		opt.RouteAPI(ctl.post),
		opt.RouteErrors(map[int]string{
			100: "num not Number",
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
	ctx.Must(err, 100)

	ctx.ResponseJSON(Bar{Msg: "bar", Num: num})
}

func (ctl Foo) post(ctx box.Ctx, api struct {
	Form struct{ Num int }
}) {
	ctx.ResponseJSON(Bar{Msg: "POST", Num: api.Form.Num})
}

func Example() {
	c := biu.New()
	c.Filter(biu.LogFilter())
	c.AddServices("/v1", nil,
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
		ContactName:  "TuoToo",
		ContactEmail: "jqs7@tuotoo.com",
		ContactURL:   "https://tuotoo.com",
		Version:      "1.0.0",
		RoutePrefix:  "/v1",
	})
	c.Add(swaggerService)
	c.Run(":8080", nil)
	// Output: Listening Addr: [::]:8080
}

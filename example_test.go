package biu_test

import (
	"github.com/emicklei/go-restful"
	"github.com/tuotoo/biu"
)

// Foo controller
type Foo struct{}

// WebService implements CtlInterface
func (ctl Foo) WebService(ws biu.WS) {
	ws.Route(ws.GET("/").Doc("Get Bar").
		Param(ws.QueryParameter("num", "number").DataType("integer")).
		DefaultReturns("Bar", Bar{}), &biu.RouteOpt{
		ID: "2F996F8F-9D08-4BE0-9D4D-ACB328D8F387",
		To: ctl.getBar,
		Errors: map[int]string{
			100: "num not Number",
		},
	})

	// add more routes as you like:
	// ws.Route(ws.POST("/foo"),nil)
	// ...
}

// Bar is the response of getBar
type Bar struct {
	Msg string `json:"msg"`
	Num int    `json:"num"`
}

func (ctl Foo) getBar(ctx biu.Ctx) {
	num, err := ctx.Query("num").Int()
	ctx.Must(err, 100)

	ctx.ResponseJSON(Bar{Msg: "bar", Num: num})
}

func Example() {
	restful.Filter(biu.LogFilter())
	biu.AddServices("/v1", nil,
		biu.NS{
			NameSpace:  "foo",
			Controller: Foo{},
			Desc:       "Foo Controller",
		},
	)
	// Note: you should add swagger service after adding services.
	// swagger document will be available at http://localhost:8080/v1/swagger
	swaggerService := biu.NewSwaggerService(biu.SwaggerInfo{
		Title:        "Foo Bar",
		Description:  "Foo Bar Service",
		ContactName:  "Tuotoo",
		ContactEmail: "jqs7@tuotoo.com",
		ContactURL:   "https://tuotoo.com",
		Version:      "1.0.0",
		RoutePrefix:  "/v1",
	})
	restful.Add(swaggerService)
	biu.Run(":8080", nil)
}

package biu_test

import (
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/tuotoo/biu"
)

// Foo controller
type Foo struct{}

// WebService implements CtlInterface
func (ctl Foo) WebService(ws biu.WS) {
	ws.Route(ws.GET("/").To(biu.Handle(ctl.getBar)).
		Param(ws.QueryParameter("num", "number").DataType("int")).
		Doc("Get Bar").DefaultReturns("Bar", Bar{}), &biu.RouteOpt{
		Errors: map[int]string{
			100: "num not Number",
		},
	})
}

// Bar is the response of getBar
type Bar struct {
	Msg string `json:"msg"`
	Num int    `json:"num"`
}

func (ctl Foo) getBar(ctx biu.Ctx) {
	numStr := ctx.QueryParameter("num")
	num, err := strconv.Atoi(numStr)
	if ctx.ContainsError(err, 100) {
		return
	}
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
	biu.Run(":8010", nil)
}

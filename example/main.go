package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	slogmulti "github.com/samber/slog-multi"

	"github.com/tuotoo/biu"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/opt"
)

// Foo controller
type Foo struct{}

// WebService implements CtlInterface
func (ctl Foo) WebService(ws biu.WS) {
	ws.Route(ws.GET("/").Doc("Get Bar"),
		opt.RouteID("example.foo"),
		opt.RouteAPI(ctl.getBar),
		opt.RouteErrors(map[int]string{
			200: "%s is not a valid Number",
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

// try with: curl "127.0.0.1:8080/v1/foo?num=hey"
func (ctl Foo) getBar(ctx box.Ctx, api struct {
	Query struct {
		Num int `desc:"number" vd:"min=3"`
	}
	Return func(Bar)
}) {
	ctx.Must(ctx.VdErr(), 200, ctx.QueryParameter("num"))
	api.Return(Bar{Msg: "bar", Num: api.Query.Num})
}

func main() {
	c := biu.NewContainer().SetLogger(
		slog.New(
			slogmulti.Fanout(
				slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelWarn,
				}),
				tint.NewHandler(colorable.NewColorable(os.Stdout), &tint.Options{
					Level:      slog.LevelDebug,
					AddSource:  true,
					TimeFormat: time.Kitchen,
					NoColor:    !isatty.IsTerminal(os.Stdout.Fd()),
				}),
			),
		)).
		DefaultLogFilter().
		DefaultResponseTransformer().
		DefaultErrorTransformer()
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

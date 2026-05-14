// IP TLS Demo — 使用 biu 框架 + 自动 Let's Encrypt IP 证书
//
// 用法:
//
//	cd example/tls-demo && go run .
//
// 要求: 服务器必须有公网 IP（绑定在网卡上），或显式传入 IP
//
//	biu.WithTLSIP("1.2.3.4", "your@email.com")
//
// 默认使用 TLS-ALPN-01 challenge（无需额外端口）。
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

// Hello controller
type Hello struct{}

func (ctl Hello) WebService(ws biu.WS) {
	ws.Route(ws.GET("/").Doc("Hello World"),
		opt.RouteID("hello.greet"),
		opt.RouteAPI(ctl.greet),
	)
}

type Greeting struct {
	Message string `json:"message"`
}

func (ctl Hello) greet(ctx box.Ctx, api struct {
	Return func(Greeting)
}) {
	api.Return(Greeting{Message: "hello world"})
}

func main() {
	email := os.Getenv("ACME_EMAIL")
	if email == "" {
		email = "admin@example.com" // 替换为你的真实邮箱
	}

	ip := os.Getenv("PUBLIC_IP") // 留空 = 自动从网卡检测

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

	c.AddServices("/", opt.ServicesFuncArr{},
		biu.NS{
			NameSpace:  "",
			Controller: Hello{},
			Desc:       "Hello World Service",
		},
	)

	// IP TLS 模式（默认 Let's Encrypt + TLS-ALPN-01，只需 443 端口）
	c.Run(":443",
		opt.WithTLSIP(ip, email),
		// 可选：指定缓存目录
		// opt.WithTLSCacheDir("/var/lib/biu/certs"),
		// 可选：使用 HTTP-01 challenge（需 80 端口）
		// opt.WithTLSHTTP01(80),
		// 可选：HTTPS 非 443 端口 + TLS-ALPN-01 独立监听 443
		// opt.WithTLSALPN01(443),
		// 可选：使用 ZeroSSL（需 EAB 凭证）
		// opt.WithTLSCA(opt.CAZeroSSL),
		// opt.WithTLSEAB("your-eab-kid", "your-eab-hmac-key"),
	)
}

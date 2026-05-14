module example/tls-demo

go 1.25.0

toolchain go1.24.1

require (
	github.com/lmittmann/tint v1.1.2
	github.com/mattn/go-colorable v0.1.14
	github.com/mattn/go-isatty v0.0.20
	github.com/samber/slog-multi v1.7.0
	github.com/tuotoo/biu v0.7.5
)

replace github.com/tuotoo/biu => ../..

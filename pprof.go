package biu

import (
	"log/slog"
	"net/http"
	"net/http/pprof"

	"github.com/tuotoo/biu/opt"
)

// pprofMux builds the standard net/http/pprof index and profile handlers
// on a dedicated ServeMux so it never touches the container's ServeMux.
func pprofMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

// pprofHandler wraps mux with an optional token gate. When token is set,
// every request must carry token=<value> as a query parameter; requests
// without a matching token get 404 (not 401, to avoid advertising the
// endpoint).
func pprofHandler(mux *http.ServeMux, token string) http.Handler {
	if token == "" {
		return mux
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != token {
			http.NotFound(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// pprofAddr returns the listen address, defaulting to loopback.
func pprofAddr(cfg *opt.PprofConfig) string {
	if cfg.Addr != "" {
		return cfg.Addr
	}
	return "127.0.0.1:6060"
}

// startPprof starts the standalone pprof server and returns the running
// server plus the channel that receives its actual listen address.
// The caller owns Shutdown.
func startPprof(cfg *opt.PprofConfig, logger *slog.Logger) (*http.Server, <-chan string) {
	srv := &http.Server{
		Addr:    pprofAddr(cfg),
		Handler: pprofHandler(pprofMux(), cfg.Token),
	}
	addrChan := make(chan string)
	go func() {
		if err := ListenAndServe(srv, addrChan); err != nil && err != http.ErrServerClosed {
			logger.Error("pprof server stopped", slog.Any("err", err))
		}
	}()
	return srv, addrChan
}

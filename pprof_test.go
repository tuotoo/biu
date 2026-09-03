package biu

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/biu/opt"
)

func TestPprofHandler_NoToken(t *testing.T) {
	mux := pprofMux()
	h := pprofHandler(mux, "")

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pprof")
}

func TestPprofHandler_TokenMissing(t *testing.T) {
	mux := pprofMux()
	h := pprofHandler(mux, "secret")

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPprofHandler_TokenWrong(t *testing.T) {
	mux := pprofMux()
	h := pprofHandler(mux, "secret")

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/?token=wrong", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPprofHandler_TokenCorrect(t *testing.T) {
	mux := pprofMux()
	h := pprofHandler(mux, "secret")

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/?token=secret", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pprof")
}

func TestPprofAddr_Default(t *testing.T) {
	cfg := &opt.PprofConfig{}
	assert.Equal(t, "127.0.0.1:6060", pprofAddr(cfg))
}

func TestPprofAddr_Explicit(t *testing.T) {
	cfg := &opt.PprofConfig{Addr: "0.0.0.0:9999"}
	assert.Equal(t, "0.0.0.0:9999", pprofAddr(cfg))
}

func TestStartPprof_StartsAndServes(t *testing.T) {
	cfg := &opt.PprofConfig{Addr: "127.0.0.1:0"}
	srv, addrChan := startPprof(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	addr := <-addrChan
	assert.NotEmpty(t, addr)

	// Give the server a moment to be ready
	resp, err := http.Get("http://" + addr + "/debug/pprof/")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	err = srv.Close()
	assert.NoError(t, err)
}

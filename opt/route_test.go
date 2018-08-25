package opt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/opt"
)

func TestRouteTo(t *testing.T) {
	cfg := &opt.Route{}
	a := 1
	opt.RouteTo(func(ctx box.Ctx) {
		a = 2
	})(cfg)
	cfg.To(box.Ctx{})
	assert.Equal(t, 2, a)
}

func TestDisableAuthPathDoc(t *testing.T) {
	cfg := &opt.Route{EnableAutoPathDoc: true}
	opt.DisableAuthPathDoc()(cfg)
	assert.Equal(t, false, cfg.EnableAutoPathDoc)
}

func TestEnableAuth(t *testing.T) {
	cfg := &opt.Route{}
	opt.EnableAuth()(cfg)
	assert.Equal(t, true, cfg.Auth)
}

func TestExtraPathDocs(t *testing.T) {
	cfg := &opt.Route{}
	opt.ExtraPathDocs("1", "2")(cfg)
	assert.Equal(t, []string{"1", "2"}, cfg.ExtraPathDocs)
}

func TestRouteErrors(t *testing.T) {
	cfg := &opt.Route{}
	opt.RouteErrors(map[int]string{1: "2"})(cfg)
	assert.Contains(t, cfg.Errors, 1)
	assert.Equal(t, "2", cfg.Errors[1])
}

func TestRouteID(t *testing.T) {
	cfg := &opt.Route{}
	opt.RouteID("routeID")(cfg)
	assert.Equal(t, "routeID", cfg.ID)
}

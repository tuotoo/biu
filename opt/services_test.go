package opt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/biu"
	"github.com/tuotoo/biu/opt"
)

func TestErrors(t *testing.T) {
	cfg := &opt.Services{}
	opt.ServiceErrors(map[int]string{1: "fuck"})(cfg)
	assert.Contains(t, cfg.Errors, 1)
	assert.Equal(t, "fuck", cfg.Errors[1])
}

func TestFilters(t *testing.T) {
	cfg := &opt.Services{}
	opt.Filters(biu.LogFilter())(cfg)
}

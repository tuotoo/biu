package run_opt

import (
	"testing"

	"github.com/jqs7/dyttRSS/vendor_/github.com/stretchr/testify/assert"
)

func TestBeforeShutDown(t *testing.T) {
	opt := &RunOpt{}
	a := 1
	BeforeShutDown(func() {
		a = 2
	})(opt)
	opt.BeforeShutDown()
	assert.Equal(t, 2, a)
}

func TestAfterShutDown(t *testing.T) {
	opt := &RunOpt{}
	a := 1
	AfterShutDown(func() {
		a = 2
	})(opt)
	opt.AfterShutDown()
	assert.Equal(t, 2, a)
}

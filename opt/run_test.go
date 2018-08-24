package opt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/biu/opt"
)

func TestBeforeShutDown(t *testing.T) {
	run := &opt.Run{}
	a := 1
	opt.BeforeShutDown(func() {
		a = 2
	})(run)
	run.BeforeShutDown()
	assert.Equal(t, 2, a)
}

func TestAfterShutDown(t *testing.T) {
	run := &opt.Run{}
	a := 1
	opt.AfterShutDown(func() {
		a = 2
	})(run)
	run.AfterShutDown()
	assert.Equal(t, 2, a)
}

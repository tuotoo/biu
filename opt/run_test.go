package opt_test

import (
	"context"
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

func TestAfterStart(t *testing.T) {
	run := &opt.Run{}
	a := 1
	opt.AfterStart(func() {
		a = 2
	})(run)
	run.AfterStart()
	assert.Equal(t, 2, a)
}

func TestWithContext(t *testing.T) {
	run := &opt.Run{}
	ctx, cancel := context.WithCancel(context.Background())
	opt.WithContext(ctx, cancel)(run)
	cancel()
	err := run.Ctx.Err()
	assert.ErrorIs(t, err, context.Canceled)
}

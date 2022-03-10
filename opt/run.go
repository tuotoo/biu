package opt

import (
	"context"
)

// RunFunc is the type of running config functions.
type RunFunc func(*Run)

// Run is the running options of container.
type Run struct {
	BeforeShutDown func()
	AfterShutDown  func()
	AfterStart     func()
	Ctx            context.Context
	Cancel         context.CancelFunc
}

func AfterStart(f func()) RunFunc {
	return func(opt *Run) {
		opt.AfterStart = f
	}
}

// BeforeShutDown will run before http server shuts down.
func BeforeShutDown(f func()) RunFunc {
	return func(opt *Run) {
		opt.BeforeShutDown = f
	}
}

// AfterShutDown will run after http server shuts down.
func AfterShutDown(f func()) RunFunc {
	return func(opt *Run) {
		opt.AfterShutDown = f
	}
}

func WithContext(ctx context.Context, cancel context.CancelFunc) RunFunc {
	return func(opt *Run) {
		opt.Ctx = ctx
		opt.Cancel = cancel
	}
}

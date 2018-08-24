package opt

// RunFunc is the type of running config functions.
type RunFunc func(*Run)

// Run is the running options of container.
type Run struct {
	BeforeShutDown func()
	AfterShutDown  func()
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

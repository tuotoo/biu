package run_opt

type RunOptFunc func(*RunOpt)

// RunOpt is the running options of container.
type RunOpt struct {
	BeforeShutDown func()
	AfterShutDown  func()
}

// BeforeShutDown will run before http server shuts down.
func BeforeShutDown(f func()) RunOptFunc {
	return func(opt *RunOpt) {
		opt.BeforeShutDown = f
	}
}

// AfterShutDown will run after http server shuts down.
func AfterShutDown(f func()) RunOptFunc {
	return func(opt *RunOpt) {
		opt.AfterShutDown = f
	}
}

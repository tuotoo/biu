package opt

import "github.com/emicklei/go-restful"

// ServicesFunc is the type of biu.AddServices options.
type ServicesFunc func(*Services)

// ServicesFuncArr is a slice of Services Functions.
type ServicesFuncArr []ServicesFunc

// Services is the options for biu.AddServices.
type Services struct {
	Filters []restful.FilterFunction
	Errors  map[int]string
}

// Filters sets a list of filters for all services.
func Filters(filters ...restful.FilterFunction) ServicesFunc {
	return func(services *Services) {
		services.Filters = filters
	}
}

// ServiceErrors declares the global errors for services.
func ServiceErrors(errors map[int]string) ServicesFunc {
	return func(services *Services) {
		services.Errors = errors
	}
}

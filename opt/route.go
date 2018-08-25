package opt

import "github.com/tuotoo/biu/box"

// RouteFunc is the type of route options functions.
type RouteFunc func(*Route)

// Route is the options of route.
type Route struct {
	ID                string
	To                func(ctx box.Ctx)
	Auth              bool
	Errors            map[int]string
	EnableAutoPathDoc bool
	ExtraPathDocs     []string
}

// RouteID sets the ID of a route.
func RouteID(id string) RouteFunc {
	return func(route *Route) {
		route.ID = id
	}
}

// RouteTo binds a function to a route.
func RouteTo(f func(ctx box.Ctx)) RouteFunc {
	return func(route *Route) {
		route.To = f
	}
}

// EnableAuth enables JWT auth for a route.
func EnableAuth() RouteFunc {
	return func(route *Route) {
		route.Auth = true
	}
}

// RouteErrors defines the errors of a route.
func RouteErrors(m map[int]string) RouteFunc {
	return func(route *Route) {
		route.Errors = m
	}
}

// DisableAuthPathDoc disables auto generate path param docs for route.
func DisableAuthPathDoc() RouteFunc {
	return func(route *Route) {
		route.EnableAutoPathDoc = false
	}
}

// ExtraPathDocs sets extra descriptions for path params.
func ExtraPathDocs(docs ...string) RouteFunc {
	return func(route *Route) {
		route.ExtraPathDocs = docs
	}
}

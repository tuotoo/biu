package opt

import "github.com/tuotoo/biu/ctx"

type RouteFunc func(*Route)

type Route struct {
	ID                string
	To                func(ctx ctx.Ctx)
	Auth              bool
	Errors            map[int]string
	EnableAutoPathDoc bool
	ExtraPathDocs     []string
}

func RouteID(id string) RouteFunc {
	return func(route *Route) {
		route.ID = id
	}
}

func RouteTo(f func(ctx ctx.Ctx)) RouteFunc {
	return func(route *Route) {
		route.To = f
	}
}

func EnableAuth() RouteFunc {
	return func(route *Route) {
		route.Auth = true
	}
}

func RouteErrors(m map[int]string) RouteFunc {
	return func(route *Route) {
		route.Errors = m
	}
}

func DisableAuthPathDoc() RouteFunc {
	return func(route *Route) {
		route.EnableAutoPathDoc = false
	}
}

func ExtraPathDocs(docs ...string) RouteFunc {
	return func(route *Route) {
		route.ExtraPathDocs = docs
	}
}

package biu

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/mpvl/errc"
	"github.com/tuotoo/biu/box"
)

// Handle transform a biu handler to a restful.RouteFunction.
func Handle(f func(ctx box.Ctx)) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		c := box.Ctx{
			Request:  request,
			Response: response,
		}
		if !box.DisableErrHandler {
			e := errc.Catch(new(error))
			defer e.Handle()
			c.ErrCatcher = e
		}
		f(c)
	}
}

// Filter transform a biu handler to a restful.FilterFunction
func Filter(f func(ctx box.Ctx)) restful.FilterFunction {
	return func(request *restful.Request, response *restful.Response,
		chain *restful.FilterChain) {
		f(box.Ctx{
			Request:     request,
			Response:    response,
			FilterChain: chain,
		})
	}
}

// WrapHandler wraps a biu handler to http.HandlerFunc
func WrapHandler(f func(ctx box.Ctx)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(box.Ctx{
			Request:  restful.NewRequest(r),
			Response: restful.NewResponse(w),
		})
	}
}

// AuthFilter checks if request contains JWT,
// and sets UserID in Attribute if exists,
func AuthFilter(code int) restful.FilterFunction {
	return Filter(func(ctx box.Ctx) {
		userID, err := ctx.IsLogin()
		if ctx.ContainsError(err, code) {
			return
		}
		ctx.SetAttribute("UserID", userID)
		ctx.ProcessFilter(ctx.Request, ctx.Response)
	})
}

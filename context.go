package biu

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/mpvl/errc"
	"github.com/tuotoo/biu/auth"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/log"
)

// Handle transform a biu handler to a restful.RouteFunction.
func Handle(f func(ctx box.Ctx)) restful.RouteFunction {
	return HandleWithLogger(f, DefaultContainer.logger)
}

func HandleWithLogger(f func(ctx box.Ctx), logger log.ILogger) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		c := box.Ctx{
			Request:  request,
			Response: response,
			Logger:   logger,
		}
		e := errc.Catch(new(error))
		defer e.Handle()
		c.ErrCatcher = e
		f(c)
	}
}

// Filter transform a biu handler to a restful.FilterFunction
func Filter(f func(ctx box.Ctx)) restful.FilterFunction {
	return FilterWithLogger(f, DefaultContainer.logger)
}

func FilterWithLogger(f func(ctx box.Ctx), logger log.ILogger) restful.FilterFunction {
	return func(request *restful.Request, response *restful.Response,
		chain *restful.FilterChain) {
		c := box.Ctx{
			Request:     request,
			Response:    response,
			FilterChain: chain,
			Logger:      logger,
		}
		e := errc.Catch(new(error))
		defer e.Handle()
		c.ErrCatcher = e
		f(c)
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
func AuthFilter(code int, i *auth.Instance) restful.FilterFunction {
	return FilterWithLogger(func(ctx box.Ctx) {
		userID, err := ctx.IsLogin(i)
		ctx.Must(err, code)
		ctx.SetAttribute(box.BiuAttrAuthUserID, userID)
		ctx.Next()
	}, DefaultContainer.logger)
}

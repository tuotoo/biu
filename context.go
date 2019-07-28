package biu

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/tuotoo/biu/auth"
	"github.com/tuotoo/biu/box"
)

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
func AuthFilter(i *auth.Instance, code int) restful.FilterFunction {
	return DefaultContainer.FilterFunc(func(ctx box.Ctx) {
		userID, err := ctx.IsLogin(i)
		if ctx.ContainsError(err, code) {
			return
		}
		ctx.SetAttribute(box.BiuAttrAuthUserID, userID)
		ctx.Next()
	})
}

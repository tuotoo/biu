package biu_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/tuotoo/biu"
)

func ExampleSign() {
	biu.JWTTimeout(2 * time.Second).
		JWTSecret("hello world")
	token, _ := biu.Sign("user")
	ctx := &biu.Ctx{
		Request: &restful.Request{
			Request: &http.Request{
				Header: map[string][]string{
					"Authorization": {token},
				},
			},
		},
	}
	u1, e1 := ctx.IsLogin()
	fmt.Println(u1, e1 == nil)
	u2, e2 := ctx.CheckToken(token)
	fmt.Println(u2, e2 == nil)

	time.Sleep(time.Second * 3)
	u3, e3 := ctx.IsLogin()
	fmt.Println(u3, e3 != nil)
	u4, e4 := ctx.CheckToken(token)
	fmt.Println(u4, e4 != nil)

	ctx2 := &biu.Ctx{
		Request: &restful.Request{
			Request: &http.Request{
				Header: map[string][]string{
					"Authorization": {"wtf"},
				},
			},
		},
	}
	u5, e5 := ctx2.IsLogin()
	fmt.Println(u5, e5 != nil)
	// Output:
	// user true
	// user true
	//  true
	//  true
	//  true
}

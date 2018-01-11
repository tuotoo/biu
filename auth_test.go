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
	if e1 != nil {
		panic(e1)
	}
	fmt.Println(u1)
	u2, e2 := ctx.CheckToken(token)
	if e2 != nil {
		panic(e2)
	}
	fmt.Println(u2)

	time.Sleep(time.Second * 3)
	_, e3 := ctx.IsLogin()
	fmt.Println(e3 != nil)
	_, e4 := ctx.CheckToken(token)
	fmt.Println(e4 != nil)

	ctx2 := &biu.Ctx{
		Request: &restful.Request{
			Request: &http.Request{
				Header: map[string][]string{
					"Authorization": {"wtf"},
				},
			},
		},
	}
	_, e5 := ctx2.IsLogin()
	fmt.Println(e5 != nil)
	// Output:
	// user
	// user
	// true
	// true
	// true
}

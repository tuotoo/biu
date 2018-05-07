package biu_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/tuotoo/biu"
)

func ExampleSign() {
	biu.JWTTimeout(4 * time.Second).
		JWTSecret(func(userID string) (secret []byte, err error) {
			return []byte("hello world"), nil
		}).
		JWTRefreshTimeout(5 * time.Second)
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
	u1, err := ctx.IsLogin()
	if err != nil {
		panic(err)
	}
	fmt.Println(u1)
	u2, err := biu.CheckToken(token)
	if err != nil {
		panic(err)
	}
	fmt.Println(u2)
	time.Sleep(time.Second * 2)
	newToken, err := biu.RefreshToken(token)
	if err != nil {
		panic(err)
	}
	_, err = biu.CheckToken(newToken)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 3)
	// token is expired, newToken is still valid
	_, err = ctx.IsLogin()
	fmt.Println(err != nil)
	_, err = biu.CheckToken(token)
	fmt.Println(err != nil)
	_, err = biu.CheckToken(newToken)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	// cant refresh token if refresh timeout is reached
	_, err = biu.RefreshToken(newToken)
	fmt.Println(err != nil)

	ctx2 := &biu.Ctx{
		Request: &restful.Request{
			Request: &http.Request{
				Header: map[string][]string{
					"Authorization": {"wtf"},
				},
			},
		},
	}
	_, err = ctx2.IsLogin()
	fmt.Println(err != nil)
	// Output:
	// user
	// user
	// true
	// true
	// true
	// true
}

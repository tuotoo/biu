package auth_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/tuotoo/biu/auth"
	"github.com/tuotoo/biu/box"
)

func ExampleSign() {
	auth.JWTTimeout(4 * time.Second)
	auth.JWTSecret(func(userID string) (secret []byte, err error) {
		return []byte("hello world"), nil
	})
	auth.JWTRefreshTimeout(5 * time.Second)
	token, _ := auth.Sign("user")
	ctx := &box.Ctx{
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
	u2, err := auth.CheckToken(token)
	if err != nil {
		panic(err)
	}
	fmt.Println(u2)
	time.Sleep(time.Second * 2)
	newToken, err := auth.RefreshToken(token)
	if err != nil {
		panic(err)
	}
	_, err = auth.CheckToken(newToken)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 3)
	// token is expired, newToken is still valid
	_, err = ctx.IsLogin()
	fmt.Println(err != nil)
	_, err = auth.CheckToken(token)
	fmt.Println(err != nil)
	_, err = auth.CheckToken(newToken)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	// cant refresh token if refresh timeout is reached
	_, err = auth.RefreshToken(newToken)
	fmt.Println(err != nil)

	ctx2 := &box.Ctx{
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

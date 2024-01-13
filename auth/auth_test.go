package auth_test

import (
	"fmt"
	"github.com/dgrijalva/jwt-go/v4"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/tuotoo/biu/auth"
	"github.com/tuotoo/biu/box"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func TestInstance_ParseToken(t *testing.T) {
	secretFunc := func(userID string) (secret interface{}, err error) { return []byte("hello world"), nil }
	instance := auth.New(time.Second*4, time.Second*5, secretFunc, jwt.SigningMethodHS256.Alg())
	token, err := instance.Sign("user")
	if err != nil {
		panic(err)
	}
	ctx := &box.Ctx{
		Request: &restful.Request{
			Request: &http.Request{
				Header: map[string][]string{
					"Authorization": {token},
				},
			},
		},
	}
	u1, err := ctx.IsLogin(instance)
	if err != nil {
		panic(err)
	}
	fmt.Println(u1)
	u2, err := instance.CheckToken(token)
	if err != nil {
		panic(err)
	}
	fmt.Println(u2)
	time.Sleep(time.Second * 2)
	newToken, err := instance.RefreshToken(token)
	if err != nil {
		panic(err)
	}
	_, err = instance.CheckToken(newToken)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 3)
	// token is expired, newToken is still valid
	_, err = ctx.IsLogin(instance)
	fmt.Println(err != nil)
	_, err = instance.CheckToken(token)
	fmt.Println(err != nil)
	_, err = instance.CheckToken(newToken)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	// cant refresh token if refresh timeout is reached
	_, err = instance.RefreshToken(newToken)
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
	_, err = ctx2.IsLogin(instance)
	fmt.Println(err != nil)
	// Output:
	// user
	// user
	// true
	// true
	// true
	// true
}

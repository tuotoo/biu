package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

func ExampleHMAC() {
	instance := InstanceBuilder(NewHMAC(
		jwt.SigningMethodHS256,
		func(uid string) ([]byte, error) {
			return []byte("secret"), nil
		}),
	).Build()
	token, err := instance.Sign("user")
	if err != nil {
		panic(err)
	}
	uid, err := instance.CheckToken(token)
	if err != nil {
		panic(err)
	}
	fmt.Println(uid)
	// Output:
	// user
}

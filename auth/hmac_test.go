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
	token, err := Sign(instance, "user")
	if err != nil {
		panic(err)
	}
	uid, err := CheckToken(instance, token)
	if err != nil {
		panic(err)
	}
	fmt.Println(uid)
	// Output:
	// user
}

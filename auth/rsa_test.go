package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

func ExampleRSA() {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	instance := InstanceBuilder(NewRSA(jwt.SigningMethodRS256,
		func(uid string) (*rsa.PrivateKey, error) {
			return privateKey, nil
		},
		func(uid string) (*rsa.PublicKey, error) {
			return &privateKey.PublicKey, nil
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

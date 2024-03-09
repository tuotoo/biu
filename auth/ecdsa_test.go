package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

func ExampleECDSA() {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	instance := InstanceBuilder(NewECDSA(
		jwt.SigningMethodES256,
		func(uid string) (*ecdsa.PrivateKey, error) {
			return privateKey, nil
		},
		func(uid string) (*ecdsa.PublicKey, error) {
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

package auth

import (
	"crypto/ecdsa"
	"crypto/ed25519"
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

func ExampleEd25519() {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	instance := InstanceBuilder(NewEd25519(
		func(uid string) (ed25519.PrivateKey, error) {
			return privateKey, nil
		},
		func(uid string) (ed25519.PublicKey, error) {
			return publicKey, nil
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

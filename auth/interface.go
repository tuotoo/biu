package auth

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"

	"github.com/golang-jwt/jwt/v5"
)

type Alg[S SigningKey, V VerifyKey, M SigningMethod] interface {
	SecretKeyFunc(uid string) (S, error)
	VerifyKeyFunc(uid string) (V, error)
	SigningMethod() M
}

type SigningKey interface {
	[]byte | *ecdsa.PrivateKey | *ed25519.PrivateKey | *rsa.PrivateKey
}

type VerifyKey interface {
	[]byte | *ecdsa.PublicKey | *ed25519.PublicKey | *rsa.PublicKey
}

type SigningMethod interface {
	jwt.SigningMethod
}

type TokenChecker interface {
	// CheckToken checks the validation of the token and returns uid and any potential error.
	CheckToken(token string) (uid string, err error)
}

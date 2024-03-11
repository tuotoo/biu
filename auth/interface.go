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

type ITokenManager interface {
	// SignWithClaims signs the token with the given claims.
	SignWithClaims(uid string, claims map[string]any) (token string, err error)
	// ParseToken parses the token string and returns a jwt.Token and an error.
	ParseToken(token string) (*jwt.Token, error)
	// RefreshToken accepts a valid token and
	// returns a new token with new expire time.
	RefreshToken(token string) (newToken string, err error)
}

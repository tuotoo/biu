package auth

import (
	"crypto/rsa"

	"github.com/golang-jwt/jwt/v5"
)

type RSA struct {
	*jwt.SigningMethodRSA
	privateKeyFunc func(string) (*rsa.PrivateKey, error)
	publicKeyFunc  func(string) (*rsa.PublicKey, error)
}

func NewRSA(method *jwt.SigningMethodRSA, s func(string) (*rsa.PrivateKey, error), v func(string) (*rsa.PublicKey, error)) *RSA {
	return &RSA{
		SigningMethodRSA: method,
		privateKeyFunc:   s,
		publicKeyFunc:    v,
	}
}

func (h *RSA) SecretKeyFunc(userID string) (*rsa.PrivateKey, error) {
	return h.privateKeyFunc(userID)
}

func (h *RSA) VerifyKeyFunc(userID string) (*rsa.PublicKey, error) {
	return h.publicKeyFunc(userID)
}

func (h *RSA) SigningMethod() *jwt.SigningMethodRSA {
	return h.SigningMethodRSA
}

type RSAPSS struct {
	*jwt.SigningMethodRSAPSS
	privateKeyFunc func(string) (*rsa.PrivateKey, error)
	publicKeyFunc  func(string) (*rsa.PublicKey, error)
}

func NewRSAPSS(method *jwt.SigningMethodRSAPSS, s func(string) (*rsa.PrivateKey, error), v func(string) (*rsa.PublicKey, error)) *RSAPSS {
	return &RSAPSS{
		SigningMethodRSAPSS: method,
		privateKeyFunc:      s,
		publicKeyFunc:       v,
	}
}

func (h *RSAPSS) SecretKeyFunc(userID string) (*rsa.PrivateKey, error) {
	return h.privateKeyFunc(userID)
}

func (h *RSAPSS) VerifyKeyFunc(userID string) (*rsa.PublicKey, error) {
	return h.publicKeyFunc(userID)
}

func (h *RSAPSS) SigningMethod() *jwt.SigningMethodRSA {
	return h.SigningMethodRSA
}

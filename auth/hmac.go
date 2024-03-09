package auth

import "github.com/golang-jwt/jwt/v5"

type HMAC struct {
	*jwt.SigningMethodHMAC
	secretFunc func(string) ([]byte, error)
}

func NewHMAC(method *jwt.SigningMethodHMAC, f func(string) ([]byte, error)) *HMAC {
	return &HMAC{
		SigningMethodHMAC: method,
		secretFunc:        f,
	}
}

func (h *HMAC) SecretKeyFunc(uid string) ([]byte, error) {
	return h.secretFunc(uid)
}

func (h *HMAC) VerifyKeyFunc(uid string) ([]byte, error) {
	return h.secretFunc(uid)
}

func (h *HMAC) SigningMethod() *jwt.SigningMethodHMAC {
	return h.SigningMethodHMAC
}

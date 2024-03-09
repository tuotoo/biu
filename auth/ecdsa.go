package auth

import (
	"crypto/ecdsa"
	"crypto/ed25519"

	"github.com/golang-jwt/jwt/v5"
)

type ECDSA struct {
	*jwt.SigningMethodECDSA
	privateKeyFunc func(string) (*ecdsa.PrivateKey, error)
	publicKeyFunc  func(string) (*ecdsa.PublicKey, error)
}

func NewECDSA(method *jwt.SigningMethodECDSA, s func(string) (*ecdsa.PrivateKey, error), v func(string) (*ecdsa.PublicKey, error)) *ECDSA {
	return &ECDSA{
		SigningMethodECDSA: method,
		privateKeyFunc:     s,
		publicKeyFunc:      v,
	}
}

func (h *ECDSA) SecretKeyFunc(uid string) (*ecdsa.PrivateKey, error) {
	return h.privateKeyFunc(uid)
}

func (h *ECDSA) VerifyKeyFunc(uid string) (*ecdsa.PublicKey, error) {
	return h.publicKeyFunc(uid)
}

func (h *ECDSA) SigningMethod() *jwt.SigningMethodECDSA {
	return h.SigningMethodECDSA
}

type Ed25519 struct {
	*jwt.SigningMethodEd25519
	privateKeyFunc func(string) (*ed25519.PrivateKey, error)
	publicKeyFunc  func(string) (*ed25519.PublicKey, error)
}

func NewEd25519(s func(string) (*ed25519.PrivateKey, error), v func(string) (*ed25519.PublicKey, error)) *Ed25519 {
	return &Ed25519{
		SigningMethodEd25519: jwt.SigningMethodEdDSA,
		privateKeyFunc:       s,
		publicKeyFunc:        v,
	}
}

func (h *Ed25519) SecretKeyFunc(uid string) (*ed25519.PrivateKey, error) {
	return h.privateKeyFunc(uid)
}

func (h *Ed25519) VerifyKeyFunc(uid string) (*ed25519.PublicKey, error) {
	return h.publicKeyFunc(uid)
}

func (h *Ed25519) SigningMethod() *jwt.SigningMethodEd25519 {
	return h.SigningMethodEd25519
}

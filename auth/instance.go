package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Builder[
	S SigningKey,
	V VerifyKey,
	M SigningMethod,
	T Alg[S, V, M],
] struct {
	manager *Manager[S, V, M, T]
}

// InstanceBuilder returns a Builder instance.
func InstanceBuilder[
	S SigningKey,
	V VerifyKey,
	M SigningMethod,
	T Alg[S, V, M],
](alg T) *Builder[S, V, M, T] {
	return &Builder[S, V, M, T]{
		manager: &Manager[S, V, M, T]{
			alg:            alg,
			timeout:        time.Minute * 5,
			refreshTimeout: time.Hour * 24 * 7,
		},
	}
}

// SetTimeout sets the timeout for the Instance.
func (b *Builder[S, V, M, T]) SetTimeout(timeout time.Duration) *Builder[S, V, M, T] {
	b.manager.timeout = timeout
	return b
}

// SetRefreshTimeout sets the refresh timeout for the Instance.
func (b *Builder[S, V, M, T]) SetRefreshTimeout(timeout time.Duration) *Builder[S, V, M, T] {
	b.manager.refreshTimeout = timeout
	return b
}

func (b *Builder[S, V, M, T]) Build() *Instance {
	return &Instance{
		TokenManager: b.manager,
	}
}

type Instance struct {
	TokenManager
}

// Sign returns a signed jwt string.
func (e *Instance) Sign(uid string) (token string, err error) {
	return e.TokenManager.SignWithClaims(uid, nil)
}

// CheckToken accept a jwt token and returns the uid in token.
func (e *Instance) CheckToken(token string) (userID string, err error) {
	t, err := e.TokenManager.ParseToken(token)
	if err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok || !t.Valid {
		return "", errors.New("unexpected token")
	}
	_uid, ok := claims["uid"].(string)
	if !ok {
		return "", errors.New("not available uid")
	}
	return _uid, nil
}

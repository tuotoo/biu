package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Manager[
	S SigningKey,
	V VerifyKey,
	M SigningMethod,
	T Alg[S, V, M],
] struct {
	alg            T
	timeout        time.Duration
	refreshTimeout time.Duration
}

// SignWithClaims signs the token with the given claims.
func (i *Manager[S, V, M, T]) SignWithClaims(uid string, claims map[string]any) (token string, err error) {
	now := time.Now()
	_claims := jwt.MapClaims{
		"uid": uid,
		"exp": now.Add(i.timeout).Unix(),
		"iat": now.Unix(),
	}
	for k, v := range claims {
		_claims[k] = v
	}
	jwtToken := jwt.NewWithClaims(i.alg.SigningMethod(), _claims)

	sec, err := i.alg.SecretKeyFunc(uid)
	if err != nil {
		return "", err
	}
	return jwtToken.SignedString(sec)
}

// ParseToken parse a token string.
func (i *Manager[S, V, M, T]) ParseToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, methodOK := token.Method.(M); !methodOK {
			signingErr := fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			return nil, signingErr
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			claimParseErr := fmt.Errorf("unexpected claims: %v", claims)
			return nil, claimParseErr
		}
		uid, ok := claims["uid"].(string)
		if !ok {
			uidErr := fmt.Errorf("unexpected uid: %v", claims["uid"])
			return nil, uidErr
		}
		return i.alg.VerifyKeyFunc(uid)
	})
}

// RefreshToken accepts a valid token and
// returns a new token with new expire time.
func (i *Manager[S, V, M, T]) RefreshToken(token string) (newToken string, err error) {
	t, err := i.ParseToken(token)
	if err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok || !t.Valid {
		return "", errors.New("unexpected token")
	}
	iatF64, ok := claims["iat"].(float64)
	if !ok {
		return "", errors.New("not available iat")
	}
	now := time.Now()
	iat := int64(iatF64)
	if iat < now.Add(-i.refreshTimeout).Unix() {
		return "", errors.New("refresh is expired")
	}
	uid, ok := claims["uid"].(string)
	if !ok {
		return "", errors.New("not available uid")
	}
	claims["exp"] = now.Add(i.timeout).Unix()
	jwtToken := jwt.NewWithClaims(i.alg.SigningMethod(), claims)
	sec, err := i.alg.SecretKeyFunc(uid)
	if err != nil {
		return "", err
	}
	return jwtToken.SignedString(sec)
}

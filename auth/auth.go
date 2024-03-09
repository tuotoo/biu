package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Instance[
	S SigningKey,
	V VerifyKey,
	M SigningMethod,
	T Alg[S, V, M],
] struct {
	alg            T
	timeout        time.Duration
	refreshTimeout time.Duration
}

type Builder[
	S SigningKey,
	V VerifyKey,
	M SigningMethod,
	T Alg[S, V, M],
] struct {
	instance *Instance[S, V, M, T]
}

// InstanceBuilder is a Go function that returns a Builder instance.
//
// It takes an alg parameter of type T and returns a pointer to Builder[S, V, M, T].
func InstanceBuilder[
	S SigningKey,
	V VerifyKey,
	M SigningMethod,
	T Alg[S, V, M],
](alg T) *Builder[S, V, M, T] {
	return &Builder[S, V, M, T]{
		instance: &Instance[S, V, M, T]{
			alg:            alg,
			timeout:        time.Minute * 5,
			refreshTimeout: time.Hour * 24 * 7,
		},
	}
}

// SetTimeout sets the timeout for the Builder function.
//
// timeout time.Duration
// *Builder[S, V, M, T]
func (b *Builder[S, V, M, T]) SetTimeout(timeout time.Duration) *Builder[S, V, M, T] {
	b.instance.timeout = timeout
	return b
}

// SetRefreshTimeout sets the refresh timeout for the Builder.
//
// timeout time.Duration
// *Builder[S, V, M, T]
func (b *Builder[S, V, M, T]) SetRefreshTimeout(timeout time.Duration) *Builder[S, V, M, T] {
	b.instance.refreshTimeout = timeout
	return b
}

func (b *Builder[S, V, M, T]) Build() *Instance[S, V, M, T] {
	return b.instance
}

// SignWithClaims signs the token with the given claims.
//
// uid string, claims map[string]any. (token string, err error).
func (i *Instance[S, V, M, T]) SignWithClaims(uid string, claims map[string]any) (token string, err error) {
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

// Sign returns a signed jwt string.
func (i *Instance[S, V, M, T]) Sign(uid string) (token string, err error) {
	return i.SignWithClaims(uid, nil)
}

// ParseToken parse a token string.
func (i *Instance[S, V, M, T]) ParseToken(token string) (*jwt.Token, error) {
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
func (i *Instance[S, V, M, T]) RefreshToken(token string) (newToken string, err error) {
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

// CheckToken accept a jwt token and returns the uid in token.
func (i *Instance[S, V, M, T]) CheckToken(token string) (uid string, err error) {
	t, err := i.ParseToken(token)
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

package auth

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"golang.org/x/xerrors"
)

type Instance struct {
	timeout        time.Duration
	refreshTimeout time.Duration
	secretFunc     func(string) ([]byte, error)
}

// New generate auth instance
func New(timeout, refreshTimeout time.Duration, secretFunc func(string) ([]byte, error)) *Instance {
	i := new(Instance)
	i.timeout = timeout
	i.refreshTimeout = refreshTimeout
	i.secretFunc = secretFunc
	return i
}

// Sign returns a signed jwt string.
func (i *Instance) Sign(userID string) (token string, err error) {
	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID,
		"exp": now.Add(i.timeout).Unix(),
		"iat": now.Unix(),
	})
	sec, err := i.secretFunc(userID)
	if err != nil {
		return "", err
	}
	return jwtToken.SignedString(sec)
}

// ParseToken parse a token string.
func (i *Instance) ParseToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, methodOK := token.Method.(*jwt.SigningMethodHMAC); !methodOK {
			signingErr := xerrors.Errorf("unexpected signing method: %v", token.Header["alg"])
			return nil, signingErr
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			claimParseErr := xerrors.Errorf("unexpected claims: %v", claims)
			return nil, claimParseErr
		}
		uid, ok := claims["uid"].(string)
		if !ok {
			uidErr := xerrors.Errorf("unexpected uid: %v", claims["uid"])
			return nil, uidErr
		}
		return i.secretFunc(uid)
	})
}

// RefreshToken accepts a valid token and
// returns a new token with new expire time.
func (i *Instance) RefreshToken(token string) (newToken string, err error) {
	t, err := i.ParseToken(token)
	if err != nil {
		return "", xerrors.Errorf("parse token: %w", err)
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
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": uid,
		"exp": now.Add(i.timeout).Unix(),
		"iat": iat,
	})
	sec, err := i.secretFunc(uid)
	if err != nil {
		return "", err
	}
	return jwtToken.SignedString(sec)
}

// CheckToken accept a jwt token and returns the uid in token.
func (i *Instance) Verify(token string) (userID string, err error) {
	t, err := i.ParseToken(token)
	if err != nil {
		return "", xerrors.Errorf("parse token: %w", err)
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok || !t.Valid {
		return "", errors.New("unexpected token")
	}
	uid, ok := claims["uid"].(string)
	if !ok {
		return "", errors.New("not available uid")
	}
	return uid, nil
}

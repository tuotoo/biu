package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/xerrors"
)

type Instance struct {
	Timeout        time.Duration
	RefreshTimeout time.Duration
	SecretFunc     func(string) ([]byte, error)
}

var DefaultInstance = &Instance{
	Timeout: time.Minute * 5,
	SecretFunc: func(userID string) (secret []byte, err error) {
		return []byte("secret"), nil
	},
	RefreshTimeout: time.Hour * 24 * 7,
}

// Sign returns a signed jwt string.
func (i *Instance) Sign(userID string) (token string, err error) {
	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID,
		"exp": now.Add(i.Timeout).Unix(),
		"iat": now.Unix(),
	})
	sec, err := i.SecretFunc(userID)
	if err != nil {
		return "", err
	}
	return jwtToken.SignedString(sec)
}

func JWTTimeout(timeout time.Duration) {
	DefaultInstance.Timeout = timeout
}

func JWTSecret(f func(string) ([]byte, error)) {
	DefaultInstance.SecretFunc = f
}

func JWTRefreshTimeout(timeout time.Duration) {
	DefaultInstance.RefreshTimeout = timeout
}

// Sign returns a signed jwt string with default instance.
func Sign(userID string) (token string, err error) {
	return DefaultInstance.Sign(userID)
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
		return i.SecretFunc(uid)
	})
}

// ParseToken parse a token string with default instance.
func ParseToken(token string) (*jwt.Token, error) {
	return DefaultInstance.ParseToken(token)
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
	if iat < now.Add(-i.RefreshTimeout).Unix() {
		return "", errors.New("refresh is expired")
	}
	uid, ok := claims["uid"].(string)
	if !ok {
		return "", errors.New("not available uid")
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": uid,
		"exp": now.Add(i.Timeout).Unix(),
		"iat": iat,
	})
	sec, err := i.SecretFunc(uid)
	if err != nil {
		return "", err
	}
	return jwtToken.SignedString(sec)
}

// RefreshToken accepts a valid token and
// returns a new token with new expire time.
func RefreshToken(token string) (newToken string, err error) {
	return DefaultInstance.RefreshToken(token)
}

// CheckToken accept a jwt token and returns the uid in token.
func (i *Instance) CheckToken(token string) (userID string, err error) {
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

// CheckToken accept a jwt token and returns the uid in token with default instance.
func CheckToken(token string) (userID string, err error) {
	return DefaultInstance.CheckToken(token)
}

package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/tuotoo/biu/log"
)

var (
	JWTTimeout = time.Minute * 5
	JWTSecret  = func(userID string) (secret []byte, err error) {
		return []byte("secret"), nil
	}
	JWTRefreshTimeout = time.Hour * 24 * 7
)

// Sign returns a signed jwt string.
func Sign(userID string) (token string, err error) {
	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID,
		"exp": now.Add(JWTTimeout).Unix(),
		"iat": now.Unix(),
	})
	sec, err := JWTSecret(userID)
	if err != nil {
		return "", err
	}
	return jwtToken.SignedString(sec)
}

// ParseToken parse a token string.
func ParseToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, methodOK := token.Method.(*jwt.SigningMethodHMAC); !methodOK {
			signingErr := fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			log.Info().Err(signingErr).Msg("parse signing method")
			return nil, signingErr
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			claimParseErr := fmt.Errorf("unexpected claims: %v", claims)
			log.Info().Err(claimParseErr).Msg("parse token claims")
			return nil, claimParseErr
		}
		uid, ok := claims["uid"].(string)
		if !ok {
			uidErr := fmt.Errorf("unexpected uid: %v", claims["uid"])
			log.Info().Err(uidErr).Msg("parse uid in token")
			return nil, uidErr
		}
		return JWTSecret(uid)
	})
}

// RefreshToken accepts a valid token and
// returns a new token with new expire time.
func RefreshToken(token string) (newToken string, err error) {
	t, err := ParseToken(token)
	if err != nil {
		log.Info().Err(err).Msg("parse token")
		return "", err
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
	if iat < now.Add(-JWTRefreshTimeout).Unix() {
		return "", errors.New("refresh is expired")
	}
	uid, ok := claims["uid"].(string)
	if !ok {
		return "", errors.New("not available uid")
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": uid,
		"exp": now.Add(JWTTimeout).Unix(),
		"iat": iat,
	})
	sec, err := JWTSecret(uid)
	if err != nil {
		return "", err
	}
	return jwtToken.SignedString(sec)
}

// CheckToken accept a jwt token and returns the uid in token.
func CheckToken(token string) (userID string, err error) {
	t, err := ParseToken(token)
	if err != nil {
		log.Info().Err(err).Msg("parse token")
		return "", err
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
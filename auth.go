package biu

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
)

var jwtInfo struct {
	timeout time.Duration
	secret  []byte
}

func SetJWTInfo(timeout int, secret string) {
	jwtInfo.timeout = time.Minute * time.Duration(timeout)
	jwtInfo.secret = []byte(secret)
}

func Sign(userID string) (token string, err error) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID,
		"exp": time.Now().Add(jwtInfo.timeout).Unix(),
	})
	return jwtToken.SignedString(jwtInfo.secret)
}

func (ctx *Ctx) CheckToken(token string) (userID string, err error) {
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, methodOK := token.Method.(*jwt.SigningMethodHMAC); !methodOK {
			signingErr := fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			Info("parse signing method", Log().Err(signingErr))
			return nil, signingErr
		}
		return jwtInfo.secret, nil
	})
	if err != nil {
		Info("parse token", Log().Err(err))
		return "", err
	}

	if claims, isMapClaims := t.Claims.(jwt.MapClaims); isMapClaims && t.Valid {
		if uid, isString := claims["uid"].(string); isString {
			return uid, nil
		}
	}
	return "", errors.New("unexpected token")
}

func (ctx *Ctx) IsLogin() (userID string, err error) {
	tokenString, err := request.OAuth2Extractor.ExtractToken(ctx.Request.Request)
	if err != nil {
		Info("no auth header", Log().Err(err))
		return "", err
	}
	return ctx.CheckToken(tokenString)
}

package biu

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/emicklei/go-restful"
)

var jwtInfo struct {
	timeout time.Duration
	secret  []byte
}

// SetJWTInfo sets the options of JWT.
// The time unit of timeout is minute.
func SetJWTInfo(timeout int, secret string) {
	jwtInfo.timeout = time.Minute * time.Duration(timeout)
	jwtInfo.secret = []byte(secret)
}

// Sign returns a signed jwt string.
func Sign(userID string) (token string, err error) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID,
		"exp": time.Now().Add(jwtInfo.timeout).Unix(),
	})
	return jwtToken.SignedString(jwtInfo.secret)
}

// CheckToken accept a jwt token and returns the uid in token.
func (ctx *Ctx) CheckToken(token string) (userID string, err error) {
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, methodOK := token.Method.(*jwt.SigningMethodHMAC); !methodOK {
			signingErr := fmt.Errorf("unexpected signing method: %v",
				token.Header["alg"])
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

// IsLogin gets JWT token in request by OAuth2Extractor,
// and parse it with CheckToken.
func (ctx *Ctx) IsLogin() (userID string, err error) {
	tokenString, err := request.OAuth2Extractor.ExtractToken(ctx.Request.Request)
	if err != nil {
		Info("no auth header", Log().Err(err))
		return "", err
	}
	return ctx.CheckToken(tokenString)
}

func AuthFilter(code int) restful.FilterFunction {
	return Filter(func(ctx Ctx) {
		userID, err := ctx.IsLogin()
		if ctx.ContainsError(err, code) {
			return
		}
		ctx.SetAttribute("UserID", userID)
		ctx.ProcessFilter(ctx.Request, ctx.Response)
	})
}

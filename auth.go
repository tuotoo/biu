package biu

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/emicklei/go-restful"
)

var globalOptions = struct {
	jwtTimeout time.Duration
	jwtSecret  []byte
}{
	jwtTimeout: time.Hour,
	jwtSecret:  []byte("secret"),
}

// Setter is a setter for setting global options.
type Setter struct{}

// JWTTimeout sets timeout for JWT.
func JWTTimeout(timeout time.Duration) Setter {
	globalOptions.jwtTimeout = timeout
	return Setter{}
}

// JWTTimeout sets timeout for JWT.
func (Setter) JWTTimeout(timeout time.Duration) Setter {
	return JWTTimeout(timeout)
}

// JWTSecret sets secret for JWT.
func JWTSecret(secret string) Setter {
	globalOptions.jwtSecret = []byte(secret)
	return Setter{}
}

// JWTSecret sets secret for JWT.
func (Setter) JWTSecret(secret string) Setter {
	return JWTSecret(secret)
}

// Sign returns a signed jwt string.
func Sign(userID string) (token string, err error) {
	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID,
		"exp": now.Add(globalOptions.jwtTimeout).Unix(),
		"iat": now.Unix(),
	})
	return jwtToken.SignedString(globalOptions.jwtSecret)
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
		return globalOptions.jwtSecret, nil
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

// AuthFilter checks if request contains JWT,
// and sets UserID in Attribute if exists,
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

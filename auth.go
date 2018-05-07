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
	jwtTimeout        time.Duration
	jwtSecret         func(string) ([]byte, error)
	jwtRefreshTimeout time.Duration
}{
	jwtTimeout: time.Minute * 5,
	jwtSecret: func(userID string) (secret []byte, err error) {
		return []byte("secret"), nil
	},
	jwtRefreshTimeout: time.Hour * 24 * 7,
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
func JWTSecret(f func(userID string) (secret []byte, err error)) Setter {
	globalOptions.jwtSecret = f
	return Setter{}
}

// JWTSecret sets secret for JWT.
func (Setter) JWTSecret(f func(userID string) (secret []byte, err error)) Setter {
	return JWTSecret(f)
}

// JWTRefreshTimeout sets refresh timeout for JWT.
func JWTRefreshTimeout(timeout time.Duration) Setter {
	globalOptions.jwtRefreshTimeout = timeout
	return Setter{}
}

// JWTRefreshTimeout sets refresh timeout for JWT.
func (Setter) JWTRefreshTimeout(timeout time.Duration) Setter {
	return JWTRefreshTimeout(timeout)
}

// Sign returns a signed jwt string.
func Sign(userID string) (token string, err error) {
	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID,
		"exp": now.Add(globalOptions.jwtTimeout).Unix(),
		"iat": now.Unix(),
	})
	sec, err := globalOptions.jwtSecret(userID)
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
			Info().Err(signingErr).Msg("parse signing method")
			return nil, signingErr
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			claimParseErr := fmt.Errorf("unexpected claims: %v", claims)
			Info().Err(claimParseErr).Msg("parse token claims")
			return nil, claimParseErr
		}
		uid, ok := claims["uid"].(string)
		if !ok {
			uidErr := fmt.Errorf("unexpected uid: %v", claims["uid"])
			Info().Err(uidErr).Msg("parse uid in token")
			return nil, uidErr
		}
		return globalOptions.jwtSecret(uid)
	})
}

// RefreshToken accepts a valid token and
// returns a new token with new expire time.
func RefreshToken(token string) (newToken string, err error) {
	t, err := ParseToken(token)
	if err != nil {
		Info().Err(err).Msg("parse token")
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
	if iat < now.Add(-globalOptions.jwtRefreshTimeout).Unix() {
		return "", errors.New("refresh is expired")
	}
	uid, ok := claims["uid"].(string)
	if !ok {
		return "", errors.New("not available uid")
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": uid,
		"exp": now.Add(globalOptions.jwtTimeout).Unix(),
		"iat": iat,
	})
	sec, err := globalOptions.jwtSecret(uid)
	if err != nil {
		return "", err
	}
	return jwtToken.SignedString(sec)
}

// CheckToken accept a jwt token and returns the uid in token.
func CheckToken(token string) (userID string, err error) {
	t, err := ParseToken(token)
	if err != nil {
		Info().Err(err).Msg("parse token")
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

// IsLogin gets JWT token in request by OAuth2Extractor,
// and parse it with CheckToken.
func (ctx *Ctx) IsLogin() (userID string, err error) {
	tokenString, err := request.OAuth2Extractor.ExtractToken(ctx.Request.Request)
	if err != nil {
		Info().Err(err).Msg("no auth header")
		return "", err
	}
	return CheckToken(tokenString)
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

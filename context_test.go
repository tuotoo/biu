package biu

import (
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/tuotoo/biu/auth"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/expect"
)

type MockAuthTokenManager struct {
}

func (m MockAuthTokenManager) SignWithClaims(uid string, claims map[string]any) (token string, err error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": uid,
	})
	return tok.SignedString([]byte(""))
}

func (m MockAuthTokenManager) ParseToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (any, error) {
		return []byte(""), nil
	})
}

func (m MockAuthTokenManager) RefreshToken(token string) (newToken string, err error) {
	panic("implement me")
}

func TestAuthFilter(t *testing.T) {
	e := New()
	authInstance := &auth.Instance{
		ITokenManager: MockAuthTokenManager{},
	}

	e.Filter(AuthFilter(100, authInstance))
	ws := e.NewWS()
	ws.Route(ws.POST("/auth"), ws.RouteAPI(func(ctx box.Ctx, api struct {
		Return func(string)
	}) {
		assert.Equal(t, "1", ctx.UserID())
		api.Return("OK")
	}))
	e.Add(ws.WebService)
	s := httptest.NewServer(e)
	defer s.Close()

	expect.Default(t, s.URL).POST("/auth").
		Expect().JSON().Object().HasValue("code", 100)

	token, err := authInstance.Sign("1")
	assert.NoError(t, err)
	expect.Default(t, s.URL).POST("/auth").WithHeader("Authorization", token).
		Expect().JSON().Object().HasValue("code", 0)
}

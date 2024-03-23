package biu

import (
	"net/http/httptest"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tuotoo/biu/auth"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/opt"
)

type MockAuthTokenManager struct {
}

func (m MockAuthTokenManager) SignWithClaims(uid string, claims map[string]any) (token string, err error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": "1",
	})
	return tok.SignedString([]byte(""))
}

func (m MockAuthTokenManager) ParseToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
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
	token, err := authInstance.Sign("")
	assert.NoError(t, err)
	e.Filter(AuthFilter(100, authInstance))
	ws := e.NewWS()
	ws.Route(ws.POST("/auth"), opt.RouteAPI(func(ctx box.Ctx, api struct {
		Return func(string)
	}) {
		assert.Equal(t, "1", ctx.UserID())
		api.Return("OK")
	}))
	e.Add(ws.WebService)
	s := httptest.NewServer(e)
	defer s.Close()

	httpexpect.Default(t, s.URL).POST("/auth").
		Expect().JSON().Object().HasValue("code", 100)

	httpexpect.Default(t, s.URL).POST("/auth").WithHeader("Authorization", token).
		Expect().JSON().Object().HasValue("code", 0)
}

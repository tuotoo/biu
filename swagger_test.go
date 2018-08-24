package biu_test

import (
	"testing"

	"github.com/jqs7/dyttRSS/vendor_/github.com/stretchr/testify/assert"
	"github.com/tuotoo/biu"
)

type SwaggerTest struct{}

func (ctl SwaggerTest) WebService(ws biu.WS) {
	authOpt := &biu.RouteOpt{Auth: true}
	ws.Route(ws.GET("/"), authOpt)
	ws.Route(ws.GET("/no/auth"), &biu.RouteOpt{})
	ws.Route(ws.POST("/"), authOpt)
	ws.Route(ws.PATCH("/"), authOpt)
	ws.Route(ws.DELETE("/"), authOpt)
	ws.Route(ws.HEAD("/"), authOpt)
	ws.Route(ws.PUT("/"), authOpt)
}

func TestNewSwaggerService(t *testing.T) {
	biu.AddServices("/v1", nil, biu.NS{
		NameSpace:  "test",
		Controller: SwaggerTest{},
	})
	routes := biu.NewSwaggerService(biu.SwaggerInfo{
		Title:          "title",
		Description:    "desc",
		TermsOfService: "tos",
		ContactName:    "contactName",
		ContactEmail:   "contactEmail",
		ContactURL:     "contactURL",
		LicenseName:    "licenseName",
		LicenseURL:     "licenseURL",
		Version:        "1.0",
		RoutePrefix:    "/v1",
	}).Routes()
	assert.Len(t, routes, 1)
	route := routes[0]
	assert.Equal(t, "GET /v1/swagger.json/", route.String())
}

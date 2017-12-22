package biu

import (
	"github.com/emicklei/go-restful"
)

// NS contains configuration of a namespace
type NS struct {
	NameSpace    string       // url parent of controller
	Controller   CtlInterface // controller implement CtlInterface
	Desc         string       // description of controller of namespace
	ExternalDesc string       // external documentation of controller
	ExternalURL  string       // external url of ExternalDesc
}

// WS extends *restful.WebService
type WS struct {
	*restful.WebService
}

// CtlInterface is the interface of controllers
type CtlInterface interface {
	WebService(WS)
}

// CommonResp with code, message and data
type CommonResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// SwaggerInfo contains configuration of swagger documents.
type SwaggerInfo struct {
	Title          string
	Description    string
	TermsOfService string
	ContactName    string
	ContactURL     string
	ContactEmail   string
	LicenseName    string
	LicenseURL     string
	Version        string
	WebServicesURL string
	DisableCORS    bool
	// route prefix of swagger service
	// swagger service will running under
	// http://<api>/<RoutePrefix>/swagger
	RoutePrefix string
}

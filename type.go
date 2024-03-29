package biu

import (
	"github.com/emicklei/go-restful/v3"
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
	namespace string
	*restful.WebService
	Container *Container
	errors    map[string]map[int]string
}

// CtlInterface is the interface of controllers
type CtlInterface interface {
	WebService(WS)
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
	// swagger service will be running under
	// http://<api>/<RoutePrefix>/<RouteSuffix>
	// by default the RouteSuffix is swagger
	RoutePrefix string
	RouteSuffix string
}

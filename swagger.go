package biu

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/tuotoo/biu/swagger-go"
	"strings"
)

// NewSwaggerService creates a swagger webservice in /swagger
func (c *Container) NewSwaggerService(info SwaggerInfo) *restful.WebService {
	return newSwaggerService(info, c.ServeMux)
}

// NewSwaggerService creates a swagger webservice in /swagger
func NewSwaggerService(info SwaggerInfo) *restful.WebService {
	return newSwaggerService(info, http.DefaultServeMux)
}

func newSwaggerService(
	info SwaggerInfo,
	serveMux *http.ServeMux,
) *restful.WebService {
	config := restfulspec.Config{
		WebServices:                   restful.RegisteredWebServices(),
		APIPath:                       info.RoutePrefix + "/swagger.json",
		DisableCORS:                   info.DisableCORS,
		WebServicesURL:                info.WebServicesURL,
		PostBuildSwaggerObjectHandler: enrichSwaggerObject(info),
	}
	serveMux.Handle(info.RoutePrefix+"/swagger/",
		http.StripPrefix(info.RoutePrefix,
			http.FileServer(swagger.FS(false)),
		),
	)
	return restfulspec.NewOpenAPIService(config)
}

func enrichSwaggerObject(info SwaggerInfo) func(swo *spec.Swagger) {
	return func(swo *spec.Swagger) {
		contact := &spec.ContactInfo{
			Name:  info.ContactName,
			Email: info.ContactEmail,
			URL:   info.ContactURL,
		}
		license := &spec.License{
			Name: info.LicenseName,
			URL:  info.LicenseURL,
		}
		infoProps := spec.InfoProps{
			Title:          info.Title,
			Description:    info.Description,
			TermsOfService: info.TermsOfService,
			Contact:        contact,
			License:        license,
			Version:        info.Version,
		}
		swo.Info = &spec.Info{
			InfoProps: infoProps,
		}
		swo.Tags = swaggerTags
		swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
			"jwt": spec.APIKeyAuth("Authorization", "header"),
		}
		for _, ws := range restful.RegisteredWebServices() {
			for _, route := range ws.Routes() {
				processAuth(swo, route)
			}
		}
	}
}

func processAuth(swo *spec.Swagger, route restful.Route) {
	_, ok := route.Metadata["jwt"]
	if !ok {
		return
	}
	pOption := getPathOption(swo, route)
	if pOption != nil {
		pOption.SecuredWith("jwt")
	}
}

func getPathOption(swo *spec.Swagger, route restful.Route) *spec.Operation {
	p, err := swo.Paths.JSONLookup(strings.TrimRight(route.Path, "/"))
	if err != nil {
		return nil
	}
	item := p.(*spec.PathItem)
	var pOption *spec.Operation
	switch method := strings.ToLower(route.Method); method {
	case "get":
		pOption = item.Get
	case "post":
		pOption = item.Post
	case "patch":
		pOption = item.Patch
	case "delete":
		pOption = item.Delete
	case "put":
		pOption = item.Put
	case "head":
		pOption = item.Head
	case "options":
		pOption = item.Options
	default:
		return nil
	}
	return pOption
}


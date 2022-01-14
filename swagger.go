package biu

import (
	"embed"
	"net/http"
	"net/url"
	"strings"

	"github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
)

//go:embed swagger/*
var swagger embed.FS

// NewSwaggerService creates a swagger webservice in /swagger
func (c *Container) NewSwaggerService(info SwaggerInfo) *restful.WebService {
	return newSwaggerService(c, info)
}

// NewSwaggerService creates a swagger webservice in /swagger
func NewSwaggerService(info SwaggerInfo) *restful.WebService {
	return newSwaggerService(DefaultContainer, info)
}

func newSwaggerService(
	container *Container,
	info SwaggerInfo,
) *restful.WebService {
	if info.RouteSuffix == "" {
		info.RouteSuffix = "swagger"
	}
	info.RoutePrefix = "/" + strings.Trim(info.RoutePrefix, "/")
	info.RouteSuffix = "/" + strings.Trim(info.RouteSuffix, "/")
	config := restfulspec.Config{
		WebServices:                   container.RegisteredWebServices(),
		APIPath:                       info.RoutePrefix + info.RouteSuffix + ".json",
		DisableCORS:                   info.DisableCORS,
		WebServicesURL:                info.WebServicesURL,
		PostBuildSwaggerObjectHandler: enrichSwaggerObject(container, info, container.ServeMux),
	}
	route := info.RoutePrefix + info.RouteSuffix
	container.ServeMux.Handle(route+"/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if p := strings.TrimPrefix(r.URL.Path, route); len(p) < len(r.URL.Path) {
				r2 := new(http.Request)
				*r2 = *r
				r2.URL = new(url.URL)
				*r2.URL = *r.URL
				r2.URL.Path = "swagger" + p
				http.FileServer(http.FS(swagger)).ServeHTTP(w, r2)
			} else {
				http.NotFound(w, r)
			}
		}),
	)
	return restfulspec.NewOpenAPIService(config)
}

func enrichSwaggerObject(container *Container, info SwaggerInfo, serveMux *http.ServeMux) func(swo *spec.Swagger) {
	return func(swo *spec.Swagger) {
		contact := &spec.ContactInfo{
			ContactInfoProps: spec.ContactInfoProps{
				Name:  info.ContactName,
				Email: info.ContactEmail,
				URL:   info.ContactURL,
			},
		}
		license := &spec.License{
			LicenseProps: spec.LicenseProps{
				Name: info.LicenseName,
				URL:  info.LicenseURL,
			},
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
		swo.Tags = container.swaggerTags[serveMux]
		swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
			"jwt": spec.APIKeyAuth("Authorization", "header"),
		}
		for _, ws := range container.RegisteredWebServices() {
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

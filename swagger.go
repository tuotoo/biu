package biu

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/tuotoo/biu/swagger-go"
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
	}
}

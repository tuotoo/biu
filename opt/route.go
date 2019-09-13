package opt

import (
	"fmt"
	"log"
	"reflect"
	"time"
	"unicode"
	_ "unsafe"

	_ "github.com/mailru/easyjson/gen"
	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/param"
)

//go:linkname camelToSnake github.com/mailru/easyjson/gen.camelToSnake
func camelToSnake(name string) string

// RouteFunc is the type of route options functions.
type RouteFunc func(*Route)

type FieldType int8

const (
	FieldUnknown FieldType = iota
	FieldHeader
	FieldPath
	FieldQuery
	FieldForm
	FieldBody
	FieldReturn
)

func (f FieldType) String() string {
	switch f {
	case FieldHeader:
		return "Header"
	case FieldPath:
		return "Path"
	case FieldQuery:
		return "Query"
	case FieldForm:
		return "Form"
	case FieldBody:
		return "Body"
	default:
		return "Unknown"
	}
}

type ParamOpt struct {
	Name      string
	Type      string
	Format    string
	Desc      string
	IsMulti   bool
	FieldType FieldType
	FieldName string
	Body      interface{}
	Return    interface{}
}

// Route is the options of route.
type Route struct {
	ID                string
	To                func(ctx box.Ctx)
	Auth              bool
	Errors            map[int]string
	EnableAutoPathDoc bool
	ExtraPathDocs     []string
	Params            []ParamOpt
}

// RouteID sets the ID of a route.
func RouteID(id string) RouteFunc {
	return func(route *Route) {
		route.ID = id
	}
}

// RouteTo binds a function to a route.
func RouteTo(f func(ctx box.Ctx)) RouteFunc {
	return func(route *Route) {
		route.To = f
	}
}

func RouteAPI(f interface{}) RouteFunc {
	vf := reflect.ValueOf(f)
	if vf.Kind() != reflect.Func {
		log.Fatal("route argument must be a function")
	}

	t := reflect.TypeOf(f)
	if t.NumIn() <= 0 {
		log.Fatal("route function must at least has a box.Ctx argument")
	}
	first := t.In(0)
	if typeSignature(first) != box.CtxSignature {
		log.Fatal("first argument of route function must be box.Ctx")
	}
	if t.NumIn() < 2 {
		return func(route *Route) {
			route.To = func(ctx box.Ctx) {
				vf.Call([]reflect.Value{
					reflect.ValueOf(ctx),
				})
			}
		}
	}

	second := t.In(1)
	var params []ParamOpt
	if header, ok := second.FieldByName(FieldHeader.String()); ok {
		params = appendParam(header.Type, FieldHeader, params)
	}
	if path, ok := second.FieldByName(FieldPath.String()); ok {
		params = appendParam(path.Type, FieldPath, params)
	}
	if query, ok := second.FieldByName(FieldQuery.String()); ok {
		params = appendParam(query.Type, FieldQuery, params)
	}
	if form, ok := second.FieldByName(FieldForm.String()); ok {
		params = appendParam(form.Type, FieldForm, params)
	}
	if body, ok := second.FieldByName(FieldBody.String()); ok {
		bodyType := body.Type
		if body.Type.Kind() == reflect.Ptr {
			bodyType = bodyType.Elem()
		}
		params = append(params, ParamOpt{
			FieldType: FieldBody,
			Body:      reflect.New(bodyType).Elem().Interface(),
			Desc:      body.Tag.Get("desc"),
		})
	}

	to := func(ctx box.Ctx) {
		sv := reflect.New(second).Elem()
		for _, v := range params {
			switch v.FieldType {
			case FieldBody:
				body := reflect.New(sv.FieldByName(FieldBody.String()).Type()).Interface()
				_ = ctx.Bind(body)
				sv.FieldByName(FieldBody.String()).Set(reflect.ValueOf(body).Elem())
			default:
				setField(sv, ctx, v)
			}
		}
		vf.Call([]reflect.Value{reflect.ValueOf(ctx), sv})
	}
	return func(route *Route) {
		route.To = to
		route.Params = params
	}
}

func setField(sv reflect.Value, ctx box.Ctx, opt ParamOpt) {
	field := sv.FieldByName(opt.FieldType.String()).FieldByName(opt.FieldName)
	var p param.Parameter
	switch opt.FieldType {
	case FieldQuery:
		p = ctx.Query(opt.Name)
	case FieldPath:
		p = ctx.Path(opt.Name)
	case FieldForm:
		p = ctx.Form(opt.Name)
	case FieldHeader:
		p = ctx.Header(opt.Name)
	default:
		return
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString(p.StringDefault(""))
	case reflect.Bool:
		field.SetBool(p.BoolDefault(false))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(p.Int64Default(0))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.SetUint(p.Uint64Default(0))
	case reflect.Float32, reflect.Float64:
		field.SetFloat(p.Float64Default(0))
	case reflect.Struct:
		switch typeSignature(field.Type()) {
		case "time.Time":
			field.Set(reflect.ValueOf(p.TimeDefault(time.RFC3339, time.Time{})))
		case box.FileSignature:
			f, fh, _ := ctx.Req().FormFile(opt.Name)
			field.Set(reflect.ValueOf(box.File{
				Header: fh,
				File:   f,
			}))
		default:
			return
		}
	case reflect.Array, reflect.Slice:
		var rst interface{}
		elem := field.Type().Elem()
		switch elem.Kind() {
		case reflect.String:
			rst, _ = p.StringArray()
		case reflect.Bool:
			rst, _ = p.BoolArray()
		case reflect.Int:
			rst, _ = p.IntArray()
		case reflect.Int8:
			rst, _ = p.Int8Array()
		case reflect.Int16:
			rst, _ = p.Int16Array()
		case reflect.Int32:
			rst, _ = p.Int32Array()
		case reflect.Int64:
			rst, _ = p.Int64Array()
		case reflect.Uint:
			rst, _ = p.UintArray()
		case reflect.Uint8:
			rst, _ = p.Uint8Array()
		case reflect.Uint16:
			rst, _ = p.Uint16Array()
		case reflect.Uint32:
			rst, _ = p.Uint32Array()
		case reflect.Uint64:
			rst, _ = p.Uint64Array()
		case reflect.Float32:
			rst, _ = p.Float32Array()
		case reflect.Float64:
			rst, _ = p.Float64Array()
		case reflect.Struct:
			switch typeSignature(elem) {
			case "time.Time":
				rst, _ = p.TimeArray(time.RFC3339)
			default:
				return
			}
		}
		field.Set(reflect.ValueOf(rst))
	default:
		return
	}
}

func appendParam(t reflect.Type, field FieldType, params []ParamOpt) []ParamOpt {
	for i := 0; i < t.NumField(); i++ {
		typ, format, multi := getBaseType(t.Field(i).Type)
		fieldName := t.Field(i).Name
		name := camelToSnake(fieldName)
		if unicode.IsLower([]rune(fieldName)[0]) {
			continue
		}
		if tagName, ok := t.Field(i).Tag.Lookup("name"); ok {
			name = tagName
		}
		params = append(params, ParamOpt{
			FieldType: field,
			Name:      name,
			Type:      typ,
			Format:    format,
			IsMulti:   multi,
			FieldName: fieldName,
			Desc:      t.Field(i).Tag.Get("desc"),
		})
	}
	return params
}

// EnableAuth enables JWT auth for a route.
func EnableAuth() RouteFunc {
	return func(route *Route) {
		route.Auth = true
	}
}

// RouteErrors defines the errors of a route.
func RouteErrors(m map[int]string) RouteFunc {
	return func(route *Route) {
		route.Errors = m
	}
}

// DisableAuthPathDoc disables auto generate path param docs for route.
func DisableAuthPathDoc() RouteFunc {
	return func(route *Route) {
		route.EnableAutoPathDoc = false
	}
}

// ExtraPathDocs sets extra descriptions for path params.
func ExtraPathDocs(docs ...string) RouteFunc {
	return func(route *Route) {
		route.ExtraPathDocs = docs
	}
}

func getBaseType(t reflect.Type) (typ, format string, multi bool) {
	switch t.Kind() {
	case reflect.String:
		typ = "string"
	case reflect.Bool:
		typ = "boolean"
	case reflect.Int, reflect.Uint,
		reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16:
		typ = "integer"
	case reflect.Int32, reflect.Uint32:
		typ = "integer"
		format = "int32"
	case reflect.Int64, reflect.Uint64:
		typ = "integer"
		format = "int64"
	case reflect.Float32:
		typ = "number"
		format = "float"
	case reflect.Float64:
		typ = "number"
		format = "double"
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() != reflect.Uint8 {
			typ, format, _ = getBaseType(t.Elem())
			return typ, format, true
		}
		return "string", "byte", true
	case reflect.Struct:
		switch typeSignature(t) {
		case "time.Time":
			return "string", "date-time", false
		case box.FileSignature:
			return "file", "", false
		default:
			log.Fatal(fmt.Errorf("type %s currently not support", t.String()))
		}
	default:
		log.Fatal(fmt.Errorf("type %s currently not support", t.String()))
	}
	return typ, format, false
}

func typeSignature(t reflect.Type) string {
	return fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
}

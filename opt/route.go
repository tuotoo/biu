package opt

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"time"
	"unicode"

	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/param"
)

// RouteFunc is the type of route options functions.
type RouteFunc func(*Route)

type Param struct {
	Name    string
	Type    string
	Format  string
	Desc    string
	IsMulti bool
}

// Route is the options of route.
type Route struct {
	ID                string
	To                func(ctx box.Ctx)
	Auth              bool
	Errors            map[int]string
	EnableAutoPathDoc bool
	ExtraPathDocs     []string
	Params            []Param
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

func RouteParam(f interface{}) RouteFunc {
	vf := reflect.ValueOf(f)
	if vf.Kind() != reflect.Func {
		panic("route argument must be a function")
	}

	t := reflect.TypeOf(f)
	if t.NumIn() <= 0 {
		panic("route function must at least has a box.Ctx argument")
	}
	first := t.In(0)
	if fmt.Sprintf("%s.%s", first.PkgPath(), first.Name()) != box.CtxSignature {
		panic("first argument of route function must be box.Ctx")
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
	params := make([]Param, second.NumField())
	for i := 0; i < second.NumField(); i++ {
		typ, format, multi := getBaseType(second.Field(i).Type)
		name := camelToSnake(second.Field(i).Name)
		if tagName, ok := second.Field(i).Tag.Lookup("name"); ok {
			name = tagName
		}
		params[i] = Param{
			Name:    name,
			Type:    typ,
			Format:  format,
			IsMulti: multi,
			Desc:    second.Field(i).Tag.Get("desc"),
		}
	}

	to := func(ctx box.Ctx) {
		sv := reflect.New(second).Elem()
		for i := 0; i < second.NumField(); i++ {
			var p param.Parameter
			switch ctx.Req().Method {
			case http.MethodGet, http.MethodDelete:
				p = ctx.Query(params[i].Name)
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				p = ctx.Form(params[i].Name)
			default:
				continue
			}
			field := sv.Field(i)
			if !field.CanSet() {
				continue
			}
			switch field.Type().Kind() {
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
				switch fmt.Sprintf("%s.%s", field.Type().PkgPath(), field.Type().Name()) {
				case "time.Time":
					field.Set(reflect.ValueOf(p.TimeDefault(time.RFC3339, time.Time{})))
				case box.FileSignature:
					f, fh, _ := ctx.Req().FormFile(params[i].Name)
					field.Set(reflect.ValueOf(box.File{
						Header: fh,
						File:   f,
					}))
				default:
					continue
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
					switch fmt.Sprintf("%s.%s", elem.PkgPath(), elem.Name()) {
					case "time.Time":
						rst, _ = p.TimeArray(time.RFC3339)
					default:
						continue
					}
				}
				field.Set(reflect.ValueOf(rst))
			default:
				continue
			}
		}
		vf.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			sv,
		})
	}
	return func(route *Route) {
		route.To = to
		route.Params = params
	}
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
		switch fmt.Sprintf("%s.%s", t.PkgPath(), t.Name()) {
		case "time.Time":
			return "string", "date-time", false
		case box.FileSignature:
			return "file", "", false
		default:
			panic(fmt.Errorf("type %s currently not support", t.String()))
		}
	default:
		panic(fmt.Errorf("type %s currently not support", t.String()))
	}
	return typ, format, false
}

// camelToSnake from https://github.com/mailru/easyjson
func camelToSnake(name string) string {
	var ret bytes.Buffer

	multipleUpper := false
	var lastUpper rune
	var beforeUpper rune

	for _, c := range name {
		// Non-lowercase character after uppercase is considered to be uppercase too.
		isUpper := unicode.IsUpper(c) || (lastUpper != 0 && !unicode.IsLower(c))

		if lastUpper != 0 {
			// Output a delimiter if last character was either the first uppercase character
			// in a row, or the last one in a row (e.g. 'S' in "HTTPServer").
			// Do not output a delimiter at the beginning of the name.

			firstInRow := !multipleUpper
			lastInRow := !isUpper

			if ret.Len() > 0 && (firstInRow || lastInRow) && beforeUpper != '_' {
				ret.WriteByte('_')
			}
			ret.WriteRune(unicode.ToLower(lastUpper))
		}

		// Buffer uppercase char, do not output it yet as a delimiter may be required if the
		// next character is lowercase.
		if isUpper {
			multipleUpper = lastUpper != 0
			lastUpper = c
			continue
		}

		ret.WriteRune(c)
		lastUpper = 0
		beforeUpper = c
		multipleUpper = false
	}

	if lastUpper != 0 {
		ret.WriteRune(unicode.ToLower(lastUpper))
	}
	return string(ret.Bytes())
}

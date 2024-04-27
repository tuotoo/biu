package opt

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/go-playground/validator/v10"

	"github.com/tuotoo/biu/box"
	"github.com/tuotoo/biu/internal"
	"github.com/tuotoo/biu/param"
)

type defaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

func (v *defaultValidator) validateStruct(ctx context.Context, obj any) error {
	v.lazyInit()
	return v.validate.StructCtx(ctx, obj)
}

func (v *defaultValidator) lazyInit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("vd")
	})
}

var Validator = &defaultValidator{}

type RouteAPIOption struct {
	logger *slog.Logger
}

type RouteAPIOpts func(*RouteAPIOption)

func WithLogger(logger *slog.Logger) RouteAPIOpts {
	return func(o *RouteAPIOption) {
		o.logger = logger
	}
}

func RouteAPI(f any, opts ...RouteAPIOpts) RouteFunc {
	o := &RouteAPIOption{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(o)
	}
	logger := logCaller(o.logger, 2)

	vf := reflect.ValueOf(f)
	if vf.Kind() != reflect.Func {
		logger.Error("route argument must be a function")
		os.Exit(1)
	}

	t := reflect.TypeOf(f)
	if t.NumIn() <= 0 {
		logger.Error("route function must at least has a box.Ctx argument")
		os.Exit(1)
	}
	first := t.In(0)
	if typeSignature(first) != box.CtxSignature {
		logger.Error("first argument of route function must be box.Ctx")
		os.Exit(1)
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
		params = appendParam(appendParamOptions{
			t:      header.Type,
			field:  FieldHeader,
			params: params,
			logger: logger,
		})
	}
	if path, ok := second.FieldByName(FieldPath.String()); ok {
		params = appendParam(appendParamOptions{
			t:      path.Type,
			field:  FieldPath,
			params: params,
			logger: logger,
		})
	}
	if query, ok := second.FieldByName(FieldQuery.String()); ok {
		params = appendParam(appendParamOptions{
			t:      query.Type,
			field:  FieldQuery,
			params: params,
			logger: logger,
		})
	}
	if form, ok := second.FieldByName(FieldForm.String()); ok {
		params = appendParam(appendParamOptions{
			t:      form.Type,
			field:  FieldForm,
			params: params,
			logger: logger,
		})
	}
	if body, ok := second.FieldByName(FieldBody.String()); ok {
		bodyType := body.Type
		if body.Type.Kind() == reflect.Ptr {
			bodyType = bodyType.Elem()
		}
		var bodyExampleValue any
		sig := typeSignature(bodyType)
		switch sig {
		case "io.ReadCloser", "io.Reader":
			bodyExampleValue = ""
		default:
			bodyExampleValue = reflect.New(bodyType).Elem().Interface()
		}
		params = append(params, ParamOpt{
			FieldType: FieldBody,
			Body:      bodyExampleValue,
			Desc:      body.Tag.Get(APITagDesc),
		})
	}
	if ret, ok := second.FieldByName(FieldReturn.String()); ok {
		if ret.Type.Kind() != reflect.Func {
			logger.Error("return must be a function")
			os.Exit(1)
		}
		if ret.Type.NumIn() < 1 {
			logger.Error("return must at least has an argument")
			os.Exit(1)
		}
		params = append(params, ParamOpt{
			FieldType: FieldReturn,
			Return:    reflect.New(ret.Type.In(0)).Interface(),
			Desc:      ret.Tag.Get(APITagDesc),
		})
	}

	to := func(ctx box.Ctx) {
		sv := reflect.New(second).Elem()
		for _, v := range params {
			switch v.FieldType {
			case FieldBody:
				bodyType := sv.FieldByName(FieldBody.String()).Type()
				switch typeSignature(bodyType) {
				case "io.ReadCloser", "io.Reader":
					sv.FieldByName(FieldBody.String()).Set(reflect.ValueOf(ctx.Req().Body))
					continue
				}
				if bodyType.Kind() != reflect.Struct && !(bodyType.Kind() == reflect.Ptr && bodyType.Elem().Kind() == reflect.Struct) {
					setField(sv, ctx, v)
					continue
				}
				body := reflect.New(bodyType).Interface()
				_ = ctx.Bind(body)
				sv.FieldByName(FieldBody.String()).Set(reflect.ValueOf(body).Elem())
			case FieldReturn:
				sv.FieldByName(FieldReturn.String()).Set(reflect.MakeFunc(sv.FieldByName(FieldReturn.String()).Type(),
					func(args []reflect.Value) (results []reflect.Value) {
						if len(args) < 1 {
							return nil
						}
						ctx.ResponseJSON(args[0].Interface())
						return nil
					}))
			default:
				setField(sv, ctx, v)
			}
		}
		if slices.ContainsFunc(params, func(opt ParamOpt) bool {
			return opt.HasVd
		}) {
			if err := Validator.validateStruct(ctx.Req().Context(), sv.Interface()); err != nil {
				ctx.Logger.DebugContext(ctx.Req().Context(), "parameter validate failed",
					slog.Group("route",
						slog.String("id", ctx.RouteID()),
						slog.String("sig", ctx.RouteSignature()),
					),
					slog.Any("error", err),
				)
				ctx.SetAttribute(box.BiuAttrVdErr, err)
			}
		}
		vf.Call([]reflect.Value{reflect.ValueOf(ctx), sv})
	}
	return func(route *Route) {
		route.To = to
		route.ID = internal.NameOfFunction(f)
		route.Params = params
	}
}

func logCaller(logger *slog.Logger, skip int) *slog.Logger {
	_, file, no, _ := runtime.Caller(skip)
	return logger.With(
		slog.Group("source",
			slog.String("file", file),
			slog.Int("line", no),
		),
	)
}

func setField(sv reflect.Value, ctx box.Ctx, opt ParamOpt) {
	var field reflect.Value
	var p param.Parameter
	switch opt.FieldType {
	case FieldQuery:
		p = ctx.Query(opt.Name)
		field = sv.FieldByName(opt.FieldType.String()).FieldByName(opt.FieldName)
	case FieldPath:
		p = ctx.Path(opt.Name)
		field = sv.FieldByName(opt.FieldType.String()).FieldByName(opt.FieldName)
	case FieldForm:
		p = ctx.Form(opt.Name)
		field = sv.FieldByName(opt.FieldType.String()).FieldByName(opt.FieldName)
	case FieldHeader:
		p = ctx.Header(opt.Name)
		field = sv.FieldByName(opt.FieldType.String()).FieldByName(opt.FieldName)
	case FieldBody:
		bodyBs, _ := io.ReadAll(ctx.Req().Body)
		p = param.NewParameter([]string{string(bodyBs)}, nil)
		field = sv.FieldByName(opt.FieldType.String())
	default:
		return
	}
	switch field.Kind() {
	case reflect.Ptr:
		setPtr(field, p)
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
		var rst any
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
		case reflect.Uint8: // bytes
			rst, _ = p.Bytes()
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
		default:
			return
		}
		field.Set(reflect.ValueOf(rst))
	default:
		return
	}
}

func setPtr(field reflect.Value, p param.Parameter) {
	switch field.Type().Elem().Kind() {
	case reflect.String:
		v, err := p.String()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Bool:
		v, err := p.Bool()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Int:
		v, err := p.Int()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Int8:
		v, err := p.Int8()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Int16:
		v, err := p.Int16()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Int32:
		v, err := p.Int32()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Int64:
		v, err := p.Int64()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Uint:
		v, err := p.Uint()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Uint8:
		v, err := p.Uint8()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Uint16:
		v, err := p.Uint16()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Uint32:
		v, err := p.Uint32()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Uint64:
		v, err := p.Uint64()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Float32:
		v, err := p.Float32()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	case reflect.Float64:
		v, err := p.Float64()
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&v))
	default:
		return
	}
}

type appendParamOptions struct {
	t      reflect.Type
	field  FieldType
	params []ParamOpt
	logger *slog.Logger
}

func appendParam(o appendParamOptions) []ParamOpt {
	for i := 0; i < o.t.NumField(); i++ {
		tags := make(map[string]string)
		if cfg, ok := o.t.Field(i).Tag.Lookup("biu"); ok {
			items := strings.Split(cfg, ";")
			for _, item := range items {
				sp := strings.Split(item, ":")
				if len(sp) > 1 {
					tags[sp[0]] = sp[1]
				} else {
					tags[sp[0]] = ""
				}
			}
		}
		_, hasVd := o.t.Field(i).Tag.Lookup("vd")
		if _, ok := tags[APITagIgnore]; ok {
			continue
		}
		fieldName := o.t.Field(i).Name
		name := internal.CamelToSnake(fieldName)
		if unicode.IsLower([]rune(fieldName)[0]) {
			continue
		}
		if tagName, ok := tags[APITagName]; ok {
			name = tagName
		}
		typ, err := getBaseType(o.t.Field(i).Type)
		if err != nil {
			o.logger.Warn("get parameter type failed", slog.Any("error", err))
			continue
		}
		if tagFormat, ok := tags[APITagFormat]; ok {
			typ.format = tagFormat
		}
		o.params = append(o.params, ParamOpt{
			FieldType: o.field,
			Name:      name,
			Type:      typ.typ,
			Format:    typ.format,
			IsMulti:   typ.multi,
			FieldName: fieldName,
			Desc:      tags[APITagDesc],
			HasVd:     hasVd,
		})
	}
	return o.params
}

type baseType struct {
	typ    string
	format string
	multi  bool
}

func getBaseType(t reflect.Type) (*baseType, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	var typ, format string
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
			baseType, err := getBaseType(t.Elem())
			if err != nil {
				return nil, err
			}
			baseType.multi = true
			return baseType, nil
		}
		return &baseType{
			typ:    "string",
			format: "byte",
			multi:  true,
		}, nil
	case reflect.Struct:
		switch typeSignature(t) {
		case "time.Time":
			return &baseType{
				typ:    "string",
				format: "date-time",
			}, nil
		case box.FileSignature:
			return &baseType{
				typ: "file",
			}, nil
		default:
			return nil, fmt.Errorf("type %s currently not support", typeSignature(t))
		}
	default:
		return nil, fmt.Errorf("type %s currently not support", t.String())
	}
	return &baseType{
		typ:    typ,
		format: format,
	}, nil
}

func typeSignature(t reflect.Type) string {
	return fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
}

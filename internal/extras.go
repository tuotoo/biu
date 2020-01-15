package internal

import (
	"regexp"
	_ "unsafe"

	_ "github.com/emicklei/go-restful/v3"
	_ "github.com/mailru/easyjson/gen"
)

type pathExpression struct {
	LiteralCount int      // the number of literal characters (means those not resulting from template variable substitution)
	VarNames     []string // the names of parameters (enclosed by {}) in the path
	VarCount     int      // the number of named parameters (enclosed by {}) in the path
	Matcher      *regexp.Regexp
	Source       string // Path as defined by the RouteBuilder
	tokens       []string
}

//go:linkname newPathExpression github.com/emicklei/go-restful/v3.newPathExpression
func newPathExpression(path string) (*pathExpression, error)

func NewPathExpression(path string) (*pathExpression, error) {
	return newPathExpression(path)
}

//go:linkname nameOfFunction github.com/emicklei/go-restful/v3.nameOfFunction
func nameOfFunction(f interface{}) string

func NameOfFunction(f interface{}) string {
	return nameOfFunction(f)
}

//go:linkname camelToSnake github.com/mailru/easyjson/gen.camelToSnake
func camelToSnake(name string) string

func CamelToSnake(name string) string {
	return camelToSnake(name)
}

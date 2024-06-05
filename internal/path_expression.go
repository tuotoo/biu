package internal

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/emicklei/go-restful/v3"
)

// PathExpression holds a compiled path expression (RegExp) needed to match against
// Http request paths and to extract path parameter values.
// COPY FROM: github.com/emicklei/go-restful/v3.newPathExpression
type PathExpression struct {
	LiteralCount int      // the number of literal characters (means those not resulting from template variable substitution)
	VarNames     []string // the names of parameters (enclosed by {}) in the path
	VarCount     int      // the number of named parameters (enclosed by {}) in the path
	Matcher      *regexp.Regexp
	Source       string // Path as defined by the RouteBuilder
	tokens       []string
}

// NewPathExpression creates a PathExpression from the input URL path.
// Returns an error if the path is invalid.
func NewPathExpression(path string) (*PathExpression, error) {
	expression, literalCount, varNames, varCount, tokens := templateToRegularExpression(path)
	compiled, err := regexp.Compile(expression)
	if err != nil {
		return nil, err
	}
	return &PathExpression{literalCount, varNames, varCount, compiled, expression, tokens}, nil
}

// http://jsr311.java.net/nonav/releases/1.1/spec/spec3.html#x3-370003.7.3
func templateToRegularExpression(template string) (expression string, literalCount int, varNames []string, varCount int, tokens []string) {
	var buffer bytes.Buffer
	buffer.WriteString("^")
	// tokens = strings.Split(template, "/")
	tokens = tokenizePath(template)
	for _, each := range tokens {
		if each == "" {
			continue
		}
		buffer.WriteString("/")
		if strings.HasPrefix(each, "{") {
			// check for regular expression in variable
			colon := strings.Index(each, ":")
			var varName string
			if colon != -1 {
				// extract expression
				varName = strings.TrimSpace(each[1:colon])
				paramExpr := strings.TrimSpace(each[colon+1 : len(each)-1])
				if paramExpr == "*" { // special case
					buffer.WriteString("(.*)")
				} else {
					buffer.WriteString(fmt.Sprintf("(%s)", paramExpr)) // between colon and closing moustache
				}
			} else {
				// plain var
				varName = strings.TrimSpace(each[1 : len(each)-1])
				buffer.WriteString("([^/]+?)")
			}
			varNames = append(varNames, varName)
			varCount += 1
		} else {
			literalCount += len(each)
			encoded := each // TODO URI encode
			buffer.WriteString(regexp.QuoteMeta(encoded))
		}
	}
	return strings.TrimRight(buffer.String(), "/") + "(/.*)?$", literalCount, varNames, varCount, tokens
}

// Tokenize a URL path using the slash separator ; the result does not have empty tokens
func tokenizePath(path string) []string {
	if "/" == path {
		return nil
	}
	if restful.TrimRightSlashEnabled {
		// 3.9.0
		return strings.Split(strings.Trim(path, "/"), "/")
	} else {
		// 3.10.2
		return strings.Split(strings.TrimLeft(path, "/"), "/")
	}
}

package log

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
)

type ILogger interface {
	Info(i interface{})
	Fatal(i interface{})
}

type BiuInternalInfo struct {
	Err    error
	Extras map[string]interface{}
}

type DefaultLogger struct{}

func (dl DefaultLogger) Info(i interface{}) {
	if i == nil {
		return
	}
	s, err := dl.toStr(i)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println(s)
}

func (dl DefaultLogger) Fatal(i interface{}) {
	if i == nil {
		log.Fatal()
		return
	}
	s, err := dl.toStr(i)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Fatal(s)
}

func (dl DefaultLogger) toStr(i interface{}) (string, error) {
	switch v := i.(type) {
	case BiuInternalInfo:
		var s []string
		if v.Err != nil {
			s = append(s, fmt.Sprintf("[ERR] %+v", v.Err))
		}
		if len(v.Extras) != 0 {
			keys, i := make([]string, len(v.Extras)), 0
			for k := range v.Extras {
				keys[i] = k
				i++
			}
			sort.Strings(keys)
			extras := make([]string, len(v.Extras))
			for i, k := range keys {
				extras[i] = fmt.Sprintf("%s: %+v", k, v.Extras[k])
			}
			s = append(s, fmt.Sprintf("[INFO] %s", strings.Join(extras, "\t")))
		}
		return strings.Join(s, "\n"), nil
	case string:
		return v, nil
	case error:
		return fmt.Sprintf("%+v", v), v
	default:
		return "", errors.New("type not found")
	}
}

package biu

import (
	"io/ioutil"

	"github.com/casbin/casbin"
	"github.com/casbin/casbin/model"
	"github.com/tuotoo/biu/rule-go"
)

func NewRBACModel() (model.Model, error) {
	m := casbin.NewModel()
	fs := rule.FS(false)
	file, err := fs.Open("/rule/rbac_model.conf")
	if err != nil {
		return m, err
	}
	s, err := ioutil.ReadAll(file)
	if err != nil {
		return m, err
	}
	m.LoadModelFromText(string(s))
	return m, nil
}

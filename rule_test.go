package biu_test

import (
	"github.com/casbin/casbin"
	"github.com/tuotoo/biu"
)

func ExampleNewRBACModel() {
	model, err := biu.NewRBACModel()
	if err != nil {
		panic(err)
	}
	casbin.NewEnforcer(model, "policy.csv")
}

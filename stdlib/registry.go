package stdlib

import "Kairo/value"

type BuiltinModule struct {
	Exports map[string]value.Value
}

var BuiltinModules = map[string]BuiltinModule{}

func RegisterModule(name string, module BuiltinModule) {
	BuiltinModules[name] = module
}

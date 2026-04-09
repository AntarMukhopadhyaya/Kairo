package vm

import (
	"Kairo/value"
	"fmt"
)

func EnsureBuiltinSlots(slots map[string]int) {
	if _, ok := slots["print"]; !ok {
		slots["print"] = len(slots)
	}
	if _, ok := slots["len"]; !ok {
		slots["len"] = len(slots)
	}
}

func NewGlobals(slots map[string]int) []VariableInfo {
	EnsureBuiltinSlots(slots)
	globals := make([]VariableInfo, len(slots))
	RegisterBuiltins(globals, slots)
	return globals
}

func EnsureGlobalsSize(globals []VariableInfo, slots map[string]int) []VariableInfo {
	EnsureBuiltinSlots(slots)
	if len(globals) >= len(slots) {
		return globals
	}
	newGlobals := make([]VariableInfo, len(slots))
	copy(newGlobals, globals)
	RegisterBuiltins(newGlobals, slots)
	return newGlobals
}

func RegisterBuiltins(globals []VariableInfo, slots map[string]int) {
	EnsureBuiltinSlots(slots)

	globals[slots["print"]] = VariableInfo{
		Value: value.MakeInternalFunction(&value.InternalFunctionObject{
			Arity: -1, // variadic
			Call: func(vm value.CallContext, args []value.Value) value.Value {
				for _, arg := range args {
					fmt.Print(arg.ToString(), " ")
				}
				fmt.Println()
				return value.MakeNull()
			},
		}),
		Type: "function",
	}

	globals[slots["len"]] = VariableInfo{
		Value: value.MakeInternalFunction(&value.InternalFunctionObject{
			Arity: 1,
			Call: func(vm value.CallContext, args []value.Value) value.Value {
				if len(args) != 1 {
					return value.MakeError(
						"len() takes exactly one argument",
						"ArgumentError",
						0,
						0,
					)
				}

				switch args[0].Kind {
				case value.StringKind, value.RopeStringKind:
					return value.MakeNumber(float64(value.StringLen(args[0])))
				case value.ArrayKind:
					return value.MakeNumber(float64(len(args[0].AsArray().Elements)))
				case value.MapKind:
					return value.MakeNumber(float64(len(args[0].AsMap().Properties)))
				default:
					return value.MakeError(
						"len() argument must be a string, array, or map",
						"TypeError",
						0,
						0,
					)
				}
			},
		}),
		Type: "function",
	}
}

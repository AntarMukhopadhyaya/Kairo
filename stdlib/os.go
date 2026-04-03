package stdlib

import (
	"Kairo/value"
	"os"
)

func init() {
	RegisterModule("os", BuiltinModule{
		Exports: map[string]value.Value{
			"getenv": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("getenv() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if !value.IsStringLike(args[0]) {
						return value.MakeError("getenv() argument must be a string", "TypeError", 0, 0)
					}
					return value.MakeString(os.Getenv(args[0].ToString()))
				},
			}),
			"cwd": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 0,
				Call: func(args []value.Value) value.Value {
					if len(args) != 0 {
						return value.MakeError("cwd() takes no arguments", "ArgumentError", 0, 0)
					}
					cwd, err := os.Getwd()
					if err != nil {
						return value.MakeError("cwd() failed", "RuntimeError", 0, 0)
					}
					return value.MakeString(cwd)
				},
			}),
			"args": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 0,
				Call: func(args []value.Value) value.Value {
					if len(args) != 0 {
						return value.MakeError("args() takes no arguments", "ArgumentError", 0, 0)
					}
					elements := make([]value.Value, 0, len(os.Args))
					for _, arg := range os.Args {
						elements = append(elements, value.MakeString(arg))
					}
					return value.MakeArray(elements)
				},
			}),
		},
	})
}

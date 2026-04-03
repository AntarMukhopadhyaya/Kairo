package stdlib

import (
	"Kairo/value"
	"math"
)

func init() {
	RegisterModule("math", BuiltinModule{
		Exports: map[string]value.Value{
			"sqrt": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("sqrt() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("sqrt() argument must be a number", "TypeError", 0, 0)
					}
					return value.MakeNumber(math.Sqrt(args[0].Num))
				},
			}),
			"sin": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("sin() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("sin() argument must be a number", "TypeError", 0, 0)
					}
					return value.MakeNumber(math.Sin(args[0].Num))
				},
			}),
			"cos": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("cos() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("cos() argument must be a number", "TypeError", 0, 0)
					}
					return value.MakeNumber(math.Cos(args[0].Num))
				},
			}),
			"floor": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("floor() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("floor() argument must be a number", "TypeError", 0, 0)
					}
					return value.MakeNumber(math.Floor(args[0].Num))
				},
			}),
			"ceil": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("ceil() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("ceil() argument must be a number", "TypeError", 0, 0)
					}
					return value.MakeNumber(math.Ceil(args[0].Num))
				},
			}),
			"abs": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("abs() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("abs() argument must be a number", "TypeError", 0, 0)
					}
					return value.MakeNumber(math.Abs(args[0].Num))
				},
			}),
		},
	})
}

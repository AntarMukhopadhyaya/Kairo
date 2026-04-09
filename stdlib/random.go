package stdlib

import (
	"Kairo/value"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	RegisterModule("random", BuiltinModule{
		Exports: map[string]value.Value{
			"random": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 0,
				Call: func(vm value.CallContext, args []value.Value) value.Value {
					if len(args) != 0 {
						return value.MakeError("random() takes no arguments", "ArgumentError", 0, 0)
					}
					return value.MakeNumber(rand.Float64())
				},
			}),
			"randint": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(vm value.CallContext, args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("randint() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("randint() argument must be a number", "TypeError", 0, 0)
					}
					if args[0].Num <= 0 {
						return value.MakeError("randint() argument must be > 0", "RangeError", 0, 0)
					}
					return value.MakeNumber(float64(rand.Intn(int(args[0].Num))))
				},
			}),
			"seed": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(vm value.CallContext, args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("seed() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("seed() argument must be a number", "TypeError", 0, 0)
					}
					rand.Seed(int64(args[0].Num))
					return value.MakeNull()
				},
			}),
		},
	})
}

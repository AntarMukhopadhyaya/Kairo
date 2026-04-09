package stdlib

import (
	"Kairo/value"
	crand "crypto/rand"
	"encoding/hex"
	"math/big"
)

func init() {
	RegisterModule("crypto", BuiltinModule{
		Exports: map[string]value.Value{
			"randomHex": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(vm value.CallContext, args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("randomHex() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("randomHex() argument must be a number", "TypeError", 0, 0)
					}
					if args[0].Num <= 0 {
						return value.MakeError("randomHex() argument must be > 0", "RangeError", 0, 0)
					}
					n := int(args[0].Num)
					buf := make([]byte, n)
					if _, err := crand.Read(buf); err != nil {
						return value.MakeError("randomHex() failed", "RuntimeError", 0, 0)
					}
					return value.MakeString(hex.EncodeToString(buf))
				},
			}),
			"randomInt": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call: func(vm value.CallContext, args []value.Value) value.Value {
					if len(args) != 1 {
						return value.MakeError("randomInt() takes exactly one argument", "ArgumentError", 0, 0)
					}
					if args[0].Kind != value.NumberKind {
						return value.MakeError("randomInt() argument must be a number", "TypeError", 0, 0)
					}
					if args[0].Num <= 0 {
						return value.MakeError("randomInt() argument must be > 0", "RangeError", 0, 0)
					}
					max := big.NewInt(int64(args[0].Num))
					n, err := crand.Int(crand.Reader, max)
					if err != nil {
						return value.MakeError("randomInt() failed", "RuntimeError", 0, 0)
					}
					return value.MakeNumber(float64(n.Int64()))
				},
			}),
		},
	})
}

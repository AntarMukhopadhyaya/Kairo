package stdlib

import (
	"Kairo/value"
	"strings"
)

func init() {
	RegisterModule("string", BuiltinModule{
		Exports: map[string]value.Value{
			"upper": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call:  upperFn,
			}),
			"lower": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call:  lowerFn,
			}),
			"strip": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: 1,
				Call:  stripFn,
			}),
			"split": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: -1, // 1 or 2
				Call:  splitFn,
			}),
			"join": value.MakeInternalFunction(&value.InternalFunctionObject{
				Arity: -1, // 1 or 2
				Call:  joinFn,
			}),
		},
	})
}
func upperFn(_ value.CallContext, args []value.Value) value.Value {
	if len(args) != 1 {
		return value.MakeError("upper() takes exactly one argument", "ArgumentError", 0, 0)
	}
	if !value.IsStringLike(args[0]) {
		return value.MakeError("upper() argument must be a string", "TypeError", 0, 0)
	}
	str := args[0].ToString()
	return value.MakeString(strings.ToUpper(str))
}
func lowerFn(_ value.CallContext, args []value.Value) value.Value {
	if len(args) != 1 {
		return value.MakeError("lower() takes exactly one argument", "ArgumentError", 0, 0)
	}
	if !value.IsStringLike(args[0]) {
		return value.MakeError("lower() argument must be a string", "TypeError", 0, 0)
	}
	str := args[0].ToString()
	return value.MakeString(strings.ToLower(str))
}
func stripFn(_ value.CallContext, args []value.Value) value.Value {
	if len(args) != 1 {
		return value.MakeError("strip() takes exactly one string argument", "ArgumentError", 0, 0)
	}
	if !value.IsStringLike(args[0]) {
		return value.MakeError("strip() argument must be a string", "TypeError", 0, 0)
	}
	str := args[0].ToString()
	return value.MakeString(strings.TrimSpace(str))
}
func splitFn(_ value.CallContext, args []value.Value) value.Value {
	if len(args) < 1 || len(args) > 2 {
		return value.MakeError("split() takes one or two string arguments", "ArgumentError", 0, 0)
	}
	if !value.IsStringLike(args[0]) {
		return value.MakeError("split() first argument must be a string", "TypeError", 0, 0)
	}
	str := args[0].ToString()
	sep := " "
	if len(args) == 2 {
		if !value.IsStringLike(args[1]) {
			return value.MakeError("split() second argument must be a string", "TypeError", 0, 0)
		}
		sep = args[1].ToString()
	}
	parts := strings.Split(str, sep)
	values := make([]value.Value, len(parts))
	for i, part := range parts {
		values[i] = value.MakeString(part)
	}
	return value.MakeArray(values)
}
func joinFn(_ value.CallContext, args []value.Value) value.Value {
	if len(args) < 1 || len(args) > 2 {
		return value.MakeError("join() takes one or two arguments", "ArgumentError", 0, 0)
	}
	if args[0].Kind != value.ArrayKind {
		return value.MakeError("join() first argument must be an array", "TypeError", 0, 0)
	}
	if len(args) == 2 && !value.IsStringLike(args[1]) {
		return value.MakeError("join() second argument must be a string", "TypeError", 0, 0)
	}
	arr := args[0].AsArray()
	sep := ""
	if len(args) == 2 {
		sep = args[1].ToString()
	}
	strs := make([]string, len(arr.Elements))
	for i, elem := range arr.Elements {
		if !value.IsStringLike(elem) {
			return value.MakeError("join() array elements must be strings", "TypeError", 0, 0)
		}
		strs[i] = elem.ToString()
	}
	return value.MakeString(strings.Join(strs, sep))
}

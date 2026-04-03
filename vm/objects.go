package vm

import "Kairo/value"

// ClosureObject is the heap-backed callable for user-defined functions.
type ClosureObject struct {
	Function *FunctionObject
	Upvalues []*Upvalue
}

// Backward-compatible alias.
type Closure = ClosureObject

func MakeFunction(fn *FunctionObject) value.Value {
	return value.Value{Kind: value.FunctionKind, Obj: fn}
}

func MakeClosure(c *ClosureObject) value.Value {
	return value.Value{Kind: value.ClosureKind, Obj: c}
}

func AsFunction(v value.Value) *FunctionObject {
	if v.Kind != value.FunctionKind {
		panic("not a function")
	}
	return v.Obj.(*FunctionObject)
}

func AsInternalFunction(v value.Value) *value.InternalFunctionObject {
	// Deprecated: prefer v.AsInternalFunction() from the value package.
	return v.AsInternalFunction()
}

func AsClosure(v value.Value) *ClosureObject {
	if v.Kind != value.ClosureKind {
		panic("not a closure")
	}
	return v.Obj.(*ClosureObject)
}

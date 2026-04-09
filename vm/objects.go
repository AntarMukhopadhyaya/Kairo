package vm

import "Kairo/value"

// ClosureObject is the heap-backed callable for user-defined functions.
type ClosureObject struct {
	Function *FunctionObject
	Upvalues []*Upvalue
}

// Backward-compatible alias.
type Closure = ClosureObject

func init() {
	value.ExternalFreeObject = freeExternalHeapObject
}

func freeExternalHeapObject(obj *value.HeapObject) {
	switch data := obj.Data.(type) {
	case *ClosureObject:
		for _, up := range data.Upvalues {
			DecUpvalue(up)
		}
	}
}

func MakeFunction(fn *FunctionObject) value.Value {
	return value.Value{Kind: value.FunctionKind, Obj: &value.HeapObject{RefCount: 0, Data: fn}}
}

func MakeClosure(c *ClosureObject) value.Value {
	return value.Value{Kind: value.ClosureKind, Obj: &value.HeapObject{RefCount: 0, Data: c}}
}

func AsFunction(v value.Value) *FunctionObject {
	if v.Kind != value.FunctionKind {
		panic("not a function")
	}
	return v.Obj.Data.(*FunctionObject)
}

func AsInternalFunction(v value.Value) *value.InternalFunctionObject {
	// Deprecated: prefer v.AsInternalFunction() from the value package.
	return v.AsInternalFunction()
}

func AsClosure(v value.Value) *ClosureObject {
	if v.Kind != value.ClosureKind {
		panic("not a closure")
	}
	return v.Obj.Data.(*ClosureObject)
}


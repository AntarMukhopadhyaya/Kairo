package vm

import "Kairo/value"

// CallFunction invokes any callable value with the provided arguments.
// It is exposed to internal functions via value.CallContext.
func (vm *VM) CallFunction(fn value.Value, args []value.Value) value.Value {
	switch fn.Kind {
	case value.InternalFunctionKind:
		internal := fn.AsInternalFunction()
		if internal.Arity != -1 && internal.Arity != len(args) {
			return value.MakeError("wrong number of arguments", "ArgumentError", 0, 0)
		}
		return internal.Call(vm, args)
	case value.FunctionKind, value.ClosureKind:
		return vm.callViaTrampoline(fn, args)
	default:
		return value.MakeError("not callable", "TypeError", 0, 0)
	}
}

func (vm *VM) callViaTrampoline(callee value.Value, args []value.Value) value.Value {
	child := NewVM(vm.Globals)
	child.SetSourceName(vm.sourceName)

	chunk := NewChunk()
	calleeIdx := chunk.AddConstant(callee)
	chunk.Emit(OP_LOAD_CONST, 0, calleeIdx, 0)

	argStart := 1
	for i, arg := range args {
		idx := chunk.AddConstant(arg)
		chunk.Emit(OP_LOAD_CONST, argStart+i, idx, 0)
	}

	dest := argStart + len(args)
	chunk.Emit(OP_CALL, dest, 0, argStart, len(args))
	chunk.Emit(OP_RETURN, dest, 0, 0)

	entryFn := &FunctionObject{
		Chunk:        chunk,
		Arity:        0,
		Name:         "<callback>",
		UpvalueCount: 0,
		Upvalues:     nil,
		ParamTypes:   nil,
		ReturnType:   "",
		MaxRegisters: dest + 1,
	}
	entry := &ClosureObject{Function: entryFn, Upvalues: nil}
	return child.Run(entry)
}

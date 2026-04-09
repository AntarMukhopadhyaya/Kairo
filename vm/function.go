package vm

type UpvalueDescriptor struct {
	Index   int
	IsLocal bool
}

// FunctionObject is the heap-backed representation of a user-defined function.
// It is referenced from value.Value (Kind == value.FunctionKind) and from closures.
type FunctionObject struct {
	Chunk        *Chunk
	Arity        int
	Name         string
	UpvalueCount int
	Upvalues     []UpvalueDescriptor
	ParamTypes   []string
	ReturnType   string
	MaxRegisters int
}

// Backward-compatible alias.
type Function = FunctionObject


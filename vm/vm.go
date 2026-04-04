package vm

import (
	"Kairo/value"
	"fmt"
	"math"
	"sort"
	"strings"
)

type VM struct {
	frames         []CallFrame
	Globals        []VariableInfo
	handlers       []ExceptionHandler
	modules        map[string]value.Value
	sourceName     string
	regFreeList    [][]value.Value
	methodRegistry map[value.ValueKind]map[string]*value.InternalFunctionObject
	profiler       *InstructionProfiler
}

type InstructionProfileEntry struct {
	Op    OpCode
	Name  string
	Count uint64
}

// InstructionProfiler counts how many times each opcode executes.
// This is a lightweight sampling-free profiler for guiding later optimizations.
type InstructionProfiler struct {
	counts [OpCodeCount]uint64
	total  uint64
}

func (p *InstructionProfiler) hit(op OpCode) {
	idx := int(op)
	if idx >= 0 && idx < len(p.counts) {
		p.counts[idx]++
		p.total++
	}
}

func (p *InstructionProfiler) Total() uint64 {
	return p.total
}

func (p *InstructionProfiler) EntriesSortedDesc() []InstructionProfileEntry {
	entries := make([]InstructionProfileEntry, 0, OpCodeCount)
	for i := 0; i < OpCodeCount; i++ {
		c := p.counts[i]
		if c == 0 {
			continue
		}
		op := OpCode(i)
		entries = append(entries, InstructionProfileEntry{Op: op, Name: OpCodeName(op), Count: c})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})
	return entries
}

// ControlSignal types - what control flow is pending after finally executes
const (
	SIGNAL_NONE     = 0 // No pending control transfer
	SIGNAL_BREAK    = 1 // Break statement pending
	SIGNAL_CONTINUE = 2 // Continue statement pending
	SIGNAL_RETURN   = 3 // Return statement pending
)

type ControlSignal struct {
	Type     int         // SIGNAL_NONE, SIGNAL_BREAK, SIGNAL_CONTINUE, SIGNAL_RETURN
	Value    value.Value // For SIGNAL_RETURN, the return value
	ResumeIP int         // Where to jump after finally executes
}

type ExceptionHandler struct {
	frameIndex    int            // Which call frame this handler belongs to
	handlerIP     int            // Address of catch block
	finallyIP     int            // Address of finally block (-1 if no finally)
	errorReg      int            // Register to store error value
	pendingSignal *ControlSignal // Control transfer pending (break/continue/return)
}
type VariableInfo struct {
	Value value.Value
	Type  string
}

type Upvalue struct {
	location *value.Value
	closed   value.Value
}

func (frame *CallFrame) captureUpvalue(localReg int) *Upvalue {
	if frame.openUpvalues == nil {
		frame.openUpvalues = make(map[int]*Upvalue)
	}
	if up, ok := frame.openUpvalues[localReg]; ok {
		return up
	}
	up := &Upvalue{location: &frame.regs[localReg]}
	frame.openUpvalues[localReg] = up
	return up
}

func (frame *CallFrame) closeUpvalues() {
	if frame.openUpvalues == nil {
		return
	}
	for _, up := range frame.openUpvalues {
		up.closed = *up.location
		up.location = &up.closed
	}
	frame.openUpvalues = nil
}

func NewVM(globals []VariableInfo) *VM {
	vm := &VM{
		frames:         []CallFrame{},
		Globals:        globals,
		modules:        make(map[string]value.Value),
		sourceName:     "<unknown>",
		regFreeList:    [][]value.Value{},
		methodRegistry: initMethodRegistry(),
	}
	return vm
}

func (vm *VM) SetSourceName(name string) {
	if name == "" {
		vm.sourceName = "<unknown>"
		return
	}
	vm.sourceName = name
}

func (vm *VM) EnableInstructionProfiling(enabled bool) {
	if enabled {
		vm.profiler = &InstructionProfiler{}
		return
	}
	vm.profiler = nil
}

func (vm *VM) InstructionProfiler() *InstructionProfiler {
	return vm.profiler
}

func (vm *VM) allocRegs(size int) []value.Value {
	for len(vm.regFreeList) > 0 {
		last := len(vm.regFreeList) - 1
		buf := vm.regFreeList[last]
		vm.regFreeList = vm.regFreeList[:last]
		if cap(buf) >= size {
			return buf[:size]
		}
		// Too small; discard and keep searching.
	}
	return make([]value.Value, size)
}

func (vm *VM) freeRegs(regs []value.Value) {
	clear(regs)
	vm.regFreeList = append(vm.regFreeList, regs[:0])
}

func (vm *VM) ensureGlobalsSize(size int) {
	if len(vm.Globals) >= size {
		return
	}
	newGlobals := make([]VariableInfo, size)
	copy(newGlobals, vm.Globals)
	vm.Globals = newGlobals
}

func withInstrLocation(errVal value.Value, instr Instruction) value.Value {
	if errVal.Kind != value.ErrorKind {
		return errVal
	}
	errObj := errVal.AsError()
	if errObj.Line > 0 || errObj.Column > 0 {
		return errVal
	}
	return value.MakeError(errObj.Message, errObj.ErrorType, instr.Line, instr.Column)
}

func runtimeErrorAt(instr Instruction, message, errorType string) value.Value {
	return value.MakeError(message, errorType, instr.Line, instr.Column)
}

func (vm *VM) buildStackTrace() []string {
	trace := []string{}
	for i := len(vm.frames) - 1; i >= 0; i-- {
		frame := vm.frames[i]
		fn := frame.closure.Function
		name := fn.Name
		if name == "" {
			name = "<anonymous>"
		}
		ip := frame.ip
		// Try to get instruction for line info
		var line, col int
		if ip > 0 && ip-1 < len(fn.Chunk.Code) {
			instr := fn.Chunk.Code[ip-1]
			line = instr.Line
			col = instr.Column
		}

		trace = append(trace,
			fmt.Sprintf("at %s (%s:%d:%d)", name, vm.sourceName, line, col),
		)
	}
	return trace
}

// handleError checks if there's an exception handler for the current frame
// and jumps to the catch block if one exists. Returns true if error was handled.
func (vm *VM) handleError(errorVal value.Value) bool {
	if len(vm.frames) == 0 {
		return false
	}

	if errorVal.Kind != value.ErrorKind {
		errorVal = value.MakeError(errorVal.ToString(), "RuntimeError", 0, 0)
	}

	errObj := errorVal.AsError()
	if errObj.StackTrace == nil {
		errObj.StackTrace = vm.buildStackTrace()
	}

	currentFrameIdx := len(vm.frames) - 1

	// Search for a handler in the current frame
	for i := len(vm.handlers) - 1; i >= 0; i-- {
		handler := vm.handlers[i]
		if handler.frameIndex == currentFrameIdx {
			// Found a handler for this frame
			frame := &vm.frames[currentFrameIdx]

			// Store the error in the error register (if specified)
			if handler.errorReg >= 0 && handler.errorReg < len(frame.regs) {
				frame.regs[handler.errorReg] = errorVal
			}

			// Jump to the catch handler
			frame.ip = handler.handlerIP

			// Remove this handler (it's been triggered)
			vm.handlers = append(vm.handlers[:i], vm.handlers[i+1:]...)

			return true
		}
	}

	return false
}

// findActiveHandlerForFrame finds an active exception handler for the current frame
// that has a finally block. Returns the handler index and whether it exists.
func (vm *VM) findActiveHandlerForFrame() (int, bool) {
	currentFrameIdx := len(vm.frames) - 1
	// Walk backwards through handlers to find the most recent one for this frame
	for i := len(vm.handlers) - 1; i >= 0; i-- {
		if vm.handlers[i].frameIndex == currentFrameIdx && vm.handlers[i].finallyIP >= 0 {
			return i, true
		}
	}
	return -1, false
}

func (vm *VM) Run(closure *ClosureObject) value.Value {
	frame := CallFrame{
		closure: closure,
		ip:      0,
		regs:    vm.allocRegs(closure.Function.MaxRegisters),
	}
	vm.frames = append(vm.frames, frame)

	for len(vm.frames) > 0 {
		frame := &vm.frames[len(vm.frames)-1]
		if frame.ip >= len(frame.closure.Function.Chunk.Code) {
			vm.frames = vm.frames[:len(vm.frames)-1]
			if len(vm.frames) == 0 {
				vm.freeRegs(frame.regs)
				return value.MakeNull()
			}
			vm.freeRegs(frame.regs)
			continue
		}

		instr := frame.closure.Function.Chunk.Code[frame.ip]
		frame.ip++
		if vm.profiler != nil {
			vm.profiler.hit(instr.Op)
		}

		switch instr.Op {
		case OP_LOAD_CONST:
			frame.regs[instr.A] = frame.closure.Function.Chunk.Constants[instr.B]

		case OP_GET_GLOBAL:
			slot := instr.B
			if slot >= 0 && slot < len(vm.Globals) {
				frame.regs[instr.A] = vm.Globals[slot].Value
			} else {
				frame.regs[instr.A] = value.MakeNull()
			}

		case OP_SET_GLOBAL:
			slot := instr.B
			if slot >= 0 {
				vm.ensureGlobalsSize(slot + 1)
				info := vm.Globals[slot]
				info.Value = frame.regs[instr.A]
				vm.Globals[slot] = info
			}

		case OP_DEFINE_GLOBAL:
			slot := instr.B
			if slot >= 0 {
				vm.ensureGlobalsSize(slot + 1)
				newVal := frame.regs[instr.A]

				// Special case: Don't overwrite existing non-null values with null
				// This happens when loading bytecode where stdlib functions were serialized as null
				if newVal.Kind == value.NullKind && slot < len(vm.Globals) {
					existingVal := vm.Globals[slot].Value
					if existingVal.Kind != value.NullKind {
						// Skip this definition - keep the existing value
						break
					}
				}

				vm.Globals[slot] = VariableInfo{
					Value: newVal,
					Type:  "",
				}
			}

		case OP_DEFINE_TYPED_GLOBAL:
			slot := instr.B
			typeStr := ""
			if instr.C >= 0 && instr.C < len(frame.closure.Function.Chunk.Constants) {
				typeVal := frame.closure.Function.Chunk.Constants[instr.C]
				if typeVal.Kind == value.StringKind {
					typeStr = typeVal.Obj.(string)
				}
			}
			if slot >= 0 {
				vm.ensureGlobalsSize(slot + 1)
				vm.Globals[slot] = VariableInfo{
					Value: frame.regs[instr.A],
					Type:  typeStr,
				}
			}

		case OP_ADD:
			result := add(frame.regs[instr.B], frame.regs[instr.C])
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}
		case OP_ADD_INT:
			a := frame.regs[instr.B]
			b := frame.regs[instr.C]
			if a.Kind == value.NumberKind && b.Kind == value.NumberKind && a.Num == math.Trunc(a.Num) && b.Num == math.Trunc(b.Num) {
				frame.regs[instr.A] = value.MakeNumber(a.Num + b.Num)
			} else {
				result := add(a, b)
				result = withInstrLocation(result, instr)
				frame.regs[instr.A] = result
				if result.Kind == value.ErrorKind {
					if vm.handleError(result) {
						continue
					}
					return result
				}
			}
		case OP_ADD_FLOAT:
			a := frame.regs[instr.B]
			b := frame.regs[instr.C]
			if a.Kind == value.NumberKind && b.Kind == value.NumberKind {
				frame.regs[instr.A] = value.MakeNumber(a.Num + b.Num)
			} else {
				result := add(a, b)
				result = withInstrLocation(result, instr)
				frame.regs[instr.A] = result
				if result.Kind == value.ErrorKind {
					if vm.handleError(result) {
						continue
					}
					return result
				}
			}
		case OP_ADD_STR:
			a := frame.regs[instr.B]
			b := frame.regs[instr.C]
			if value.IsStringLike(a) && value.IsStringLike(b) {
				frame.regs[instr.A] = value.ConcatStrings(a, b)
			} else {
				result := add(a, b)
				result = withInstrLocation(result, instr)
				frame.regs[instr.A] = result
				if result.Kind == value.ErrorKind {
					if vm.handleError(result) {
						continue
					}
					return result
				}
			}
		case OP_ADD_LOCAL:
			a := frame.regs[instr.A]
			b := frame.regs[instr.B]
			result := add(a, b)
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if instr.C >= 0 && instr.C < len(frame.regs) {
				frame.regs[instr.C] = result
			}
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}
		case OP_SUB_LOCAL:
			a := frame.regs[instr.A]
			b := frame.regs[instr.B]
			result := sub(a, b)
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if instr.C >= 0 && instr.C < len(frame.regs) {
				frame.regs[instr.C] = result
			}
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}
		case OP_MUL_LOCAL:
			a := frame.regs[instr.A]
			b := frame.regs[instr.B]
			result := mul(a, b)
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if instr.C >= 0 && instr.C < len(frame.regs) {
				frame.regs[instr.C] = result
			}
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}
		case OP_SUB:
			result := sub(frame.regs[instr.B], frame.regs[instr.C])
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}
		case OP_MUL:
			result := mul(frame.regs[instr.B], frame.regs[instr.C])
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}
		case OP_DIV:
			result := div(frame.regs[instr.B], frame.regs[instr.C])
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}
		case OP_EQUAL:
			frame.regs[instr.A] = equal(frame.regs[instr.B], frame.regs[instr.C])
		case OP_GREATER:
			result := greater(frame.regs[instr.B], frame.regs[instr.C])
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}
		case OP_LESS:
			result := less(frame.regs[instr.B], frame.regs[instr.C])
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}
		case OP_NOT_EQUAL:
			eqResult := equal(frame.regs[instr.B], frame.regs[instr.C])
			frame.regs[instr.A] = value.MakeBool(!eqResult.Bool)
		case OP_LESS_EQUAL:
			frame.regs[instr.A] = value.MakeBool(frame.regs[instr.B].Num <= frame.regs[instr.C].Num)
		case OP_GREATER_EQUAL:
			frame.regs[instr.A] = value.MakeBool(frame.regs[instr.B].Num >= frame.regs[instr.C].Num)
		case OP_NOT:

			dest := instr.A
			src := instr.B
			v := frame.regs[src]
			frame.regs[dest] = value.MakeBool(isFalsey(v))
		case OP_JUMP:
			// Check if there's an active handler with a finally block
			if handlerIdx, hasHandler := vm.findActiveHandlerForFrame(); hasHandler {
				// Set pending signal and jump to finally block
				handler := &vm.handlers[handlerIdx]
				handler.pendingSignal = &ControlSignal{
					Type:     SIGNAL_BREAK, // JUMP can come from break statements
					Value:    value.MakeNull(),
					ResumeIP: frame.ip + instr.A, // Where to jump after finally
				}
				frame.ip = handler.finallyIP
			} else {
				// No finally block, just jump normally
				frame.ip += instr.A
			}
		case OP_JUMP_IF_FALSE:
			if isFalsey(frame.regs[instr.A]) {
				frame.ip += instr.B
			}
		case OP_JUMP_IF_TRUE:
			if !isFalsey(frame.regs[instr.A]) {
				frame.ip += instr.B
			}
		case OP_LOOP:
			// Check if there's an active handler with a finally block
			if handlerIdx, hasHandler := vm.findActiveHandlerForFrame(); hasHandler {
				// Set pending signal and jump to finally block
				handler := &vm.handlers[handlerIdx]
				handler.pendingSignal = &ControlSignal{
					Type:     SIGNAL_CONTINUE, // LOOP can come from continue statements
					Value:    value.MakeNull(),
					ResumeIP: frame.ip - instr.A, // Where to jump after finally
				}
				frame.ip = handler.finallyIP
			} else {
				// No finally block, just loop normally
				frame.ip -= instr.A
			}

		case OP_RETURN:
			result := frame.regs[instr.A]

			// Check if there's an active handler with a finally block
			if handlerIdx, hasHandler := vm.findActiveHandlerForFrame(); hasHandler {
				// Store the return value temporarily and jump to finally
				handler := &vm.handlers[handlerIdx]
				handler.pendingSignal = &ControlSignal{
					Type:     SIGNAL_RETURN,
					Value:    result,
					ResumeIP: -1, // Special marker: return from finally, use stored value
				}
				frame.ip = handler.finallyIP
			} else {
				// No finally block, execute return normally
				frame.closeUpvalues()
				vm.freeRegs(frame.regs)
				vm.frames = vm.frames[:len(vm.frames)-1]
				if len(vm.frames) == 0 {
					return result
				}
				caller := &vm.frames[len(vm.frames)-1]
				caller.regs[frame.ReturnReg] = result
			}

		case OP_MOVE:
			frame.regs[instr.A] = frame.regs[instr.B]

		case OP_CALL:
			dest := instr.A
			funcReg := instr.B
			argStart := instr.C
			argCount := instr.D

			callee := frame.regs[funcReg]
			switch callee.Kind {
			case value.InternalFunctionKind:
				fn := callee.AsInternalFunction()
				if fn.Arity != -1 && fn.Arity != argCount {
					errVal := runtimeErrorAt(instr, "wrong number of arguments", "ArgumentError")
					frame.regs[dest] = errVal
					if vm.handleError(errVal) {
						continue
					}
					return errVal
				}
				var args []value.Value
				if argCount > 0 {
					args = frame.regs[argStart : argStart+argCount]
				}
				result := fn.Call(args)
				result = withInstrLocation(result, instr)
				frame.regs[dest] = result
				if result.Kind == value.ErrorKind {
					if vm.handleError(result) {
						continue
					}
					return result
				}

			case value.ClosureKind:
				cl := AsClosure(callee)
				if cl.Function.Arity != argCount {
					errVal := runtimeErrorAt(instr, "wrong number of arguments", "ArgumentError")
					frame.regs[dest] = errVal
					if vm.handleError(errVal) {
						continue
					}
					return errVal
				}
				newFrame := CallFrame{
					closure:   cl,
					ip:        0,
					regs:      vm.allocRegs(cl.Function.MaxRegisters),
					ReturnReg: dest,
				}
				for i := 0; i < argCount; i++ {
					newFrame.regs[i] = frame.regs[argStart+i]
				}
				vm.frames = append(vm.frames, newFrame)

			case value.FunctionKind:
				fn := AsFunction(callee)
				if fn.Arity != argCount {
					errVal := runtimeErrorAt(instr, "wrong number of arguments", "ArgumentError")
					frame.regs[dest] = errVal
					if vm.handleError(errVal) {
						continue
					}
					return errVal
				}
				cl := &ClosureObject{Function: fn, Upvalues: nil}
				newFrame := CallFrame{
					closure:   cl,
					ip:        0,
					regs:      vm.allocRegs(fn.MaxRegisters),
					ReturnReg: dest,
				}
				for i := 0; i < argCount; i++ {
					newFrame.regs[i] = frame.regs[argStart+i]
				}
				vm.frames = append(vm.frames, newFrame)

			default:
				errVal := runtimeErrorAt(instr, "not callable", "TypeError")
				frame.regs[dest] = errVal
				if vm.handleError(errVal) {
					continue
				}
				return errVal
			}

		case OP_CLOSURE:
			idx := instr.B
			fnVal := frame.closure.Function.Chunk.Constants[idx]
			fn := AsFunction(fnVal)
			closure := &ClosureObject{
				Function: fn,
				Upvalues: make([]*Upvalue, fn.UpvalueCount),
			}
			if fn.UpvalueCount > 0 {
				for i, desc := range fn.Upvalues {
					if i >= len(closure.Upvalues) {
						break
					}
					if desc.IsLocal {
						closure.Upvalues[i] = frame.captureUpvalue(desc.Index)
					} else {
						closure.Upvalues[i] = frame.closure.Upvalues[desc.Index]
					}
				}
			}
			frame.regs[instr.A] = MakeClosure(closure)

		case OP_BUILD_ARRAY:
			count := instr.B
			elements := make([]value.Value, count)
			for i := 0; i < count; i++ {
				elements[i] = frame.regs[instr.C+i]
			}
			frame.regs[instr.A] = value.MakeArray(elements)

		case OP_BUILD_MAP:
			count := instr.B
			props := make(map[string]value.Value, count)
			for i := 0; i < count; i++ {
				// key is at C+2*i, value is at C+2*i+1
				keyVal := frame.regs[instr.C+2*i]
				if !value.IsStringLike(keyVal) {
					errVal := runtimeErrorAt(instr, "map key must be a string", "TypeError")
					frame.regs[instr.A] = errVal
					if vm.handleError(errVal) {
						continue
					}
					return errVal
				}
				val := frame.regs[instr.C+2*i+1]
				props[keyVal.ToString()] = val
			}
			frame.regs[instr.A] = value.MakeMap(props)

		case OP_GET_PROPERTY:
			keyIdx := instr.C
			keyVal := frame.closure.Function.Chunk.Constants[keyIdx]
			if keyVal.Kind != value.StringKind {
				errVal := value.MakeError(
					"Attempting to access property with non-string key",
					"TypeError",
					instr.Line,
					instr.Column,
				)
				frame.regs[instr.A] = errVal
				if vm.handleError(errVal) {
					continue
				}
				return errVal
			}
			obj := frame.regs[instr.B]
			if obj.Kind != value.MapKind {
				errVal := value.MakeError(
					"Attempting to access property of non-map",
					"TypeError",
					instr.Line,
					instr.Column,
				)
				frame.regs[instr.A] = errVal
				if vm.handleError(errVal) {
					continue
				}
				return errVal
			} else {
				m := obj.AsMap()
				key := keyVal.Obj.(string)
				if val, exists := m.Properties[key]; exists {
					frame.regs[instr.A] = val
				} else {
					frame.regs[instr.A] = value.MakeNull()
				}
			}

		case OP_SET_PROPERTY:
			keyIdx := instr.C
			keyVal := frame.closure.Function.Chunk.Constants[keyIdx]
			if keyVal.Kind != value.StringKind {
				errVal := value.MakeError("Attempting to set property with non-string key", "TypeError", instr.Line, instr.Column)
				frame.regs[instr.A] = errVal
				if vm.handleError(errVal) {
					continue
				}
				return errVal
			}
			obj := frame.regs[instr.B]
			val := frame.regs[instr.A]
			if obj.Kind == value.MapKind {
				obj.AsMap().Properties[keyVal.Obj.(string)] = val
			}

		case OP_GET_INDEX:
			objReg := instr.B
			idxReg := instr.C
			obj := frame.regs[objReg]
			idx := frame.regs[idxReg]

			switch obj.Kind {
			case value.ArrayKind:
				if idx.Kind == value.NumberKind {
					i := int(idx.Num)
					arr := obj.AsArray()
					if i >= 0 && i < len(arr.Elements) {
						frame.regs[instr.A] = arr.Elements[i]
					} else {
						frame.regs[instr.A] = value.MakeNull()
					}
				}
			case value.MapKind:
				if value.IsStringLike(idx) {
					m := obj.AsMap()
					key := idx.ToString()
					if val, exists := m.Properties[key]; exists {
						frame.regs[instr.A] = val
					} else {
						frame.regs[instr.A] = value.MakeNull()
					}
				}
			}

		case OP_SET_INDEX:
			objReg := instr.B
			idxReg := instr.C
			obj := frame.regs[objReg]
			idx := frame.regs[idxReg]
			val := frame.regs[instr.A]

			switch obj.Kind {
			case value.ArrayKind:
				if idx.Kind == value.NumberKind {
					i := int(idx.Num)
					arr := obj.AsArray()
					if i < 0 {
						break
					}
					if i < len(arr.Elements) {
						break
					}
					// Auto-grow with amortized capacity expansion.
					needed := i + 1
					if needed > cap(arr.Elements) {
						newCap := cap(arr.Elements) * 2
						if newCap < 8 {
							newCap = 8
						}
						for newCap < needed {
							newCap *= 2
						}
						newElems := make([]value.Value, needed, newCap)
						copy(newElems, arr.Elements)
						arr.Elements = newElems
					} else {
						arr.Elements = arr.Elements[:needed]
					}
					arr.Elements[i] = val
				}
			case value.MapKind:
				if value.IsStringLike(idx) {
					obj.AsMap().Properties[idx.ToString()] = val
				}
			}

		case OP_LOAD_MODULE:
			dest := instr.A
			nameConst := frame.closure.Function.Chunk.Constants[instr.B]
			if nameConst.Kind != value.StringKind {
				errVal := runtimeErrorAt(instr, "module name must be a string", "TypeError")
				frame.regs[dest] = errVal
				if vm.handleError(errVal) {
					continue
				}
				return errVal
			}
			moduleName := nameConst.Obj.(string)

			module, ok := vm.modules[moduleName]
			if !ok {
				errVal := runtimeErrorAt(instr, "Module not found: "+moduleName, "ModuleError")
				frame.regs[dest] = errVal
				if vm.handleError(errVal) {
					continue
				}
				return errVal
			}
			frame.regs[dest] = module

		case OP_POP:
			// Do nothing - register-based VM doesn't have explicit stack

		case OP_TRY_BEGIN:
			// Register the exception handler for this try block
			// A = catch handler address, B = error register
			// C = finally block address (will be set by compiler in step 2)
			vm.handlers = append(vm.handlers, ExceptionHandler{
				frameIndex:    len(vm.frames) - 1,
				handlerIP:     instr.A,
				finallyIP:     instr.C, // Finally block address from compiler
				errorReg:      instr.B,
				pendingSignal: nil, // No pending control transfer yet
			})

		case OP_TRY_END:
			// Check if there's a pending control signal from break/continue/return
			if len(vm.handlers) > 0 {
				handler := &vm.handlers[len(vm.handlers)-1]

				// Only apply pending signal if this handler is for the current frame
				if handler.frameIndex == len(vm.frames)-1 && handler.pendingSignal != nil {
					signal := handler.pendingSignal
					handler.pendingSignal = nil // Clear the signal

					switch signal.Type {
					case SIGNAL_BREAK:
						// Jump to the resume point (after the loop)
						vm.handlers = vm.handlers[:len(vm.handlers)-1]
						frame.ip = signal.ResumeIP
						continue // Skip normal processing

					case SIGNAL_CONTINUE:
						// Jump to the loop start
						vm.handlers = vm.handlers[:len(vm.handlers)-1]
						frame.ip = signal.ResumeIP
						continue // Skip normal processing

					case SIGNAL_RETURN:
						// Return from function with stored value
						result := signal.Value
						vm.handlers = vm.handlers[:len(vm.handlers)-1]
						frame.closeUpvalues()
						vm.freeRegs(frame.regs)
						vm.frames = vm.frames[:len(vm.frames)-1]
						if len(vm.frames) == 0 {
							return result
						}
						caller := &vm.frames[len(vm.frames)-1]
						caller.regs[frame.ReturnReg] = result
						continue // Skip normal processing
					}
				}

				// No pending signal or not for this frame - normal cleanup
				vm.handlers = vm.handlers[:len(vm.handlers)-1]
			}

		case OP_GET_UPVALUE:
			dest := instr.A
			idx := instr.B
			up := frame.closure.Upvalues[idx]
			frame.regs[dest] = *up.location
		case OP_SET_UPVALUE:
			src := instr.A
			idx := instr.B
			up := frame.closure.Upvalues[idx]
			*up.location = frame.regs[src]

		case OP_MOD:
			result := mod(frame.regs[instr.B], frame.regs[instr.C])
			result = withInstrLocation(result, instr)
			frame.regs[instr.A] = result
			if result.Kind == value.ErrorKind {
				if vm.handleError(result) {
					continue
				}
				return result
			}

		case OP_CALL_METHOD:
			// instr.A = dest register
			// instr.B = object register
			// instr.C = method name constant index
			// instr.D = (argStart << 16) | argCount
			objVal := frame.regs[instr.B]
			methodNameVal := frame.closure.Function.Chunk.Constants[instr.C]

			if methodNameVal.Kind != value.StringKind {
				errVal := runtimeErrorAt(instr, "method name must be a string", "TypeError")
				frame.regs[instr.A] = errVal
				if vm.handleError(errVal) {
					continue
				}
				return errVal
			}

			methodName := methodNameVal.Obj.(string)

			// Lookup method by type
			methodMap, exists := vm.methodRegistry[objVal.Kind]
			if !exists {
				errVal := runtimeErrorAt(instr, fmt.Sprintf("Type %v has no methods", objVal.Kind), "TypeError")
				frame.regs[instr.A] = errVal
				if vm.handleError(errVal) {
					continue
				}
				return errVal
			}

			method, exists := methodMap[methodName]
			if !exists {
				errVal := runtimeErrorAt(instr, fmt.Sprintf("Method '%s' not found on type %v", methodName, objVal.Kind), "AttributeError")
				frame.regs[instr.A] = errVal
				if vm.handleError(errVal) {
					continue
				}
				return errVal
			}

			// Decode argStart and argCount from D operand
			argStart := instr.D >> 16    // upper 16 bits
			argCount := instr.D & 0xFFFF // lower 16 bits

			// Build full arg list: obj + args
			fullArgs := make([]value.Value, argCount+1)
			fullArgs[0] = objVal
			for i := 0; i < argCount; i++ {
				fullArgs[i+1] = frame.regs[argStart+i]
			}

			// Call method
			result := method.Call(fullArgs)
			result = withInstrLocation(result, instr)

			// Check for error
			if result.Kind == value.ErrorKind {
				if !vm.handleError(result) {
					return result
				}
				continue
			}

			frame.regs[instr.A] = result
		case OP_NOP:
			// Do nothing
		}
	}

	return value.MakeNull()
}

func add(a, b value.Value) value.Value {
	if value.IsStringLike(a) || value.IsStringLike(b) {
		return value.ConcatStrings(a, b)
	}
	switch a.Kind {
	case value.NumberKind:
		if b.Kind == value.NumberKind {
			return value.MakeNumber(a.Num + b.Num)
		}
	}
	return value.MakeError("Invalid operands for +", "TypeError", 0, 0)
}

func sub(a, b value.Value) value.Value {
	if a.Kind == value.NumberKind && b.Kind == value.NumberKind {
		return value.MakeNumber(a.Num - b.Num)
	}
	return value.MakeError("Invalid operands for -", "TypeError", 0, 0)
}

func mul(a, b value.Value) value.Value {
	switch a.Kind {
	case value.NumberKind:
		if b.Kind == value.NumberKind {
			return value.MakeNumber(a.Num * b.Num)
		}
	case value.StringKind, value.RopeStringKind:
		if b.Kind == value.NumberKind {
			return value.MakeString(repeatString(a.ToString(), int(b.Num)))
		}
	}
	return value.MakeError("Invalid operands for *", "TypeError", 0, 0)
}

func div(a, b value.Value) value.Value {
	if a.Kind == value.NumberKind && b.Kind == value.NumberKind {
		if b.Num == 0 {
			return value.MakeError("Division by zero", "ZeroDivisionError", 0, 0)
		}
		return value.MakeNumber(a.Num / b.Num)
	}
	return value.MakeError("Invalid operands for /", "TypeError", 0, 0)
}

func mod(a, b value.Value) value.Value {
	if a.Kind == value.NumberKind && b.Kind == value.NumberKind {
		if b.Num == 0 {
			return value.MakeError("Modulo by zero", "ZeroDivisionError", 0, 0)
		}
		return value.MakeNumber(math.Mod(a.Num, b.Num))
	}
	return value.MakeError("Invalid operands for %", "TypeError", 0, 0)
}

func isFalsey(val value.Value) bool {
	switch val.Kind {
	case value.NullKind:
		return true
	case value.BoolKind:
		return !val.Bool
	case value.NumberKind:
		return val.Num == 0
	}
	return false
}

func greater(a, b value.Value) value.Value {
	switch a.Kind {
	case value.NumberKind:
		if b.Kind == value.NumberKind {
			return value.MakeBool(a.Num > b.Num)
		}
	case value.StringKind, value.RopeStringKind:
		if value.IsStringLike(b) {
			return value.MakeBool(a.ToString() > b.ToString())
		}
	}
	return value.MakeError("Invalid operands for >", "TypeError", 0, 0)
}

func less(a, b value.Value) value.Value {
	switch a.Kind {
	case value.NumberKind:
		if b.Kind == value.NumberKind {
			return value.MakeBool(a.Num < b.Num)
		}
	case value.StringKind, value.RopeStringKind:
		if value.IsStringLike(b) {
			return value.MakeBool(a.ToString() < b.ToString())
		}
	}
	return value.MakeError("Invalid operands for <", "TypeError", 0, 0)
}

func equal(a, b value.Value) value.Value {
	if value.IsStringLike(a) && value.IsStringLike(b) {
		return value.MakeBool(a.ToString() == b.ToString())
	}
	if a.Kind != b.Kind {
		return value.MakeBool(false)
	}
	switch a.Kind {
	case value.NumberKind:
		return value.MakeBool(a.Num == b.Num)
	case value.BoolKind:
		return value.MakeBool(a.Bool == b.Bool)
	case value.NullKind:
		return value.MakeBool(true)
	default:
		return value.MakeBool(a.Obj == b.Obj)
	}
}

func repeatString(s string, count int) string {
	if count <= 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s) * count)
	for i := 0; i < count; i++ {
		b.WriteString(s)
	}
	return b.String()
}

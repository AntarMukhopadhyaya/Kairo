package vm

import (
	"Kairo/frontend"
	"Kairo/stdlib"
	"Kairo/value"
	"fmt"
	"strings"
)

type Compiler struct {
	chunk       *Chunk
	parent      *Compiler
	scopes      []Scope
	upvalues    []UpvalueDescriptor
	globalSlots map[string]int
	globalTypes map[string]string
	loopStack   []LoopContext
	nextReg     int
	MaxRegUsed  int
	optimizer   *Optimizer
	currentLine int
	currentCol  int
}
type LoopContext struct {
	breakJumps    []int
	continueJumps []int
	loopStart     int
}
type Scope struct {
	locals   map[string]int
	types    map[string]string
	regStart int
}

func NewCompiler() *Compiler {
	c := &Compiler{
		chunk:       NewChunk(),
		scopes:      []Scope{},
		upvalues:    []UpvalueDescriptor{},
		globalSlots: make(map[string]int),
		globalTypes: make(map[string]string),
		loopStack:   []LoopContext{},
		nextReg:     0,
		MaxRegUsed:  0,
	}
	c.optimizer = NewOptimizer(c)
	return c

}

func (c *Compiler) addUpvalue(index int, isLocal bool) int {
	for i, up := range c.upvalues {
		if up.Index == index && up.IsLocal == isLocal {
			return i
		}
	}
	c.upvalues = append(c.upvalues, UpvalueDescriptor{Index: index, IsLocal: isLocal})
	return len(c.upvalues) - 1
}

func (c *Compiler) resolveUpvalue(name string) (int, bool) {
	if c.parent == nil {
		return -1, false
	}
	if slot, ok := c.parent.lookupLocal(name); ok {
		return c.addUpvalue(slot, true), true
	}
	if up, ok := c.parent.resolveUpvalue(name); ok {
		return c.addUpvalue(up, false), true
	}
	return -1, false
}

func (c *Compiler) newRegister() int {
	r := c.nextReg
	c.nextReg++
	if c.nextReg > c.MaxRegUsed {
		c.MaxRegUsed = c.nextReg
	}
	return r
}

func (c *Compiler) freeRegister(reg int) {
	if reg == c.nextReg-1 {
		c.nextReg--
	}
}

func (c *Compiler) globalSlotsMap() map[string]int {
	if c.parent != nil {
		return c.parent.globalSlotsMap()
	}
	return c.globalSlots
}

func (c *Compiler) GlobalSlots() map[string]int {
	return c.globalSlotsMap()
}

func (c *Compiler) globalTypesMap() map[string]string {
	if c.parent != nil {
		return c.parent.globalTypesMap()
	}
	return c.globalTypes
}

func (c *Compiler) SetGlobalSlots(slots map[string]int) {
	c.globalSlots = slots
}

func (c *Compiler) getGlobalSlot(name string) int {
	slots := c.globalSlotsMap()
	if slot, ok := slots[name]; ok {
		return slot
	}
	slot := len(slots)
	slots[name] = slot
	return slot
}

func (c *Compiler) enterScope() {
	c.scopes = append(c.scopes, Scope{
		locals:   make(map[string]int),
		types:    make(map[string]string),
		regStart: c.nextReg,
	})
}
func (c *Compiler) exitScope() {
	scope := c.scopes[len(c.scopes)-1]
	c.nextReg = scope.regStart
	c.scopes = c.scopes[:len(c.scopes)-1]
}

func (c *Compiler) lookupLocal(name string) (int, bool) {
	// Search scope stack from top → bottom
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if slot, ok := c.scopes[i].locals[name]; ok {
			return slot, true
		}
	}
	return -1, false
}

func (c *Compiler) Compile(program frontend.Program) *Chunk {
	if c.optimizer != nil {
		c.optimizer.ResetStats()
	}
	for _, stmt := range program.Body {
		c.compileStatement(stmt)
	}
	resultReg := c.newRegister()
	c.emit(OP_LOAD_CONST, resultReg, c.chunk.AddConstant(value.MakeNull()), 0)
	c.emit(OP_RETURN, resultReg, 0, 0)
	if c.optimizer != nil {
		c.optimizer.Optimize()
	}
	return c.chunk
}

func (c *Compiler) CompileWithDiagnostics(program frontend.Program) (chunk *Chunk, diagnostics []frontend.Diagnostic) {
	diagnostics = []frontend.Diagnostic{}

	defer func() {
		if r := recover(); r != nil {
			diagnostics = append(diagnostics, frontend.Diagnostic{
				Message: fmt.Sprint(r),
				Phase:   "compile",
			})
			chunk = nil
		}
	}()

	chunk = c.Compile(program)
	return chunk, diagnostics
}

func (c *Compiler) EnableOptimizations(enabled bool) {
	if c.optimizer == nil {
		return
	}
	// keep it explicit; add more as you implement them
	c.optimizer.EnableOptimization("constant_folding", enabled)
	c.optimizer.EnableOptimization("dead_code", enabled)
	c.optimizer.EnableOptimization("peephole", enabled)
}

func (c *Compiler) OptimizationStats() (OptimizationStats, bool) {
	if c.optimizer == nil {
		return OptimizationStats{}, false
	}
	return c.optimizer.GetStats(), true
}

func (c *Compiler) compileStatement(stmt frontend.Statement) {
	prevLine, prevCol := c.currentLine, c.currentCol
	line, col := statementLocation(stmt)
	if line > 0 {
		c.currentLine = line
		c.currentCol = col
	}
	defer func() {
		c.currentLine = prevLine
		c.currentCol = prevCol
	}()

	switch s := stmt.(type) {
	case frontend.VariableDeclaration:
		valueReg := c.compileExpression(s.Value)
		inferredType := c.inferExprType(s.Value)
		declType := normalizeType(s.TypeAnnotation)
		if declType == "" {
			declType = inferredType
		}

		if len(c.scopes) > 0 {
			// Inside a scope (function, loop, if, etc) - store in local
			slot := c.newRegister()
			current := &c.scopes[len(c.scopes)-1]
			current.locals[s.Identifier] = slot
			if declType != "" {
				current.types[s.Identifier] = declType
			}
			c.emit(OP_MOVE, slot, valueReg, 0)
			c.freeRegister(valueReg)
		} else {
			// Global scope
			slot := c.getGlobalSlot(s.Identifier)
			if declType != "" {
				c.globalTypesMap()[s.Identifier] = declType
			}
			if s.TypeAnnotation != "" {
				typeIdx := c.chunk.AddConstant(value.MakeString(s.TypeAnnotation))
				c.emit(OP_DEFINE_TYPED_GLOBAL, valueReg, slot, typeIdx)
			} else {
				c.emit(OP_DEFINE_GLOBAL, valueReg, slot, 0)
			}
			c.freeRegister(valueReg)
		}

	case frontend.IfStatement:
		// Dead-code elimination for constant conditions.
		if c.optimizer != nil && c.optimizer.Enabled("dead_code") {
			if isConst, condValue := c.optimizer.isConstantCondition(s.Condition); isConst {
				if condValue {
					c.optimizer.NoteDeadCodeRemoved()
					c.enterScope()
					for _, stmt := range s.Consequent {
						c.compileStatement(stmt)
					}
					c.exitScope()
					return
				}
				// condition false
				if len(s.Alternate) == 0 {
					c.optimizer.NoteDeadCodeRemoved()
					return
				}
				c.optimizer.NoteDeadCodeRemoved()
				c.enterScope()
				for _, stmt := range s.Alternate {
					c.compileStatement(stmt)
				}
				c.exitScope()
				return
			}
		}

		condReg := c.compileExpression(s.Condition)
		jumpIfFalse := c.emitJumpIfFalse(condReg)
		c.freeRegister(condReg)
		c.enterScope()
		for _, stmt := range s.Consequent {
			c.compileStatement(stmt)
		}
		c.exitScope()
		if len(s.Alternate) > 0 {
			jumpOverElse := c.emitJump()
			c.patchJumpIfFalse(jumpIfFalse)
			c.enterScope()
			for _, stmt := range s.Alternate {
				c.compileStatement(stmt)
			}
			c.exitScope()
			c.patchJump(jumpOverElse)
		} else {
			c.patchJumpIfFalse(jumpIfFalse)
		}

	case frontend.WhileStatement:
		loopStart := len(c.chunk.Code)
		c.loopStack = append(c.loopStack, LoopContext{
			breakJumps:    []int{},
			continueJumps: []int{},
			loopStart:     loopStart,
		})
		condReg := c.compileExpression(s.Condition)
		exitJump := c.emitJumpIfFalse(condReg)
		c.freeRegister(condReg)
		c.enterScope()
		for _, stmt := range s.Body {
			c.compileStatement(stmt)
		}
		c.exitScope()
		offset := loopStart - len(c.chunk.Code) - 1
		c.emit(OP_JUMP, offset, 0, 0)
		c.patchJumpIfFalse(exitJump)
		// pop loop context

		loop := c.popLoop()
		// Patch break jumps to jump here (after loop)
		loopEnd := len(c.chunk.Code)
		for _, idx := range loop.breakJumps {
			offset := loopEnd - idx - 1
			c.chunk.Code[idx].A = offset
		}
		// Patch continue jumps to jump to loop start
		for _, idx := range loop.continueJumps {
			offset := loop.loopStart - idx - 1
			c.chunk.Code[idx].A = offset
		}

	case frontend.ImportStatement:
		// Resolve module at compile time
		module, ok := stdlib.BuiltinModules[s.Source]
		if !ok {
			panic("unknown module: " + s.Source)
		}

		// For each import specifier, add to globals
		for _, spec := range s.Specifiers {
			exportValue, ok := module.Exports[spec.Imported]
			if !ok {
				panic("module " + s.Source + " has no export: " + spec.Imported)
			}

			// Assign to local name as global
			slot := c.getGlobalSlot(spec.Local)
			constIdx := c.chunk.AddConstant(exportValue)
			tempReg := c.newRegister()
			c.emit(OP_LOAD_CONST, tempReg, constIdx, 0)
			c.emit(OP_DEFINE_GLOBAL, tempReg, slot, 0)
			c.freeRegister(tempReg)
		}

	case frontend.ForStatement:
		// For-loop gets its own scope
		c.enterScope()
		if s.Initial != nil {
			c.compileStatement(s.Initial)
		}
		conditionStart := len(c.chunk.Code)
		c.loopStack = append(c.loopStack, LoopContext{
			breakJumps:    []int{},
			continueJumps: []int{},
			loopStart:     conditionStart,
		})
		var exitJump int
		if s.Condition != nil {
			condReg := c.compileExpression(s.Condition)
			exitJump = c.emitJumpIfFalse(condReg)
			c.freeRegister(condReg)
		}
		for _, stmt := range s.Body {
			c.compileStatement(stmt)
		}
		iterationStart := len(c.chunk.Code)
		if s.Iteration != nil {
			c.compileStatement(s.Iteration)
		}
		offset := conditionStart - len(c.chunk.Code) - 1
		c.emit(OP_JUMP, offset, 0, 0)
		if s.Condition != nil {
			c.patchJumpIfFalse(exitJump)
		}
		loop := c.popLoop()
		loopEnd := len(c.chunk.Code)
		for _, idx := range loop.breakJumps {
			offset := loopEnd - idx - 1
			c.chunk.Code[idx].A = offset
		}
		for _, idx := range loop.continueJumps {
			offset := iterationStart - idx - 1
			c.chunk.Code[idx].A = offset
		}
		c.exitScope()
		// loopStart := len(c.chunk.Code)
		// var exitJump int
		// if s.Condition != nil {
		// 	condReg := c.compileExpression(s.Condition)
		// 	exitJump = c.emitJumpIfFalse(condReg)
		// 	c.freeRegister(condReg)
		// }
		// for _, stmt := range s.Body {
		// 	c.compileStatement(stmt)
		// }
		// if s.Iteration != nil {
		// 	c.compileStatement(s.Iteration)
		// }
		// offset := loopStart - len(c.chunk.Code) - 1
		// c.emit(OP_JUMP, offset, 0, 0)
		// if s.Condition != nil {
		// 	c.patchJumpIfFalse(exitJump)
		// }
		// c.exitScope()

	case frontend.SwitchStatement:
		switchReg := c.compileExpression(s.Expr)
		endJumps := make([]int, 0, len(s.Cases))

		for _, caseClause := range s.Cases {
			testReg := c.compileExpression(caseClause.Test)
			matchReg := c.newRegister()
			c.emit(OP_EQUAL, matchReg, switchReg, testReg)
			c.freeRegister(testReg)

			jumpIfNotMatch := c.emitJumpIfFalse(matchReg)
			c.freeRegister(matchReg)

			c.enterScope()
			for _, stmt := range caseClause.Consequent {
				c.compileStatement(stmt)
			}
			c.exitScope()

			endJumps = append(endJumps, c.emitJump())
			c.patchJumpIfFalse(jumpIfNotMatch)
		}

		if len(s.Default) > 0 {
			c.enterScope()
			for _, stmt := range s.Default {
				c.compileStatement(stmt)
			}
			c.exitScope()
		}

		for _, jump := range endJumps {
			c.patchJump(jump)
		}

		c.freeRegister(switchReg)

	case frontend.FunctionDeclaration:
		funcChunk := NewChunk()
		childCompiler := &Compiler{
			chunk:       funcChunk,
			scopes:      []Scope{},
			parent:      c,
			upvalues:    []UpvalueDescriptor{},
			globalSlots: c.globalSlotsMap(),
			globalTypes: c.globalTypesMap(),
			nextReg:     0,
		}

		// Function body gets its own scope
		childCompiler.enterScope()

		// Add parameters to the first scope's locals
		for i, param := range s.Parameters {
			childCompiler.scopes[0].locals[param.Name] = i
			childCompiler.scopes[0].types[param.Name] = normalizeType(param.Type)
			childCompiler.nextReg = i + 1
		}
		if childCompiler.nextReg > childCompiler.MaxRegUsed {
			childCompiler.MaxRegUsed = childCompiler.nextReg
		}

		// Compile body
		for _, stmt := range s.Body {
			childCompiler.compileStatement(stmt)
		}

		// Add return at end
		returnReg := childCompiler.newRegister()
		funcChunk.Emit(OP_LOAD_CONST, returnReg, funcChunk.AddConstant(value.MakeNull()), 0)
		funcChunk.Emit(OP_RETURN, returnReg, 0, 0)

		childCompiler.exitScope()

		// Create function value
		fnValue := &FunctionObject{
			Chunk:        funcChunk,
			Arity:        len(s.Parameters),
			Name:         s.Name,
			UpvalueCount: len(childCompiler.upvalues),
			Upvalues:     childCompiler.upvalues,
			MaxRegisters: childCompiler.MaxRegUsed,
			ParamTypes:   getParameterTypes(s.Parameters),
			ReturnType:   s.ReturnType,
		}

		idx := c.chunk.AddConstant(MakeFunction(fnValue))
		if c.parent != nil {
			// Nested function - store as local in current scope
			slot := c.newRegister()
			if len(c.scopes) > 0 {
				current := &c.scopes[len(c.scopes)-1]
				current.locals[s.Name] = slot
			}
			c.emit(OP_CLOSURE, slot, idx, 0)
		} else {
			// Global function
			slot := c.getGlobalSlot(s.Name)
			tempReg := c.newRegister()
			c.emit(OP_CLOSURE, tempReg, idx, 0)
			c.emit(OP_DEFINE_GLOBAL, tempReg, slot, 0)
			c.freeRegister(tempReg)
		}

	case frontend.ReturnStatement:
		if s.Value != nil {
			resultReg := c.compileExpression(s.Value)
			c.emit(OP_RETURN, resultReg, 0, 0)
		} else {
			nullReg := c.newRegister()
			c.emit(OP_LOAD_CONST, nullReg, c.chunk.AddConstant(value.MakeNull()), 0)
			c.emit(OP_RETURN, nullReg, 0, 0)
		}
	case frontend.TryCatchStatement:
		// Allocate a register for the error (will be used in catch blocks)
		errorReg := c.newRegister()

		// Jump table: tryBeginIndex will store the catch start address
		tryBeginIndex := len(c.chunk.Code)
		c.emit(OP_TRY_BEGIN, -1, errorReg, -1) // C=-1 for now, will patch with finally address

		// Compile try block
		c.enterScope()
		for _, tryStmt := range s.TryBlock {
			c.compileStatement(tryStmt)
		}
		c.exitScope()

		// Emit TRY_END to pop handler on successful completion
		c.emit(OP_TRY_END, 0, 0, 0)

		// End of try block, jump over all catch blocks
		tryEndJump := c.emitJump()
		catchStartIndex := len(c.chunk.Code)
		c.chunk.Code[tryBeginIndex].A = catchStartIndex

		// Compile all catch blocks
		for _, catchBlock := range s.CatchBlock {
			c.enterScope()

			// Map the error variable name to the pre-allocated error register
			current := &c.scopes[len(c.scopes)-1]
			current.locals[catchBlock.VarName] = errorReg

			// Compile catch body
			for _, catchStmt := range catchBlock.Body {
				c.compileStatement(catchStmt)
			}

			c.exitScope()
		}

		// Patch the jump that skips catches
		c.patchJump(tryEndJump)

		// Record where the finally block will start (if it exists)
		var finallyStartIndex int
		if len(s.FinallyBlock) > 0 {
			finallyStartIndex = len(c.chunk.Code)
			// Patch OP_TRY_BEGIN.C with the finally block address
			c.chunk.Code[tryBeginIndex].C = finallyStartIndex
		}
		// else: finallyIP stays -1 (no finally block)

		// Free the error register
		c.freeRegister(errorReg)

		// Compile finally block if present
		if len(s.FinallyBlock) > 0 {
			c.enterScope()
			for _, finallyStmt := range s.FinallyBlock {
				c.compileStatement(finallyStmt)
			}
			c.exitScope()

			// Emit TRY_END to pop handler after finally block completes
			// This handles pending signals that were set before jumping to finally
			c.emit(OP_TRY_END, 0, 0, 0)
		}
	case frontend.BreakStatement:
		c.emitBreakJump()
	case frontend.ContinueStatement:
		c.emitContinueJump()
	case frontend.Expression:
		exprReg := c.compileExpression(s)
		c.freeRegister(exprReg)
	}
}

func (c *Compiler) compileExpression(expr frontend.Expression) int {
	prevLine, prevCol := c.currentLine, c.currentCol
	line, col := expressionLocation(expr)
	if line > 0 {
		c.currentLine = line
		c.currentCol = col
	}
	defer func() {
		c.currentLine = prevLine
		c.currentCol = prevCol
	}()

	// Constant folding (compile-time evaluation)
	if c.optimizer != nil && c.optimizer.Enabled("constant_folding") {
		if constVal, ok := c.optimizer.tryFoldConstant(expr); ok {
			c.optimizer.NoteConstantFolded()
			reg := c.newRegister()
			constIndex := c.chunk.AddConstant(constVal)
			c.emit(OP_LOAD_CONST, reg, constIndex, 0)
			return reg
		}
	}

	switch e := expr.(type) {
	case frontend.NumericLiteral:
		reg := c.newRegister()
		constIndex := c.chunk.AddConstant(value.MakeNumber(e.Value))
		c.emit(OP_LOAD_CONST, reg, constIndex, 0)
		return reg

	case frontend.FloatLiteral:
		reg := c.newRegister()
		constIndex := c.chunk.AddConstant(value.MakeNumber(e.Value))
		c.emit(OP_LOAD_CONST, reg, constIndex, 0)
		return reg
	case frontend.BooleanLiteral:
		reg := c.newRegister()
		constIndex := c.chunk.AddConstant(value.MakeBool(e.Value))
		c.emit(OP_LOAD_CONST, reg, constIndex, 0)
		return reg

	case frontend.StringLiteral:
		reg := c.newRegister()
		constIndex := c.chunk.AddConstant(value.MakeString(e.Value))
		c.emit(OP_LOAD_CONST, reg, constIndex, 0)
		return reg

	case frontend.Identifier:
		reg := c.newRegister()
		if slot, ok := c.lookupLocal(e.Symbol); ok {
			// Local variable - search scope stack
			c.emit(OP_MOVE, reg, slot, 0)
		} else if up, ok := c.resolveUpvalue(e.Symbol); ok {
			c.emit(OP_GET_UPVALUE, reg, up, 0)
		} else {
			// Global variable
			slot := c.getGlobalSlot(e.Symbol)
			c.emit(OP_GET_GLOBAL, reg, slot, 0)
		}
		return reg

	case frontend.BinaryExpression:
		// Handle logical operators with lazy evaluation (short-circuiting)
		if e.Operator == "&&" {
			left := c.compileExpression(e.Left)
			result := c.newRegister()
			// If left is false, jump to end and return left (false)
			jumpIfFalse := c.emitJumpIfFalse(left)
			// Left is true, evaluate right
			right := c.compileExpression(e.Right)
			c.emit(OP_MOVE, result, right, 0)
			jumpOver := c.emitJump()
			// Left was false, return left (false)
			c.patchJumpIfFalse(jumpIfFalse)
			c.emit(OP_MOVE, result, left, 0)
			c.patchJump(jumpOver)
			c.freeRegister(right)
			c.freeRegister(left)
			return result
		}

		if e.Operator == "||" {
			left := c.compileExpression(e.Left)
			result := c.newRegister()
			// If left is true, jump to end and return left (true)
			jumpIfTrue := c.emitJumpIfTrue(left)
			// Left is false, evaluate right
			right := c.compileExpression(e.Right)
			c.emit(OP_MOVE, result, right, 0)
			jumpOver := c.emitJump()
			// Left was true, return left (true)
			c.patchJumpIfTrue(jumpIfTrue)
			c.emit(OP_MOVE, result, left, 0)
			c.patchJump(jumpOver)
			c.freeRegister(right)
			c.freeRegister(left)
			return result
		}

		// For non-logical operators, evaluate both operands first
		left := c.compileExpression(e.Left)
		right := c.compileExpression(e.Right)
		result := c.newRegister()

		switch e.Operator {
		case "+":
			c.emit(c.selectAddOp(e.Left, e.Right), result, left, right)
		case "-":
			c.emit(OP_SUB, result, left, right)
		case "*":
			c.emit(OP_MUL, result, left, right)
		case "/":
			c.emit(OP_DIV, result, left, right)
		case "%":
			c.emit(OP_MOD, result, left, right)
		case ">":
			c.emit(OP_GREATER, result, left, right)
		case "<":
			c.emit(OP_LESS, result, left, right)
		case "==":
			c.emit(OP_EQUAL, result, left, right)
		case "!=":
			c.emit(OP_NOT_EQUAL, result, left, right)
		case ">=":
			c.emit(OP_GREATER_EQUAL, result, left, right)
		case "<=":
			c.emit(OP_LESS_EQUAL, result, left, right)
		default:
			panic("unsupported binary operator: " + e.Operator)
		}
		c.freeRegister(right)
		c.freeRegister(left)
		return result
	case frontend.UnaryExpression:
		operand := c.compileExpression(e.Operand)
		result := c.newRegister()
		switch e.Operator {
		case "!":
			c.emit(OP_NOT, result, operand, 0)
		}
		c.freeRegister(operand)
		return result
	case frontend.ArrayLiteral:
		// Allocate contiguous registers for elements
		count := len(e.Elements)
		if count == 0 {
			reg := c.newRegister()
			c.emit(OP_BUILD_ARRAY, reg, 0, 0)
			return reg
		}

		// Allocate all element registers contiguously
		elementRegs := make([]int, count)
		for i := range elementRegs {
			elementRegs[i] = c.newRegister()
		}

		// Compile expressions into allocated registers
		for i, elem := range e.Elements {
			tempReg := c.compileExpression(elem)
			c.emit(OP_MOVE, elementRegs[i], tempReg, 0)
			c.freeRegister(tempReg)
		}

		reg := c.newRegister()
		c.emit(OP_BUILD_ARRAY, reg, count, elementRegs[0])

		// Free element registers
		for i := len(elementRegs) - 1; i >= 0; i-- {
			c.freeRegister(elementRegs[i])
		}
		return reg

	case frontend.MapLiteral:
		count := len(e.Properties)
		if count == 0 {
			reg := c.newRegister()
			c.emit(OP_BUILD_MAP, reg, 0, 0)
			return reg
		}

		// Allocate contiguous registers for key-value pairs
		// Layout: [key0, val0, key1, val1, ...]
		pairRegs := make([]int, count*2)
		for i := range pairRegs {
			pairRegs[i] = c.newRegister()
		}

		// Compile key-value pairs into allocated registers
		for i, prop := range e.Properties {
			keyIdx := c.chunk.AddConstant(value.MakeString(prop.Key))
			keyReg := pairRegs[i*2]
			valReg := pairRegs[i*2+1]

			c.emit(OP_LOAD_CONST, keyReg, keyIdx, 0)
			tempValReg := c.compileExpression(prop.Value)
			c.emit(OP_MOVE, valReg, tempValReg, 0)
			c.freeRegister(tempValReg)
		}

		reg := c.newRegister()
		c.emit(OP_BUILD_MAP, reg, count, pairRegs[0])

		// Free pair registers
		for i := len(pairRegs) - 1; i >= 0; i-- {
			c.freeRegister(pairRegs[i])
		}
		return reg

	case frontend.MemberExpression:
		if e.Computed {
			objReg := c.compileExpression(e.Object)
			propReg := c.compileExpression(e.Property)
			result := c.newRegister()
			c.emit(OP_GET_INDEX, result, objReg, propReg)
			c.freeRegister(propReg)
			c.freeRegister(objReg)
			return result
		} else {
			objReg := c.compileExpression(e.Object)
			if ident, ok := e.Property.(frontend.Identifier); ok {
				keyIdx := c.chunk.AddConstant(value.MakeString(ident.Symbol))
				result := c.newRegister()
				c.emit(OP_GET_PROPERTY, result, objReg, keyIdx)
				c.freeRegister(objReg)
				return result
			} else {
				panic("expected identifier in non-computed member expression")
			}
		}

	case frontend.CallExpression:
		// Check if this is a method call: callee is MemberExpression
		if memberExpr, isMember := e.Callee.(frontend.MemberExpression); isMember && !memberExpr.Computed {
			// This is a method call: obj.method(args)

			objReg := c.compileExpression(memberExpr.Object)

			// Get method name
			var methodName string
			if ident, ok := memberExpr.Property.(frontend.Identifier); ok {
				methodName = ident.Symbol
			} else {
				// Fallback to regular call if not simple identifier
				goto regularCall
			}

			// Compile arguments
			argCount := len(e.Arguments)
			var argStart int
			if argCount > 0 {
				argStart = c.newRegister()
				for i := 1; i < argCount; i++ {
					c.newRegister()
				}
			}

			for i, arg := range e.Arguments {
				tempReg := c.compileExpression(arg)
				c.emit(OP_MOVE, argStart+i, tempReg, 0)
				c.freeRegister(tempReg)
			}

			// Emit OP_CALL_METHOD
			dest := c.newRegister()
			methodNameIdx := c.chunk.AddConstant(value.MakeString(methodName))

			// Encode argStart and argCount into D operand
			argStartAndCount := (argStart << 16) | argCount

			c.emit(OP_CALL_METHOD, dest, objReg, methodNameIdx, argStartAndCount)

			// Free registers
			for i := argCount - 1; i >= 0; i-- {
				c.freeRegister(argStart + i)
			}
			c.freeRegister(objReg)

			return dest
		}

	regularCall:
		funcReg := c.compileExpression(e.Callee)

		// Allocate contiguous registers for arguments
		argCount := len(e.Arguments)
		var argStart int
		if argCount > 0 {
			argStart = c.newRegister()
			for i := 1; i < argCount; i++ {
				c.newRegister()
			}
		}

		// Compile arguments into allocated registers
		for i, arg := range e.Arguments {
			tempReg := c.compileExpression(arg)
			c.emit(OP_MOVE, argStart+i, tempReg, 0)
			c.freeRegister(tempReg)
		}

		dest := c.newRegister()
		c.emit(OP_CALL, dest, funcReg, argStart, argCount)

		// Free argument registers
		for i := argCount - 1; i >= 0; i-- {
			c.freeRegister(argStart + i)
		}
		c.freeRegister(funcReg)
		return dest

	case frontend.FunctionExpression:
		funcChunk := NewChunk()
		childCompiler := &Compiler{
			chunk:       funcChunk,
			scopes:      []Scope{},
			parent:      c,
			upvalues:    []UpvalueDescriptor{},
			globalSlots: c.globalSlotsMap(),
			globalTypes: c.globalTypesMap(),
			nextReg:     0,
		}

		// Function body gets its own scope
		childCompiler.enterScope()

		// Add parameters to the first scope's locals
		for i, param := range e.Parameters {
			childCompiler.scopes[0].locals[param.Name] = i
			childCompiler.scopes[0].types[param.Name] = normalizeType(param.Type)
			childCompiler.nextReg = i + 1
		}
		if childCompiler.nextReg > childCompiler.MaxRegUsed {
			childCompiler.MaxRegUsed = childCompiler.nextReg
		}

		// Compile body
		for _, stmt := range e.Body {
			childCompiler.compileStatement(stmt)
		}

		// Add return at end
		returnReg := childCompiler.newRegister()
		funcChunk.Emit(OP_LOAD_CONST, returnReg, funcChunk.AddConstant(value.MakeNull()), 0)
		funcChunk.Emit(OP_RETURN, returnReg, 0, 0)

		// Create function value
		fnValue := &FunctionObject{
			Chunk:        funcChunk,
			Arity:        len(e.Parameters),
			Name:         "<anonymous>",
			UpvalueCount: len(childCompiler.upvalues),
			Upvalues:     childCompiler.upvalues,
			MaxRegisters: childCompiler.MaxRegUsed,
			ParamTypes:   getParameterTypes(e.Parameters),
			ReturnType:   e.ReturnType,
		}

		reg := c.newRegister()
		idx := c.chunk.AddConstant(MakeFunction(fnValue))
		c.emit(OP_CLOSURE, reg, idx, 0)

		return reg

	case frontend.AssignmentExpression:
		switch e.Operator {
		case "=":
			switch target := e.Assignee.(type) {
			case frontend.Identifier:
				if slot, ok := c.lookupLocal(target.Symbol); ok {
					if bin, ok := e.Value.(frontend.BinaryExpression); ok && bin.Operator == "+" {
						if ident, ok := bin.Left.(frontend.Identifier); ok && ident.Symbol == target.Symbol {
							rhsReg := c.compileExpression(bin.Right)
							resultReg := c.newRegister()
							c.emit(OP_ADD_LOCAL, slot, rhsReg, resultReg)
							c.freeRegister(rhsReg)
							return resultReg
						}
						if ident, ok := bin.Right.(frontend.Identifier); ok && ident.Symbol == target.Symbol {
							lhsReg := c.compileExpression(bin.Left)
							resultReg := c.newRegister()
							c.emit(OP_ADD_LOCAL, slot, lhsReg, resultReg)
							c.freeRegister(lhsReg)
							return resultReg
						}
					}
				}

				valueReg := c.compileExpression(e.Value)
				if slot, ok := c.lookupLocal(target.Symbol); ok {
					// Local assignment - search scope stack
					c.emit(OP_MOVE, slot, valueReg, 0)
					if inferred := c.inferExprType(e.Value); inferred != "" {
						for i := len(c.scopes) - 1; i >= 0; i-- {
							if _, exists := c.scopes[i].locals[target.Symbol]; exists {
								c.scopes[i].types[target.Symbol] = inferred
								break
							}
						}
					}
				} else if up, ok := c.resolveUpvalue(target.Symbol); ok {
					c.emit(OP_SET_UPVALUE, valueReg, up, 0)
				} else {
					// Global assignment - A=valueReg, B=slot
					slot := c.getGlobalSlot(target.Symbol)
					c.emit(OP_SET_GLOBAL, valueReg, slot, 0)
					if inferred := c.inferExprType(e.Value); inferred != "" {
						c.globalTypesMap()[target.Symbol] = inferred
					}
				}
				return valueReg

			case frontend.MemberExpression:
				valueReg := c.compileExpression(e.Value)
				objReg := c.compileExpression(target.Object)
				if target.Computed {
					propReg := c.compileExpression(target.Property)
					// OP_SET_INDEX: A=valueReg, B=objReg, C=propReg
					c.emit(OP_SET_INDEX, valueReg, objReg, propReg)
					c.freeRegister(propReg)
					c.freeRegister(objReg)
				} else {
					if ident, ok := target.Property.(frontend.Identifier); ok {
						keyIdx := c.chunk.AddConstant(value.MakeString(ident.Symbol))
						// OP_SET_PROPERTY: A=valueReg, B=objReg, C=keyIdx
						c.emit(OP_SET_PROPERTY, valueReg, objReg, keyIdx)
						c.freeRegister(objReg)
					}
				}
				return valueReg
			}
		case "+=", "-=", "*=", "/=":
			valueReg := c.compileExpression(e.Value)
			switch target := e.Assignee.(type) {
			case frontend.Identifier:
				if slot, ok := c.lookupLocal(target.Symbol); ok {
					switch e.Operator {
					case "+=":
						c.emit(OP_ADD_LOCAL, slot, valueReg, valueReg)
					case "-=":
						c.emit(OP_SUB_LOCAL, slot, valueReg, valueReg)
					case "*=":
						c.emit(OP_MUL_LOCAL, slot, valueReg, valueReg)
					case "/=":
						c.emit(OP_DIV, valueReg, slot, valueReg)
					}
					// Store result back to local
					c.emit(OP_MOVE, slot, valueReg, 0)
				} else {
					slot := c.getGlobalSlot(target.Symbol)
					temp := c.newRegister()
					c.emit(OP_GET_GLOBAL, temp, slot, 0)
					switch e.Operator {
					case "+=":
						c.emit(OP_ADD, valueReg, temp, valueReg)
					case "-=":
						c.emit(OP_SUB, valueReg, temp, valueReg)
					case "*=":
						c.emit(OP_MUL, valueReg, temp, valueReg)
					case "/=":
						c.emit(OP_DIV, valueReg, temp, valueReg)
					}
					c.emit(OP_SET_GLOBAL, valueReg, slot, 0)
					c.freeRegister(temp)
				}
				return valueReg
			case frontend.MemberExpression:
				objReg := c.compileExpression(target.Object)
				if target.Computed {
					propReg := c.compileExpression(target.Property)
					// Get current value
					temp := c.newRegister()
					c.emit(OP_GET_INDEX, temp, objReg, propReg)
					// Perform operation
					switch e.Operator {
					case "+=":
						c.emit(OP_ADD, valueReg, temp, valueReg)
					case "-=":
						c.emit(OP_SUB, valueReg, temp, valueReg)
					case "*=":
						c.emit(OP_MUL, valueReg, temp, valueReg)
					case "/=":
						c.emit(OP_DIV, valueReg, temp, valueReg)
					}
					// Set the result
					c.emit(OP_SET_INDEX, valueReg, objReg, propReg)
					c.freeRegister(temp)
					c.freeRegister(propReg)
					c.freeRegister(objReg)
				} else {
					if ident, ok := target.Property.(frontend.Identifier); ok {
						keyIdx := c.chunk.AddConstant(value.MakeString(ident.Symbol))
						// Get current value
						temp := c.newRegister()
						c.emit(OP_GET_PROPERTY, temp, objReg, keyIdx)
						// Perform operation
						switch e.Operator {
						case "+=":
							c.emit(OP_ADD, valueReg, temp, valueReg)
						case "-=":
							c.emit(OP_SUB, valueReg, temp, valueReg)
						case "*=":
							c.emit(OP_MUL, valueReg, temp, valueReg)
						case "/=":
							c.emit(OP_DIV, valueReg, temp, valueReg)
						}
						// Set the result
						c.emit(OP_SET_PROPERTY, valueReg, objReg, keyIdx)
						c.freeRegister(temp)
						c.freeRegister(objReg)
					}
				}
				return valueReg
			}
		}

	}

	panic("not implemented")
}

func (c *Compiler) emitJumpIfFalse(condReg int) int {
	c.emit(OP_JUMP_IF_FALSE, condReg, -1, 0)
	return len(c.chunk.Code) - 1
}

func (c *Compiler) patchJumpIfFalse(index int) {
	offset := len(c.chunk.Code) - index - 1
	c.chunk.Code[index].B = offset
}

func (c *Compiler) emitJumpIfTrue(condReg int) int {
	c.emit(OP_JUMP_IF_TRUE, condReg, -1, 0)
	return len(c.chunk.Code) - 1
}

func (c *Compiler) patchJumpIfTrue(index int) {
	offset := len(c.chunk.Code) - index - 1
	c.chunk.Code[index].B = offset
}

func (c *Compiler) emitJump() int {
	c.emit(OP_JUMP, -1, 0, 0)
	return len(c.chunk.Code) - 1
}

func (c *Compiler) patchJump(index int) {
	offset := len(c.chunk.Code) - index - 1
	c.chunk.Code[index].A = offset
}

func (c *Compiler) emit(op OpCode, a, b, c2 int, extra ...int) {
	c.chunk.EmitAt(op, a, b, c2, c.currentLine, c.currentCol, extra...)
}

func statementLocation(stmt frontend.Statement) (int, int) {
	switch s := stmt.(type) {
	case frontend.VariableDeclaration:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		if s.Value != nil {
			return expressionLocation(s.Value)
		}
		return 0, 0
	case frontend.IfStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		return expressionLocation(s.Condition)
	case frontend.WhileStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		return expressionLocation(s.Condition)
	case frontend.ForStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		if s.Initial != nil {
			return statementLocation(s.Initial)
		}
		if s.Condition != nil {
			return expressionLocation(s.Condition)
		}
		if s.Iteration != nil {
			return expressionLocation(s.Iteration)
		}
		return 0, 0
	case frontend.FunctionDeclaration:
		return s.Line, s.Column
	case frontend.ReturnStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		if s.Value != nil {
			return expressionLocation(s.Value)
		}
		return 0, 0
	case frontend.TryCatchStatement:
		return s.Line, s.Column
	case frontend.BreakStatement:
		return s.Line, s.Column
	case frontend.ContinueStatement:
		return s.Line, s.Column
	case frontend.ImportStatement:
		return s.Line, s.Column
	case frontend.ExportStatement:
		return s.Line, s.Column
	case frontend.SwitchStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		return expressionLocation(s.Expr)
	case frontend.Expression:
		return expressionLocation(s)
	default:
		return 0, 0
	}
}

func expressionLocation(expr frontend.Expression) (int, int) {
	switch e := expr.(type) {
	case frontend.NumericLiteral:
		return e.Line, e.Column
	case frontend.FloatLiteral:
		return e.Line, e.Column
	case frontend.BooleanLiteral:
		return e.Line, e.Column
	case frontend.StringLiteral:
		return e.Line, e.Column
	case frontend.Identifier:
		return e.Line, e.Column
	case frontend.UnaryExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		return expressionLocation(e.Operand)
	case frontend.BinaryExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		if line, col := expressionLocation(e.Left); line > 0 {
			return line, col
		}
		return expressionLocation(e.Right)
	case frontend.AssignmentExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		if line, col := expressionLocation(e.Assignee); line > 0 {
			return line, col
		}
		return expressionLocation(e.Value)
	case frontend.ArrayLiteral:
		return e.Line, e.Column
	case frontend.MapLiteral:
		return e.Line, e.Column
	case frontend.MemberExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		if line, col := expressionLocation(e.Object); line > 0 {
			return line, col
		}
		return expressionLocation(e.Property)
	case frontend.CallExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		if line, col := expressionLocation(e.Callee); line > 0 {
			return line, col
		}
		if len(e.Arguments) > 0 {
			return expressionLocation(e.Arguments[0])
		}
		return 0, 0
	case frontend.FunctionExpression:
		return e.Line, e.Column
	case frontend.AwaitExpression:
		return e.Line, e.Column
	default:
		return 0, 0
	}
}

func getParameterTypes(params []frontend.Parameter) []string {
	types := make([]string, len(params))
	for i, p := range params {
		types[i] = p.Type
	}
	return types
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func normalizeType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "int", "integer":
		return "int"
	case "float", "double", "number":
		return "number"
	case "string":
		return "string"
	default:
		return ""
	}
}

func isNumericType(t string) bool {
	return t == "int" || t == "number"
}

func (c *Compiler) lookupType(name string) string {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if t, ok := c.scopes[i].types[name]; ok {
			return t
		}
	}
	if t, ok := c.globalTypesMap()[name]; ok {
		return t
	}
	return ""
}

func (c *Compiler) inferExprType(expr frontend.Expression) string {
	switch e := expr.(type) {
	case frontend.NumericLiteral:
		return "int"
	case frontend.FloatLiteral:
		return "number"
	case frontend.StringLiteral:
		return "string"
	case frontend.Identifier:
		return c.lookupType(e.Symbol)
	case frontend.BinaryExpression:
		lt := c.inferExprType(e.Left)
		rt := c.inferExprType(e.Right)
		switch e.Operator {
		case "+":
			if lt == "string" && rt == "string" {
				return "string"
			}
			if isNumericType(lt) && isNumericType(rt) {
				if lt == "int" && rt == "int" {
					return "int"
				}
				return "number"
			}
		case "-", "*", "/", "%":
			if isNumericType(lt) && isNumericType(rt) {
				return "number"
			}
		}
	}
	return ""
}

func (c *Compiler) selectAddOp(left, right frontend.Expression) OpCode {
	lt := c.inferExprType(left)
	rt := c.inferExprType(right)
	if lt == "string" && rt == "string" {
		return OP_ADD_STR
	}
	if lt == "int" && rt == "int" {
		return OP_ADD_INT
	}
	if isNumericType(lt) && isNumericType(rt) {
		return OP_ADD_FLOAT
	}
	return OP_ADD
}
func (c *Compiler) popLoop() LoopContext {
	if len(c.loopStack) == 0 {
		panic("loop stack underflow")
	}
	loop := c.loopStack[len(c.loopStack)-1]
	c.loopStack = c.loopStack[:len(c.loopStack)-1]
	return loop

}
func (c *Compiler) currentLoop() *LoopContext {
	if len(c.loopStack) == 0 {
		return nil
	}
	return &c.loopStack[len(c.loopStack)-1]
}
func (c *Compiler) emitBreakJump() int {
	loop := c.currentLoop()
	if loop == nil {
		panic("break statement not within a loop")
	}
	jumpIndex := c.emitJump()
	loop.breakJumps = append(loop.breakJumps, jumpIndex)
	return jumpIndex

}
func (c *Compiler) emitContinueJump() int {
	loop := c.currentLoop()
	if loop == nil {
		panic("continue statement not within a loop")
	}
	jumpIndex := c.emitJump()
	loop.continueJumps = append(loop.continueJumps, jumpIndex)
	return jumpIndex
}


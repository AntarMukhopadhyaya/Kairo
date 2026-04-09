package vm

import (
	"Kairo/frontend"
	"Kairo/value"
	"math"
)

type Optimizer struct {
	compiler *Compiler
	enabled  map[string]bool
	stats    OptimizationStats
}
type OptimizationStats struct {
	ConstantFolded   int
	DeadCodeRemoved  int
	PeepholeRewrites int
}

func NewOptimizer(compiler *Compiler) *Optimizer {
	return &Optimizer{
		compiler: compiler,
		enabled: map[string]bool{
			// Default OFF; main/compiler can toggle with -O.
			"constant_folding":     false,
			"dead_code":            false,
			"peephole":             false,
			"constant_propagation": false,
		},
	}
}

func (o *Optimizer) EnableOptimization(name string, enabled bool) {
	o.enabled[name] = enabled
}

func (o *Optimizer) Enabled(name string) bool {
	return o.enabled[name]
}

func (o *Optimizer) ResetStats() {
	o.stats = OptimizationStats{}
}

func (o *Optimizer) Optimize() {
	// Post-pass optimizations only. Expression/statement-level optimizations
	// are applied during compilation.
	if o.Enabled("peephole") {
		o.peepholeOptimize()
		o.compactNops()
	}
}

func (o *Optimizer) compactNops() {
	oldCode := o.compiler.chunk.Code
	if len(oldCode) == 0 {
		return
	}

	hasNops := false
	for i := 0; i < len(oldCode); i++ {
		if oldCode[i].Op == OP_NOP {
			hasNops = true
			break
		}
	}
	if !hasNops {
		return
	}

	// nextKept[i] gives the next non-NOP instruction index at or after i.
	// nextKept[len(oldCode)] == len(oldCode) is the virtual "end" address.
	nextKept := make([]int, len(oldCode)+1)
	nextKept[len(oldCode)] = len(oldCode)
	for i := len(oldCode) - 1; i >= 0; i-- {
		if oldCode[i].Op == OP_NOP {
			nextKept[i] = nextKept[i+1]
		} else {
			nextKept[i] = i
		}
	}

	newCode := make([]Instruction, 0, len(oldCode))
	newToOld := make([]int, 0, len(oldCode))
	oldToNewKept := make([]int, len(oldCode))
	for i := range oldToNewKept {
		oldToNewKept[i] = -1
	}
	for oldIdx := 0; oldIdx < len(oldCode); oldIdx++ {
		if oldCode[oldIdx].Op == OP_NOP {
			continue
		}
		oldToNewKept[oldIdx] = len(newCode)
		newCode = append(newCode, oldCode[oldIdx])
		newToOld = append(newToOld, oldIdx)
	}

	redirect := func(oldTarget int) int {
		if oldTarget < 0 {
			return 0
		}
		if oldTarget >= len(oldCode) {
			return len(newCode)
		}
		kept := nextKept[oldTarget]
		if kept >= len(oldCode) {
			return len(newCode)
		}
		mapped := oldToNewKept[kept]
		if mapped < 0 {
			return len(newCode)
		}
		return mapped
	}

	// Rewrite control-flow to match the new instruction indices.
	for newIdx := 0; newIdx < len(newCode); newIdx++ {
		instr := &newCode[newIdx]
		oldIdx := newToOld[newIdx]
		switch instr.Op {
		case OP_JUMP:
			oldTarget := oldIdx + 1 + instr.A
			newTarget := redirect(oldTarget)
			instr.A = newTarget - (newIdx + 1)
		case OP_JUMP_IF_FALSE, OP_JUMP_IF_TRUE:
			oldTarget := oldIdx + 1 + instr.B
			newTarget := redirect(oldTarget)
			instr.B = newTarget - (newIdx + 1)
		case OP_LOOP:
			oldTarget := oldIdx + 1 - instr.A
			newTarget := redirect(oldTarget)
			instr.A = (newIdx + 1) - newTarget
		case OP_TRY_BEGIN:
			if instr.A >= 0 {
				instr.A = redirect(instr.A)
			}
			if instr.C >= 0 {
				instr.C = redirect(instr.C)
			}
		}
	}

	o.compiler.chunk.Code = newCode
}
func (o *Optimizer) tryFoldConstant(expr frontend.Expression) (value.Value, bool) {
	switch e := expr.(type) {
	case frontend.NumericLiteral:
		return value.MakeNumber(e.Value), true
	case frontend.FloatLiteral:
		return value.MakeNumber(e.Value), true
	case frontend.StringLiteral:
		return value.MakeString(e.Value), true
	case frontend.BooleanLiteral:
		return value.MakeBool(e.Value), true
	case frontend.BinaryExpression:
		return o.tryFoldBinaryOp(e)
	case frontend.UnaryExpression:
		return o.tryFoldUnaryOp(e)

	}
	return value.Value{}, false
}
func (o *Optimizer) tryFoldBinaryOp(e frontend.BinaryExpression) (value.Value, bool) {
	// Recursively try to fold left and right
	left, leftOk := o.tryFoldConstant(e.Left)
	right, rightOk := o.tryFoldConstant(e.Right)

	if !leftOk || !rightOk {
		return value.Value{}, false
	}

	// Both sides are constants - fold them!
	if left.Kind == value.NumberKind && right.Kind == value.NumberKind {
		var result float64

		switch e.Operator {
		case "+":
			result = left.Num + right.Num
		case "-":
			result = left.Num - right.Num
		case "*":
			result = left.Num * right.Num
		case "/":
			if right.Num == 0 {
				return value.Value{}, false // Can't fold division by zero
			}
			result = left.Num / right.Num
		case "%":
			if right.Num == 0 {
				return value.Value{}, false
			}
			result = math.Mod(left.Num, right.Num)
		case ">":
			return value.MakeBool(left.Num > right.Num), true
		case "<":
			return value.MakeBool(left.Num < right.Num), true
		case ">=":
			return value.MakeBool(left.Num >= right.Num), true
		case "<=":
			return value.MakeBool(left.Num <= right.Num), true
		case "==":
			return value.MakeBool(left.Num == right.Num), true
		case "!=":
			return value.MakeBool(left.Num != right.Num), true
		default:
			return value.Value{}, false
		}

		return value.MakeNumber(result), true
	}

	// Handle string concatenation: VM concatenates if either side is a string.
	if e.Operator == "+" && (left.Kind == value.StringKind || right.Kind == value.StringKind) {
		return value.MakeString(left.ToString() + right.ToString()), true
	}

	// Handle boolean operators
	if e.Operator == "&&" && left.Kind == value.BoolKind && right.Kind == value.BoolKind {
		return value.MakeBool(left.Bool && right.Bool), true
	}

	if e.Operator == "||" && left.Kind == value.BoolKind && right.Kind == value.BoolKind {
		return value.MakeBool(left.Bool || right.Bool), true
	}

	return value.Value{}, false
}
func (o *Optimizer) tryFoldUnaryOp(e frontend.UnaryExpression) (value.Value, bool) {
	operand, ok := o.tryFoldConstant(e.Operand)
	if !ok {
		return value.Value{}, false
	}
	switch e.Operator {
	case "-":
		if operand.Kind == value.NumberKind {
			return value.MakeNumber(-operand.Num), true
		}
	case "!":
		if operand.Kind == value.BoolKind {
			return value.MakeBool(!operand.Bool), true
		}
		return value.MakeBool(isFalsey(operand)), true
	}
	return value.Value{}, false
}
func (o *Optimizer) isConstantCondition(expr frontend.Expression) (bool, bool) {
	val, ok := o.tryFoldConstant(expr)
	if !ok {
		return false, false
	}
	// Important: must match VM truthiness exactly.
	return true, !isFalsey(val)
}

func (o *Optimizer) shouldEliminateStatement(stmt frontend.Statement) bool {
	switch s := stmt.(type) {
	case frontend.IfStatement:
		isConst, value := o.isConstantCondition(s.Condition)
		if isConst {
			if !value && len(s.Alternate) == 0 {
				return true // if (false) { ... } with no else can be removed
			}
		}
	}
	return false
}

func (o *Optimizer) NoteConstantFolded() {
	o.stats.ConstantFolded++
}

func (o *Optimizer) NoteDeadCodeRemoved() {
	o.stats.DeadCodeRemoved++
}
func (o *Optimizer) peepholeOptimize() {
	// Important: Do NOT delete instructions here unless you also rewrite
	// jump offsets. We only do local opcode rewrites that preserve code length.
	code := o.compiler.chunk.Code
	constants := o.compiler.chunk.Constants

	for i := 0; i < len(code); i++ {
		instr := code[i]

		// Superinstruction fusion:
		//   ADD tmp, local, rhs ; MOVE local, tmp  => ADD_LOCAL local, rhs, tmp
		//   ADD tmp, lhs, local ; MOVE local, tmp  => ADD_LOCAL local, lhs, tmp
		if instr.Op == OP_ADD && i+1 < len(code) {
			next := code[i+1]
			if next.Op == OP_MOVE && next.B == instr.A {
				localSlot := next.A
				if instr.B == localSlot {
					code[i] = Instruction{Op: OP_ADD_LOCAL, A: localSlot, B: instr.C, C: instr.A}
					code[i+1] = Instruction{Op: OP_NOP}
					o.stats.PeepholeRewrites++
					continue
				}
				if instr.C == localSlot {
					code[i] = Instruction{Op: OP_ADD_LOCAL, A: localSlot, B: instr.B, C: instr.A}
					code[i+1] = Instruction{Op: OP_NOP}
					o.stats.PeepholeRewrites++
					continue
				}
			}
		}

		// Superinstruction fusion:
		//   SUB tmp, local, rhs ; MOVE local, tmp  => SUB_LOCAL local, rhs, tmp
		//   MUL tmp, local, rhs ; MOVE local, tmp  => MUL_LOCAL local, rhs, tmp
		if (instr.Op == OP_SUB || instr.Op == OP_MUL) && i+1 < len(code) {
			next := code[i+1]
			if next.Op == OP_MOVE && next.B == instr.A {
				localSlot := next.A
				if instr.B == localSlot {
					op := OP_SUB_LOCAL
					if instr.Op == OP_MUL {
						op = OP_MUL_LOCAL
					}
					code[i] = Instruction{Op: op, A: localSlot, B: instr.C, C: instr.A}
					code[i+1] = Instruction{Op: OP_NOP}
					o.stats.PeepholeRewrites++
					continue
				}
			}
		}

		// Zero-offset control flow is a no-op.
		// (ip already advanced past this instruction when the VM applies offsets)
		switch instr.Op {
		case OP_JUMP:
			if instr.A == 0 {
				code[i] = Instruction{Op: OP_NOP}
				o.stats.PeepholeRewrites++
				continue
			}
		case OP_JUMP_IF_FALSE, OP_JUMP_IF_TRUE:
			if instr.B == 0 {
				code[i] = Instruction{Op: OP_NOP}
				o.stats.PeepholeRewrites++
				continue
			}
		}

		// Pattern: ADD dst, x, const0Reg  => MOVE dst, x
		if instr.Op == OP_ADD && i > 0 {
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				prev := code[j]
				if prev.Op == OP_LOAD_CONST && prev.A == instr.C {
					cv := constants[prev.B]
					if cv.Kind == value.NumberKind && cv.Num == 0 {
						code[i] = Instruction{Op: OP_MOVE, A: instr.A, B: instr.B, C: 0}
						o.stats.PeepholeRewrites++
					}
					break
				}
			}
		}

		// Pattern: MUL dst, x, const1Reg  => MOVE dst, x
		if instr.Op == OP_MUL && i > 0 {
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				prev := code[j]
				if prev.Op == OP_LOAD_CONST && prev.A == instr.C {
					cv := constants[prev.B]
					if cv.Kind == value.NumberKind && cv.Num == 1 {
						code[i] = Instruction{Op: OP_MOVE, A: instr.A, B: instr.B, C: 0}
						o.stats.PeepholeRewrites++
					}
					break
				}
			}
		}
		if instr.Op == OP_SUB && i > 0 {
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				prev := code[j]
				if prev.Op == OP_LOAD_CONST && prev.A == instr.C {
					cv := constants[prev.B]
					if cv.Kind == value.NumberKind && cv.Num == 0 {
						code[i] = Instruction{Op: OP_MOVE, A: instr.A, B: instr.B, C: 0}
						o.stats.PeepholeRewrites++
					}
					break
				}
			}
		}
		// MUL x, 0 -> LOAD_CONST 0
		if instr.Op == OP_MUL && i > 0 {
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				prev := code[j]
				if prev.Op == OP_LOAD_CONST && prev.A == instr.C {
					cv := constants[prev.B]
					if cv.Kind == value.NumberKind && cv.Num == 0 {
						code[i] = Instruction{Op: OP_LOAD_CONST, A: instr.A, B: prev.B}
						o.stats.PeepholeRewrites++
					}
					break
				}
			}
		}
		if instr.Op == OP_LOAD_CONST && i > 0 {
			prev := code[i-1]
			if prev.Op == OP_LOAD_CONST && prev.A == instr.A && prev.B == instr.B {
				code[i] = Instruction{Op: OP_NOP, A: 0, B: 0, C: 0}
				o.stats.PeepholeRewrites++
			}
		}

		if instr.Op == OP_MOVE && i > 0 {
			prev := code[i-1]
			if prev.Op == OP_MOVE && prev.A == instr.B && prev.B == instr.A {
				code[i] = Instruction{Op: OP_NOP, A: 0, B: 0, C: 0}
				o.stats.PeepholeRewrites++
			}
		}
		if instr.Op == OP_MOVE && instr.A == instr.B {
			code[i] = Instruction{Op: OP_NOP}
			o.stats.PeepholeRewrites++
		}
	}

	o.compiler.chunk.Code = code
}
func (o *Optimizer) GetStats() OptimizationStats {
	return o.stats
}


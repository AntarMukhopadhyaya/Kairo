package vm

import "Kairo/value"

type CallFrame struct {
	closure   *ClosureObject
	ip        int
	regs      []value.Value
	ReturnReg int // register to store return value in caller

	openUpvalues map[int]*Upvalue
}

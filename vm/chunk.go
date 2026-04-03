package vm

import "Kairo/value"

type Instruction struct {
	Op OpCode
	A  int
	B  int
	C  int
	D  int
}
type Chunk struct {
	Code      []Instruction
	Constants []value.Value
}

func NewChunk() *Chunk {
	return &Chunk{
		Code:      []Instruction{},
		Constants: []value.Value{},
	}
}

func (c *Chunk) AddConstant(val value.Value) int {
	c.Constants = append(c.Constants, val)
	return len(c.Constants) - 1
}

func (c *Chunk) Emit(op OpCode, a, b, c2 int, extra ...int) {
	d := 0
	if len(extra) > 0 {
		d = extra[0]
	}
	c.Code = append(c.Code, Instruction{
		Op: op,
		A:  a,
		B:  b,
		C:  c2,
		D:  d,
	})
}

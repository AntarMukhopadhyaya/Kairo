package vm

import (
	"Kairo/value"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Bytecode file format for Kairo (.krc files)
// This format is designed to be simple to parse in C++ or other languages
//
// Format:
// ┌─────────────────────────────────────────────────┐
// │ Magic Number: "KIRO" (4 bytes)                  │
// │ Version: Major (1 byte), Minor (1 byte)         │
// │ Constants Count (4 bytes)                       │
// │ Constants Section (variable)                    │
// │ Code Count (4 bytes)                            │
// │ Code Section (variable)                         │
// │ Global Slots Count (4 bytes)                    │
// │ Global Slots Section (variable)                 │
// │ MaxRegisters (4 bytes)                          │
// └─────────────────────────────────────────────────┘

const (
	// Magic number for bytecode files
	MagicNumber = "KIRO" // Kairo Compiled

	// Version
	VersionMajor = 1
	VersionMinor = 1

	// Constant type tags
	ConstNull         byte = 0
	ConstNumber       byte = 1
	ConstString       byte = 2
	ConstBool         byte = 3
	ConstInternalFunc byte = 4
	ConstFunction     byte = 5
)

// BytecodeWriter writes compiled bytecode to a binary format
type BytecodeWriter struct {
	writer io.Writer
}

func NewBytecodeWriter(w io.Writer) *BytecodeWriter {
	return &BytecodeWriter{writer: w}
}

// WriteChunk serializes a compiled chunk to bytecode format
func (bw *BytecodeWriter) WriteChunk(chunk *Chunk, maxRegisters int, slots map[string]int) error {
	// Write magic number
	if _, err := bw.writer.Write([]byte(MagicNumber)); err != nil {
		return err
	}

	// Write version
	if err := bw.writeByte(VersionMajor); err != nil {
		return err
	}
	if err := bw.writeByte(VersionMinor); err != nil {
		return err
	}

	// Write constants section
	if err := bw.writeConstants(chunk.Constants); err != nil {
		return err
	}

	// Write code section
	if err := bw.writeCode(chunk.Code); err != nil {
		return err
	}

	// Write global slots section
	if err := bw.writeGlobalSlots(slots); err != nil {
		return err
	}

	// Write max registers
	if err := bw.writeUint32(uint32(maxRegisters)); err != nil {
		return err
	}

	return nil
}

func (bw *BytecodeWriter) writeByte(b byte) error {
	_, err := bw.writer.Write([]byte{b})
	return err
}

func (bw *BytecodeWriter) writeUint32(val uint32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, val)
	_, err := bw.writer.Write(buf)
	return err
}

func (bw *BytecodeWriter) writeUint64(val uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, val)
	_, err := bw.writer.Write(buf)
	return err
}

func (bw *BytecodeWriter) writeFloat64(val float64) error {
	return bw.writeUint64(math.Float64bits(val))
}

func (bw *BytecodeWriter) writeString(s string) error {
	// Write length
	if err := bw.writeUint32(uint32(len(s))); err != nil {
		return err
	}
	// Write string data
	_, err := bw.writer.Write([]byte(s))
	return err
}

func (bw *BytecodeWriter) writeConstants(constants []value.Value) error {
	// Write count
	if err := bw.writeUint32(uint32(len(constants))); err != nil {
		return err
	}

	// Write each constant
	for _, c := range constants {
		if err := bw.writeConstant(c); err != nil {
			return err
		}
	}

	return nil
}

func (bw *BytecodeWriter) writeConstant(c value.Value) error {
	switch c.Kind {
	case value.NullKind:
		return bw.writeByte(ConstNull)

	case value.NumberKind:
		if err := bw.writeByte(ConstNumber); err != nil {
			return err
		}
		return bw.writeFloat64(c.Num)

	case value.StringKind:
		if err := bw.writeByte(ConstString); err != nil {
			return err
		}
		return bw.writeString(c.Obj.Data.(string))

	case value.BoolKind:
		if err := bw.writeByte(ConstBool); err != nil {
			return err
		}
		val := byte(0)
		if c.Bool {
			val = 1
		}
		return bw.writeByte(val)

	case value.FunctionKind:
		if err := bw.writeByte(ConstFunction); err != nil {
			return err
		}
		fn := AsFunction(c)
		return bw.writeFunction(fn)

	case value.InternalFunctionKind:
		// Internal functions: we'll just mark them and reconstruct from globals
		if err := bw.writeByte(ConstInternalFunc); err != nil {
			return err
		}
		// Write a marker - will be reconstructed when loading
		return bw.writeString("")

	default:
		return fmt.Errorf("unsupported constant type: %v", c.Kind)
	}
}

func (bw *BytecodeWriter) writeFunction(fn *FunctionObject) error {
	// Write function name
	if err := bw.writeString(fn.Name); err != nil {
		return err
	}

	// Write arity
	if err := bw.writeUint32(uint32(fn.Arity)); err != nil {
		return err
	}

	// Write max registers
	if err := bw.writeUint32(uint32(fn.MaxRegisters)); err != nil {
		return err
	}

	// Write upvalue count
	if err := bw.writeUint32(uint32(fn.UpvalueCount)); err != nil {
		return err
	}

	// Write upvalues
	for _, uv := range fn.Upvalues {
		if err := bw.writeByte(boolToByte(uv.IsLocal)); err != nil {
			return err
		}
		if err := bw.writeUint32(uint32(uv.Index)); err != nil {
			return err
		}
	}

	// Write nested constants
	if err := bw.writeConstants(fn.Chunk.Constants); err != nil {
		return err
	}

	// Write nested code
	return bw.writeCode(fn.Chunk.Code)
}

func (bw *BytecodeWriter) writeCode(code []Instruction) error {
	// Write instruction count
	if err := bw.writeUint32(uint32(len(code))); err != nil {
		return err
	}

	// Write each instruction: 1-byte opcode + 3-byte padding + 6x int32 fields.
	for _, instr := range code {
		if err := bw.writeByte(byte(instr.Op)); err != nil {
			return err
		}
		// Pad to align
		if err := bw.writeByte(0); err != nil {
			return err
		}
		if err := bw.writeByte(0); err != nil {
			return err
		}
		if err := bw.writeByte(0); err != nil {
			return err
		}

		if err := bw.writeInt32(int32(instr.A)); err != nil {
			return err
		}
		if err := bw.writeInt32(int32(instr.B)); err != nil {
			return err
		}
		if err := bw.writeInt32(int32(instr.C)); err != nil {
			return err
		}
		if err := bw.writeInt32(int32(instr.D)); err != nil {
			return err
		}
		if err := bw.writeInt32(int32(instr.Line)); err != nil {
			return err
		}
		if err := bw.writeInt32(int32(instr.Column)); err != nil {
			return err
		}
	}

	return nil
}

func (bw *BytecodeWriter) writeInt32(val int32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(val))
	_, err := bw.writer.Write(buf)
	return err
}

func (bw *BytecodeWriter) writeGlobalSlots(slots map[string]int) error {
	// Write count
	if err := bw.writeUint32(uint32(len(slots))); err != nil {
		return err
	}

	// Write each slot mapping
	for name, slot := range slots {
		if err := bw.writeString(name); err != nil {
			return err
		}
		if err := bw.writeUint32(uint32(slot)); err != nil {
			return err
		}
	}

	return nil
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// BytecodeReader reads compiled bytecode from binary format
type BytecodeReader struct {
	reader       io.Reader
	versionMinor byte
}

func NewBytecodeReader(r io.Reader) *BytecodeReader {
	return &BytecodeReader{reader: r}
}

// ReadChunk deserializes bytecode into a chunk
func (br *BytecodeReader) ReadChunk() (*Chunk, int, map[string]int, error) {
	// Read and verify magic number
	magic := make([]byte, 4)
	if _, err := io.ReadFull(br.reader, magic); err != nil {
		return nil, 0, nil, err
	}
	if string(magic) != MagicNumber {
		return nil, 0, nil, fmt.Errorf("invalid bytecode file: wrong magic number")
	}

	// Read version
	major, err := br.readByte()
	if err != nil {
		return nil, 0, nil, err
	}
	minor, err := br.readByte()
	if err != nil {
		return nil, 0, nil, err
	}

	if major != VersionMajor {
		return nil, 0, nil, fmt.Errorf("incompatible bytecode version: %d.%d (expected %d.x)",
			major, minor, VersionMajor)
	}
	br.versionMinor = minor

	// First, read global slots so we can use them for stdlib lookup
	// Read constants count to advance past constants
	constCount, err := br.readUint32()
	if err != nil {
		return nil, 0, nil, err
	}

	chunk := &Chunk{}

	// Read constants (will need slots for stdlib lookup)
	// We'll read slots first, then come back
	// Actually, we need to reorganize this...

	// Read constants
	constants, err := br.readConstantsWithCount(constCount)
	if err != nil {
		return nil, 0, nil, err
	}
	chunk.Constants = constants

	// Read code
	code, err := br.readCode()
	if err != nil {
		return nil, 0, nil, err
	}
	chunk.Code = code

	// Read global slots
	slots, err := br.readGlobalSlots()
	if err != nil {
		return nil, 0, nil, err
	}

	// Read max registers
	maxRegs, err := br.readUint32()
	if err != nil {
		return nil, 0, nil, err
	}

	return chunk, int(maxRegs), slots, nil
}

func (br *BytecodeReader) readConstantsWithCount(count uint32) ([]value.Value, error) {
	constants := make([]value.Value, count)
	for i := uint32(0); i < count; i++ {
		c, err := br.readConstant()
		if err != nil {
			return nil, err
		}
		constants[i] = c
	}

	return constants, nil
}

func (br *BytecodeReader) readByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := io.ReadFull(br.reader, buf)
	return buf[0], err
}

func (br *BytecodeReader) readUint32() (uint32, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(br.reader, buf); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}

func (br *BytecodeReader) readInt32() (int32, error) {
	val, err := br.readUint32()
	return int32(val), err
}

func (br *BytecodeReader) readUint64() (uint64, error) {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(br.reader, buf); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func (br *BytecodeReader) readFloat64() (float64, error) {
	bits, err := br.readUint64()
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(bits), nil
}

func (br *BytecodeReader) readString() (string, error) {
	length, err := br.readUint32()
	if err != nil {
		return "", err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(br.reader, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (br *BytecodeReader) readConstants() ([]value.Value, error) {
	count, err := br.readUint32()
	if err != nil {
		return nil, err
	}

	constants := make([]value.Value, count)
	for i := uint32(0); i < count; i++ {
		c, err := br.readConstant()
		if err != nil {
			return nil, err
		}
		constants[i] = c
	}

	return constants, nil
}

func (br *BytecodeReader) readConstant() (value.Value, error) {
	tag, err := br.readByte()
	if err != nil {
		return value.Value{}, err
	}

	switch tag {
	case ConstNull:
		return value.MakeNull(), nil

	case ConstNumber:
		num, err := br.readFloat64()
		if err != nil {
			return value.Value{}, err
		}
		return value.MakeNumber(num), nil

	case ConstString:
		s, err := br.readString()
		if err != nil {
			return value.Value{}, err
		}
		return value.MakeString(s), nil

	case ConstBool:
		b, err := br.readByte()
		if err != nil {
			return value.Value{}, err
		}
		return value.MakeBool(b != 0), nil

	case ConstFunction:
		fn, err := br.readFunction()
		if err != nil {
			return value.Value{}, err
		}
		return MakeFunction(fn), nil

	case ConstInternalFunc:
		// Internal functions will be loaded from globals during execution
		// For now, we create a placeholder that will be replaced
		_, err := br.readString() // Read the marker
		if err != nil {
			return value.Value{}, err
		}
		// Return a null placeholder - the actual function will come from globals
		return value.MakeNull(), nil

	default:
		return value.Value{}, fmt.Errorf("unknown constant tag: %d", tag)
	}
}

func (br *BytecodeReader) readFunction() (*FunctionObject, error) {
	fn := &FunctionObject{}

	// Read name
	name, err := br.readString()
	if err != nil {
		return nil, err
	}
	fn.Name = name

	// Read arity
	arity, err := br.readUint32()
	if err != nil {
		return nil, err
	}
	fn.Arity = int(arity)

	// Read max registers
	maxRegs, err := br.readUint32()
	if err != nil {
		return nil, err
	}
	fn.MaxRegisters = int(maxRegs)

	// Read upvalue count
	upvalCount, err := br.readUint32()
	if err != nil {
		return nil, err
	}
	fn.UpvalueCount = int(upvalCount)

	// Read upvalues
	fn.Upvalues = make([]UpvalueDescriptor, fn.UpvalueCount)
	for i := 0; i < fn.UpvalueCount; i++ {
		isLocal, err := br.readByte()
		if err != nil {
			return nil, err
		}
		index, err := br.readUint32()
		if err != nil {
			return nil, err
		}
		fn.Upvalues[i] = UpvalueDescriptor{
			IsLocal: isLocal != 0,
			Index:   int(index),
		}
	}

	// Read nested chunk
	fn.Chunk = &Chunk{}

	// Read constants
	constants, err := br.readConstants()
	if err != nil {
		return nil, err
	}
	fn.Chunk.Constants = constants

	// Read code
	code, err := br.readCode()
	if err != nil {
		return nil, err
	}
	fn.Chunk.Code = code

	return fn, nil
}

func (br *BytecodeReader) readCode() ([]Instruction, error) {
	count, err := br.readUint32()
	if err != nil {
		return nil, err
	}

	instructions := make([]Instruction, count)
	for i := uint32(0); i < count; i++ {
		// Read opcode
		op, err := br.readByte()
		if err != nil {
			return nil, err
		}

		// Read padding (3 bytes)
		if _, err := br.readByte(); err != nil {
			return nil, err
		}
		if _, err := br.readByte(); err != nil {
			return nil, err
		}
		if _, err := br.readByte(); err != nil {
			return nil, err
		}

		// Read operands
		a, err := br.readInt32()
		if err != nil {
			return nil, err
		}
		b, err := br.readInt32()
		if err != nil {
			return nil, err
		}
		c, err := br.readInt32()
		if err != nil {
			return nil, err
		}
		d, err := br.readInt32()
		if err != nil {
			return nil, err
		}
		line := int32(0)
		column := int32(0)
		if br.versionMinor >= 1 {
			line, err = br.readInt32()
			if err != nil {
				return nil, err
			}
			column, err = br.readInt32()
			if err != nil {
				return nil, err
			}
		}

		instructions[i] = Instruction{
			Op:     OpCode(op),
			A:      int(a),
			B:      int(b),
			C:      int(c),
			D:      int(d),
			Line:   int(line),
			Column: int(column),
		}
	}

	return instructions, nil
}

func (br *BytecodeReader) readGlobalSlots() (map[string]int, error) {
	count, err := br.readUint32()
	if err != nil {
		return nil, err
	}

	slots := make(map[string]int)
	for i := uint32(0); i < count; i++ {
		name, err := br.readString()
		if err != nil {
			return nil, err
		}
		slot, err := br.readUint32()
		if err != nil {
			return nil, err
		}
		slots[name] = int(slot)
	}

	return slots, nil
}


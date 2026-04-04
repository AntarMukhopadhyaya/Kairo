package value

import (
	"strconv"
	"strings"
)

type ValueKind uint8

const (
	NullKind ValueKind = iota
	NumberKind
	StringKind
	RopeStringKind
	BoolKind
	ArrayKind
	MapKind
	FunctionKind
	InternalFunctionKind
	ClosureKind
	ErrorKind
)

type ropeString struct {
	left     Value
	right    Value
	length   int
	cached   string
	hasCache bool
}

func (r *ropeString) flatten() string {
	if r.hasCache {
		return r.cached
	}
	var b strings.Builder
	if r.length > 0 {
		b.Grow(r.length)
	}
	appendStringLike(&b, r.left)
	appendStringLike(&b, r.right)
	r.cached = b.String()
	r.hasCache = true
	return r.cached
}

func appendStringLike(b *strings.Builder, v Value) {
	switch v.Kind {
	case StringKind:
		b.WriteString(v.Obj.(string))
	case RopeStringKind:
		r := v.Obj.(*ropeString)
		appendStringLike(b, r.left)
		appendStringLike(b, r.right)
	default:
		b.WriteString(v.ToString())
	}
}

type Value struct {
	Kind ValueKind
	Num  float64
	Bool bool
	Obj  any
}

// InternalFunctionObject is a heap-backed builtin function.
// Call receives a non-owning slice view of the caller's arguments.
type InternalFunctionObject struct {
	Arity int // -1 means variadic
	Call  func(args []Value) Value
}

type ErrorObject struct {
	Message    string
	ErrorType  string
	Line       int
	Column     int
	StackTrace []string
}
type ArrayObject struct {
	Elements []Value
}
type MapObject struct {
	Properties map[string]Value
}

func MakeNull() Value {
	return Value{Kind: NullKind}
}
func MakeNumber(num float64) Value {
	return Value{Kind: NumberKind, Num: num}
}
func MakeString(str string) Value {
	return Value{Kind: StringKind, Obj: str}
}

func MakeBool(b bool) Value {
	return Value{Kind: BoolKind, Bool: b}
}

func MakeArray(elements []Value) Value {
	return Value{Kind: ArrayKind, Obj: &ArrayObject{Elements: elements}}
}
func MakeMap(properties map[string]Value) Value {
	return Value{Kind: MapKind, Obj: &MapObject{Properties: properties}}
}

func MakeInternalFunction(fn *InternalFunctionObject) Value {
	return Value{Kind: InternalFunctionKind, Obj: fn}
}

func MakeError(message, errorType string, line, column int) Value {
	return Value{Kind: ErrorKind, Obj: &ErrorObject{Message: message, ErrorType: errorType, Line: line, Column: column, StackTrace: nil}}
}

func (v Value) IsNull() bool {
	return v.Kind == NullKind
}
func (v Value) IsNumber() bool {
	return v.Kind == NumberKind
}
func (v Value) IsString() bool {
	return v.Kind == StringKind || v.Kind == RopeStringKind
}

func IsStringLike(v Value) bool {
	return v.Kind == StringKind || v.Kind == RopeStringKind
}

func StringLen(v Value) int {
	switch v.Kind {
	case StringKind:
		return len(v.Obj.(string))
	case RopeStringKind:
		return v.Obj.(*ropeString).length
	default:
		return len(v.ToString())
	}
}

// ConcatStrings implements the language semantics of `+` when either side is a string:
// it stringifies non-string operands, but avoids repeatedly copying the growing prefix
// by building a rope that flattens only when needed.
func ConcatStrings(a, b Value) Value {
	if !IsStringLike(a) {
		a = MakeString(a.ToString())
	}
	if !IsStringLike(b) {
		b = MakeString(b.ToString())
	}

	// Fast path for small flat strings.
	if a.Kind == StringKind && b.Kind == StringKind {
		as := a.Obj.(string)
		bs := b.Obj.(string)
		if len(as)+len(bs) <= 64 {
			return MakeString(as + bs)
		}
		return Value{Kind: RopeStringKind, Obj: &ropeString{left: a, right: b, length: len(as) + len(bs)}}
	}

	return Value{Kind: RopeStringKind, Obj: &ropeString{left: a, right: b, length: StringLen(a) + StringLen(b)}}
}

func (v Value) AsArray() *ArrayObject {
	if v.Kind != ArrayKind {
		panic("Value is not an array")
	}
	return v.Obj.(*ArrayObject)
}
func (v Value) AsMap() *MapObject {
	if v.Kind != MapKind {
		panic("Value is not a map")
	}
	return v.Obj.(*MapObject)
}

func (v Value) AsInternalFunction() *InternalFunctionObject {
	if v.Kind != InternalFunctionKind {
		panic("Value is not an internal function")
	}
	return v.Obj.(*InternalFunctionObject)
}

func (v Value) AsError() *ErrorObject {
	if v.Kind != ErrorKind {
		panic("Value is not an error")
	}
	return v.Obj.(*ErrorObject)
}

func (v Value) ToString() string {
	switch v.Kind {
	case NullKind:
		return "null"
	case NumberKind:
		return strconv.FormatFloat(v.Num, 'g', -1, 64)
	case BoolKind:
		if v.Bool {
			return "true"
		}
		return "false"
	case StringKind:
		return v.Obj.(string)
	case RopeStringKind:
		return v.Obj.(*ropeString).flatten()
	case ArrayKind:
		arr := v.AsArray()
		result := ""
		for i, el := range arr.Elements {
			if i > 0 {
				result += ", "
			}
			result += el.ToString()
		}
		return "[" + result + "]"
	case MapKind:
		m := v.AsMap()
		result := ""
		i := 0
		for k, val := range m.Properties {
			if i > 0 {
				result += ", "
			}
			result += k + ": " + val.ToString()
			i++
		}
		return "{" + result + "}"
	case FunctionKind:
		return "<function>"
	case InternalFunctionKind:
		return "<internal_function>"
	case ClosureKind:
		return "<closure>"
	case ErrorKind:
		err := v.AsError()
		loc := ""
		if err.Line > 0 {
			loc = " at " + strconv.Itoa(err.Line) + ":" + strconv.Itoa(err.Column)
		}
		if err.ErrorType == "" {
			return "[Error]" + loc + " " + err.Message
		}
		return "[Error] " + err.ErrorType + loc + ": " + err.Message
	default:
		return "<object>"
	}
}

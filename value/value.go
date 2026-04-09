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

type HeapObject struct {
	RefCount int
	Data     any
}

var ExternalFreeObject func(obj *HeapObject)

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
		b.WriteString(v.Obj.Data.(string))
	case RopeStringKind:
		r := v.Obj.Data.(*ropeString)
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
	Obj  *HeapObject
}

// CallContext is the execution context passed to internal functions.
// VM implements this interface to allow builtins/methods to invoke callbacks.
type CallContext interface {
	CallFunction(fn Value, args []Value) Value
}

// InternalFunctionObject is a heap-backed builtin function.
// Call receives a non-owning slice view of the caller's arguments.
type InternalFunctionObject struct {
	Arity int // -1 means variadic
	Call  func(ctx CallContext, args []Value) Value
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
	return Value{Kind: StringKind, Obj: &HeapObject{RefCount: 0, Data: str}}
}

func MakeBool(b bool) Value {
	return Value{Kind: BoolKind, Bool: b}
}

func MakeArray(elements []Value) Value {
	for _, element := range elements {
		Inc(element)
	}
	return Value{Kind: ArrayKind, Obj: &HeapObject{RefCount: 0, Data: &ArrayObject{Elements: elements}}}
}
func MakeMap(properties map[string]Value) Value {
	for _, valueItem := range properties {
		Inc(valueItem)
	}
	return Value{Kind: MapKind, Obj: &HeapObject{RefCount: 0, Data: &MapObject{Properties: properties}}}
}

func MakeInternalFunction(fn *InternalFunctionObject) Value {
	return Value{Kind: InternalFunctionKind, Obj: &HeapObject{RefCount: 0, Data: fn}}
}

func MakeError(message, errorType string, line, column int) Value {
	return Value{Kind: ErrorKind, Obj: &HeapObject{RefCount: 0, Data: &ErrorObject{Message: message, ErrorType: errorType, Line: line, Column: column, StackTrace: nil}}}
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
		return len(v.Obj.Data.(string))
	case RopeStringKind:
		return v.Obj.Data.(*ropeString).length
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
		as := a.Obj.Data.(string)
		bs := b.Obj.Data.(string)
		if len(as)+len(bs) <= 64 {
			return MakeString(as + bs)
		}
		Inc(a)
		Inc(b)
		return Value{Kind: RopeStringKind, Obj: &HeapObject{RefCount: 1, Data: &ropeString{left: a, right: b, length: len(as) + len(bs)}}}
	}

	Inc(a)
	Inc(b)
	return Value{Kind: RopeStringKind, Obj: &HeapObject{RefCount: 1, Data: &ropeString{left: a, right: b, length: StringLen(a) + StringLen(b)}}}
}

func (v Value) AsArray() *ArrayObject {
	if v.Kind != ArrayKind {
		panic("Value is not an array")
	}
	return v.Obj.Data.(*ArrayObject)
}
func (v Value) AsMap() *MapObject {
	if v.Kind != MapKind {
		panic("Value is not a map")
	}
	return v.Obj.Data.(*MapObject)
}

func (v Value) AsInternalFunction() *InternalFunctionObject {
	if v.Kind != InternalFunctionKind {
		panic("Value is not an internal function")
	}
	return v.Obj.Data.(*InternalFunctionObject)
}

func (v Value) AsError() *ErrorObject {
	if v.Kind != ErrorKind {
		panic("Value is not an error")
	}
	return v.Obj.Data.(*ErrorObject)
}

func IsHeap(v Value) bool {
	switch v.Kind {
	case ArrayKind, MapKind, StringKind, RopeStringKind, ClosureKind, ErrorKind:
		return true
	}
	return false
}

func Inc(v Value) {
	if !IsHeap(v) || v.Obj == nil {
		return
	}
	v.Obj.RefCount++
}

func Dec(v Value) {
	if !IsHeap(v) || v.Obj == nil {
		return
	}

	obj := v.Obj
	obj.RefCount--

	if obj.RefCount == 0 {
		FreeObject(obj)
	}
}

func FreeObject(obj *HeapObject) {
	switch data := obj.Data.(type) {
	case *ArrayObject:
		for _, v := range data.Elements {
			Dec(v)
		}

	case *MapObject:
		for _, v := range data.Properties {
			Dec(v)
		}

	case *ropeString:
		Dec(data.left)
		Dec(data.right)

	case *ErrorObject:
		// No nested heap values to release.

	case string:
		// Flat strings contain no nested heap values.

	default:
		if ExternalFreeObject != nil {
			ExternalFreeObject(obj)
		}
	}
}

func Assign(dst *Value, src Value) {
	Inc(src)
	Dec(*dst)
	*dst = src
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
		return v.Obj.Data.(string)
	case RopeStringKind:
		return v.Obj.Data.(*ropeString).flatten()
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

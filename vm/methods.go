package vm

import (
	"Kairo/value"
	"strings"
)

func initMethodRegistry() map[value.ValueKind]map[string]*value.InternalFunctionObject {
	registry := make(map[value.ValueKind]map[string]*value.InternalFunctionObject)

	// Initialize maps for each type
	registry[value.StringKind] = make(map[string]*value.InternalFunctionObject)
	registry[value.RopeStringKind] = make(map[string]*value.InternalFunctionObject)
	registry[value.ArrayKind] = make(map[string]*value.InternalFunctionObject)
	registry[value.MapKind] = make(map[string]*value.InternalFunctionObject)
	registry[value.NumberKind] = make(map[string]*value.InternalFunctionObject)

	// Populate with built-in methods
	registerStringMethods(registry[value.StringKind])
	registerStringMethods(registry[value.RopeStringKind])
	registerArrayMethods(registry[value.ArrayKind])
	registerMapMethods(registry[value.MapKind])

	return registry
}

func registerStringMethods(methods map[string]*value.InternalFunctionObject) {
	// toUpperCase(str) -> string
	methods["toUpperCase"] = &value.InternalFunctionObject{
		Arity: 1, // just the string itself
		Call: func(args []value.Value) value.Value {
			if len(args) < 1 || !value.IsStringLike(args[0]) {
				return value.MakeError("toUpperCase requires string", "TypeError", 0, 0)
			}
			str := args[0].ToString()
			return value.MakeString(strings.ToUpper(str))
		},
	}

	// toLowerCase(str) -> string
	methods["toLowerCase"] = &value.InternalFunctionObject{
		Arity: 1,
		Call: func(args []value.Value) value.Value {
			if len(args) < 1 || !value.IsStringLike(args[0]) {
				return value.MakeError("toLowerCase requires string", "TypeError", 0, 0)
			}
			str := args[0].ToString()
			return value.MakeString(strings.ToLower(str))
		},
	}

	// split(str, separator) -> array
	methods["split"] = &value.InternalFunctionObject{
		Arity: 2, // str + separator
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || !value.IsStringLike(args[0]) || !value.IsStringLike(args[1]) {
				return value.MakeError("split(str, sep) requires two strings", "TypeError", 0, 0)
			}
			str := args[0].ToString()
			sep := args[1].ToString()
			parts := strings.Split(str, sep)

			elements := make([]value.Value, len(parts))
			for i, part := range parts {
				elements[i] = value.MakeString(part)
			}
			return value.MakeArray(elements)
		},
	}

	// charAt(str, index) -> string
	methods["charAt"] = &value.InternalFunctionObject{
		Arity: 2,
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || !value.IsStringLike(args[0]) || args[1].Kind != value.NumberKind {
				return value.MakeError("charAt(str, index) requires string and number", "TypeError", 0, 0)
			}
			str := args[0].ToString()
			idx := int(args[1].Num)

			if idx < 0 || idx >= len(str) {
				return value.MakeString("")
			}
			return value.MakeString(string(str[idx]))
		},
	}

	// includes(str, search) -> boolean
	methods["includes"] = &value.InternalFunctionObject{
		Arity: 2,
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || !value.IsStringLike(args[0]) || !value.IsStringLike(args[1]) {
				return value.MakeError("includes(str, search) requires two strings", "TypeError", 0, 0)
			}
			str := args[0].ToString()
			search := args[1].ToString()
			return value.MakeBool(strings.Contains(str, search))
		},
	}

	// indexOf(str, search) -> number
	methods["indexOf"] = &value.InternalFunctionObject{
		Arity: 2,
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || !value.IsStringLike(args[0]) || !value.IsStringLike(args[1]) {
				return value.MakeError("indexOf(str, search) requires two strings", "TypeError", 0, 0)
			}
			str := args[0].ToString()
			search := args[1].ToString()
			return value.MakeNumber(float64(strings.Index(str, search)))
		},
	}

	// trim(str) -> string
	methods["trim"] = &value.InternalFunctionObject{
		Arity: 1,
		Call: func(args []value.Value) value.Value {
			if len(args) < 1 || !value.IsStringLike(args[0]) {
				return value.MakeError("trim requires string", "TypeError", 0, 0)
			}
			str := args[0].ToString()
			return value.MakeString(strings.TrimSpace(str))
		},
	}

	// substring(str, start, end?) -> string
	methods["substring"] = &value.InternalFunctionObject{
		Arity: -1, // 2 or 3 args
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || !value.IsStringLike(args[0]) || args[1].Kind != value.NumberKind {
				return value.MakeError("substring requires string and number", "TypeError", 0, 0)
			}
			str := args[0].ToString()
			start := int(args[1].Num)
			end := len(str)

			if len(args) >= 3 && args[2].Kind == value.NumberKind {
				end = int(args[2].Num)
			}

			// Clamp values
			if start < 0 {
				start = 0
			}
			if end > len(str) {
				end = len(str)
			}
			if start > end {
				start = end
			}

			return value.MakeString(str[start:end])
		},
	}

	// replace(str, search, replacement) -> string
	methods["replace"] = &value.InternalFunctionObject{
		Arity: 3,
		Call: func(args []value.Value) value.Value {
			if len(args) < 3 || !value.IsStringLike(args[0]) || !value.IsStringLike(args[1]) || !value.IsStringLike(args[2]) {
				return value.MakeError("replace(str, search, replacement) requires three strings", "TypeError", 0, 0)
			}
			str := args[0].ToString()
			search := args[1].ToString()
			replacement := args[2].ToString()
			return value.MakeString(strings.Replace(str, search, replacement, 1))
		},
	}
}

func registerArrayMethods(methods map[string]*value.InternalFunctionObject) {
	// push(arr, ...items) -> number
	methods["push"] = &value.InternalFunctionObject{
		Arity: -1, // variadic
		Call: func(args []value.Value) value.Value {
			if len(args) < 1 || args[0].Kind != value.ArrayKind {
				return value.MakeError("push requires array", "TypeError", 0, 0)
			}
			arr := args[0].AsArray()

			// Add all remaining args to array
			for i := 1; i < len(args); i++ {
				arr.Elements = append(arr.Elements, args[i])
			}

			return value.MakeNumber(float64(len(arr.Elements)))
		},
	}

	// pop(arr) -> value
	methods["pop"] = &value.InternalFunctionObject{
		Arity: 1,
		Call: func(args []value.Value) value.Value {
			if len(args) < 1 || args[0].Kind != value.ArrayKind {
				return value.MakeError("pop requires array", "TypeError", 0, 0)
			}
			arr := args[0].AsArray()

			if len(arr.Elements) == 0 {
				return value.MakeNull()
			}

			lastIdx := len(arr.Elements) - 1
			val := arr.Elements[lastIdx]
			arr.Elements = arr.Elements[:lastIdx]
			return val
		},
	}

	// slice(arr, start, end?) -> array
	methods["slice"] = &value.InternalFunctionObject{
		Arity: -1, // 2 or 3 args
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || args[0].Kind != value.ArrayKind || args[1].Kind != value.NumberKind {
				return value.MakeError("slice requires array and number", "TypeError", 0, 0)
			}
			arr := args[0].AsArray()
			start := int(args[1].Num)
			end := len(arr.Elements)

			if len(args) >= 3 && args[2].Kind == value.NumberKind {
				end = int(args[2].Num)
			}

			// Handle negative indices
			if start < 0 {
				start = len(arr.Elements) + start
			}
			if end < 0 {
				end = len(arr.Elements) + end
			}

			// Clamp
			if start < 0 {
				start = 0
			}
			if end > len(arr.Elements) {
				end = len(arr.Elements)
			}
			if start > end {
				start = end
			}

			newElements := make([]value.Value, end-start)
			copy(newElements, arr.Elements[start:end])

			return value.MakeArray(newElements)
		},
	}

	// join(arr, separator) -> string
	methods["join"] = &value.InternalFunctionObject{
		Arity: 2,
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || args[0].Kind != value.ArrayKind {
				return value.MakeError("join requires array", "TypeError", 0, 0)
			}
			arr := args[0].AsArray()
			sep := ""
			if value.IsStringLike(args[1]) {
				sep = args[1].ToString()
			}

			parts := make([]string, len(arr.Elements))
			for i, elem := range arr.Elements {
				parts[i] = elem.ToString()
			}

			return value.MakeString(strings.Join(parts, sep))
		},
	}

	// reverse(arr) -> array
	methods["reverse"] = &value.InternalFunctionObject{
		Arity: 1,
		Call: func(args []value.Value) value.Value {
			if len(args) < 1 || args[0].Kind != value.ArrayKind {
				return value.MakeError("reverse requires array", "TypeError", 0, 0)
			}
			arr := args[0].AsArray()

			// Reverse in place
			for i, j := 0, len(arr.Elements)-1; i < j; i, j = i+1, j-1 {
				arr.Elements[i], arr.Elements[j] = arr.Elements[j], arr.Elements[i]
			}

			return args[0]
		},
	}

	// includes(arr, item) -> boolean
	methods["includes"] = &value.InternalFunctionObject{
		Arity: 2,
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || args[0].Kind != value.ArrayKind {
				return value.MakeError("includes requires array", "TypeError", 0, 0)
			}
			arr := args[0].AsArray()
			search := args[1]

			for _, elem := range arr.Elements {
				// Simple equality check
				switch {
				case elem.Kind == value.NumberKind && search.Kind == value.NumberKind && elem.Num == search.Num:
					return value.MakeBool(true)
				case value.IsStringLike(elem) && value.IsStringLike(search) && elem.ToString() == search.ToString():
					return value.MakeBool(true)
				case elem.Kind == value.BoolKind && search.Kind == value.BoolKind && elem.Bool == search.Bool:
					return value.MakeBool(true)
				case elem.Kind == value.NullKind && search.Kind == value.NullKind:
					return value.MakeBool(true)
				}
			}

			return value.MakeBool(false)
		},
	}

	// indexOf(arr, item) -> number
	methods["indexOf"] = &value.InternalFunctionObject{
		Arity: 2,
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || args[0].Kind != value.ArrayKind {
				return value.MakeError("indexOf requires array", "TypeError", 0, 0)
			}
			arr := args[0].AsArray()
			search := args[1]

			for i, elem := range arr.Elements {
				switch {
				case elem.Kind == value.NumberKind && search.Kind == value.NumberKind && elem.Num == search.Num:
					return value.MakeNumber(float64(i))
				case value.IsStringLike(elem) && value.IsStringLike(search) && elem.ToString() == search.ToString():
					return value.MakeNumber(float64(i))
				case elem.Kind == value.BoolKind && search.Kind == value.BoolKind && elem.Bool == search.Bool:
					return value.MakeNumber(float64(i))
				case elem.Kind == value.NullKind && search.Kind == value.NullKind:
					return value.MakeNumber(float64(i))
				}
			}

			return value.MakeNumber(-1)
		},
	}
}

func registerMapMethods(methods map[string]*value.InternalFunctionObject) {
	// keys(map) -> array
	methods["keys"] = &value.InternalFunctionObject{
		Arity: 1,
		Call: func(args []value.Value) value.Value {
			if len(args) < 1 || args[0].Kind != value.MapKind {
				return value.MakeError("keys requires map", "TypeError", 0, 0)
			}
			m := args[0].AsMap()

			keys := make([]value.Value, 0, len(m.Properties))
			for k := range m.Properties {
				keys = append(keys, value.MakeString(k))
			}

			return value.MakeArray(keys)
		},
	}

	// values(map) -> array
	methods["values"] = &value.InternalFunctionObject{
		Arity: 1,
		Call: func(args []value.Value) value.Value {
			if len(args) < 1 || args[0].Kind != value.MapKind {
				return value.MakeError("values requires map", "TypeError", 0, 0)
			}
			m := args[0].AsMap()

			values := make([]value.Value, 0, len(m.Properties))
			for _, v := range m.Properties {
				values = append(values, v)
			}

			return value.MakeArray(values)
		},
	}

	// hasKey(map, key) -> boolean
	methods["hasKey"] = &value.InternalFunctionObject{
		Arity: 2,
		Call: func(args []value.Value) value.Value {
			if len(args) < 2 || args[0].Kind != value.MapKind || !value.IsStringLike(args[1]) {
				return value.MakeError("hasKey(map, key) requires map and string", "TypeError", 0, 0)
			}
			m := args[0].AsMap()
			key := args[1].ToString()

			_, exists := m.Properties[key]
			return value.MakeBool(exists)
		},
	}
}

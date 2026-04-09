package tests

import (
	"Kairo/compiler"
	"Kairo/frontend"
	"Kairo/semantic"
	"Kairo/value"
	"Kairo/vm"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type scriptResult struct {
	Output       string
	Result       value.Value
	ParseDiags   []frontend.Diagnostic
	SemanticDiag []frontend.Diagnostic
	CompileDiag  []frontend.Diagnostic
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = orig
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stdout pipe: %v", err)
	}
	return string(data)
}

func runKairoSource(t *testing.T, sourceName string, source string) scriptResult {
	t.Helper()

	lexer := frontend.NewLexer()
	tokens, err := lexer.Tokenize(source)
	if err != nil {
		t.Fatalf("tokenize failed for %s: %v", sourceName, err)
	}

	parser := frontend.Parser{}
	program, parseDiags := parser.Parse(tokens)
	if len(parseDiags) > 0 {
		return scriptResult{ParseDiags: parseDiags}
	}

	semanticDiags := semantic.Analyze(program)
	if len(semanticDiags) > 0 {
		return scriptResult{SemanticDiag: semanticDiags}
	}

	slots := make(map[string]int)
	vm.EnsureBuiltinSlots(slots)

	comp := compiler.NewCompiler()
	comp.SetGlobalSlots(slots)
	chunk, compileDiags := comp.CompileWithDiagnostics(program)
	if len(compileDiags) > 0 {
		return scriptResult{CompileDiag: compileDiags}
	}

	mainFn := &vm.FunctionObject{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: comp.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm.ClosureObject{Function: mainFn, Upvalues: nil}

	globals := vm.NewGlobals(slots)
	machine := vm.NewVM(globals)
	machine.SetSourceName(sourceName)

	var result value.Value
	output := captureStdout(t, func() {
		result = machine.Run(mainClosure)
	})

	return scriptResult{Output: output, Result: result}
}

func TestBenchmarkFixturesRegressionSuite(t *testing.T) {
	testCases := []struct {
		file            string
		expectOutput    []string
		expectErrorKind bool
		errorContains   string
	}{
		{file: "closure_test.kr", expectOutput: []string{"4"}},
		{file: "compound_assignment_test.kr", expectOutput: []string{"After arr[2] *= 2:"}},
		{file: "comprehensive_types_test.kr", expectOutput: []string{"Hello, SlimScript!"}},
		{file: "error_details_test.kr", expectOutput: []string{"All tests completed!"}},
		{file: "error_propagation_test.kr", expectOutput: []string{"All error propagation tests completed!"}},
		{file: "finally_break_test.kr", expectOutput: []string{"final outerIdx:"}},
		{file: "finally_edge_case.kr", expectOutput: []string{"final breakFinallyTest: 3"}},
		{file: "inline_function.kr", expectOutput: []string{"4"}},
		{file: "loop_control_comprehensive_test.kr", expectOutput: []string{"LOOP CONTROL COMPREHENSIVE END"}},
		{file: "map_filter_reduce.kr", expectOutput: []string{"=== map/filter/reduce basic ===", "mapped: 2,4,6,8,10", "filtered: 2,4", "sum: 15", "original: 1,2,3,4,5"}},
		{file: "map_filter_reduce_chain.kr", expectOutput: []string{"=== map/filter/reduce chain ===", "chain: 24"}},
		{file: "map_filter_reduce_nested.kr", expectOutput: []string{"=== map/filter/reduce nested ===", "mapped: 3,6,9,12", "filtered: 9,12", "total: 21"}},
		{file: "map_filter_reduce_edge_cases.kr", expectOutput: []string{"=== map/filter/reduce edge cases ===", "empty map len: 0", "empty filter len: 0", "caught empty reduce", "single reduce: 42", "empty reduce with init: 10"}},
		{file: "map_filter_reduce_arity_error.kr", expectErrorKind: true, errorContains: "wrong number of arguments"},
		{file: "map_filter_reduce_callback_error.kr", expectOutput: []string{"=== map/filter/reduce callback error ==="}, expectErrorKind: true, errorContains: "ZeroDivisionError"},
		{file: "math_test.kr", expectOutput: []string{"Testing math module:"}},
		{file: "method_dispatch_test.kr", expectOutput: []string{"=== Tests Complete ==="}},
		{file: "short_circuit_test.kr", expectOutput: []string{"Test 4 - false || increment():"}},
		{file: "simple_error_test.kr", expectOutput: []string{"Done"}},
		{file: "simple_method_test.kr", expectOutput: []string{"Testing basic method dispatch:"}},
		{file: "simple_test.kr", expectOutput: []string{"15"}},
		{file: "stack_trace.kr", expectErrorKind: true, errorContains: "ZeroDivisionError"},
		{file: "test.kr", expectOutput: []string{"hello world"}},
		{file: "try_catch_comprehensive_test.kr", expectOutput: []string{"All tests completed successfully!"}},
		{file: "try_catch_test.kr", expectOutput: []string{"After try-catch-finally"}},
		{file: "type_test.kr", expectOutput: []string{"Alice"}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			path := filepath.Join("..", "benchmark", tc.file)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed reading fixture %s: %v", tc.file, err)
			}

			res := runKairoSource(t, tc.file, string(data))

			if len(res.ParseDiags) > 0 {
				t.Fatalf("unexpected parse diagnostics for %s: %+v", tc.file, res.ParseDiags)
			}
			if len(res.SemanticDiag) > 0 {
				t.Fatalf("unexpected semantic diagnostics for %s: %+v", tc.file, res.SemanticDiag)
			}
			if len(res.CompileDiag) > 0 {
				t.Fatalf("unexpected compile diagnostics for %s: %+v", tc.file, res.CompileDiag)
			}

			if tc.expectErrorKind {
				if res.Result.Kind != value.ErrorKind {
					t.Fatalf("expected runtime error result for %s, got kind=%v output=%q", tc.file, res.Result.Kind, res.Output)
				}
				if tc.errorContains != "" && !strings.Contains(res.Result.ToString(), tc.errorContains) {
					t.Fatalf("expected runtime error for %s to contain %q, got %q", tc.file, tc.errorContains, res.Result.ToString())
				}
				return
			}

			if res.Result.Kind == value.ErrorKind {
				t.Fatalf("unexpected runtime error for %s: %s", tc.file, res.Result.ToString())
			}

			for _, want := range tc.expectOutput {
				if !strings.Contains(res.Output, want) {
					t.Fatalf("output for %s missing %q\nactual output:\n%s", tc.file, want, res.Output)
				}
			}
		})
	}
}

func TestSwitchArrowAndBlockSyntax(t *testing.T) {
	source := `
var x = 2;
var y = 10;

switch x {
  case 1 => print("one")
  case 2 => print("two")
  default => print("other")
}

switch y {
  case 5 {
    print("five")
    print("again")
  }
  case 10 {
    print("ten")
    print("ten-again")
  }
  default {
    print("fallback")
  }
}
`
	res := runKairoSource(t, "switch_arrow_block", source)
	if len(res.ParseDiags) > 0 {
		t.Fatalf("unexpected parse diagnostics: %+v", res.ParseDiags)
	}
	if len(res.SemanticDiag) > 0 {
		t.Fatalf("unexpected semantic diagnostics: %+v", res.SemanticDiag)
	}
	if len(res.CompileDiag) > 0 {
		t.Fatalf("unexpected compile diagnostics: %+v", res.CompileDiag)
	}
	if res.Result.Kind == value.ErrorKind {
		t.Fatalf("unexpected runtime error: %s", res.Result.ToString())
	}
	for _, want := range []string{"two", "ten", "ten-again"} {
		if !strings.Contains(res.Output, want) {
			t.Fatalf("switch output missing %q\nactual output:\n%s", want, res.Output)
		}
	}
}

func TestSwitchNoFallthrough(t *testing.T) {
	source := `
var x = 1;
switch x {
  case 1 => print("first")
  case 1 => print("second")
  default => print("default")
}
`
	res := runKairoSource(t, "switch_no_fallthrough", source)
	if len(res.ParseDiags) > 0 {
		t.Fatalf("unexpected parse diagnostics: %+v", res.ParseDiags)
	}
	if len(res.SemanticDiag) > 0 {
		t.Fatalf("unexpected semantic diagnostics: %+v", res.SemanticDiag)
	}
	if len(res.CompileDiag) > 0 {
		t.Fatalf("unexpected compile diagnostics: %+v", res.CompileDiag)
	}
	if strings.Contains(res.Output, "second") {
		t.Fatalf("switch should not fall through, got output:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "first") {
		t.Fatalf("switch output missing first case marker:\n%s", res.Output)
	}
}

func TestSwitchDuplicateDefaultYieldsParseDiagnostic(t *testing.T) {
	source := `
var x = 1;
switch x {
  default => print("a")
  default => print("b")
}
`
	lexer := frontend.NewLexer()
	tokens, err := lexer.Tokenize(source)
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}
	parser := frontend.Parser{}
	_, diags := parser.Parse(tokens)
	if len(diags) == 0 {
		t.Fatalf("expected parse diagnostics for duplicate default")
	}
	if diags[0].Phase != "parse" {
		t.Fatalf("expected parse phase diagnostic, got %+v", diags[0])
	}
}

func TestSwitchMissingClauseBodyYieldsParseDiagnostic(t *testing.T) {
	source := `
var x = 1;
switch x {
  case 1
}
`
	lexer := frontend.NewLexer()
	tokens, err := lexer.Tokenize(source)
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}
	parser := frontend.Parser{}
	_, diags := parser.Parse(tokens)
	if len(diags) == 0 {
		t.Fatalf("expected parse diagnostics for malformed case clause")
	}
}

func TestArrayMapFilterReduceMethods(t *testing.T) {
	source := `
var arr = [1, 2, 3];
var mapped = arr.map(fn(x:number): number => x * 2);
print(mapped.join(","));

var filtered = arr.filter(fn(x:number): bool => x > 1);
print(filtered.join(","));

var reduced = arr.reduce(fn(acc:number, x:number): number => acc + x, 0);
print(reduced);
`

	res := runKairoSource(t, "array_map_filter_reduce", source)
	if len(res.ParseDiags) > 0 {
		t.Fatalf("unexpected parse diagnostics: %+v", res.ParseDiags)
	}
	if len(res.SemanticDiag) > 0 {
		t.Fatalf("unexpected semantic diagnostics: %+v", res.SemanticDiag)
	}
	if len(res.CompileDiag) > 0 {
		t.Fatalf("unexpected compile diagnostics: %+v", res.CompileDiag)
	}
	if res.Result.Kind == value.ErrorKind {
		t.Fatalf("unexpected runtime error: %s", res.Result.ToString())
	}

	for _, want := range []string{"2,4,6", "2,3", "6"} {
		if !strings.Contains(res.Output, want) {
			t.Fatalf("output missing %q\nactual output:\n%s", want, res.Output)
		}
	}
}

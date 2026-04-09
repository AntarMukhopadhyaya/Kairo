package vm

import (
	"Kairo/frontend"
	"Kairo/value"
	"strings"
	"testing"
)

func TestRunReturnsErrorWithStackTrace(t *testing.T) {
	source := `fn a() {
    return b();
}

fn b() {
    return c();
}

fn c() {
    return 1 / 0;
}

a();
`

	lexer := frontend.NewLexer()
	tokens, err := lexer.Tokenize(source)
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	parser := frontend.Parser{}
	program := parser.GenerateAST(tokens)

	comp := NewCompiler()
	slots := comp.GlobalSlots()
	EnsureBuiltinSlots(slots)
	chunk := comp.Compile(program)

	globals := NewGlobals(slots)
	mainFn := &FunctionObject{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: comp.MaxRegUsed,
	}
	mainClosure := &ClosureObject{Function: mainFn, Upvalues: nil}

	machine := NewVM(globals)
	machine.SetSourceName("stack_trace.kr")
	result := machine.Run(mainClosure)

	if result.Kind != value.ErrorKind {
		t.Fatalf("expected error result, got kind=%v", result.Kind)
	}

	errObj := result.AsError()
	if errObj.ErrorType != "ZeroDivisionError" {
		t.Fatalf("expected ZeroDivisionError, got %s", errObj.ErrorType)
	}
	if len(errObj.StackTrace) < 4 {
		t.Fatalf("expected at least 4 frames, got %d: %v", len(errObj.StackTrace), errObj.StackTrace)
	}

	joined := strings.Join(errObj.StackTrace, "\n")
	for _, want := range []string{
		"at c (stack_trace.kr:",
		"at b (stack_trace.kr:",
		"at a (stack_trace.kr:",
		"at main (stack_trace.kr:",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("stack trace missing %q in:\n%s", want, joined)
		}
	}
}

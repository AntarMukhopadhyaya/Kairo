package main

import (
	"Kairo/value"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe create failed: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = orig
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe failed: %v", err)
	}
	return string(data)
}

func TestPrintVMResultTracebackFormatting(t *testing.T) {
	errVal := value.MakeError("Division by zero", "ZeroDivisionError", 9, 14)
	errObj := errVal.AsError()
	errObj.StackTrace = []string{
		"at c (stack_trace.kr:9:14)",
		"at b (stack_trace.kr:6:12)",
		"at a (stack_trace.kr:3:12)",
	}

	out := captureStdout(t, func() {
		printVMResult(errVal)
	})

	wantLines := []string{
		"Traceback (most recent call last):",
		"  at c (stack_trace.kr:9:14)",
		"  at b (stack_trace.kr:6:12)",
		"  at a (stack_trace.kr:3:12)",
		"ZeroDivisionError: Division by zero",
	}

	for _, want := range wantLines {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nactual:\n%s", want, out)
		}
	}
}

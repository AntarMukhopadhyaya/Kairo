package benchmark

import (
	"Kairo/compiler"
	"Kairo/frontend"
	"Kairo/vm"
	"os"
	"testing"
)

var benchmarkOptimEnabled = os.Getenv("KAIRO_OPT") == "1"

func compileMainClosure(source string) (*vm.ClosureObject, map[string]int) {
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	comp := compiler.NewCompiler()
	comp.EnableOptimizations(benchmarkOptimEnabled)
	slots := comp.GlobalSlots()
	vm.EnsureBuiltinSlots(slots)
	chunk := comp.Compile(program)

	mainFn := &vm.FunctionObject{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: comp.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}

	return &vm.ClosureObject{Function: mainFn, Upvalues: nil}, slots
}

func runBenchmarkSource(b *testing.B, source string) {
	mainClosure, slots := compileMainClosure(source)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		machine := vm.NewVM(globals)
		machine.Run(mainClosure)
	}
}

func BenchmarkVMSimpleLoop(b *testing.B) {
	source := `
        var x: number = 0;
        var i: number = 0;
        while (i < 100000) {
            x = x + 1;
            i = i + 1;
        }
    `
	runBenchmarkSource(b, source)
}

func BenchmarkVMSimpleArithmetic(b *testing.B) {
	source := `
        var x: number = 5;
        var y: number = 10;
        var z: number = 0;
        var i: number = 0;
        while (i < 50000) {
            z = x + y;
            z = z * 2;
            z = z - x;
            i = i + 1;
        }
    `
	runBenchmarkSource(b, source)
}

func BenchmarkVMNestedLoops(b *testing.B) {
	source := `
        var result: number = 0;
        var i: number = 0;
        while (i < 10000) {
            var j: number = 2;
            var sum: number = 0;
            while (j < 20) {
                sum = sum + j * i;
                j = j + 1;
            }
            result = result + sum;
            i = i + 1;
        }
    `
	runBenchmarkSource(b, source)
}

func BenchmarkVMArrayIndexing(b *testing.B) {
	source := `
        var arr: array = [10, 23, 45, 70, 11, 15, 36, 89, 52, 47, 33, 28, 91, 14, 67];
        var z: number = 0;
        var i: number = 0;
        while (i < 20000) {
            var idx: number = 0;
            while (idx < 15) {
                z = z + arr[idx];
                idx = idx + 1;
            }
            i = i + 1;
        }
    `
	runBenchmarkSource(b, source)
}

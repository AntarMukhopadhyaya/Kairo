package benchmark

import (
	"Kairo/frontend"
	runtimeinterp "Kairo/runtime"
	runtimebuiltins "Kairo/runtime/builtins"
	"Kairo/vm"
	"Kairo/vm2"
	"os"
	"testing"
)

var registerVMOptimEnabled = os.Getenv("KAIRO_OPT") == "1"

func newRegisterCompiler() *vm2.Compiler {
	compiler := vm2.NewCompiler()
	compiler.EnableOptimizations(registerVMOptimEnabled)
	vm2.EnsureBuiltinSlots(compiler.GlobalSlots())
	return compiler
}

func BenchmarkStackVMSimpleLoop(b *testing.B) {
	source := `
        var x: number = 0;
        var i: number = 0;
        while (i < 100000) {
            x = x + 1;
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMSimpleLoop(b *testing.B) {
	source := `
        var x: number = 0;
        var i: number = 0;
        while (i < 100000) {
            x = x + 1;
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	// Create main closure
	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

func BenchmarkInterpreterSimpleLoop(b *testing.B) {
	source := `
        var x: number = 0;
        var i: number = 0;
        while (i < 100000) {
            x = x + 1;
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := runtimeinterp.CreateGlobalEnvironment()
		runtimebuiltins.RegisterBuiltins(env)
		runtimeinterp.Evaluate(program, env)
	}
}

func BenchmarkStackVMSimpleArithmetic(b *testing.B) {
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
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMSimpleArithmetic(b *testing.B) {
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
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	// Create main closure
	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

func BenchmarkStackVMPrimes(b *testing.B) {
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
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMPrimes(b *testing.B) {
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
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	// Create main closure
	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

func BenchmarkStackVMBubbleSort(b *testing.B) {
	source := `
        var arr: array = [64, 34, 25, 12, 22, 11, 90, 88, 45, 50, 23, 36, 18, 77, 15];
        var result: number = 0;
        var iterations: number = 0;

        while (iterations < 500) {
            var i: number = 0;
            while (i < 14) {
                var j: number = i + 1;
                while (j < 15) {
                    result = result + arr[i] * arr[j];
                    j = j + 1;
                }
                i = i + 1;
            }
            iterations = iterations + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMBubbleSort(b *testing.B) {
	source := `
        var arr:array = [64, 34, 25, 12, 22, 11, 90, 88, 45, 50, 23, 36, 18, 77, 15];
        var result: number = 0;
        var iterations: number = 0;

        while (iterations < 500) {
            var i: number = 0;
            while (i < 14) {
                var j: number = i + 1;
                while (j < 15) {
                    result = result + arr[i] * arr[j];
                    j = j + 1;
                }
                i = i + 1;
            }
            iterations = iterations + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	// Create main closure
	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

func BenchmarkStackVMLinearSearch(b *testing.B) {
	source := `
        var arr: array = [10, 23, 45, 70, 11, 15, 36, 89, 52, 47, 33, 28, 91, 14, 67];
        var result: number = 0;
        var n: number = 15;

        var iterations: number = 0;
        while (iterations < 5000) {
            var i: number = 0;
            while (i < n) {
                result = result + arr[i];
                i = i + 1;
            }
            iterations = iterations + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMLinearSearch(b *testing.B) {
	source := `
        var arr: array = [10, 23, 45, 70, 11, 15, 36, 89, 52, 47, 33, 28, 91, 14, 67];
        var result: number = 0;
        var n: number = 15;

        var iterations: number = 0;
        while (iterations < 5000) {
            var i: number = 0;
            while (i < n) {
                result = result + arr[i];
                i = i + 1;
            }
            iterations = iterations + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	// Create main closure
	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

func BenchmarkStackVMFibonacci(b *testing.B) {
	source := `
        var a: number = 0;
        var b: number = 1;
        var iterations: number = 0;

        while (iterations < 50000) {
            var temp: number = a + b;
            a = b;
            b = temp;
            iterations = iterations + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMFibonacci(b *testing.B) {
	source := `
        var a: number = 0;
        var b: number = 1;
        var iterations: number = 0;

        while (iterations < 50000) {
            var temp: number = a + b;
            a = b;
            b = temp;
            iterations = iterations + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	// Create main closure
	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Deep Expression Trees - tests stack stress with complex arithmetic
func BenchmarkStackVMDeepExpressionTree(b *testing.B) {
	source := `
        var x: number = 0;
        var a: number = 1;
        var b: number = 2;
        var c: number = 3;
        var d: number = 4;
        var e: number = 5;
        var f: number = 6;
        var g: number = 7;
        var h: number = 8;
        var i: number = 0;
        while (i < 50000) {
            x = (x + 1) * 2 - 3 + 4 * 5 - 6 + 7;
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMDeepExpressionTree(b *testing.B) {
	source := `
        var x: number = 0;
        var a: number = 1;
        var b: number = 2;
        var c: number = 3;
        var d: number = 4;
        var e: number = 5;
        var f: number = 6;
        var g: number = 7;
        var h: number = 8;
        var i: number = 0;
        while (i < 50000) {
            x = (x + 1) * 2 - 3 + 4 * 5 - 6 + 7;
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Heavy Temporary Usage - tests register allocation with many intermediate vars
func BenchmarkStackVMHeavyTemporaryUsage(b *testing.B) {
	source := `
        var i: number = 0;
        var x: number = 1;
        while (i < 50000) {
            var a: number = x + 1;
            var b: number = a + 2;
            var c: number = b + 3;
            var d: number = c + 4;
            var e: number = d + 5;
            x = e;
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMHeavyTemporaryUsage(b *testing.B) {
	source := `
        var i: number = 0;
        var x: number = 1;
        while (i < 50000) {
            var a: number = x + 1;
            var b: number = a + 2;
            var c: number = b + 3;
            var d: number = c + 4;
            var e: number = d + 5;
            x = e;
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Function Call Intensive - tests argument passing and frame setup
func BenchmarkStackVMFunctionCallIntensive(b *testing.B) {
	source := `
        fn add(a: number, b: number) {
            return a + b;
        }

        var i: number = 0;
        var x: number = 0;
        while (i < 10000) {
            x = add(x, i);
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMFunctionCallIntensive(b *testing.B) {
	source := `
        fn add(a: number, b: number) {
            return a + b;
        }

        var i: number = 0;
        var x: number = 0;
        while (i < 10000) {
            x = add(x, i);
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Recursion Benchmark - tests frame creation and recursion overhead
func BenchmarkStackVMRecursion(b *testing.B) {
	source := `
        fn fib(n: number) {
            if (n < 2) {
                return n;
            }
            return fib(n - 1) + fib(n - 2);
        }

        var i: number = 0;
        var result: number = 0;
        while (i < 100) {
            result = fib(15);
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMRecursion(b *testing.B) {
	source := `
        fn fib(n: number) {
            if (n < 2) {
                return n;
            }
            return fib(n - 1) + fib(n - 2);
        }

        var i: number = 0;
        var result: number = 0;
        while (i < 100) {
            result = fib(15);
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

func BenchmarkInterpreterRecursion(b *testing.B) {
	source := `
		fn fib(n: number) {
			if (n < 2) {
				return n;
			}
			return fib(n - 1) + fib(n - 2);
		}

		var i: number = 0;
		var result: number = 0;
		while (i < 100) {
			result = fib(15);
			i = i + 1;
		}
	`
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := runtimeinterp.CreateGlobalEnvironment()
		runtimebuiltins.RegisterBuiltins(env)
		runtimeinterp.Evaluate(program, env)
	}
}

// Type Conversions - tests type coercion and comparison overhead
func BenchmarkStackVMTypeConversions(b *testing.B) {
	source := `
        var i: number = 0;
        var count: number = 0;
        var x: number = 0;
        var y: number = 0;
        while (i < 100000) {
            x = 5;
            y = 10;
            if (x < y) {
                count = count + 1;
            }
            if (x == y) {
                count = count - 1;
            }
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMTypeConversions(b *testing.B) {
	source := `
        var i: number = 0;
        var count: number = 0;
        var x: number = 0;
        var y: number = 0;
        while (i < 100000) {
            x = 5;
            y = 10;
            if (x < y) {
                count = count + 1;
            }
            if (x == y) {
                count = count - 1;
            }
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Closure Capture - tests upvalue capture + get/set overhead
func BenchmarkStackVMClosureCapture(b *testing.B) {
	source := `
        fn makeCounter() {
            var x: number = 0;
            fn inc() {
                x = x + 1;
                return x;
            }
            return inc;
        }

        var c = makeCounter();
        var i: number = 0;
        var sum: number = 0;
        while (i < 50000) {
            sum = sum + c();
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMClosureCapture(b *testing.B) {
	source := `
        fn makeCounter() {
            var x: number = 0;
            fn inc() {
                x = x + 1;
                return x;
            }
            return inc;
        }

        var c = makeCounter();
        var i: number = 0;
        var sum: number = 0;
        while (i < 50000) {
            sum = sum + c();
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Nested Closures - tests upvalue chaining (local + non-local)
func BenchmarkStackVMNestedClosures(b *testing.B) {
	source := `
        fn outer() {
            var x: number = 1;
            fn middle() {
                var y: number = 2;
                fn inner() {
                    x = x + 1;
                    y = y + 1;
                    return x + y;
                }
                return inner;
            }
            return middle();
        }

        var f = outer();
        var i: number = 0;
        var sum: number = 0;
        while (i < 30000) {
            sum = sum + f();
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMNestedClosures(b *testing.B) {
	source := `
        fn outer() {
            var x: number = 1;
            fn middle() {
                var y: number = 2;
                fn inner() {
                    x = x + 1;
                    y = y + 1;
                    return x + y;
                }
                return inner;
            }
            return middle();
        }

        var f = outer();
        var i: number = 0;
        var sum: number = 0;
        while (i < 30000) {
            sum = sum + f();
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Array Intensive - stresses indexing + auto-grow
func BenchmarkStackVMArrayIntensive(b *testing.B) {
	source := `
		var arr: array = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
		var i: number = 0;
		var sum: number = 0;
		while (i < 200000) {
			var idx: number = i % 64;
			arr[idx] = arr[idx] + 1;
			sum = sum + arr[idx];
			i = i + 1;
		}
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMArrayIntensive(b *testing.B) {
	source := `
		var arr: array = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
		var i: number = 0;
		var sum: number = 0;
		while (i < 200000) {
			var idx: number = i % 64;
			arr[idx] = arr[idx] + 1;
			sum = sum + arr[idx];
			i = i + 1;
		}
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Map Intensive - stresses dynamic string keys + index set/get
func BenchmarkStackVMMapIntensive(b *testing.B) {
	source := `
        var m = {};
        var i: number = 0;
        while (i < 20000) {
            var k = "k" + i;
            m[k] = i;
            i = i + 1;
        }

        var j: number = 0;
        var sum: number = 0;
        while (j < 20000) {
            var k2 = "k" + j;
            sum = sum + m[k2];
            j = j + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMMapIntensive(b *testing.B) {
	source := `
        var m = {};
        var i: number = 0;
        while (i < 20000) {
            var k = "k" + i;
            m[k] = i;
            i = i + 1;
        }

        var j: number = 0;
        var sum: number = 0;
        while (j < 20000) {
            var k2 = "k" + j;
            sum = sum + m[k2];
            j = j + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Boolean Heavy - stresses branching and comparisons
func BenchmarkStackVMBooleanHeavy(b *testing.B) {
	source := `
        var i: number = 0;
        var t: number = 0;
        var f: number = 0;
        while (i < 100000) {
            if ((i % 2) == 0) {
                t = t + 1;
            } else {
                f = f + 1;
            }
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMBooleanHeavy(b *testing.B) {
	source := `
        var i: number = 0;
        var t: number = 0;
        var f: number = 0;
        while (i < 100000) {
            if ((i % 2) == 0) {
                t = t + 1;
            } else {
                f = f + 1;
            }
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Object Property Access - stresses map property get/set via dot
func BenchmarkStackVMObjectPropertyAccess(b *testing.B) {
	source := `
        var o = {a: 1, b: 2, c: 3, d: 4};
        var i: number = 0;
        var sum: number = 0;
        while (i < 80000) {
            sum = sum + o.a + o.b + o.c + o.d;
            o.a = o.a + 1;
            o.b = o.b + 1;
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMObjectPropertyAccess(b *testing.B) {
	source := `
        var o = {a: 1, b: 2, c: 3, d: 4};
        var i: number = 0;
        var sum: number = 0;
        while (i < 80000) {
            sum = sum + o.a + o.b + o.c + o.d;
            o.a = o.a + 1;
            o.b = o.b + 1;
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// String Concat - stresses string allocations
func BenchmarkStackVMStringConcat(b *testing.B) {
	source := `
        var i: number = 0;
        var s = "";
        while (i < 30000) {
            s = s + "a";
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMStringConcat(b *testing.B) {
	source := `
        var i: number = 0;
        var s = "";
        while (i < 30000) {
            s = s + "a";
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Tail Recursion - call-heavy with predictable work
func BenchmarkStackVMTailRecursion(b *testing.B) {
	source := `
        fn fact(n: number, acc: number) {
            if (n == 0) {
                return acc;
            }
            return fact(n - 1, acc * n);
        }

        var i: number = 0;
        var r: number = 0;
        while (i < 2000) {
            r = fact(12, 1);
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMTailRecursion(b *testing.B) {
	source := `
        fn fact(n: number, acc: number) {
            if (n == 0) {
                return acc;
            }
            return fact(n - 1, acc * n);
        }

        var i: number = 0;
        var r: number = 0;
        while (i < 2000) {
            r = fact(12, 1);
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Mixed Workload - combines calls, arrays, maps, branching
func BenchmarkStackVMMixedWorkload(b *testing.B) {
	source := `
        fn step(x: number) {
            return x * 3 + 1;
        }

		var arr: array = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
        var m = {};
        var i: number = 0;
        var x: number = 1;

        while (i < 20000) {
            x = step(x);
			var idx: number = i % 64;
			arr[idx] = x;
            m["v" + i] = x;
            if ((i % 3) == 0) {
                x = x + m["v" + i];
            }
            i = i + 1;
        }

        var j: number = 0;
        var sum: number = 0;
        while (j < 20000) {
			var idx2: number = j % 64;
			sum = sum + arr[idx2];
            j = j + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMMixedWorkload(b *testing.B) {
	source := `
        fn step(x: number) {
            return x * 3 + 1;
        }

		var arr: array = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
        var m = {};
        var i: number = 0;
        var x: number = 1;

        while (i < 20000) {
            x = step(x);
			var idx: number = i % 64;
			arr[idx] = x;
            m["v" + i] = x;
            if ((i % 3) == 0) {
                x = x + m["v" + i];
            }
            i = i + 1;
        }

        var j: number = 0;
        var sum: number = 0;
        while (j < 20000) {
			var idx2: number = j % 64;
			sum = sum + arr[idx2];
            j = j + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

// Dispatch Micro - stresses call dispatch with small closures
func BenchmarkStackVMDispatchMicro(b *testing.B) {
	source := `
        fn f1(x: number) { return x + 1; }
        fn f2(x: number) { return x + 2; }
        fn f3(x: number) { return x + 3; }

        var i: number = 0;
        var sum: number = 0;
        while (i < 60000) {
            if ((i % 3) == 0) {
                sum = sum + f1(i);
            } else {
                if ((i % 3) == 1) {
                    sum = sum + f2(i);
                } else {
                    sum = sum + f3(i);
                }
            }
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := vm.NewCompiler()
	vm.EnsureBuiltinSlots(compiler.GlobalSlots())
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm.NewGlobals(slots)
		vmachine := vm.NewVM(chunk, globals)
		vmachine.Run()
	}
}

func BenchmarkRegisterVMDispatchMicro(b *testing.B) {
	source := `
        fn f1(x: number) { return x + 1; }
        fn f2(x: number) { return x + 2; }
        fn f3(x: number) { return x + 3; }

        var i: number = 0;
        var sum: number = 0;
        while (i < 60000) {
            if ((i % 3) == 0) {
                sum = sum + f1(i);
            } else {
                if ((i % 3) == 1) {
                    sum = sum + f2(i);
                } else {
                    sum = sum + f3(i);
                }
            }
            i = i + 1;
        }
    `
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}
	tokens, _ := lexer.Tokenize(source)
	program := parser.GenerateAST(tokens)

	compiler := newRegisterCompiler()
	chunk := compiler.Compile(program)
	slots := compiler.GlobalSlots()

	mainFn := &vm2.Function{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: compiler.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm2.Closure{
		Function: mainFn,
		Upvalues: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globals := vm2.NewGlobals(slots)
		vmachine := vm2.NewVM(globals)
		vmachine.Run(mainClosure)
	}
}

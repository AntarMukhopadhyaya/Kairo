package main

import (
	"Kairo/compiler"
	"Kairo/frontend"
	"Kairo/value"
	"Kairo/vm"

	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ANSI color codes for styling
const (
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	reset  = "\033[0m"
	red    = "\033[31m"
)

func printDiagnostics(diags []frontend.Diagnostic) {
	for _, d := range diags {
		loc := ""
		if d.Line > 0 {
			loc = fmt.Sprintf(" at %d:%d", d.Line, d.Column)
		}
		phase := d.Phase
		if phase == "" {
			phase = "unknown"
		}
		fmt.Printf(red+"[%s]%s %s\n"+reset, phase, loc, d.Message)
	}
}

// Function to execute SlimScript source code
func startREPL(optimize bool, profile bool) {
	fmt.Println(green + "Welcome to Kairo REPL! Type 'exit' to quit." + reset)

	scanner := bufio.NewScanner(os.Stdin)

	// 🔥 Create environment once and reuse it

	slots := make(map[string]int)
	vm.EnsureBuiltinSlots(slots)
	globals := vm.NewGlobals(slots)

	for {
		fmt.Print(blue + "Kairo> " + reset)
		scanner.Scan()
		line := scanner.Text()

		// Exit condition
		if strings.TrimSpace(line) == "exit" {
			fmt.Println(yellow + "Exiting Kairo REPL..." + reset)
			break
		}

		// Execute input with the same environment
		globals = executeSourceCodeWithEnv(line, globals, slots, optimize, profile)
	}
}

func executeSourceCodeWithEnv(src string, globals []vm.VariableInfo, slots map[string]int, optimize bool, profile bool) []vm.VariableInfo {
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}

	tokens, err := lexer.Tokenize(src)
	if err != nil {
		fmt.Println(yellow, "Error during tokenization:", err, reset)
		return globals
	}

	program, parseDiags := parser.Parse(tokens)
	if len(parseDiags) > 0 {
		printDiagnostics(parseDiags)
		return globals
	}

	// 🔥 Use the same environment across executions
	comp := compiler.NewCompiler()
	comp.SetGlobalSlots(slots)
	comp.EnableOptimizations(optimize)
	vm.EnsureBuiltinSlots(slots)
	chunk, compileDiags := comp.CompileWithDiagnostics(program)
	if len(compileDiags) > 0 {
		printDiagnostics(compileDiags)
		return globals
	}

	globals = vm.EnsureGlobalsSize(globals, slots)

	// Create main closure for register-based VM
	mainFn := &vm.FunctionObject{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: comp.MaxRegUsed,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm.ClosureObject{
		Function: mainFn,
		Upvalues: nil,
	}

	machine := vm.NewVM(globals)
	machine.SetSourceName("<repl>")
	machine.EnableInstructionProfiling(profile)
	result := machine.Run(mainClosure)
	printVMResult(result)
	if profile {
		p := machine.InstructionProfiler()
		if p != nil {
			fmt.Printf(yellow+"Executed %d instructions\n"+reset, p.Total())
		}
	}
	return globals
}

// Function to read and execute a file
func executeFile(filename string, optimize bool, profile bool) {
	slots := make(map[string]int)
	vm.EnsureBuiltinSlots(slots)
	globals := vm.NewGlobals(slots)
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println(yellow, "Error reading file:", err, reset)
		return
	}

	fmt.Println(blue+"Executing file:", filename+reset)
	lexer := frontend.NewLexer()
	parser := frontend.Parser{}

	tokens, err := lexer.Tokenize(string(data))
	if err != nil {
		fmt.Println(yellow, "Error during tokenization:", err, reset)
		return
	}
	program, parseDiags := parser.Parse(tokens)
	if len(parseDiags) > 0 {
		printDiagnostics(parseDiags)
		return
	}

	comp := compiler.NewCompiler()
	comp.SetGlobalSlots(slots)
	comp.EnableOptimizations(optimize)
	vm.EnsureBuiltinSlots(slots)
	chunk, compileDiags := comp.CompileWithDiagnostics(program)
	if len(compileDiags) > 0 {
		printDiagnostics(compileDiags)
		return
	}

	globals = vm.EnsureGlobalsSize(globals, slots)

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

	machine := vm.NewVM(globals)
	machine.SetSourceName(filepath.Base(filename))
	machine.EnableInstructionProfiling(profile)
	result := machine.Run(mainClosure)
	printVMResult(result)

	if profile {
		p := machine.InstructionProfiler()
		if p != nil {
			fmt.Printf(yellow+"\nExecuted %d instructions\n"+reset, p.Total())
			entries := p.EntriesSortedDesc()
			max := 12
			if len(entries) < max {
				max = len(entries)
			}
			fmt.Println(yellow + "Top opcodes:" + reset)
			for i := 0; i < max; i++ {
				e := entries[i]
				fmt.Printf("  %s: %d\n", e.Name, e.Count)
			}
		}
	}
}

// Compile source file to bytecode
func compileFile(inputFile string, outputFile string, optimize bool) {
	fmt.Println(blue + "Compiling: " + inputFile + reset)

	// Read source file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Println(red+"Error reading file:", err, reset)
		return
	}

	// Parse
	lexer := frontend.NewLexer()
	tokens, err := lexer.Tokenize(string(data))
	if err != nil {
		fmt.Println(red+"Tokenization error:", err, reset)
		return
	}

	parser := frontend.Parser{}
	program, parseDiags := parser.Parse(tokens)
	if len(parseDiags) > 0 {
		printDiagnostics(parseDiags)
		return
	}

	// Compile
	slots := make(map[string]int)
	vm.EnsureBuiltinSlots(slots)

	comp := compiler.NewCompiler()
	comp.SetGlobalSlots(slots)
	comp.EnableOptimizations(optimize)
	chunk, compileDiags := comp.CompileWithDiagnostics(program)
	if len(compileDiags) > 0 {
		printDiagnostics(compileDiags)
		return
	}

	// Determine output filename
	if outputFile == "" {
		ext := filepath.Ext(inputFile)
		outputFile = strings.TrimSuffix(inputFile, ext) + ".kbc"
	}

	// Write bytecode
	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Println(red+"Error creating output file:", err, reset)
		return
	}
	defer outFile.Close()

	writer := vm.NewBytecodeWriter(outFile)
	if err := writer.WriteChunk(chunk, comp.MaxRegUsed, slots); err != nil {
		fmt.Println(red+"Error writing bytecode:", err, reset)
		return
	}

	fmt.Println(green+"✓ Compiled successfully to:", outputFile, reset)
	fmt.Printf(yellow+"  Instructions: %d\n", len(chunk.Code))
	fmt.Printf("  Constants: %d\n", len(chunk.Constants))
	fmt.Printf("  Max Registers: %d%s\n", comp.MaxRegUsed, reset)
	if optimize {
		if stats, ok := comp.OptimizationStats(); ok {
			fmt.Println(yellow + "Optimization stats:" + reset)
			fmt.Printf("  Constant folded: %d\n", stats.ConstantFolded)
			fmt.Printf("  Dead-code removed: %d\n", stats.DeadCodeRemoved)
			fmt.Printf("  Peephole rewrites: %d\n", stats.PeepholeRewrites)
		}
	}
}

// Execute bytecode file
func executeBytecode(filename string, profile bool) {
	fmt.Println(blue+"Loading bytecode:", filename, reset)

	// Open bytecode file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(red+"Error opening bytecode file:", err, reset)
		return
	}
	defer file.Close()

	// Read bytecode
	reader := vm.NewBytecodeReader(file)
	chunk, maxRegs, slots, err := reader.ReadChunk()
	if err != nil {
		fmt.Println(red+"Error reading bytecode:", err, reset)
		return
	}

	fmt.Println(green+"✓ Bytecode loaded successfully", reset)

	// Prepare globals with builtins
	vm.EnsureBuiltinSlots(slots)
	globals := vm.NewGlobals(slots)
	globals = vm.EnsureGlobalsSize(globals, slots)

	// Register builtin functions (print, len, etc.)
	vm.RegisterBuiltins(globals, slots)

	// Register stdlib functions based on slots (for imports like math, crypto, etc.)
	vm.RegisterStdlibGlobals(globals, slots)

	// IMPORTANT: Patch constants in the chunk to restore stdlib functions
	// (Internal functions become null during serialization, we restore them here)
	vm.PatchConstantsFromGlobals(chunk, globals, slots)

	// Create main function and closure
	mainFn := &vm.FunctionObject{
		Chunk:        chunk,
		Arity:        0,
		Name:         "main",
		UpvalueCount: 0,
		MaxRegisters: maxRegs,
		ParamTypes:   []string{},
		ReturnType:   "",
	}
	mainClosure := &vm.ClosureObject{
		Function: mainFn,
		Upvalues: nil,
	}

	// Execute
	fmt.Println(blue + "Executing bytecode..." + reset)
	machine := vm.NewVM(globals)
	machine.SetSourceName(filepath.Base(filename))
	machine.EnableInstructionProfiling(profile)
	result := machine.Run(mainClosure)
	printVMResult(result)

	if profile {
		p := machine.InstructionProfiler()
		if p != nil {
			fmt.Printf(yellow+"\nExecuted %d instructions\n"+reset, p.Total())
			entries := p.EntriesSortedDesc()
			max := 12
			if len(entries) < max {
				max = len(entries)
			}
			fmt.Println(yellow + "Top opcodes:" + reset)
			for i := 0; i < max; i++ {
				e := entries[i]
				fmt.Printf("  %s: %d\n", e.Name, e.Count)
			}
		}
	}
}

func printVMResult(result value.Value) {
	if result.Kind != value.ErrorKind {
		fmt.Println(yellow, "Result:", result.ToString(), reset)
		return
	}

	err := result.AsError()
	if len(err.StackTrace) > 0 {
		fmt.Println("Traceback (most recent call last):")
		for _, frame := range err.StackTrace {
			fmt.Printf("  %s\n", frame)
		}
	}

	label := err.ErrorType
	if label == "" {
		label = "Error"
	}
	fmt.Printf("%s: %s\n", label, err.Message)
}

// Main function to decide between REPL and file execution
func main() {
	fmt.Println(green + "Kairo Interpreter v1.0" + reset)

	// Define command-line flags
	compileMode := flag.Bool("c", false, "Compile source to bytecode (.kbc)")
	compileModeLong := flag.Bool("compile", false, "Compile source to bytecode (.kbc)")
	outputFile := flag.String("o", "", "Output file for compiled bytecode")
	runBytecode := flag.Bool("r", false, "Run from bytecode file")
	runBytecodeLong := flag.Bool("run-bytecode", false, "Run from bytecode file")
	optimize := flag.Bool("O", false, "Enable compiler optimizations")
	profile := flag.Bool("prof", false, "Enable VM opcode execution profiling")

	flag.Parse()

	// Get non-flag arguments
	args := flag.Args()

	// Determine mode
	isCompile := *compileMode || *compileModeLong
	isRunBytecode := *runBytecode || *runBytecodeLong

	if isCompile {
		// Compile mode
		if len(args) == 0 {
			fmt.Println(red + "Error: No input file specified for compilation" + reset)
			fmt.Println("Usage: kairo -c <input.kr> [-o <output.krc>]")
			return
		}
		compileFile(args[0], *outputFile, *optimize)
	} else if isRunBytecode {
		// Run bytecode mode
		if len(args) == 0 {
			fmt.Println(red + "Error: No bytecode file specified" + reset)
			fmt.Println("Usage: kairo -r <file.krc>")
			return
		}
		executeBytecode(args[0], *profile)
	} else if len(args) > 0 {
		// Execute source file
		executeFile(args[0], *optimize, *profile)
	} else {
		// REPL mode
		startREPL(*optimize, *profile)
	}
}

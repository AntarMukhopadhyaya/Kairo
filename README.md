# Kairo Language

Kairo is a Go-based programming language project with its own lexer, parser, compiler, bytecode format, and virtual machine.

The project focuses on building a practical, dynamic language with modern syntax pieces (functions, closures, arrays/maps, try/catch/finally, methods), while keeping the implementation clear and hackable for language engineering experiments.

## Project At A Glance

- Language name: Kairo
- Implementation language: Go
- Module name: `Kairo`
- Execution model: source -> AST -> bytecode -> VM
- Extra mode: compile to `.kbc` bytecode and run later
- Current architecture (top-level):
  - `frontend/`: lexer, parser, AST
  - `compiler/`: public compiler facade
  - `vm/`: bytecode compiler/runtime objects, VM, opcodes, optimizer
  - `value/`: runtime value system
  - `stdlib/`: built-in modules
  - `benchmark/`: language test programs and benchmark scaffolding
  - `docs/`: architecture notes, roadmap, references

## Quick Start

### 1) Requirements

- Go 1.24+

### 2) Run REPL

```bash
go run .
```

Type `exit` to quit.

### 3) Run a source file

```bash
go run . demo/basic.kr
```

### 4) Compile source to bytecode

```bash
go run . -c demo/basic.kr
```

Optional output file:

```bash
go run . -c demo/basic.kr -o demo/basic.kbc
```

### 5) Run bytecode

```bash
go run . -r demo/basic.kbc
```

### 6) Enable optimizer and profiling

```bash
go run . -O demo/basic.kr
go run . -prof demo/basic.kr
```

## Language Syntax (Current)

Kairo currently uses semicolon-terminated statements and block syntax with braces.

## Variables and constants

```kairo
var x = 10;
const y = 20;
var name: string = "Kairo";
var count: number = 42;
```

## Functions

Named function with block body:

```kairo
fn add(a: number, b: number): number {
    return a + b;
}
```

Named function with arrow body:

```kairo
fn square(x: number): number => x * x
```

Function expression (closure support):

```kairo
var inc = fn(n: number): number {
    return n + 1;
};
```

## Control flow

```kairo
if (x > 0) {
    print("positive");
} else {
    print("non-positive");
}

while (x < 10) {
    x += 1;
}

for (var i = 0; i < 5; i = i + 1) {
    if (i == 2) continue;
    if (i == 4) break;
    print(i);
}
```

## Error handling

```kairo
try {
    print("work");
} catch Exception e {
    print("caught:", e);
} finally {
    print("cleanup always runs");
}
```

Multiple catch blocks are supported:

```kairo
try {
    print("run");
} catch RuntimeError err {
    print(err);
} catch TypeError te {
    print(te);
} finally {
    print("done");
}
```

## Collections and access

```kairo
var arr = [1, 2, 3];
arr[1] = 99;

var obj = {name: "Alice", age: 30};
obj.age = 31;
print(obj["name"]);
```

## Method calls

```kairo
var text = "hello";
print(text.toUpperCase());

var nums = [1, 2, 3];
nums.push(4);
print(nums.join(","));
```

## Modules

```kairo
import sqrt from "math";
var r = sqrt(16);
```

Notes:

- Imports currently resolve built-in stdlib modules registered by the runtime.
- Import syntax for side-effect modules is also parsed: `import "mod";`.
- `export` syntax is parsed in the frontend, but full module-export semantics are still pending in compiler/runtime.

## What Is Completed

The following are implemented and exercised across the repo tests/bench programs:

- Lexer + parser pipeline with line/column diagnostics
- AST for statements and expressions
- Bytecode VM with a rich opcode set
- Function declarations, function expressions, closures, upvalues
- Variables, constants, assignment and compound assignment (`+=`, `-=`, `*=`, `/=`)
- Arithmetic, comparison, logical operators with short-circuit behavior
- Arrays and maps (object-like string-keyed records)
- Property access and index access (`obj.prop`, `obj[key]`, `arr[i]`)
- Method dispatch for core types (string, array, map)
- Loop control (`break`, `continue`) in nested loops
- `try/catch/finally` support, including robust handling with `break/continue/return`
- Bytecode serialization/deserialization (`.kbc`) and execution
- Optional optimization passes (`-O`) including folding/peephole pathways
- VM opcode profiling (`-prof`)

## Current Standard Library Surface

Built-in modules under `stdlib/` include:

- `math`
- `string`
- `os`
- `random`
- `crypto`

Built-in methods include common operations such as:

- String: `toUpperCase`, `toLowerCase`, `split`, `charAt`, `includes`, `indexOf`, `trim`, `substring`, `replace`
- Array: `push`, `pop`, `slice`, `join`, `reverse`, `includes`, `indexOf`
- Map/Object: key/value helper methods (`keys`, `values`, `hasKey`, etc.)

## What Is Pending / Known Gaps

Based on current source and roadmap docs, these areas remain incomplete or in-progress:

- `switch/case` parsing/execution is not fully wired in the current parser/compiler path
- `throw` statement semantics are not fully exposed as user-facing syntax despite VM throw support
- Full `export` semantics are pending in compiler/runtime execution paths
- Advanced modern syntax: destructuring, spread/rest, template literals, ternary, optional chaining
- Full async/await execution model (tokens/AST support exists but runtime model is not complete)
- Expanded stdlib modules (JSON, regex, networking, richer filesystem, dates)

## Future Scope

The roadmap naturally breaks into three tracks:

### Near term

- Complete control-flow polish (`switch/case`, final throw semantics)
- Add high-value syntax sugar (template literals, default params, ternary)
- Keep strengthening loop and error-path correctness tests

### Mid term

- Expand stdlib with JSON and broader developer utility modules
- Improve optimization quality and memory behavior in VM hot paths
- Harden import/module ergonomics and packaging strategy

### Long term

- Consider stronger type tooling and diagnostics
- Explore performance upgrades (better register management, lower allocation pressure, optional JIT research)

## Repository Navigation

- Start language syntax work in `frontend/lexer.go`, `frontend/parser.go`, `frontend/ast.go`
- Start compiler work in `vm/compiler.go`
- Start runtime execution work in `vm/vm.go`
- Start value/runtime type work in `value/value.go`
- Start stdlib work in `stdlib/*.go`

## Notes For Contributors

- For active development, treat the current root package layout (`frontend`, `compiler`, `vm`, `value`, `stdlib`) as the canonical structure.
- Add language behavior tests as `.kr` programs under `benchmark/` and keep docs updated when syntax/semantics change.

## License

MIT

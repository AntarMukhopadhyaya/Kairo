package compiler

import "Kairo/vm"

// Compiler is the public compiler type exposed by the dedicated compiler package.
type Compiler = vm.Compiler

// OptimizationStats is re-exported for callers that inspect optimization metrics.
type OptimizationStats = vm.OptimizationStats

// NewCompiler creates a new compiler instance.
func NewCompiler() *Compiler {
	return vm.NewCompiler()
}

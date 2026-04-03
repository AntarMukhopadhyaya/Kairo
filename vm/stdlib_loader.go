package vm

import (
	"Kairo/stdlib"
)

// RegisterStdlibGlobals populates globals with stdlib functions based on slot names
// This is used when loading bytecode to restore imported stdlib functions
func RegisterStdlibGlobals(globals []VariableInfo, slots map[string]int) {
	// For each slot, check if it matches a stdlib export
	for name, slot := range slots {
		// Skip builtins (they're handled separately)
		if name == "print" || name == "len" {
			continue
		}

		// Search through all stdlib modules for this export
		for _, module := range stdlib.BuiltinModules {
			if exportVal, ok := module.Exports[name]; ok {
				// Found it! Store in globals
				if slot >= 0 && slot < len(globals) {
					globals[slot] = VariableInfo{
						Value: exportVal,
						Type:  "",
					}
				}
				break
			}
		}
	}
}

// PatchConstantsFromGlobals is no longer needed but kept for API compatibility
func PatchConstantsFromGlobals(chunk *Chunk, globals []VariableInfo, slots map[string]int) {
	// No-op: The VM now handles this automatically
}

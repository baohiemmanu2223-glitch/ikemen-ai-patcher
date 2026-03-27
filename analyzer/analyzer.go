package analyzer

import (
	"ikemen-ai-patcher/parser"
)

// Analyze runs the full analysis pipeline on parsed character data.
// Returns a comprehensive Analysis struct with all findings.
func Analyze(data *parser.CharacterData) *Analysis {
	result := &Analysis{
		CharName:    data.Name,
		TotalStates: len(data.States),
	}

	// 1. Categorize moves
	result.Normals, result.Specials, result.Hypers = CategorizeStates(data)

	// 2. Detect combo chains
	result.ComboChains = DetectComboChains(data)

	// 3. Scan variables
	result.Vars, result.Fvars, result.Sysvars = ScanVariables(data)

	// 4. Find unused vars for AI injection (Phase 4 Memory AI needs 5 vars, 8 fvars)
	// Now scans all available indices instead of just a fixed amount
	result.UnusedVars = FindAllUnusedVars(result.Vars, 60)
	result.UnusedFvars = FindAllUnusedVars(result.Fvars, 40)
	result.UnusedSysvars = FindAllUnusedVars(result.Sysvars, 5)
	
	// Detailed variable functional tracing
	result.VarUsages = AnalyzeVariableUsage(data)

	result.VarAllocation = make(map[string]int)
	result.FvarAllocation = make(map[string]int)
	
	// Helper to allocate preferred vars or fallback to unused
	allocVar := func(preferred int, unused *[]int) int {
		if !containsInt(result.Vars, preferred) && !containsInt(result.Sysvars, preferred) {
			return preferred
		}
		if len(*unused) > 0 {
			val := (*unused)[0]
			*unused = (*unused)[1:]
			return val
		}
		return -1
	}

	allocFvar := func(preferred int, unused *[]int) int {
		if !containsInt(result.Fvars, preferred) {
			return preferred
		}
		if len(*unused) > 0 {
			val := (*unused)[0]
			*unused = (*unused)[1:]
			return val
		}
		return -1
	}

	result.VarAllocation["AILevel"] = allocVar(59, &result.UnusedVars)
	result.VarAllocation["TargetIndex"] = allocVar(57, &result.UnusedVars)
	result.VarAllocation["Archetype"] = allocVar(58, &result.UnusedVars)
	result.VarAllocation["AdaptiveMode"] = allocVar(56, &result.UnusedVars)
	result.VarAllocation["AdaptiveTimer"] = allocVar(55, &result.UnusedVars)

	result.FvarAllocation["EnemyXVel"] = allocFvar(17, &result.UnusedFvars)
	result.FvarAllocation["GuardCheck"] = allocFvar(15, &result.UnusedFvars)
	result.FvarAllocation["GrabCD"] = allocFvar(16, &result.UnusedFvars)
	result.FvarAllocation["AirDef"] = allocFvar(22, &result.UnusedFvars)
	result.FvarAllocation["EnemyIdle"] = allocFvar(23, &result.UnusedFvars)
	result.FvarAllocation["ZoningTime"] = allocFvar(31, &result.UnusedFvars)
	result.FvarAllocation["HybridPriority"] = allocFvar(32, &result.UnusedFvars)
	result.FvarAllocation["EnemyJumpCount"] = allocFvar(33, &result.UnusedFvars)

	// 5. Frame Data Analysis (Phase 4)
	result.FrameData = ParseFrameData(data)

	// 6. Advanced Categorization (Phase 5)
	result.Classified = CategorizeAdvanced(data)

	// 7. Risk analysis
	result.RiskStates = AnalyzeRisk(data)

	return result
}

func containsInt(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

package analyzer

import (
	"fmt"
	"strings"

	"ikemen-ai-patcher/parser"
	"ikemen-ai-patcher/utils"
)

// ScanVariables scans all states for var(), fvar(), and sysvar() usage.
// Returns which indices are in use for each type.
func ScanVariables(data *parser.CharacterData) (vars, fvars, sysvars []int) {
	varSet := make(map[int]bool)
	fvarSet := make(map[int]bool)
	sysvarSet := make(map[int]bool)

	for _, state := range data.States {
		for _, ctrl := range state.Controllers {
			// Scan all trigger expressions
			for _, t := range ctrl.Triggers {
				scanLine(t.Raw, varSet, fvarSet, sysvarSet)
			}
			// Scan all values
			for _, v := range ctrl.Values {
				scanLine(v, varSet, fvarSet, sysvarSet)
			}
			// Scan controller type-specific references
			if strings.EqualFold(ctrl.Type, "VarSet") || strings.EqualFold(ctrl.Type, "VarAdd") {
				if v, ok := ctrl.Values["v"]; ok {
					for _, idx := range utils.ExtractVarIndices("var(" + v + ")") {
						varSet[idx] = true
					}
				}
				if fv, ok := ctrl.Values["fv"]; ok {
					for _, idx := range utils.ExtractFvarIndices("fvar(" + fv + ")") {
						fvarSet[idx] = true
					}
				}
			}
		}
	}

	vars = mapKeysToSlice(varSet)
	fvars = mapKeysToSlice(fvarSet)
	sysvars = mapKeysToSlice(sysvarSet)
	return
}

// FindUnusedVars finds N unused var indices in [0, maxVar).
func FindUnusedVars(usedVars []int, count, maxVar int) []int {
	return utils.FindUnusedVars(usedVars, count, maxVar)
}

// FindUnusedFvars finds N unused fvar indices.
func FindUnusedFvars(usedFvars []int, count, maxFvar int) []int {
	return utils.FindUnusedVars(usedFvars, count, maxFvar)
}

// FindAllUnusedVars finds ALL unused indices up to max.
func FindAllUnusedVars(used []int, max int) []int {
	usedSet := map[int]bool{}
	for _, v := range used {
		usedSet[v] = true
	}
	var result []int
	for i := 0; i < max; i++ {
		if !usedSet[i] {
			result = append(result, i)
		}
	}
	return result
}

// AnalyzeVariableUsage determines where variables are read and written.
func AnalyzeVariableUsage(data *parser.CharacterData) []VarUsage {
	usageMap := make(map[string]*VarUsage)
	
	getUsage := func(id int, t string) *VarUsage {
		key := fmt.Sprintf("%s:%d", t, id)
		if u, ok := usageMap[key]; ok {
			return u
		}
		u := &VarUsage{Index: id, Type: t}
		usageMap[key] = u
		return u
	}
	
	for _, state := range data.States {
		for _, ctrl := range state.Controllers {
			// Find reads in triggers/values
			for _, t := range ctrl.Triggers {
				for _, v := range utils.ExtractVarIndices(t.Raw) { getUsage(v, "var").UsedIn = append(getUsage(v, "var").UsedIn, state.Def.ID) }
				for _, fv := range utils.ExtractFvarIndices(t.Raw) { getUsage(fv, "fvar").UsedIn = append(getUsage(fv, "fvar").UsedIn, state.Def.ID) }
				for _, sv := range utils.ExtractSysvarIndices(t.Raw) { getUsage(sv, "sysvar").UsedIn = append(getUsage(sv, "sysvar").UsedIn, state.Def.ID) }
			}
			for _, val := range ctrl.Values {
				for _, v := range utils.ExtractVarIndices(val) { getUsage(v, "var").UsedIn = append(getUsage(v, "var").UsedIn, state.Def.ID) }
				for _, fv := range utils.ExtractFvarIndices(val) { getUsage(fv, "fvar").UsedIn = append(getUsage(fv, "fvar").UsedIn, state.Def.ID) }
				for _, sv := range utils.ExtractSysvarIndices(val) { getUsage(sv, "sysvar").UsedIn = append(getUsage(sv, "sysvar").UsedIn, state.Def.ID) }
			}
			
			// Find writes in VarSet/VarAdd etc
			isSet := strings.EqualFold(ctrl.Type, "VarSet") || strings.EqualFold(ctrl.Type, "VarAdd")
			if isSet {
				if v, ok := ctrl.Values["v"]; ok {
					for _, idx := range utils.ExtractVarIndices("var(" + v + ")") { getUsage(idx, "var").SetIn = append(getUsage(idx, "var").SetIn, state.Def.ID) }
				}
				if fv, ok := ctrl.Values["fv"]; ok {
					for _, idx := range utils.ExtractFvarIndices("fvar(" + fv + ")") { getUsage(idx, "fvar").SetIn = append(getUsage(idx, "fvar").SetIn, state.Def.ID) }
				}
				if sv, ok := ctrl.Values["sysvar"]; ok {
					for _, idx := range utils.ExtractSysvarIndices("sysvar(" + sv + ")") { getUsage(idx, "sysvar").SetIn = append(getUsage(idx, "sysvar").SetIn, state.Def.ID) }
				}
			}
		}
	}
	
	var result []VarUsage
	for _, u := range usageMap {
		u.UsedIn = utils.UniqueInts(u.UsedIn)
		u.SetIn = utils.UniqueInts(u.SetIn)
		result = append(result, *u)
	}
	return result
}

// scanLine extracts var/fvar/sysvar indices from a text line.
func scanLine(line string, varSet, fvarSet, sysvarSet map[int]bool) {
	for _, idx := range utils.ExtractVarIndices(line) {
		varSet[idx] = true
	}
	for _, idx := range utils.ExtractFvarIndices(line) {
		fvarSet[idx] = true
	}
	for _, idx := range utils.ExtractSysvarIndices(line) {
		sysvarSet[idx] = true
	}
}

func mapKeysToSlice(m map[int]bool) []int {
	result := make([]int, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

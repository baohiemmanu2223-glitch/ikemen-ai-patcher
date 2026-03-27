// Package validator performs pre-patch safety checks on character data.
package validator

import (
	"fmt"
	"strings"

	"ikemen-ai-patcher/analyzer"
	"ikemen-ai-patcher/parser"
)

// ValidationResult holds all validation findings.
type ValidationResult struct {
	Errors   []string // Critical issues that prevent patching
	Warnings []string // Non-critical issues to report
	Valid    bool     // True if no errors (warnings are OK)
}

// Validate performs pre-patch validation on the character data and analysis.
// Checks for: missing states, invalid transitions, infinite loops, var conflicts.
func Validate(data *parser.CharacterData, analysis *analyzer.Analysis) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// 1. Check that all ChangeState targets reference existing states
	checkStateReferences(data, result)

	// 2. Check for infinite combo loops (circular ChangeState chains)
	checkInfiniteLoops(data, result)

	// 3. Check that AI vars don't conflict with existing usage
	checkVarConflicts(analysis, result)

	// 4. Check for minimum requirements
	checkMinimumRequirements(data, analysis, result)

	return result
}

// checkStateReferences validates all ChangeState targets exist as defined states.
func checkStateReferences(data *parser.CharacterData, result *ValidationResult) {
	for id, state := range data.States {
		for _, ctrl := range state.Controllers {
			if !strings.EqualFold(ctrl.Type, "ChangeState") {
				continue
			}
			val, ok := ctrl.Values["value"]
			if !ok {
				continue
			}
			target := parseSimpleInt(val)
			// Skip expressions (non-literal values) and common system states
			if target == 0 && val != "0" {
				continue // Expression, can't validate statically
			}
			// System states (0-199) are always valid
			if target >= 0 && target < 200 {
				continue
			}
			// Negative states (common states) are engine-provided
			if target < 0 {
				continue
			}
			// Check if target state exists
			if _, exists := data.States[target]; !exists {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("State %d references unknown state %d via ChangeState", id, target))
			}
		}
	}
}

// checkInfiniteLoops detects circular ChangeState chains.
func checkInfiniteLoops(data *parser.CharacterData, result *ValidationResult) {
	// Build adjacency graph of state transitions
	graph := make(map[int][]int)
	for id, state := range data.States {
		if id < 0 {
			continue
		}
		targets := state.GetChangeStateTargets()
		for _, t := range targets {
			if t > 0 { // Skip system states
				graph[id] = append(graph[id], t)
			}
		}
	}

	// DFS cycle detection
	visited := make(map[int]int) // 0=unvisited, 1=in-stack, 2=done
	var stack []int

	var dfs func(node int) bool
	dfs = func(node int) bool {
		visited[node] = 1 // In progress
		stack = append(stack, node)

		for _, next := range graph[node] {
			if visited[next] == 1 {
				// Found a cycle — but only flag it if the cycle is short (< 5 states)
				// since long cycles may be intentional state machines
				cycleStart := -1
				for i, s := range stack {
					if s == next {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycleLen := len(stack) - cycleStart
					if cycleLen <= 3 {
						cycle := stack[cycleStart:]
						result.Warnings = append(result.Warnings,
							fmt.Sprintf("Potential infinite loop detected: %v (cycle length %d)", cycle, cycleLen))
					}
				}
				return true
			}
			if visited[next] == 0 {
				dfs(next)
			}
		}

		stack = stack[:len(stack)-1]
		visited[node] = 2 // Done
		return false
	}

	for node := range graph {
		if visited[node] == 0 {
			dfs(node)
		}
	}
}

// checkVarConflicts ensures AI vars don't conflict with existing character vars.
func checkVarConflicts(analysis *analyzer.Analysis, result *ValidationResult) {
	if len(analysis.UnusedVars) < 5 {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Not enough unused vars for AI (need 5, found %d). Character uses too many vars.", len(analysis.UnusedVars)))
		result.Valid = false
	}
}

// checkMinimumRequirements ensures the character has enough data to generate AI.
func checkMinimumRequirements(data *parser.CharacterData, analysis *analyzer.Analysis, result *ValidationResult) {
	if len(data.States) < 3 {
		result.Errors = append(result.Errors,
			"Too few states found — character may not be properly parsed")
		result.Valid = false
	}

	if len(analysis.Normals) == 0 && len(analysis.Specials) == 0 {
		result.Warnings = append(result.Warnings,
			"No attack states detected — AI will have limited actions")
	}

	if data.Neg1State == nil {
		result.Warnings = append(result.Warnings,
			"No [Statedef -1] found — AI injection may need manual placement")
	}
}

func parseSimpleInt(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	neg := false
	start := 0
	if len(s) > 0 && s[0] == '-' {
		neg = true
		start = 1
	}
	for i := start; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n = n*10 + int(s[i]-'0')
		} else {
			break
		}
	}
	if neg {
		return -n
	}
	return n
}

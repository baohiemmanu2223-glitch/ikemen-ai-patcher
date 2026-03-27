package style

import (
	"fmt"
	"strings"

	"ikemen-ai-patcher/parser"
)

// ValidateStyle checks if a style is compatible with a character's state data.
// Removes decisions that reference non-existent states and returns validation results.
func ValidateStyle(s *AIStyle, charData *parser.CharacterData) *StyleValidation {
	result := &StyleValidation{Valid: true}

	var validDecisions []AIDecision

	for i, d := range s.Decisions {
		// Check if target state exists in character
		if _, exists := charData.States[d.State]; !exists {
			// System states (0-199) are always valid
			if d.State >= 200 {
				result.InvalidStates = append(result.InvalidStates, d.State)
				result.RemovedDecisions = append(result.RemovedDecisions, i)
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Decision P%d references non-existent state %d — removed",
						d.Priority, d.State))
				continue
			}
		}
		validDecisions = append(validDecisions, d)
	}

	// Replace decisions with validated set
	s.Decisions = validDecisions

	if len(validDecisions) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors,
			"No valid decisions remain after validation — style is incompatible with this character")
	}

	// Check for potential infinite loops (same state in multiple consecutive priorities)
	checkStyleLoops(s, result)

	return result
}

// checkStyleLoops detects potential infinite loop patterns in style decisions.
func checkStyleLoops(s *AIStyle, result *StyleValidation) {
	stateCount := make(map[int]int)
	for _, d := range s.Decisions {
		stateCount[d.State]++
		if stateCount[d.State] > 3 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("State %d referenced %d times — potential spam risk",
					d.State, stateCount[d.State]))
		}
	}
}

// MapStrategies categorizes all decisions by strategy type.
// Returns a map of strategy -> decisions for that strategy.
func MapStrategies(s *AIStyle) map[string][]AIDecision {
	result := make(map[string][]AIDecision)
	for _, d := range s.Decisions {
		result[d.Strategy] = append(result[d.Strategy], d)
	}
	return result
}

// EnsureSafety adds missing safety features to a style:
// 1. Anti-spam cooldown on decisions that lack it
// 2. Ensures no decision has RandWeight of 0 (would always fire)
func EnsureSafety(s *AIStyle) {
	for i := range s.Decisions {
		// Ensure minimum cooldown
		if s.Decisions[i].Cooldown <= 0 {
			// Auto-assign based on strategy
			switch s.Decisions[i].Strategy {
			case "punish":
				s.Decisions[i].Cooldown = 10
			case "combo":
				s.Decisions[i].Cooldown = 5
			case "defense":
				s.Decisions[i].Cooldown = 8
			default:
				s.Decisions[i].Cooldown = 15
			}
		}

		// Prevent deterministic firing (RandWeight = 0 means always)
		if s.Decisions[i].RandWeight <= 0 {
			s.Decisions[i].RandWeight = 700
		}
		// Cap at 950 to prevent near-impossible firing
		if s.Decisions[i].RandWeight > 950 {
			s.Decisions[i].RandWeight = 950
		}
	}
}

// GenerateMidMatchSwitchCode creates MUGEN code for switching styles mid-match.
// Uses life ratio to determine when to shift between aggressive/defensive modes.
func GenerateMidMatchSwitchCode(behaviorVar int, aiActivation string) string {
	return fmt.Sprintf(`
; -------------------------------------------------------
; MID-MATCH STYLE SWITCHING
; Adapts behavior based on health ratio.
; var(%d): 0=balanced, 1=aggressive (high hp), 2=defensive (low hp)
; -------------------------------------------------------
[State -1, Style Switch: Aggressive]
type = VarSet
triggerall = %s
trigger1 = Life > (LifeMax * 0.6)
trigger1 = EnemyNear, Life < (EnemyNear, LifeMax * 0.5)
v = %d
value = 1

[State -1, Style Switch: Defensive]
type = VarSet
triggerall = %s
trigger1 = Life < (LifeMax * 0.3)
v = %d
value = 2

[State -1, Style Switch: Balanced]
type = VarSet
triggerall = %s
triggerall = var(%d) != 0
trigger1 = Life = [LifeMax * 0.3, LifeMax * 0.6]
v = %d
value = 0

`, behaviorVar, aiActivation, behaviorVar,
		aiActivation, behaviorVar,
		aiActivation, behaviorVar, behaviorVar)
}

// FormatStyleSummary returns a human-readable summary of a style.
func FormatStyleSummary(s *AIStyle) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Style: %s\n", s.Name))
	if s.Description != "" {
		sb.WriteString(fmt.Sprintf("  %s\n", s.Description))
	}
	sb.WriteString(fmt.Sprintf("  Decisions: %d\n", len(s.Decisions)))

	strategies := MapStrategies(s)
	for strat, decs := range strategies {
		sb.WriteString(fmt.Sprintf("  [%s] %d decision(s)\n", strat, len(decs)))
		for _, d := range decs {
			sb.WriteString(fmt.Sprintf("    P%d → State %d (cooldown:%d, rand:%d)\n",
				d.Priority, d.State, d.Cooldown, d.RandWeight))
		}
	}

	if s.ReactionDelay > 0 {
		sb.WriteString(fmt.Sprintf("  Reaction Delay: %d ticks\n", s.ReactionDelay))
	}
	if s.MidMatchSwitch {
		sb.WriteString("  Mid-Match Switching: enabled\n")
	}

	return sb.String()
}

// ExpandAbstractStates replaces decisions using abstract @categories with concrete state IDs.
// It iterates over Decisions, looks up the abstract category in the classified map,
// and duplicates the decision for every matching state ID found by the analyzer.
func ExpandAbstractStates(s *AIStyle, classified map[string][]int) {
	var expanded []AIDecision
	for _, d := range s.Decisions {
		if d.StateAbstract != "" {
			if states, ok := classified[d.StateAbstract]; ok && len(states) > 0 {
				// Duplicates for each state ID found
				for _, id := range states {
					newD := d
					newD.StateAbstract = "" // Clear it to avoid infinite loops
					newD.State = id
					
					// Clarify the label to distinguish generated variants
					if newD.Label != "" {
						newD.Label = fmt.Sprintf("%s (%s %d)", newD.Label, d.StateAbstract, id)
					}
					expanded = append(expanded, newD)
				}
			}
			// If abstract category not found or empty, it is dropped from expansion
		} else {
			expanded = append(expanded, d)
		}
	}
	s.Decisions = expanded
}


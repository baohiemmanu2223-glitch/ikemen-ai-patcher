// Package style implements the pluggable AI Style System.
// Styles are structured AI behavior definitions loaded from .zss files.
// They are parsed into an internal model, validated, blended, and converted
// to MUGEN state controller code — NEVER injected as raw code.
package style

// AIDecision represents a single AI behavior rule with priority ordering.
type AIDecision struct {
	Priority      int      // Lower number = higher priority (1 = highest)
	Conditions    []string // Trigger conditions (MUGEN expressions)
	State         int      // Target state ID to transition to
	StateAbstract string   // Abstract category (e.g. "projectile" from "@projectile")
	Strategy      string   // Auto-mapped: "punish", "combo", "neutral", "defense", "bait"
	Cooldown      int      // Ticks to wait before reuse (0 = no cooldown)
	RandWeight    int      // Randomization threshold (0-1000, 0=always, 1000=never)
	Label         string   // Human-readable description
}

// AIStyle represents a complete AI behavior profile loaded from a .zss file.
type AIStyle struct {
	Name        string       // Style name (e.g., "BOSS_AGGRESSIVE")
	Description string       // Human-readable description
	Decisions   []AIDecision // Ordered list of decisions (priority-sorted)
	FilePath    string       // Source file path

	// Advanced features
	AdaptiveVars  map[string]int // Named vars for behavior tuning (e.g., "aggression" -> var index)
	ReactionDelay int            // Simulated reaction delay in ticks (0 = instant)
	MidMatchSwitch bool          // Whether style can switch mid-match based on health
}

// StyleBlend represents a weighted combination of multiple styles.
type StyleBlend struct {
	Styles  []*AIStyle // Styles being blended
	Weights []float64  // Weight for each style (should sum to 1.0)
}

// StyleValidation holds results of compatibility checking against character data.
type StyleValidation struct {
	Valid          bool
	InvalidStates  []int    // States referenced but not found in character
	RemovedDecisions []int  // Indices of decisions removed due to invalid states
	Warnings       []string
	Errors         []string
}

// Blend merges multiple styles into a single AIStyle using weights.
// Higher-weighted styles get proportionally higher randomization thresholds.
func (sb *StyleBlend) Blend() *AIStyle {
	if len(sb.Styles) == 0 {
		return &AIStyle{Name: "empty_blend"}
	}
	if len(sb.Styles) == 1 {
		return sb.Styles[0]
	}

	merged := &AIStyle{
		Name:        buildBlendName(sb),
		Description: "Blended style",
		AdaptiveVars: make(map[string]int),
	}

	// Merge decisions from all styles, adjusting RandWeight by blend weight
	for i, s := range sb.Styles {
		weight := 1.0
		if i < len(sb.Weights) {
			weight = sb.Weights[i]
		}

		for _, d := range s.Decisions {
			blended := d // copy
			// Scale randomization threshold by weight
			// Weight 1.0 = full chance, 0.5 = half chance
			if blended.RandWeight > 0 {
				blended.RandWeight = int(float64(blended.RandWeight) * weight)
			} else {
				blended.RandWeight = int(800.0 * weight) // Default 800 scaled
			}
			blended.Label = s.Name + ": " + blended.Label
			merged.Decisions = append(merged.Decisions, blended)
		}

		// Merge adaptive vars
		for k, v := range s.AdaptiveVars {
			merged.AdaptiveVars[k] = v
		}

		// Use highest reaction delay
		if s.ReactionDelay > merged.ReactionDelay {
			merged.ReactionDelay = s.ReactionDelay
		}
	}

	// Sort by priority (lower = higher priority)
	sortDecisions(merged.Decisions)

	// Resolve conflicts: same priority + same state → keep highest weight
	merged.Decisions = resolveConflicts(merged.Decisions)

	return merged
}

// sortDecisions sorts decisions by priority (ascending = highest first).
func sortDecisions(decisions []AIDecision) {
	n := len(decisions)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if decisions[j].Priority > decisions[j+1].Priority {
				decisions[j], decisions[j+1] = decisions[j+1], decisions[j]
			}
		}
	}
}

// resolveConflicts removes duplicate state entries at the same priority.
// Keeps the one with higher RandWeight (more likely to fire).
func resolveConflicts(decisions []AIDecision) []AIDecision {
	seen := make(map[[2]int]int) // [priority, state] -> index in result
	var result []AIDecision

	for _, d := range decisions {
		key := [2]int{d.Priority, d.State}
		if idx, exists := seen[key]; exists {
			// Keep the one with higher RandWeight
			if d.RandWeight > result[idx].RandWeight {
				result[idx] = d
			}
		} else {
			seen[key] = len(result)
			result = append(result, d)
		}
	}
	return result
}

func buildBlendName(sb *StyleBlend) string {
	var names []string
	for _, s := range sb.Styles {
		names = append(names, s.Name)
	}
	return joinStrings(names, "+")
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += sep + ss[i]
	}
	return result
}

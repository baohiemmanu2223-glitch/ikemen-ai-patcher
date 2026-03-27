package style

import (
	"fmt"
	"strconv"
	"strings"

	"ikemen-ai-patcher/utils"
)

// ParseStyleFile reads a .zss style definition file and returns an AIStyle.
// This is NOT raw code injection — it parses the structured format into
// the internal AIStyle model for validation and conversion.
func ParseStyleFile(filePath string) (*AIStyle, error) {
	lines, err := utils.ReadFileLines(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read style file: %w", err)
	}

	style := &AIStyle{
		FilePath:     filePath,
		AdaptiveVars: make(map[string]int),
	}

	// State machine for parsing
	var currentDecision *AIDecision
	inDecisionBlock := false
	priority := 0

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		// --- Comment-based metadata ---
		if strings.HasPrefix(line, ";") {
			meta := strings.TrimSpace(line[1:])
			parseMeta(style, meta)
			continue
		}

		if line == "" {
			// End current decision if we have one with data
			if currentDecision != nil && (currentDecision.State > 0 || currentDecision.StateAbstract != "") {
				style.Decisions = append(style.Decisions, *currentDecision)
				currentDecision = nil
			}
			continue
		}

		// --- Section headers ---
		lower := strings.ToLower(line)

		if strings.HasPrefix(lower, "[function") || strings.HasPrefix(lower, "[decision") {
			inDecisionBlock = true
			priority = 0
			continue
		}

		if strings.HasPrefix(line, "[") {
			inDecisionBlock = false
			continue
		}

		if !inDecisionBlock {
			continue
		}

		// --- Parse trigger/value lines ---
		if strings.HasPrefix(lower, "trigger") {
			// Extract trigger level and condition
			level, condition := parseTriggerLine(line)
			if condition == "" {
				continue
			}

			if currentDecision == nil || level != currentDecision.Priority {
				// Save previous decision
				if currentDecision != nil && (currentDecision.State > 0 || currentDecision.StateAbstract != "") {
					style.Decisions = append(style.Decisions, *currentDecision)
				}
				priority++
				currentDecision = &AIDecision{
					Priority:   priority,
					RandWeight: 700, // Default
				}
			}
			currentDecision.Priority = priority
			_ = level // We use positional priority instead
			currentDecision.Conditions = append(currentDecision.Conditions, condition)

		} else if strings.HasPrefix(lower, "value") {
			// value = N or value = @category
			_, val := splitKeyValue(line)
			if currentDecision != nil {
				valTrim := strings.ToLower(strings.TrimSpace(val))
				if strings.HasPrefix(valTrim, "@") {
					currentDecision.StateAbstract = valTrim[1:]
				} else {
					n, err := strconv.Atoi(valTrim)
					if err == nil {
						currentDecision.State = n
					}
				}
			}

		} else if strings.HasPrefix(lower, "cooldown") {
			_, val := splitKeyValue(line)
			if currentDecision != nil {
				n, _ := strconv.Atoi(strings.TrimSpace(val))
				currentDecision.Cooldown = n
			}

		} else if strings.HasPrefix(lower, "weight") || strings.HasPrefix(lower, "random") {
			_, val := splitKeyValue(line)
			if currentDecision != nil {
				n, _ := strconv.Atoi(strings.TrimSpace(val))
				currentDecision.RandWeight = n
			}

		} else if strings.HasPrefix(lower, "label") || strings.HasPrefix(lower, "name") {
			_, val := splitKeyValue(line)
			if currentDecision != nil {
				currentDecision.Label = strings.Trim(val, "\"")
			}

		} else if strings.HasPrefix(lower, "strategy") {
			_, val := splitKeyValue(line)
			if currentDecision != nil {
				currentDecision.Strategy = strings.ToLower(strings.TrimSpace(val))
			}
		}
	}

	// Save last decision
	if currentDecision != nil && (currentDecision.State > 0 || currentDecision.StateAbstract != "") {
		style.Decisions = append(style.Decisions, *currentDecision)
	}

	// Auto-map strategies if not explicitly set
	for i := range style.Decisions {
		if style.Decisions[i].Strategy == "" {
			style.Decisions[i].Strategy = autoMapStrategy(style.Decisions[i])
		}
		if style.Decisions[i].Label == "" {
			style.Decisions[i].Label = fmt.Sprintf("Decision P%d -> State %d",
				style.Decisions[i].Priority, style.Decisions[i].State)
		}
	}

	if style.Name == "" {
		// Derive name from filename
		name := filePath
		if idx := strings.LastIndexAny(name, "/\\"); idx >= 0 {
			name = name[idx+1:]
		}
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[:idx]
		}
		style.Name = strings.ToUpper(name)
	}

	return style, nil
}

// parseMeta extracts metadata from comment lines.
func parseMeta(style *AIStyle, meta string) {
	upper := strings.ToUpper(meta)

	if strings.HasPrefix(upper, "AI STYLE:") {
		style.Name = strings.TrimSpace(meta[len("AI STYLE:"):])
	} else if strings.HasPrefix(upper, "DESCRIPTION:") {
		style.Description = strings.TrimSpace(meta[len("DESCRIPTION:"):])
	} else if strings.HasPrefix(upper, "REACTION_DELAY:") || strings.HasPrefix(upper, "REACTION DELAY:") {
		val := strings.TrimSpace(meta[strings.Index(meta, ":")+1:])
		n, _ := strconv.Atoi(val)
		style.ReactionDelay = n
	} else if strings.HasPrefix(upper, "MID_MATCH_SWITCH:") || strings.HasPrefix(upper, "ADAPTIVE:") {
		val := strings.ToLower(strings.TrimSpace(meta[strings.Index(meta, ":")+1:]))
		style.MidMatchSwitch = val == "true" || val == "yes" || val == "1"
	} else if strings.HasPrefix(upper, "VAR:") {
		// VAR: aggression = 40
		rest := strings.TrimSpace(meta[4:])
		parts := strings.SplitN(rest, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			idx, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			style.AdaptiveVars[name] = idx
		}
	}
}

// parseTriggerLine extracts trigger level and condition.
// "trigger1 = EnemyNear, MoveType = A" → (1, "EnemyNear, MoveType = A")
func parseTriggerLine(line string) (int, string) {
	eqIdx := strings.Index(line, "=")
	if eqIdx < 0 {
		return 0, ""
	}

	prefix := strings.TrimSpace(strings.ToLower(line[:eqIdx]))
	condition := strings.TrimSpace(line[eqIdx+1:])

	if prefix == "triggerall" {
		return 0, condition
	}

	// Extract number from "triggerN"
	numStr := strings.TrimPrefix(prefix, "trigger")
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, condition
	}
	return n, condition
}

// autoMapStrategy guesses the strategy category from trigger conditions.
func autoMapStrategy(d AIDecision) string {
	for _, cond := range d.Conditions {
		lower := strings.ToLower(cond)

		// Punish indicators
		if strings.Contains(lower, "gethitvar") ||
			strings.Contains(lower, "animtime") && strings.Contains(lower, "<") ||
			strings.Contains(lower, "movetype = a") && strings.Contains(lower, "enemynear") {
			return "punish"
		}

		// Combo indicators
		if strings.Contains(lower, "movehit") ||
			strings.Contains(lower, "movecontact") ||
			strings.Contains(lower, "moveguarded") {
			return "combo"
		}

		// Defense indicators
		if strings.Contains(lower, "inguarddist") ||
			strings.Contains(lower, "numproj") {
			return "defense"
		}
	}

	// Default to neutral
	return "neutral"
}

// splitKeyValue splits "key = value" respecting the first = only.
func splitKeyValue(line string) (string, string) {
	idx := strings.Index(line, "=")
	if idx < 0 {
		return line, ""
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:])
}

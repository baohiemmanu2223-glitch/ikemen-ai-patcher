package analyzer

import (
	"strings"
	"ikemen-ai-patcher/parser"
)

// CategorizeAdvanced parses character states to group them into abstract combat archetypes.
// These archetypes (projectile, antiair, reversal, overhead, low, grab, dash)
// allow the style system to generate universal ZSS files without hardcoding character-specific state IDs.
func CategorizeAdvanced(data *parser.CharacterData) map[string][]int {
	classified := make(map[string][]int)
	classified["projectile"] = []int{}
	classified["antiair"] = []int{}
	classified["reversal"] = []int{}
	classified["overhead"] = []int{}
	classified["low"] = []int{}
	classified["grab"] = []int{}
	classified["dash"] = []int{}
	classified["launcher"] = []int{}

	for _, state := range data.States {
		id := state.Def.ID

		// Ignore system states, focus on typical custom states
		if id < 200 && id != 100 && id != 105 {
			continue
		}

		// Detect Dash
		if id == 100 || id == 105 {
			classified["dash"] = append(classified["dash"], id)
			continue
		}

		// Detect Projectiles
		if state.HasController("Projectile") || state.HasController("Helper") {
			classified["projectile"] = append(classified["projectile"], id)
		}

		// Detect Reversals (Invincibility frames)
		if state.HasController("NotHitBy") || state.HasController("HitOverride") {
			classified["reversal"] = append(classified["reversal"], id)
		}

		// Detect state properties via HitDef and other Controllers
		for _, ctrl := range state.Controllers {
			if strings.EqualFold(ctrl.Type, "HitDef") {
				attr := strings.ToLower(ctrl.Values["attr"])
				flags := strings.ToLower(ctrl.Values["guardflag"])
				fall := strings.ToLower(ctrl.Values["fall"])

				// Command Grabs (Catch Throws)
				if strings.Contains(attr, "ct") {
					classified["grab"] = append(classified["grab"], id)
				}

				// Overheads (Ground attack must be blocked high)
				if strings.Contains(attr, "s,") && strings.Contains(flags, "h") && !strings.Contains(flags, "l") {
					classified["overhead"] = append(classified["overhead"], id)
				}

				// Lows / Sweeps (Must be blocked low)
				if strings.Contains(flags, "l") && !strings.Contains(flags, "h") {
					classified["low"] = append(classified["low"], id)
				}

				// Launchers (Causes fall with upwards velocity)
				if fall == "1" {
					if yv, ok := ctrl.Values["yvelocity"]; ok && strings.HasPrefix(yv, "-") {
						classified["launcher"] = append(classified["launcher"], id)
					}
				}
			}
		}

		// Heuristic Fallbacks for Anti-Air and Sweeps based on standard Marvel/Capcom/SNK MUGEN mapping
		if id == 420 || id == 450 || id == 430 {
			classified["antiair"] = append(classified["antiair"], id)
		}
		if id == 440 || id == 450 {
			if !contains(classified["low"], id) {
				classified["low"] = append(classified["low"], id)
			}
		}
		if id == 800 || id == 810 {
			if !contains(classified["grab"], id) {
				classified["grab"] = append(classified["grab"], id)
			}
		}
	}

	return classified
}

func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

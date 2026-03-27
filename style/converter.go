package style

import (
	"fmt"
	"strings"
)

// ConvertToMugen transforms an AIStyle into MUGEN [State -1] code blocks.
// This is the final conversion step — the style has already been parsed,
// validated, and safety-checked before reaching this point.
//
// Parameters:
//   - s: The validated AIStyle to convert
//   - aiActivation: The AI condition string (e.g., "AILevel > 0")
//   - cooldownVar: var() index for cooldown timer
//   - trackVar: var() index for last-state tracking
//   - behaviorVar: var() index for mid-match behavior switching (0 if unused)
func ConvertToMugen(s *AIStyle, aiActivation string, cooldownVar, trackVar, behaviorVar int) string {
	var sb strings.Builder

	// Style header comment
	sb.WriteString(fmt.Sprintf(`
; -------------------------------------------------------
; AI STYLE: %s
; %s
; Decisions: %d | Reaction Delay: %d ticks
; -------------------------------------------------------
`, s.Name, s.Description, len(s.Decisions), s.ReactionDelay))

	// Mid-match switching code (if enabled)
	if s.MidMatchSwitch && behaviorVar > 0 {
		sb.WriteString(GenerateMidMatchSwitchCode(behaviorVar, aiActivation))
	}

	// Convert each decision to a ChangeState controller
	for _, d := range s.Decisions {
		sb.WriteString(convertDecision(d, s, aiActivation, cooldownVar, trackVar, behaviorVar))
	}

	return sb.String()
}

// convertDecision generates MUGEN code for a single AIDecision.
func convertDecision(d AIDecision, s *AIStyle, aiActivation string, cooldownVar, trackVar, behaviorVar int) string {
	var sb strings.Builder

	label := d.Label
	if label == "" {
		label = fmt.Sprintf("Style %s P%d", d.Strategy, d.Priority)
	}

	// --- ChangeState controller ---
	sb.WriteString(fmt.Sprintf("\n; --- %s (State %d, %s) ---\n", label, d.State, d.Strategy))
	sb.WriteString(fmt.Sprintf("[State -1, AI Style %s: %s]\n", s.Name, label))
	sb.WriteString("type = ChangeState\n")
	sb.WriteString(fmt.Sprintf("value = %d\n", d.State))

	// Mandatory AI activation gate
	sb.WriteString(fmt.Sprintf("triggerall = %s\n", aiActivation))
	sb.WriteString("triggerall = RoundState = 2\n")

	// Anti-spam cooldown check
	sb.WriteString(fmt.Sprintf("triggerall = var(%d) = 0\n", cooldownVar))

	// Prevent same-state repetition
	sb.WriteString(fmt.Sprintf("triggerall = var(%d) != %d\n", trackVar, d.State))

	// Randomization gate
	if d.RandWeight > 0 && d.RandWeight < 1000 {
		sb.WriteString(fmt.Sprintf("triggerall = Random < %d\n", d.RandWeight))
	}

	// Reaction delay (simulated human reaction time)
	if s.ReactionDelay > 0 {
		sb.WriteString(fmt.Sprintf("triggerall = GameTime %% %d = 0\n", s.ReactionDelay+1))
	}

	// Mid-match behavior modifier (Phase 4 Bitmask: 1=Rush, 2=Def, 4=Grap, 8=Zone)
	if s.MidMatchSwitch && behaviorVar > 0 {
		switch d.Strategy {
		case "combo", "rushdown", "aggressive":
			sb.WriteString(fmt.Sprintf("triggerall = (var(%d) = 0) || ((var(%d) & 1) = 1)\n", behaviorVar, behaviorVar))
		case "defense", "defensive", "counter":
			sb.WriteString(fmt.Sprintf("triggerall = (var(%d) = 0) || ((var(%d) & 2) = 2)\n", behaviorVar, behaviorVar))
		case "throw", "grappler":
			sb.WriteString(fmt.Sprintf("triggerall = (var(%d) = 0) || ((var(%d) & 4) = 4)\n", behaviorVar, behaviorVar))
		case "projectile", "zoner", "zoning":
			sb.WriteString(fmt.Sprintf("triggerall = (var(%d) = 0) || ((var(%d) & 8) = 8)\n", behaviorVar, behaviorVar))
		}
	}

	// User-defined conditions from the .zss file
	for i, cond := range d.Conditions {
		sb.WriteString(fmt.Sprintf("trigger1 = %s\n", cond))
		_ = i // All conditions are ANDed under trigger1
	}

	// If no conditions specified, use ctrl as the base trigger
	if len(d.Conditions) == 0 {
		sb.WriteString("trigger1 = ctrl\n")
	}

	sb.WriteString("\n")

	// --- Cooldown setter ---
	sb.WriteString(fmt.Sprintf("[State -1, Style Cooldown: %s]\n", label))
	sb.WriteString("type = VarSet\n")
	sb.WriteString(fmt.Sprintf("triggerall = %s\n", aiActivation))
	sb.WriteString(fmt.Sprintf("trigger1 = StateNo = %d\n", d.State))
	sb.WriteString("trigger1 = Time = 0\n")
	sb.WriteString(fmt.Sprintf("v = %d\n", cooldownVar))
	sb.WriteString(fmt.Sprintf("value = %d\n", d.Cooldown))
	sb.WriteString("\n")

	// --- State tracker ---
	sb.WriteString(fmt.Sprintf("[State -1, Style Track: %s]\n", label))
	sb.WriteString("type = VarSet\n")
	sb.WriteString(fmt.Sprintf("triggerall = %s\n", aiActivation))
	sb.WriteString(fmt.Sprintf("trigger1 = StateNo = %d\n", d.State))
	sb.WriteString("trigger1 = Time = 0\n")
	sb.WriteString(fmt.Sprintf("v = %d\n", trackVar))
	sb.WriteString(fmt.Sprintf("value = %d\n", d.State))
	sb.WriteString("\n")

	return sb.String()
}

// ConvertBlendToMugen converts a StyleBlend result into MUGEN code.
// The blend is first merged via Blend(), then converted normally.
func ConvertBlendToMugen(blend *StyleBlend, aiActivation string, cooldownVar, trackVar, behaviorVar int) string {
	merged := blend.Blend()
	return ConvertToMugen(merged, aiActivation, cooldownVar, trackVar, behaviorVar)
}

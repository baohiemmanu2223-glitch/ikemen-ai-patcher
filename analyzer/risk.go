package analyzer

import (
	"fmt"
	"strings"

	"ikemen-ai-patcher/parser"
)

// AnalyzeRisk identifies potentially unsafe states in the character data.
// Unsafe states include: long recovery without guard, hypers without invincibility,
// attacks that leave the character in a vulnerable position.
func AnalyzeRisk(data *parser.CharacterData) []RiskEntry {
	var risks []RiskEntry

	for id, state := range data.States {
		if id < 0 {
			continue
		}

		// Only analyze attack states
		if state.Def.MoveType != "" && !isAttack(state.Def.MoveType) {
			continue
		}

		// Risk: Attack state with no guard fallback
		if isAttack(state.Def.MoveType) && !hasGuardTransition(state) {
			// Not all states need this, but specials/hypers should ideally have it
			if id >= 1000 {
				risks = append(risks, RiskEntry{
					StateID:  id,
					Reason:   "No guard transition — vulnerable during recovery",
					Severity: "medium",
				})
			}
		}

		// Risk: Hyper without invincibility (NotHitBy/HitOverride)
		if id >= 3000 && id < 5000 {
			if !hasInvincibility(state) {
				risks = append(risks, RiskEntry{
					StateID:  id,
					Reason:   "Super without invincibility (no NotHitBy/HitOverride)",
					Severity: "low",
				})
			}
		}

		// Risk: State has ChangeState to itself (potential infinite loop)
		targets := state.GetChangeStateTargets()
		for _, t := range targets {
			if t == id {
				risks = append(risks, RiskEntry{
					StateID:  id,
					Reason:   fmt.Sprintf("Self-referencing ChangeState (state %d -> %d)", id, t),
					Severity: "high",
				})
			}
		}

		// Risk: Attack with no ctrl recovery
		if isAttack(state.Def.MoveType) && state.Def.Ctrl == "0" {
			hasCtrlSet := false
			for _, ctrl := range state.Controllers {
				if strings.EqualFold(ctrl.Type, "CtrlSet") {
					hasCtrlSet = true
					break
				}
			}
			if !hasCtrlSet && !hasChangeStateOut(state) {
				risks = append(risks, RiskEntry{
					StateID:  id,
					Reason:   "Attack state with ctrl=0 and no CtrlSet or exit transition",
					Severity: "high",
				})
			}
		}
	}

	return risks
}

func isAttack(moveType string) bool {
	return len(moveType) > 0 && (moveType[0] == 'A' || moveType[0] == 'a')
}

func hasGuardTransition(state *parser.State) bool {
	for _, ctrl := range state.Controllers {
		if strings.EqualFold(ctrl.Type, "ChangeState") {
			for _, t := range ctrl.Triggers {
				lower := strings.ToLower(t.Raw)
				if strings.Contains(lower, "inhitpause") ||
					strings.Contains(lower, "gethitvar") {
					return true
				}
			}
		}
	}
	return false
}

func hasInvincibility(state *parser.State) bool {
	for _, ctrl := range state.Controllers {
		ctype := strings.ToLower(ctrl.Type)
		if ctype == "nothitby" || ctype == "hitoverride" || ctype == "hitby" {
			return true
		}
	}
	return false
}

func hasChangeStateOut(state *parser.State) bool {
	for _, ctrl := range state.Controllers {
		if strings.EqualFold(ctrl.Type, "ChangeState") {
			val, ok := ctrl.Values["value"]
			if ok {
				target := parseSimpleInt(val)
				if target != state.Def.ID {
					return true
				}
			}
		}
	}
	return false
}

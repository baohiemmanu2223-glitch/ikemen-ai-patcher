package ai

import (
	"fmt"
	"strings"

	"ikemen-ai-patcher/analyzer"
)

// GenerateAI creates the complete boss-level AI code blocks based on character analysis.
// Returns AI blocks for injection into .cmd (State -1 logic).
//
// 3-Layer Architecture:
//   Layer 1: Situation Awareness (distance, enemy state, advantage)
//   Layer 2: Strategy Selection (OFFENSE / DEFENSE / PUNISH / BAIT)
//   Layer 3: Action Selection (ChangeState based on strategy + analysis results)
func GenerateAI(analysis *analyzer.Analysis) []AIBlock {
	// Build AI config with unused vars
	config := buildConfig(analysis)

	var blocks []AIBlock

	// Generate the main CMD AI block (State -1 decision logic)
	cmdBlock := generateCmdBlock(config)
	blocks = append(blocks, AIBlock{
		Section: "cmd",
		Content: cmdBlock,
		Label:   "AI Decision Logic (State -1)",
	})

	blocks = append(blocks, AIBlock{
		Section: "cns",
		Content: generateCnsHelpers(config),
		Label:   "AI Background Helpers",
	})

	return blocks
}

// GenerateAIWithStyle creates AI code that incorporates style-based decisions.
// The styleCode has already been converted from the style system's internal model
// into valid MUGEN format. This function wraps it with infrastructure
// (cooldown, strategy) and falls back to auto-generation for uncovered strategies.
func GenerateAIWithStyle(analysis *analyzer.Analysis, styleName, styleCode string) []AIBlock {
	config := buildConfig(analysis)
	config.StyleName = styleName
	config.StyleCode = styleCode

	var blocks []AIBlock

	cmdBlock := generateCmdBlock(config)
	blocks = append(blocks, AIBlock{
		Section: "cmd",
		Content: cmdBlock,
		Label:   fmt.Sprintf("AI Style: %s (State -1)", styleName),
	})

	blocks = append(blocks, AIBlock{
		Section: "cns",
		Content: generateCnsHelpers(config),
		Label:   "AI Background Helpers",
	})

	return blocks
}

// GetConfig returns a populated AIConfig for external use by the style system.
func GetConfig(analysis *analyzer.Analysis) *AIConfig {
	return buildConfig(analysis)
}

// buildConfig assigns unused variables for AI use.
func buildConfig(analysis *analyzer.Analysis) *AIConfig {
	config := &AIConfig{
		Analysis:       analysis,
		AIActivation:   "AILevel > 0",
		VarAllocation:  analysis.VarAllocation,
		FvarAllocation: analysis.FvarAllocation,
	}

	// Assign unused vars (analyzer found these)
	unused := analysis.UnusedVars
	if len(unused) >= 5 {
		config.CooldownVar = unused[0]
		config.StrategyVar = unused[1]
		config.ComboTrackVar = unused[2]
		config.RandSeedVar = unused[3]
		config.StateTrackVar = unused[4]
	} else {
		// Fallback: use high-numbered vars that are commonly free
		config.CooldownVar = 55
		config.StrategyVar = 56
		config.ComboTrackVar = 57
		config.RandSeedVar = 58
		config.StateTrackVar = 50
	}

	return config
}

// generateCmdBlock builds the complete State -1 AI injection code.
// If a style is provided (config.StyleCode != ""), it uses the style's decisions
// as the primary action layer and adds auto-generated fallbacks for movement/guard.
func generateCmdBlock(config *AIConfig) string {
	var sb strings.Builder

	// --- Header ---
	sb.WriteString(aiHeader(config))

	// --- Layer 0: Infrastructure (always present) ---
	sb.WriteString(cooldownTimerBlock(config))
	sb.WriteString(randomSeedBlock(config))
	sb.WriteString(memoryInitBlock(config))

	if config.StyleCode != "" {
		// ===== STYLE-BASED GENERATION =====
		sb.WriteString(fmt.Sprintf("\n; === AI Style: %s ===\n", config.StyleName))

		// Strategy selection still runs for fallback guard/movement
		sb.WriteString(strategyBlock(config))

		// Inject style-converted decisions (primary action layer)
		sb.WriteString(config.StyleCode)

		// Fallback: guard and movement still auto-generated
		sb.WriteString(guardBlock(config))
		sb.WriteString(movementBlock(config))

	} else {
		// ===== AUTO-GENERATED (no style) =====
		// --- Layer 2: Strategy Selection ---
		sb.WriteString(strategyBlock(config))

		// --- Layer 3: Actions ---
		sb.WriteString(generatePunishActions(config))
		sb.WriteString(generateComboActions(config))
		sb.WriteString(generateOffenseActions(config))
		sb.WriteString(guardBlock(config))
		sb.WriteString(generateBaitActions(config))
		sb.WriteString(movementBlock(config))
	}

	// --- Footer ---
	sb.WriteString(aiFooter())

	return sb.String()
}

// generatePunishActions creates punish-priority attack controllers.
// Uses fast normals and specials when enemy is in recovery.
func generatePunishActions(config *AIConfig) string {
	var sb strings.Builder
	sb.WriteString(`
; -------------------------------------------------------
; PRIORITY 1: PUNISH ACTIONS
; Used when enemy whiffs or is in recovery.
; Frame-aware: checks EnemyNear, GetHitVar(HitTime)
; -------------------------------------------------------
`)

	analysis := config.Analysis

	// Use fastest normals for punishing (200-series standing normals)
	punishNormals := filterRange(analysis.Normals, 200, 299)
	if len(punishNormals) == 0 {
		punishNormals = analysis.Normals // Fallback to any normals
	}

	for i, stateID := range punishNormals {
		if i >= 3 { // Limit to top 3 punish normals
			break
		}
		randThreshold := 700 + i*100 // High chance for first, descending
		sb.WriteString(generateAttackBlock(
			config, stateID,
			fmt.Sprintf("Punish Normal %d", stateID),
			"PUNISH",
			"P2BodyDist X < 80",                            // Close range
			"EnemyNear, GetHitVar(HitTime) <= 0",           // Enemy not in hitstun (whiffed)
			15,            // Short cooldown for punishes
			randThreshold, // Randomization
		))
	}

	// Punish with specials if in range
	for i, stateID := range analysis.Specials {
		if i >= 2 { // Limit to 2 punish specials
			break
		}
		sb.WriteString(generateAttackBlock(
			config, stateID,
			fmt.Sprintf("Punish Special %d", stateID),
			"PUNISH",
			"P2BodyDist X < 150",
			"EnemyNear, AnimTime < -8",
			30, // Longer cooldown for specials
			500,
		))
	}

	return sb.String()
}

// generateComboActions creates combo follow-up controllers.
// Links combo chains discovered by the analyzer.
func generateComboActions(config *AIConfig) string {
	var sb strings.Builder
	sb.WriteString(`
; -------------------------------------------------------
; PRIORITY 2: COMBO FOLLOW-UPS
; Chains moves on hit confirmation (MoveContact).
; Uses analyzed combo chains from character data.
; -------------------------------------------------------
`)

	analysis := config.Analysis

	for _, chain := range analysis.ComboChains {
		if len(chain.States) < 2 {
			continue
		}
		// Link each state in the chain to the next
		for j := 0; j < len(chain.States)-1; j++ {
			from := chain.States[j]
			to := chain.States[j+1]
			sb.WriteString(comboFollowupBlock(
				config, from, to,
				fmt.Sprintf("Chain %d->%d", from, to),
			))
		}
	}

	// If no chains detected, create basic normal->special links
	if len(analysis.ComboChains) == 0 && len(analysis.Normals) > 0 && len(analysis.Specials) > 0 {
		sb.WriteString("\n; No explicit combo chains detected — generating basic links\n")
		// Link first normal to first special
		from := analysis.Normals[0]
		to := analysis.Specials[0]
		sb.WriteString(comboFollowupBlock(config, from, to,
			fmt.Sprintf("Auto-link Normal->Special %d->%d", from, to),
		))
	}

	return sb.String()
}

// generateOffenseActions creates neutral/offensive attack controllers.
// Selects moves based on distance and randomization.
func generateOffenseActions(config *AIConfig) string {
	var sb strings.Builder
	sb.WriteString(`
; -------------------------------------------------------
; PRIORITY 3: OFFENSE ACTIONS
; Neutral game attacks based on distance.
; Layer 1 (Situation Awareness) feeds into action selection.
; -------------------------------------------------------
`)

	analysis := config.Analysis

	// Close range: light normals (200-series)
	closeNormals := filterRange(analysis.Normals, 200, 250)
	for i, stateID := range closeNormals {
		if i >= 2 {
			break
		}
		
		dist := "P2BodyDist X < 60"
		if xvel, ok := config.FvarAllocation["EnemyXVel"]; ok {
			dist = fmt.Sprintf("P2BodyDist X = [-5, 60 + floor(10 * fvar(%d))]", xvel)
		}

		extra := "StateType = S"
		if frame, ok := analysis.FrameData[stateID]; ok && frame.Classification == "Unsafe" {
			extra += "\ntriggerall = MoveHit = 1 || EnemyNear, ctrl = 0 ; Unsafe confirm"
		}

		sb.WriteString(generateAttackBlock(
			config, stateID,
			fmt.Sprintf("Close Normal %d", stateID),
			"OFFENSE",
			dist,
			extra,
			10,
			600-i*100,
		))
	}

	// Mid range: heavy normals, crouching attacks
	midNormals := filterRange(analysis.Normals, 210, 299)
	midNormals = append(midNormals, filterRange(analysis.Normals, 400, 499)...)
	for i, stateID := range midNormals {
		if i >= 3 {
			break
		}
		sb.WriteString(generateAttackBlock(
			config, stateID,
			fmt.Sprintf("Mid Normal %d", stateID),
			"OFFENSE",
			"P2BodyDist X = [40, 120]",
			"",
			12,
			500-i*80,
		))
	}

	// Far range: specials (projectiles, long-range)
	for i, stateID := range analysis.Specials {
		if i >= 3 {
			break
		}
		sb.WriteString(generateAttackBlock(
			config, stateID,
			fmt.Sprintf("Offensive Special %d", stateID),
			"OFFENSE",
			"P2BodyDist X = [80, 250]",
			"",
			25,
			400-i*80,
		))
	}

	// Hypers: use when power >= 1000 (more conservative)
	for i, stateID := range analysis.Hypers {
		if i >= 2 {
			break
		}
		powerReq := 1000
		if i > 0 {
			powerReq = 2000
		}
		sb.WriteString(generateAttackBlock(
			config, stateID,
			fmt.Sprintf("Super %d", stateID),
			"OFFENSE",
			"P2BodyDist X < 200",
			fmt.Sprintf("Power >= %d", powerReq),
			60, // Long cooldown for supers
			300,
		))
	}

	return sb.String()
}

// generateBaitActions creates BAIT strategy controllers.
// Uses feints and deliberate whiffs to condition the opponent.
func generateBaitActions(config *AIConfig) string {
	var sb strings.Builder
	sb.WriteString(`
; -------------------------------------------------------
; PRIORITY 5: BAIT ACTIONS
; Deliberate spacing and baiting behavior.
; Creates openings for punish on next cycle.
; -------------------------------------------------------
`)

	analysis := config.Analysis

	// Use a light normal at just-outside range (whiff punish setup)
	if len(analysis.Normals) > 0 {
		stateID := analysis.Normals[0]
		sb.WriteString(generateAttackBlock(
			config, stateID,
			fmt.Sprintf("Bait Whiff %d", stateID),
			"BAIT",
			"P2BodyDist X = [100, 180]", // Deliberately outside of range
			"",
			20,
			300,
		))
	}

	// Crouch to bait jumps
	sb.WriteString(fmt.Sprintf(`
; Bait crouch (fake low threat)
[State -1, AI Bait Crouch]
type = ChangeState
value = 11
triggerall = %s
triggerall = RoundState = 2
triggerall = var(%d) = 3
triggerall = ctrl
triggerall = StateType = S
trigger1 = P2BodyDist X = [80, 200]
trigger1 = Random < 250

`, config.AIActivation, config.StrategyVar))

	return sb.String()
}

// filterRange returns state IDs within [lo, hi] from a sorted list.
func filterRange(states []int, lo, hi int) []int {
	var result []int
	for _, id := range states {
		if id >= lo && id <= hi {
			result = append(result, id)
		}
	}
	return result
}

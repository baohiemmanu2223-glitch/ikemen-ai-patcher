// Package ai generates boss-level AI logic for Ikemen GO characters.
package ai

import (
	"ikemen-ai-patcher/analyzer"
)

// AIConfig holds configuration for the AI generator.
type AIConfig struct {
	// Variable assignments (found by analyzer as unused)
	CooldownVar   int // var(N) used for anti-spam cooldown timer
	StrategyVar   int // var(N) used for current strategy mode
	ComboTrackVar int // var(N) used for combo state tracking
	RandSeedVar   int // var(N) used for randomization seed
	StateTrackVar int // var(N) used for last-used state tracking

	// AI activation condition
	AIActivation string // e.g., "var(59) = 1" or "AILevel > 0"

	// Style system integration
	StyleName string // Name of AI style applied (empty = auto-generated)
	StyleCode string // Pre-converted MUGEN code from style system

	// Phase 4: Memory AI Maps
	VarAllocation  map[string]int
	FvarAllocation map[string]int

	// Character analysis results
	Analysis *analyzer.Analysis
}

// AIBlock represents a generated block of AI code to be injected.
type AIBlock struct {
	Section string // "cmd" or "cns"
	Content string // The generated MUGEN-format code
	Label   string // Human-readable label for the block
}

// Strategy represents one of the 4 AI strategies.
type Strategy int

const (
	StrategyOFFENSE Strategy = iota
	StrategyDEFENSE
	StrategyPUNISH
	StrategyBAIT
)

// String returns the strategy name.
func (s Strategy) String() string {
	switch s {
	case StrategyOFFENSE:
		return "OFFENSE"
	case StrategyDEFENSE:
		return "DEFENSE"
	case StrategyPUNISH:
		return "PUNISH"
	case StrategyBAIT:
		return "BAIT"
	default:
		return "UNKNOWN"
	}
}

// Distance represents range categories.
type Distance int

const (
	DistClose Distance = iota // < 50 pixels
	DistMid                   // 50-150 pixels
	DistFar                   // > 150 pixels
)

// EnemyState represents detected enemy conditions.
type EnemyState int

const (
	EnemyIdle    EnemyState = iota
	EnemyAttack
	EnemyHitstun
	EnemyAir
)

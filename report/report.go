// Package report generates human-readable AI analysis reports.
package report

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"ikemen-ai-patcher/analyzer"
	"ikemen-ai-patcher/utils"
	"ikemen-ai-patcher/validator"
)

// StyleInfo holds optional style information for the report.
type StyleInfo struct {
	Name      string
	Decisions []StyleDecisionInfo
	BlendInfo string // e.g., "aggressive(60%) + defensive(40%)"
}

// StyleDecisionInfo is a simplified view of a style decision for reporting.
type StyleDecisionInfo struct {
	Priority int
	Strategy string
	State    int
	Label    string
}

// GenerateReport creates a detailed AI analysis report file.
func GenerateReport(analysis *analyzer.Analysis, validation *validator.ValidationResult, outputDir string) (string, error) {
	return GenerateReportWithStyle(analysis, validation, nil, outputDir)
}

// GenerateReportWithStyle creates a report that optionally includes style information.
func GenerateReportWithStyle(analysis *analyzer.Analysis, validation *validator.ValidationResult, styleInfo *StyleInfo, outputDir string) (string, error) {
	reportPath := filepath.Join(outputDir, "ai_report.txt")
	content := buildReport(analysis, validation, styleInfo)

	if err := utils.WriteFile(reportPath, content); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}

	return reportPath, nil
}

// buildReport constructs the report content string.
func buildReport(analysis *analyzer.Analysis, validation *validator.ValidationResult, styleInfo *StyleInfo) string {
	var sb strings.Builder

	// Header
	sb.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║           IKEMEN-AI-PATCHER — AI ANALYSIS REPORT           ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════╝\n\n")

	sb.WriteString(fmt.Sprintf("  Character:    %s\n", analysis.CharName))
	sb.WriteString(fmt.Sprintf("  Total States: %d\n\n", analysis.TotalStates))

	// --- AI Style Applied ---
	if styleInfo != nil {
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("  AI STYLE APPLIED\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		sb.WriteString(fmt.Sprintf("  Style Name: %s\n", styleInfo.Name))
		if styleInfo.BlendInfo != "" {
			sb.WriteString(fmt.Sprintf("  Blend:      %s\n", styleInfo.BlendInfo))
		}
		sb.WriteString("\n  Decisions:\n")
		for _, d := range styleInfo.Decisions {
			sb.WriteString(fmt.Sprintf("    P%d [%s] → State %d (%s)\n",
				d.Priority, d.Strategy, d.State, d.Label))
		}
		sb.WriteString("\n")
	}

	// --- States Used ---
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("  STATES USED\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	sb.WriteString(fmt.Sprintf("  Normals  (%d): %s\n", len(analysis.Normals), intSliceStr(analysis.Normals)))
	sb.WriteString(fmt.Sprintf("  Specials (%d): %s\n", len(analysis.Specials), intSliceStr(analysis.Specials)))
	sb.WriteString(fmt.Sprintf("  Hypers   (%d): %s\n\n", len(analysis.Hypers), intSliceStr(analysis.Hypers)))

	// --- Combo Chains ---
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("  COMBO CHAINS\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(analysis.ComboChains) == 0 {
		sb.WriteString("  No explicit combo chains detected.\n")
		sb.WriteString("  AI will generate basic normal -> special links.\n\n")
	} else {
		for i, chain := range analysis.ComboChains {
			sb.WriteString(fmt.Sprintf("  Chain %d: %s\n", i+1, chain.Description))
			sb.WriteString(fmt.Sprintf("           States: %s\n\n", intSliceStr(chain.States)))
		}
	}

	// --- Variables ---
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("  VARIABLES\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	sb.WriteString(fmt.Sprintf("  var()  in use (%d): %s\n", len(analysis.Vars), intSliceStr(analysis.Vars)))
	sb.WriteString(fmt.Sprintf("  fvar() in use (%d): %s\n", len(analysis.Fvars), intSliceStr(analysis.Fvars)))
	sb.WriteString(fmt.Sprintf("  sysvar() in use (%d): %s\n\n", len(analysis.Sysvars), intSliceStr(analysis.Sysvars)))

	sb.WriteString(fmt.Sprintf("  Unused vars allocated for AI: %s\n\n", intSliceStr(analysis.UnusedVars)))

	// --- AI Style Summary ---
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("  AI STYLE SUMMARY\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	sb.WriteString(deriveStyle(analysis))
	sb.WriteString("\n")

	// --- Risk Analysis ---
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("  RISK ANALYSIS\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(analysis.RiskStates) == 0 {
		sb.WriteString("  No risk issues detected. ✓\n\n")
	} else {
		for _, risk := range analysis.RiskStates {
			icon := "⚠"
			if risk.Severity == "high" {
				icon = "🔴"
			} else if risk.Severity == "low" {
				icon = "🟡"
			}
			sb.WriteString(fmt.Sprintf("  %s State %d [%s]: %s\n", icon, risk.StateID, risk.Severity, risk.Reason))
		}
		sb.WriteString("\n")
	}

	// --- Validation ---
	if validation != nil {
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("  VALIDATION RESULTS\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		if validation.Valid {
			sb.WriteString("  ✅ All checks passed — safe to patch.\n\n")
		} else {
			sb.WriteString("  ❌ Validation FAILED:\n")
			for _, e := range validation.Errors {
				sb.WriteString(fmt.Sprintf("    ERROR: %s\n", e))
			}
			sb.WriteString("\n")
		}

		if len(validation.Warnings) > 0 {
			sb.WriteString("  Warnings:\n")
			for _, w := range validation.Warnings {
				sb.WriteString(fmt.Sprintf("    ⚠ %s\n", w))
			}
			sb.WriteString("\n")
		}
	}

	// Footer
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("  Generated by ikemen-ai-patcher v2.0\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return sb.String()
}

// deriveStyle creates a text summary of the AI's generated playstyle.
func deriveStyle(a *analyzer.Analysis) string {
	var sb strings.Builder

	totalAttacks := len(a.Normals) + len(a.Specials) + len(a.Hypers)
	if totalAttacks == 0 {
		sb.WriteString("  Archetype:     MINIMAL (few attack states found)\n")
		sb.WriteString("  The AI will focus on movement and basic interactions.\n")
		return sb.String()
	}

	hasProjectiles := false
	for _, id := range a.Specials {
		if id >= 1000 && id < 1500 {
			hasProjectiles = true
			break
		}
	}

	if len(a.Hypers) >= 3 && len(a.Specials) >= 3 {
		sb.WriteString("  Archetype:     RUSHDOWN / AGGRESSIVE\n")
		sb.WriteString("  Description:   Large moveset with multiple supers.\n")
		sb.WriteString("                 AI will aggressively combo into supers.\n")
	} else if hasProjectiles && len(a.Normals) > 4 {
		sb.WriteString("  Archetype:     BALANCED / ALL-ROUNDER\n")
		sb.WriteString("  Description:   Good mix of normals and specials.\n")
		sb.WriteString("                 AI will adapt between zoning and pressure.\n")
	} else if len(a.Normals) > 6 {
		sb.WriteString("  Archetype:     FOOTSIES / NEUTRAL\n")
		sb.WriteString("  Description:   Heavy normal-based game.\n")
		sb.WriteString("                 AI will focus on spacing and pokes.\n")
	} else {
		sb.WriteString("  Archetype:     COMPACT / EFFICIENT\n")
		sb.WriteString("  Description:   Focused moveset.\n")
		sb.WriteString("                 AI will use available tools precisely.\n")
	}

	sb.WriteString(fmt.Sprintf("  Combo Potential: %d chain(s) detected\n", len(a.ComboChains)))

	if len(a.Hypers) > 0 {
		sb.WriteString(fmt.Sprintf("  Super Usage:   %d supers available — AI will use with power management\n", len(a.Hypers)))
	}

	return sb.String()
}

func intSliceStr(nums []int) string {
	if len(nums) == 0 {
		return "(none)"
	}
	sorted := make([]int, len(nums))
	copy(sorted, nums)
	sort.Ints(sorted)

	parts := make([]string, len(sorted))
	for i, n := range sorted {
		parts[i] = fmt.Sprintf("%d", n)
	}
	return strings.Join(parts, ", ")
}

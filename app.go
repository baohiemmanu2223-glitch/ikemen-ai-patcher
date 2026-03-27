package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	aiPkg "ikemen-ai-patcher/ai"
	"ikemen-ai-patcher/analyzer"
	"ikemen-ai-patcher/parser"
	"ikemen-ai-patcher/patcher"
	"ikemen-ai-patcher/report"
	"ikemen-ai-patcher/style"
	"ikemen-ai-patcher/validator"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct — bound to the frontend via Wails.
// All methods here are callable from JavaScript as window.go.main.App.MethodName()
type App struct {
	ctx context.Context
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// startup is called when the Wails app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ─────────────────────────────────────────────
//  API 1: Select Character Folder
// ─────────────────────────────────────────────

// SelectCharacter opens a native folder dialog and returns the selected path.
func (a *App) SelectCharacter() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Character Folder",
	})
	if err != nil {
		return ""
	}
	return dir
}

// ─────────────────────────────────────────────
//  API 2: Analyze Character
// ─────────────────────────────────────────────

// AnalysisResult is the JSON-friendly analysis output sent to the frontend.
type AnalysisResult struct {
	Success     bool              `json:"success"`
	Error       string            `json:"error,omitempty"`
	CharName    string            `json:"charName"`
	TotalStates int               `json:"totalStates"`
	FileCount   int               `json:"fileCount"`
	Normals     []int             `json:"normals"`
	Specials    []int             `json:"specials"`
	Hypers      []int             `json:"hypers"`
	ComboChains []ComboChainInfo  `json:"comboChains"`
	Vars          []int               `json:"vars"`
	Fvars         []int               `json:"fvars"`
	Sysvars       []int               `json:"sysvars"`
	UnusedVars    []int               `json:"unusedVars"`
	UnusedFvars   []int               `json:"unusedFvars"`
	UnusedSysvars []int               `json:"unusedSysvars"`
	VarUsages     []analyzer.VarUsage `json:"varUsages"`
	RiskStates    []RiskInfo          `json:"riskStates"`
	Validation    ValidationInfo      `json:"validation"`
}

type ComboChainInfo struct {
	Description string `json:"description"`
	States      []int  `json:"states"`
}

type RiskInfo struct {
	StateID  int    `json:"stateId"`
	Severity string `json:"severity"`
	Reason   string `json:"reason"`
}

type ValidationInfo struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// AnalyzeCharacter parses, analyzes, and validates a character folder.
// Returns JSON result.
func (a *App) AnalyzeCharacter(charPath string) string {
	result := AnalysisResult{Success: true}

	// Parse
	data, err := parser.ParseCharacter(charPath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Parse error: %v", err)
		return toJSON(result)
	}

	// Analyze
	analysis := analyzer.Analyze(data)

	// Validate
	validation := validator.Validate(data, analysis)

	// Map results
	result.CharName = analysis.CharName
	result.TotalStates = analysis.TotalStates
	result.FileCount = len(data.Files)
	result.Normals = analysis.Normals
	result.Specials = analysis.Specials
	result.Hypers = analysis.Hypers
	result.Vars = analysis.Vars
	result.Fvars = analysis.Fvars
	result.Sysvars = analysis.Sysvars
	result.UnusedVars = analysis.UnusedVars
	result.UnusedFvars = analysis.UnusedFvars
	result.UnusedSysvars = analysis.UnusedSysvars
	result.VarUsages = analysis.VarUsages

	for _, chain := range analysis.ComboChains {
		result.ComboChains = append(result.ComboChains, ComboChainInfo{
			Description: chain.Description,
			States:      chain.States,
		})
	}

	for _, risk := range analysis.RiskStates {
		result.RiskStates = append(result.RiskStates, RiskInfo{
			StateID:  risk.StateID,
			Severity: risk.Severity,
			Reason:   risk.Reason,
		})
	}

	result.Validation = ValidationInfo{
		Valid:    validation.Valid,
		Errors:   validation.Errors,
		Warnings: validation.Warnings,
	}

	return toJSON(result)
}

// ─────────────────────────────────────────────
//  API 3: Load AI Styles
// ─────────────────────────────────────────────

// StyleInfo is the JSON-friendly style definition for the frontend.
type StyleListItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	FilePath    string `json:"filePath"`
	Decisions   int    `json:"decisions"`
	HasAdaptive bool   `json:"hasAdaptive"`
	Delay       int    `json:"reactionDelay"`
}

// LoadStyles scans the styles directory and returns available style definitions.
func (a *App) LoadStyles() string {
	stylesDir := findStylesDir()
	var items []StyleListItem

	entries, err := os.ReadDir(stylesDir)
	if err != nil {
		return toJSON(items)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".zss") {
			continue
		}
		fullPath := filepath.Join(stylesDir, entry.Name())
		s, err := style.ParseStyleFile(fullPath)
		if err != nil {
			continue
		}
		items = append(items, StyleListItem{
			Name:        s.Name,
			Description: s.Description,
			FilePath:    fullPath,
			Decisions:   len(s.Decisions),
			HasAdaptive: s.MidMatchSwitch,
			Delay:       s.ReactionDelay,
		})
	}

	return toJSON(items)
}

// PreviewStyle parses a single style file and returns detailed info.
func (a *App) PreviewStyle(stylePath string) string {
	s, err := style.ParseStyleFile(stylePath)
	if err != nil {
		return toJSON(map[string]string{"error": err.Error()})
	}
	return toJSON(map[string]interface{}{
		"name":        s.Name,
		"description": s.Description,
		"decisions":   len(s.Decisions),
		"summary":     style.FormatStyleSummary(s),
		"adaptive":    s.MidMatchSwitch,
		"delay":       s.ReactionDelay,
	})
}

// ─────────────────────────────────────────────
//  API 4: Apply AI Patch
// ─────────────────────────────────────────────

// PatchConfig is the config received from the frontend.
type PatchConfig struct {
	StylePath  string  `json:"stylePath"`
	AntiSpam   bool    `json:"antiSpam"`
	BossMode   bool    `json:"bossMode"`
	Randomness float64 `json:"randomness"` // 0.0 to 1.0
}

// PatchResult is the outcome of a patch operation.
type PatchResult struct {
	Success      bool     `json:"success"`
	Error        string   `json:"error,omitempty"`
	PatchedFiles []string `json:"patchedFiles"`
	BackupFiles  []string `json:"backupFiles"`
	StyleUsed    string   `json:"styleUsed"`
	ReportPath   string   `json:"reportPath"`
}

// ApplyPatch runs the AI patcher with an optional style.
func (a *App) ApplyPatch(charPath string, configJSON string) string {
	result := PatchResult{Success: true}

	// Parse config
	var cfg PatchConfig
	if configJSON != "" {
		json.Unmarshal([]byte(configJSON), &cfg)
	}

	// Parse character
	data, err := parser.ParseCharacter(charPath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Parse error: %v", err)
		return toJSON(result)
	}

	// Analyze
	analysis := analyzer.Analyze(data)

	// Validate
	validation := validator.Validate(data, analysis)
	if !validation.Valid {
		result.Success = false
		result.Error = "Validation failed: " + strings.Join(validation.Errors, "; ")
		return toJSON(result)
	}

	// Generate AI blocks
	var aiBlocks []aiPkg.AIBlock
	var styleInfo *report.StyleInfo

	if cfg.StylePath != "" {
		// Style-based patching
		s, err := style.ParseStyleFile(cfg.StylePath)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Style error: %v", err)
			return toJSON(result)
		}

		style.ExpandAbstractStates(s, analysis.Classified)
		styleValidation := style.ValidateStyle(s, data)
		if !styleValidation.Valid {
			result.Success = false
			result.Error = "Style incompatible: " + strings.Join(styleValidation.Errors, "; ")
			return toJSON(result)
		}

		style.EnsureSafety(s)
		config := aiPkg.GetConfig(analysis)
		behaviorVar := 0
		if s.MidMatchSwitch && len(analysis.UnusedVars) > 5 {
			behaviorVar = analysis.UnusedVars[len(analysis.UnusedVars)-1]
		}
		styleCode := style.ConvertToMugen(s, config.AIActivation,
			config.CooldownVar, config.StateTrackVar, behaviorVar)

		aiBlocks = aiPkg.GenerateAIWithStyle(analysis, s.Name, styleCode)
		result.StyleUsed = s.Name

		// Build style info for report
		info := &report.StyleInfo{Name: s.Name}
		for _, d := range s.Decisions {
			info.Decisions = append(info.Decisions, report.StyleDecisionInfo{
				Priority: d.Priority,
				Strategy: d.Strategy,
				State:    d.State,
				Label:    d.Label,
			})
		}
		styleInfo = info
	} else {
		// Auto-generated patching
		aiBlocks = aiPkg.GenerateAI(analysis)
		result.StyleUsed = "Auto-Generated"
	}

	// 4. Find CMD file
	scan, err := parser.ScanFolder(charPath)
	if err != nil || len(scan.CmdFiles) == 0 {
		result.Success = false
		result.Error = "No .cmd file found for patching"
		return toJSON(result)
	}

	cmdFile := scan.CmdFiles[0]
	cnsFile := ""
	if len(scan.CnsFiles) > 0 {
		cnsFile = scan.CnsFiles[0]
	}

	// Apply
	patchResult := patcher.ApplyPatch(cmdFile, cnsFile, aiBlocks)
	if !patchResult.Success {
		result.Success = false
		result.Error = "Patch failed: " + strings.Join(patchResult.Errors, "; ")
		return toJSON(result)
	}

	result.PatchedFiles = patchResult.PatchedFiles
	result.BackupFiles = patchResult.BackupFiles

	// Generate report
	reportPath, err := report.GenerateReportWithStyle(analysis, validation, styleInfo, charPath)
	if err == nil {
		result.ReportPath = reportPath
	}

	return toJSON(result)
}

// ─────────────────────────────────────────────
//  API 5: Generate Report
// ─────────────────────────────────────────────

// GenerateReport analyzes a character and returns the report text content.
func (a *App) GenerateReport(charPath string) string {
	data, err := parser.ParseCharacter(charPath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	analysis := analyzer.Analyze(data)
	validation := validator.Validate(data, analysis)

	reportPath, err := report.GenerateReport(analysis, validation, charPath)
	if err != nil {
		return fmt.Sprintf("Error generating report: %v", err)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Sprintf("Generated at: %s\n(file read error: %v)", reportPath, err)
	}

	return string(content)
}

// ─────────────────────────────────────────────
//  Helper
// ─────────────────────────────────────────────

func toJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

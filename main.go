// ikemen-ai-patcher v2.0 — Boss-Level AI Generator with Desktop UI
//
// HYBRID MODE:
//   GUI:  ikemen-ai-patcher.exe                     (launches desktop app)
//   CLI:  ikemen-ai-patcher.exe analyze <char>       (command-line mode)
//         ikemen-ai-patcher.exe patch <char>
//         ikemen-ai-patcher.exe style apply <char> <style>

package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	aiPkg "ikemen-ai-patcher/ai"
	"ikemen-ai-patcher/analyzer"
	"ikemen-ai-patcher/parser"
	"ikemen-ai-patcher/patcher"
	"ikemen-ai-patcher/report"
	"ikemen-ai-patcher/style"
	"ikemen-ai-patcher/validator"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend
var assets embed.FS

const version = "2.0.0"

func main() {
	// If CLI arguments are passed, run CLI mode
	if len(os.Args) >= 2 {
		cmd := strings.ToLower(os.Args[1])
		if isCLICommand(cmd) {
			runCLI()
			return
		}
	}

	// Launch Wails GUI
	launchGUI()
}

func isCLICommand(cmd string) bool {
	cliCommands := []string{
		"analyze", "patch", "report", "style",
		"version", "--version", "-v",
		"help", "--help", "-h", "--cli",
	}
	for _, c := range cliCommands {
		if cmd == c {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────
//  WAILS GUI LAUNCH
// ─────────────────────────────────────────────

func launchGUI() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Ikemen AI Patcher",
		Width:  1100,
		Height: 750,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 14, B: 23, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
	})

	if err != nil {
		fmt.Printf("Error launching GUI: %v\n", err)
		os.Exit(1)
	}
}

// ─────────────────────────────────────────────
//  CLI MODE (preserved from v2.0)
// ─────────────────────────────────────────────

func runCLI() {
	command := strings.ToLower(os.Args[1])

	switch command {
	case "analyze":
		requireArg(3, "analyze <char_folder>")
		cmdAnalyze(os.Args[2])
	case "patch":
		requireArg(3, "patch <char_folder>")
		cmdPatch(os.Args[2])
	case "report":
		requireArg(3, "report <char_folder>")
		cmdReport(os.Args[2])
	case "style":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ikemen-ai-patcher style <list|load|apply|blend>")
			os.Exit(1)
		}
		subCmd := strings.ToLower(os.Args[2])
		switch subCmd {
		case "list":
			cmdStyleList()
		case "load":
			requireArg(4, "style load <file.zss>")
			cmdStyleLoad(os.Args[3])
		case "apply":
			requireArg(5, "style apply <char_folder> <style.zss>")
			cmdStyleApply(os.Args[3], os.Args[4])
		case "blend":
			requireArg(6, "style blend <char_folder> <s1.zss> <s2.zss>")
			cmdStyleBlend(os.Args[3], os.Args[4], os.Args[5])
		default:
			fmt.Printf("Unknown: style %s\n", subCmd)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Printf("ikemen-ai-patcher v%s\n", version)
	case "help", "--help", "-h", "--cli":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func requireArg(n int, usage string) {
	if len(os.Args) < n {
		fmt.Printf("Error: missing arguments\nUsage: ikemen-ai-patcher %s\n", usage)
		os.Exit(1)
	}
}

// ─── CLI: Analyze ───
func cmdAnalyze(charPath string) {
	charPath = resolvePath(charPath)
	printHeader("ANALYZE MODE")

	fmt.Printf("📂 Parsing: %s\n", charPath)
	data, err := parser.ParseCharacter(charPath)
	exitOnErr(err, "Parse error")
	fmt.Printf("✅ Parsed %d files, found %d states\n\n", len(data.Files), len(data.States))

	fmt.Println("🔍 Running analysis...")
	analysis := analyzer.Analyze(data)

	fmt.Printf("\n  Character: %s\n  Total States: %d\n\n", analysis.CharName, analysis.TotalStates)
	fmt.Printf("  📋 Normals  (%d): %s\n", len(analysis.Normals), formatInts(analysis.Normals))
	fmt.Printf("  📋 Specials (%d): %s\n", len(analysis.Specials), formatInts(analysis.Specials))
	fmt.Printf("  📋 Hypers   (%d): %s\n\n", len(analysis.Hypers), formatInts(analysis.Hypers))

	fmt.Printf("  🔗 Combo Chains: %d\n", len(analysis.ComboChains))
	for i, chain := range analysis.ComboChains {
		fmt.Printf("     Chain %d: %s\n", i+1, chain.Description)
	}
	fmt.Println()

	fmt.Printf("  📊 Variables: var(%s) fvar(%s) sysvar(%s)\n",
		formatInts(analysis.Vars), formatInts(analysis.Fvars), formatInts(analysis.Sysvars))
	fmt.Printf("  🆓 Unused vars for AI: %s\n\n", formatInts(analysis.UnusedVars))

	fmt.Printf("  ⚠️  Risk States: %d\n", len(analysis.RiskStates))
	for _, risk := range analysis.RiskStates {
		fmt.Printf("     [%s] State %d: %s\n", risk.Severity, risk.StateID, risk.Reason)
	}
	fmt.Println()

	validation := validator.Validate(data, analysis)
	printValidation(validation)
}

// ─── CLI: Patch ───
func cmdPatch(charPath string) {
	charPath = resolvePath(charPath)
	printHeader("PATCH MODE")

	data, analysis, validation := parseAnalyzeValidate(charPath)
	if !validation.Valid {
		fmt.Println("❌ Validation FAILED — cannot patch safely")
		os.Exit(1)
	}

	fmt.Println("🤖 Generating boss-level AI...")
	aiBlocks := aiPkg.GenerateAI(analysis)
	fmt.Printf("✅ Generated %d AI block(s)\n", len(aiBlocks))

	applyAndReport(charPath, data, analysis, validation, aiBlocks, nil)
}

// ─── CLI: Report ───
func cmdReport(charPath string) {
	charPath = resolvePath(charPath)
	printHeader("REPORT MODE")

	fmt.Printf("📂 Parsing: %s\n", charPath)
	data, err := parser.ParseCharacter(charPath)
	exitOnErr(err, "Parse error")
	_ = data
	fmt.Println("🔍 Analyzing...")
	analysis := analyzer.Analyze(data)
	validation := validator.Validate(data, analysis)

	fmt.Println("📄 Generating report...")
	reportPath, err := report.GenerateReport(analysis, validation, charPath)
	exitOnErr(err, "Report error")
	fmt.Printf("\n✅ Report saved: %s\n", reportPath)
}

// ─── CLI: Style commands ───
func cmdStyleList() {
	printHeader("STYLE LIST")
	stylesDir := findStylesDir()
	entries, err := os.ReadDir(stylesDir)
	if err != nil {
		fmt.Printf("📂 Styles directory: %s (not found)\n", stylesDir)
		return
	}
	fmt.Printf("📂 Styles directory: %s\n\n", stylesDir)
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".zss") {
			continue
		}
		count++
		s, err := style.ParseStyleFile(filepath.Join(stylesDir, entry.Name()))
		if err != nil {
			fmt.Printf("  %d. %-20s  ❌ %v\n", count, entry.Name(), err)
			continue
		}
		fmt.Printf("  %d. %-20s  %s (%d decisions)\n", count, entry.Name(), s.Name, len(s.Decisions))
		if s.Description != "" {
			fmt.Printf("     %s\n", s.Description)
		}
	}
	if count == 0 {
		fmt.Println("  No .zss style files found.")
	}
	fmt.Println()
}

func cmdStyleLoad(stylePath string) {
	stylePath = resolvePath(stylePath)
	printHeader("STYLE LOAD")
	fmt.Printf("📂 Loading: %s\n\n", stylePath)
	s, err := style.ParseStyleFile(stylePath)
	exitOnErr(err, "Style parse error")
	fmt.Print(style.FormatStyleSummary(s))
	fmt.Println()
}

func cmdStyleApply(charPath, stylePath string) {
	charPath = resolvePath(charPath)
	stylePath = resolveStylePath(stylePath)
	printHeader("STYLE APPLY")

	fmt.Printf("🎨 Loading style: %s\n", stylePath)
	s, err := style.ParseStyleFile(stylePath)
	exitOnErr(err, "Style parse error")
	fmt.Printf("✅ Style loaded: %s (%d decisions)\n", s.Name, len(s.Decisions))

	data, analysis, validation := parseAnalyzeValidate(charPath)
	if !validation.Valid {
		fmt.Println("❌ Validation FAILED")
		os.Exit(1)
	}

	fmt.Println("🛡️  Validating style compatibility...")
	style.ExpandAbstractStates(s, analysis.Classified)
	sv := style.ValidateStyle(s, data)
	for _, w := range sv.Warnings {
		fmt.Printf("   ⚠ %s\n", w)
	}
	if !sv.Valid {
		fmt.Println("❌ Style validation FAILED")
		os.Exit(1)
	}
	fmt.Printf("✅ Style compatible (%d valid decisions)\n", len(s.Decisions))

	style.EnsureSafety(s)
	config := aiPkg.GetConfig(analysis)
	behaviorVar := 0
	if s.MidMatchSwitch && len(analysis.UnusedVars) > 5 {
		behaviorVar = analysis.UnusedVars[len(analysis.UnusedVars)-1]
	}
	styleCode := style.ConvertToMugen(s, config.AIActivation,
		config.CooldownVar, config.StateTrackVar, behaviorVar)

	aiBlocks := aiPkg.GenerateAIWithStyle(analysis, s.Name, styleCode)
	fmt.Printf("✅ Generated %d AI block(s)\n", len(aiBlocks))

	styleInfo := buildStyleInfo(s, "")
	applyAndReport(charPath, data, analysis, validation, aiBlocks, styleInfo)
}

func cmdStyleBlend(charPath, s1Path, s2Path string) {
	charPath = resolvePath(charPath)
	s1Path = resolveStylePath(s1Path)
	s2Path = resolveStylePath(s2Path)
	printHeader("STYLE BLEND")

	fmt.Printf("🎨 Loading: %s\n", s1Path)
	s1, err := style.ParseStyleFile(s1Path)
	exitOnErr(err, "Style 1 error")
	fmt.Printf("🎨 Loading: %s\n", s2Path)
	s2, err := style.ParseStyleFile(s2Path)
	exitOnErr(err, "Style 2 error")
	fmt.Printf("✅ Blending: %s + %s (50%%/50%%)\n\n", s1.Name, s2.Name)

	data, analysis, validation := parseAnalyzeValidate(charPath)
	if !validation.Valid {
		fmt.Println("❌ Validation FAILED")
		os.Exit(1)
	}

	style.ExpandAbstractStates(s1, analysis.Classified)
	style.ExpandAbstractStates(s2, analysis.Classified)
	style.ValidateStyle(s1, data)
	style.ValidateStyle(s2, data)
	blend := &style.StyleBlend{Styles: []*style.AIStyle{s1, s2}, Weights: []float64{0.5, 0.5}}
	merged := blend.Blend()
	style.EnsureSafety(merged)
	fmt.Printf("✅ Merged: %d decisions\n", len(merged.Decisions))

	config := aiPkg.GetConfig(analysis)
	behaviorVar := 0
	if merged.MidMatchSwitch && len(analysis.UnusedVars) > 5 {
		behaviorVar = analysis.UnusedVars[len(analysis.UnusedVars)-1]
	}
	styleCode := style.ConvertToMugen(merged, config.AIActivation,
		config.CooldownVar, config.StateTrackVar, behaviorVar)
	aiBlocks := aiPkg.GenerateAIWithStyle(analysis, merged.Name, styleCode)

	blendInfo := fmt.Sprintf("%s(50%%) + %s(50%%)", s1.Name, s2.Name)
	applyAndReport(charPath, data, analysis, validation, aiBlocks, buildStyleInfo(merged, blendInfo))
}

// ─── Shared CLI Helpers ───
func parseAnalyzeValidate(charPath string) (*parser.CharacterData, *analyzer.Analysis, *validator.ValidationResult) {
	fmt.Printf("📂 Parsing: %s\n", charPath)
	data, err := parser.ParseCharacter(charPath)
	exitOnErr(err, "Parse error")
	fmt.Printf("✅ Parsed %d files, %d states\n", len(data.Files), len(data.States))
	fmt.Println("🔍 Analyzing...")
	analysis := analyzer.Analyze(data)
	fmt.Println("🛡️  Validating...")
	validation := validator.Validate(data, analysis)
	for _, w := range validation.Warnings {
		fmt.Printf("   ⚠ %s\n", w)
	}
	if validation.Valid {
		fmt.Println("✅ Validation passed")
	}
	return data, analysis, validation
}

func applyAndReport(charPath string, data *parser.CharacterData, analysis *analyzer.Analysis,
	validation *validator.ValidationResult, aiBlocks []aiPkg.AIBlock, styleInfo *report.StyleInfo) {

	scan, err := parser.ScanFolder(charPath)
	if err != nil || len(scan.CmdFiles) == 0 {
		fmt.Println("❌ No .cmd file found")
		os.Exit(1)
	}
	cmdFile := scan.CmdFiles[0]
	cnsFile := ""
	if len(scan.CnsFiles) > 0 {
		cnsFile = scan.CnsFiles[0]
	}

	fmt.Printf("📝 Patching: %s\n", cmdFile)
	result := patcher.ApplyPatch(cmdFile, cnsFile, aiBlocks)
	if result.Success {
		fmt.Println("\n✅ PATCH SUCCESSFUL!")
		fmt.Printf("   Patched: %s\n", strings.Join(result.PatchedFiles, ", "))
		fmt.Printf("   Backup:  %s\n", strings.Join(result.BackupFiles, ", "))
		fmt.Println("\n   💡 To undo: rename .bak back to original.")
	} else {
		fmt.Println("❌ PATCH FAILED:")
		for _, e := range result.Errors {
			fmt.Printf("   %s\n", e)
		}
		os.Exit(1)
	}
	fmt.Println()
	reportPath, err := report.GenerateReportWithStyle(analysis, validation, styleInfo, charPath)
	if err == nil {
		fmt.Printf("📄 Report saved: %s\n", reportPath)
	}
}

func buildStyleInfo(s *style.AIStyle, blendInfo string) *report.StyleInfo {
	info := &report.StyleInfo{Name: s.Name, BlendInfo: blendInfo}
	for _, d := range s.Decisions {
		info.Decisions = append(info.Decisions, report.StyleDecisionInfo{
			Priority: d.Priority, Strategy: d.Strategy, State: d.State, Label: d.Label,
		})
	}
	return info
}

func findStylesDir() string {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "styles")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "styles")
}

func resolveStylePath(p string) string {
	if _, err := os.Stat(p); err == nil {
		abs, _ := filepath.Abs(p)
		return abs
	}
	stylesDir := findStylesDir()
	for _, candidate := range []string{
		filepath.Join(stylesDir, p),
		filepath.Join(stylesDir, p+".zss"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	abs, _ := filepath.Abs(p)
	return abs
}

func printHeader(mode string) {
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Printf("║     ikemen-ai-patcher — %-20s║\n", mode)
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
}

func printValidation(v *validator.ValidationResult) {
	if v.Valid {
		fmt.Println("  ✅ Validation: PASSED")
	} else {
		fmt.Println("  ❌ Validation: FAILED")
		for _, e := range v.Errors {
			fmt.Printf("     ERROR: %s\n", e)
		}
	}
	for _, w := range v.Warnings {
		fmt.Printf("     ⚠ %s\n", w)
	}
	fmt.Println()
}

func exitOnErr(err error, prefix string) {
	if err != nil {
		fmt.Printf("❌ %s: %v\n", prefix, err)
		os.Exit(1)
	}
}

func resolvePath(p string) string {
	abs, _ := filepath.Abs(p)
	return abs
}

func formatInts(nums []int) string {
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

func printUsage() {
	fmt.Printf(`ikemen-ai-patcher v%s — Boss AI Generator + Desktop UI

GUI MODE:
  ikemen-ai-patcher.exe                        Launch desktop application

CLI MODE:
  ikemen-ai-patcher.exe analyze <char_folder>
  ikemen-ai-patcher.exe patch   <char_folder>
  ikemen-ai-patcher.exe report  <char_folder>
  ikemen-ai-patcher.exe style list
  ikemen-ai-patcher.exe style load   <file.zss>
  ikemen-ai-patcher.exe style apply  <char> <style.zss>
  ikemen-ai-patcher.exe style blend  <char> <s1> <s2>
  ikemen-ai-patcher.exe version
  ikemen-ai-patcher.exe help
`, version)
}

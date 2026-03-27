// Package patcher handles non-destructive AI injection into character files.
package patcher

import (
	"fmt"
	"strings"

	"ikemen-ai-patcher/ai"
	"ikemen-ai-patcher/utils"
)

const (
	markerStart = ";===== IKEMEN-AI-PATCHER START ====="
	markerEnd   = ";===== IKEMEN-AI-PATCHER END ====="
)

// PatchResult holds the outcome of a patch operation.
type PatchResult struct {
	PatchedFiles []string // Files that were modified
	BackupFiles  []string // Backup files created
	Errors       []string // Any errors encountered
	Success      bool
}

// ApplyPatch injects AI blocks into the character's .cmd and .cns files.
// Non-destructive: creates backups and uses marker comments for re-patching.
func ApplyPatch(cmdFilePath, cnsFilePath string, blocks []ai.AIBlock) *PatchResult {
	result := &PatchResult{Success: true}

	for _, block := range blocks {
		if block.Section == "cmd" && cmdFilePath != "" {
			if err := patchCmdFile(cmdFilePath, block.Content, result); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("CMD patch failed: %v", err))
				result.Success = false
			}
		} else if block.Section == "cns" && cnsFilePath != "" {
			if err := patchCnsFile(cnsFilePath, block.Content, result); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("CNS patch failed: %v", err))
				result.Success = false
			}
		}
	}

	return result
}

// patchCmdFile injects AI code into a .cmd file's State -1 section.
func patchCmdFile(filePath, aiCode string, result *PatchResult) error {
	// Read original file
	lines, err := utils.ReadFileLines(filePath)
	if err != nil {
		return err
	}

	// Create backup before modifying
	bakPath, err := utils.CreateBackup(filePath)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	result.BackupFiles = append(result.BackupFiles, bakPath)

	// User request: Remove [State -1, Tick Fix] blocks to avoid conflicts
	lines = removeTickFixBlocks(lines)
	
	// User request: Strip generated triggerless blocks
	aiCode = removeTriggerlessBlocks(aiCode)

	// Check if we have an existing patch (markers present)
	content := strings.Join(lines, "\n")
	if strings.Contains(content, markerStart) {
		// Replace existing patch
		content = replaceBetweenMarkers(content, aiCode)
	} else {
		// Find insertion point: after [Statedef -1] section header
		content = insertAfterStateDef(lines, aiCode)
	}

	// Write patched file
	if err := utils.WriteFile(filePath, content); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	result.PatchedFiles = append(result.PatchedFiles, filePath)

	return nil
}

// patchCnsFile appends or updates AI helper Statedefs in the .cns file.
func patchCnsFile(filePath, aiCode string, result *PatchResult) error {
	lines, err := utils.ReadFileLines(filePath)
	if err != nil {
		return err
	}

	bakPath, err := utils.CreateBackup(filePath)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	result.BackupFiles = append(result.BackupFiles, bakPath)

	content := strings.Join(lines, "\n")
	if strings.Contains(content, markerStart) {
		content = replaceBetweenMarkers(content, aiCode)
	} else {
		// Just append to the end for CNS helpers
		content = content + "\n\n" + aiCode + "\n"
	}

	if err := utils.WriteFile(filePath, content); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	result.PatchedFiles = append(result.PatchedFiles, filePath)

	return nil
}

// replaceBetweenMarkers replaces content between the AI patcher markers.
// This allows re-patching without duplicating AI code.
func replaceBetweenMarkers(content, newCode string) string {
	startIdx := strings.Index(content, markerStart)
	endIdx := strings.Index(content, markerEnd)

	if startIdx < 0 || endIdx < 0 || endIdx <= startIdx {
		// Markers not found or malformed — append instead
		return content + "\n" + newCode
	}

	// Replace everything from start marker to end marker (inclusive)
	before := content[:startIdx]
	after := content[endIdx+len(markerEnd):]

	return before + newCode + after
}

// insertAfterStateDef finds [Statedef -1] and inserts AI code after it.
func insertAfterStateDef(lines []string, aiCode string) string {
	var result strings.Builder
	inserted := false
	inNeg1 := false
	foundFirstController := false

	for i, line := range lines {
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}

		trimmed := strings.TrimSpace(strings.ToLower(line))

		// Detect [Statedef -1]
		if strings.Contains(trimmed, "[statedef -1]") {
			inNeg1 = true
			continue
		}

		// After finding Statedef -1, insert before the first [State -1, ...] controller
		if inNeg1 && !inserted && !foundFirstController {
			if strings.HasPrefix(trimmed, "[state -1,") || strings.HasPrefix(trimmed, "[state -1 ,") {
				foundFirstController = true
				// Insert AI code BEFORE this controller
				result.WriteString("\n")
				result.WriteString(aiCode)
				result.WriteString("\n")
				inserted = true
			}
		}
	}

	// If we never found a good insertion point, just append at the end
	if !inserted {
		result.WriteString("\n")
		result.WriteString(aiCode)
	}

	return result.String()
}

// removeTickFixBlocks actively deletes any block named [State -1, Tick Fix] to prevent conflicts.
func removeTickFixBlocks(lines []string) []string {
	var result []string
	inTickFix := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.ToLower(line))

		// Start of a new controller block
		if strings.HasPrefix(trimmed, "[state") {
			if strings.Contains(trimmed, "tick fix") {
				inTickFix = true // We are now inside a Tick Fix block
				continue
			} else {
				inTickFix = false // Normal block, resume keeping lines
			}
		} else if strings.HasPrefix(trimmed, "[statedef") {
			inTickFix = false
		}

		if !inTickFix {
			result = append(result, line)
		}
	}

	return result
}

// removeTriggerlessBlocks scrubs generated AI code blocks to remove any [State -1] 
// controllers that completely lack triggers (which causes MUGEN parsing crashes).
func removeTriggerlessBlocks(aiCode string) string {
	var result strings.Builder
	// Split by state definition
	// Since we know our generator outputs [State -1, ...] we can split by "[State "
	blocks := strings.Split(aiCode, "[State ")
	
	if len(blocks) > 0 {
		result.WriteString(blocks[0]) // Keep initial comments
	}
	
	for i := 1; i < len(blocks); i++ {
		fullBlock := "[State " + blocks[i]
		lower := strings.ToLower(fullBlock)
		
		// If it's a State block, check for triggers
		if strings.Contains(lower, "type") && strings.Contains(lower, "=") {
			hasTrigger := strings.Contains(lower, "triggerall") || 
			              strings.Contains(lower, "trigger1") || 
			              strings.Contains(lower, "trigger2")
			
			if hasTrigger {
				result.WriteString(fullBlock)
			}
			// If missing completely, drop it silently
		} else {
			// Not a real state block (e.g. comments matching the split)
			result.WriteString(fullBlock)
		}
	}
	
	return result.String()
}

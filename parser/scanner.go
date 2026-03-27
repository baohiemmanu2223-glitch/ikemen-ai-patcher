package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanCharacterFolder discovers all relevant files in a character folder.
// It reads the .def file to find referenced state files, then collects
// all .cmd, .cns, .st, .zss files.
type ScanResult struct {
	DefFile   string   // Path to the .def file
	CmdFiles  []string // .cmd files
	CnsFiles  []string // .cns files
	StFiles   []string // .st files
	ZssFiles  []string // .zss files
	CharName  string   // Character name extracted from .def
}

// ScanFolder scans the given character directory for relevant files.
func ScanFolder(charPath string) (*ScanResult, error) {
	info, err := os.Stat(charPath)
	if err != nil {
		return nil, fmt.Errorf("character path not found: %s", charPath)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", charPath)
	}

	result := &ScanResult{}

	// Find .def file first
	entries, err := os.ReadDir(charPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		fullPath := filepath.Join(charPath, entry.Name())

		switch ext {
		case ".def":
			if result.DefFile == "" {
				result.DefFile = fullPath
			}
		case ".cmd":
			result.CmdFiles = append(result.CmdFiles, fullPath)
		case ".cns":
			result.CnsFiles = append(result.CnsFiles, fullPath)
		case ".st":
			result.StFiles = append(result.StFiles, fullPath)
		case ".zss":
			result.ZssFiles = append(result.ZssFiles, fullPath)
		}
	}

	// Also scan subdirectories (States/, etc.)
	subdirs := []string{"States", "states", "AI", "ai"}
	for _, sub := range subdirs {
		subPath := filepath.Join(charPath, sub)
		if _, err := os.Stat(subPath); err != nil {
			continue
		}
		subEntries, err := os.ReadDir(subPath)
		if err != nil {
			continue
		}
		for _, entry := range subEntries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			fullPath := filepath.Join(subPath, entry.Name())
			switch ext {
			case ".cmd":
				result.CmdFiles = append(result.CmdFiles, fullPath)
			case ".cns":
				result.CnsFiles = append(result.CnsFiles, fullPath)
			case ".st":
				result.StFiles = append(result.StFiles, fullPath)
			case ".zss":
				result.ZssFiles = append(result.ZssFiles, fullPath)
			}
		}
	}

	// Parse .def to get referenced files and character name
	if result.DefFile != "" {
		result.parseDefFile(charPath)
	}

	return result, nil
}

// parseDefFile reads the .def file to extract character name and referenced files.
func (sr *ScanResult) parseDefFile(charPath string) {
	data, err := os.ReadFile(sr.DefFile)
	if err != nil {
		return
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	currentSection := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}

		// Section header
		if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "]") {
			end := strings.Index(trimmed, "]")
			currentSection = strings.ToLower(strings.TrimSpace(trimmed[1:end]))
			continue
		}

		// Key=value in relevant sections
		eqIdx := strings.Index(trimmed, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(trimmed[:eqIdx]))
		val := strings.TrimSpace(trimmed[eqIdx+1:])

		// Remove inline comments
		if scIdx := strings.Index(val, ";"); scIdx >= 0 {
			val = strings.TrimSpace(val[:scIdx])
		}

		switch currentSection {
		case "info":
			if key == "name" || key == "displayname" {
				sr.CharName = strings.Trim(val, "\"")
			}
		case "files":
			// Add referenced state files that we haven't already found
			if strings.Contains(key, "st") || key == "cmd" || key == "cns" || key == "stcommon" {
				refPath := filepath.Join(charPath, filepath.FromSlash(val))
				if _, err := os.Stat(refPath); err == nil {
					ext := strings.ToLower(filepath.Ext(refPath))
					switch ext {
					case ".cmd":
						sr.CmdFiles = addUnique(sr.CmdFiles, refPath)
					case ".cns":
						sr.CnsFiles = addUnique(sr.CnsFiles, refPath)
					case ".st":
						sr.StFiles = addUnique(sr.StFiles, refPath)
					case ".zss":
						sr.ZssFiles = addUnique(sr.ZssFiles, refPath)
					}
				}
			}
		}
	}
}

// addUnique adds a path to a slice if not already present.
func addUnique(slice []string, item string) []string {
	abs1, _ := filepath.Abs(item)
	for _, existing := range slice {
		abs2, _ := filepath.Abs(existing)
		if strings.EqualFold(abs1, abs2) {
			return slice
		}
	}
	return append(slice, item)
}

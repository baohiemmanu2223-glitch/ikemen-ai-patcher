// Package utils provides shared utility functions for ikemen-ai-patcher.
package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// --- File I/O Helpers ---

// ReadFileLines reads a file and returns its lines (handles \r\n).
func ReadFileLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")
	return lines, nil
}

// WriteFile writes content string to a file, creating directories as needed.
func WriteFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// CreateBackup copies originalPath to originalPath.bak.
// Returns the backup path or error.
func CreateBackup(originalPath string) (string, error) {
	bakPath := originalPath + ".bak"
	src, err := os.Open(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open %s for backup: %w", originalPath, err)
	}
	defer src.Close()

	dst, err := os.Create(bakPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup %s: %w", bakPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy to backup: %w", err)
	}
	return bakPath, nil
}

// --- String Helpers ---

// TrimComment removes inline comments (; and everything after) from a line.
func TrimComment(line string) string {
	// Don't strip inside quoted strings
	inQuote := false
	for i, ch := range line {
		if ch == '"' {
			inQuote = !inQuote
		}
		if ch == ';' && !inQuote {
			return strings.TrimSpace(line[:i])
		}
	}
	return strings.TrimSpace(line)
}

// IsBlankOrComment returns true if the line is empty or a comment.
func IsBlankOrComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, ";")
}

// ParseKeyValue splits "key = value" into (key, value).
// Returns empty strings if not a key=value pair.
func ParseKeyValue(line string) (string, string) {
	idx := strings.Index(line, "=")
	if idx < 0 {
		return "", ""
	}
	key := strings.TrimSpace(line[:idx])
	val := strings.TrimSpace(line[idx+1:])
	return strings.ToLower(key), val
}

// --- Number Helpers ---

// ParseInt safely parses an integer string, returning 0 on failure.
func ParseInt(s string) int {
	s = strings.TrimSpace(s)
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// InRange checks if n is in [lo, hi] inclusive.
func InRange(n, lo, hi int) bool {
	return n >= lo && n <= hi
}

// --- Regex Helpers ---

var (
	VarPattern  = regexp.MustCompile(`(?i)\bvar\((\d+)\)`)
	FvarPattern = regexp.MustCompile(`(?i)\bfvar\((\d+)\)`)
	SysvarPattern = regexp.MustCompile(`(?i)\bsysvar\((\d+)\)`)
)

// ExtractVarIndices returns all var(N) indices found in a string.
func ExtractVarIndices(s string) []int {
	return extractIndices(VarPattern, s)
}

// ExtractFvarIndices returns all fvar(N) indices found in a string.
func ExtractFvarIndices(s string) []int {
	return extractIndices(FvarPattern, s)
}

// ExtractSysvarIndices returns all sysvar(N) indices found in a string.
func ExtractSysvarIndices(s string) []int {
	return extractIndices(SysvarPattern, s)
}

func extractIndices(re *regexp.Regexp, s string) []int {
	matches := re.FindAllStringSubmatch(s, -1)
	var result []int
	seen := map[int]bool{}
	for _, m := range matches {
		if len(m) >= 2 {
			n := ParseInt(m[1])
			if !seen[n] {
				seen[n] = true
				result = append(result, n)
			}
		}
	}
	return result
}

// --- Collection Helpers ---

// UniqueInts returns a deduplicated sorted-ish slice of ints.
func UniqueInts(items []int) []int {
	seen := map[int]bool{}
	var result []int
	for _, v := range items {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// ContainsInt checks if a slice contains a value.
func ContainsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// FindUnusedVars finds N unused var indices in [0, max) not present in usedVars.
func FindUnusedVars(usedVars []int, count, max int) []int {
	usedSet := map[int]bool{}
	for _, v := range usedVars {
		usedSet[v] = true
	}
	var result []int
	// Search from high indices down (less likely to conflict)
	for i := max - 1; i >= 0 && len(result) < count; i-- {
		if !usedSet[i] {
			result = append(result, i)
		}
	}
	return result
}

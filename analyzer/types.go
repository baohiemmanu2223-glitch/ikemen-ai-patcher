// Package analyzer provides analysis of parsed Ikemen GO character data.
package analyzer

// Analysis holds the complete analysis results for a character.
type Analysis struct {
	CharName    string       // Character name
	Normals     []int        // Normal move state IDs (200-299, 400-499, 600-699)
	Specials    []int        // Special move state IDs (1000-1999)
	Hypers      []int        // Hyper/super state IDs (>=3000)
	ComboChains []ComboChain // Detected combo chains
	Vars        []int        // var() indices in use
	Fvars       []int        // fvar() indices in use
	Sysvars     []int        // sysvar() indices in use
	UnusedVars  []int        // Available var() indices for AI (0-59)
	UnusedFvars []int        // Available fvar() indices for AI (0-39)
	UnusedSysvars []int      // Available sysvar() indices for AI (0-4)
	
	VarUsages   []VarUsage   // Detailed usage of each variable
	// Phase 4: Memory AI & Frame Data
	VarAllocation   map[string]int // Assigned vars, e.g., "AILevel": 59
	FvarAllocation  map[string]int // Assigned fvars, e.g., "EnemyXVel": 17
	FrameData       map[int]FrameInfo // State ID -> FrameInfo

	RiskStates  []RiskEntry  // States flagged as risky
	TotalStates int          // Total number of parsed states

	// Phase 5 Categories (for Universal ZSS)
	Classified map[string][]int // Maps category names (e.g. "projectile", "antiair") to State IDs
}

// ComboChain represents a detected chain of state transitions forming a combo.
type ComboChain struct {
	States      []int  // Ordered list of state IDs in the chain
	Description string // Human-readable description
}

// RiskEntry describes a potentially unsafe state.
type RiskEntry struct {
	StateID     int
	Reason      string
	Severity    string // "low", "medium", "high"
}

// VarUsage tracks how a variable is used across states.
type VarUsage struct {
	Index     int
	Type      string // "var", "fvar", "sysvar"
	UsedIn    []int  // State IDs where referenced
	SetIn     []int  // State IDs where assigned
}

// FrameInfo holds safety data and properties of an attack state.
type FrameInfo struct {
	StateID        int
	Classification string // "Safe", "Semi-Safe", "Unsafe"
	HitDefFound    bool
	PauseTime      int    // extracted from HitDef pausetime
}

// Package parser provides AST-like structures for Ikemen GO character files.
package parser

// CharacterData holds all parsed data from a character's files.
type CharacterData struct {
	Name     string            // Character name from .def
	BasePath string            // Character folder path
	States   map[int]*State    // StateID -> State (from .cns, .st, .cmd)
	Commands []Command         // Command definitions from .cmd
	Neg1State *StateDef        // The special [Statedef -1] block
	Files    []string          // List of parsed files
}

// StateDef represents a [Statedef N] block header and its properties.
type StateDef struct {
	ID       int
	Type     string // S, C, A, L (Stand, Crouch, Air, Lie)
	MoveType string // A, I, H (Attack, Idle, GetHit)
	Physics  string // S, C, A, N (Stand, Crouch, Air, None)
	Anim     string // Animation number (may be expression)
	Ctrl     string // Control value
	Juggle   string // Juggle points
	Poweradd string // Power addition
	Velset   string // Velocity set
	Raw      map[string]string // All other key-value pairs
}

// State represents a full state: its definition and all controllers.
type State struct {
	Def         StateDef
	Controllers []Controller
}

// Controller represents a [State N, Name] block.
type Controller struct {
	StateID   int      // Parent state ID
	Name      string   // Controller label (e.g., "Punch Hit")
	Type      string   // Controller type (ChangeState, HitDef, VarSet, etc.)
	Triggers  []Trigger
	Values    map[string]string // key=value pairs (value, x, y, etc.)
	Raw       []string // Raw lines for reconstruction
}

// Trigger represents a trigger line (triggerall, trigger1, trigger2, etc.).
type Trigger struct {
	Level int    // 0 = triggerall, 1 = trigger1, 2 = trigger2, etc.
	Raw   string // Raw expression text
}

// Command represents a [Command] block from .cmd files.
type Command struct {
	Name    string
	Input   string // command = value
	Time    int
	Buffer  int
}

// --- Helper Methods ---

// HasController checks if the state has a controller of the given type.
func (s *State) HasController(ctrlType string) bool {
	for _, c := range s.Controllers {
		if eqFold(c.Type, ctrlType) {
			return true
		}
	}
	return false
}

// GetControllersByType returns all controllers matching the given type.
func (s *State) GetControllersByType(ctrlType string) []Controller {
	var result []Controller
	for _, c := range s.Controllers {
		if eqFold(c.Type, ctrlType) {
			result = append(result, c)
		}
	}
	return result
}

// GetChangeStateTargets returns all state IDs this state can transition to.
func (s *State) GetChangeStateTargets() []int {
	var targets []int
	for _, c := range s.Controllers {
		if eqFold(c.Type, "ChangeState") {
			if val, ok := c.Values["value"]; ok {
				n := parseIntSafe(val)
				if n != 0 || val == "0" {
					targets = append(targets, n)
				}
			}
		}
	}
	return targets
}

// eqFold does case-insensitive string comparison.
func eqFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func parseIntSafe(s string) int {
	// Handle simple integer values; ignore expressions
	n := 0
	negative := false
	s = trimSpaces(s)
	if len(s) == 0 {
		return 0
	}
	start := 0
	if s[0] == '-' {
		negative = true
		start = 1
	}
	for i := start; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n = n*10 + int(s[i]-'0')
		} else {
			break // Stop at non-digit (expressions like "1000+var(0)")
		}
	}
	if negative {
		return -n
	}
	return n
}

func trimSpaces(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

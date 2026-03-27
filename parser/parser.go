package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"ikemen-ai-patcher/utils"
)

// --- Regex patterns for section headers ---
var (
	statedefPattern = regexp.MustCompile(`(?i)^\[statedef\s+(-?\d+)\s*(?:,\s*(.+))?\]$`)
	statePattern    = regexp.MustCompile(`(?i)^\[state\s+(-?\d+)\s*,\s*(.+?)\]$`)
	commandPattern  = regexp.MustCompile(`(?i)^\[command\]$`)
	sectionPattern  = regexp.MustCompile(`(?i)^\[(.+?)\]$`)
	triggerPattern  = regexp.MustCompile(`(?i)^(triggerall|trigger(\d+))\s*=\s*(.+)$`)
)

// ParseCharacter parses all files in a character folder and returns CharacterData.
func ParseCharacter(charPath string) (*CharacterData, error) {
	scan, err := ScanFolder(charPath)
	if err != nil {
		return nil, err
	}

	cd := &CharacterData{
		Name:     scan.CharName,
		BasePath: charPath,
		States:   make(map[int]*State),
	}

	if cd.Name == "" {
		cd.Name = "Unknown"
	}

	// Parse all file types
	allFiles := make([]string, 0)
	allFiles = append(allFiles, scan.CmdFiles...)
	allFiles = append(allFiles, scan.CnsFiles...)
	allFiles = append(allFiles, scan.StFiles...)
	allFiles = append(allFiles, scan.ZssFiles...)

	for _, f := range allFiles {
		ext := strings.ToLower(f[strings.LastIndex(f, "."):])
		if ext == ".zss" {
			if err := parseZSSFile(cd, f); err != nil {
				fmt.Printf("  [WARN] Error parsing ZSS %s: %v\n", f, err)
			}
		} else {
			if err := parseMugenFile(cd, f); err != nil {
				fmt.Printf("  [WARN] Error parsing %s: %v\n", f, err)
			}
		}
		cd.Files = append(cd.Files, f)
	}

	return cd, nil
}

// parseMugenFile parses a single .cmd/.cns/.st file using a state-machine approach.
func parseMugenFile(cd *CharacterData, filePath string) error {
	lines, err := utils.ReadFileLines(filePath)
	if err != nil {
		return err
	}

	type parseMode int
	const (
		modeNone parseMode = iota
		modeStateDef
		modeController
		modeCommand
		modeOther
	)

	mode := modeNone
	var currentStateDef *StateDef
	var currentController *Controller
	var currentCommand *Command
	currentStateID := 0

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		// Skip empty/comment lines (but preserve raw lines for controllers)
		if line == "" || strings.HasPrefix(line, ";") {
			if mode == modeController && currentController != nil {
				currentController.Raw = append(currentController.Raw, rawLine)
			}
			continue
		}

		// Remove inline comments for parsing (but keep raw)
		cleanLine := utils.TrimComment(line)
		if cleanLine == "" {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(cleanLine, "[") {
			// --- [Statedef N] ---
			if m := statedefPattern.FindStringSubmatch(cleanLine); m != nil {
				// Save previous controller if any
				saveController(cd, currentStateDef, currentController, currentStateID)
				currentController = nil

				id, _ := strconv.Atoi(m[1])
				currentStateID = id
				currentStateDef = &StateDef{
					ID:  id,
					Raw: make(map[string]string),
				}
				mode = modeStateDef

				// Ensure state exists in map
				if _, exists := cd.States[id]; !exists {
					cd.States[id] = &State{Def: *currentStateDef}
				}

				// Special: statedef -1
				if id == -1 {
					cd.Neg1State = currentStateDef
				}
				continue
			}

			// --- [State N, Name] ---
			if m := statePattern.FindStringSubmatch(cleanLine); m != nil {
				// Save previous controller
				saveController(cd, currentStateDef, currentController, currentStateID)

				id, _ := strconv.Atoi(m[1])
				currentController = &Controller{
					StateID: id,
					Name:    strings.TrimSpace(m[2]),
					Values:  make(map[string]string),
					Raw:     []string{rawLine},
				}
				mode = modeController
				continue
			}

			// --- [Command] ---
			if commandPattern.MatchString(cleanLine) {
				saveController(cd, currentStateDef, currentController, currentStateID)
				currentController = nil
				currentCommand = &Command{}
				mode = modeCommand
				continue
			}

			// --- Other sections ([Remap], [Defaults], etc.) ---
			if sectionPattern.MatchString(cleanLine) {
				saveController(cd, currentStateDef, currentController, currentStateID)
				currentController = nil
				currentCommand = nil
				mode = modeOther
				continue
			}
		}

		// Process content based on current mode
		switch mode {
		case modeStateDef:
			key, val := utils.ParseKeyValue(cleanLine)
			if key == "" {
				continue
			}
			switch key {
			case "type":
				currentStateDef.Type = val
			case "movetype":
				currentStateDef.MoveType = val
			case "physics":
				currentStateDef.Physics = val
			case "anim":
				currentStateDef.Anim = val
			case "ctrl":
				currentStateDef.Ctrl = val
			case "juggle":
				currentStateDef.Juggle = val
			case "poweradd":
				currentStateDef.Poweradd = val
			case "velset":
				currentStateDef.Velset = val
			default:
				currentStateDef.Raw[key] = val
			}
			// Update the state in map
			if st, exists := cd.States[currentStateDef.ID]; exists {
				st.Def = *currentStateDef
			}

		case modeController:
			if currentController == nil {
				continue
			}
			currentController.Raw = append(currentController.Raw, rawLine)

			// Check for trigger lines
			if tm := triggerPattern.FindStringSubmatch(cleanLine); tm != nil {
				level := 0 // triggerall
				if tm[2] != "" {
					level, _ = strconv.Atoi(tm[2])
				}
				currentController.Triggers = append(currentController.Triggers, Trigger{
					Level: level,
					Raw:   strings.TrimSpace(tm[3]),
				})
				continue
			}

			// Key=value pairs
			key, val := utils.ParseKeyValue(cleanLine)
			if key != "" {
				if key == "type" {
					currentController.Type = val
				} else {
					currentController.Values[key] = val
				}
			}

		case modeCommand:
			if currentCommand == nil {
				continue
			}
			key, val := utils.ParseKeyValue(cleanLine)
			if key == "" {
				continue
			}
			switch key {
			case "name":
				currentCommand.Name = strings.Trim(val, "\"")
			case "command":
				currentCommand.Input = val
			case "time":
				currentCommand.Time = utils.ParseInt(val)
			case "buffer.time", "command.buffer.time":
				currentCommand.Buffer = utils.ParseInt(val)
			}

		case modeOther:
			// Ignore content in other sections
		}
	}

	// Save last controller
	saveController(cd, currentStateDef, currentController, currentStateID)

	// Save last command
	if currentCommand != nil && currentCommand.Name != "" {
		cd.Commands = append(cd.Commands, *currentCommand)
	}

	return nil
}

// saveController adds a completed controller to the correct state in CharacterData.
func saveController(cd *CharacterData, sd *StateDef, ctrl *Controller, stateID int) {
	if ctrl == nil || ctrl.Type == "" {
		return
	}

	// For State -1 controllers, use -1 as the key
	id := stateID
	if sd != nil {
		id = sd.ID
	}

	state, exists := cd.States[id]
	if !exists {
		state = &State{}
		if sd != nil {
			state.Def = *sd
		} else {
			state.Def = StateDef{ID: id}
		}
		cd.States[id] = state
	}
	state.Controllers = append(state.Controllers, *ctrl)
}

// parseZSSFile provides basic parsing for .zss (Ikemen GO modern format).
// ZSS uses bracket-based syntax instead of INI-style.
func parseZSSFile(cd *CharacterData, filePath string) error {
	lines, err := utils.ReadFileLines(filePath)
	if err != nil {
		return err
	}

	// ZSS format example:
	//   [Statedef 200]
	//   [State 200, Attack]
	//   type = ChangeState
	//   ...
	// Essentially the same section-based format but may use different conventions.
	// We handle it the same way as MUGEN format since Ikemen GO supports both.

	// Check if this is actually a ZSS-style file or traditional
	isModernZSS := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") || strings.Contains(trimmed, ":=") {
			isModernZSS = true
			break
		}
	}

	if isModernZSS {
		return parseModernZSS(cd, lines)
	}
	// Fallback: parse as traditional MUGEN format
	return parseMugenFileFromLines(cd, lines)
}

// parseModernZSS handles the newer ZSS scripting format.
// This is a simplified parser that extracts state definitions.
func parseModernZSS(cd *CharacterData, lines []string) error {
	stateDefRe := regexp.MustCompile(`(?i)^\[statedef\s+(-?\d+)\]`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := stateDefRe.FindStringSubmatch(trimmed); m != nil {
			id, _ := strconv.Atoi(m[1])
			if _, exists := cd.States[id]; !exists {
				cd.States[id] = &State{
					Def: StateDef{ID: id},
				}
			}
		}
	}
	return nil
}

// parseMugenFileFromLines is the same as parseMugenFile but works from pre-read lines.
func parseMugenFileFromLines(cd *CharacterData, lines []string) error {
	type parseMode int
	const (
		modeNone parseMode = iota
		modeStateDef
		modeController
		modeCommand
		modeOther
	)

	mode := modeNone
	var currentStateDef *StateDef
	var currentController *Controller
	currentStateID := 0

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		cleanLine := utils.TrimComment(line)
		if cleanLine == "" {
			continue
		}

		if strings.HasPrefix(cleanLine, "[") {
			if m := statedefPattern.FindStringSubmatch(cleanLine); m != nil {
				saveController(cd, currentStateDef, currentController, currentStateID)
				currentController = nil
				id, _ := strconv.Atoi(m[1])
				currentStateID = id
				currentStateDef = &StateDef{ID: id, Raw: make(map[string]string)}
				mode = modeStateDef
				if _, exists := cd.States[id]; !exists {
					cd.States[id] = &State{Def: *currentStateDef}
				}
				if id == -1 {
					cd.Neg1State = currentStateDef
				}
				continue
			}
			if m := statePattern.FindStringSubmatch(cleanLine); m != nil {
				saveController(cd, currentStateDef, currentController, currentStateID)
				id, _ := strconv.Atoi(m[1])
				currentController = &Controller{
					StateID: id,
					Name:    strings.TrimSpace(m[2]),
					Values:  make(map[string]string),
				}
				mode = modeController
				continue
			}
			if commandPattern.MatchString(cleanLine) {
				saveController(cd, currentStateDef, currentController, currentStateID)
				currentController = nil
				mode = modeOther
				continue
			}
			if sectionPattern.MatchString(cleanLine) {
				saveController(cd, currentStateDef, currentController, currentStateID)
				currentController = nil
				mode = modeOther
				continue
			}
		}

		switch mode {
		case modeStateDef:
			key, val := utils.ParseKeyValue(cleanLine)
			if key != "" && currentStateDef != nil {
				switch key {
				case "type":
					currentStateDef.Type = val
				case "movetype":
					currentStateDef.MoveType = val
				case "physics":
					currentStateDef.Physics = val
				case "anim":
					currentStateDef.Anim = val
				}
				if st, exists := cd.States[currentStateDef.ID]; exists {
					st.Def = *currentStateDef
				}
			}
		case modeController:
			if currentController == nil {
				continue
			}
			if tm := triggerPattern.FindStringSubmatch(cleanLine); tm != nil {
				level := 0
				if tm[2] != "" {
					level, _ = strconv.Atoi(tm[2])
				}
				currentController.Triggers = append(currentController.Triggers, Trigger{
					Level: level,
					Raw:   strings.TrimSpace(tm[3]),
				})
				continue
			}
			key, val := utils.ParseKeyValue(cleanLine)
			if key == "type" {
				currentController.Type = val
			} else if key != "" {
				currentController.Values[key] = val
			}
		}
	}
	saveController(cd, currentStateDef, currentController, currentStateID)
	return nil
}

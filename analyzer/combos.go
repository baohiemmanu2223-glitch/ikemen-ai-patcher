package analyzer

import (
	"strings"

	"ikemen-ai-patcher/parser"
)

// DetectComboChains analyzes state transitions to find combo chains.
// A combo chain is a sequence of states linked by ChangeState controllers
// whose triggers reference MoveHit, MoveContact, or similar hit-confirm conditions.
func DetectComboChains(data *parser.CharacterData) []ComboChain {
	// Build adjacency map: state -> list of targets (only hit-confirmed transitions)
	transitions := make(map[int][]int)

	for id, state := range data.States {
		if id < 0 {
			continue
		}
		for _, ctrl := range state.Controllers {
			if !isChangeState(ctrl.Type) {
				continue
			}
			target := getChangeStateTarget(ctrl)
			if target < 0 {
				continue
			}
			// Only count transitions with hit-confirm triggers
			if hasHitConfirmTrigger(ctrl) {
				transitions[id] = append(transitions[id], target)
			}
		}
	}

	// Also analyze State -1 (command state) for combo routes
	if neg1, exists := data.States[-1]; exists {
		for _, ctrl := range neg1.Controllers {
			if !isChangeState(ctrl.Type) {
				continue
			}
			target := getChangeStateTarget(ctrl)
			if target < 0 {
				continue
			}
			// Check if this -1 controller has hit-confirm triggers AND a stateno condition
			if hasHitConfirmTrigger(ctrl) {
				sourceStates := extractStatenoConditions(ctrl)
				for _, src := range sourceStates {
					transitions[src] = append(transitions[src], target)
				}
			}
		}
	}

	// Find chains by walking the adjacency graph from normal moves
	var chains []ComboChain
	visited := make(map[int]bool)

	// Start from normals (200-699 range)
	for startID := range transitions {
		if startID < 200 || startID >= 700 {
			continue
		}
		if visited[startID] {
			continue
		}

		chain := walkChain(startID, transitions, visited, 20) // Max depth 20
		if len(chain) >= 2 {
			chains = append(chains, ComboChain{
				States:      chain,
				Description: describeChain(chain),
			})
		}
	}

	// Also walk from specials that start chains
	for startID := range transitions {
		if startID < 1000 {
			continue
		}
		if visited[startID] {
			continue
		}
		chain := walkChain(startID, transitions, visited, 20)
		if len(chain) >= 2 {
			chains = append(chains, ComboChain{
				States:      chain,
				Description: describeChain(chain),
			})
		}
	}

	return chains
}

// walkChain walks the transition graph from a starting state, collecting chain.
func walkChain(start int, transitions map[int][]int, visited map[int]bool, maxDepth int) []int {
	chain := []int{start}
	visited[start] = true
	current := start

	for depth := 0; depth < maxDepth; depth++ {
		targets, ok := transitions[current]
		if !ok || len(targets) == 0 {
			break
		}
		// Follow the first unvisited target
		found := false
		for _, t := range targets {
			if !visited[t] {
				chain = append(chain, t)
				visited[t] = true
				current = t
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return chain
}

// isChangeState checks if the controller type is ChangeState.
func isChangeState(ctrlType string) bool {
	return strings.EqualFold(ctrlType, "ChangeState")
}

// getChangeStateTarget extracts the target state ID from a ChangeState controller.
func getChangeStateTarget(ctrl parser.Controller) int {
	val, ok := ctrl.Values["value"]
	if !ok {
		return -1
	}
	return parseSimpleInt(val)
}

// hasHitConfirmTrigger checks if any trigger references hit-confirm conditions.
func hasHitConfirmTrigger(ctrl parser.Controller) bool {
	for _, t := range ctrl.Triggers {
		lower := strings.ToLower(t.Raw)
		if strings.Contains(lower, "movehit") ||
			strings.Contains(lower, "movecontact") ||
			strings.Contains(lower, "moveguarded") ||
			strings.Contains(lower, "gethitvar") ||
			strings.Contains(lower, "animtime") ||
			strings.Contains(lower, "animelem") {
			return true
		}
	}
	return false
}

// extractStatenoConditions extracts state numbers referenced in trigger conditions.
func extractStatenoConditions(ctrl parser.Controller) []int {
	var stateNos []int
	for _, t := range ctrl.Triggers {
		lower := strings.ToLower(t.Raw)
		if strings.Contains(lower, "stateno") {
			// Try to extract "stateno = N" or "stateno = [N, M]"
			parts := strings.Split(lower, "stateno")
			for _, part := range parts[1:] {
				part = strings.TrimSpace(part)
				if len(part) > 0 && (part[0] == '=' || part[0] == '!') {
					numStr := strings.TrimLeft(part, "=! ")
					n := parseSimpleInt(numStr)
					if n > 0 {
						stateNos = append(stateNos, n)
					}
				}
			}
		}
	}
	return stateNos
}

// describeChain creates a human-readable description for a combo chain.
func describeChain(chain []int) string {
	if len(chain) == 0 {
		return "empty chain"
	}
	var parts []string
	for _, id := range chain {
		label := "?"
		switch {
		case id >= 200 && id < 300:
			label = "StNormal"
		case id >= 400 && id < 500:
			label = "CrNormal"
		case id >= 600 && id < 700:
			label = "AirNormal"
		case id >= 1000 && id < 3000:
			label = "Special"
		case id >= 3000:
			label = "Super"
		}
		parts = append(parts, strings.Join([]string{label, itoa(id)}, ":"))
	}
	return strings.Join(parts, " -> ")
}

func parseSimpleInt(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	neg := false
	start := 0
	if len(s) > 0 && s[0] == '-' {
		neg = true
		start = 1
	}
	for i := start; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n = n*10 + int(s[i]-'0')
		} else {
			break
		}
	}
	if neg {
		return -n
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

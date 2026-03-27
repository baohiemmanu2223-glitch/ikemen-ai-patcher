package analyzer

import (
	"ikemen-ai-patcher/parser"
	"ikemen-ai-patcher/utils"
)

// CategorizeStates sorts states into normals, specials, and hypers
// based on their state IDs and properties. Does NOT hardcode specific states —
// uses ID ranges as defined in MUGEN convention.
func CategorizeStates(data *parser.CharacterData) (normals, specials, hypers []int) {
	for id, state := range data.States {
		if id < 0 {
			continue // Skip negative states (system states)
		}

		// Skip non-attack states
		if state.Def.MoveType != "" && !isAttackType(state.Def.MoveType) {
			// Still include if it has HitDef controllers
			if !state.HasController("HitDef") {
				continue
			}
		}

		switch {
		// Ground normals: 200-299 (standing), 400-499 (crouching)
		// Air normals: 600-699
		case utils.InRange(id, 200, 299) || utils.InRange(id, 400, 499) || utils.InRange(id, 600, 699):
			normals = append(normals, id)

		// Specials: 1000-2999 (standard range)
		// Also handles large custom state IDs like 1050000, 1650000, 1700000
		case utils.InRange(id, 1000, 2999):
			specials = append(specials, id)
		case id >= 1000000 && id < 3000000:
			// Large custom special state IDs (e.g., Kid Goku's 1050000, 1650000)
			specials = append(specials, id)

		// Hypers/Supers: 3000+
		case id >= 3000 && id < 1000000:
			hypers = append(hypers, id)
		case id >= 3000000:
			hypers = append(hypers, id)
		}
	}

	normals = utils.UniqueInts(normals)
	specials = utils.UniqueInts(specials)
	hypers = utils.UniqueInts(hypers)
	return
}

// isAttackType checks if the movetype indicates an attack.
func isAttackType(mt string) bool {
	if len(mt) == 0 {
		return false
	}
	ch := mt[0]
	return ch == 'A' || ch == 'a'
}

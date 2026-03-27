package analyzer

import (
	"strconv"
	"strings"

	"ikemen-ai-patcher/parser"
)

// ParseFrameData extracts frame data from states, specifically looking at HitDef controllers
func ParseFrameData(data *parser.CharacterData) map[int]FrameInfo {
	frames := make(map[int]FrameInfo)

	for _, state := range data.States {
		info := FrameInfo{
			StateID:        state.Def.ID,
			Classification: "Safe", // Default
			HitDefFound:    false,
		}

		for _, ctrl := range state.Controllers {
			if strings.EqualFold(ctrl.Type, "HitDef") {
				info.HitDefFound = true
				
				// Try to parse pausetime (format: p1_pausetime, p2_pausetime)
				if pt, ok := ctrl.Values["pausetime"]; ok {
					parts := strings.Split(pt, ",")
					if len(parts) > 0 {
						if p, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
							info.PauseTime = p
						}
					}
				}

				// Basic classification heuristic based on pause time
				if info.PauseTime > 12 {
					info.Classification = "Unsafe"
				} else if info.PauseTime > 8 {
					info.Classification = "Semi-Safe"
				}
				
				// If damage is huge, it's likely a heavy impact move or hyper, usually unsafe if whiffed
				if dmg, ok := ctrl.Values["damage"]; ok {
					parts := strings.Split(dmg, ",")
					if d, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && d > 100 {
						info.Classification = "Unsafe"
					}
				}

				// If it causes fall = 1, it's often a sweep or heavy move
				if fall, ok := ctrl.Values["fall"]; ok && strings.TrimSpace(fall) == "1" {
					if info.Classification == "Safe" {
						info.Classification = "Semi-Safe"
					}
				}
			}
		}

		frames[state.Def.ID] = info
	}

	return frames
}

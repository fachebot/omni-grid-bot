package util

import "github.com/fachebot/omni-grid-bot/internal/ent"

func StrategyName(record *ent.Strategy) string {
	return record.GUID[len(record.GUID)-4:]
}

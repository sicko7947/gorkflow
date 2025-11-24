package engine

import (
	"time"

	workflow "github.com/sicko7947/gorkflow"
)

// calculateBackoff is a wrapper around the internal helper
func calculateBackoff(baseDelayMs int, attempt int, strategy string) time.Duration {
	return workflow.CalculateBackoff(baseDelayMs, attempt, strategy)
}

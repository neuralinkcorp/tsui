package ui

import (
	"fmt"
	"math"
	"time"
)

// Format a Duration to a human-friendly string.
func FormatDuration(duration time.Duration) string {
	if duration.Hours() >= 1 {
		hours := int(math.Floor(duration.Hours()))
		return fmt.Sprintf("%dh", hours)
	} else {
		minutes := int(math.Floor(duration.Minutes()))
		return fmt.Sprintf("%dm", minutes)
	}
}

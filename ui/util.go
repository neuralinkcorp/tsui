package ui

import (
	"fmt"
	"math"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Format a Duration to a human-friendly string.
func FormatDuration(duration time.Duration) string {
	days := duration.Hours() / 24
	months := days / 30.437 // Average days in a month, we don't need to be precise.

	if months >= 2 {
		months := int(math.Floor(months))
		// It's somewhat arbitrary that months are the only one we use the full word for,
		// but "5mo" doesn't look as aesthetically pleasing to me.
		return fmt.Sprintf("%d Months", months)
	} else if days >= 1 {
		days := int(math.Floor(days))
		return fmt.Sprintf("%dd", days)
	} else if duration.Hours() >= 1 {
		hours := int(math.Floor(duration.Hours()))
		return fmt.Sprintf("%dh", hours)
	} else {
		minutes := int(math.Floor(duration.Minutes()))
		return fmt.Sprintf("%dm", minutes)
	}
}

// Formate a byte count to a human-friendly string.
func FormatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KiB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MiB", float64(bytes)/1024/1024)
	} else if bytes < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GiB", float64(bytes)/1024/1024/1024)
	} else {
		return fmt.Sprintf("%.2f TiB", float64(bytes)/1024/1024/1024/1024)
	}
}

// Combine a left-aligned and a right-aligned string into one fixed-width line.
// Takes a style which is used for formatting the left-side padding, in case
// a uniform background is required.
func RenderSplit(left string, right string, width int, style lipgloss.Style) string {
	left = style.
		Width(width - lipgloss.Width(right)).
		Render(left)

	return left + right
}

package ui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

const animWidth = 22
const animHeight = 11

var animLines = [animHeight]string{
	` ....    ....    .... `,
	`......  ......  ......`,
	` ....    ....    .... `,
	`                      `,
	` TTTT    UUUU    IIII `,
	`TTTTTT  UUUUUU  IIIIII`,
	` TTTT    UUUU    IIII `,
	`                      `,
	` ....    SSSS    .... `,
	`......  SSSSSS  ......`,
	` ....    SSSS    .... `,
}

var targetLetters = []byte{'T', 'S', 'U', 'I'}

// Designed rendering rate of the PoggersAnimationFrame animation.
const PoggersAnimationInterval = 80 * time.Millisecond

// Render a frame of the cool loading animation I designed.
func PoggersAnimationFrame(t int) string {
	frame := ""

	targetLetter := targetLetters[(t/animWidth)%len(targetLetters)]

	for y := 0; y < animHeight; y++ {
		if y > 0 {
			frame += "\n"
		}

		// Line equation determined by bruteforce.
		waveX := ((t+6)%animWidth-y)*2 - 4

		for x := 0; x < animWidth; x++ {
			char := animLines[y][x]
			style := lipgloss.NewStyle()

			switch char {
			case '.':
				style = style.
					Faint(true)

			case 'T', 'S', 'U', 'I':
				if char == targetLetter {
					style = style.Foreground(Secondary)
				}

				isWave := x == waveX || x == waveX+1 || x == waveX+2 || x == waveX+3 // Thick
				if isWave {
					style = style.Bold(true)
				} else {
					// Make lowercase.
					char = char - 'A' + 'a'
				}
			}

			frame += style.Render(string(char))
		}
	}

	return frame
}

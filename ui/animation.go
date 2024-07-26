package ui

import "github.com/charmbracelet/lipgloss"

var animLines = []string{
	` ....    ....    .... `,
	`......  ......  ......`,
	` ....    ....    .... `,
	`                      `,
	` TTTT    SSSS    UUUU `,
	`TTTTTT  SSSSSS  UUUUUU`,
	` TTTT    SSSS    UUUU `,
	`                      `,
	` ....    IIII    .... `,
	`......  IIIIII  ......`,
	` ....    IIII    .... `,
}
var animWidth = len(animLines[0])
var animHeight = len(animLines)

func Go() string {
	frame := ""

	for y := 0; y < animHeight; y++ {
		if y > 0 {
			frame += "\n"
		}

		for x := 0; x < animWidth; x++ {
			char := animLines[y][x]
			style := lipgloss.NewStyle()

			switch char {
			case '.':
				style = style.
					Faint(true)

			case 'T', 'S', 'U', 'I':
				style = style.
					Foreground(Primary)
			}

			frame += style.Render(string(char))
		}
	}

	return frame
}

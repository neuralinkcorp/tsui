package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AppmenuItem struct {
	LeftLabel  string
	RightLabel string
	Submenu    Submenu
}

func (i *AppmenuItem) render(isSelected bool, isAnySubmenuOpen bool) string {
	style := lipgloss.NewStyle()

	if isSelected {
		if isAnySubmenuOpen {
			style = style.
				Background(DarkGray)
		} else {
			style = style.
				Background(Primary).
				Foreground(Black)
		}
	} else {
		if isAnySubmenuOpen {
			style = style.
				Faint(true)
		} else {
			// Unstyled.
		}
	}

	left := style.
		Width(15).
		Padding(0, 1).
		Render(i.LeftLabel)
	right := style.
		Width(35).
		Faint(true).
		AlignHorizontal(lipgloss.Right).
		Render(i.RightLabel)
	arrow := style.
		Padding(0, 1).
		Render(">")

	return left + right + arrow
}

type Appmenu struct {
	Items  []*AppmenuItem
	cursor int
	isOpen bool
}

func (appmenu *Appmenu) Render() string {
	s := ""

	for i, item := range appmenu.Items {
		if i > 0 {
			s += "\n"
		}
		s += item.render(i == appmenu.cursor, appmenu.isOpen)
	}

	// Render the submenu to the right of the appmenu.
	s = lipgloss.JoinHorizontal(lipgloss.Top, s, appmenu.Items[appmenu.cursor].Submenu.Render(appmenu.isOpen))

	return s
}

func (appmenu *Appmenu) CursorUp() {
	if appmenu.isOpen {
		// Move the cursor in the submenu.
		appmenu.Items[appmenu.cursor].Submenu.CursorUp()
	} else {
		// Move the cursor in the appmenu.
		if appmenu.cursor > 0 {
			appmenu.cursor--
		}
	}
}

func (appmenu *Appmenu) CursorDown() {
	if appmenu.isOpen {
		// Move the cursor in the submenu.
		appmenu.Items[appmenu.cursor].Submenu.CursorDown()
	} else {
		// Move the cursor in the appmenu.
		if appmenu.cursor < len(appmenu.Items)-1 {
			appmenu.cursor++
		}
	}
}

func (appmenu *Appmenu) Activate() tea.Cmd {
	if appmenu.isOpen {
		// Activate the item in the submenu.
		return appmenu.Items[appmenu.cursor].Submenu.Activate()
	} else {
		// Open the submenu.
		appmenu.isOpen = true
		appmenu.Items[appmenu.cursor].Submenu.ResetCursor()
		return nil
	}
}

func (appmenu *Appmenu) IsSubmenuOpen() bool {
	return appmenu.isOpen
}

func (appmenu *Appmenu) CloseSubmenu() {
	appmenu.isOpen = false
}

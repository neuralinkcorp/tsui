package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// An item in the main menu, containing a submenu.
type AppmenuItem struct {
	// Left-aligned label text.
	LeftLabel string
	// Right-aligned label text.
	RightLabel string
	// Submenu that this item reveals.
	Submenu Submenu
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
		} // else unstyled
	}

	left := style.
		PaddingLeft(1).
		Render(i.LeftLabel)
	right := style.
		Width(35 - lipgloss.Width(left)).
		Faint(true).
		AlignHorizontal(lipgloss.Right).
		Render(i.RightLabel)
	arrow := style.
		Padding(0, 1).
		Render(">")

	return left + right + arrow
}

// A state container for the main application menu.
// Each menu item contains a submenu which can be opened and closed.
type Appmenu struct {
	// List of menu items.
	items []*AppmenuItem
	// Current menu item index.
	cursor int
	// Whether the selected submenu is open.
	isOpen bool
}

// Render the menu to a string.
func (appmenu *Appmenu) Render() string {
	if len(appmenu.items) == 0 {
		return ""
	}

	s := ""

	for i, item := range appmenu.items {
		if i > 0 {
			s += "\n"
		}
		s += item.render(i == appmenu.cursor, appmenu.isOpen)
	}

	// Render the submenu to the right of the appmenu.
	s = lipgloss.JoinHorizontal(lipgloss.Top, s, appmenu.items[appmenu.cursor].Submenu.Render(appmenu.isOpen))

	return s
}

// Move the cursor to the next selectable item in the currently active menu.
func (appmenu *Appmenu) CursorDown() {
	if appmenu.isOpen {
		// Move the cursor in the submenu.
		appmenu.items[appmenu.cursor].Submenu.CursorDown()
	} else {
		// Move the cursor in the appmenu.
		if appmenu.cursor < len(appmenu.items)-1 {
			appmenu.cursor++
		}
	}
}

// Move the cursor to the previous selectable item in the currently active menu.
func (appmenu *Appmenu) CursorUp() {
	if appmenu.isOpen {
		// Move the cursor in the submenu.
		appmenu.items[appmenu.cursor].Submenu.CursorUp()
	} else {
		// Move the cursor in the appmenu.
		if appmenu.cursor > 0 {
			appmenu.cursor--
		}
	}
}

// Set the items list and ensure the cursor is within bounds.
func (appmenu *Appmenu) SetItems(items []*AppmenuItem) {
	appmenu.items = items
	appmenu.clampCursor()
}

// Ensure the cursor is within bounds.
func (appmenu *Appmenu) clampCursor() {
	if len(appmenu.items) == 0 {
		appmenu.cursor = 0
		appmenu.isOpen = false
		return
	}

	if appmenu.cursor > len(appmenu.items)-1 {
		appmenu.cursor = len(appmenu.items) - 1
	}
}

// If a submenu is open, activate the item in the submenu.
// Otherwise, open the currently selected submenu.
func (appmenu *Appmenu) Activate() tea.Cmd {
	if appmenu.isOpen {
		// Activate the item in the submenu.
		return appmenu.items[appmenu.cursor].Submenu.Activate()
	} else if len(appmenu.items) > 0 {
		// Open the submenu.
		appmenu.isOpen = true
		appmenu.items[appmenu.cursor].Submenu.ResetCursor()
	}
	return nil
}

// Returns true if a submenu is currently open.
func (appmenu *Appmenu) IsSubmenuOpen() bool {
	return appmenu.isOpen
}

// Close the submenu.
func (appmenu *Appmenu) CloseSubmenu() {
	appmenu.isOpen = false
}

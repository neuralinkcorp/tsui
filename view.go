package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/neuralink/tsui/libts"
	"tailscale.com/ipn"
)

var logo = strings.Join([]string{
	`   __             _ `,
	`  / /________  __(_)`,
	` / __/ ___/ / / / / `,
	`/ /_(__  ) /_/ / /  `,
	`\__/____/\__,_/_/   `,
	`          by n7k    `,
}, "\n")

const (
	primary   = lipgloss.Color("207")
	secondary = lipgloss.Color("135")

	red      = lipgloss.Color("203")
	blue     = lipgloss.Color("039")
	green    = lipgloss.Color("040")
	yellow   = lipgloss.Color("214")
	white    = lipgloss.Color("231")
	darkGray = lipgloss.Color("237")
	black    = lipgloss.Color("016")
)

// Format a list item for a submenu. Adds a newline to the end.
func listItem(label string, isSelected bool, isActive bool, isDim bool) string {
	style := lipgloss.NewStyle().
		Padding(0, 1)

	if isActive {
		label = "*" + label
		style = style.
			Bold(true).
			Foreground(secondary)
	} else {
		label = " " + label
	}

	if isSelected {
		style = style.
			Background(secondary).
			Foreground(black)
	} else if isDim {
		style = style.
			Faint(true)
	}

	return style.Width(35).Render(label) + "\n"
}

// Format the status button in the header bar.
func statusButton(backendState string, isUsingExitNode bool) string {
	buttonStyle := lipgloss.NewStyle().Padding(0, 1)

	switch backendState {
	case ipn.NeedsLogin.String():
		return buttonStyle.
			Background(yellow).
			Render("Needs Login")

	case ipn.NeedsMachineAuth.String():
		return buttonStyle.
			Background(yellow).
			Render("Needs Machine Auth")

	case ipn.Starting.String():
		return buttonStyle.
			Background(blue).
			Foreground(white).
			Render("Starting...")

	case ipn.Running.String():
		text := "Connected"
		if isUsingExitNode {
			text += " - Using Exit Node"
		}

		return buttonStyle.
			Background(green).
			Render(text)
	}

	return buttonStyle.
		Background(red).
		Render("Not Connected")
}

// Bubbletea's main render function. Called after state updates.
func (m model) View() string {
	// Don't render anything before we have our initial terminal info.
	if m.terminalWidth == 0 || m.terminalHeight == 0 {
		return ""
	}

	s := strings.Builder{}

	// Header bar.
	{
		logo := lipgloss.NewStyle().
			Foreground(primary).
			MarginRight(4).
			Render(logo)

		status := "Tailscale Status: "

		status += statusButton(m.state.BackendState, m.state.CurrentExitNode != nil)

		// Extra info; either auth URL or user login name, depending on the backend state.
		if m.state.AuthURL != "" {
			status += "\nAuth URL:       "
			status += lipgloss.NewStyle().
				Underline(true).
				Foreground(blue).
				Render(m.state.AuthURL)
		} else {
			status += "\n"
			status += lipgloss.NewStyle().
				Faint(true).
				Render(m.state.User.LoginName)
		}

		left := lipgloss.JoinHorizontal(lipgloss.Center, logo, status)

		// App versions.
		right := "tsui:      " + version + "\n"
		right += "tailscale: "
		if m.state.TSVersion != "" {
			right += m.state.TSVersion
		} else {
			right += "loading..."
		}
		right = lipgloss.NewStyle().
			Faint(true).
			Render(right)

		left = lipgloss.NewStyle().
			Width(m.terminalWidth - lipgloss.Width(right)).
			Render(left)
		section := lipgloss.JoinHorizontal(lipgloss.Center, left, right)
		s.WriteString(section + "\n\n\n")
	}

	// Menus.
	{
		mainMenu := lipgloss.NewStyle().
			Background(darkGray).
			Padding(0, 1).
			Render(lipgloss.NewStyle().
				Width(45).
				Render("Exit Nodes") + ">")

		// Exit node submenu. (Currently hardcoded, in the future there will be multiple.)
		subMenu := m.exitNodeSubmenu()

		section := lipgloss.JoinHorizontal(lipgloss.Top, mainMenu, subMenu)
		s.WriteString(section)
	}

	return s.String()
}

func (m *model) exitNodeSubmenu() string {
	subMenu := ""
	if m.keyMode == searchKeyMode {
		subMenu += "Search: "
		subMenu += lipgloss.NewStyle().
			Underline(true).
			Foreground(white).
			Render(m.searchString)
		subMenu += "\n"
	}

	subMenu += listItem("None", m.selectedExitNode == "", m.state.CurrentExitNode == nil, false)

	subMenu += lipgloss.NewStyle().
		Faint(true).
		Render("  --") + "\n"

	nodes := m.filteredExitNodes
	for _, choice := range nodes {
		isActive := m.state.CurrentExitNode != nil && choice.ID == *m.state.CurrentExitNode
		isSelected := m.selectedExitNode == choice.ID
		name := libts.PeerName(choice)
		subMenu += listItem(name, isSelected, isActive, !choice.Online)
	}
	return subMenu
}

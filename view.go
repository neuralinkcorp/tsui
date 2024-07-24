package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/neuralink/tsui/ui"
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

// Format the status button in the header bar.
func statusButton(backendState string, isUsingExitNode bool) string {
	buttonStyle := lipgloss.NewStyle().Padding(0, 1)

	switch backendState {
	case ipn.NeedsLogin.String():
		return buttonStyle.
			Background(ui.Yellow).
			Foreground(ui.Black).
			Render("Needs Login")

	case ipn.NeedsMachineAuth.String():
		return buttonStyle.
			Background(ui.Yellow).
			Foreground(ui.Black).
			Render("Needs Machine Auth")

	case ipn.Starting.String():
		return buttonStyle.
			Background(ui.Blue).
			Foreground(ui.White).
			Render("Starting...")

	case ipn.Running.String():
		text := "Connected"
		if isUsingExitNode {
			text += " - Exit Node"
		}

		return buttonStyle.
			Background(ui.Green).
			Foreground(ui.Black).
			Render(text)
	}

	return buttonStyle.
		Background(ui.Red).
		Foreground(ui.Black).
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
			Foreground(ui.Primary).
			MarginRight(4).
			Render(logo)

		status := "Tailscale Status: "
		status += statusButton(m.state.BackendState, m.state.CurrentExitNode != nil)
		if m.state.BackendState == ipn.Running.String() {
			status += lipgloss.NewStyle().
				Faint(true).
				PaddingLeft(1).
				Render("(press . to stop)")
		}
		status += "\n"

		// Extra info; either auth URL or user login name, depending on the backend state.
		if m.state.AuthURL != "" {
			status += "Auth URL:       "
			status += lipgloss.NewStyle().
				Underline(true).
				Foreground(ui.Blue).
				Render(m.state.AuthURL)
		} else if m.state.User != nil {
			status += lipgloss.NewStyle().
				Faint(true).
				Render(m.state.User.LoginName)
		} else {
			status += lipgloss.NewStyle().
				Faint(true).
				Render("--")
		}

		// App versions.
		versions := "tsui:      " + version + "\n"
		versions += "tailscale: "
		if m.state.TSVersion != "" {
			versions += m.state.TSVersion
		} else {
			versions += "(not connected)"
		}
		versions = lipgloss.NewStyle().
			Faint(true).
			Render(versions)

		// Spacer between the left content and the right content.
		spacer := lipgloss.NewStyle().
			Width(m.terminalWidth - lipgloss.Width(versions) - lipgloss.Width(status) - lipgloss.Width(logo)).
			Render(" ")

		section := lipgloss.JoinHorizontal(lipgloss.Center, logo, status, spacer, versions)
		s.WriteString(section + "\n\n\n")
	}

	s.WriteString(m.menu.Render())

	return s.String()
}

package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/neuralinkcorp/tsui/browser"
	"github.com/neuralinkcorp/tsui/ui"
	"tailscale.com/ipn"
)

// Format the status button in the header bar.
func renderStatusButton(backendState string, isUsingExitNode bool) string {
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 1)

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

	case ipn.Stopped.String():
		return buttonStyle.
			Background(ui.Red).
			Foreground(ui.Black).
			Render("Not Connected")

	case ipn.NoState.String():
		return buttonStyle.
			Background(ui.Blue).
			Foreground(ui.White).
			Render("Loading...")
	}

	return "???"
}

// Render the locked out warning. Returns static output; should be called conditionally.
func renderLockedOutWarning(m *model) string {
	heading := lipgloss.NewStyle().
		Background(ui.Yellow).
		Foreground(ui.Black).
		Bold(true).
		Padding(0, 1).
		Render("Warning: Locked Out")

	bodyText := "This node is locked out by tailnet lock. Please contact an administrator of your Tailscale network to authorize your connection."

	lockedOutWarning := lipgloss.NewStyle().
		Foreground(ui.Yellow).
		Width(80).
		Align(lipgloss.Center).
		Render(heading + "\n" + bodyText)

	return lipgloss.PlaceHorizontal(m.terminalWidth, lipgloss.Center, lockedOutWarning)
}

// Format the top header section.
func renderHeader(m *model) string {
	logo := lipgloss.NewStyle().
		Foreground(ui.Primary).
		MarginRight(4).
		Render(ui.Logo)

	status := "Status: "
	status += renderStatusButton(m.state.BackendState, m.state.CurrentExitNode != nil)
	if m.state.BackendState == ipn.Running.String() {
		status += lipgloss.NewStyle().
			Faint(true).
			PaddingLeft(1).
			Render("(press . to disconnect)")
	}
	status += "\n"

	// Extra info; either auth URL or user login name, depending on the backend state.
	if m.state.User == nil || m.state.User.LoginName == "" {
		status += lipgloss.NewStyle().
			Faint(true).
			Render("--")
	} else {
		status += lipgloss.NewStyle().
			Faint(true).
			Render(m.state.User.LoginName)
	}

	// App versions.
	versions := "tsui:      " + Version + "\n"
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

	return lipgloss.JoinHorizontal(lipgloss.Center, logo, status, spacer, versions)
}

// Render a banner/modal for the middle of the screen.
func renderMiddleBanner(m *model, height int, text string) string {
	divider := lipgloss.NewStyle().
		Faint(true).
		Render(strings.Repeat("=", lipgloss.Width(text)))

	return lipgloss.Place(m.terminalWidth, height, lipgloss.Center, lipgloss.Center,
		divider+"\n\n"+text+"\n\n"+divider)
}

// Render the bottom status bar.
func renderStatusBar(m *model) string {
	var text string

	if m.statusText == "" && m.canWrite {
		// Only show up/down if we have data.
		if m.state.BackendState == ipn.Running.String() {
			text = ""
		} else {
			text = lipgloss.NewStyle().
				Faint(true).
				Render(fmt.Sprintf(
					"▲ %s | %s ▼",
					ui.FormatBytes(m.state.TxBytes),
					ui.FormatBytes(m.state.RxBytes),
				))
		}
	} else if m.statusText == "" && !m.canWrite {
		text = lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.Yellow).
			Render("Read-only mode.")
		text += lipgloss.NewStyle().
			Foreground(ui.Yellow).
			Render(" To edit preferences, you may have to run tsui as root.")
	} else {
		var color lipgloss.Color

		switch m.statusType {
		case statusTypeError:
			color = ui.Red

			text = lipgloss.NewStyle().
				Foreground(color).
				Bold(true).
				Render("Error: ")

		case statusTypeSuccess:
			color = ui.Green

		case statusTypeTip:
			color = ui.Blue

			text = lipgloss.NewStyle().
				Foreground(color).
				Bold(true).
				Render("Tip! ")
		}

		text += lipgloss.NewStyle().
			Foreground(color).
			Render(m.statusText)
	}

	right := lipgloss.NewStyle().
		Faint(true).
		Render("q to quit")

	left := lipgloss.NewStyle().
		Width(m.terminalWidth - lipgloss.Width(right)).
		PaddingLeft(lipgloss.Width(right)).
		Align(lipgloss.Center).
		Render(text)

	return left + right
}

// Bubbletea's main render function. Called after state updates.
func (m model) View() string {
	// Don't render anything before we have our initial terminal info.
	if m.terminalWidth == 0 || m.terminalHeight == 0 {
		return ""
	}

	// Render the top of the page (header bar, locked out warning, etc).
	top := renderHeader(&m) + "\n\n"
	if m.state.IsLockedOut {
		top += renderLockedOutWarning(&m) + "\n\n"
	}
	top += "\n"

	// Render the bottom of the page (status bar, error text, etc).
	bottom := "\n" + renderStatusBar(&m)

	// Now, draw the middle, and make it take up the remaining space.
	middleHeight := m.terminalHeight - lipgloss.Height(top) - lipgloss.Height(bottom)
	var middle string

	styledAuthUrl := lipgloss.NewStyle().
		Underline(true).
		Foreground(ui.Blue).
		Render(m.state.AuthURL)

	switch m.state.BackendState {
	case ipn.Running.String():
		middle = lipgloss.NewStyle().
			Height(middleHeight).
			Render(m.menu.Render())

	case ipn.NeedsMachineAuth.String():
		// TODO: Figure out what this state actually is so we can be helpful to the user.
		middle = renderMiddleBanner(&m, middleHeight, "Tailscale status is NeedsMachineAuth.")

	case ipn.NeedsLogin.String():
		lines := []string{
			lipgloss.NewStyle().
				Bold(true).
				Render(`Login Required`),
			``,
			`You need to login to Tailscale before you can connect to the tailnet.`,
			``,
		}

		if m.state.AuthURL == "" {
			lines = append(lines,
				`Press . to authenticate.`,
			)
		} else {
			lines = append(lines,
				fmt.Sprintf(`Login URL: %s`, styledAuthUrl),
			)
			if browser.IsSupported() {
				// We can't open the browser for them if running as the root user on Linux.
				lines = append(lines,
					``,
					`Press . to open in browser.`,
				)
			}
		}

		middle = renderMiddleBanner(&m, middleHeight, strings.Join(lines, "\n"))

	case ipn.Stopped.String():
		middle = renderMiddleBanner(&m, middleHeight, strings.Join([]string{
			`The Tailscale daemon isn't running.`,
			``,
			`Press . to bring Tailscale up.`,
		}, "\n"))

	case ipn.NoState.String():
		middle = renderMiddleBanner(&m, middleHeight, ui.PoggersAnimationFrame(m.animationT))

	case ipn.Starting.String():
		if m.state.AuthURL == "" {
			middle = renderMiddleBanner(&m, middleHeight, ui.PoggersAnimationFrame(m.animationT))
		} else {
			// If we have an AuthURL in the Starting state, that means the user is reauthenticating!
			lines := []string{
				lipgloss.NewStyle().
					Bold(true).
					Render(`Reauthenticate with Tailscale`),
				``,
				fmt.Sprintf(`Login URL: %s`, styledAuthUrl),
			}
			if browser.IsSupported() {
				// We can't open the browser for them if running as the root user on Linux.
				lines = append(lines,
					``,
					`Press . to open in browser.`,
				)
			}
			middle = renderMiddleBanner(&m, middleHeight, strings.Join(lines, "\n"))
		}
	}

	return top + "\n" + middle + "\n" + bottom
}

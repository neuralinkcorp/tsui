package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/libts"
	"tailscale.com/ipn"
)

// Message triggered on each poller tick.
type tickMsg struct{}

// Creates a tea.Tick command that generates tickMsg messages.
func makeTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// Message representing a Tailscale state update.
type stateMsg libts.State

// Command that retrives a new Tailscale state and triggers a stateMsg.
// This will be run in a goroutine by the bubbletea runtime.
func updateState() tea.Msg {
	status, _ := libts.Status(ctx)
	state := libts.MakeState(status)
	return stateMsg(state)
}

// Bubbletea update function.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Create our ticker command which will be our "default return" in the absence of any other commands.
	tick := makeTick(5 * m.tickInterval)

	switch msg := msg.(type) {
	// On tick, fetch a new state.
	case tickMsg:
		return m, updateState

	// When the state updater returns, update our model.
	case stateMsg:
		m.updateFromState(libts.State(msg))

	case tea.WindowSizeMsg:
		needsClear := msg.Width < m.terminalWidth || msg.Height > m.terminalHeight

		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

		// Needed to clear artifacts in certain terminals.
		if needsClear {
			return m, tea.ClearScreen
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.menu.IsSubmenuOpen() {
				m.menu.CloseSubmenu()
			} else {
				return m, tea.Quit
			}

		case "left", "h":
			m.menu.CloseSubmenu()
		case "up", "k":
			m.menu.CursorUp()
		case "down", "j":
			m.menu.CursorDown()
		case "right", "l":
			if !m.menu.IsSubmenuOpen() {
				return m, m.menu.Activate()
			}

		case "enter", " ":
			return m, m.menu.Activate()

		case ".":
			switch m.state.BackendState {
			case ipn.Running.String():
				return m, func() tea.Msg {
					libts.SetWantRunning(ctx, false)
					return updateState()
				}

			case ipn.NoState.String():
			case ipn.Stopped.String():
				return m, func() tea.Msg {
					libts.SetWantRunning(ctx, true)
					return updateState()
				}
			}
		}
	}

	return m, tick
}

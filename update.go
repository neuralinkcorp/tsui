package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/libts"
	"tailscale.com/ipn/ipnstate"
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
func getState() tea.Msg {
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
		return m, getState

	// When the state updater returns, update our model.
	case stateMsg:
		m.state = libts.State(msg)

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "up":
			if m.exitNodeCursor > -1 {
				m.exitNodeCursor--
			}

		case "down":
			if m.exitNodeCursor < len(m.state.SortedExitNodes)-1 {
				m.exitNodeCursor++
			}

		// Select exit node on enter or space.
		case "enter", " ":
			var exitNode *ipnstate.PeerStatus

			// If they have an exit node selected and not "None", use it.
			if m.exitNodeCursor > -1 {
				exitNode = m.state.SortedExitNodes[m.exitNodeCursor]

				// If this exit node is already selected, don't do anything.
				if m.state.CurrentExitNode != nil && *m.state.CurrentExitNode == exitNode.ID {
					return m, tick
				}
			}

			// Eagerly update the model, and therefore the UI.
			if exitNode == nil {
				m.state.CurrentExitNode = nil
			} else {
				m.state.CurrentExitNode = &exitNode.ID
			}

			// Asynchronously use the Tailscale API to update the exit node.
			cmd := func() tea.Msg {
				libts.SetExitNode(ctx, exitNode)
				return nil
			}
			return m, cmd
		}
	}

	// Make sure the exit node cursor doesn't end up out of bounds if we lose exit nodes.
	if m.exitNodeCursor > len(m.state.SortedExitNodes)-1 {
		m.exitNodeCursor = -1
	}

	return m, tick
}

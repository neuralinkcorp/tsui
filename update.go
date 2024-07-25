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

// Message representing a Tailscale state update and any error that occurred (optional).
type stateMsg *libts.State

// Message representing some transient error.
type errorMsg error

// A success message to temporarily display.
type successMsg string

// Message to clear a status because its visiblity time elapsed.
// Stores an int corresponding to the statusGen, and this message should be
// ignored if the current statusGen is later.
type statusExpiredMsg int

// Command that retrives a new Tailscale state and triggers a stateMsg.
// This will be run in a goroutine by the bubbletea runtime.
func updateState() tea.Msg {
	status, err := libts.Status(ctx)
	if err != nil {
		return errorMsg(err)
	}

	prefs, err := libts.Prefs(ctx)
	if err != nil {
		return errorMsg(err)
	}

	lock, err := libts.LockStatus(ctx)
	if err != nil {
		return errorMsg(err)
	}

	state, err := libts.MakeState(status, prefs, lock)
	if err != nil {
		return errorMsg(err)
	}

	return stateMsg(state)
}

// Command that updates the Tailscale preferences and triggers a state update.
func editPrefs(maskedPrefs *ipn.MaskedPrefs) tea.Msg {
	err := libts.EditPrefs(ctx, maskedPrefs)
	if err != nil {
		return errorMsg(err)
	}
	return updateState()
}

// Bubbletea update function.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Create our ticker command which will be our "default return" in the absence of any other commands.
	tick := makeTick(tickInterval)

	switch msg := msg.(type) {
	// On tick, fetch a new state.
	case tickMsg:
		return m, updateState

	// When the state updater returns, update our model.
	case stateMsg:
		m.updateFromState(msg)

	// Display errors.
	case errorMsg:
		m.statusType = statusTypeError
		m.statusText = msg.Error()
		m.statusGen++
		return m, tea.Batch(
			// Make sure the state is up-to-date.
			updateState,

			func() tea.Msg {
				time.Sleep(errorLifetime)
				return statusExpiredMsg(m.statusGen)
			},
		)

	// Display successes, too!
	case successMsg:
		m.statusType = statusTypeSuccess
		m.statusText = string(msg)
		m.statusGen++
		return m, func() tea.Msg {
			time.Sleep(successLifetime)
			return statusExpiredMsg(m.statusGen)
		}

	// Clear the status when it expires.
	case statusExpiredMsg:
		if int(msg) >= m.statusGen {
			m.statusText = ""
		}

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
					err := libts.Down(ctx)
					if err != nil {
						return errorMsg(err)
					}
					return updateState()
				}

			case ipn.NoState.String():
			case ipn.Stopped.String():
				return m, func() tea.Msg {
					err := libts.Up(ctx)
					if err != nil {
						return errorMsg(err)
					}
					return updateState()
				}
			}
		}
	}

	return m, tick
}

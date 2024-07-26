package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/browser"
	"github.com/neuralink/tsui/libts"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

// Message triggered on each main poller tick.
type tickMsg struct{}

// Message triggered on each ping poller tick.
type pingTickMsg struct{}

// Message containing the result of a successful Tailscale state update.
type stateMsg libts.State

// Message with ping results ready to be stored in the model.
type pingResultsMsg map[tailcfg.StableNodeID]*ipnstate.PingResult

// Message representing some transient error.
type errorMsg error

// Messaging containing a success message to temporarily display.
type successMsg string

// Message containing a notice to temporarily display.
type tipMsg string

// Message to clear a status because its visiblity time elapsed.
// Stores an int corresponding to the statusGen, and this message should be
// ignored if the current statusGen is later.
type statusExpiredMsg int

// Command that retrieves a new Tailscale state and triggers a stateMsg.
// This will be run in a goroutine by the bubbletea runtime.
func updateState() tea.Msg {
	state, err := libts.GetState(ctx)
	if err != nil {
		return errorMsg(err)
	}
	return stateMsg(state)
}

// Creates a command to gets the current latency of the specified peers. Takes some time.
func makeDoPings(peers []*ipnstate.PeerStatus) tea.Cmd {
	return func() tea.Msg {
		pings := make(map[tailcfg.StableNodeID]*ipnstate.PingResult)

		for _, peer := range peers {
			ctx, cancel := context.WithTimeout(ctx, pingTimeout)
			result, err := libts.PingPeer(ctx, peer)
			cancel()

			if err != nil {
				continue
			}
			pings[peer.ID] = result
		}

		return pingResultsMsg(pings)
	}
}

// Command that updates the Tailscale preferences and triggers a state update.
func editPrefs(maskedPrefs *ipn.MaskedPrefs) tea.Msg {
	err := libts.EditPrefs(ctx, maskedPrefs)
	if err != nil {
		return errorMsg(err)
	}
	return updateState()
}

// Bubbletea update function; our main "event" handler.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

		// Global action hotkey.
		case ".":
			switch m.state.BackendState {
			// If running, stop Tailscale.
			case ipn.Running.String():
				return m, func() tea.Msg {
					err := libts.Down(ctx)
					if err != nil {
						return errorMsg(err)
					}
					return updateState()
				}

			// If stopped, start Tailscale.
			case ipn.Stopped.String():
				return m, func() tea.Msg {
					err := libts.Up(ctx)
					if err != nil {
						return errorMsg(err)
					}
					return updateState()
				}

			// If we need to login...
			case ipn.NeedsLogin.String():
				if m.state.AuthURL == "" {
					// If we haven't started the login flow yet, do so.
					// Tailscale will open their browser for us.
					return m, func() tea.Msg {
						err := libts.StartLoginInteractive(ctx)
						if err != nil {
							return errorMsg(err)
						}
						return successMsg("Starting login flow. This may take a few seconds.")
					}
				} else if browser.IsSupported() {
					// If the auth flow has already started, we need to open the browser ourselves.
					return m, func() tea.Msg {
						err := browser.OpenURL(m.state.AuthURL)
						if err != nil {
							return errorMsg(err)
						}
						return nil
					}
				}

			case ipn.Starting.String():
				// If we have an AuthURL in the Starting state, that means the user is reauthenticating
				// and we also need to open the browser!
				// (But not if we're root on Linux.)
				if m.state.AuthURL != "" && browser.IsSupported() {
					return m, func() tea.Msg {
						err := browser.OpenURL(m.state.AuthURL)
						if err != nil {
							return errorMsg(err)
						}
						return nil
					}
				}
			}
		}

	// On ticks, run the appropriate commands, and kick off the next tick.
	case tickMsg:
		return m, tea.Batch(
			updateState,
			tea.Tick(tickInterval, func(_ time.Time) tea.Msg {
				return tickMsg{}
			}),
		)
	case pingTickMsg:
		// For now we'll just run this on our exit nodes.
		return m, tea.Batch(
			makeDoPings(m.state.SortedExitNodes),
			tea.Tick(pingTickInterval, func(_ time.Time) tea.Msg {
				return pingTickMsg{}
			}),
		)

	// When our updaters return, update our model and refresh the menus.
	case stateMsg:
		m.state = libts.State(msg)
		m.updateMenus()
		if m.state.BackendState == ipn.NoState.String() {
			// Do updates more frequently if we have no state because it should load soon.
			return m, tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
				return tickMsg{}
			})
		}
	case pingResultsMsg:
		m.pings = msg
		m.updateMenus()

	// Display status bar notices.
	case errorMsg, successMsg, tipMsg:
		var lifetime time.Duration

		switch msg := msg.(type) {
		case errorMsg:
			m.statusType = statusTypeError
			m.statusText = msg.Error()
			lifetime = errorLifetime
		case successMsg:
			m.statusType = statusTypeSuccess
			m.statusText = string(msg)
			lifetime = successLifetime
		case tipMsg:
			m.statusType = statusTypeTip
			m.statusText = string(msg)
			lifetime = tipLifetime
		}

		m.statusGen++
		return m, tea.Batch(
			// Make sure the state is up-to-date.
			updateState,
			// Clear after the relevant interval.
			tea.Tick(lifetime, func(_ time.Time) tea.Msg {
				return statusExpiredMsg(m.statusGen)
			}),
		)

	// Clear the status when it expires.
	case statusExpiredMsg:
		if int(msg) >= m.statusGen {
			m.statusText = ""
		}
	}

	return m, nil
}

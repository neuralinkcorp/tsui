package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralinkcorp/tsui/libts"
	"github.com/neuralinkcorp/tsui/ui"
	"github.com/neuralinkcorp/tsui/version"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

// Message triggered on each main poller tick.
type tickMsg struct{}

// Message triggered on each ping poller tick.
type pingTickMsg struct{}

// Message to increment the animation frame counter.
type animationTickMsg struct{}

// Message containing the result of a successful Tailscale state update.
type stateMsg libts.State

// Message with ping results ready to be stored in the model.
type pingResultsMsg map[tailcfg.StableNodeID]*ipnstate.PingResult

// Message containing the latest version of tsui fetched from GitHub.
type latestVersionMsg string

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

// Command that starts the interactive login flow.
func startLoginInteractive() tea.Msg {
	err := libts.StartLoginInteractive(ctx)
	if err != nil {
		return errorMsg(err)
	}
	return successMsg("Starting login flow. This may take a few seconds.")
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

// Command that fetches the latest version of tsui.
func fetchLatestVersion() tea.Msg {
	latestVersion, err := version.FetchLatestVersion()
	if err != nil {
		return nil
	}
	return latestVersionMsg(latestVersion)
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
			// If in filter mode, exit filter mode
			if m.exitNodeFilterMode {
				m.exitNodeFilterMode = false
				m.exitNodeFilter = ""
				m.updateMenus()
			} else if m.menu.IsSubmenuOpen() {
				m.menu.CloseSubmenu()
			} else {
				return m, tea.Quit
			}

		case "left", "h":
			if !m.exitNodeFilterMode {
				m.menu.CloseSubmenu()
			}
		case "a":
			if m.exitNodeFilterMode {
				m.exitNodeFilter += "a"
				m.updateMenus()
			} else {
				m.menu.CloseSubmenu()
			}
		case "up":
			m.menu.CursorUp()
		case "down":
			m.menu.CursorDown()
		case "k", "w":
			if !m.exitNodeFilterMode {
				m.menu.CursorUp()
			} else {
				m.exitNodeFilter += msg.String()
				m.updateMenus()
			}
		case "j", "s":
			if !m.exitNodeFilterMode {
				m.menu.CursorDown()
			} else {
				m.exitNodeFilter += msg.String()
				m.updateMenus()
			}
		case "right", "l", "d":
			if m.exitNodeFilterMode && (msg.String() == "l" || msg.String() == "d") {
				m.exitNodeFilter += msg.String()
				m.updateMenus()
			} else if !m.exitNodeFilterMode && !m.menu.IsSubmenuOpen() {
				// Show a tip when entering the exit nodes menu
				cmd := m.menu.Activate()
				if m.menu.GetSelectedItem() == m.exitNodes && len(m.state.ExitNodes) > 0 {
					m.statusType = statusTypeTip
					m.statusText = "Press / to search exit nodes"
					m.statusGen++
					return m, tea.Batch(
						cmd,
						tea.Tick(tipLifetime, func(_ time.Time) tea.Msg {
							return statusExpiredMsg(m.statusGen)
						}),
					)
				}
				return m, cmd
			}

		case "enter", " ":
			return m, m.menu.Activate()

		case "backspace":
			if m.exitNodeFilterMode && len(m.exitNodeFilter) > 0 {
				m.exitNodeFilter = m.exitNodeFilter[:len(m.exitNodeFilter)-1]
				m.updateMenus()
			}

		case "/":
			// Enable filter mode when viewing exit nodes submenu
			if m.menu.IsSubmenuOpen() && m.menu.GetSelectedItem() == m.exitNodes && !m.exitNodeFilterMode {
				m.exitNodeFilterMode = true
			}

		// Global action hotkey.
		case ".":
			if m.exitNodeFilterMode {
				// Allow typing period in filter mode
				m.exitNodeFilter += "."
				m.updateMenus()
			} else {
				switch m.state.BackendState {
				// If running, stop Tailscale.
				case ipn.Running:
					return m, func() tea.Msg {
						err := libts.Down(ctx)
						if err != nil {
							return errorMsg(err)
						}
						return updateState()
					}

				// If stopped, start Tailscale.
				case ipn.Stopped:
					return m, func() tea.Msg {
						err := libts.Up(ctx)
						if err != nil {
							return errorMsg(err)
						}
						return updateState()
					}

				// If we need to login...
				case ipn.NeedsLogin:
					return m, startLoginInteractive

				case ipn.Starting:
					// If we have an AuthURL in the Starting state, that means the user is reauthenticating
					// and we want to open the browser for them (if supported).
					if m.state.AuthURL != "" && libts.StartLoginInteractiveWillOpenBrowser() {
						return m, startLoginInteractive
					}
				}
			}

		default:
			// Handle typing in filter mode
			if m.exitNodeFilterMode {
				// Only allow printable characters
				if len(msg.String()) == 1 {
					m.exitNodeFilter += msg.String()
					m.updateMenus()
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
			makeDoPings(m.state.ExitNodes),
			tea.Tick(pingTickInterval, func(_ time.Time) tea.Msg {
				return pingTickMsg{}
			}),
		)
	case animationTickMsg:
		m.animationT++
		return m, tea.Tick(ui.PoggersAnimationInterval, func(_ time.Time) tea.Msg {
			return animationTickMsg{}
		})

	// When our updaters return, update our model and refresh the menus.
	case stateMsg:
		m.state = libts.State(msg)
		m.updateMenus()
	case pingResultsMsg:
		m.pings = msg
		m.updateMenus()

	// When we get our latest version, just store it for (potential) display on exit.
	case latestVersionMsg:
		m.latestVersion = string(msg)

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

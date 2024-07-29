package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/neuralink/tsui/libts"
	"github.com/neuralink/tsui/ui"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

// Injected at build time by the flake.nix.
// This has to be a var or -X can't override it.
var Version = "local"

const (
	// Rate at which to poll Tailscale for status updates.
	tickInterval = 3 * time.Second

	// Rate at which to gather latency from peers.
	pingTickInterval = 6 * time.Second
	// Per-peer ping timeout.
	pingTimeout = 1 * time.Second

	// How long to keep messages in the bottom bar.
	errorLifetime   = 6 * time.Second
	successLifetime = 3 * time.Second
	tipLifetime     = 3 * time.Second
)

// The type of the bottom bar status message:
//
//	statusTypeError, statusTypeSuccess
type statusType int

const (
	statusTypeError statusType = iota
	statusTypeSuccess
	statusTypeTip
)

var ctx = context.Background()

// Central model containing application state.
type model struct {
	// Current Tailscale state info.
	state libts.State
	// Ping results per peer.
	pings map[tailcfg.StableNodeID]*ipnstate.PingResult

	// Main menu.
	menu       ui.Appmenu
	deviceInfo *ui.AppmenuItem
	exitNodes  *ui.AppmenuItem
	settings   *ui.AppmenuItem

	// Current width of the terminal.
	terminalWidth int
	// Current height of the terminal.
	terminalHeight int

	// Type of the status message.
	statusType statusType
	// Error text displayed at the bottom of the screen.
	statusText string
	// Current "generation" number for the status. Incremented every time the status
	// is updated and used to keep track of status expiration messages.
	statusGen int

	// Result of the update checker.
	latestVersion string

	// Frame counter for the loading animation. This is always running in the background,
	// even if the animation is not visible.
	animationT int
}

// Initialize the application state.
func initialModel() (model, error) {
	m := model{
		// Main menu items.
		deviceInfo: &ui.AppmenuItem{Label: "This Device"},
		exitNodes: &ui.AppmenuItem{Label: "Exit Nodes",
			Submenu: ui.Submenu{Exclusivity: ui.SubmenuExclusivityOne},
		},
		settings: &ui.AppmenuItem{Label: "Settings"},
	}

	state, err := libts.GetState(ctx)
	if err != nil {
		return m, err
	}

	m.state = state
	m.updateMenus()

	return m, nil
}

// Bubbletea init function.
func (m model) Init() tea.Cmd {
	return tea.Batch(
		// Perform our initial state fetch to populate menus
		updateState,
		// Run an initial batch of pings.
		makeDoPings(m.state.SortedExitNodes),
		// Kick off our ticks.
		tea.Tick(tickInterval, func(_ time.Time) tea.Msg {
			return tickMsg{}
		}),
		tea.Tick(pingTickInterval, func(_ time.Time) tea.Msg {
			return pingTickMsg{}
		}),
		tea.Tick(ui.PoggersAnimationInterval, func(_ time.Time) tea.Msg {
			return animationTickMsg{}
		}),
		// And fetch the latest version.
		fetchLatestVersion,
	)
}

func mainError(err error) {
	text := lipgloss.NewStyle().
		Foreground(ui.Red).
		Render(err.Error())
	fmt.Fprintln(os.Stderr, text)
	os.Exit(1)
}

func main() {
	m, err := initialModel()
	if err != nil {
		mainError(err)
	}

	// Enable "alternate screen" mode, a terminal convention designed for rendering
	// full-screen, interactive UIs.
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		mainError(err)
	}
	m = finalModel.(model)

	if m.latestVersion != "" && Version != "local" && m.latestVersion != Version {
		text := lipgloss.NewStyle().
			Foreground(ui.Yellow).
			Bold(true).
			Render("Update available!")
		text += lipgloss.NewStyle().
			Foreground(ui.Yellow).
			Render(fmt.Sprintf(" To upgrade tsui from %s to %s, run:", Version, m.latestVersion))
		text += lipgloss.NewStyle().
			Foreground(ui.Blue).
			Render("\n    curl -sS https://tsui.neuralink.com/ | bash")
		fmt.Println(text)
	}
}

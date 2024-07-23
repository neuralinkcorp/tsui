package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/libts"
	"github.com/neuralink/tsui/ui"
)

const (
	version = "0.0.1-beta1"

	defaultTickInterval = 2 * time.Second
)

var ctx = context.Background()

// Central model containing application state.
type model struct {
	// Current Tailscale state info.
	state libts.State

	// Main menu.
	menu      ui.Appmenu
	exitNodes *ui.AppmenuItem

	// Rate at which to poll Tailscale for status updates.
	tickInterval time.Duration
	// Current width of the terminal.
	terminalWidth int
	// Current height of the terminal.
	terminalHeight int
}

// Initialize the application state.
func initialModel() model {
	// Discard the error because this just implies that Tailscale is off.
	status, _ := libts.Status(ctx)
	state := libts.MakeState(status)

	// Construct the main menu
	exitNodes := &ui.AppmenuItem{
		LeftLabel:  "Exit Nodes",
		RightLabel: state.CurrentExitNodeName,
		Submenu:    ui.Submenu{Exclusivity: ui.SubmenuExclusivityOne},
	}
	menu := ui.Appmenu{
		Items: []*ui.AppmenuItem{
			exitNodes,
			&ui.AppmenuItem{
				LeftLabel: "Test",
			},
		},
	}

	return model{
		terminalWidth: 0,
		tickInterval:  defaultTickInterval,
		state:         state,

		menu:      menu,
		exitNodes: exitNodes,
	}
}

// Bubbletea init function.
func (m model) Init() tea.Cmd {
	// Perform our initial state fetch to populate menus.
	return updateState
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("fatal error: %v\n", err)
		os.Exit(1)
	}
}

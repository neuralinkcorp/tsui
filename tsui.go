package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/libts"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

const (
	version = "0.0.1-beta1"

	defaultTickInterval = 2 * time.Second
)

var ctx = context.Background()

const (
	normalKeyMode = iota
	searchKeyMode
)

// Central model containing application state.
type model struct {
	// Rate at which to poll Tailscale for status updates.
	tickInterval time.Duration
	// Current Tailscale state info.
	state libts.State
	// ID of the selected exit node or "" if none selected.
	selectedExitNode tailcfg.StableNodeID
	// Current width of the terminal.
	terminalWidth int
	// Current height of the terminal.
	terminalHeight int
	// Current key mode.
	keyMode int
	// Current search string (in searchKeyMode).
	searchString string
	// Exit nodes filtered using searchString.
	filteredExitNodes []*ipnstate.PeerStatus
}

// Initialize the application state.
func initialModel() model {
	// Discard the error because this just implies that Tailscale is off.
	status, _ := libts.Status(ctx)
	state := libts.MakeState(status)

	return model{
		terminalWidth:     0,
		tickInterval:      defaultTickInterval,
		state:             state,
		selectedExitNode:  "",
		keyMode:           normalKeyMode,
		searchString:      "",
		filteredExitNodes: nil,
	}
}

// Bubbletea init function.
func (m model) Init() tea.Cmd {
	// Start our Tailscale poller.
	return makeTick(m.tickInterval)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("fatal error: %v\n", err)
		os.Exit(1)
	}
}

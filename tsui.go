package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/libts"
	"github.com/neuralink/tsui/ui"
	"tailscale.com/ipn"
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

func (m *model) updateFromState(state libts.State) {
	m.state = state

	// Update the exit node submenu.
	{
		exitNodeItems := make([]ui.SubmenuItem, 2+len(m.state.SortedExitNodes))
		exitNodeItems[0] = &ui.ToggleableSubmenuItem{
			Label: "None",
			OnActivate: func() tea.Msg {
				libts.SetExitNode(ctx, nil)
				return updateState()
			},
			IsActive: m.state.CurrentExitNode == nil,
		}
		exitNodeItems[1] = &ui.DividerSubmenuItem{}
		for i, exitNode := range m.state.SortedExitNodes {
			// Offset for the "None" item and the divider.
			i += 2

			label := libts.PeerName(exitNode)
			if !exitNode.Online {
				label += " (offline)"
			}

			exitNodeItems[i] = &ui.ToggleableSubmenuItem{
				Label: label,
				OnActivate: func() tea.Msg {
					libts.SetExitNode(ctx, exitNode)
					return updateState()
				},
				IsActive: m.state.CurrentExitNode != nil && exitNode.ID == *m.state.CurrentExitNode,
				IsDim:    !exitNode.Online,
			}
		}

		m.exitNodes.RightLabel = m.state.CurrentExitNodeName
		m.exitNodes.Submenu.SetItems(exitNodeItems)
	}

	if m.state.BackendState == ipn.Running.String() {
		m.menu.Items = []*ui.AppmenuItem{
			m.exitNodes,
		}
	} else {
		m.menu.Items = []*ui.AppmenuItem{}
	}
	m.menu.ClampCursor()
}

// Initialize the application state.
func initialModel() model {
	m := model{
		tickInterval: defaultTickInterval,
		menu: ui.Appmenu{
			PlaceholderText: "The Tailscale daemon isn't started.\n\nPress . to bring Tailscale up.",
		},

		// Main menu items.
		exitNodes: &ui.AppmenuItem{
			LeftLabel: "Exit Nodes",
			Submenu:   ui.Submenu{Exclusivity: ui.SubmenuExclusivityOne},
		},
	}

	// Discard the error because this just implies that Tailscale is off.
	status, _ := libts.Status(ctx)
	state := libts.MakeState(status)
	m.updateFromState(state)

	return m
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

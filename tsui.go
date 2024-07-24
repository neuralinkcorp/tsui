package main

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/libts"
	"github.com/neuralink/tsui/ui"
	"tailscale.com/ipn"
)

// Injected at build time by the flake.nix.
// This has to be a var or -X can't override it.
var Version = "local"

const defaultTickInterval = 2 * time.Second

var ctx = context.Background()

// Central model containing application state.
type model struct {
	// Current Tailscale state info.
	state libts.State

	// Main menu.
	menu       ui.Appmenu
	deviceInfo *ui.AppmenuItem
	exitNodes  *ui.AppmenuItem

	// Rate at which to poll Tailscale for status updates.
	tickInterval time.Duration
	// Current width of the terminal.
	terminalWidth int
	// Current height of the terminal.
	terminalHeight int
}

// Initialize the application state.
func initialModel() model {
	m := model{
		tickInterval: defaultTickInterval,
		menu: ui.Appmenu{
			PlaceholderText: "The Tailscale daemon isn't started.\n\nPress . to bring Tailscale up.",
		},

		// Main menu items.
		deviceInfo: &ui.AppmenuItem{LeftLabel: "Device Info"},
		exitNodes: &ui.AppmenuItem{
			LeftLabel: "Exit Nodes",
			Submenu:   ui.Submenu{Exclusivity: ui.SubmenuExclusivityOne},
		},
	}

	// Discard the error because this just implies that Tailscale is off.
	status, _ := libts.Status(ctx)
	lock, _ := libts.LockStatus(ctx)
	state := libts.MakeState(status, lock)
	m.updateFromState(state)

	return m
}

func (m *model) updateFromState(state libts.State) {
	m.state = state

	if m.state.BackendState == ipn.Running.String() {
		// Update the device info submenu.
		{
			submenuItems := []ui.SubmenuItem{
				&ui.TitleSubmenuItem{Label: "Name"},
				&ui.LabeledSubmenuItem{
					Label: state.Self.DNSName[:len(state.Self.DNSName)-1],
				},
				&ui.SpacerSubmenuItem{},
				&ui.TitleSubmenuItem{Label: "IPs"},
			}

			for _, addr := range state.Self.TailscaleIPs {
				submenuItems = append(submenuItems, &ui.LabeledSubmenuItem{
					Label: addr.String(),
				})
			}

			submenuItems = append(submenuItems,
				&ui.SpacerSubmenuItem{},
				&ui.TitleSubmenuItem{Label: "Dev Info"},
				&ui.LabeledSubmenuItem{
					Label: string(state.Self.ID),
				},
				// &ui.SpacerSubmenuItem{},
				&ui.LabeledSubmenuItem{
					Label: state.Self.PublicKey.String(),
				},
			)

			if state.LockKey != nil {
				statusText := "Online"
				if state.IsLockedOut {
					statusText = "Locked Out"
				}

				submenuItems = append(submenuItems,
					&ui.SpacerSubmenuItem{},
					&ui.TitleSubmenuItem{Label: "Tailnet Lock: " + statusText},
					&ui.LabeledSubmenuItem{
						Label: state.LockKey.CLIString(),
					},
				)
			}

			m.deviceInfo.Submenu.SetItems(submenuItems)
		}

		// Update the exit node submenu.
		{
			exitNodeItems := make([]ui.SubmenuItem, 2+len(m.state.SortedExitNodes))
			exitNodeItems[0] = &ui.ToggleableSubmenuItem{
				LabeledSubmenuItem: ui.LabeledSubmenuItem{
					Label: "None",
					OnActivate: func() tea.Msg {
						libts.SetExitNode(ctx, nil)
						return updateState()
					},
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
					LabeledSubmenuItem: ui.LabeledSubmenuItem{
						Label: label,
						OnActivate: func() tea.Msg {
							libts.SetExitNode(ctx, exitNode)
							return updateState()
						},
						IsDim: !exitNode.Online,
					},
					IsActive: m.state.CurrentExitNode != nil && exitNode.ID == *m.state.CurrentExitNode,
				}
			}

			m.exitNodes.RightLabel = m.state.CurrentExitNodeName
			m.exitNodes.Submenu.SetItems(exitNodeItems)
		}

		// Make sure the menu items are visible.
		m.menu.SetItems([]*ui.AppmenuItem{
			m.deviceInfo,
			m.exitNodes,
		})
	} else {
		// Hide the menu items if not connected.
		m.menu.SetItems([]*ui.AppmenuItem{})
	}
}

// Bubbletea init function.
func (m model) Init() tea.Cmd {
	// Perform our initial state fetch to populate menus.
	return updateState
}

// func main() {
// 	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
// 	if _, err := p.Run(); err != nil {
// 		fmt.Printf("fatal error: %v\n", err)
// 		os.Exit(1)
// 	}
// }

func main() {
	err := libts.Up(ctx)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
	}
}

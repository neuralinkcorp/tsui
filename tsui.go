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
	"tailscale.com/ipn"
)

// Injected at build time by the flake.nix.
// This has to be a var or -X can't override it.
var Version = "local"

const (
	// Default rate at which to poll Tailscale for status updates.
	tickInterval = 2 * time.Second
	// How long to display status/error messages in the bottom bar before clearing them.
	statusLifetime = 5 * time.Second
)

var ctx = context.Background()

// Central model containing application state.
type model struct {
	// Current Tailscale state info.
	state libts.State

	// Main menu.
	menu       ui.Appmenu
	deviceInfo *ui.AppmenuItem
	exitNodes  *ui.AppmenuItem

	// Current width of the terminal.
	terminalWidth int
	// Current height of the terminal.
	terminalHeight int

	// Error text displayed at the bottom of the screen.
	statusText string
	// Current "generation" number for the status. Incremented every time the status
	// is updated and used to keep track of status expiration messages.
	statusGen int
}

// Initialize the application state.
func initialModel() (model, error) {
	m := model{
		// Main menu items.
		deviceInfo: &ui.AppmenuItem{LeftLabel: "Device Info"},
		exitNodes: &ui.AppmenuItem{
			LeftLabel: "Exit Nodes",
			Submenu:   ui.Submenu{Exclusivity: ui.SubmenuExclusivityOne},
		},
	}

	status, err := libts.Status(ctx)
	if err != nil {
		return m, err
	}

	lock, err := libts.LockStatus(ctx)
	if err != nil {
		return m, err
	}

	state := libts.MakeState(status, lock)
	m.updateFromState(state)

	return m, nil
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
						err := libts.SetExitNode(ctx, nil)
						if err != nil {
							return errorMsg(err)
						}
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
							err := libts.SetExitNode(ctx, exitNode)
							if err != nil {
								return errorMsg(err)
							}
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

func renderMainError(err error) string {
	return lipgloss.NewStyle().
		Foreground(ui.Red).
		Render(err.Error())
}

func main() {
	m, err := initialModel()
	if err != nil {
		fmt.Fprintln(os.Stderr, renderMainError(err))
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, renderMainError(err))
		os.Exit(1)
	}
}

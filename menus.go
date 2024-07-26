package main

import (
	"fmt"
	"math"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/libts"
	"github.com/neuralink/tsui/ui"
	"golang.design/x/clipboard"
	"tailscale.com/ipn"
	"tailscale.com/types/opt"
	"tailscale.com/types/preftype"
)

// Update all of the menu UIs from the current state.
func (m *model) updateMenus() {
	if m.state.BackendState == ipn.Running.String() {
		// Update the device info submenu.
		{
			submenuItems := []ui.SubmenuItem{
				&ui.TitleSubmenuItem{Label: "Name"},
				&ui.LabeledSubmenuItem{
					Label: m.state.Self.DNSName[:len(m.state.Self.DNSName)-1], // Remove the trailing dot.
					OnActivate: func() tea.Msg {
						clipboard.Write(clipboard.FmtText, []byte(m.state.Self.DNSName[:len(m.state.Self.DNSName)-1]))
						return successMsg("Copied full domain to clipboard.")
					},
				},
				&ui.SpacerSubmenuItem{},
				&ui.TitleSubmenuItem{Label: "IPs"},
			}

			for _, addr := range m.state.Self.TailscaleIPs {
				submenuItems = append(submenuItems, &ui.LabeledSubmenuItem{
					Label: addr.String(),
					OnActivate: func() tea.Msg {
						clipboard.Write(clipboard.FmtText, []byte(addr.String()))

						var versionName string
						if addr.Is4() {
							versionName = "IPv4"
						} else {
							versionName = "IPv6"
						}

						return successMsg(fmt.Sprintf("Copied %s address to clipboard.", versionName))
					},
				})
			}

			submenuItems = append(submenuItems,
				&ui.SpacerSubmenuItem{},
				&ui.TitleSubmenuItem{Label: "Debug Info"},
				&ui.LabeledSubmenuItem{
					Label: fmt.Sprintf("ID: %s", m.state.Self.ID),
					OnActivate: func() tea.Msg {
						clipboard.Write(clipboard.FmtText, []byte(string(m.state.Self.ID)))
						return successMsg("Copied Tailscale node ID to clipboard.")
					},
				},
				&ui.LabeledSubmenuItem{
					Label: m.state.Self.PublicKey.String(),
					OnActivate: func() tea.Msg {
						clipboard.Write(clipboard.FmtText, []byte(m.state.Self.PublicKey.String()))
						return successMsg("Copied node key to clipboard.")
					},
				},
			)

			if m.state.LockKey != nil {
				statusText := "Online"
				if m.state.IsLockedOut {
					statusText = "Locked Out"
				}

				submenuItems = append(submenuItems,
					&ui.SpacerSubmenuItem{},
					&ui.TitleSubmenuItem{Label: "Tailnet Lock: " + statusText},
					&ui.LabeledSubmenuItem{
						Label: m.state.LockKey.CLIString(),
						OnActivate: func() tea.Msg {
							clipboard.Write(clipboard.FmtText, []byte(m.state.LockKey.CLIString()))
							return successMsg("Copied tailnet lock key to clipboard.")
						},
					},
				)
			}

			submenuItems = append(submenuItems,
				&ui.SpacerSubmenuItem{},
				&ui.LabeledSubmenuItem{
					Label:   "[Disconnect from Tailscale]",
					Variant: ui.SubmenuItemVariantAccent,
					OnActivate: func() tea.Msg {
						err := libts.Down(ctx)
						if err != nil {
							return errorMsg(err)
						}
						return tipMsg("You can also simply press . to disconnect.")
					},
				},
			)

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

				pingLabel := "???"
				if !exitNode.Online {
					pingLabel = "Offline"
				} else if m.pings[exitNode.ID] != nil {
					pingLabel = fmt.Sprintf("%dms", int(math.Round(m.pings[exitNode.ID].LatencySeconds*1000)))
				}

				exitNodeItems[i] = &ui.ToggleableSubmenuItem{
					LabeledSubmenuItem: ui.LabeledSubmenuItem{
						Label:           libts.PeerName(exitNode),
						AdditionalLabel: pingLabel,
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

			m.exitNodes.AdditionalLabel = m.state.CurrentExitNodeName
			m.exitNodes.Submenu.SetItems(exitNodeItems)
		}

		// Update the settings submenu.
		{
			exitNode := "No"
			if m.state.Prefs.AdvertisesExitNode() {
				exitNode = "Exit Node"
			}

			accountTitle := "Account"
			reauthenticateButtonLabel := "[Reauthenticate]"
			if m.state.Self.KeyExpiry != nil {
				reauthenticateButtonLabel = "[Reauthenticate Now]"

				duration := time.Until(*m.state.Self.KeyExpiry)
				accountTitle += " - Key Expires in " + ui.FormatDuration(duration)
			}

			submenuItems := []ui.SubmenuItem{
				&ui.TitleSubmenuItem{Label: "General"},

				ui.NewYesNoSettingsSubmenuItem("Allow Incoming Connections",
					!m.state.Prefs.ShieldsUp,
					func(newValue bool) tea.Msg {
						return editPrefs(&ipn.MaskedPrefs{
							Prefs: ipn.Prefs{
								ShieldsUp: !newValue,
							},
							ShieldsUpSet: true,
						})
					},
				),

				ui.NewYesNoSettingsSubmenuItem("Use Subnet Routes",
					m.state.Prefs.RouteAll,
					func(newValue bool) tea.Msg {
						return editPrefs(&ipn.MaskedPrefs{
							Prefs: ipn.Prefs{
								RouteAll: newValue,
							},
							RouteAllSet: true,
						})
					},
				),

				ui.NewYesNoSettingsSubmenuItem("Use DNS Settings",
					m.state.Prefs.CorpDNS,
					func(newValue bool) tea.Msg {
						return editPrefs(&ipn.MaskedPrefs{
							Prefs: ipn.Prefs{
								CorpDNS: newValue,
							},
							CorpDNSSet: true,
						})
					},
				),

				&ui.SpacerSubmenuItem{},
				&ui.TitleSubmenuItem{Label: "Exit Nodes"},

				ui.NewYesNoSettingsSubmenuItem("Enable Local Network Access",
					m.state.Prefs.ExitNodeAllowLANAccess,
					func(newValue bool) tea.Msg {
						return editPrefs(&ipn.MaskedPrefs{
							Prefs: ipn.Prefs{
								ExitNodeAllowLANAccess: newValue,
							},
							ExitNodeAllowLANAccessSet: true,
						})
					},
				),

				ui.NewSettingsSubmenuItem("Advertise Exit Node",
					[]string{"Exit Node", "No"},
					exitNode,
					func(newLabel string) tea.Msg {
						var prefs ipn.Prefs
						prefs.SetAdvertiseExitNode(newLabel == "Exit Node")
						return editPrefs(&ipn.MaskedPrefs{
							Prefs:              prefs,
							AdvertiseRoutesSet: true,
						})
					},
				),

				&ui.SpacerSubmenuItem{},
				&ui.TitleSubmenuItem{Label: accountTitle},

				&ui.LabeledSubmenuItem{
					Label: reauthenticateButtonLabel,
					OnActivate: func() tea.Msg {
						// Reauthenticating is basically the same as the first-time login flow.
						err := libts.StartLoginInteractive(ctx)
						if err != nil {
							return errorMsg(err)
						}
						return successMsg("Starting reauthentication. This may take a few seconds.")
					},
				},

				&ui.LabeledSubmenuItem{
					Label:   "[Log Out]",
					Variant: ui.SubmenuItemVariantDanger,
					OnActivate: func() tea.Msg {
						err := libts.Logout(ctx)
						if err != nil {
							return errorMsg(err)
						}
						return successMsg("Logged out.")
					},
				},
			}

			// On Linux, show the advanced Linux settings.
			if runtime.GOOS == "linux" {
				var netfilterMode string
				switch m.state.Prefs.NetfilterMode {
				case preftype.NetfilterOn:
					netfilterMode = "On"
				case preftype.NetfilterNoDivert:
					netfilterMode = "No Divert"
				case preftype.NetfilterOff:
					netfilterMode = "Off"
				}

				noStatefulFiltering, _ := m.state.Prefs.NoStatefulFiltering.Get()

				submenuItems = append(submenuItems,
					&ui.SpacerSubmenuItem{},
					&ui.TitleSubmenuItem{Label: "Advanced - Linux"},

					ui.NewSettingsSubmenuItem("NetFilter Mode",
						[]string{"On", "No Divert", "Off"},
						netfilterMode,
						func(newLabel string) tea.Msg {
							var netfilterMode preftype.NetfilterMode
							switch newLabel {
							case "On":
								netfilterMode = preftype.NetfilterOn
							case "No Divert":
								netfilterMode = preftype.NetfilterNoDivert
							case "Off":
								netfilterMode = preftype.NetfilterOff
							}

							return editPrefs(&ipn.MaskedPrefs{
								Prefs: ipn.Prefs{
									NetfilterMode: netfilterMode,
								},
								NetfilterModeSet: true,
							})
						},
					),

					ui.NewYesNoSettingsSubmenuItem("Enable Stateful Filtering",
						!noStatefulFiltering,
						func(newValue bool) tea.Msg {
							return editPrefs(&ipn.MaskedPrefs{
								Prefs: ipn.Prefs{
									NoStatefulFiltering: opt.NewBool(!newValue),
								},
								NoStatefulFilteringSet: true,
							})
						},
					),
				)
			}

			m.settings.Submenu.SetItems(submenuItems)
		}

		// Make sure the menu items are visible.
		m.menu.SetItems([]*ui.AppmenuItem{
			m.deviceInfo,
			m.exitNodes,
			m.settings,
		})
	} else {
		// Hide the menu items if not connected.
		// I mean, they won't be visible anyway, but extra safety is always nice!
		m.menu.SetItems([]*ui.AppmenuItem{})
	}
}

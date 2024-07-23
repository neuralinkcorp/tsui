package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neuralink/tsui/libts"
	"tailscale.com/ipn/ipnstate"
)

// Message triggered on each poller tick.
type tickMsg struct{}

// Creates a tea.Tick command that generates tickMsg messages.
func makeTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// Message representing a Tailscale state update.
type stateMsg libts.State

// Command that retrives a new Tailscale state and triggers a stateMsg.
// This will be run in a goroutine by the bubbletea runtime.
func getState() tea.Msg {
	status, _ := libts.Status(ctx)
	state := libts.MakeState(status)
	return stateMsg(state)
}

// Bubbletea update function.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Create our ticker command which will be our "default return" in the absence of any other commands.
	tick := makeTick(5 * m.tickInterval)

	if m.searchString == "" {
		m.refreshFilteredExitNodes()
	}

	switch msg := msg.(type) {
	// On tick, fetch a new state.
	case tickMsg:
		return m, getState

	// When the state updater returns, update our model.
	case stateMsg:
		m.state = libts.State(msg)
		m.refreshFilteredExitNodes()

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

	case tea.KeyMsg:
		switch m.keyMode {
		case normalKeyMode:
			return m.handleNormalModeKeys(msg, tick)
		case searchKeyMode:
			return m.handleSearchModeKeys(msg, tick)
		}
	}

	m.validateSelectedExitNode()
	return m, tick
}

func (m *model) findSelectedExitNode() (int, *ipnstate.PeerStatus) {
	if m.selectedExitNode == "" {
		return -1, nil
	}
	for i, node := range m.filteredExitNodes {
		if node.ID == m.selectedExitNode {
			return i, node
		}
	}
	return -1, nil
}

func (m *model) validateSelectedExitNode() {
	// Make sure the exit node cursor doesn't point to a non-existing node.
	if m.selectedExitNode != "" {
		i, _ := m.findSelectedExitNode()
		if i == -1 {
			m.selectedExitNode = ""
		}
	}
}

func (m *model) selectPreviousExitNode() {
	i, _ := m.findSelectedExitNode()
	if i > 0 {
		m.selectedExitNode = m.filteredExitNodes[i-1].ID
	} else {
		m.selectedExitNode = ""
	}
}

func (m *model) selectNextExitNode() {
	i, _ := m.findSelectedExitNode()
	if i+1 < len(m.filteredExitNodes) {
		m.selectedExitNode = m.filteredExitNodes[i+1].ID
	}
}

func (m *model) handleNormalModeKeys(msg tea.KeyMsg, tick tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "/":
		m.keyMode = searchKeyMode
		return m, tick

	case "ctrl+c", "q":
		return m, tea.Quit

	case "up":
		m.selectPreviousExitNode()

	case "down":
		m.selectNextExitNode()

	case "r":
		m.refreshFilteredExitNodes()

	// Select exit node on enter or space.
	case "enter", " ":
		// If they have an exit node selected and not "None", use it.
		_, exitNode := m.findSelectedExitNode()
		return m.selectExitNode(exitNode, tick)
	}
	return m, tick
}

func (m *model) selectExitNode(exitNode *ipnstate.PeerStatus, tick tea.Cmd) (tea.Model, tea.Cmd) {
	// If this exit node is already selected, don't do anything.
	if exitNode != nil && m.state.CurrentExitNode != nil && *m.state.CurrentExitNode == exitNode.ID {
		return m, tick
	}

	// Eagerly update the model, and therefore the UI.
	if exitNode == nil {
		m.state.CurrentExitNode = nil
	} else {
		m.state.CurrentExitNode = &exitNode.ID
	}

	// Asynchronously use the Tailscale API to update the exit node.
	cmd := func() tea.Msg {
		libts.SetExitNode(ctx, exitNode)
		return nil
	}
	return m, cmd
}

func isHostnameCharacter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-'
}

func (m *model) handleSearchModeKeys(msg tea.KeyMsg, tick tea.Cmd) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "esc":
		m.keyMode = normalKeyMode
		m.searchString = ""
		m.refreshFilteredExitNodes()

	case len(key) == 1 && isHostnameCharacter(key[0]):
		m.searchString += key
		m.refreshFilteredExitNodes()

	case key == tea.KeyBackspace.String() && len(m.searchString) > 0:
		m.searchString = m.searchString[:len(m.searchString)-1]
		m.refreshFilteredExitNodes()

	case key == "up":
		m.selectPreviousExitNode()

	case key == "down":
		m.selectNextExitNode()

	case key == "enter":
		// If they have an exit node selected and not "None", use it.
		_, exitNode := m.findSelectedExitNode()
		return m.selectExitNode(exitNode, tick)
	}
	return m, tick
}

func matchesSearchString(haystack string, needle string) bool {
	haystackPos := 0
	for _, b := range []byte(needle) {
		if haystackPos >= len(haystack) {
			return false
		}
		found := false
		for ; haystackPos < len(haystack) && !found; haystackPos++ {
			if haystack[haystackPos] == b {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (m *model) refreshFilteredExitNodes() {
	if m.searchString == "" {
		m.filteredExitNodes = m.state.SortedExitNodes
	}
	m.filteredExitNodes = nil
	for _, choice := range m.state.SortedExitNodes {
		isActive := m.state.CurrentExitNode != nil && choice.ID == *m.state.CurrentExitNode
		name := libts.PeerName(choice)
		isSelected := m.selectedExitNode == choice.ID
		if isActive || isSelected || matchesSearchString(name, m.searchString) {
			m.filteredExitNodes = append(m.filteredExitNodes, choice)
		}
	}
}

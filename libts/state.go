package libts

import (
	"slices"
	"strings"

	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

// Opinionated, sanitized subset of Tailscale state.
type State struct {
	// Current Tailscale backend state.
	//  "NoState", "NeedsLogin", "NeedsMachineAuth", "Stopped",
	//  "Starting", "Running".
	BackendState string
	// Auth URL. Empty if the user doesn't need to be authenticated.
	AuthURL string
	// Current Tailscale version. This is a shortened version string like "1.70.0".
	TSVersion string
	// ID of the currently selected exit node or nil if none is selected.
	CurrentExitNode *tailcfg.StableNodeID
	// User profile of the currently logged in user or nil if unknown.
	User *tailcfg.UserProfile
	// List of exit node peers, alphabetically pre-sorted by the result of the PeerName function.
	SortedExitNodes []*ipnstate.PeerStatus
}

// Get a sorted list of exit node peers, alphabetically pre-sorted by the result of the PeerName function.
func getSortedExitNodes(tsStatus *ipnstate.Status) []*ipnstate.PeerStatus {
	exitNodes := make([]*ipnstate.PeerStatus, 0)

	if tsStatus == nil {
		return exitNodes
	}

	for _, peer := range tsStatus.Peer {
		if peer.ExitNodeOption {
			exitNodes = append(exitNodes, peer)
		}
	}

	slices.SortFunc(exitNodes, func(a, b *ipnstate.PeerStatus) int {
		return strings.Compare(PeerName(a), PeerName(b))
	})

	return exitNodes
}

// Make a State from an ipnstate.Status. Safely returns an empty state value if the status is nil.
func MakeState(tsStatus *ipnstate.Status) State {
	if tsStatus == nil {
		return State{
			BackendState: ipn.NoState.String(),
		}
	}

	state := State{
		AuthURL:         tsStatus.AuthURL,
		BackendState:    tsStatus.BackendState,
		TSVersion:       tsStatus.Version,
		SortedExitNodes: getSortedExitNodes(tsStatus),
	}

	versionSplitIndex := strings.IndexByte(state.TSVersion, '-')
	if versionSplitIndex != -1 {
		state.TSVersion = state.TSVersion[:versionSplitIndex]
	}

	if tsStatus.Self != nil {
		user := tsStatus.User[tsStatus.Self.UserID]
		state.User = &user
	}

	if tsStatus.ExitNodeStatus != nil {
		state.CurrentExitNode = &tsStatus.ExitNodeStatus.ID
	}

	return state
}

package libts

import (
	"math"
	"slices"
	"strings"

	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
)

// Opinionated, sanitized subset of Tailscale state.
type State struct {
	// Tailscale preferences.
	Prefs *ipn.Prefs

	// Current Tailscale backend state.
	//  "NoState", "NeedsLogin", "NeedsMachineAuth", "Stopped",
	//  "Starting", "Running".
	BackendState string
	// Current Tailscale version. This is a shortened version string like "1.70.0".
	TSVersion string

	// Auth URL. Empty if the user doesn't need to be authenticated.
	AuthURL string
	// User profile of the currently logged in user or nil if unknown.
	User *tailcfg.UserProfile

	// Peer status of the local node.
	Self *ipnstate.PeerStatus

	// Tailnet lock key. Nil if not enabled.
	LockKey *key.NLPublic
	// True if the node is locked out by tailnet lock.
	IsLockedOut bool

	// State of each peer, keyed by each peer's current public key.
	Peers []*ipnstate.PeerStatus
	// List of exit node peers, alphabetically pre-sorted by the result of the PeerName function.
	SortedExitNodes []*ipnstate.PeerStatus
	// ID of the currently selected exit node or nil if none is selected.
	CurrentExitNode *tailcfg.StableNodeID
	// Name of the currently selected exit node or an empty string if none is selected.
	CurrentExitNodeName string
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

// Get a sorted network node list, sorted by the amount of traffic we're doing to that node
func getSortedNetworkNodes(tsStatus *ipnstate.Status) []*ipnstate.PeerStatus {
	networkNodes := make([]*ipnstate.PeerStatus, 0)

	if tsStatus == nil {
		return networkNodes
	}

	for _, peer := range tsStatus.Peer {
		networkNodes = append(networkNodes, peer)
	}

	slices.SortFunc(networkNodes, func(a, b *ipnstate.PeerStatus) int {
		trafficDiff := int((a.RxBytes + a.TxBytes) - (b.RxBytes + b.TxBytes))

		// if there is a noticeable traffic difference, prefer the more trafficked node
		if math.Abs(float64(trafficDiff)) > 500000 { // 500KB
			return trafficDiff
		}

		// if one of the nodes is online and the other is offline, prefer the one that is online
		if a.Online && !b.Online {
			return -1
		}

		if !a.Online && b.Online {
			return 1
		}

		// otherwise, sort alphabetically
		return strings.Compare(PeerName(a), PeerName(b))
	})

	return networkNodes
}

// Make a State from an ipnstate.Status. Safely returns an empty state value if the status is nil.
func MakeState(status *ipnstate.Status, prefs *ipn.Prefs, lock *ipnstate.NetworkLockStatus) State {
	if status == nil {
		return State{
			Prefs:        prefs,
			BackendState: ipn.NoState.String(),
		}
	}

	state := State{
		Prefs:           prefs,
		AuthURL:         status.AuthURL,
		BackendState:    status.BackendState,
		TSVersion:       status.Version,
		Self:            status.Self,
		SortedExitNodes: getSortedExitNodes(status),
		Peers:           getSortedNetworkNodes(status),
	}

	versionSplitIndex := strings.IndexByte(state.TSVersion, '-')
	if versionSplitIndex != -1 {
		state.TSVersion = state.TSVersion[:versionSplitIndex]
	}

	if status.Self != nil {
		user := status.User[status.Self.UserID]
		state.User = &user
	}

	if lock.Enabled && lock.NodeKey != nil && !lock.PublicKey.IsZero() {
		state.LockKey = &lock.PublicKey

		if !lock.NodeKeySigned && state.BackendState == ipn.Running.String() {
			state.IsLockedOut = true
		}
	}

	if status.ExitNodeStatus != nil {
		state.CurrentExitNode = &status.ExitNodeStatus.ID

		for _, peer := range state.SortedExitNodes {
			if peer.ID == status.ExitNodeStatus.ID {
				state.CurrentExitNodeName = PeerName(peer)
				break
			}
		}
	}

	return state
}

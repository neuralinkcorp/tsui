package libts

import (
	"context"
	"fmt"

	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
)

var ts tailscale.LocalClient

// Return the Tailscale daemon status. Returns an error if the daemon is not running.
func Status(ctx context.Context) (*ipnstate.Status, error) {
	return ts.Status(ctx)
}

// Return the tailnet lock status of the current node.
func LockStatus(ctx context.Context) (*ipnstate.NetworkLockStatus, error) {
	return ts.NetworkLockStatus(ctx)
}

// Start the Tailscale daemon.
func Up(ctx context.Context) error {
	return setWantRunning(ctx, true)
}

// Stop the Tailscale daemon.
func Down(ctx context.Context) error {
	return setWantRunning(ctx, false)
}

// Start or stop the Tailscale daemon, if we already have a socket connection.
func setWantRunning(ctx context.Context, wantRunning bool) error {
	_, err := ts.EditPrefs(ctx, &ipn.MaskedPrefs{
		Prefs: ipn.Prefs{
			WantRunning: wantRunning,
		},
		WantRunningSet: true,
	})
	return err
}

// Set the exit node to the given peer, or clear the exit node if peer is nil.
func SetExitNode(ctx context.Context, peer *ipnstate.PeerStatus) error {
	var prefs ipn.Prefs

	if peer == nil {
		prefs.ClearExitNode()
	} else {
		status, err := ts.Status(ctx)
		if err != nil {
			return fmt.Errorf("get status: %w", err)
		}

		prefs.SetExitNodeIP(peer.TailscaleIPs[0].String(), status)
	}

	_, err := ts.EditPrefs(ctx, &ipn.MaskedPrefs{
		Prefs:         prefs,
		ExitNodeIDSet: true,
		ExitNodeIPSet: true,
	})
	if err != nil {
		return fmt.Errorf("edit prefs: %w", err)
	}

	return nil
}

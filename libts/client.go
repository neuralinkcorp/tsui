package libts

import (
	"context"
	"net/netip"

	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

var ts tailscale.LocalClient

// Return the Tailscale daemon status. Returns an error if the daemon is not running.
func Status(ctx context.Context) (*ipnstate.Status, error) {
	return ts.Status(ctx)
}

// Start an interactive login flow. This will automatically open the user's web browser.
// Note that this will NOT DO ANYTHING if the session has already started; i.e. an
// AuthURL is already populated in the state.
func StartLoginInteractive(ctx context.Context) error {
	// Workaround for a Tailscale bug (?) where the AuthURL isn't populated when calling
	// StartLoginInteractive the first time if the user is already logged in. For some reason,
	// calling Start first with no options makes the AuthURL populate.
	err := ts.Start(ctx, ipn.Options{})
	if err != nil {
		return err
	}

	return ts.StartLoginInteractive(ctx)
}

// Ping a peer.
func PingPeer(ctx context.Context, peer *ipnstate.PeerStatus) (*ipnstate.PingResult, error) {
	addr, err := netip.ParseAddr(peer.CurAddr)
	if err != nil {
		return nil, err
	}

	// Discovery ping is the most reliable because it doesn't rely on the host accepting ICMP or anything.
	// This is what `tailscale ping` uses by default.
	return ts.Ping(ctx, addr, tailcfg.PingDisco)
}

// Logs you out.
func Logout(ctx context.Context) error {
	return ts.Logout(ctx)
}

// Get current preferences.
func Prefs(ctx context.Context) (*ipn.Prefs, error) {
	return ts.GetPrefs(ctx)
}

// Update preferences.
func EditPrefs(ctx context.Context, maskedPrefs *ipn.MaskedPrefs) error {
	_, err := ts.EditPrefs(ctx, maskedPrefs)
	return err
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
			return err
		}

		prefs.SetExitNodeIP(peer.TailscaleIPs[0].String(), status)
	}

	_, err := ts.EditPrefs(ctx, &ipn.MaskedPrefs{
		Prefs:         prefs,
		ExitNodeIDSet: true,
		ExitNodeIPSet: true,
	})
	if err != nil {
		return err
	}

	return nil
}

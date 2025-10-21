package libts

import (
	"context"
	"runtime"

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

// Returns true if StartLoginInteractive will (probably) open the user's web browser.
// Can be used to decide whether to display UI elements related to interactive login.
func StartLoginInteractiveWillOpenBrowser() bool {
	return runtime.GOOS == "darwin"
}

// Start an interactive login flow. On macOS, this will automatically open the user's web browser.
func StartLoginInteractive(ctx context.Context) error {
	// Workaround for a Tailscale bug where Tailscale will go into the Starting... state
	// without populating the AuthURL when reauthenticating. For some reason, calling
	// Start first with no options makes the AuthURL populate.
	//
	// We need AuthURL so we can display UI elements related to the login process.
	err := ts.Start(ctx, ipn.Options{})
	if err != nil {
		return err
	}

	return ts.StartLoginInteractive(ctx)
}

// Ping a peer.
func PingPeer(ctx context.Context, peer *ipnstate.PeerStatus) (*ipnstate.PingResult, error) {
	// Discovery ping is the most reliable because it doesn't rely on the host accepting ICMP or anything.
	// This is what `tailscale ping` uses by default.
	return ts.Ping(ctx, peer.TailscaleIPs[0], tailcfg.PingDisco)
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

// Returns true if the user has write permissions to the Tailscale config.
// If false, the user may have to run tsui with sudo.
func CanWrite(ctx context.Context) bool {
	err := EditPrefs(ctx, &ipn.MaskedPrefs{})
	return err == nil
}

// Return the tailnet lock status of the current node.
func LockStatus(ctx context.Context) (*ipnstate.NetworkLockStatus, error) {
	return ts.NetworkLockStatus(ctx)
}

// Start the Tailscale daemon.
func Up(ctx context.Context) error {
	return EditPrefs(ctx, &ipn.MaskedPrefs{
		Prefs: ipn.Prefs{
			WantRunning: true,
		},
		WantRunningSet: true,
	})
}

// Stop the Tailscale daemon.
func Down(ctx context.Context) error {
	return EditPrefs(ctx, &ipn.MaskedPrefs{
		Prefs: ipn.Prefs{
			WantRunning: false,
		},
		WantRunningSet: true,
	})
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

// List all profiles on this Tailscale client.
func ListProfiles(ctx context.Context) ([]ipn.LoginProfile, error) {
	_, all, err := ts.ProfileStatus(ctx)
	if err != nil {
		return nil, err
	}
	return all, nil
}

// Switch to the given Tailscale profile.
func SwitchProfile(ctx context.Context, profileID ipn.ProfileID) error {
	return ts.SwitchProfile(ctx, profileID)
}

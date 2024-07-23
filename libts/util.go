package libts

import (
	"strings"

	"tailscale.com/ipn/ipnstate"
)

// Get the user-friendly name of a Tailscale peer such as "foobar-router-2".
// This is distinct from the device name, which can have duplicates on the network,
// and from the DNS name, which includes the tailnet suffix.
func PeerName(peer *ipnstate.PeerStatus) string {
	dotIndex := strings.IndexByte(peer.DNSName, '.')

	if dotIndex == -1 {
		return peer.DNSName
	}

	return peer.DNSName[:dotIndex]
}

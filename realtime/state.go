package realtime

import (
	"github.com/digineo/go-ping"
	"net"
	"tailscale.com/ipn/ipnstate"
	"time"
)

// A nullable latency type
type Latency int64

func GetExitNodeRtt(nodes []*ipnstate.PeerStatus) map[*ipnstate.PeerStatus]Latency {
	latency := make(map[*ipnstate.PeerStatus]Latency)

	if pinger, err := ping.New("0.0.0.0", ""); err == nil {
		for _, node := range nodes {
			ip, err := net.ResolveIPAddr("ip4", "1.1.1.1")
			if err != nil {
				panic(err)
			}
			rtt, err := pinger.PingAttempts(ip, time.Second, 1)
			if err != nil {
				panic(err)
			}
			latency[node] = Latency(rtt.Milliseconds())
		}
	}

	return latency
}

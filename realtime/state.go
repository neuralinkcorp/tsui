package realtime

import (
	"github.com/digineo/go-ping"
	"net"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tstime/rate"
	"time"
)

// A nullable latency type
type Latency *int64

var (
	LastLatencyResult  = make(map[string]Latency)
	LatencyTestLimiter = rate.NewLimiter(10, 1)
)

func GetExitNodeRtt(nodes []*ipnstate.PeerStatus) (map[string]Latency, error) {
	if LatencyTestLimiter.Allow() {
		latencies := make(map[string]Latency)

		if pinger, err := ping.New("0.0.0.0", ""); err == nil {
			for _, node := range nodes {
				// the first address is the ipv4 one
				ip, err := net.ResolveIPAddr("ip4", node.TailscaleIPs[0].String())
				if err != nil {
					return nil, err
				}

				// for obvious reasons, ping is highly fallible
				// fail silently
				rtt, _ := pinger.Ping(ip, 500*time.Millisecond)

				ms := rtt.Milliseconds()

				latencies[node.HostName] = &ms
			}
		}

		LastLatencyResult = latencies
	}

	return LastLatencyResult, nil
}

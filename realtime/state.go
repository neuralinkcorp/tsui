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
	LatencyAverage     = make(map[*ipnstate.PeerStatus][]Latency)
	LatencyTestLimiter = rate.NewLimiter(1, 1)
)

func GetExitNodeRtt(nodes []*ipnstate.PeerStatus) (map[*ipnstate.PeerStatus]Latency, error) {
	for node, lastTenTries := range LatencyAverage {
		if len(lastTenTries) > 0 {
			// kick out the last entry
			LatencyAverage[node] = lastTenTries[1:]
		}
	}

	if LatencyTestLimiter.Allow() {
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

				if nodeAverage := LatencyAverage[node]; len(nodeAverage) > 0 {
					LatencyAverage[node] = append(nodeAverage, &ms)
				} else {
					LatencyAverage[node] = []Latency{&ms}
				}
			}
		}
	}

	latency := make(map[*ipnstate.PeerStatus]Latency)

	for node, lastTenTries := range LatencyAverage {
		sum := int64(0)

		for _, m := range lastTenTries {
			sum += *m
		}

		if len(lastTenTries) != 0 {
			nodeLatency := sum / int64(len(lastTenTries))

			latency[node] = &nodeLatency
		} else {
			latency[node] = nil
		}
	}

	return latency, nil
}

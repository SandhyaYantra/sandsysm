package monitor

import (
	"context"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/username/sandsysm/internal/system"
)

type NetStat struct {
	Iface   string
	RxBytes uint64
	TxBytes uint64
	RxSpeed float64
	TxSpeed float64
	IP      string
}

func StartNet(ctx context.Context) <-chan []NetStat {
	ch := make(chan []NetStat, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		previous := make(map[string]system.NetDevStats)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				counters, err := system.ReadProcNetDev()
				if err != nil {
					continue
				}

				interfaces, err := net.Interfaces()
				if err != nil {
					continue
				}
				ipByInterface := make(map[string]string, len(interfaces))
				for _, iface := range interfaces {
					addresses, err := iface.Addrs()
					if err != nil {
						continue
					}
					for _, address := range addresses {
						ipText, _, _ := strings.Cut(address.String(), "/")
						ip := net.ParseIP(ipText)
						if ip == nil {
							continue
						}
						if ip.To4() != nil {
							ipByInterface[iface.Name] = ip.String()
							break
						}
						if ipByInterface[iface.Name] == "" {
							ipByInterface[iface.Name] = ip.String()
						}
					}
				}

				names := make([]string, 0, len(counters))
				for iface := range counters {
					names = append(names, iface)
				}
				sort.Strings(names)

				stats := make([]NetStat, 0, len(names))
				for _, iface := range names {
					current := counters[iface]
					prior, hasPrevious := previous[iface]
					rxSpeed := float64(0)
					txSpeed := float64(0)
					if hasPrevious && current.RX >= prior.RX {
						rxSpeed = float64(current.RX-prior.RX) / (1024 * 1024)
					}
					if hasPrevious && current.TX >= prior.TX {
						txSpeed = float64(current.TX-prior.TX) / (1024 * 1024)
					}
					stats = append(stats, NetStat{
						Iface:   iface,
						RxBytes: current.RX,
						TxBytes: current.TX,
						RxSpeed: rxSpeed,
						TxSpeed: txSpeed,
						IP:      ipByInterface[iface],
					})
					previous[iface] = current
				}
				select {
				case ch <- stats:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}

// Package discovery implements bounded IPv4 SSDP announcements for Curated Server.
package discovery

import (
	"bufio"
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/ipv4"
)

const ServiceType = "urn:curated:service:Library:1"
const group = "239.255.255.250:1900"
const MaxAge = 120

type Options struct {
	ID, HTTPAddr string
	Logger       *zap.Logger
}

// Start advertises only private interfaces that match the real HTTP binding.
// Failure is diagnostic only: HTTP and manual connections remain usable.
func Start(ctx context.Context, opts Options) func() {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	host, port, err := net.SplitHostPort(opts.HTTPAddr)
	if err != nil || opts.ID == "" {
		return func() {}
	}
	multicast, _ := net.ResolveUDPAddr("udp4", group)
	interfaces, err := net.Interfaces()
	if err != nil {
		logger.Warn("SSDP interfaces unavailable", zap.Error(err))
		return func() {}
	}
	var stops []func()
	for _, nic := range interfaces {
		if nic.Flags&net.FlagUp == 0 || nic.Flags&net.FlagMulticast == 0 || nic.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := nic.Addrs()
		for _, addr := range addrs {
			ip, _, parseErr := net.ParseCIDR(addr.String())
			if parseErr != nil || ip.To4() == nil || !ip.IsPrivate() || ip.IsLoopback() {
				continue
			}
			if host != "" && host != "0.0.0.0" && host != "::" && host != ip.String() {
				continue
			}
			socket, listenErr := net.ListenMulticastUDP("udp4", &nic, multicast)
			if listenErr != nil {
				logger.Warn("SSDP multicast unavailable", zap.String("interface", nic.Name), zap.Error(listenErr))
				continue
			}
			packet := ipv4.NewPacketConn(socket)
			_ = packet.SetMulticastInterface(&nic)
			_ = packet.SetMulticastTTL(2)
			_ = packet.SetMulticastLoopback(true)
			location := "http://" + net.JoinHostPort(ip.String(), port) + "/discovery/description.xml"
			stopCtx, cancel := context.WithCancel(ctx)
			done := make(chan struct{})
			notify := func(state string) {
				_, _ = socket.WriteToUDP([]byte(Notification(opts.ID, location, state)), multicast)
			}
			notify("ssdp:alive")
			go func() {
				ticker := time.NewTicker(60 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-stopCtx.Done():
						return
					case <-ticker.C:
						notify("ssdp:alive")
					}
				}
			}()
			go func() {
				defer close(done)
				buffer := make([]byte, 2048)
				// One pending response per socket, plus a small global response rate bound.
				var nextResponse time.Time
				for {
					_ = socket.SetReadDeadline(time.Now().Add(time.Second))
					n, peer, err := socket.ReadFromUDP(buffer)
					if err != nil {
						if stopCtx.Err() != nil {
							return
						}
						if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
							continue
						}
						return
					}
					mx, valid := SearchMX(string(buffer[:n]))
					if !valid || !peer.IP.IsPrivate() || peer.IP.IsLoopback() || time.Now().Before(nextResponse) {
						continue
					}
					timer := time.NewTimer(time.Duration(rand.IntN(mx*1000)) * time.Millisecond)
					select {
					case <-stopCtx.Done():
						timer.Stop()
						return
					case <-timer.C:
					}
					_, _ = socket.WriteToUDP([]byte(Response(opts.ID, location)), peer)
					nextResponse = time.Now().Add(100 * time.Millisecond)
				}
			}()
			var once sync.Once
			stops = append(stops, func() { once.Do(func() { cancel(); notify("ssdp:byebye"); _ = socket.Close(); <-done }) })
			break // One advertised IPv4 address per interface.
		}
	}
	var once sync.Once
	stop := func() {
		once.Do(func() {
			for _, closeSocket := range stops {
				closeSocket()
			}
		})
	}
	go func() { <-ctx.Done(); stop() }()
	return stop
}

func SearchMX(message string) (int, bool) {
	if len(message) > 2048 {
		return 0, false
	}
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(message)))
	if err != nil {
		return 0, false
	}
	defer req.Body.Close()
	if req.Method != "M-SEARCH" || req.RequestURI != "*" || req.Host != group || req.Header.Get("Man") != `"ssdp:discover"` || req.Header.Get("St") != ServiceType {
		return 0, false
	}
	mx, err := strconv.Atoi(req.Header.Get("Mx"))
	return mx, err == nil && mx >= 1 && mx <= 5
}
func Response(id, location string) string {
	return fmt.Sprintf("HTTP/1.1 200 OK\r\nCACHE-CONTROL: max-age=%d\r\nEXT:\r\nLOCATION: %s\r\nSERVER: Curated/1 UPnP/1.1\r\nST: %s\r\nUSN: uuid:%s::%s\r\n\r\n", MaxAge, location, ServiceType, id, ServiceType)
}
func Notification(id, location, state string) string {
	return fmt.Sprintf("NOTIFY * HTTP/1.1\r\nHOST: %s\r\nCACHE-CONTROL: max-age=%d\r\nLOCATION: %s\r\nNT: %s\r\nNTS: %s\r\nUSN: uuid:%s::%s\r\n\r\n", group, MaxAge, location, ServiceType, state, id, ServiceType)
}

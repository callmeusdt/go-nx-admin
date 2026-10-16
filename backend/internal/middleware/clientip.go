package middleware

import (
	"log"
	"net"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"go-nx-admin/internal/config"
)

// ClientIP is the single source of truth for the caller's address.
//
// The TCP peer is the only value that cannot be forged, so forwarding headers
// are honoured only when the peer is a trusted proxy (loopback unless
// server.trusted_proxies says otherwise) and only the one header named by
// server.client_ip_header is read. Cookie mode keeps its own BrowserIP rule.
func ClientIP(c *fiber.Ctx) string {
	if CookieMode() {
		return BrowserIP(c)
	}
	peer := c.Context().RemoteIP()
	if config.AppConfig == nil {
		return peer.String()
	}
	server := config.AppConfig.Server
	header := strings.TrimSpace(server.ClientIPHeader)
	trusted := trustedProxies(server.TrustedProxies)
	if header == "" || !trusted.contains(peer) {
		return peer.String()
	}
	value := strings.TrimSpace(c.Get(header))
	if value == "" {
		return peer.String()
	}
	if !strings.EqualFold(header, "X-Forwarded-For") {
		if ip := net.ParseIP(value); ip != nil {
			return ip.String()
		}
		return peer.String()
	}
	// X-Forwarded-For is appended to by every hop, so only the right-hand side
	// is vouched for by our proxies. The first untrusted hop from the right is
	// the client; anything further left is whatever the client chose to send.
	hops := strings.Split(value, ",")
	for i := len(hops) - 1; i >= 0; i-- {
		ip := net.ParseIP(strings.TrimSpace(hops[i]))
		if ip == nil {
			return peer.String()
		}
		if !trusted.contains(ip) || i == 0 {
			return ip.String()
		}
	}
	return peer.String()
}

type proxySet []*net.IPNet

func (s proxySet) contains(ip net.IP) bool {
	for _, n := range s {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

var (
	proxyMu    sync.Mutex
	proxyKey   string
	proxyCache proxySet
)

func trustedProxies(entries []string) proxySet {
	key := strings.Join(entries, ",")
	proxyMu.Lock()
	defer proxyMu.Unlock()
	if proxyCache != nil && key == proxyKey {
		return proxyCache
	}
	set := proxySet{}
	if len(entries) == 0 {
		entries = []string{"127.0.0.0/8", "::1/128"}
	}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				entry = ip.String() + "/32"
				if ip.To4() == nil {
					entry = ip.String() + "/128"
				}
			}
		}
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			log.Printf("server.trusted_proxies: ignoring %q: %v", entry, err)
			continue
		}
		set = append(set, network)
	}
	proxyKey, proxyCache = key, set
	return set
}

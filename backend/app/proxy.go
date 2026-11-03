package server

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ParseTrustedProxies accepts explicit network ranges, never hostnames whose DNS
// answers could change the request trust boundary after deployment.
func ParseTrustedProxies(value string) ([]netip.Prefix, error) {
	prefixes := []netip.Prefix{}
	if strings.TrimSpace(value) == "" {
		return prefixes, nil
	}
	for _, part := range strings.Split(value, ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(part))
		if err != nil || prefix.Bits() == 0 || prefix.Addr().Is4In6() {
			return nil, errors.New("TRUSTED_PROXY_CIDRS must contain explicit IPv4/IPv6 CIDRs, without a default route")
		}
		prefixes = append(prefixes, prefix.Masked())
		if len(prefixes) > 32 {
			return nil, errors.New("at most 32 trusted proxy CIDRs are supported")
		}
	}
	return prefixes, nil
}
func trustedProxy(address netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

// requestPeer walks from the socket toward the client, stopping at the first
// untrusted hop. A client-prepended address can never jump that boundary. The
// ingress must overwrite X-Forwarded-For or append its actual socket peer.
func requestPeer(r *http.Request, prefixes []netip.Prefix) string {
	peer := r.RemoteAddr
	if host, _, err := net.SplitHostPort(peer); err == nil {
		peer = host
	}
	address, err := netip.ParseAddr(peer)
	if err != nil {
		return peer
	}
	address = address.Unmap()
	fallback := address.String()
	if !trustedProxy(address, prefixes) {
		return fallback
	}
	raw := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
	if len(raw) == 0 || len(raw) > 4096 {
		return fallback
	}
	hops := strings.Split(raw, ",")
	if len(hops) > 32 {
		return fallback
	}
	// Validate the entire header before accepting any address; malformed chains
	// fall back to the authenticated network boundary rather than guessing.
	addresses := make([]netip.Addr, len(hops))
	for i, hop := range hops {
		parsed, err := netip.ParseAddr(strings.TrimSpace(hop))
		if err != nil || parsed.Zone() != "" {
			return fallback
		}
		addresses[i] = parsed.Unmap()
	}
	for i := len(addresses) - 1; i >= 0; i-- {
		if !trustedProxy(address, prefixes) {
			break
		}
		address = addresses[i]
	}
	return address.String()
}

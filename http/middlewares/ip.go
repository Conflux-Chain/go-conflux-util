package middlewares

import (
	"context"
	"net"
	"net/http"
	"strings"
)

var CtxKeyRealIP = CtxKey("X-Real-IP")

// RealIP injects the address selected by GetIPAddress into the request context.
// It does not verify proxy identity; the trusted-proxy deployment requirements
// documented by GetIPAddress also apply to this middleware.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), CtxKeyRealIP, GetIPAddress(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRealIPFromContext get real remote IP address from the specified `ctx`.
func GetRealIPFromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(CtxKeyRealIP).(string)
	return val, ok
}

// GetIPAddress selects an IP address from X-Forwarded-For, then X-Real-IP,
// falling back to RemoteAddr when neither header contains an eligible address.
// Each header is scanned from right to left for an address that passes the IP
// filters. In a multi-proxy chain, this may select a proxy rather than the client.
// Header addresses must be global unicast and outside private and explicitly
// excluded subnets. These filters do not apply to the RemoteAddr fallback.
//
// This function trusts forwarding headers without verifying the connection's
// peer. Services using it must only accept requests through trusted proxies that
// remove or rebuild client-supplied forwarding headers. Setting X-Real-IP alone
// is insufficient if an untrusted X-Forwarded-For header is preserved, since
// X-Forwarded-For takes precedence.
//
// IP filtering does not establish trust. Without this deployment boundary,
// clients can spoof the result; it must not be relied on for rate limiting or
// access control in that case.
func GetIPAddress(r *http.Request) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		addresses := strings.Split(r.Header.Get(h), ",")

		// Select the rightmost address that passes the filters. This does not
		// validate the proxy chain or guarantee that the address is the client.
		for i := len(addresses) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(addresses[i])
			// header can contain spaces too, strip those out.
			realIP := net.ParseIP(ip)
			if realIP.IsGlobalUnicast() && !isPrivateSubnet(realIP) {
				return ip
			}
		}
	}

	// otherwise, use the RemoteAddr from HTTP request
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}

	return r.RemoteAddr
}

// Preserve the existing exclusions beyond RFC 1918 private space. These special
// ranges are not all private networks; this is not a complete routability list.
var excludedIPSubnets = []net.IPNet{
	{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}, // shared address space
	{IP: net.IPv4(192, 0, 0, 0), Mask: net.CIDRMask(24, 32)},  // IETF protocol assignments
	{IP: net.IPv4(198, 18, 0, 0), Mask: net.CIDRMask(15, 32)}, // benchmarking
}

// isPrivateSubnet reports whether ip is private (IPv4 RFC 1918 or IPv6 ULA)
// or belongs to one of the additional subnets excluded from forwarding headers.
func isPrivateSubnet(ip net.IP) bool {
	if ip.IsPrivate() {
		return true
	}

	for _, subnet := range excludedIPSubnets {
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}

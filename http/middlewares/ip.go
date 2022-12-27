package middlewares

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
)

var CtxKeyRealIP = CtxKey("X-Real-IP")

// RealIP injects real remote IP address in context.
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

// GetIPAddress returns the real remote IP address.
func GetIPAddress(r *http.Request) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		addresses := strings.Split(r.Header.Get(h), ",")

		// march from right to left until we get a public address
		// that will be the address right before our proxy.
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

// Remote IP Address with Go:
// https://husobee.github.io/golang/ip-address/2015/12/17/remote-ip-go.html

// ipRange is a structure that holds the start and end of a range of ip addresses.
type ipRange struct {
	start net.IP // inclusive
	end   net.IP // exclusive
}

func parseIpRange(start, end string) ipRange {
	return ipRange{
		start: net.ParseIP(start),
		end:   net.ParseIP(end),
	}
}

// includes checks if the specified `ip` is within the range.
func (rng *ipRange) includes(ip net.IP) bool {
	return bytes.Compare(ip, rng.start) >= 0 && bytes.Compare(ip, rng.end) < 0
}

var privateIpRanges = []ipRange{
	parseIpRange("10.0.0.0", "10.255.255.255"),
	parseIpRange("100.64.0.0", "100.127.255.255"),
	parseIpRange("172.16.0.0", "172.31.255.255"),
	parseIpRange("192.0.0.0", "192.0.0.255"),
	parseIpRange("192.168.0.0", "192.168.255.255"),
	parseIpRange("198.18.0.0", "198.19.255.255"),
}

// isPrivateSubnet - check to see if this ip is in a private subnet
func isPrivateSubnet(ip net.IP) bool {
	// my use case is only concerned with ipv4 atm
	if ipv4 := ip.To4(); ipv4 != nil {
		// iterate over all our ranges
		for _, r := range privateIpRanges {
			// check if this ip is in a private range
			if r.includes(ipv4) {
				return true
			}
		}
	}

	return false
}

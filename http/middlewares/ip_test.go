package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetIPAddressFiltering(t *testing.T) {
	for _, tc := range []struct {
		ip       string
		filtered bool
	}{
		{"9.255.255.255", false},
		{"10.0.0.0", true},
		{"10.255.255.255", true},
		{"11.0.0.0", false},
		{"100.63.255.255", false},
		{"100.64.0.0", true},
		{"100.127.255.255", true},
		{"100.128.0.0", false},
		{"172.15.255.255", false},
		{"172.16.0.0", true},
		{"172.31.255.255", true},
		{"172.32.0.0", false},
		{"191.255.255.255", false},
		{"192.0.0.0", true},
		{"192.0.0.255", true},
		{"192.0.1.0", false},
		{"192.167.255.255", false},
		{"192.168.0.0", true},
		{"192.168.255.255", true},
		{"192.169.0.0", false},
		{"198.17.255.255", false},
		{"198.18.0.0", true},
		{"198.19.255.255", true},
		{"198.20.0.0", false},
		{"fc00::", true},
		{"fd00::1", true},
		{"fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", true},
		{"2606:4700:4700::1111", false},
		{"::ffff:10.0.0.1", true},
		{"::ffff:100.127.255.255", true},
		{"::ffff:8.8.8.8", false},
		{"8.8.8.8", false},
		{"127.0.0.1", true},
		{"::1", true},
		{"fe80::1", true},
		{"224.0.0.1", true},
		{"invalid", true},
	} {
		t.Run(tc.ip, func(t *testing.T) {
			want := tc.ip
			if tc.filtered {
				want = "9.9.9.9"
			}
			for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = "9.9.9.9:1234"
				r.Header.Set(header, tc.ip)
				assert.Equal(t, want, GetIPAddress(r), header)
			}
		})
	}
}

func TestGetIPAddressHeaderSelection(t *testing.T) {
	for _, tc := range []struct {
		name string
		xff  string
		xri  string
		want string
	}{
		{"skip private IPv4", "8.8.8.8, 10.0.0.1", "", "8.8.8.8"},
		{"skip private IPv6", "2606:4700:4700::1111, fd00::1", "", "2606:4700:4700::1111"},
		{"skip special range", "8.8.8.8, 100.127.255.255", "", "8.8.8.8"},
		{"rightmost eligible address", "1.1.1.1, 8.8.8.8, 10.0.0.1", "1.0.0.1", "8.8.8.8"},
		{"fallback to real IP header", "10.0.0.1, fd00::1", "8.8.8.8", "8.8.8.8"},
		{"fallback to peer", "10.0.0.1", "fd00::1", "192.168.1.2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = "192.168.1.2:1234"
			r.Header.Set("X-Forwarded-For", tc.xff)
			r.Header.Set("X-Real-IP", tc.xri)
			assert.Equal(t, tc.want, GetIPAddress(r))
		})
	}
}

func TestGetIPAddressRemoteAddr(t *testing.T) {
	for _, tc := range []struct {
		name   string
		remote string
		want   string
	}{
		{"IPv4 with port", "192.168.1.2:1234", "192.168.1.2"},
		{"IPv4 without port", "192.168.1.2", "192.168.1.2"},
		{"IPv6 with port", "[2001:db8::1]:1234", "2001:db8::1"},
		{"IPv6 without port", "2001:db8::1", "2001:db8::1"},
		{"IPv6 zone with port", "[fe80::1%eth0]:1234", "fe80::1%eth0"},
		{"IPv6 zone without port", "fe80::1%eth0", "fe80::1%eth0"},
		{"empty", "", ""},
		{"nonstandard", "unknown", "unknown"},
		{"malformed brackets", "[2001:db8::1", "[2001:db8::1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remote
			assert.Equal(t, tc.want, GetIPAddress(r))
		})
	}
}

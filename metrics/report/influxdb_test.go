package report

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMakeClientClosesOldConnections(t *testing.T) {
	closed := make(chan struct{}, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateClosed {
			closed <- struct{}{}
		}
	}
	server.Start()
	defer server.Close()

	reporter := influxdbReporter{config: InfluxDBConfig{Host: server.URL, Timeout: time.Second}}
	if err := reporter.makeClient(); err != nil {
		t.Fatal(err)
	}
	oldClient := reporter.client
	defer oldClient.Close()

	// A failed Ping can leave a reusable connection in the old client's pool.
	if _, _, err := oldClient.Ping(0); err == nil {
		t.Fatal("expected Ping to fail with HTTP 503")
	}
	if err := reporter.makeClient(); err != nil {
		t.Fatal(err)
	}
	defer reporter.client.Close()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("replacing the client did not close its old idle connection")
	}
}

func TestMakeClientPreservesClientOnError(t *testing.T) {
	reporter := influxdbReporter{config: InfluxDBConfig{Host: "http://127.0.0.1:8086"}}
	if err := reporter.makeClient(); err != nil {
		t.Fatal(err)
	}
	oldClient := reporter.client
	defer oldClient.Close()

	reporter.config.Host = "://invalid"
	if err := reporter.makeClient(); err == nil {
		t.Fatal("expected an error for the invalid host")
	}
	if reporter.client != oldClient {
		t.Fatal("failed client creation replaced the existing client")
	}
}

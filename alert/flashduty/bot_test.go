package flashduty

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	robot *Robot
)

// Please set the following environments before running tests:
// `TEST_FLASHDUTY_WEBHOOK`: FlashDuty webhook;
// `TEST_FLASHDUTY_SECRET`: FlashDuty secret.

func TestMain(m *testing.M) {
	webhook := os.Getenv("TEST_FLASHDUTY_WEBHOOK")
	secret := os.Getenv("TEST_FLASHDUTY_SECRET")

	if len(webhook) > 0 && len(secret) > 0 {
		robot = NewRobot(webhook, secret)
	}

	os.Exit(m.Run())
}

func TestSend(t *testing.T) {
	if robot == nil {
		t.SkipNow()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := robot.Send(ctx, "test", MsgLevelInfo, "# Hello, test!", "description", map[string]string{"service": "engine", "resourcetype": "cpu"})
	assert.NoError(t, err)
}

func TestSendResponse(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		response  string
		wantError string
	}{
		{"success", http.StatusOK, `{"request_id":"test","data":{"alert_key":"example"}}`, ""},
		{"provider error", http.StatusOK, `{"error":{"code":"Unauthorized","message":"invalid integration key"}}`, "flashduty error: code = Unauthorized, message = invalid integration key"},
		{"unauthorized", http.StatusUnauthorized, `{}`, "flashduty send failed: HTTP 401 Unauthorized"},
		{"rate limited", http.StatusTooManyRequests, `{}`, "flashduty send failed: HTTP 429 Too Many Requests"},
		{"server error", http.StatusInternalServerError, `{}`, "flashduty send failed: HTTP 500 Internal Server Error"},
		{"non-JSON error", http.StatusBadGateway, "upstream unavailable", "flashduty send failed: HTTP 502 Bad Gateway"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.response)
			}))
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := NewRobot(server.URL, "").Send(ctx, "test", MsgLevelInfo, "", "description", nil)
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantError)
			}
		})
	}
}

func TestSendQueryParameters(t *testing.T) {
	for _, tc := range []struct {
		name, path, secret string
		wantQuery          url.Values
	}{
		{"no query", "/hook", "test-key", url.Values{"integration_key": {"test-key"}}},
		{"existing query and special characters", "/hook?source=app", "test+a&extra=1", url.Values{"source": {"app"}, "integration_key": {"test+a&extra=1"}}},
		{"override key", "/hook?integration_key=old&integration_key=older", "new-key", url.Values{"integration_key": {"new-key"}}},
		{"key in URL", "/hook?integration_key=url%2Bkey&source=app", "", url.Values{"source": {"app"}, "integration_key": {"url+key"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/hook", r.URL.Path)
				assert.Equal(t, tc.wantQuery, r.URL.Query())
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"data":{"alert_key":"test"}}`)
			}))
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := NewRobot(server.URL+tc.path, tc.secret).Send(ctx, "test", MsgLevelInfo, "", "", nil)
			assert.NoError(t, err)
		})
	}
}

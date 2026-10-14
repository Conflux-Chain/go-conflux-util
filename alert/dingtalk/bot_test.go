package dingtalk

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	robot *Robot
)

// Please set the following environments before running tests:
// `TEST_DINGTALK_WEBHOOK`: DingTalk webhook;
// `TEST_DINGTALK_SECRET`: DingTalk secret.

func TestMain(m *testing.M) {
	webhook := os.Getenv("TEST_DINGTALK_WEBHOOK")
	secret := os.Getenv("TEST_DINGTALK_SECRET")

	if len(webhook) > 0 && len(secret) > 0 {
		robot = NewRobot(webhook, secret)
	}

	os.Exit(m.Run())
}

func TestSendMarkdown(t *testing.T) {
	if robot == nil {
		t.SkipNow()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Please manually check if message sent to dingtalk group chat
	err := robot.SendMarkdown(ctx, "test", "# Hello, test!", nil, false)
	assert.NoError(t, err)
}

func TestSendResponse(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		response  string
		wantError string
	}{
		{"success", http.StatusOK, `{"errcode":0,"errmsg":"ok"}`, ""},
		{"provider error", http.StatusOK, `{"errcode":310000,"errmsg":"invalid token"}`, "dingrobot send failed: invalid token"},
		{"unauthorized", http.StatusUnauthorized, `{}`, "dingtalk send failed: HTTP 401 Unauthorized"},
		{"rate limited", http.StatusTooManyRequests, `{}`, "dingtalk send failed: HTTP 429 Too Many Requests"},
		{"server error", http.StatusInternalServerError, `{}`, "dingtalk send failed: HTTP 500 Internal Server Error"},
		{"non-JSON error", http.StatusBadGateway, "upstream unavailable", "dingtalk send failed: HTTP 502 Bad Gateway"},
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
			err := NewRobot(server.URL, "").SendText(ctx, "test", nil, false)
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantError)
			}
		})
	}
}

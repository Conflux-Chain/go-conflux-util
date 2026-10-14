package flashduty

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
		response  string
		wantError string
	}{
		{
			name:     "success",
			response: `{"request_id":"test","data":{"alert_key":"example"}}`,
		},
		{
			name:      "provider error",
			response:  `{"error":{"code":"Unauthorized","message":"invalid integration key"}}`,
			wantError: "flashduty error: code = Unauthorized, message = invalid integration key",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
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

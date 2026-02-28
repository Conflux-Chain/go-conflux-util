package flashduty

import (
	"context"
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

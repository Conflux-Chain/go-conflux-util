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

// Please set the following enviroments before running tests:
// `TEST_FLASHDUTY_WEBHOOK`: FlashDuty webhook;
// `TEST_FLASHDUTY_SECRET`: FlashDuty secret.

func TestMain(m *testing.M) {
	webhook := os.Getenv("TEST_FLASHDUTY_WEBHOOK")
	secrect := os.Getenv("TEST_FLASHDUTY_SECRET")

	if len(webhook) > 0 && len(secrect) > 0 {
		robot = NewRobot(webhook, secrect)
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

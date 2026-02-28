package dingtalk

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

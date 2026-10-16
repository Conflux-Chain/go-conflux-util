package alert

import (
	"errors"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/alert/flashduty"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlashDutySeverity(t *testing.T) {
	ch := &FlashDutyChannel{}
	for _, tc := range []struct {
		severity Severity
		want     string
	}{
		{SeverityLow, flashduty.MsgLevelInfo},
		{SeverityMedium, flashduty.MsgLevelWarning},
		{SeverityHigh, flashduty.MsgLevelCritical},
		{SeverityCritical, flashduty.MsgLevelCritical},
	} {
		t.Run(tc.severity.String(), func(t *testing.T) {
			assert.Equal(t, tc.want, ch.adaptSeverity(tc.severity))
		})
	}
}

func TestFlashDutyContent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content interface{}
		want    string
	}{
		{"error", errors.New("database unavailable"), "database unavailable"},
		{"string", "database unavailable", "database unavailable"},
		{"map", map[string]int{"attempt": 3}, `{"attempt":3}`},
		{"struct", struct{ Count int }{3}, `{"Count":3}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := formatDefault(&Notification{Content: tc.content})
			require.NoError(t, err)
			assert.Equal(t, tc.want, data["content"])
		})
	}
	entry := logrus.NewEntry(logrus.New())
	entry.Data = logrus.Fields{"cause": errors.New("timeout"), logrus.ErrorKey: errors.New("failed")}
	data, err := formatLogrusEntry(&Notification{Content: entry})
	require.NoError(t, err)
	assert.Equal(t, "timeout", data["cause"])
	assert.Equal(t, "failed", data["error"])
}

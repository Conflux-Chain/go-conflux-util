package alert

import (
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/alert/flashduty"
	"github.com/stretchr/testify/assert"
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

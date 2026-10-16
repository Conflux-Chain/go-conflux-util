package alert

import (
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/assert"
)

func TestPagerDutyPayloadDefaultsDoNotMutateInput(t *testing.T) {
	input := &pagerduty.V2Payload{}
	note := &Notification{Title: "database failed", Severity: SeverityHigh, Content: input}
	for _, source := range []string{"source-a", "source-b"} {
		ch := NewPagerDutyChannel(source, []string{source}, PagerDutyConfig{Source: source})
		payload := ch.assemblePayloadDefault(note)
		assert.Equal(t, source, payload.Source)
		assert.Equal(t, source, payload.Group)
		assert.Equal(t, "database failed", payload.Summary)
		assert.Equal(t, "error", payload.Severity)
		assert.Equal(t, &pagerduty.V2Payload{}, input)
	}
}

func TestPagerDutyPayloadConcurrentDefaults(t *testing.T) {
	input := &pagerduty.V2Payload{}
	note := &Notification{Title: "alert", Content: input}
	for _, source := range []string{"source-a", "source-b"} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			ch := NewPagerDutyChannel(source, nil, PagerDutyConfig{Source: source})
			for i := 0; i < 100; i++ {
				assert.Equal(t, source, ch.assemblePayloadDefault(note).Source)
			}
			assert.Equal(t, &pagerduty.V2Payload{}, input)
		})
	}
}

func TestPagerDutyPayloadPreservesExplicitFields(t *testing.T) {
	input := &pagerduty.V2Payload{
		Summary: "explicit summary", Source: "explicit source", Severity: "critical",
		Group: "explicit group", Component: "database", Class: "storage", Timestamp: "2026-10-16T00:00:00Z",
		Details: map[string]interface{}{"attempt": 3},
	}
	ch := NewPagerDutyChannel("test", []string{"default"}, PagerDutyConfig{Source: "default"})
	payload := ch.assemblePayloadDefault(&Notification{Title: "default", Content: input})
	assert.Equal(t, input, payload)
	assert.NotSame(t, input, payload)
}

package alert

import (
	"context"
	"fmt"
	"strings"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/sirupsen/logrus"
)

var (
	_ Channel = (*PagerDutyChannel)(nil)
)

type PagerDutyConfig struct {
	// The auth token for the PagerDuty service.
	AuthToken string
	// The routinng key for the service integration.
	RoutingKey string
	// The unique location of the affected system, preferably a hostname or FQDN.
	Source string
}

// PagerDutyChannel represents a PagerDuty notification channel.
type PagerDutyChannel struct {
	ID     string          // the identifier of the channel
	Config PagerDutyConfig // the configuration for the PagerDuty channel

	tags   []string
	client *pagerduty.Client
}

// NewPagerDutyChannel creates a new PagerDuty channel with the given ID and configuration
func NewPagerDutyChannel(chID string, tags []string, conf PagerDutyConfig) *PagerDutyChannel {
	return &PagerDutyChannel{
		ID: chID, Config: conf, tags: tags,
		client: pagerduty.NewClient(conf.AuthToken),
	}
}

// Name returns the ID of the channel
func (c *PagerDutyChannel) Name() string {
	return c.ID
}

// Type returns the type of the channel, which is PagerDuty
func (c *PagerDutyChannel) Type() ChannelType {
	return ChannelTypePagerDuty
}

// Send sends notification using the PagerDuty channel.
func (c *PagerDutyChannel) Send(ctx context.Context, note *Notification) error {
	var payload *pagerduty.V2Payload
	switch note.Content.(type) {
	case *logrus.Entry:
		payload = c.assemblePayloadFromLogEntry(note)
	case *pagerduty.V2Payload:
		payload = c.assemblePayloadDefault(note)
	default:
		return ErrInvalidNotification
	}

	return c.SendRaw(ctx, payload)
}

// SendRaw sends raw payload using the PagerDuty channel.
func (c *PagerDutyChannel) SendRaw(ctx context.Context, content interface{}) error {
	payload, ok := content.(*pagerduty.V2Payload)
	if !ok {
		return ErrInvalidContentType
	}

	// Validate the payload before sending.
	if len(payload.Source) == 0 || len(payload.Summary) == 0 || len(payload.Severity) == 0 {
		return ErrInvalidNotification
	}

	// Refer to PD-CEF (https://support.pagerduty.com/docs/pd-cef) for more info.
	event := &pagerduty.V2Event{
		RoutingKey: c.Config.RoutingKey,
		Action:     "trigger",
		Payload:    payload,
	}

	_, err := c.client.ManageEventWithContext(ctx, event)
	return err
}

func (c *PagerDutyChannel) assemblePayloadDefault(note *Notification) *pagerduty.V2Payload {
	payload := note.Content.(*pagerduty.V2Payload)

	// Use the title of the notification as the summary by default.
	if len(payload.Summary) == 0 {
		payload.Summary = note.Title
	}

	// Use the source of the notification as the source by default.
	if len(payload.Source) == 0 {
		payload.Source = c.Config.Source
	}

	// Adapts to PagerDuty severity levels (info, warning, error, critical) by default.
	if len(payload.Severity) == 0 {
		payload.Severity = c.adaptSeverity(note.Severity)
	}

	// Use the tags of the channel by default.
	if len(payload.Group) == 0 {
		payload.Group = strings.Join(c.tags, ",")
	}

	return payload
}

func (c *PagerDutyChannel) assemblePayloadFromLogEntry(note *Notification) *pagerduty.V2Payload {
	entry := note.Content.(*logrus.Entry)

	ctxFields := make(map[string]interface{})
	for k, v := range entry.Data {
		if k == logrus.ErrorKey {
			continue
		}
		ctxFields[k] = v
	}

	payload := &pagerduty.V2Payload{
		Summary:  entry.Message,
		Source:   c.Config.Source,
		Severity: c.adaptSeverity(note.Severity),
		Group:    strings.Join(c.tags, ","),
		Details:  ctxFields,
	}

	entryError, ok := entry.Data[logrus.ErrorKey].(error)
	if ok && entryError != nil {
		payload.Summary = fmt.Sprintf("%s: %v", entry.Message, entryError)
	}

	return payload
}

// adaptSeverity adapts notification severity level to PagerDuty severity level.
func (c *PagerDutyChannel) adaptSeverity(severity Severity) string {
	switch severity {
	case SeverityLow:
		return "info"
	case SeverityMedium:
		return "warning"
	case SeverityHigh:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return ""
	}
}

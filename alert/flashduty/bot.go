package flashduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// Robot represents a flashduty custom robot that can send messages to groups.
type Robot struct {
	pushUri string
}

// NewRobot creates a robot. A non-empty secret overrides the webhook URL's
// integration_key parameter.
func NewRobot(pushUri, secret string) *Robot {
	if len(secret) != 0 {
		// Leave invalid URLs unchanged so Send reports the error.
		if webhook, err := url.Parse(pushUri); err == nil {
			query := webhook.Query()
			query.Set("integration_key", secret)
			webhook.RawQuery = query.Encode()
			pushUri = webhook.String()
		}
	}
	return &Robot{pushUri: pushUri}
}

// Send sends a flashduty message.
func (r Robot) Send(ctx context.Context, title, level, alertKey, description string, data map[string]string) error {
	return r.send(ctx, &message{
		Title:       title,
		Status:      level,
		AlertKey:    alertKey,
		Description: description,
		Data:        data,
	})
}

type responseError struct {
	Code string `json:"code"`
	Msg  string `json:"message"`
}

func (e *responseError) String() string {
	return fmt.Sprintf("flashduty error: code = %s, message = %s", e.Code, e.Msg)
}

func (e *responseError) Error() string {
	return e.String()
}

type repsonseData struct {
	AlertKey string `json:"alert_key"`
}

type response struct {
	RequestId string        `json:"request_id"`
	Data      repsonseData  `json:"data"`
	Error     responseError `json:"error"`
}

func (r Robot) send(ctx context.Context, msg interface{}) error {
	jm, err := json.Marshal(msg)
	if err != nil {
		return errors.WithMessage(err, "failed to marshal message")
	}

	req, errRequest := http.NewRequestWithContext(ctx, http.MethodPost, r.pushUri, bytes.NewReader(jm))
	if errRequest != nil {
		return errors.WithMessage(errRequest, "failed to create request")
	}

	req.Header.Add("Content-Type", "application/json")
	resp, errDo := http.DefaultClient.Do(req)
	if errDo != nil {
		return errors.WithMessage(errDo, "failed to do http request")
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("flashduty send failed: HTTP %s", resp.Status)
	}

	body, errReadBody := io.ReadAll(resp.Body)
	if errReadBody != nil {
		return errors.WithMessage(errReadBody, "failed to read http response body")
	}

	var fdr response
	err = json.Unmarshal(body, &fdr)
	if err != nil {
		return errors.WithMessagef(err, "failed to unmarshal response: %s", string(body))
	}
	if len(fdr.Error.Code) > 0 {
		return errors.WithMessage(&fdr.Error, "flashDutyRobot send failed")
	}

	return nil
}

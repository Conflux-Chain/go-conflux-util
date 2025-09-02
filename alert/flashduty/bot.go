package flashduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func ErrMsgTypeNotSupported(msgType string) error {
	return errors.Errorf("message type %s not supported", msgType)
}

// Robot represents a flashduty custom robot that can send messages to groups.
type Robot struct {
	pushUri string
	secret  string
}

func NewRobot(pushUri, secret string) *Robot {
	return &Robot{pushUri: pushUri, secret: secret}
}

// Send send a flashduty message.
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

	webURL := r.pushUri
	if len(r.secret) != 0 {
		webURL += fmt.Sprintf("?integration_key=%s", r.secret)
	}

	req, errRequest := http.NewRequestWithContext(ctx, http.MethodPost, webURL, bytes.NewReader(jm))
	if errRequest != nil {
		return errors.WithMessage(errRequest, "failed to create request")
	}

	req.Header.Add("Content-Type", "application/json")
	resp, errDo := http.DefaultClient.Do(req)
	if errDo != nil {
		return errors.WithMessage(errDo, "failed to do http request")
	}
	defer resp.Body.Close()

	body, errReadBody := io.ReadAll(resp.Body)
	if errReadBody != nil {
		return errors.WithMessage(errReadBody, "failed to read http response body")
	}

	var fdr response
	err = json.Unmarshal(body, &fdr)
	if err != nil {
		return err
	}
	if len(fdr.Error.Code) > 0 {
		return fmt.Errorf("flashDutyRobot send failed: %v", fdr.Error.Error())
	}

	return nil
}

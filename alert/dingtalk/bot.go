package dingtalk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

const maxResponseBodySize = 1 << 20 // 1 MiB

func ErrMsgTypeNotSupported(msgType string) error {
	return errors.Errorf("message type %s not supported", msgType)
}

// Robot represents a dingtalk custom robot that can send messages to groups.
type Robot struct {
	webHook string
	secret  string
}

func NewRobot(webhook, secret string) *Robot {
	return &Robot{webHook: webhook, secret: secret}
}

// Send sends a dingtalk message.
// It uses ctx for cancellation and does not add a timeout. Callers should provide
// a context with a deadline to bound the request duration.
// Redirects are rejected; configure the final webhook URL.
func (r Robot) Send(ctx context.Context, msgType, title, text string, atMobiles []string, isAtAll bool) error {
	switch msgType {
	case MsgTypeText:
		return r.SendText(ctx, text, atMobiles, isAtAll)
	case MsgTypeMarkdown:
		return r.SendMarkdown(ctx, title, text, atMobiles, isAtAll)
	default:
		return ErrMsgTypeNotSupported(msgType)
	}
}

// SendText send a text type message.
// As with Send, callers should provide a context with a deadline.
func (r Robot) SendText(ctx context.Context, content string, atMobiles []string, isAtAll bool) error {
	return r.send(ctx, &textMessage{
		MsgType: MsgTypeText,
		Text: textParams{
			Content: content,
		},
		At: atParams{
			AtMobiles: atMobiles,
			IsAtAll:   isAtAll,
		},
	})
}

// SendMarkdown send a markdown type message.
// As with Send, callers should provide a context with a deadline.
func (r Robot) SendMarkdown(ctx context.Context, title, text string, atMobiles []string, isAtAll bool) error {
	return r.send(ctx, &markdownMessage{
		MsgType: MsgTypeMarkdown,
		Markdown: markdownParams{
			Title: title,
			Text:  text,
		},
		At: atParams{
			AtMobiles: atMobiles,
			IsAtAll:   isAtAll,
		},
	})
}

type dingResponse struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

func (r Robot) send(ctx context.Context, msg interface{}) error {
	jm, err := json.Marshal(msg)
	if err != nil {
		return errors.WithMessage(err, "failed to marshal message")
	}

	webURL, err := genSignedURL(r.webHook, r.secret)
	if err != nil {
		return errors.WithMessage(err, "failed to generate signed URL")
	}

	req, errRequest := http.NewRequestWithContext(ctx, http.MethodPost, webURL, bytes.NewReader(jm))
	if errRequest != nil {
		return errors.WithMessage(errRequest, "failed to create request")
	}

	req.Header.Add("Content-Type", "application/json")
	client := *http.DefaultClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, errDo := client.Do(req)
	if errDo != nil {
		return errors.WithMessage(errDo, "failed to do http request")
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("dingtalk send failed: HTTP %s", resp.Status)
	}

	body, errReadBody := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize+1))
	if errReadBody != nil {
		return errors.WithMessage(errReadBody, "failed to read http response body")
	}
	if len(body) > maxResponseBodySize {
		return fmt.Errorf("dingtalk response body exceeds %d bytes", maxResponseBodySize)
	}

	var dr dingResponse
	err = json.Unmarshal(body, &dr)
	if err != nil {
		return err
	}
	if dr.Errcode != 0 {
		return fmt.Errorf("dingrobot send failed: %v", dr.Errmsg)
	}

	return nil
}

func genSignedURL(webhook, secret string) (string, error) {
	if len(secret) == 0 {
		return webhook, nil
	}

	webURL, err := url.Parse(webhook)
	if err != nil {
		return "", err
	}

	timeStr := fmt.Sprintf("%d", time.Now().UnixNano()/1e6)
	sign := fmt.Sprintf("%s\n%s", timeStr, secret)
	query := webURL.Query()
	query.Set("timestamp", timeStr)
	query.Set("sign", computeHmacSha256(sign, secret))
	webURL.RawQuery = query.Encode()
	return webURL.String(), nil
}

func computeHmacSha256(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

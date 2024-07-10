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

func ErrMsgTypeNotSupported(msgType string) error {
	return errors.Errorf("message type %s not supported", msgType)
}

// Robot represents a dingtalk custom robot that can send messages to groups.
type Robot struct {
	webHook string
	secret  string
}

func NewRobot(webhook, secrect string) *Robot {
	return &Robot{webHook: webhook, secret: secrect}
}

// Send send a dingtalk message.
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

	webURL := r.webHook
	if len(r.secret) != 0 {
		webURL += genSignedURL(r.secret)
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

func genSignedURL(secret string) string {
	timeStr := fmt.Sprintf("%d", time.Now().UnixNano()/1e6)
	sign := fmt.Sprintf("%s\n%s", timeStr, secret)
	signData := computeHmacSha256(sign, secret)
	encodeURL := url.QueryEscape(signData)
	return fmt.Sprintf("&timestamp=%s&sign=%s", timeStr, encodeURL)
}

func computeHmacSha256(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

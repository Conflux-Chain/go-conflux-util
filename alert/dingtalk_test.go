package alert

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDingTalkMessageTypeCase(t *testing.T) {
	for _, msgType := range []string{"text", "TEXT", "tExT", "markdown", "Markdown", "MARKDOWN"} {
		t.Run(msgType, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var message map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&message); !assert.NoError(t, err) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				assert.Equal(t, strings.ToLower(msgType), message["msgtype"])
				assert.Contains(t, message, strings.ToLower(msgType))
				io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
			}))
			defer server.Close()
			ch, err := parseAlertChannel("test", map[string]interface{}{
				"platform": "dingtalk", "msgtype": msgType, "webhook": server.URL,
			}, nil)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			require.NoError(t, ch.Send(ctx, &Notification{Title: "alert", Content: "test"}))
		})
	}
}

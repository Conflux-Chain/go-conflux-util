package alert

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/go-telegram/bot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type telegramTestClient func(*http.Request) (*http.Response, error)

func (f telegramTestClient) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestTelegramLongMessageSend(t *testing.T) {
	formatter, err := NewTelegramMarkdownFormatter(nil, nil)
	require.NoError(t, err)
	ch, err := NewTelegramChannel("test", formatter, TelegramConfig{ApiToken: "test-token", ChatId: "123"})
	require.NoError(t, err)
	requests := 0
	ch.Bot, err = bot.New("test-token", bot.WithSkipGetMe(), bot.WithHTTPClient(time.Second, telegramTestClient(func(r *http.Request) (*http.Response, error) {
		requests++
		require.NoError(t, r.ParseMultipartForm(1<<20))
		defer r.MultipartForm.RemoveAll()
		assert.Equal(t, "123", r.FormValue("chat_id"))
		assert.Equal(t, "MarkdownV2", r.FormValue("parse_mode"))
		text := r.FormValue("text")
		plain := telegramVisibleText(text)
		assert.LessOrEqual(t, len(utf16.Encode([]rune(plain))), 4096)
		assert.True(t, strings.HasSuffix(plain, "..."))
		assert.Equal(t, escapeMarkdown(plain), text)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":1}}`))}, nil
	})))
	require.NoError(t, err)
	require.NoError(t, ch.Send(context.Background(), &Notification{Content: strings.Repeat("🚨", 3000)}))
	assert.Equal(t, 1, requests)
}

package alert

import (
	"errors"
	"io"
	"net/mail"
	"regexp"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramMarkdownFormatterContent(t *testing.T) {
	formatter, err := NewTelegramMarkdownFormatter(nil, nil)
	require.NoError(t, err)
	for _, tc := range []struct {
		name    string
		content interface{}
		want    string
	}{
		{"string", "connection_failed", `connection\_failed`},
		{"int", 42, "42"},
		{"error", errors.New("connection_failed"), `connection\_failed`},
		{"map", map[string]int{"attempts": 3}, `map\[attempts:3\]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := formatter.Format(&Notification{Title: "alert", Content: tc.content})
			require.NoError(t, err)
			assert.Contains(t, msg, "*"+tc.want+"*")
		})
	}
}

func TestTelegramMarkdownBackslashes(t *testing.T) {
	formatter, err := NewTelegramMarkdownFormatter(nil, nil)
	require.NoError(t, err)
	for _, tc := range []struct{ input, want string }{
		{`path\`, `path\\`},
		{`C:\logs\error`, `C:\\logs\\error`},
		{`\*`, `\\\*`},
		{`_*[]()~` + "`" + `>#+-=|{}.!`, `\_\*\[\]\(\)\~` + "\\`" + `\>\#\+\-\=\|\{\}\.\!`},
	} {
		t.Run(tc.input, func(t *testing.T) {
			msg, err := formatter.Format(&Notification{Title: "alert", Content: tc.input})
			require.NoError(t, err)
			assert.Contains(t, msg, "*"+tc.want+"*\n")
		})
	}
}

func TestSmtpHtmlFormatterSubject(t *testing.T) {
	formatter, err := NewSmtpHtmlFormatter(SmtpConfig{
		From: "sender@example.com", To: []string{"recipient@example.com"},
	}, nil)
	require.NoError(t, err)

	for _, tc := range []struct {
		name, title string
		wantError   bool
	}{
		{"plain", "database alert", false},
		{"unicode", "数据库告警", false},
		{"empty", "", false},
		{"CR", "database\ralert", true},
		{"LF", "database\nalert", true},
		{"injected header", "database alert\r\nX-Injected: unexpected-header", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := formatter.Format(&Notification{Title: tc.title, Content: "connection failed"})
			if tc.wantError {
				assert.ErrorIs(t, err, ErrInvalidNotification)
				assert.ErrorContains(t, err, "title must not contain CR or LF")
				assert.Empty(t, msg)
				return
			}
			require.NoError(t, err)
			parsed, err := mail.ReadMessage(strings.NewReader(msg))
			require.NoError(t, err)
			assert.Equal(t, tc.title, parsed.Header.Get("Subject"))
			assert.Empty(t, parsed.Header.Get("X-Injected"))
		})
	}
}

func TestTelegramMarkdownMentions(t *testing.T) {
	formatter, err := NewTelegramMarkdownFormatter(nil, []string{"ops_team", "oncall"})
	require.NoError(t, err)
	for _, content := range []interface{}{"test", logrus.NewEntry(logrus.New())} {
		msg, err := formatter.Format(&Notification{Content: content})
		require.NoError(t, err)
		assert.Contains(t, msg, `@ops\_team @oncall`)
	}
}

func TestTelegramMessageLength(t *testing.T) {
	formatter, err := NewTelegramMarkdownFormatter(nil, nil)
	require.NoError(t, err)
	base := &Notification{Title: "alert", Content: ""}
	msg, err := formatter.Format(base)
	require.NoError(t, err)
	remaining := maxAlertMsgLength - len(utf16.Encode([]rune(telegramVisibleText(msg))))
	for _, size := range []int{remaining - 1, remaining, remaining + 1} {
		note := &Notification{Title: "alert", Content: strings.Repeat("a", size)}
		msg, err := formatter.Format(note)
		require.NoError(t, err)
		visible := telegramVisibleText(msg)
		assert.LessOrEqual(t, len(utf16.Encode([]rune(visible))), maxAlertMsgLength)
		if size <= remaining {
			assert.Contains(t, msg, "*alert*")
			assert.Contains(t, visible, note.Content)
		} else {
			assert.NotContains(t, msg, "*alert*")
			plain, err := formatter.textFormatter.Format(note)
			require.NoError(t, err)
			if len(utf16.Encode([]rune(plain))) > maxAlertMsgLength {
				assert.True(t, strings.HasSuffix(visible, "..."))
			} else {
				assert.Equal(t, plain, visible)
			}
		}
	}
	// Escapes consume wire bytes, but not additional visible characters.
	for _, content := range []string{strings.Repeat("中", 1400), strings.Repeat("_*\\", 1000), strings.Repeat("🚨", 1000)} {
		msg, err := formatter.Format(&Notification{Title: "alert", Content: content})
		require.NoError(t, err)
		assert.Contains(t, msg, "*alert*")
		assert.Contains(t, telegramVisibleText(msg), content)
		assert.True(t, utf8.ValidString(msg))
	}
}

func TestTelegramLongNotifications(t *testing.T) {
	formatter, err := NewTelegramMarkdownFormatter([]string{"production"}, []string{"ops_team"})
	require.NoError(t, err)
	entry := logrus.NewEntry(logrus.New())
	entry.Message = "database failed"
	entry.Data = logrus.Fields{"error": errors.New(strings.Repeat("reason", 1000))}
	fieldsEntry := logrus.NewEntry(logrus.New())
	fieldsEntry.Data = logrus.Fields{"a": strings.Repeat("x", 2500), "b": strings.Repeat("y", 2500)}
	for _, note := range []*Notification{
		{Title: strings.Repeat("title", 1000), Content: "test"},
		{Content: strings.Repeat("中", 5000)},
		{Content: strings.Repeat("🚨", 3000)},
		{Content: strings.Repeat("\\_*", 2000)},
		{Content: entry},
		{Content: fieldsEntry},
	} {
		msg, err := formatter.Format(note)
		require.NoError(t, err)
		plain := telegramVisibleText(msg)
		assert.True(t, utf8.ValidString(msg))
		assert.LessOrEqual(t, len(utf16.Encode([]rune(plain))), maxAlertMsgLength)
		assert.True(t, strings.HasSuffix(plain, "..."))
		// Fallback text contains no active formatting characters.
		assert.Equal(t, escapeMarkdown(plain), msg)
		full, err := formatter.textFormatter.Format(note)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(full, strings.TrimSuffix(plain, "...")))
	}
}

// Decode the limited Markdown produced by our templates independently of the
// production length counter: escaped characters are literal, bold markers aren't.
func telegramVisibleText(msg string) string {
	return regexp.MustCompile(`(?s)\\.|\*`).ReplaceAllStringFunc(msg, func(token string) string {
		if token == "*" {
			return ""
		}
		return token[1:]
	})
}

func TestSmtpHtmlEscapesDynamicText(t *testing.T) {
	const input = `<b>unsafe & text</b>`
	formatter, err := NewSmtpHtmlFormatter(SmtpConfig{}, []string{input})
	require.NoError(t, err)
	entry := logrus.NewEntry(logrus.New())
	entry.Message = input
	entry.Data = logrus.Fields{logrus.ErrorKey: errors.New(input), input: input}
	for _, content := range []interface{}{input, entry} {
		msg, err := formatter.Format(&Notification{Title: input, Content: content})
		require.NoError(t, err)
		parsed, err := mail.ReadMessage(strings.NewReader(msg))
		require.NoError(t, err)
		// Header handling is separate from HTML escaping.
		assert.Equal(t, input, parsed.Header.Get("Subject"))
		body, err := io.ReadAll(parsed.Body)
		require.NoError(t, err)
		assert.NotContains(t, string(body), input)
		assert.Contains(t, string(body), "&lt;b&gt;unsafe &amp; text&lt;/b&gt;")
		assert.Contains(t, string(body), "<html>")
	}
}

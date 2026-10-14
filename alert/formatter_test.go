package alert

import (
	"net/mail"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

package alert

import (
	"fmt"
	"strings"
	"time"
)

// Formatter defines how messages are formatted.
type Formatter interface {
	Format(note *Notification) (string, error)
}

type SimpleTextFormatter struct {
	tags []string
}

func NewSimpleTextFormatter(tags []string) *SimpleTextFormatter {
	return &SimpleTextFormatter{tags: tags}
}

func (f *SimpleTextFormatter) Format(note *Notification) (string, error) {
	tagStr := strings.Join(f.tags, "/")
	nowStr := time.Now().Format("2006-01-02T15:04:05-0700")
	str := fmt.Sprintf(
		"%v\nseverity:\t%s;\ntags:\t%v;\n%v;\ntime:\t%v",
		note.Title, note.Severity, tagStr, note.Content, nowStr,
	)
	return str, nil
}

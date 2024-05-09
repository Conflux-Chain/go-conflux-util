package alert

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/go-telegram/bot"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Formatter defines how messages are formatted.
type Formatter interface {
	Format(note *Notification) (string, error)
}

type markdownFormatter struct {
	tags []string

	defaultTpl  *template.Template
	logEntryTpl *template.Template
}

func newMarkdownFormatter(
	tags []string, funcMap template.FuncMap, defaultStrTpl, logEntryStrTpl string,
) (f *markdownFormatter, err error) {
	var tpls [2]*template.Template

	strTemplates := [2]string{defaultStrTpl, logEntryStrTpl}
	for i := range strTemplates {
		tpls[i], err = template.New("markdown").Funcs(funcMap).Parse(strTemplates[i])
		if err != nil {
			return nil, errors.WithMessage(err, "bad template")
		}
	}

	return &markdownFormatter{
		tags:        tags,
		defaultTpl:  tpls[0],
		logEntryTpl: tpls[1],
	}, nil
}

func (f *markdownFormatter) Format(note *Notification) (string, error) {
	if _, ok := note.Content.(*logrus.Entry); ok {
		return f.formatLogrusEntry(note)
	}

	return f.formatDefault(note)
}

func (f *markdownFormatter) formatLogrusEntry(note *Notification) (string, error) {
	entry := note.Content.(*logrus.Entry)
	entryError, _ := entry.Data[logrus.ErrorKey].(error)

	ctxFields := make(map[string]interface{})
	for k, v := range entry.Data {
		if k == logrus.ErrorKey {
			continue
		}
		ctxFields[k] = v
	}

	buffer := bytes.Buffer{}
	err := f.logEntryTpl.Execute(&buffer, struct {
		Level     logrus.Level
		Tags      []string
		Time      time.Time
		Msg       string
		Error     error
		CtxFields map[string]interface{}
	}{entry.Level, f.tags, entry.Time, entry.Message, entryError, ctxFields})
	if err != nil {
		return "", errors.WithMessage(err, "template exec error")
	}

	return buffer.String(), nil
}

func (f *markdownFormatter) formatDefault(note *Notification) (string, error) {
	buffer := bytes.Buffer{}
	err := f.defaultTpl.Execute(&buffer, struct {
		Title    string
		Tags     []string
		Severity Severity
		Time     time.Time
		Content  interface{}
	}{note.Title, f.tags, note.Severity, time.Now(), note.Content})
	if err != nil {
		return "", errors.WithMessage(err, "template exec error")
	}

	return buffer.String(), nil
}

type DingTalkMarkdownFormatter struct {
	*markdownFormatter
}

func NewDingtalkMarkdownFormatter(tags []string) (*DingTalkMarkdownFormatter, error) {
	funcMap := template.FuncMap{"formatRFC3339": formatRFC3339}
	mf, err := newMarkdownFormatter(
		tags, funcMap, dingTalkMarkdownTemplates[0], dingTalkMarkdownTemplates[1],
	)
	if err != nil {
		return nil, err
	}

	return &DingTalkMarkdownFormatter{markdownFormatter: mf}, nil
}

func formatRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

type TelegramMarkdownFormatter struct {
	*markdownFormatter
}

func NewTelegramMarkdownFormatter(tags []string) (f *TelegramMarkdownFormatter, err error) {
	funcMap := template.FuncMap{
		"escapeMarkdown": escapeMarkdown,
		"formatRFC3339":  formatRFC3339,
	}
	mf, err := newMarkdownFormatter(
		tags, funcMap, telegramMarkdownTemplates[0], telegramMarkdownTemplates[1],
	)
	if err != nil {
		return nil, err
	}

	return &TelegramMarkdownFormatter{markdownFormatter: mf}, nil
}

func escapeMarkdown(v interface{}) string {
	return bot.EscapeMarkdown(fmt.Sprintf("%v", v))
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

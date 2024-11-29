package alert

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/Conflux-Chain/go-conflux-util/alert/dingtalk"
	"github.com/go-telegram/bot"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	maxAlertMsgLength = 4096 // max length for the alert message
)

// Formatter defines how messages are formatted.
type Formatter interface {
	Format(note *Notification) (string, error)
}

type tplFormatter struct {
	tags []string

	defaultTpl  *template.Template
	logEntryTpl *template.Template
}

func newTplFormatter(
	tags []string, defaultTpl, logEntryTpl *template.Template) *tplFormatter {
	return &tplFormatter{
		tags: tags, defaultTpl: defaultTpl, logEntryTpl: logEntryTpl,
	}
}

func (f *tplFormatter) Format(note *Notification) (string, error) {
	if _, ok := note.Content.(*logrus.Entry); ok {
		return f.formatLogrusEntry(note)
	}

	return f.formatDefault(note)
}

func (f *tplFormatter) formatLogrusEntry(note *Notification) (string, error) {
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

func (f *tplFormatter) formatDefault(note *Notification) (string, error) {
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

type markdownFormatter struct {
	*tplFormatter
}

func newMarkdownFormatter(
	tags []string, funcMap template.FuncMap, defaultStrTpl, logEntryStrTpl string,
) (f *markdownFormatter, err error) {
	var tpls [2]*template.Template

	strTemplates := [2]string{defaultStrTpl, logEntryStrTpl}
	for i := range strTemplates {
		tpls[i], err = template.New("markdown").Funcs(funcMap).Parse(strTemplates[i])
		if err != nil {
			return nil, errors.WithMessage(err, "bad markdown template")
		}
	}

	return &markdownFormatter{
		tplFormatter: newTplFormatter(tags, tpls[0], tpls[1]),
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

func NewDingtalkMarkdownFormatter(tags, mentions []string) (*DingTalkMarkdownFormatter, error) {
	funcMap := template.FuncMap{
		"formatRFC3339": formatRFC3339,
		"mentions":      func() []string { return mentions },
	}
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

func NewTelegramMarkdownFormatter(tags, atUsers []string) (f *TelegramMarkdownFormatter, err error) {
	funcMap := template.FuncMap{
		"escapeMarkdown":         escapeMarkdown,
		"formatRFC3339":          formatRFC3339,
		"truncateStringWithTail": truncateStringWithTail,
		"mentions":               func() []string { return atUsers },
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

// truncateStringWithTail is used to immediately truncate the input string to the max length limit.
// A tail "..." is then added to the end of the string, if the string was longer than max length.
func truncateStringWithTail(s string) string {
	if len(s) > maxAlertMsgLength {
		// Ttrim the string and add "..."
		return s[:maxAlertMsgLength] + "..."
	}

	return s
}

type htmlFormatter struct {
	*tplFormatter
}

func newHtmlFormatter(
	tags []string, funcMap template.FuncMap, defaultStrTpl, logEntryStrTpl string,
) (f *htmlFormatter, err error) {
	var tpls [2]*template.Template

	strTemplates := [2]string{defaultStrTpl, logEntryStrTpl}
	for i := range strTemplates {
		tpls[i], err = template.New("html").Funcs(funcMap).Parse(strTemplates[i])
		if err != nil {
			return nil, errors.WithMessage(err, "bad html template")
		}
	}

	return &htmlFormatter{
		tplFormatter: newTplFormatter(tags, tpls[0], tpls[1]),
	}, nil
}

type SmtpHtmlFormatter struct {
	*htmlFormatter
	conf SmtpConfig
}

func NewSmtpHtmlFormatter(
	conf SmtpConfig, tags []string) (f *SmtpHtmlFormatter, err error) {
	funcMap := template.FuncMap{
		"formatRFC3339": formatRFC3339,
	}
	hf, err := newHtmlFormatter(
		tags, funcMap, htmlTemplates[0], htmlTemplates[1],
	)
	if err != nil {
		return nil, err
	}

	return &SmtpHtmlFormatter{conf: conf, htmlFormatter: hf}, nil
}

func (f *SmtpHtmlFormatter) Format(note *Notification) (msg string, err error) {
	body, err := f.htmlFormatter.Format(note)
	if err != nil {
		return "", err
	}

	header := make(map[string]string)
	header["From"] = f.conf.From
	header["To"] = strings.Join(f.conf.To, ";")
	header["Subject"] = note.Title
	header["Content-Type"] = "text/html; charset=UTF-8"

	for k, v := range header {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	msg += "\r\n" + body
	return msg, nil
}

type SimpleTextFormatter struct {
	*tplFormatter
	tags []string
}

func NewSimpleTextFormatter(tags, mentions []string) (fmt *SimpleTextFormatter, err error) {
	strTemplates := [2]string{simpleTextTemplates[0], simpleTextTemplates[1]}
	funcMap := template.FuncMap{
		"formatRFC3339": formatRFC3339,
		"mentions":      func() []string { return mentions },
	}

	var tpls [2]*template.Template
	for i := range strTemplates {
		tpls[i], err = template.New("text").Funcs(funcMap).Parse(strTemplates[i])
		if err != nil {
			return nil, errors.WithMessage(err, "bad text template")
		}
	}

	fmt = &SimpleTextFormatter{
		tplFormatter: newTplFormatter(tags, tpls[0], tpls[1]),
		tags:         tags,
	}
	return fmt, nil
}

func (f *SimpleTextFormatter) Format(note *Notification) (string, error) {
	msg, err := f.tplFormatter.Format(note)
	if err != nil {
		return "", err
	}

	msg = strings.TrimFunc(msg, func(r rune) bool {
		return unicode.IsSpace(r)
	})

	return msg, nil
}

func newDingtalkMsgFormatter(msgType string, tags []string, mentions []string) (Formatter, error) {
	switch {
	case strings.EqualFold(msgType, dingtalk.MsgTypeText):
		return NewSimpleTextFormatter(tags, mentions)
	case strings.EqualFold(msgType, dingtalk.MsgTypeMarkdown):
		return NewDingtalkMarkdownFormatter(tags, mentions)
	default:
		return nil, dingtalk.ErrMsgTypeNotSupported(msgType)
	}
}

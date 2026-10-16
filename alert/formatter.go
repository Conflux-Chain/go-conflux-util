package alert

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"io"
	"strings"
	"text/template"
	"time"
	"unicode"
	"unicode/utf16"

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

type templateExecutor interface {
	Execute(io.Writer, interface{}) error
}

type tplFormatter struct {
	tags []string

	defaultTpl  templateExecutor
	logEntryTpl templateExecutor
}

func newTplFormatter(
	tags []string, defaultTpl, logEntryTpl templateExecutor) *tplFormatter {
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
		"upper":                  toUpper,
		"formatLogLevel":         formatLogLevel,
		"formatRFC3339":          formatRFC3339,
		"formatSeverity":         formatSeverity,
		"isHighPriorityLogLevel": isHighPriorityLogLevel,
		"isHighPrioritySeverity": isHighPrioritySeverity,
		"isWarningLogLevel":      isWarningLogLevel,
		"isWarningSeverity":      isWarningSeverity,
		"mentions":               func() []string { return mentions },
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

func formatLogLevel(level logrus.Level) string {
	return strings.ToUpper(level.String())
}

func formatSeverity(severity Severity) string {
	return strings.ToUpper(severity.String())
}

func toUpper(text string) string {
	return strings.ToUpper(text)
}

func isHighPriorityLogLevel(level logrus.Level) bool {
	return level <= logrus.ErrorLevel
}

func isHighPrioritySeverity(severity Severity) bool {
	return severity >= SeverityHigh
}

func isWarningLogLevel(level logrus.Level) bool {
	return level == logrus.WarnLevel
}

func isWarningSeverity(severity Severity) bool {
	return severity == SeverityMedium
}

type TelegramMarkdownFormatter struct {
	*markdownFormatter
	textFormatter *SimpleTextFormatter
}

func NewTelegramMarkdownFormatter(tags, atUsers []string) (f *TelegramMarkdownFormatter, err error) {
	funcMap := template.FuncMap{
		"formatLogLevel":         formatLogLevel,
		"toString":               toString,
		"upper":                  toUpper,
		"escapeMarkdown":         escapeMarkdown,
		"formatSeverity":         formatSeverity,
		"formatRFC3339":          formatRFC3339,
		"isHighPriorityLogLevel": isHighPriorityLogLevel,
		"isHighPrioritySeverity": isHighPrioritySeverity,
		"isWarningLogLevel":      isWarningLogLevel,
		"isWarningSeverity":      isWarningSeverity,
		"mentions":               func() []string { return atUsers },
	}
	mf, err := newMarkdownFormatter(
		tags, funcMap, telegramMarkdownTemplates[0], telegramMarkdownTemplates[1],
	)
	if err != nil {
		return nil, err
	}

	tf, err := NewSimpleTextFormatter(tags, atUsers)
	if err != nil {
		return nil, err
	}
	return &TelegramMarkdownFormatter{markdownFormatter: mf, textFormatter: tf}, nil
}

func (f *TelegramMarkdownFormatter) Format(note *Notification) (string, error) {
	msg, err := f.markdownFormatter.Format(note)
	if err != nil || telegramMarkdownLength(msg) <= maxAlertMsgLength {
		return msg, err
	}

	msg, err = f.textFormatter.Format(note)
	if err != nil {
		return "", err
	}
	return escapeMarkdown(truncateStringWithTail(msg)), nil
}

// telegramMarkdownLength counts visible UTF-16 units conservatively. The built-in
// templates use only bold markers and escaped text, not arbitrary Markdown.
func telegramMarkdownLength(s string) (length int) {
	escaped := false
	for _, r := range s {
		if !escaped {
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '*' {
				continue
			}
		}
		length += utf16.RuneLen(r)
		escaped = false
	}
	return length
}

func toString(val interface{}) string {
	if str, ok := val.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", val)
}

func escapeMarkdown(v interface{}) string {
	return bot.EscapeMarkdown(strings.ReplaceAll(toString(v), `\`, `\\`))
}

// truncateStringWithTail bounds plain text in UTF-16 units, including the tail,
// without splitting a Unicode character. Markdown escaping must happen after it.
func truncateStringWithTail(s string) string {
	length, cut := 0, len(s)
	for i, r := range s {
		length += utf16.RuneLen(r)
		if length > maxAlertMsgLength-len("...") && cut == len(s) {
			cut = i
		}
		if length > maxAlertMsgLength {
			return s[:cut] + "..."
		}
	}
	return s
}

type htmlFormatter struct {
	*tplFormatter
}

func newHtmlFormatter(
	tags []string, funcMap template.FuncMap, defaultStrTpl, logEntryStrTpl string,
) (f *htmlFormatter, err error) {
	var tpls [2]*htmltemplate.Template

	strTemplates := [2]string{defaultStrTpl, logEntryStrTpl}
	for i := range strTemplates {
		tpls[i], err = htmltemplate.New("html").Funcs(htmltemplate.FuncMap(funcMap)).Parse(strTemplates[i])
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
		"formatLogLevel":         formatLogLevel,
		"formatRFC3339":          formatRFC3339,
		"formatSeverity":         formatSeverity,
		"isHighPriorityLogLevel": isHighPriorityLogLevel,
		"isHighPrioritySeverity": isHighPrioritySeverity,
		"isWarningLogLevel":      isWarningLogLevel,
		"isWarningSeverity":      isWarningSeverity,
		"upper":                  toUpper,
	}
	hf, err := newHtmlFormatter(
		tags, funcMap, htmlTemplates[0], htmlTemplates[1],
	)
	if err != nil {
		return nil, err
	}

	return &SmtpHtmlFormatter{conf: conf, htmlFormatter: hf}, nil
}

// Format builds an HTML email and rejects titles containing CR or LF to prevent
// the title from introducing additional email headers.
func (f *SmtpHtmlFormatter) Format(note *Notification) (msg string, err error) {
	if strings.ContainsAny(note.Title, "\r\n") {
		return "", errors.WithMessage(ErrInvalidNotification, "SMTP title must not contain CR or LF")
	}

	body, err := f.htmlFormatter.Format(note)
	if err != nil {
		return "", err
	}

	header := make(map[string]string)
	header["From"] = f.conf.From
	header["To"] = strings.Join(f.conf.To, ", ")
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

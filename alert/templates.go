package alert

var (
	simpleTextTemplates = []string{
		`{{- /* default text template */ -}}
{{.Title}}

Tags: {{.Tags}}
Severity: {{.Severity}}
Time: {{.Time | formatRFC3339}}

{{.Content}}
{{ range mentions }}@{{ . }} {{ end }}
`,
		`{{- /* logrus entry text template */ -}}
{{.Level}}

Tags: {{.Tags}}
Time: {{.Time | formatRFC3339}}

Message
{{.Msg}}

Reason
{{with .Error}}{{.Error}}{{ else }}N/A{{ end }}

{{ if .CtxFields }}Context Fields{{ range $Key, $Val := .CtxFields }}
{{$Key}}: {{$Val}}{{ end }}{{ end }}
{{ range mentions }}@{{ . }} {{ end }}
`,
	}

	dingTalkMarkdownTemplates = []string{
		`{{- /* default markdown template */ -}}
{{- if isHighPrioritySeverity .Severity }}# 🚨<font color="#dd0000">{{.Title | upper}}</font>
{{ else if isWarningSeverity .Severity }}# ⚠️<font color="#d48806">{{.Title | upper}}</font>
{{ else }}# {{.Title}}
{{ end }}

- **Tags**: {{.Tags}}
- **Severity**: {{.Severity}}
- **Time**: {{.Time | formatRFC3339}}

**{{.Content}}**
{{ range mentions }}@{{ . }} {{ end }}
`,
		`{{- /* logrus entry markdown template */ -}}
{{- if isHighPriorityLogLevel .Level }}# 🚨<font color="#dd0000">{{.Level | formatLogLevel}}</font>
{{ else if isWarningLogLevel .Level }}# ⚠️<font color="#d48806">{{.Level | formatLogLevel}}</font>
{{ else }}# {{.Level}}
{{ end }}

- **Tags**: {{.Tags}}
- **Time**: {{.Time | formatRFC3339}}

---

## Message
{{.Msg}}

{{with .Error}}
---

## Reason
{{.Error}}
{{ end }}

{{ if .CtxFields }}
---

## Context Fields

{{ range $Key, $Val := .CtxFields }}
- **{{$Key}}**: {{$Val}}
{{ end }}
{{ end }}
{{ range mentions }}@{{ . }} {{ end }}
`,
	}

	telegramMarkdownTemplates = []string{
		`{{- /* default markdown template */ -}}
{{- if isHighPrioritySeverity .Severity }}* 🚨{{.Title | escapeMarkdown | upper }}*
{{ else if isWarningSeverity .Severity }}* ⚠️{{.Title | escapeMarkdown | upper }}*
{{ else }}*{{.Title | escapeMarkdown}}*
{{ end }}
*Tags*: {{.Tags | escapeMarkdown}}
*Severity*: {{.Severity | escapeMarkdown}}
*Time*: {{.Time | formatRFC3339 | escapeMarkdown}}
*{{.Content | truncateStringWithTail | escapeMarkdown}}*
{{ range mentions }}@{{ . }} {{ end }}
`,
		`{{- /* logrus entry markdown template */ -}}
{{- if isHighPriorityLogLevel .Level }}* 🚨{{.Level | formatLogLevel | escapeMarkdown | upper}}*
{{ else if isWarningLogLevel .Level }}* ⚠️{{.Level | formatLogLevel | escapeMarkdown | upper}}*
{{ else }}*{{.Level | escapeMarkdown}}*
{{ end }}
*Tags*: {{.Tags | escapeMarkdown}}
*Time*: {{.Time | formatRFC3339 | escapeMarkdown}}

*Message*
{{.Msg | truncateStringWithTail | escapeMarkdown}}

{{with .Error}}*Reason*
{{.Error | escapeMarkdown}}

{{else}}{{ end }}{{ if .CtxFields }}*Context Fields*:{{ range $Key, $Val := .CtxFields }}
    *{{$Key | escapeMarkdown}}*: {{$Val | toString | truncateStringWithTail | escapeMarkdown}}{{ end }}{{ end }}
{{ range mentions }}@{{ . }} {{ end }}
`,
	}

	htmlTemplates = []string{
		`{{- /* default HTML template */ -}}
<!doctype html>
<html>
{{- if isHighPrioritySeverity .Severity }}
<body style="background-color:#fff1f0;">
<h1 style="color:#dd0000;"> 🚨{{.Title | upper}}</h1>
{{- else if isWarningSeverity .Severity }}
<body style="background-color:#fff7e6;">
<h1 style="color:#d48806;"> ⚠️{{.Title | upper}}</h1>
{{- else }}
<body>
<h1>{{.Title}}</h1>
{{- end }}
<ul>
	<li><b>Tags</b>: {{.Tags}}</li>
	<li><b>Severity</b>: {{.Severity}}</li>
	<li><b>Time</b>: {{.Time | formatRFC3339}}</li>
</ul>
<p>{{.Content}}</p>
</body>
</html>
`,
		`{{- /* logrus entry HTML template */ -}}
<!doctype html>
<html>
{{- if isHighPriorityLogLevel .Level }}
<body style="background-color:#fff1f0;">
<h1 style="color:#dd0000;"> 🚨{{.Level | formatLogLevel}}</h1>
{{- else if isWarningLogLevel .Level }}
<body style="background-color:#fff7e6;">
<h1 style="color:#d48806;"> ⚠️{{.Level | formatLogLevel}}</h1>
{{- else }}
<body>
<h1>{{.Level}}</h1>
{{- end }}
<ul>
<li><b>Tags</b>: {{.Tags}}</li>
<li><b>Time</b>: {{.Time | formatRFC3339}}</li>
</ul>
<hr/>
<h2>Message</h2>
<p>{{.Msg}}</p>
{{with .Error}}
<hr/>
<h2>Reason</h2>
<p>{{.Error}}</p>
{{ end }}
{{ if .CtxFields }}
<hr/>
<h2>Context Fields</h2>
<ul>
{{ range $Key, $Val := .CtxFields }}
<li><b>{{$Key}}</b>: {{$Val}}</li>
{{ end }}
{{ end }}
</body>
</html>
`,
	}
)

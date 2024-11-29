package alert

var (
	simpleTextTemplates = []string{
		`{{- /* default text template */ -}}
{{.Title}}

Tags: {{.Tags}}
Severity: {{.Severity}}
Time: {{.Time | formatRFC3339}}

{{.Content}}
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
# {{.Title}}

- **Tags**: {{.Tags}}
- **Severity**: {{.Severity}}
- **Time**: {{.Time | formatRFC3339}}

**{{.Content}}**
`,
		`{{- /* logrus entry markdown template */ -}}
# {{.Level}}

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
*{{.Title | escapeMarkdown}}*
*Tags*: {{.Tags | escapeMarkdown}}
*Severity*: {{.Severity | escapeMarkdown}}
*Time*: {{.Time | formatRFC3339 | escapeMarkdown}}
*{{.Content | truncateStringWithTail | escapeMarkdown}}*
{{ range mentions }}@{{ . }} {{ end }}
`,
		`{{- /* logrus entry markdown template */ -}}
*{{.Level | escapeMarkdown}}*
*Tags*: {{.Tags | escapeMarkdown}}
*Time*: {{.Time | formatRFC3339 | escapeMarkdown}}

*Message*
{{.Msg | truncateStringWithTail | escapeMarkdown}}

{{with .Error}}*Reason*
{{.Error | escapeMarkdown}}

{{else}}{{ end }}{{ if .CtxFields }}*Context Fields*:{{ range $Key, $Val := .CtxFields }}
    *{{$Key | escapeMarkdown}}*: {{$Val | truncateStringWithTail | escapeMarkdown}}{{ end }}{{ end }}
{{ range mentions }}@{{ . }} {{ end }}
`,
	}

	htmlTemplates = []string{
		`{{- /* default HTML template */ -}}
<h1>{{.Title}}</h1>
<ul>
	<li><b>Tags</b>: {{.Tags}}</li>
	<li><b>Severity</b>: {{.Severity}}</li>
	<li><b>Time</b>: {{.Time | formatRFC3339}}</li>
</ul>
<p>{{.Content}}</p>
`,
		`{{- /* logrus entry HTML template */ -}}
<h1>{{.Level}}</h1>
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
`,
	}
)

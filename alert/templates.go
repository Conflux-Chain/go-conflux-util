package alert

var (
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
`,
	}

	telegramMarkdownTemplates = []string{
		`{{- /* default markdown template */ -}}
*{{.Title | escapeMarkdown}}*
*Tags*: {{.Tags | escapeMarkdown}}
*Severity*: {{.Severity | escapeMarkdown}}
*Time*: {{.Time | formatRFC3339 | escapeMarkdown}}
*{{.Content | escapeMarkdown}}*
`,
		`{{- /* logrus entry markdown template */ -}}
*{{.Level | escapeMarkdown}}*
*Tags*: {{.Tags | escapeMarkdown}}
*Time*: {{.Time | formatRFC3339 | escapeMarkdown}}

*Message*
{{.Msg | escapeMarkdown}}

{{with .Error}}*Reason*
{{.Error | escapeMarkdown}}{{ end }}

{{ if .CtxFields }}*Context Fields*:{{ range $Key, $Val := .CtxFields }}
    *{{$Key | escapeMarkdown}}*: {{$Val | escapeMarkdown}}{{ end }}{{ end }}
`,
	}
)

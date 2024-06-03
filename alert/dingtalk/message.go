package dingtalk

const (
	MsgTypeMarkdown = "markdown"
	MsgTypeText     = "text"
)

type TextMessage struct {
	MsgType string     `json:"msgtype"`
	Text    TextParams `json:"text"`
	At      AtParams   `json:"at"`
}

type TextParams struct {
	Content string `json:"content"`
}

type AtParams struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

type MarkdownMessage struct {
	MsgType  string         `json:"msgtype"`
	Markdown MarkdownParams `json:"markdown"`
	At       AtParams       `json:"at"`
}

type MarkdownParams struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

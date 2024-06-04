package dingtalk

const (
	msgTypeMarkdown = "markdown"
	msgTypeText     = "text"
)

type textMessage struct {
	MsgType string     `json:"msgtype"`
	Text    textParams `json:"text"`
	At      atParams   `json:"at"`
}

type textParams struct {
	Content string `json:"content"`
}

type atParams struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

type markdownMessage struct {
	MsgType  string         `json:"msgtype"`
	Markdown markdownParams `json:"markdown"`
	At       atParams       `json:"at"`
}

type markdownParams struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

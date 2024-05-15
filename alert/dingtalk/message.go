package dingtalk

const (
	msgTypeText       = "text"
	msgTypeLink       = "link"
	msgTypeMarkdown   = "markdown"
	msgTypeActionCard = "actionCard"
)

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

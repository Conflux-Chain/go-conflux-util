package flashduty

const (
	MsgLevelOkay     = "Ok"
	MsgLevelInfo     = "Info"
	MsgLevelWarning  = "Warning"
	MsgLevelCritical = "Critical"
)

type message struct {
	Title       string            `json:"title_rule"`
	Status      string            `json:"event_status"`
	AlertKey    string            `json:"alert_key"`
	Description string            `json:"description"`
	Data        map[string]string `json:"labels"`
}

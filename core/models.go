package core

type ActionModel struct {
	Action  HandlerName `json:"action"`
	Payload string      `json:"payload"`
}

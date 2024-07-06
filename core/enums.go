package core

type HookTypes string

const (
	CLIENT_ADDED             HookTypes = "CLIENT_ADDED"
	CLIENT_CLOSED_CONNECTION HookTypes = "CLIENT_CLOSED_CONNECTION"
	ERROR                    HookTypes = "ERROR"
)

const (
	ERR_HANDLER HandlerName = "ERR_HANDLER"
	ERR_DECODE  HandlerName = "ERR_DECODE"
	USER_ID     HandlerName = "USER_ID"
)

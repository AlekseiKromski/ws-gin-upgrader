package core

type Handler interface {
	Handle(payload string, client *Client, clients Clients)
}

type HandlerName string

type Handlers map[HandlerName]Handler

func (hs Handlers) DefineHandler(action HandlerName) Handler {
	return hs[action]
}

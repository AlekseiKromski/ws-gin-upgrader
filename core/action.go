package core

import "encoding/json"

type ActionHandlerInterface interface {
	SetData(data string)
	Do()
	TrigType() string
	SetClient(client *Client)
}

type ActionHandler struct {
	ActionType string
	Data       string
	Action     ActionHandlerInterface
}

type Action struct {
	ActionType string `json:"actionType"`
	Data       string `json:"data"`
}

type ActionsWorker struct {
	actions []*ActionHandler
}

func (aw *ActionsWorker) registerHandler(handler *ActionHandler) {
	aw.actions = append(aw.actions, handler)
}

func (aw *ActionsWorker) defineAction(message []byte) (*ActionHandler, error) {
	var action Action
	err := json.Unmarshal(message, &action)
	if err != nil {
		return nil, err
	}
	for _, actionHandler := range aw.actions {
		if actionHandler.ActionType == action.ActionType {
			actionHandler.Action.SetData(action.Data)
			return actionHandler, nil
		}
	}
	return nil, nil
}

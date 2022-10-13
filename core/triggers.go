package core

type TriggerHandlerInterface interface {
	Do()
	SetData(data string)
	SetClient(client *Client)
	SetClients(client []*Client)
}

type TriggerHandler struct {
	TriggerType string
	data        string
	Action      TriggerHandlerInterface
}

type TriggersWorker struct {
	triggers []*TriggerHandler
}

func (th *TriggersWorker) registerHandler(handler *TriggerHandler) {
	th.triggers = append(th.triggers, handler)
}

func (th *TriggersWorker) defineTrigger(triggerType string) (*TriggerHandler, error) {
	for _, triggerHandler := range th.triggers {
		if triggerHandler.TriggerType == triggerType {
			return triggerHandler, nil
		}
	}
	return nil, nil
}

package main

import (
	"fmt"
	"github.com/AlekseiKromski/at-socket-server/core"
)

var actionHandlers = []*core.ActionHandler{
	{},
}

var triggerHandlers = []*core.TriggerHandler{
	{},
}

func main() {
	_, err := core.Start(actionHandlers, triggerHandlers)
	if err != nil {
		fmt.Println(err)
	}
}

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
	app, err := core.Start(actionHandlers, triggerHandlers)
	if err != nil {
		fmt.Println(err)
	}

	//Example of working with hooks
	go func() {
		for {
			select {
			case hook := <-app.Hooks:
				if hook.HookType == core.CLIENT_ADDED {
					fmt.Println("hook :D")
				}
			}
		}
	}()

	for {
	}

}

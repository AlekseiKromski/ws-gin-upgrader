package e2e

import (
	"encoding/json"
	"fmt"
	"github.com/AlekseiKromski/at-socket-server/core"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/url"
	"sync"
	"testing"
)

type senderPayload struct {
	ReceiverID string `json:"receiverID"`
	Message    string `json:"message"`
}

type SenderHandler struct{}

func (sh *SenderHandler) Handle(payload string, client *core.Client, clients core.Clients) {
	sp := senderPayload{}

	if err := json.Unmarshal([]byte(payload), &sp); err != nil {
		if err := client.Conn.WriteJSON(core.ActionModel{
			Action:  core.ERR_DECODE,
			Payload: fmt.Sprintf("cannot decode payload: %v", err),
		}); err != nil {
			fmt.Printf("cannot send error back: %v", err)
		}
	}

	if err := clients[sp.ReceiverID].Conn.WriteJSON(core.ActionModel{
		Action:  "NEW_MESSAGE",
		Payload: sp.Message,
	}); err != nil {
		if err := client.Conn.WriteJSON(core.ActionModel{
			Action:  core.HandlerName(core.ERR_DECODE),
			Payload: fmt.Sprintf("cannot decode payload: %v", err),
		}); err != nil {
			fmt.Printf("cannot send error back: %v", err)
		}
	}
}

func Test_MessageSend(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	handlers := make(core.Handlers)
	handlers["SEND_MESSAGE"] = &SenderHandler{}

	conf := &core.Config{
		CorsOptions: cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{
				http.MethodGet,
				http.MethodPost,
			},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		},
		Host:  "localhost",
		Port:  3000,
		Debug: true,
	}

	app, err := core.Start(&handlers, conf)
	if err != nil {
		fmt.Println(err)
	}

	//waiting server start event
blocked:
	for {
		hook := <-app.Hooks
		switch hook.HookType {
		case core.SERVER_STARTED:
			break blocked
		}
	}

	server_url := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/"}
	ready := make(chan string)

	//receiver
	go func() {
		c, _, err := websocket.DefaultDialer.Dial(server_url.String(), nil)
		if err != nil {
			log.Fatal("dial:", err)
		}
		defer c.Close()

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			fmt.Printf("Received: %s", string(message))

			am := core.ActionModel{}
			if err := json.Unmarshal(message, &am); err != nil {
				t.Errorf("cannot unmarshal message: %v", err)
				return
			}

			switch am.Action {
			case "USER_ID":
				ready <- am.Payload
				break
			case "NEW_MESSAGE":
				assert.Equal(t, "Hello", am.Payload)
				wg.Done()
				return
			}
		}
	}()

	//sender
	go func() {
		c, _, err := websocket.DefaultDialer.Dial(server_url.String(), nil)
		if err != nil {
			log.Fatal("dial:", err)
		}
		defer c.Close()

		id := <-ready

		message, err := json.Marshal(senderPayload{
			ReceiverID: id,
			Message:    "Hello",
		})
		if err != nil {
			t.Errorf("cannot decode message: %v", err)
			return
		}

		if err := c.WriteJSON(core.ActionModel{
			Action:  "SEND_MESSAGE",
			Payload: string(message),
		}); err != nil {
			t.Errorf("cannot send message: %v", err)
			return
		}

		wg.Wait()
	}()

	wg.Wait()
}

func Test_disconnect(t *testing.T) {
	handlers := make(core.Handlers)
	handlers["SEND_MESSAGE"] = &SenderHandler{}

	conf := &core.Config{
		CorsOptions: cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{
				http.MethodGet,
				http.MethodPost,
			},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		},
		Host:  "localhost",
		Port:  3000,
		Debug: true,
	}

	app, err := core.Start(&handlers, conf)
	if err != nil {
		fmt.Println(err)
	}

	//waiting server start event
blockedOpen:
	for {
		hook := <-app.Hooks
		switch hook.HookType {
		case core.SERVER_STARTED:
			break blockedOpen
		}
	}

	server_url := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(server_url.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	c.Close()

	//waiting server start event
blockedClosed:
	for {
		hook := <-app.Hooks
		fmt.Println(hook.HookType)
		switch hook.HookType {
		case core.CLIENT_CLOSED_CONNECTION:
			break blockedClosed
		case core.ERROR:
			break blockedClosed
		}
	}
}

func Test_callUndefinedHandlerAction(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	handlers := make(core.Handlers)
	handlers["SEND_MESSAGE"] = &SenderHandler{}

	conf := &core.Config{
		CorsOptions: cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{
				http.MethodGet,
				http.MethodPost,
			},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		},
		Host:  "localhost",
		Port:  3000,
		Debug: true,
	}

	app, err := core.Start(&handlers, conf)
	if err != nil {
		fmt.Println(err)
	}

	go func() {
		for {
			hook := <-app.Hooks
			switch hook.HookType {
			case core.CLIENT_ADDED:
				fmt.Printf("Client added: %s", hook.Data)
			case core.CLIENT_CLOSED_CONNECTION:
				fmt.Printf("Client closed connection: %s", hook.Data)
			case core.ERROR:
				fmt.Printf("Error: %s", hook.Data)
			}
		}
	}()

	//waiting server start event
blockedOpen:
	for {
		hook := <-app.Hooks
		switch hook.HookType {
		case core.SERVER_STARTED:
			break blockedOpen
		}
	}

	server_url := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(server_url.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	go func() {
		//waiting server start event
		for {
			_, message, _ := c.ReadMessage()
			am := core.ActionModel{}
			if err := json.Unmarshal(message, &am); err != nil {
				t.Fatalf("cannot decode incoming message: %v", err)
			}

			if am.Action == core.ERR_HANDLER {
				wg.Done()
				return
			}
		}
	}()

	am := core.ActionModel{
		Action:  "RANDOM",
		Payload: "RANDOM",
	}

	if err := c.WriteJSON(am); err != nil {
		t.Fatalf("cannot send payload: %v", err)
	}

	wg.Wait()
}

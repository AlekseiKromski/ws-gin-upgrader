package internal

import (
	"encoding/json"
	"github.com/AlekseiKromski/ws-gin-upgrader/core"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

const MESSAGE_SENT = "MESSAGE_SENT"

type MessageSentHandler struct {
	callback func(p string, c *core.Client)
}

func (msh *MessageSentHandler) Handle(payload string, client *core.Client, clients core.Clients) {
	msh.callback(payload, client)
}

func middleware(c *gin.Context) {
	c.Set("uid", "mocked-client-uuid")
}

/*
Test checks, that we can start HTTP server with minimal upgrader setup
*/
func Test_CreateUpgrader(t *testing.T) {
	engine := gin.Default()
	gin.SetMode(gin.ReleaseMode) // to remove GIN logs
	httpServer := httptest.NewUnstartedServer(engine)

	_, _ = core.Start(engine, &core.Handlers{}, func(c *gin.Context) {}, &core.Config{JwtSecret: make([]byte, 0)})

	httpServer.Start()
	defer httpServer.Close()
}

func Test_ConnectionEndpoint(t *testing.T) {
	engine := gin.Default()
	httpServer := httptest.NewUnstartedServer(engine)

	_, _ = core.Start(engine, &core.Handlers{}, middleware, &core.Config{JwtSecret: make([]byte, 0)})

	httpServer.Start()
	defer httpServer.Close()

	server_url := url.URL{Scheme: "ws", Host: httpServer.Listener.Addr().String(), Path: "/ws/connect"}
	ws, response, err := websocket.DefaultDialer.Dial(server_url.String(), nil)
	if err != nil {
		t.Errorf("cannot make connection to websocket endpoint: %v", err)
		return
	}

	assert.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
	defer ws.Close()
}

func Test_AppEvents(t *testing.T) {
	engine := gin.Default()
	httpServer := httptest.NewUnstartedServer(engine)

	app, _ := core.Start(engine, &core.Handlers{}, middleware, &core.Config{JwtSecret: make([]byte, 0)})

	httpServer.Start()
	defer httpServer.Close()

	// Create events listener
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			select {
			case event := <-app.Hooks:
				if event.HookType == core.CLIENT_ADDED {
					assert.Equal(t, "mocked-client-uuid", event.Data)
					log.Printf("Client added, uid from middleware: %s", event.Data)
					wg.Done()
					return
				}
			}
		}
	}()

	server_url := url.URL{Scheme: "ws", Host: httpServer.Listener.Addr().String(), Path: "/ws/connect"}
	ws, response, err := websocket.DefaultDialer.Dial(server_url.String(), nil)
	if err != nil {
		t.Errorf("cannot make connection to websocket endpoint: %v", err)
		return
	}
	defer ws.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)

	wg.Wait()
}

func Test_SendingMessageToServer(t *testing.T) {
	engine := gin.Default()
	httpServer := httptest.NewUnstartedServer(engine)

	wg := sync.WaitGroup{}
	wg.Add(1)

	_, _ = core.Start(engine, &core.Handlers{
		MESSAGE_SENT: &MessageSentHandler{
			callback: func(p string, c *core.Client) {
				defer wg.Done()
				log.Printf("Message received in handler: %s", p)
			},
		},
	}, middleware, &core.Config{JwtSecret: make([]byte, 0)})

	httpServer.Start()
	defer httpServer.Close()

	server_url := url.URL{Scheme: "ws", Host: httpServer.Listener.Addr().String(), Path: "/ws/connect"}
	wsConnection, response, err := websocket.DefaultDialer.Dial(server_url.String(), nil)
	if err != nil {
		t.Errorf("cannot make connection to websocket endpoint: %v", err)
		return
	}
	defer wsConnection.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)

	if err := wsConnection.WriteJSON(core.ActionModel{
		Action:  MESSAGE_SENT,
		Payload: "some message information...",
	}); err != nil {
		t.Errorf("cannot send message to server: %v", err)
		return
	}

	wg.Wait()
}

func Test_ReceivingMessageFromServer(t *testing.T) {
	engine := gin.Default()
	httpServer := httptest.NewUnstartedServer(engine)

	_, _ = core.Start(engine, &core.Handlers{
		MESSAGE_SENT: &MessageSentHandler{
			callback: func(p string, client *core.Client) {
				log.Printf("Message received in handler: %s", p)
				if err := client.Conn.WriteJSON(core.ActionModel{
					Action:  MESSAGE_SENT,
					Payload: "some message from server",
				}); err != nil {
					t.Errorf("cannot send message back to client: %v", err)
					return
				}
			},
		},
	}, middleware, &core.Config{JwtSecret: make([]byte, 0)})

	httpServer.Start()
	defer httpServer.Close()

	server_url := url.URL{Scheme: "ws", Host: httpServer.Listener.Addr().String(), Path: "/ws/connect"}
	wsConnection, response, err := websocket.DefaultDialer.Dial(server_url.String(), nil)
	if err != nil {
		t.Errorf("cannot make connection to websocket endpoint: %v", err)
		return
	}
	defer wsConnection.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)

	for {
		_, payload, err := wsConnection.ReadMessage()
		if err != nil {
			t.Errorf("cannot read message: %v", err)
			return
		}

		info := &core.ActionModel{}
		if err := json.Unmarshal(payload, info); err != nil {
			t.Errorf("cannot unmarshal message: %v", err)
			return
		}

		// Should receive message sent from handler
		if info.Action == MESSAGE_SENT {
			log.Printf("Received message from server: %s", info.Payload)
			assert.Equal(t, "some message from server", info.Payload)
			break
		}

		if info.Action == core.USER_ID {
			log.Printf("Received user id event from server: %s", info.Payload)
			if err := wsConnection.WriteJSON(core.ActionModel{
				Action:  MESSAGE_SENT,
				Payload: "some payload from client",
			}); err != nil {
				t.Errorf("cannot send message to server: %v", err)
			}
			continue
		}
	}
}

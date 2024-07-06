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
	callback func(p string, s *core.Session, clients core.Clients)
}

func (msh *MessageSentHandler) Handle(payload string, client *core.Session, clients core.Clients) {
	msh.callback(payload, client, clients)
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
					log.Printf("Session added, uid from middleware: %s", event.Data)
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
			callback: func(p string, s *core.Session, c core.Clients) {
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
			callback: func(p string, s *core.Session, clients core.Clients) {
				log.Printf("Message received in handler: %s", p)
				if err := s.Conn.WriteJSON(core.ActionModel{
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

func Test_ReceivingMessageFromServerToAllWsClients(t *testing.T) {
	engine := gin.Default()
	httpServer := httptest.NewUnstartedServer(engine)

	_, _ = core.Start(engine, &core.Handlers{
		MESSAGE_SENT: &MessageSentHandler{
			callback: func(p string, s *core.Session, clients core.Clients) {
				log.Printf("Message received in handler: %s", p)
				if err := clients.Send(s.ID, "some message from server", MESSAGE_SENT); err != nil {
					t.Errorf("cannot send message back to client: %v", err)
					return
				}
			},
		},
	}, middleware, &core.Config{JwtSecret: make([]byte, 0)})

	httpServer.Start()
	defer httpServer.Close()

	wgConnections := sync.WaitGroup{}
	wgConnections.Add(10)

	wgMessageFromServer := sync.WaitGroup{}
	wgMessageFromServer.Add(10)

	server_url := url.URL{Scheme: "ws", Host: httpServer.Listener.Addr().String(), Path: "/ws/connect"}

	for i := 0; i < 10; i++ {
		go func() {
			wsConnection := createWebsocketConnection(t, server_url)
			defer wsConnection.Close()

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

					wgMessageFromServer.Done()

					break
				}

				if info.Action == core.USER_ID {
					log.Printf("Received user id event from server: %s", info.Payload)
					wgConnections.Done()
					continue
				}
			}
		}()
	}

	wgConnections.Wait()

	wsConnection := createWebsocketConnection(t, server_url)

	if err := wsConnection.WriteJSON(core.ActionModel{
		Action:  MESSAGE_SENT,
		Payload: "some payload from client",
	}); err != nil {
		t.Errorf("cannot send message to server: %v", err)
	}

	wgMessageFromServer.Wait()
}

func createWebsocketConnection(t *testing.T, server_url url.URL) *websocket.Conn {
	wsConnection, response, err := websocket.DefaultDialer.Dial(server_url.String(), nil)
	if err != nil {
		t.Errorf("cannot make connection to websocket endpoint: %v", err)
		return nil
	}
	assert.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)

	return wsConnection
}

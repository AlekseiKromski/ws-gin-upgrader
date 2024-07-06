package core

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

type HookType struct {
	HookType HookTypes
	Data     string
}

func NewHook(ht HookTypes, data string) HookType {
	return HookType{
		HookType: ht,
		Data:     data,
	}
}

type App struct {
	Hooks                  chan HookType
	Config                 *Config
	Clients                Clients
	engine                 *gin.Engine
	handlers               *Handlers
	server                 string
	httpConnectionUpgraded websocket.Upgrader
	mutex                  sync.Mutex
	middleware             func(c *gin.Context)
}

func Start(engine *gin.Engine, hs *Handlers, middleware func(c *gin.Context), conf *Config) (*App, error) {
	app := App{engine: engine, Config: conf, Clients: Clients{
		make(map[string][]*Session),
	}, mutex: sync.Mutex{}, middleware: middleware}

	//Start application
	app.runApp(hs)

	//Configure server handlers and setup ws upgrader
	if err := app.serverConfigure(); err != nil {
		return nil, fmt.Errorf("cannot up server: %v", err)
	}

	return &app, nil
}

func (app *App) runApp(hs *Handlers) {
	app.initHooksChannel()
	app.registerHandlers(hs)
}

func (app *App) registerHandlers(hs *Handlers) {
	app.handlers = hs
	app.log("handlers registered")
}

func (app *App) initHooksChannel() {
	app.Hooks = make(chan HookType)
	app.log("hooks initiated")
}

func (app *App) serverConfigure() error {
	app.httpConnectionUpgraded = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	app.log("WebsocketUpgrader config: ")
	app.log("	- Check origin: true (configured in function)") // Default value is true always
	app.log(fmt.Sprintf("	- Read buffer size: %d", app.httpConnectionUpgraded.ReadBufferSize))
	app.log(fmt.Sprintf("	- Write buffer size: %d", app.httpConnectionUpgraded.WriteBufferSize))

	wsGroup := app.engine.Group("/ws/").Use(app.middleware)
	{
		wsGroup.GET("/connect", func(c *gin.Context) {
			userID, exists := c.Get("uid")
			if !exists {
				c.Status(http.StatusForbidden)
				return
			}

			conn, err := app.httpConnectionUpgraded.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				log.Printf("problem while upgrade http connection to webscket: %v", err)
				return
			}
			client := app.addClient(userID.(string), conn)

			app.sendHook(NewHook(CLIENT_ADDED, client.ID))

			if err := client.Handler(app); err != nil {
				fmt.Printf("cannot handle client: %v", err)
			}
		})
	}

	app.log("Created new router group '/ws/'")
	app.log("Created new router endpoint with handler '/connect/'")
	return nil
}

func (app *App) sendHook(h HookType) {
	select {
	case app.Hooks <- h:
	default:
	}
}

func (app *App) addClient(userID string, conn *websocket.Conn) *Session {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	c := CreateNewClient(userID, conn)

	app.Clients.Storage[c.ID] = append(app.Clients.Storage[c.ID], c)
	return c
}

func (app *App) removeClient(id, sid string) {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if app.Clients.Storage[id] == nil {
		return
	}

	// Filter out from the client by session id
	clients := []*Session{}
	for _, client := range app.Clients.Storage[id] {
		if client.SID == sid {
			continue
		}

		clients = append(clients, client)
	}

	// If we don't have active connections, no need to save this client
	if len(clients) == 0 {
		delete(app.Clients.Storage, id)
		return
	}

	app.Clients.Storage[id] = clients
}

func (app *App) log(l string) {
	log.Printf("[%s] %s", "WS-GIN-UPGRADER", l)
}

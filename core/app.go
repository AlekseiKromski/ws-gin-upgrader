package core

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"net/http"
	"sync"
)

type WebSocket struct{}

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
	clients                Clients
	handlers               *Handlers
	config                 *Config
	server                 string
	httpConnectionUpgraded websocket.Upgrader
	mutex                  sync.Mutex
}

func Start(hs *Handlers, conf *Config) (*App, error) {
	app := App{config: conf, clients: make(Clients), mutex: sync.Mutex{}}

	//Start application
	app.runApp(hs)
	//Up server and handle controller
	go app.serverUp()

	return &app, nil
}

func (app *App) runApp(hs *Handlers) {
	app.initHooksChannel()
	app.registerHandlers(hs)
}

func (app *App) registerHandlers(hs *Handlers) {
	app.handlers = hs
}

func (app *App) initHooksChannel() {
	app.Hooks = make(chan HookType)
}

func (app *App) serverUp() error {
	mux := http.NewServeMux()
	corsSettings := cors.New(app.config.CorsOptions)
	app.httpConnectionUpgraded = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := app.httpConnectionUpgraded.Upgrade(w, r, nil)
		if err != nil {
			fmt.Printf("problem while upgrade http connection to webscket: %v", err)
			return
		}
		client := app.addClient(conn)

		app.sendHook(NewHook(CLIENT_ADDED, client.ID))

		if err := client.Handler(app); err != nil {
			fmt.Printf("cannot handle client: %v", err)
		}
	})

	handler := corsSettings.Handler(mux)
	app.sendHook(NewHook(SERVER_STARTED, fmt.Sprintf("started on: %s", app.config.GetServerString())))
	http.ListenAndServe(app.config.GetServerString(), handler)
	return nil
}

func (app *App) sendHook(h HookType) {
	select {
	case app.Hooks <- h:
	default:
	}
}

func (app *App) addClient(conn *websocket.Conn) *Client {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	for {
		c := CreateNewClient(conn)
		if app.clients[c.ID] == nil {
			app.clients[c.ID] = c
			return c
		}
	}
}

func (app *App) removeClient(id string) {
	app.mutex.Lock()
	defer app.mutex.Unlock()

	if app.clients[id] == nil {
		return
	}

	delete(app.clients, id)
}

package core

import (
	"fmt"
	"github.com/gorilla/websocket"
	cors "github.com/rs/cors"
	"net/http"
	"path/filepath"
)

type App struct {
	config                 Config
	server                 string
	httpConnectionUpgraded websocket.Upgrader
	clients                []*Client
	ActionsWorker          *ActionsWorker
	TriggersWorker         *TriggersWorker
	Hooks                  chan HookType
}

type WebSocket struct{}

type HookType struct {
	HookType HookTypes
	Data     string
}

func Start(actions []*ActionHandler, triggers []*TriggerHandler) (*App, error) {
	//Try to load configuration
	path := filepath.Join(".", "config.json")
	config, err := LoadConfig(path)
	if err != nil {
		return &App{}, err
	}

	app := App{config: config}
	//Start application
	app.runApp(actions, triggers)
	//Up server and handle controller
	go app.serverUp()
	if err != nil {
		return &App{}, err
	}
	return &app, nil
}

func (app *App) registerTriggers(triggers []*TriggerHandler) {
	for _, trigger := range triggers {
		app.TriggersWorker.registerHandler(trigger)
	}
}

func (app *App) registerActions(actions []*ActionHandler) {
	for _, action := range actions {
		app.ActionsWorker.registerHandler(action)
	}
}

func (app *App) registerWorkers() {
	app.ActionsWorker = &ActionsWorker{}
	app.TriggersWorker = &TriggersWorker{}
}

func (app *App) runApp(actions []*ActionHandler, triggers []*TriggerHandler) {
	app.initHooksChannel()
	app.registerWorkers()
	app.registerActions(actions)
	app.registerTriggers(triggers)
}

func (app *App) initHooksChannel() {
	app.Hooks = make(chan HookType)
}

func (app *App) serverUp() error {
	fmt.Println("Start server")

	mux := http.NewServeMux()
	corsSettings := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
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
		fmt.Println("Client was connected")
		client := app.addClient(conn)

		app.Hooks <- HookType{
			HookType: CLIENT_ADDED,
			Data:     string(client.ID),
		}

		client.Handler(app)
	})

	handler := corsSettings.Handler(mux)
	http.ListenAndServe(app.config.GetServerString(), handler)
	return nil
}

func (app *App) addClient(conn *websocket.Conn) *Client {
	client := CreateNewClient(conn, &app.config)
	app.clients = append(app.clients, client)
	return client
}

func (app *App) getInstance() *App {
	return app
}

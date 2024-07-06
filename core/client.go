package core

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Clients struct {
	Storage map[string][]*Session
}

func (c *Clients) Send(uid, payload string, action HandlerName) error {
	for _, session := range c.Storage[uid] {
		am := ActionModel{
			Action:  action,
			Payload: payload,
		}

		err := session.Conn.WriteJSON(am)
		if err != nil {
			return err
		}
	}

	return nil
}

type Session struct {
	Conn *websocket.Conn
	ID   string
	SID  string
}

func CreateNewClient(clientID string, connection *websocket.Conn) *Session {
	return &Session{
		Conn: connection,
		ID:   clientID,
		SID:  uuid.New().String(),
	}
}

func (s *Session) Handler(app *App) error {
	if err := s.Conn.WriteJSON(ActionModel{
		Action:  USER_ID,
		Payload: s.ID,
	}); err != nil {
		return fmt.Errorf("cannot send USER_ID: %v", err)
	}
	go s.startReceiveChannel(app)
	return nil
}

func (s *Session) Send(payload string, action HandlerName) error {
	am := ActionModel{
		Action:  action,
		Payload: payload,
	}

	return s.Conn.WriteJSON(am)
}

func (s *Session) startReceiveChannel(app *App) {
	defer func() {
		e := s.Conn.Close()
		if e != nil {
			return
		}
		app.removeClient(s.ID, s.SID)
	}()

	for {
		_, message, err := s.Conn.ReadMessage()
		if err != nil {
			if err, ok := err.(*websocket.CloseError); ok {
				app.sendHook(NewHook(CLIENT_CLOSED_CONNECTION, fmt.Sprintf("connection closed: %v", err)))
				return
			}

			app.sendHook(NewHook(ERROR, fmt.Sprintf("error in clinet: %v", err)))
			return
		}

		am := ActionModel{}
		if err := json.Unmarshal(message, &am); err != nil {
			if err := s.Conn.WriteJSON(ActionModel{
				Action:  ERR_DECODE,
				Payload: fmt.Sprintf("cannot decode your message: %v", err),
			}); err != nil {
				fmt.Printf("cannot send message: %v", err)
			}
		}

		handler := app.handlers.DefineHandler(am.Action)
		if handler == nil {
			app.sendHook(NewHook(ERROR, fmt.Sprintf("cannot define handler: %s", am.Action)))
			s.sendError(fmt.Sprintf("cannot define handler: %s", am.Action), app)
			continue
		}

		handler.Handle(am.Payload, s, app.Clients)
	}
}

func (s *Session) sendError(message string, app *App) {
	am := ActionModel{
		Action:  ERR_HANDLER,
		Payload: message,
	}

	if err := s.Conn.WriteJSON(am); err != nil {
		app.sendHook(NewHook(ERROR, fmt.Sprintf("cannot send message (%s): %v", message, err)))
	}
}

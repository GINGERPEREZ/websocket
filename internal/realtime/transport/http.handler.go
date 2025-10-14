package transport

import (
	"mesaYaWs/internal/realtime/infrastructure"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewWebsocketHandler(hub *infrastructure.Hub) func(echo.Context) error {
	return func(c echo.Context) error {
		topic := c.QueryParam("topic")
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		client := &infrastructure.Client{
			conn: conn,
			send: make(chan []byte, 8),
		}

		hub.Register(topic, client)
		go client.WritePump()
		return nil
	}
}

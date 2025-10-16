package transport

import (
	"net/http"
	"strings"

	"mesaYaWs/internal/realtime/infrastructure"
	"mesaYaWs/internal/shared/auth"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWebsocketHandler crea un handler que expone la ruta /ws/restaurant/:section/:token
// donde :section es el namespace/section y :token es el JWT emitido por el servicio Nest.
// Valida el JWT localmente con el validador proporcionado y registra al cliente en el hub.
func NewWebsocketHandler(hub *infrastructure.Hub, validator auth.TokenValidator) func(echo.Context) error {
	return func(c echo.Context) error {
		section := c.Param("section")
		token := c.Param("token")

		if strings.TrimSpace(section) == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "missing section")
		}

		// upgrade
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		// validate token
		claims, err := validator.Validate(token)
		if err != nil {
			_ = conn.Close()
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		userID := claims.RegisteredClaims.Subject
		sessionID := claims.SessionID

		// create client and attach
		client := infrastructure.NewClient(hub, conn, userID, sessionID, 8)
		// Attach client subscribed to the section's snapshot topic by default
		defaultTopic := section + ".snapshot"
		hub.AttachClient(client, []string{defaultTopic})

		// start pumps
		go client.WritePump()
		go client.ReadPump()
		return nil
	}
}

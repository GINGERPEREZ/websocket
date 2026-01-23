package transport

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"mesaYaWs/internal/modules/realtime/application/usecase"
	"mesaYaWs/internal/modules/realtime/domain"
)

// BroadcastRequest represents the payload for broadcasting a message via REST API
type BroadcastRequest struct {
	Event         string                 `json:"event"`
	Topic         string                 `json:"topic,omitempty"`
	ReservationID string                 `json:"reservation_id,omitempty"`
	PaymentID     string                 `json:"payment_id,omitempty"`
	Status        string                 `json:"status,omitempty"`
	Amount        float64                `json:"amount,omitempty"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	Data          map[string]interface{} `json:"data,omitempty"`
}

// BroadcastResponse represents the response after broadcasting
type BroadcastResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Event   string `json:"event"`
}

// NewBroadcastHTTPHandler creates a REST endpoint for broadcasting messages
// This is used by n8n workflows to push payment notifications to connected clients
func NewBroadcastHTTPHandler(broadcastUC *usecase.BroadcastUseCase) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req BroadcastRequest
		if err := c.Bind(&req); err != nil {
			slog.Warn("broadcast http: invalid request body", slog.Any("error", err))
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}

		// Validate required field
		if req.Event == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "event field is required")
		}

		// Determine topic from event or use provided topic
		topic := req.Topic
		if topic == "" {
			topic = req.Event // Use event as topic if not specified
		}

		// Build the data payload
		data := make(map[string]interface{})
		data["event"] = req.Event
		if req.ReservationID != "" {
			data["reservation_id"] = req.ReservationID
		}
		if req.PaymentID != "" {
			data["payment_id"] = req.PaymentID
		}
		if req.Status != "" {
			data["status"] = req.Status
		}
		if req.Amount > 0 {
			data["amount"] = req.Amount
		}
		if req.Timestamp != "" {
			data["timestamp"] = req.Timestamp
		}
		// Merge additional data
		for k, v := range req.Data {
			data[k] = v
		}

		// Create the domain message using correct struct fields
		msg := &domain.Message{
			Topic:     topic,
			Entity:    "payment",
			Action:    "status-changed",
			Data:      data,
			Timestamp: time.Now(),
		}

		// If we have a reservation ID, use it as ResourceID
		if req.ReservationID != "" {
			msg.ResourceID = req.ReservationID
		} else if req.PaymentID != "" {
			msg.ResourceID = req.PaymentID
		}

		// Execute broadcast
		broadcastUC.Execute(c.Request().Context(), msg)

		slog.Info("broadcast http: message sent",
			slog.String("event", req.Event),
			slog.String("topic", topic),
			slog.String("reservationId", req.ReservationID),
			slog.String("paymentId", req.PaymentID),
		)

		return c.JSON(http.StatusOK, BroadcastResponse{
			Success: true,
			Message: "Message broadcasted successfully",
			Event:   req.Event,
		})
	}
}

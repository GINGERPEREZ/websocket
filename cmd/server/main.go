package main

import (
	"context"
	"log"
	"mesaYaWs/internal/app"
	"mesaYaWs/internal/broker"
	"mesaYaWs/internal/config"
	handler "mesaYaWs/internal/realtime/application/handler"
	usecase "mesaYaWs/internal/realtime/application/usecase"
	"mesaYaWs/internal/realtime/infrastructure"
	"mesaYaWs/internal/realtime/transport"
	"mesaYaWs/internal/shared/auth"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
)

func main() {
	cfg := config.Load()
	hub := app.NewAppHub()
	registry := infrastructure.NewHandlerRegistry()

	// Registrar handlers de tópicos (cada feature)
	registry.Register(&handler.UserCreatedHandler{UseCase: usecase.NewBroadcastUseCase(hub)})

	// Iniciar Kafka consumers (registrar topics desde config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// gather topics from config
	topics := make([]string, 0, len(cfg.EntityTopics))
	for _, t := range cfg.EntityTopics {
		topics = append(topics, t)
	}
	broker.StartKafkaConsumers(ctx, registry, cfg.KafkaBrokers, topics)

	// Echo server
	e := echo.New()

	// JWT validator used to validate tokens issued by the Nest auth service
	validator := auth.NewJWTValidator(cfg.JWTSecret)

	// expose websocket route for restaurant sections: /ws/restaurant/:section/:token
	e.GET("/ws/restaurant/:section/:token", transport.NewWebsocketHandler(hub, validator))

	go func() {
		if err := e.Start(":" + cfg.ServerPort); err != nil {
			log.Fatal(err)
		}
	}()

	// Esperar señales
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down...")
	e.Close()
}

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

	// Iniciar Kafka consumers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	broker.StartKafkaConsumers(ctx, registry, cfg.KafkaBrokers, []string{"user.created"})

	// Echo server
	e := echo.New()
	e.GET("/ws", transport.NewWebsocketHandler(hub))

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

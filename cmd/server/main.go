package main

import (
	"context"
	"fmt"
	"io"
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
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	cfg := config.Load()

	logFile, err := setupLogging(cfg.LogDir)
	if err != nil {
		log.Fatalf("failed to initialize logging: %v", err)
	}
	defer logFile.Close()

	hub := app.NewAppHub()
	registry := infrastructure.NewHandlerRegistry()

	// Use cases
	broadcastUC := usecase.NewBroadcastUseCase(hub)

	// Echo server
	e := echo.New()
	e.Logger.SetOutput(log.Writer())

	// JWT validator used to validate tokens issued by the Nest auth service
	validator := auth.NewJWTValidator(cfg.JWTSecret)
	snapshotFetcher := infrastructure.NewSectionSnapshotHTTPClient(cfg.RestBaseURL, nil)
	connectUC := usecase.NewConnectSectionUseCase(validator, snapshotFetcher)

	// Registrar handlers de tópicos (cada feature)
	registry.Register(&handler.UserCreatedHandler{UseCase: broadcastUC})
	for entity, topic := range cfg.EntityTopics {
		registry.Register(handler.NewEntityStreamHandler(entity, topic, cfg.AllowedActions, broadcastUC, connectUC))
	}

	// Iniciar Kafka consumers (registrar topics desde config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// gather topics from config
	topics := make([]string, 0, len(cfg.EntityTopics))
	for _, t := range cfg.EntityTopics {
		topics = append(topics, t)
	}
	broker.StartKafkaConsumers(ctx, registry, cfg.KafkaBrokers, topics)

	// expose websocket route for restaurant sections: /ws/restaurant/:section/:token
	e.GET("/ws/restaurant/:section/:token", transport.NewWebsocketHandler(hub, connectUC, "restaurants", cfg.AllowedActions))

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

func setupLogging(dir string) (*os.File, error) {
	if dir == "" {
		dir = "./logs"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	fileName := filepath.Join(dir, time.Now().UTC().Format("2006-01-02")+".log")
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	mw := io.MultiWriter(os.Stdout, file)
	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	log.Printf("logging initialized file=%s", fileName)
	return file, nil
}

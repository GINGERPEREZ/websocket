package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mesaYaWs/internal/app"
	"mesaYaWs/internal/broker"
	"mesaYaWs/internal/config"
	handler "mesaYaWs/internal/realtime/application/handler"
	usecase "mesaYaWs/internal/realtime/application/usecase"
	"mesaYaWs/internal/realtime/infrastructure"
	"mesaYaWs/internal/realtime/transport"
	"mesaYaWs/internal/shared/auth"
	"mesaYaWs/internal/shared/logging"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load error: %v\n", err)
		os.Exit(1)
	}

	logFile, logger, err := setupLogging(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logging setup error: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	slog.SetDefault(logger)
	slog.Info("logging initialized", slog.String("directory", cfg.Logging.Directory), slog.String("level", cfg.Logging.Level), slog.String("format", cfg.Logging.Format))

	hub := app.NewAppHub()
	registry := infrastructure.NewHandlerRegistry()

	// Use cases
	broadcastUC := usecase.NewBroadcastUseCase(hub)

	// Echo server
	e := echo.New()
	e.Logger.SetOutput(log.Writer())

	// JWT validator used to validate tokens issued by the Nest auth service
	validator := auth.NewJWTValidator(cfg.Security.JWTSecret)
	snapshotFetcher := infrastructure.NewSectionSnapshotHTTPClient(cfg.REST.BaseURL, cfg.REST.Timeout, nil)
	connectUC := usecase.NewConnectSectionUseCase(validator, snapshotFetcher)

	// Registrar handlers de tópicos (cada feature)
	registry.Register(&handler.UserCreatedHandler{UseCase: broadcastUC})
	for entity, topic := range cfg.Kafka.Topics {
		registry.Register(handler.NewEntityStreamHandler(entity, topic, cfg.Websocket.AllowedActions, broadcastUC, connectUC))
	}

	// Iniciar Kafka consumers (registrar topics desde config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// gather topics from config
	topics := make([]string, 0, len(cfg.Kafka.Topics))
	for _, t := range cfg.Kafka.Topics {
		topics = append(topics, t)
	}
	broker.StartKafkaConsumers(ctx, registry, cfg.Kafka.Brokers, cfg.Kafka.GroupID, topics)

	// expose websocket route for restaurant sections: /ws/restaurant/:section/:token
	e.GET("/ws/restaurant/:section/:token", transport.NewWebsocketHandler(hub, connectUC, cfg.Websocket.DefaultEntity, cfg.Websocket.AllowedActions))

	go func() {
		if err := e.Start(":" + cfg.Server.Port); err != nil {
			slog.Error("http server stopped", slog.Any("error", err))
		}
	}()

	// Esperar señales
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
	e.Close()
}

func setupLogging(cfg config.LoggingConfig) (*os.File, *slog.Logger, error) {
	dir := cfg.Directory
	if dir == "" {
		dir = "./logs"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create log dir: %w", err)
	}
	fileName := filepath.Join(dir, time.Now().UTC().Format("2006-01-02")+".log")
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	writer := io.MultiWriter(os.Stdout, file)
	logger := logging.New(writer, logging.Config{
		Level:     cfg.Level,
		Format:    cfg.Format,
		AddSource: true,
	})
	log.SetOutput(writer)
	log.SetFlags(0)
	log.SetPrefix("")

	return file, logger, nil
}

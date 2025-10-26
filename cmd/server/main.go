package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	"mesaYaWs/internal/config"
	handler "mesaYaWs/internal/modules/realtime/application/handler"
	usecase "mesaYaWs/internal/modules/realtime/application/usecase"
	"mesaYaWs/internal/modules/realtime/infrastructure"
	transport "mesaYaWs/internal/modules/realtime/interface"
	"mesaYaWs/internal/platform/broker"
	"mesaYaWs/internal/shared/auth"
	"mesaYaWs/internal/shared/logging"
)

func main() {
	// Attempt to load variables from .env so local runs honour configuration tweaks.
	if err := godotenv.Overload(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, ".env load warning: %v\n", err)
		}
	}
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
	slog.Info("kafka env snapshot", slog.String("KAFKA_BROKERS", os.Getenv("KAFKA_BROKERS")), slog.String("KAFKA_BROKER", os.Getenv("KAFKA_BROKER")))
	slog.Info("kafka config resolved", slog.Any("brokers", cfg.Kafka.Brokers), slog.String("group", cfg.Kafka.GroupID))

	hub := infrastructure.NewHub()
	registry := infrastructure.NewHandlerRegistry()

	// Use cases
	broadcastUC := usecase.NewBroadcastUseCase(hub)

	// Echo server
	e := echo.New()
	e.Logger.SetOutput(log.Writer())

	// JWT validator used to validate tokens issued by the Nest auth service
	validator := auth.NewJWTValidator(cfg.Security.JWTSecret)
	snapshotFetcher := infrastructure.NewSectionSnapshotHTTPClient(cfg.REST.BaseURL, cfg.REST.Timeout, nil)
	analyticsFetcher := infrastructure.NewAnalyticsHTTPClient(cfg.REST.BaseURL, cfg.REST.Timeout, nil)
	connectUC := usecase.NewConnectSectionUseCase(validator, snapshotFetcher)
	analyticsUC := usecase.NewAnalyticsUseCase(validator, analyticsFetcher)

	// Registrar handlers de tópicos (cada feature)
	registry.Register(&handler.UserCreatedHandler{UseCase: broadcastUC})
	for entity, topics := range cfg.Kafka.Topics {
		for _, topic := range topics {
			registry.Register(handler.NewEntityStreamHandler(entity, topic, cfg.Websocket.AllowedActions, broadcastUC, connectUC))
		}
	}

	// Iniciar Kafka consumers (registrar topics desde config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// gather topics from config
	topics := make([]string, 0)
	for _, topicList := range cfg.Kafka.Topics {
		topics = append(topics, topicList...)
	}
	broker.StartKafkaConsumers(ctx, registry, cfg.Kafka.Brokers, cfg.Kafka.GroupID, topics)

	wsHandler := transport.NewWebsocketHandler(hub, connectUC, cfg.Websocket.DefaultEntity, cfg.Websocket.AllowedActions)
	notificationsHandler := transport.NewNotificationsWebsocketHandler(hub)
	analyticsHandler := transport.NewAnalyticsWebsocketHandler(hub, analyticsUC)
	// Generic entity routes: allow token in path or via query/header fallback
	e.GET("/ws/:entity/:section/:token", wsHandler)
	e.GET("/ws/:entity/:section", wsHandler)
	// Backwards compatible restaurant-specific routes
	e.GET("/ws/restaurant/:section/:token", wsHandler)
	e.GET("/ws/restaurant/:section", wsHandler)
	// Broadcast notifications stream
	e.GET("/ws/notifications", notificationsHandler)
	// Analytics websocket endpoints
	e.GET("/ws/analytics/:scope/:entity", analyticsHandler)

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

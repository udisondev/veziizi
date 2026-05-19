package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/heartbeat"
	"github.com/udisondev/veziizi/backend/internal/pkg/logging"
)

type Config struct {
	Name          string
	Topic         string
	ConsumerGroup string
	LogFile       string

	// Handler — legacy-стиль: один raw NoPublishHandlerFunc на топик, внутри
	// которого хендлер сам распаковывает EventEnvelope и делает type switch.
	// Используется воркерами, которые ещё не мигрировали на CQRS.
	Handler func(f *factory.Factory) message.NoPublishHandlerFunc

	// Setup — CQRS-стиль: фабрика регистрирует типизированные хендлеры на
	// EventGroupProcessor. Все хендлеры группы шарят один subscriber и общий
	// offset (поведенчески эквивалентно одному ConsumerGroup, как и у Handler).
	// Если задан Setup, поле Handler игнорируется.
	Setup func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error

	// Marshaler — формат сообщений. По умолчанию EventEnvelopeMarshaler (для
	// доменных событий из event store). Для топиков с JSON-командами
	// (notification.telegram/email) задай messaging.NotificationMarshaler().
	Marshaler cqrs.CommandEventMarshaler
}

func Run(cfg Config) {
	appCfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logFile, err := logging.Setup(appCfg.App.LogLevel, cfg.LogFile)
	if err != nil {
		slog.Error("failed to setup logger", "error", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer func() {
			if err := logFile.Close(); err != nil {
				slog.Error("failed to close log file", "error", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create factory - all dependencies are lazily initialized
	f := factory.New(appCfg)
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close factory", slog.String("error", err.Error()))
		}
	}()

	// Get pool for subscriber (triggers lazy initialization)
	pool := f.MustPool()
	slog.Info(fmt.Sprintf("%s worker connected to database", cfg.Name))

	// Heartbeat
	hb := heartbeat.New(pool, cfg.Name, "event", appCfg.Worker.HeartbeatInterval)
	if err := hb.Start(ctx); err != nil {
		slog.Error("failed to start heartbeat", "error", err)
	}
	defer hb.Stop()

	wmLogger := watermill.NewSlogLogger(slog.Default())

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		slog.Error("failed to create router", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Publisher для poison queue использует тот же default publisher, что и весь
	// EventPublisher: AutoInitializeSchema создаст таблицу watermill_messages_*
	// для DLQ-топика при первой публикации.
	var poisonPub message.Publisher
	if appCfg.Worker.DeadLetterTopic != "" {
		pub, err := f.Publisher()
		if err != nil {
			slog.Error("failed to get publisher for poison queue", slog.String("error", err.Error()))
			os.Exit(1)
		}
		poisonPub = pub.RawPublisher()
	}
	if err := applyStandardMiddleware(router, appCfg.Worker, wmLogger, poisonPub); err != nil {
		slog.Error("failed to apply middleware", slog.String("error", err.Error()))
		os.Exit(1)
	}

	switch {
	case cfg.Setup != nil:
		marshaler := cfg.Marshaler
		if marshaler == nil {
			marshaler = messaging.EventEnvelopeMarshaler{}
		}
		ep, err := messaging.NewEventGroupProcessor(router, pool, cfg.Topic, cfg.ConsumerGroup, marshaler, wmLogger)
		if err != nil {
			slog.Error("failed to create event group processor", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if err := cfg.Setup(f, ep); err != nil {
			slog.Error("worker setup failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case cfg.Handler != nil:
		subscriber, err := messaging.NewSQLSubscriber(pool, cfg.Topic, cfg.ConsumerGroup, wmLogger)
		if err != nil {
			slog.Error("failed to create subscriber", slog.String("error", err.Error()))
			os.Exit(1)
		}
		router.AddConsumerHandler(cfg.Name+"_handler", cfg.Topic, subscriber, cfg.Handler(f))
	default:
		slog.Error("worker config has neither Setup nor Handler")
		os.Exit(1)
	}

	go func() {
		if err := router.Run(ctx); err != nil {
			slog.Error("router error", slog.String("error", err.Error()))
			stop()
		}
	}()

	slog.Info(fmt.Sprintf("%s worker started", cfg.Name))

	<-ctx.Done()

	slog.Info(fmt.Sprintf("shutting down %s worker...", cfg.Name))

	// Graceful shutdown с таймаутом
	shutdownDone := make(chan struct{})
	go func() {
		if err := router.Close(); err != nil {
			slog.Error("failed to close router", slog.String("error", err.Error()))
		}
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		slog.Info(fmt.Sprintf("%s worker shutdown complete", cfg.Name))
	case <-time.After(appCfg.Worker.ShutdownTimeout):
		slog.Error(fmt.Sprintf("%s worker shutdown timed out, forcing exit", cfg.Name))
		os.Exit(1)
	}
}

// ScheduledConfig configures a scheduled (cron-like) worker
type ScheduledConfig struct {
	Name    string
	LogFile string

	// IntervalFunc returns interval from config. Called once at startup.
	IntervalFunc func(cfg *config.Config) time.Duration

	// Handler receives Factory and returns a function to execute on each tick
	Handler func(f *factory.Factory) func(ctx context.Context) error
}

// RunScheduled runs a scheduled worker that executes at regular intervals
func RunScheduled(cfg ScheduledConfig) {
	appCfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logFile, err := logging.Setup(appCfg.App.LogLevel, cfg.LogFile)
	if err != nil {
		slog.Error("failed to setup logger", "error", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer func() {
			if err := logFile.Close(); err != nil {
				slog.Error("failed to close log file", "error", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create factory - all dependencies are lazily initialized
	f := factory.New(appCfg)
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close factory", slog.String("error", err.Error()))
		}
	}()

	// Trigger pool initialization and log connection
	pool := f.MustPool()
	slog.Info(fmt.Sprintf("%s scheduled worker connected to database", cfg.Name))

	// Heartbeat
	hb := heartbeat.New(pool, cfg.Name, "scheduled", appCfg.Worker.HeartbeatInterval)
	if err := hb.Start(ctx); err != nil {
		slog.Error("failed to start heartbeat", "error", err)
	}
	defer hb.Stop()

	// Get handler
	handler := cfg.Handler(f)

	// Get interval from config
	interval := cfg.IntervalFunc(appCfg)

	// Start ticker
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info(fmt.Sprintf("%s scheduled worker started (interval: %s)", cfg.Name, interval))

	// Run immediately on start
	go func() {
		if err := handler(ctx); err != nil {
			slog.Error("handler error on startup", slog.String("error", err.Error()))
		}
	}()

	for {
		select {
		case <-ticker.C:
			if err := handler(ctx); err != nil {
				slog.Error("handler error", slog.String("error", err.Error()))
			}
		case <-ctx.Done():
			slog.Info(fmt.Sprintf("shutting down %s scheduled worker...", cfg.Name))
			return
		}
	}
}

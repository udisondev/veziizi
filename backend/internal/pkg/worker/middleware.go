package worker

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// applyStandardMiddleware регистрирует стандартный набор middleware на router'е.
//
// Watermill применяет middleware LIFO: первый добавленный — самый внешний по
// отношению к handler'у. Целевой порядок выполнения для исходящей ошибки:
//
//	handler error
//	  → Retry (retries N times)
//	  → PoisonQueue (если err остался — publish в DLQ, err = nil)
//	  → Recoverer (passthrough, panic→err уже сработал на входе)
//	  → CorrelationID (passthrough)
//	  → Ack
//
// Чтобы получить такой порядок, регистрируем от внешнего к внутреннему:
// CorrelationID → Recoverer → PoisonQueue → Retry.
//
// Если poisonPub == nil или DeadLetterTopic пуст, PoisonQueue не подключается —
// тогда после исчерпания Retry-попыток сообщение возвращается как nack
// и subscriber попытается доставить его снова. Это деградирует до поведения
// этапа 1.
func applyStandardMiddleware(
	r *message.Router,
	cfg config.WorkerConfig,
	logger watermill.LoggerAdapter,
	poisonPub message.Publisher,
) error {
	r.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
	)

	if poisonPub != nil && cfg.DeadLetterTopic != "" {
		pq, err := middleware.PoisonQueue(poisonPub, cfg.DeadLetterTopic)
		if err != nil {
			return fmt.Errorf("create poison queue middleware: %w", err)
		}
		r.AddMiddleware(pq)
	}

	r.AddMiddleware(middleware.Retry{
		MaxRetries:      cfg.RetryMaxRetries,
		InitialInterval: cfg.RetryInitialInterval,
		MaxInterval:     cfg.RetryMaxInterval,
		Multiplier:      cfg.RetryMultiplier,
		Logger:          logger,
	}.Middleware)

	return nil
}

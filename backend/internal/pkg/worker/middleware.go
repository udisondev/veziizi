package worker

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// applyStandardMiddleware регистрирует стандартный набор middleware на router'е
// в обязательном порядке:
//
//	CorrelationID — копирует correlation_id из входящего сообщения в исходящие,
//	               которые handler публикует. Цепочка событий остаётся прослеживаемой
//	               в логах через все асинхронные хопы.
//	Recoverer     — паника в handler'е превращается в ошибку → Nack → retry,
//	               сам процесс воркера не падает.
//	Retry         — exponential backoff между попытками. Без этого watermill-sql
//	               subscriber делает горячий FIFO retry, ядовитое сообщение
//	               крутится в цикле и зашумляет логи.
//
// PoisonQueue будет добавлен отдельным этапом после поднятия DLQ-инфраструктуры —
// порядок будет CorrelationID → Recoverer → PoisonQueue → Retry, чтобы ядовитое
// сообщение после исчерпания попыток уезжало в DLQ-топик с финальным Ack.
func applyStandardMiddleware(r *message.Router, cfg config.WorkerConfig, logger watermill.LoggerAdapter) {
	r.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
		middleware.Retry{
			MaxRetries:      cfg.RetryMaxRetries,
			InitialInterval: cfg.RetryInitialInterval,
			MaxInterval:     cfg.RetryMaxInterval,
			Multiplier:      cfg.RetryMultiplier,
			Logger:          logger,
		}.Middleware,
	)
}

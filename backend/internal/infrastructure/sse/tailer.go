package sse

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

const (
	defaultTailBlock = 5 * time.Second
	tailBatch        = 64
	tailMinRetry     = time.Second
	tailMaxRetry     = 30 * time.Second
)

// Tailer хвостом читает Redis-стримы доменных событий: XREAD BLOCK со
// стартового ID, разрешённого один раз через XINFO (НЕ "$" — см.
// resolveStartID), БЕЗ consumer group — каждый инстанс API видит весь поток
// (broadcast, не балансировка), а ack/pending-механика группы здесь не нужна:
// пропущенные события клиент компенсирует refetch'ем при переподключении.
//
// ВАЖНО: клиент должен быть выделен tailer'у, а не разделяться с
// watermill-redisstream подписчиками — их Subscriber.Close() закрывает клиент
// целиком, и ConnPool.Close дедлочится на соединении, занятом нашим блокирующим
// XREAD (mutex пула + fd-lock).
type Tailer struct {
	client  redis.UniversalClient
	router  *Router
	streams []string
	block   time.Duration
}

// NewTailer создает tailer для перечисленных стримов. block — таймаут XREAD
// BLOCK (обычно cfg.Redis.BlockTime): он же — верхняя граница задержки выхода
// горутины стрима при остановке.
func NewTailer(client redis.UniversalClient, router *Router, streams []string, block time.Duration) *Tailer {
	if block <= 0 {
		block = defaultTailBlock
	}
	return &Tailer{client: client, router: router, streams: streams, block: block}
}

// Run блокируется до отмены ctx: по горутине на стрим.
func (t *Tailer) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, stream := range t.streams {
		wg.Go(func() {
			t.tail(ctx, stream)
		})
	}
	wg.Wait()
}

func (t *Tailer) tail(ctx context.Context, stream string) {
	// Стартуем с последнего ID стрима (только новые события — реплей хвоста не
	// нужен, актуальное состояние клиенты загружают обычными GET). Резолвим ID
	// через XINFO один раз вместо XREAD с "$": "$" вычисляется заново на каждом
	// вызове, и события, попавшие в зазор между BLOCK-таймаутом и следующим
	// XREAD, были бы потеряны.
	lastID := ""
	retry := tailMinRetry

	for {
		if ctx.Err() != nil {
			return
		}

		if lastID == "" {
			id, err := t.resolveStartID(ctx, stream)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Error("sse: resolve stream start id failed",
					slog.String("stream", stream),
					slog.String("error", err.Error()))
				select {
				case <-ctx.Done():
					return
				case <-time.After(retry):
				}
				retry = min(retry*2, tailMaxRetry)
				continue
			}
			lastID = id
			retry = tailMinRetry
		}

		res, err := t.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{stream, lastID},
			Count:   tailBatch,
			Block:   t.block,
		}).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, redis.Nil) { // BLOCK-таймаут, событий нет
				continue
			}
			slog.Error("sse: xread failed",
				slog.String("stream", stream),
				slog.String("error", err.Error()))
			select {
			case <-ctx.Done():
				return
			case <-time.After(retry):
			}
			retry = min(retry*2, tailMaxRetry)
			continue
		}
		retry = tailMinRetry

		for _, str := range res {
			for _, msg := range str.Messages {
				lastID = msg.ID
				t.handle(ctx, stream, msg)
			}
		}
	}
}

// resolveStartID возвращает последний ID стрима; для ещё не созданного стрима
// — "0": когда forwarder создаст стрим первым XADD, мы прочитаем его с начала
// (история «несозданного» стрима и есть новые события).
func (t *Tailer) resolveStartID(ctx context.Context, stream string) (string, error) {
	info, err := t.client.XInfoStream(ctx, stream).Result()
	if err != nil {
		if strings.Contains(err.Error(), "no such key") {
			return "0", nil
		}
		return "", err
	}
	return info.LastGeneratedID, nil
}

// handle разбирает запись стрима тем же unmarshaller'ом, что и
// watermill-подписчики (wire-формат один — расходиться с ними нельзя), и
// извлечённый EventEnvelope отдаёт роутеру.
func (t *Tailer) handle(ctx context.Context, stream string, msg redis.XMessage) {
	wmMsg, err := redisstream.DefaultMarshallerUnmarshaller{}.Unmarshal(msg.Values)
	if err != nil {
		slog.Warn("sse: unmarshal stream message failed",
			slog.String("stream", stream),
			slog.String("id", msg.ID),
			slog.String("error", err.Error()))
		return
	}

	var env eventstore.EventEnvelope
	if err := json.Unmarshal(wmMsg.Payload, &env); err != nil {
		slog.Warn("sse: unmarshal envelope failed",
			slog.String("stream", stream),
			slog.String("id", msg.ID),
			slog.String("error", err.Error()))
		return
	}

	t.router.Route(ctx, env)
}

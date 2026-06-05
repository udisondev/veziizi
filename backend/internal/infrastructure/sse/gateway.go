package sse

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// Gateway собирает SSE-шлюз целиком: hub + router + tailer и владеет их
// жизненным циклом. Единственный конструктор для cmd/api/main.go и e2e suite —
// wiring не дублируется, prod и тесты не разъезжаются.
//
// Gateway создаёт СОБСТВЕННЫЙ redis-клиент: делить клиент с
// watermill-подписчиками нельзя — их Subscriber.Close() закрывает клиент
// целиком и дедлочится на блокирующем XREAD tailer'а (mutex пула + fd-lock).
type Gateway struct {
	hub    *Hub
	tailer *Tailer
	client redis.UniversalClient

	cancel context.CancelFunc
	done   chan struct{}
}

// NewGateway создает шлюз поверх готового hub'а (hub живёт в factory — его же
// использует HTTP-хендлер). freight/support — проекции для роутинга получателей.
func NewGateway(
	cfg *config.Config,
	hub *Hub,
	freight freightLookup,
	support supportLookup,
) (*Gateway, error) {
	client, err := messaging.NewRedisClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("sse: create redis client: %w", err)
	}

	router := NewRouter(hub, freight, support)
	tailer := NewTailer(client, router, []string{
		messaging.TopicNotificationEvents,
		messaging.TopicFreightRequestEvents,
		messaging.TopicSupportEvents,
	}, cfg.SSE.TailBlock)

	return &Gateway{hub: hub, tailer: tailer, client: client}, nil
}

// Start запускает tailer в фоне. ctx ограничивает время жизни шлюза сверху;
// штатная остановка — Stop().
func (g *Gateway) Start(ctx context.Context) {
	tailerCtx, cancel := context.WithCancel(ctx)
	g.cancel = cancel
	g.done = make(chan struct{})
	go func() {
		defer close(g.done)
		g.tailer.Run(tailerCtx)
	}()
}

// Stop останавливает шлюз: закрывает клиентские стримы (иначе server.Shutdown
// виснет на живых SSE-соединениях), дожидается выхода tailer'а и только потом
// закрывает redis-клиент (Close под блокирующим XREAD = deadlock).
func (g *Gateway) Stop() {
	g.hub.Shutdown()
	if g.cancel != nil {
		g.cancel()
		<-g.done
	}
	_ = g.client.Close()
}

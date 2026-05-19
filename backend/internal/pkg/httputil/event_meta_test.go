package httputil_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
)

func TestEventMeta_ToMap_IncludesCorrelationID(t *testing.T) {
	meta := httputil.EventMeta{
		CorrelationID: "req-abc-123",
	}

	m := meta.ToMap()
	require.Equal(t, "req-abc-123", m["correlation_id"])
}

func TestEventMeta_ToMap_OmitsEmpty(t *testing.T) {
	meta := httputil.EventMeta{} // всё пусто
	require.Nil(t, meta.ToMap(),
		"пустая EventMeta должна вернуть nil, чтобы не засорять envelope.Metadata")
}

func TestWithEventMeta_RoundTrip(t *testing.T) {
	original := httputil.EventMeta{
		MemberID:      uuid.New(),
		CorrelationID: "abc",
	}

	ctx := httputil.WithEventMeta(context.Background(), original)
	got, ok := httputil.EventMetaFromCtx(ctx)

	require.True(t, ok)
	require.Equal(t, original, got)
}

// TestEventMetaFromRequest_PicksChiRequestID — chi.RequestID middleware
// ставит X-Request-ID в response и кладёт в ctx; EventMetaFromRequest должен
// его поднять и использовать как CorrelationID. Это та единственная связка,
// благодаря которой HTTP-запрос становится прослеживаем через async pipeline.
func TestEventMetaFromRequest_PicksChiRequestID(t *testing.T) {
	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)

	var captured httputil.EventMeta
	r.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
		captured = httputil.EventMetaFromRequest(req, uuid.Nil, uuid.Nil)
	})

	srv := httptest.NewServer(r)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/ping", nil)
	require.NoError(t, err)
	req.Header.Set("X-Request-Id", "supplied-by-client")

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()

	require.NotEmpty(t, captured.CorrelationID,
		"chi RequestID должен дать CorrelationID")
	require.Equal(t, "supplied-by-client", captured.CorrelationID,
		"если клиент прислал X-Request-Id, chi его сохраняет (а не генерит новый)")
}

func TestEventMetaFromRequest_GeneratesIDWhenMissing(t *testing.T) {
	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)

	var captured httputil.EventMeta
	r.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
		captured = httputil.EventMetaFromRequest(req, uuid.Nil, uuid.Nil)
	})

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/ping")
	require.NoError(t, err)
	_ = resp.Body.Close()

	require.NotEmpty(t, captured.CorrelationID,
		"chi должен сгенерить request_id если клиент не прислал свой")
}

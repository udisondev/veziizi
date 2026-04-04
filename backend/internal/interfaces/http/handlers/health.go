package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	healthStatusPass = "pass"
	healthStatusWarn = "warn"
	healthStatusFail = "fail"

	workerStaleThreshold = 2 * time.Minute
)

// HealthHandler обслуживает /livez, /readyz, /healthz эндпоинты.
type HealthHandler struct {
	pool      *pgxpool.Pool
	startedAt time.Time
}

// NewHealthHandler создаёт health handler.
func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{
		pool:      pool,
		startedAt: time.Now(),
	}
}

// RegisterRoutes регистрирует health эндпоинты на отдельном роутере (без middleware).
func (h *HealthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/livez", h.Livez)
	r.Get("/readyz", h.Readyz)
	r.Get("/healthz", h.Healthz)
}

// Livez — liveness probe. Процесс жив, никаких зависимостей не проверяет.
func (h *HealthHandler) Livez(w http.ResponseWriter, _ *http.Request) {
	writeHealthJSON(w, http.StatusOK, map[string]any{
		"status": healthStatusPass,
	})
}

// Readyz — readiness probe. Проверяет критические зависимости (PostgreSQL).
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	pgStatus := healthStatusPass
	pgErr := ""

	if err := h.pool.Ping(ctx); err != nil {
		pgStatus = healthStatusFail
		pgErr = err.Error()
	}

	status := pgStatus
	code := http.StatusOK
	if status == healthStatusFail {
		code = http.StatusServiceUnavailable
	}

	resp := map[string]any{
		"status": status,
		"checks": map[string]any{
			"postgres": map[string]any{
				"status": pgStatus,
			},
		},
	}
	if pgErr != "" {
		resp["checks"].(map[string]any)["postgres"].(map[string]any)["error"] = pgErr
	}

	writeHealthJSON(w, code, resp)
}

// workerHeartbeat — строка из таблицы worker_heartbeats.
type workerHeartbeat struct {
	Name          string    `json:"name"`
	WorkerType    string    `json:"worker_type"`
	StartedAt     time.Time `json:"started_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	Status        string    `json:"status"`
}

// Healthz — полный отчёт для мониторинга. Postgres + pool stats + workers.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	overallStatus := healthStatusPass

	// Postgres ping
	pgStatus := healthStatusPass
	pgErr := ""
	if err := h.pool.Ping(ctx); err != nil {
		pgStatus = healthStatusFail
		pgErr = err.Error()
		overallStatus = healthStatusFail
	}

	// Pool stats
	stat := h.pool.Stat()
	poolCheck := map[string]any{
		"status":           healthStatusPass,
		"total_conns":      stat.TotalConns(),
		"idle_conns":       stat.IdleConns(),
		"acquired_conns":   stat.AcquiredConns(),
		"max_conns":        stat.MaxConns(),
		"constructing":     stat.ConstructingConns(),
		"empty_acquire":    stat.EmptyAcquireCount(),
		"canceled_acquire": stat.CanceledAcquireCount(),
	}
	if stat.TotalConns() == 0 {
		poolCheck["status"] = healthStatusFail
		overallStatus = healthStatusFail
	}

	// Workers heartbeats
	workers, workersStatus := h.checkWorkers(ctx)
	if workersStatus == healthStatusFail {
		if overallStatus != healthStatusFail {
			overallStatus = healthStatusWarn
		}
	} else if workersStatus == healthStatusWarn && overallStatus == healthStatusPass {
		overallStatus = healthStatusWarn
	}

	pgCheck := map[string]any{
		"status": pgStatus,
	}
	if pgErr != "" {
		pgCheck["error"] = pgErr
	}

	code := http.StatusOK
	if overallStatus == healthStatusFail {
		code = http.StatusServiceUnavailable
	}

	writeHealthJSON(w, code, map[string]any{
		"status":      overallStatus,
		"uptime":      time.Since(h.startedAt).String(),
		"description": "Veziizi API Server",
		"checks": map[string]any{
			"postgres": pgCheck,
			"pool":     poolCheck,
			"workers":  workers,
		},
	})
}

func (h *HealthHandler) checkWorkers(ctx context.Context) (map[string]any, string) {
	rows, err := h.pool.Query(ctx,
		`SELECT name, worker_type, started_at, last_heartbeat, status FROM worker_heartbeats ORDER BY name`,
	)
	if err != nil {
		slog.Error("failed to query worker heartbeats", "error", err)
		return map[string]any{"status": healthStatusFail, "error": err.Error()}, healthStatusFail
	}
	defer rows.Close()

	now := time.Now()
	workersStatus := healthStatusPass
	workerList := make([]map[string]any, 0)

	for rows.Next() {
		var hb workerHeartbeat
		if err := rows.Scan(&hb.Name, &hb.WorkerType, &hb.StartedAt, &hb.LastHeartbeat, &hb.Status); err != nil {
			slog.Error("failed to scan worker heartbeat", "error", err)
			continue
		}

		wStatus := healthStatusPass
		if hb.Status == "stopped" {
			wStatus = healthStatusWarn
			if workersStatus == healthStatusPass {
				workersStatus = healthStatusWarn
			}
		} else if now.Sub(hb.LastHeartbeat) > workerStaleThreshold {
			wStatus = healthStatusFail
			workersStatus = healthStatusFail
		}

		workerList = append(workerList, map[string]any{
			"name":           hb.Name,
			"type":           hb.WorkerType,
			"status":         wStatus,
			"last_heartbeat": hb.LastHeartbeat.Format(time.RFC3339),
			"started_at":     hb.StartedAt.Format(time.RFC3339),
			"uptime":         now.Sub(hb.StartedAt).String(),
		})
	}

	if err := rows.Err(); err != nil {
		slog.Error("rows iteration error", "error", err)
	}

	return map[string]any{
		"status":  workersStatus,
		"workers": workerList,
	}, workersStatus
}

func writeHealthJSON(w http.ResponseWriter, code int, data map[string]any) {
	w.Header().Set("Content-Type", "application/health+json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode health response", "error", err)
	}
}

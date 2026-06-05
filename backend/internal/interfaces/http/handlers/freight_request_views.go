package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// trackView synchronously UPSERTs into freight_request_views. Errors are logged
// and swallowed — view tracking must never break the user-facing GET. The
// previous fire-and-forget goroutine spawned an unbounded number of goroutines
// per request and detached from the request context.
func (h *FreightRequestHandler) trackView(ctx context.Context, memberID, freightID uuid.UUID) {
	if h.viewsProjection == nil {
		return
	}
	if err := h.viewsProjection.Touch(ctx, memberID, freightID, time.Now().UTC()); err != nil {
		slog.Warn("track view failed",
			slog.String("member_id", memberID.String()),
			slog.String("freight_request_id", freightID.String()),
			slog.String("error", err.Error()))
	}
}

type viewedResponseItem struct {
	FreightRequestID uuid.UUID `json:"freight_request_id"`
	FirstViewedAt    time.Time `json:"first_viewed_at"`
	LastViewedAt     time.Time `json:"last_viewed_at"`
	ViewCount        int       `json:"view_count"`
}

type viewedResponse struct {
	Items      []viewedResponseItem `json:"items"`
	NextCursor *string              `json:"next_cursor,omitempty"`
	HasMore    bool                 `json:"has_more"`
}

func (h *FreightRequestHandler) ListViewed(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	var cursor *projections.ViewsCursor
	if cursorStr := r.URL.Query().Get("cursor"); cursorStr != "" {
		decoded, err := base64.URLEncoding.DecodeString(cursorStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
		var c projections.ViewsCursor
		if err := json.Unmarshal(decoded, &c); err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
		cursor = &c
	}

	items, err := h.viewsProjection.ListViewedByMember(r.Context(), memberID, cursor, limit+1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list viewed")
		return
	}

	resp := viewedResponse{Items: make([]viewedResponseItem, 0, len(items))}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	for _, it := range items {
		resp.Items = append(resp.Items, viewedResponseItem{
			FreightRequestID: it.FreightRequestID,
			FirstViewedAt:    it.FirstViewedAt,
			LastViewedAt:     it.LastViewedAt,
			ViewCount:        it.ViewCount,
		})
	}
	resp.HasMore = hasMore
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		raw, _ := json.Marshal(projections.ViewsCursor{LastViewedAt: last.LastViewedAt, FreightID: last.FreightRequestID})
		encoded := base64.URLEncoding.EncodeToString(raw)
		resp.NextCursor = &encoded
	}
	writeJSON(w, http.StatusOK, resp)
}

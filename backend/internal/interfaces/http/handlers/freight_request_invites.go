package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/application/freightrequest"
	freightrequestDomain "github.com/udisondev/veziizi/backend/internal/domain/freightrequest"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
)

type InviteCarrierRequest struct {
	VehicleID uuid.UUID `json:"vehicle_id"`
}

func (h *FreightRequestHandler) InviteCarrier(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	freightID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid freight request id")
		return
	}
	var req InviteCarrierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.VehicleID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "vehicle_id is required")
		return
	}

	err = h.service.InviteCarrier(r.Context(), freightrequest.InviteCarrierInput{
		FreightRequestID: freightID,
		ActorOrgID:       orgID,
		ActorMemberID:    memberID,
		VehicleID:        req.VehicleID,
	})
	if err != nil {
		writeInviteCarrierError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type InviteResponse struct {
	ID           uuid.UUID `json:"id"`
	CarrierOrgID uuid.UUID `json:"carrier_org_id"`
	VehicleID    uuid.UUID `json:"vehicle_id"`
	InvitedBy    uuid.UUID `json:"invited_by"`
	InvitedAt    time.Time `json:"invited_at"`
}

// ListInvites returns who the current customer organization has already pinged
// for this freight request. Available to members of the customer organization.
func (h *FreightRequestHandler) ListInvites(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetMemberID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	freightID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid freight request id")
		return
	}

	fr, err := h.service.Get(r.Context(), freightID)
	if err != nil {
		if errors.Is(err, freightrequestDomain.ErrFreightRequestNotFound) {
			writeError(w, http.StatusNotFound, "freight request not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load freight request")
		return
	}
	if fr.CustomerOrgID() != orgID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	items, err := h.invitesProjection.ListByFreight(r.Context(), freightID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list invites")
		return
	}
	resp := make([]InviteResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, InviteResponse{
			ID:           it.ID,
			CarrierOrgID: it.CarrierOrgID,
			VehicleID:    it.VehicleID,
			InvitedBy:    it.InvitedBy,
			InvitedAt:    it.InvitedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp})
}

// CarrierInvitationsResponse — ответ списка входящих инвайтов с cursor-based пагинацией.
type CarrierInvitationsResponse struct {
	Items      []projections.CarrierInviteItem `json:"items"`
	NextCursor *string                         `json:"next_cursor,omitempty"`
	HasMore    bool                            `json:"has_more"`
}

// ListCarrierInvitations returns incoming invitations for the current carrier organization.
func (h *FreightRequestHandler) ListCarrierInvitations(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetMemberID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query()
	limit := 20
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	var cursor *projections.InviteCursor
	if cursorStr := q.Get("cursor"); cursorStr != "" {
		decoded, err := httputil.DecodeCursor[projections.InviteCursor](cursorStr)
		if err != nil {
			slog.Warn("invalid carrier invitations cursor", slog.String("error", err.Error()))
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
		cursor = decoded
	}

	items, err := h.invitesProjection.ListByCarrierOrg(r.Context(), orgID, cursor, limit+1)
	if err != nil {
		slog.Error("list carrier invitations failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list invitations")
		return
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var nextCursor *string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		encoded, err := httputil.EncodeCursor(projections.InviteCursor{
			InvitedAt: last.InvitedAt,
			ID:        last.ID,
		})
		if err != nil {
			slog.Error("failed to encode invite cursor", slog.String("error", err.Error()))
			hasMore = false
		} else {
			nextCursor = &encoded
		}
	}

	writeJSON(w, http.StatusOK, CarrierInvitationsResponse{Items: items, NextCursor: nextCursor, HasMore: hasMore})
}

func writeInviteCarrierError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, freightrequestDomain.ErrFreightRequestNotFound):
		writeError(w, http.StatusNotFound, "freight request not found")
	case errors.Is(err, freightrequestDomain.ErrFreightRequestExpired):
		writeError(w, http.StatusConflict, "freight request has expired")
	case errors.Is(err, freightrequestDomain.ErrFreightRequestNotPublished):
		writeError(w, http.StatusConflict, "freight request is not published")
	case errors.Is(err, freightrequestDomain.ErrCannotInviteOwnOrg):
		writeError(w, http.StatusBadRequest, "cannot invite own organization")
	case errors.Is(err, freightrequestDomain.ErrCarrierAlreadyInvited):
		writeError(w, http.StatusConflict, "carrier has already been invited")
	case errors.Is(err, freightrequestDomain.ErrInvitationLimitReached):
		writeError(w, http.StatusUnprocessableEntity, "invitation limit reached")
	case errors.Is(err, freightrequestDomain.ErrVehicleNotEligible):
		writeError(w, http.StatusUnprocessableEntity, "vehicle is not verified")
	case errors.Is(err, freightrequestDomain.ErrNotFreightRequestOwner):
		writeError(w, http.StatusForbidden, "not freight request owner")
	default:
		// Don't leak internal error details (wrapped DB messages, UUIDs).
		// Same pattern as 10fc397.
		slog.Warn("invite carrier failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to invite carrier")
	}
}

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/application/organization"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
)

type AdminVehicleHandler struct {
	orgService      *organization.Service
	pendingVehicles *projections.PendingVehiclesProjection
	vehicles        *projections.VehiclesProjection
	session         *session.AdminManager
}

func NewAdminVehicleHandler(
	orgService *organization.Service,
	pendingVehicles *projections.PendingVehiclesProjection,
	vehicles *projections.VehiclesProjection,
	session *session.AdminManager,
) *AdminVehicleHandler {
	return &AdminVehicleHandler{
		orgService:      orgService,
		pendingVehicles: pendingVehicles,
		vehicles:        vehicles,
		session:         session,
	}
}

// RegisterRoutes is mounted under /api/v1/admin.
func (h *AdminVehicleHandler) RegisterRoutes(r chi.Router) {
	r.Get("/vehicles/pending", h.ListPending)
	r.Post("/organizations/{orgId}/vehicles/{vid}/verify", h.Verify)
	r.Post("/organizations/{orgId}/vehicles/{vid}/reject", h.Reject)
}

type adminPendingVehicleResponse struct {
	ID                 uuid.UUID `json:"id"`
	OrgID              uuid.UUID `json:"org_id"`
	RegistrationNumber string    `json:"registration_number"`
	Brand              *string   `json:"brand,omitempty"`
	Model              *string   `json:"model,omitempty"`
	VehicleType        string    `json:"vehicle_type"`
	VehicleSubType     string    `json:"vehicle_subtype"`
	SubmittedAt        string    `json:"submitted_at"`
}

func (h *AdminVehicleHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.pendingVehicles.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pending vehicles")
		return
	}
	resp := make([]adminPendingVehicleResponse, 0, len(items))
	for _, v := range items {
		resp = append(resp, adminPendingVehicleResponse{
			ID:                 v.ID,
			OrgID:              v.OrgID,
			RegistrationNumber: v.RegistrationNumber,
			Brand:              v.Brand,
			Model:              v.Model,
			VehicleType:        v.VehicleType,
			VehicleSubType:     v.VehicleSubType,
			SubmittedAt:        v.SubmittedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp})
}

func (h *AdminVehicleHandler) Verify(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	vid, err := uuid.Parse(chi.URLParam(r, "vid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}
	if err := h.orgService.VerifyVehicle(r.Context(), organization.VerifyVehicleInput{
		OrganizationID: orgID,
		VehicleID:      vid,
		AdminID:        adminID,
	}); err != nil {
		writeVehicleDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type rejectVehicleRequest struct {
	Reason string `json:"reason"`
}

func (h *AdminVehicleHandler) Reject(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	vid, err := uuid.Parse(chi.URLParam(r, "vid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}
	var req rejectVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Reason == "" {
		writeError(w, http.StatusBadRequest, "reason is required")
		return
	}
	if err := h.orgService.RejectVehicle(r.Context(), organization.RejectVehicleInput{
		OrganizationID: orgID,
		VehicleID:      vid,
		AdminID:        adminID,
		Reason:         req.Reason,
	}); err != nil {
		writeVehicleDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

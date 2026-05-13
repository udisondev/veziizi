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

	"github.com/udisondev/veziizi/backend/internal/application/organization"
	orgDomain "github.com/udisondev/veziizi/backend/internal/domain/organization"
	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/domain/transport"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
)

type VehicleHandler struct {
	service         *organization.Service
	projection      *projections.VehiclesProjection
	pendingVehicles *projections.PendingVehiclesProjection
	session         *session.Manager
}

func NewVehicleHandler(
	service *organization.Service,
	projection *projections.VehiclesProjection,
	pendingVehicles *projections.PendingVehiclesProjection,
	session *session.Manager,
) *VehicleHandler {
	return &VehicleHandler{
		service:         service,
		projection:      projection,
		pendingVehicles: pendingVehicles,
		session:         session,
	}
}

func (h *VehicleHandler) RegisterRoutes(r chi.Router) {
	// Mine
	r.Post("/api/v1/organizations/{id}/vehicles", h.Add)
	r.Patch("/api/v1/organizations/{id}/vehicles/{vid}", h.Update)
	r.Delete("/api/v1/organizations/{id}/vehicles/{vid}", h.Archive)
	r.Get("/api/v1/organizations/{id}/vehicles", h.ListByOrganization)

	// Public marketplace
	r.Get("/api/v1/vehicles", h.List)
	r.Get("/api/v1/vehicles/{vid}", h.Get)
}

// VehicleRequest is the JSON payload accepted on add/update.
type VehicleRequest struct {
	RegistrationNumber string                  `json:"registration_number"`
	Brand              string                  `json:"brand"`
	Model              string                  `json:"model"`
	VehicleType        transport.VehicleType   `json:"vehicle_type"`
	VehicleSubType     transport.VehicleSubType `json:"vehicle_subtype"`
	LoadingTypes       []transport.LoadingType `json:"loading_types,omitempty"`
	Capacity           float64                 `json:"capacity,omitempty"`
	Volume             float64                 `json:"volume,omitempty"`
	Length             float64                 `json:"length,omitempty"`
	Width              float64                 `json:"width,omitempty"`
	Height             float64                 `json:"height,omitempty"`
	RequiresADR        bool                    `json:"requires_adr,omitempty"`
	Temperature        *transport.Temperature  `json:"temperature,omitempty"`
	Thermograph        bool                    `json:"thermograph,omitempty"`
}

func (req VehicleRequest) toSpecs() organization.VehicleSpecsInput {
	return organization.VehicleSpecsInput{
		RegistrationNumber: req.RegistrationNumber,
		Brand:              req.Brand,
		Model:              req.Model,
		VehicleType:        req.VehicleType,
		VehicleSubType:     req.VehicleSubType,
		LoadingTypes:       req.LoadingTypes,
		Capacity:           req.Capacity,
		Volume:             req.Volume,
		Length:             req.Length,
		Width:              req.Width,
		Height:             req.Height,
		RequiresADR:        req.RequiresADR,
		Temperature:        req.Temperature,
		Thermograph:        req.Thermograph,
	}
}

type VehicleResponse struct {
	ID                 uuid.UUID              `json:"id"`
	OrgID              uuid.UUID              `json:"org_id"`
	RegistrationNumber string                 `json:"registration_number"`
	Brand              *string                `json:"brand,omitempty"`
	Model              *string                `json:"model,omitempty"`
	VehicleType        string                 `json:"vehicle_type"`
	VehicleSubType     string                 `json:"vehicle_subtype"`
	LoadingTypes       []string               `json:"loading_types,omitempty"`
	Capacity           *float64               `json:"capacity,omitempty"`
	Volume             *float64               `json:"volume,omitempty"`
	Length             *float64               `json:"length,omitempty"`
	Width              *float64               `json:"width,omitempty"`
	Height             *float64               `json:"height,omitempty"`
	RequiresADR        bool                   `json:"requires_adr"`
	Temperature        *transport.Temperature `json:"temperature,omitempty"`
	Thermograph        bool                   `json:"thermograph"`
	Status             string                 `json:"status"`
	RejectionReason    *string                `json:"rejection_reason,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

func mapVehicleToResponse(v projections.VehicleListItem) VehicleResponse {
	resp := VehicleResponse{
		ID:                 v.ID,
		OrgID:              v.OrgID,
		RegistrationNumber: v.RegistrationNumber,
		Brand:              v.Brand,
		Model:              v.Model,
		VehicleType:        v.VehicleType,
		VehicleSubType:     v.VehicleSubType,
		LoadingTypes:       v.LoadingTypes,
		Capacity:           v.Capacity,
		Volume:             v.Volume,
		Length:             v.Length,
		Width:              v.Width,
		Height:             v.Height,
		RequiresADR:        v.RequiresADR,
		Thermograph:        v.Thermograph,
		Status:             v.Status,
		RejectionReason:    v.RejectionReason,
		CreatedAt:          v.CreatedAt,
		UpdatedAt:          v.UpdatedAt,
	}
	if v.HasTemperature && v.TempMin != nil && v.TempMax != nil {
		resp.Temperature = &transport.Temperature{Min: *v.TempMin, Max: *v.TempMax}
	}
	return resp
}

func (h *VehicleHandler) Add(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	actorID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sessionOrg, ok := h.session.GetOrganizationID(r)
	if !ok || sessionOrg != orgID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var req VehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id, err := h.service.AddVehicle(r.Context(), organization.AddVehicleInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		Specs:          req.toSpecs(),
	})
	if err != nil {
		writeVehicleDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"vehicle_id": id.String()})
}

func (h *VehicleHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	vid, err := uuid.Parse(chi.URLParam(r, "vid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}
	actorID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sessionOrg, ok := h.session.GetOrganizationID(r)
	if !ok || sessionOrg != orgID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var req VehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateVehicle(r.Context(), organization.UpdateVehicleInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		VehicleID:      vid,
		Specs:          req.toSpecs(),
	}); err != nil {
		writeVehicleDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *VehicleHandler) Archive(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	vid, err := uuid.Parse(chi.URLParam(r, "vid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}
	actorID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sessionOrg, ok := h.session.GetOrganizationID(r)
	if !ok || sessionOrg != orgID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.service.ArchiveVehicle(r.Context(), organization.ArchiveVehicleInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		VehicleID:      vid,
	}); err != nil {
		writeVehicleDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListByOrganization returns vehicles of the given organization including pending/rejected
// when called by a member of that organization. Non-members see only verified.
func (h *VehicleHandler) ListByOrganization(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	isMember := false
	if sessionOrg, ok := h.session.GetOrganizationID(r); ok && sessionOrg == orgID {
		isMember = true
	}

	opts := []projections.VehicleFilterOption{projections.WithVehicleOrgID(orgID)}
	if isMember {
		opts = append(opts, projections.WithVehicleStatuses([]string{
			orgValues.VehicleStatusPending.String(),
			orgValues.VehicleStatusVerified.String(),
			orgValues.VehicleStatusRejected.String(),
		}))
	} else {
		opts = append(opts, projections.WithVehicleStatus(orgValues.VehicleStatusVerified.String()))
	}

	items, err := h.projection.List(r.Context(), opts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list vehicles")
		return
	}
	resp := make([]VehicleResponse, 0, len(items))
	for _, v := range items {
		resp = append(resp, mapVehicleToResponse(v))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp})
}

// List returns the platform-wide fleet (verified only by default).
func (h *VehicleHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetMemberID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query()
	opts := []projections.VehicleFilterOption{
		projections.WithVehicleStatus(orgValues.VehicleStatusVerified.String()),
	}
	if v := q.Get("vehicle_type"); v != "" {
		opts = append(opts, projections.WithFleetVehicleType(v))
	}
	if v := q.Get("vehicle_subtype"); v != "" {
		opts = append(opts, projections.WithFleetVehicleSubType(v))
	}
	if v := q.Get("min_capacity"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts = append(opts, projections.WithMinCapacity(f))
		}
	}
	if v := q.Get("min_volume"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts = append(opts, projections.WithMinFleetVolume(f))
		}
	}
	if v := q.Get("requires_adr"); v == "true" {
		opts = append(opts, projections.WithRequiresADR(true))
	}
	if v := q.Get("loading_type"); v != "" {
		opts = append(opts, projections.WithLoadingType(v))
	}
	if v := q.Get("org_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			opts = append(opts, projections.WithVehicleOrgID(id))
		}
	}
	limit := 50
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	opts = append(opts, projections.WithVehicleLimit(limit))

	items, err := h.projection.List(r.Context(), opts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list vehicles")
		return
	}
	resp := make([]VehicleResponse, 0, len(items))
	for _, v := range items {
		resp = append(resp, mapVehicleToResponse(v))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp})
}

func (h *VehicleHandler) Get(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetMemberID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	vid, err := uuid.Parse(chi.URLParam(r, "vid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid vehicle id")
		return
	}
	item, err := h.projection.GetByID(r.Context(), vid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load vehicle")
		return
	}
	if item == nil {
		writeError(w, http.StatusNotFound, "vehicle not found")
		return
	}

	// Non-verified vehicles (pending/rejected/archived) are visible only to
	// members of the owning organization. Other authenticated users see 404 —
	// otherwise we'd leak moderation status and rejection_reason.
	if item.Status != orgValues.VehicleStatusVerified.String() {
		sessionOrg, ok := h.session.GetOrganizationID(r)
		if !ok || sessionOrg != item.OrgID {
			writeError(w, http.StatusNotFound, "vehicle not found")
			return
		}
	}
	writeJSON(w, http.StatusOK, mapVehicleToResponse(*item))
}

func writeVehicleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, orgDomain.ErrOrganizationNotFound):
		writeError(w, http.StatusNotFound, "organization not found")
	case errors.Is(err, orgDomain.ErrOrganizationNotActive):
		writeError(w, http.StatusConflict, "organization is not active")
	case errors.Is(err, orgDomain.ErrMemberNotFound):
		writeError(w, http.StatusForbidden, "member not found in organization")
	case errors.Is(err, orgDomain.ErrInsufficientPermissions):
		writeError(w, http.StatusForbidden, "insufficient permissions")
	case errors.Is(err, orgDomain.ErrVehicleNotFound):
		writeError(w, http.StatusNotFound, "vehicle not found")
	case errors.Is(err, orgDomain.ErrVehicleAlreadyExists):
		writeError(w, http.StatusConflict, "vehicle with this registration number already exists")
	case errors.Is(err, orgDomain.ErrVehicleArchived):
		writeError(w, http.StatusConflict, "vehicle is archived")
	case errors.Is(err, orgDomain.ErrVehicleNotPending):
		writeError(w, http.StatusConflict, "vehicle is not pending moderation")
	default:
		// Validation errors from VehicleSpecsInput include user-supplied
		// values ("invalid vehicle_type: <raw>"). Keep the raw value in slog
		// for audit, return a generic message — same pattern as 10fc397.
		slog.Warn("vehicle request rejected", slog.String("error", err.Error()))
		writeError(w, http.StatusBadRequest, "invalid vehicle data")
	}
}

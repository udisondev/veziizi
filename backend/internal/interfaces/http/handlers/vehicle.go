package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/application/organization"
	orgDomain "github.com/udisondev/veziizi/backend/internal/domain/organization"
	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/domain/transport"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
)

// VehicleListResponse — ответ списка транспорта с cursor-based пагинацией.
type VehicleListResponse struct {
	Items      []VehicleResponse `json:"items"`
	NextCursor *string           `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

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
	r.Post("/api/v1/organizations/{id}/vehicles/{vid}/submit", h.Submit)
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
	OrgName            *string                `json:"org_name,omitempty"`
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
	// Verified — публичный флаг подтверждения (status == verified).
	Verified bool `json:"verified"`
	// Status и RejectionReason отдаются только членам владеющей организации;
	// в публичных ответах пустые (см. mapVehicleToPublicResponse).
	Status          string    `json:"status,omitempty"`
	RejectionReason *string   `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func mapVehicleToResponse(v projections.VehicleListItem) VehicleResponse {
	resp := VehicleResponse{
		ID:                 v.ID,
		OrgID:              v.OrgID,
		OrgName:            v.OrgName,
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
		Verified:           v.Status == orgValues.VehicleStatusVerified.String(),
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

// mapVehicleToPublicResponse — ответ для пользователей вне владеющей организации:
// статус модерации и причина отклонения не раскрываются, наружу уходит только
// флаг verified.
func mapVehicleToPublicResponse(v projections.VehicleListItem) VehicleResponse {
	resp := mapVehicleToResponse(v)
	resp.Status = ""
	resp.RejectionReason = nil
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

// Submit отправляет транспорт на модерацию по запросу владельца
// (unconfirmed|rejected → pending).
func (h *VehicleHandler) Submit(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.SubmitVehicle(r.Context(), organization.SubmitVehicleInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		VehicleID:      vid,
	}); err != nil {
		writeVehicleDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListByOrganization returns vehicles of the given organization including moderation
// statuses when called by a member of that organization. Non-members see only
// publicly visible vehicles (no archived/rejected) through the public DTO
// (no status/rejection_reason).
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
			orgValues.VehicleStatusUnconfirmed.String(),
			orgValues.VehicleStatusPending.String(),
			orgValues.VehicleStatusVerified.String(),
			orgValues.VehicleStatusRejected.String(),
		}))
	} else {
		opts = append(opts, projections.WithVehiclePubliclyVisible())
	}

	items, err := h.projection.List(r.Context(), opts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list vehicles")
		return
	}
	resp := make([]VehicleResponse, 0, len(items))
	for _, v := range items {
		if isMember {
			resp = append(resp, mapVehicleToResponse(v))
		} else {
			resp = append(resp, mapVehicleToPublicResponse(v))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": resp})
}

// List returns the platform-wide fleet: publicly visible vehicles (no
// archived/rejected) through the public DTO (moderation status is reduced to
// the verified flag).
func (h *VehicleHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetMemberID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query()
	opts := []projections.VehicleFilterOption{
		projections.WithVehiclePubliclyVisible(),
	}

	if v := q.Get("vehicle_type"); v != "" {
		opts = append(opts, projections.WithFleetVehicleType(v))
	}
	// vehicle_subtypes — запятая-разделённый список подтипов кузова.
	if v := q.Get("vehicle_subtypes"); v != "" {
		subs := strings.Split(v, ",")
		opts = append(opts, projections.WithFleetVehicleSubTypes(subs))
	}
	if v := q.Get("min_capacity"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts = append(opts, projections.WithMinCapacity(f))
		}
	}
	if v := q.Get("max_capacity"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts = append(opts, projections.WithMaxCapacity(f))
		}
	}
	if v := q.Get("min_volume"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts = append(opts, projections.WithMinFleetVolume(f))
		}
	}
	if v := q.Get("max_volume"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			opts = append(opts, projections.WithMaxFleetVolume(f))
		}
	}
	if q.Get("requires_adr") == "true" {
		opts = append(opts, projections.WithRequiresADR(true))
	}
	if q.Get("thermograph") == "true" {
		opts = append(opts, projections.WithThermograph(true))
	}
	// loading_types — запятая-разделённый список типов погрузки.
	if v := q.Get("loading_types"); v != "" {
		types := strings.Split(v, ",")
		opts = append(opts, projections.WithLoadingTypes(types))
	}
	if v := q.Get("org_name"); v != "" {
		opts = append(opts, projections.WithOrgName(v))
	}
	if v := q.Get("org_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			opts = append(opts, projections.WithVehicleOrgID(id))
		}
	}
	if v := q.Get("exclude_org_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			opts = append(opts, projections.WithExcludeOrgID(id))
		}
	}

	// Cursor-based pagination
	if cursorStr := q.Get("cursor"); cursorStr != "" {
		cursor, err := httputil.DecodeCursor[projections.VehicleCursor](cursorStr)
		if err != nil {
			slog.Warn("invalid vehicle cursor", slog.String("cursor", cursorStr), slog.String("error", err.Error()))
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
		opts = append(opts, projections.WithVehicleCursor(*cursor))
	}

	limit := 20
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	opts = append(opts, projections.WithVehicleLimit(limit+1))

	items, err := h.projection.List(r.Context(), opts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list vehicles")
		return
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var nextCursor *string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		encoded, err := httputil.EncodeCursor(projections.VehicleCursor{
			CreatedAt:          last.CreatedAt,
			RegistrationNumber: last.RegistrationNumber,
		})
		if err != nil {
			slog.Error("failed to encode vehicle cursor", slog.String("error", err.Error()))
			hasMore = false
		} else {
			nextCursor = &encoded
		}
	}

	resp := make([]VehicleResponse, 0, len(items))
	for _, v := range items {
		resp = append(resp, mapVehicleToPublicResponse(v))
	}
	writeJSON(w, http.StatusOK, VehicleListResponse{Items: resp, NextCursor: nextCursor, HasMore: hasMore})
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

	// Члены владеющей организации видят полный ответ (status, rejection_reason).
	// Остальным — публичный DTO без статуса модерации; archived и rejected для
	// них 404 (зеркало WithVehiclePubliclyVisible — иначе скрытая из списков
	// машина остаётся доступной по прямому ID).
	sessionOrg, isMember := h.session.GetOrganizationID(r)
	if isMember && sessionOrg == item.OrgID {
		writeJSON(w, http.StatusOK, mapVehicleToResponse(*item))
		return
	}
	if item.Status == orgValues.VehicleStatusArchived.String() ||
		item.Status == orgValues.VehicleStatusRejected.String() {
		writeError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	writeJSON(w, http.StatusOK, mapVehicleToPublicResponse(*item))
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
	case errors.Is(err, orgDomain.ErrVehicleNotSubmittable):
		writeError(w, http.StatusConflict, "vehicle cannot be submitted for verification")
	default:
		// Validation errors from VehicleSpecsInput include user-supplied
		// values ("invalid vehicle_type: <raw>"). Keep the raw value in slog
		// for audit, return a generic message — same pattern as 10fc397.
		slog.Warn("vehicle request rejected", slog.String("error", err.Error()))
		writeError(w, http.StatusBadRequest, "invalid vehicle data")
	}
}

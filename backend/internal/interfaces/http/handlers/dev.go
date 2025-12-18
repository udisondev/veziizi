package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"codeberg.org/udison/veziizi/backend/internal/application/organization"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type DevHandler struct {
	cfg        *config.Config
	members    *projections.MembersProjection
	orgService *organization.Service
	session    *session.Manager
}

func NewDevHandler(
	cfg *config.Config,
	members *projections.MembersProjection,
	orgService *organization.Service,
	session *session.Manager,
) *DevHandler {
	return &DevHandler{
		cfg:        cfg,
		members:    members,
		orgService: orgService,
		session:    session,
	}
}

func (h *DevHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/dev/status", h.Status).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/dev/users", h.ListUsers).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/dev/switch", h.SwitchUser).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/dev/users/{id}", h.DeleteUser).Methods(http.MethodDelete)
}

type DevUserResponse struct {
	ID                 string `json:"id"`
	OrganizationID     string `json:"organization_id"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Role               string `json:"role"`
	Status             string `json:"status"`
	OrganizationName   string `json:"organization_name"`
	OrganizationStatus string `json:"organization_status"`
}

type DevStatusResponse struct {
	Enabled bool `json:"enabled"`
}

func (h *DevHandler) Status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, DevStatusResponse{
		Enabled: h.cfg.IsDevelopment(),
	})
}

func (h *DevHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	members, err := h.members.ListAll(r.Context(), search, limit)
	if err != nil {
		slog.Error("failed to list members", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Collect unique org IDs
	orgIDs := make([]uuid.UUID, 0)
	seen := make(map[uuid.UUID]bool)
	for _, m := range members {
		if !seen[m.OrganizationID] {
			seen[m.OrganizationID] = true
			orgIDs = append(orgIDs, m.OrganizationID)
		}
	}

	// Get organization data
	type orgInfo struct {
		Name   string
		Status string
	}
	orgsData := make(map[uuid.UUID]orgInfo)
	for _, orgID := range orgIDs {
		org, err := h.orgService.Get(r.Context(), orgID)
		if err != nil {
			slog.Error("failed to get organization", slog.String("org_id", orgID.String()), slog.String("error", err.Error()))
			continue
		}
		orgsData[orgID] = orgInfo{
			Name:   org.Name(),
			Status: org.Status().String(),
		}
	}

	// Build response
	result := make([]DevUserResponse, 0, len(members))
	for _, m := range members {
		orgData := orgsData[m.OrganizationID]
		result = append(result, DevUserResponse{
			ID:                 m.ID.String(),
			OrganizationID:     m.OrganizationID.String(),
			Email:              m.Email,
			Name:               m.Name,
			Role:               m.Role,
			Status:             m.Status,
			OrganizationName:   orgData.Name,
			OrganizationStatus: orgData.Status,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

type SwitchUserRequest struct {
	MemberID string `json:"member_id"`
}

func (h *DevHandler) SwitchUser(w http.ResponseWriter, r *http.Request) {
	var req SwitchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	memberID, err := uuid.Parse(req.MemberID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member_id")
		return
	}

	member, err := h.members.GetByID(r.Context(), memberID)
	if err != nil {
		slog.Error("failed to get member", slog.String("member_id", req.MemberID), slog.String("error", err.Error()))
		writeError(w, http.StatusNotFound, "member not found")
		return
	}

	if err := h.session.SetAuth(r, w, member.ID, member.OrganizationID, member.Role); err != nil {
		slog.Error("failed to set session", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Get organization for response
	org, err := h.orgService.Get(r.Context(), member.OrganizationID)
	if err != nil {
		slog.Error("failed to get organization", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, DevUserResponse{
		ID:                 member.ID.String(),
		OrganizationID:     member.OrganizationID.String(),
		Email:              member.Email,
		Name:               member.Name,
		Role:               member.Role,
		Status:             member.Status,
		OrganizationName:   org.Name(),
		OrganizationStatus: org.Status().String(),
	})
}

func (h *DevHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memberID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member id")
		return
	}

	member, err := h.members.GetByID(r.Context(), memberID)
	if err != nil {
		slog.Error("failed to get member", slog.String("member_id", memberID.String()), slog.String("error", err.Error()))
		writeError(w, http.StatusNotFound, "member not found")
		return
	}

	if member.Role == "owner" {
		writeError(w, http.StatusForbidden, "cannot delete owner")
		return
	}

	if err := h.orgService.DevRemoveMember(r.Context(), member.OrganizationID, memberID); err != nil {
		slog.Error("failed to remove member", slog.String("member_id", memberID.String()), slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

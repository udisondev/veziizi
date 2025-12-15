package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"codeberg.org/udison/veziizi/backend/internal/application/admin"
	orgDomain "codeberg.org/udison/veziizi/backend/internal/domain/organization"
	adminRepo "codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/admin"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	service   *admin.Service
	adminRepo *adminRepo.Repository
	session   *session.AdminManager
}

func NewAdminHandler(service *admin.Service, adminRepo *adminRepo.Repository, session *session.AdminManager) *AdminHandler {
	return &AdminHandler{
		service:   service,
		adminRepo: adminRepo,
		session:   session,
	}
}

func (h *AdminHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/admin/auth/login", h.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/auth/logout", h.Logout).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/organizations", h.ListPending).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/admin/organizations/{id}", h.GetOrganization).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/admin/organizations/{id}/approve", h.Approve).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/organizations/{id}/reject", h.Reject).Methods(http.MethodPost)
}

type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AdminLoginResponse struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	adm, err := h.adminRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		slog.Error("failed to get admin", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if !adm.IsActive {
		writeError(w, http.StatusForbidden, "account is disabled")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(adm.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := h.session.SetAuth(r, w, adm.ID); err != nil {
		slog.Error("failed to set session", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, AdminLoginResponse{
		AdminID: adm.ID.String(),
		Email:   adm.Email,
		Name:    adm.Name,
	})
}

func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.session.Clear(r, w); err != nil {
		slog.Error("failed to clear session", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type PendingOrganizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	INN       string `json:"inn"`
	LegalName string `json:"legal_name"`
	Country   string `json:"country"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func (h *AdminHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_ = adminID

	orgs, err := h.service.ListPendingOrganizations(r.Context())
	if err != nil {
		slog.Error("failed to list pending organizations", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response := make([]PendingOrganizationResponse, 0, len(orgs))
	for _, org := range orgs {
		response = append(response, PendingOrganizationResponse{
			ID:        org.ID.String(),
			Name:      org.Name,
			INN:       org.INN,
			LegalName: org.LegalName,
			Country:   org.Country,
			Email:     org.Email,
			CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AdminHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_ = adminID

	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	org, err := h.service.GetOrganization(r.Context(), id)
	if err != nil {
		if errors.Is(err, orgDomain.ErrOrganizationNotFound) {
			writeError(w, http.StatusNotFound, "organization not found")
			return
		}
		slog.Error("failed to get organization", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, mapOrganizationToResponse(org))
}

func (h *AdminHandler) Approve(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	if err := h.service.Approve(r.Context(), admin.ApproveInput{
		AdminID:        adminID,
		OrganizationID: orgID,
	}); err != nil {
		handleAdminDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type RejectRequest struct {
	Reason string `json:"reason"`
}

func (h *AdminHandler) Reject(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	var req RejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Reject(r.Context(), admin.RejectInput{
		AdminID:        adminID,
		OrganizationID: orgID,
		Reason:         req.Reason,
	}); err != nil {
		handleAdminDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleAdminDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, orgDomain.ErrOrganizationNotFound):
		writeError(w, http.StatusNotFound, "organization not found")
	case errors.Is(err, orgDomain.ErrOrganizationNotPending):
		writeError(w, http.StatusBadRequest, "organization is not pending")
	default:
		slog.Error("unhandled domain error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

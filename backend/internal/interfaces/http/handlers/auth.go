package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/application/organization"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	members    *projections.MembersProjection
	orgService *organization.Service
	session    *session.Manager
}

func NewAuthHandler(
	members *projections.MembersProjection,
	orgService *organization.Service,
	session *session.Manager,
) *AuthHandler {
	return &AuthHandler{
		members:    members,
		orgService: orgService,
		session:    session,
	}
}

func (h *AuthHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/auth/login", h.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/auth/logout", h.Logout).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/auth/me", h.Me).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/members/{id}", h.GetMemberProfile).Methods(http.MethodGet)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	MemberID       string `json:"member_id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	Role           string `json:"role"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	member, err := h.members.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		slog.Error("failed to get member", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if member.Status != "active" {
		writeError(w, http.StatusForbidden, "account is blocked")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(member.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := h.session.SetAuth(r, w, member.ID, member.OrganizationID, member.Role); err != nil {
		slog.Error("failed to set session", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		MemberID:       member.ID.String(),
		OrganizationID: member.OrganizationID.String(),
		Email:          member.Email,
		Name:           member.Name,
		Role:           member.Role,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.session.Clear(r, w); err != nil {
		slog.Error("failed to clear session", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type MeResponse struct {
	MemberID       string               `json:"member_id"`
	OrganizationID string               `json:"organization_id"`
	Role           string               `json:"role"`
	Email          string               `json:"email"`
	Name           string               `json:"name"`
	Phone          *string              `json:"phone,omitempty"`
	TelegramID     *int64               `json:"telegram_id,omitempty"`
	Organization   *OrganizationBrief   `json:"organization,omitempty"`
}

type OrganizationBrief struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orgID, _ := h.session.GetOrganizationID(r)

	// Get member from projection
	member, err := h.members.GetByID(r.Context(), memberID)
	if err != nil {
		slog.Error("failed to get member", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Get organization from event store
	org, err := h.orgService.Get(r.Context(), orgID)
	if err != nil {
		slog.Error("failed to get organization", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, MeResponse{
		MemberID:       memberID.String(),
		OrganizationID: orgID.String(),
		Role:           member.Role,
		Email:          member.Email,
		Name:           member.Name,
		Phone:          member.Phone,
		TelegramID:     member.TelegramID,
		Organization: &OrganizationBrief{
			Name:   org.Name(),
			Status: org.Status().String(),
		},
	})
}

// MemberProfileResponse represents public member profile
type MemberProfileResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	Phone            *string   `json:"phone,omitempty"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	CreatedAt        time.Time `json:"created_at"`
}

func (h *AuthHandler) GetMemberProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	member, err := h.members.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		slog.Error("failed to get member", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Get organization name
	org, err := h.orgService.Get(r.Context(), member.OrganizationID)
	if err != nil {
		slog.Error("failed to get organization", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, MemberProfileResponse{
		ID:               member.ID.String(),
		Name:             member.Name,
		Email:            member.Email,
		Phone:            member.Phone,
		OrganizationID:   member.OrganizationID.String(),
		OrganizationName: org.Name(),
		CreatedAt:        member.CreatedAt,
	})
}

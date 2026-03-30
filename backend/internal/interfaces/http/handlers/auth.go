package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/udisondev/veziizi/backend/internal/application/organization"
	sessionApp "github.com/udisondev/veziizi/backend/internal/application/session"
	orgDomain "github.com/udisondev/veziizi/backend/internal/domain/organization"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/geoip"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	members             *projections.MembersProjection
	freightRequestsProj *projections.FreightRequestsProjection
	orgService          *organization.Service
	session             *session.Manager
	sessionAnalyzer     *sessionApp.SessionAnalyzer
	geoIP               *geoip.Service
}

func NewAuthHandler(
	members *projections.MembersProjection,
	freightRequestsProj *projections.FreightRequestsProjection,
	orgService *organization.Service,
	session *session.Manager,
	sessionAnalyzer *sessionApp.SessionAnalyzer,
	geoIP *geoip.Service,
) *AuthHandler {
	return &AuthHandler{
		members:             members,
		freightRequestsProj: freightRequestsProj,
		orgService:          orgService,
		session:             session,
		sessionAnalyzer:     sessionAnalyzer,
		geoIP:               geoIP,
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

	// Validate required fields
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	// Extract client metadata for fraud tracking
	meta := httputil.GetClientMetadata(r)

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
		// Record failed login (blocked)
		if err := h.members.RecordLoginHistory(
			r.Context(), member.ID, member.OrganizationID,
			meta.IP, meta.Fingerprint, meta.UserAgent, "failed_blocked",
		); err != nil {
			slog.Error("failed to record login history", slog.String("error", err.Error()))
		}
		writeError(w, http.StatusForbidden, "account is blocked")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(member.PasswordHash), []byte(req.Password)); err != nil {
		// Record failed login (wrong password)
		if err := h.members.RecordLoginHistory(
			r.Context(), member.ID, member.OrganizationID,
			meta.IP, meta.Fingerprint, meta.UserAgent, "failed_password",
		); err != nil {
			slog.Error("failed to record login history", slog.String("error", err.Error()))
		}
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Record successful login in history
	if err := h.members.RecordLoginHistory(
		r.Context(), member.ID, member.OrganizationID,
		meta.IP, meta.Fingerprint, meta.UserAgent, "success",
	); err != nil {
		slog.Error("failed to record login history", slog.String("error", err.Error()))
	}

	// Update last_login_* fields
	if err := h.members.RecordLogin(r.Context(), member.ID, meta.IP, meta.Fingerprint); err != nil {
		slog.Error("failed to update last login", slog.String("error", err.Error()))
	}

	// Analyze login for fraud signals (geo jump, session anomaly)
	if h.sessionAnalyzer != nil {
		// Enrich with geo data if GeoIP is available
		var geoCountry, geoCity string
		var geoLat, geoLon float64
		if h.geoIP != nil && h.geoIP.IsAvailable() {
			geo := h.geoIP.Lookup(meta.IP)
			geoCountry = geo.Country
			geoCity = geo.City
			geoLat = geo.Latitude
			geoLon = geo.Longitude
		}

		loginInput := sessionApp.LoginAnalysisInput{
			MemberID:       member.ID,
			OrganizationID: member.OrganizationID,
			IPAddress:      meta.IP,
			Fingerprint:    meta.Fingerprint,
			UserAgent:      meta.UserAgent,
			GeoCountry:     geoCountry,
			GeoCity:        geoCity,
			GeoLat:         geoLat,
			GeoLon:         geoLon,
			LoginTime:      time.Now(),
		}
		if result, err := h.sessionAnalyzer.AnalyzeLogin(r.Context(), loginInput); err != nil {
			slog.Error("failed to analyze login", slog.String("error", err.Error()))
		} else if result.IsSuspicious {
			slog.Warn("suspicious login detected",
				slog.String("member_id", member.ID.String()),
				slog.Any("signals", result.Signals),
			)
		}
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
	Status         string               `json:"status"`
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
		Status:         member.Status,
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
	Role             string    `json:"role"`
	Status           string    `json:"status"`
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

	// SEC-017: Проверяем авторизацию
	sessionOrgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	member, err := h.orgService.GetMemberByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, orgDomain.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		slog.Error("failed to get member", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// SEC-017: Разрешаем доступ к членам своей организации или контрагентам по перевозкам
	if member.OrganizationID != sessionOrgID {
		hasShared, err := h.freightRequestsProj.HaveSharedConfirmedFreight(r.Context(), sessionOrgID, member.OrganizationID)
		if err != nil {
			slog.Error("failed to check shared freight requests",
				slog.String("error", err.Error()),
				slog.String("session_org_id", sessionOrgID.String()),
				slog.String("member_org_id", member.OrganizationID.String()),
			)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if !hasShared {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
	}

	var phone *string
	if member.Phone != "" {
		phone = &member.Phone
	}

	writeJSON(w, http.StatusOK, MemberProfileResponse{
		ID:               member.ID.String(),
		Name:             member.Name,
		Email:            member.Email,
		Phone:            phone,
		Role:             member.Role,
		Status:           member.Status,
		OrganizationID:   member.OrganizationID.String(),
		OrganizationName: member.OrganizationName,
		CreatedAt:        member.CreatedAt,
	})
}

package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"codeberg.org/udison/veziizi/backend/internal/application/admin"
	reviewApp "codeberg.org/udison/veziizi/backend/internal/application/review"
	orgDomain "codeberg.org/udison/veziizi/backend/internal/domain/organization"
	"codeberg.org/udison/veziizi/backend/internal/domain/review"
	adminRepo "codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/admin"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	service           *admin.Service
	adminRepo         *adminRepo.Repository
	session           *session.AdminManager
	reviewService     *reviewApp.Service
	reviewsProjection *projections.ReviewsProjection
	fraudProjection   *projections.FraudDataProjection
}

func NewAdminHandler(
	service *admin.Service,
	adminRepo *adminRepo.Repository,
	session *session.AdminManager,
	reviewService *reviewApp.Service,
	reviewsProjection *projections.ReviewsProjection,
	fraudProjection *projections.FraudDataProjection,
) *AdminHandler {
	return &AdminHandler{
		service:           service,
		adminRepo:         adminRepo,
		session:           session,
		reviewService:     reviewService,
		reviewsProjection: reviewsProjection,
		fraudProjection:   fraudProjection,
	}
}

func (h *AdminHandler) RegisterRoutes(r *mux.Router) {
	// Auth
	r.HandleFunc("/api/v1/admin/auth/login", h.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/auth/logout", h.Logout).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/auth/me", h.Me).Methods(http.MethodGet)

	// Organizations
	r.HandleFunc("/api/v1/admin/organizations", h.ListPending).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/admin/organizations/{id}", h.GetOrganization).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/admin/organizations/{id}/approve", h.Approve).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/organizations/{id}/reject", h.Reject).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/organizations/{id}/mark-fraudster", h.MarkFraudster).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/organizations/{id}/unmark-fraudster", h.UnmarkFraudster).Methods(http.MethodPost)

	// Fraudsters
	r.HandleFunc("/api/v1/admin/fraudsters", h.ListFraudsters).Methods(http.MethodGet)

	// Reviews moderation
	r.HandleFunc("/api/v1/admin/reviews", h.ListPendingReviews).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/admin/reviews/{id}", h.GetReview).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/admin/reviews/{id}/approve", h.ApproveReview).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/reviews/{id}/reject", h.RejectReview).Methods(http.MethodPost)
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

type AdminMeResponse struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

func (h *AdminHandler) Me(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	adm, err := h.adminRepo.GetByID(r.Context(), adminID)
	if err != nil {
		slog.Error("failed to get admin", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if !adm.IsActive {
		writeError(w, http.StatusForbidden, "account is disabled")
		return
	}

	writeJSON(w, http.StatusOK, AdminMeResponse{
		AdminID: adm.ID.String(),
		Email:   adm.Email,
		Name:    adm.Name,
	})
}

type PendingOrganizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	INN       string `json:"inn"`
	LegalName string `json:"legal_name"`
	Country   string `json:"country"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func (h *AdminHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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
			Status:    "pending",
			CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AdminHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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
		writeError(w, http.StatusConflict, "organization is not pending")
	case errors.Is(err, orgDomain.ErrAlreadyFraudster):
		writeError(w, http.StatusConflict, "organization is already marked as fraudster")
	case errors.Is(err, orgDomain.ErrNotFraudster):
		writeError(w, http.StatusConflict, "organization is not marked as fraudster")
	default:
		slog.Error("unhandled domain error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// === Review Moderation ===

type PendingReviewsResponse struct {
	Reviews []ReviewForModerationResponse `json:"reviews"`
	Total   int                           `json:"total"`
}

type ReviewForModerationResponse struct {
	ID             string               `json:"id"`
	OrderID        string               `json:"order_id"`
	ReviewerOrgID  string               `json:"reviewer_org_id"`
	ReviewedOrgID  string               `json:"reviewed_org_id"`
	Rating         int                  `json:"rating"`
	Comment        string               `json:"comment"`
	OrderAmount    int64                `json:"order_amount"`
	OrderCurrency  string               `json:"order_currency"`
	RawWeight      float64              `json:"raw_weight"`
	FraudScore     float64              `json:"fraud_score"`
	FraudSignals   []FraudSignalResponse `json:"fraud_signals"`
	ActivationDate *string              `json:"activation_date,omitempty"`
	CreatedAt      string               `json:"created_at"`
	AnalyzedAt     *string              `json:"analyzed_at,omitempty"`
}

type FraudSignalResponse struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	ScoreImpact float64 `json:"score_impact"`
}

func (h *AdminHandler) ListPendingReviews(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	reviews, total, err := h.reviewsProjection.ListPendingModeration(r.Context(), limit, offset)
	if err != nil {
		slog.Error("failed to list pending reviews", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response := PendingReviewsResponse{
		Reviews: make([]ReviewForModerationResponse, 0, len(reviews)),
		Total:   total,
	}

	for _, rev := range reviews {
		item := ReviewForModerationResponse{
			ID:            rev.ID.String(),
			OrderID:       rev.OrderID.String(),
			ReviewerOrgID: rev.ReviewerOrgID.String(),
			ReviewedOrgID: rev.ReviewedOrgID.String(),
			Rating:        rev.Rating,
			Comment:       rev.Comment,
			OrderAmount:   rev.OrderAmount,
			OrderCurrency: rev.OrderCurrency,
			RawWeight:     rev.RawWeight,
			FraudScore:    rev.FraudScore,
			FraudSignals:  make([]FraudSignalResponse, 0, len(rev.FraudSignals)),
			CreatedAt:     rev.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if rev.ActivationDate != nil {
			s := rev.ActivationDate.Format("2006-01-02T15:04:05Z")
			item.ActivationDate = &s
		}
		if rev.AnalyzedAt != nil {
			s := rev.AnalyzedAt.Format("2006-01-02T15:04:05Z")
			item.AnalyzedAt = &s
		}
		for _, sig := range rev.FraudSignals {
			item.FraudSignals = append(item.FraudSignals, FraudSignalResponse{
				Type:        sig.Type,
				Severity:    sig.Severity,
				Description: sig.Description,
				ScoreImpact: sig.ScoreImpact,
			})
		}
		response.Reviews = append(response.Reviews, item)
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AdminHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	reviewID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	rev, err := h.reviewsProjection.GetReviewByID(r.Context(), reviewID)
	if err != nil {
		slog.Error("failed to get review", slog.String("error", err.Error()))
		writeError(w, http.StatusNotFound, "review not found")
		return
	}

	signals, err := h.reviewsProjection.GetFraudSignalsByReviewID(r.Context(), reviewID)
	if err != nil {
		slog.Error("failed to get fraud signals", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response := ReviewForModerationResponse{
		ID:            rev.ID.String(),
		OrderID:       rev.OrderID.String(),
		ReviewerOrgID: rev.ReviewerOrgID.String(),
		ReviewedOrgID: rev.ReviewedOrgID.String(),
		Rating:        rev.Rating,
		Comment:       rev.Comment,
		OrderAmount:   rev.OrderAmount,
		OrderCurrency: rev.OrderCurrency,
		RawWeight:     rev.RawWeight,
		FraudScore:    rev.FraudScore,
		FraudSignals:  make([]FraudSignalResponse, 0, len(signals)),
		CreatedAt:     rev.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if rev.ActivationDate != nil {
		s := rev.ActivationDate.Format("2006-01-02T15:04:05Z")
		response.ActivationDate = &s
	}
	if rev.AnalyzedAt != nil {
		s := rev.AnalyzedAt.Format("2006-01-02T15:04:05Z")
		response.AnalyzedAt = &s
	}
	for _, sig := range signals {
		response.FraudSignals = append(response.FraudSignals, FraudSignalResponse{
			Type:        sig.Type,
			Severity:    sig.Severity,
			Description: sig.Description,
			ScoreImpact: sig.ScoreImpact,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

type ApproveReviewRequest struct {
	FinalWeight float64 `json:"final_weight"`
	Note        string  `json:"note,omitempty"`
}

func (h *AdminHandler) ApproveReview(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	reviewID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	var req ApproveReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FinalWeight <= 0 {
		req.FinalWeight = 1.0
	}

	if err := h.reviewService.Approve(r.Context(), reviewID, adminID, req.FinalWeight, req.Note); err != nil {
		handleReviewDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type RejectReviewRequest struct {
	Reason string `json:"reason"`
}

func (h *AdminHandler) RejectReview(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	reviewID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	var req RejectReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Reason == "" {
		writeError(w, http.StatusBadRequest, "reason is required")
		return
	}

	if err := h.reviewService.Reject(r.Context(), reviewID, adminID, req.Reason); err != nil {
		handleReviewDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleReviewDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, review.ErrReviewNotPendingMod):
		writeError(w, http.StatusBadRequest, "review is not pending moderation")
	case errors.Is(err, review.ErrReviewAlreadyAnalyzed):
		writeError(w, http.StatusBadRequest, "review already analyzed")
	default:
		slog.Error("unhandled review domain error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// === Fraudster Management ===

type MarkFraudsterRequest struct {
	IsConfirmed bool   `json:"is_confirmed"`
	Reason      string `json:"reason"`
}

func (h *AdminHandler) MarkFraudster(w http.ResponseWriter, r *http.Request) {
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

	var req MarkFraudsterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Reason == "" {
		writeError(w, http.StatusBadRequest, "reason is required")
		return
	}

	if err := h.service.MarkFraudster(r.Context(), admin.MarkFraudsterInput{
		AdminID:        adminID,
		OrganizationID: orgID,
		IsConfirmed:    req.IsConfirmed,
		Reason:         req.Reason,
	}); err != nil {
		handleAdminDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type UnmarkFraudsterRequest struct {
	Reason string `json:"reason"`
}

func (h *AdminHandler) UnmarkFraudster(w http.ResponseWriter, r *http.Request) {
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

	var req UnmarkFraudsterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UnmarkFraudster(r.Context(), admin.UnmarkFraudsterInput{
		AdminID:        adminID,
		OrganizationID: orgID,
		Reason:         req.Reason,
	}); err != nil {
		handleAdminDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type FraudsterResponse struct {
	OrgID              string  `json:"org_id"`
	OrgName            string  `json:"org_name"`
	IsConfirmed        bool    `json:"is_confirmed"`
	MarkedAt           string  `json:"marked_at"`
	Reason             string  `json:"reason"`
	TotalReviewsLeft   int     `json:"total_reviews_left"`
	DeactivatedReviews int     `json:"deactivated_reviews"`
	ReputationScore    float64 `json:"reputation_score"`
}

type FraudstersResponse struct {
	Fraudsters []FraudsterResponse `json:"fraudsters"`
	Total      int                 `json:"total"`
}

func (h *AdminHandler) ListFraudsters(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	fraudsters, total, err := h.fraudProjection.ListFraudsters(r.Context(), limit, offset)
	if err != nil {
		slog.Error("failed to list fraudsters", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response := FraudstersResponse{
		Fraudsters: make([]FraudsterResponse, 0, len(fraudsters)),
		Total:      total,
	}

	for _, f := range fraudsters {
		response.Fraudsters = append(response.Fraudsters, FraudsterResponse{
			OrgID:              f.OrgID.String(),
			OrgName:            f.OrgName,
			IsConfirmed:        f.IsConfirmed,
			MarkedAt:           f.MarkedAt.Format("2006-01-02T15:04:05Z"),
			Reason:             f.Reason,
			TotalReviewsLeft:   f.TotalReviewsLeft,
			DeactivatedReviews: f.DeactivatedReviews,
			ReputationScore:    f.ReputationScore,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

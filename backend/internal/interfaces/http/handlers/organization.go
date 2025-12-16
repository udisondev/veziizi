package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"codeberg.org/udison/veziizi/backend/internal/application/organization"
	orgDomain "codeberg.org/udison/veziizi/backend/internal/domain/organization"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type OrganizationHandler struct {
	service           *organization.Service
	ratingsProjection *projections.OrganizationRatingsProjection
	session           *session.Manager
}

func NewOrganizationHandler(
	service *organization.Service,
	ratingsProjection *projections.OrganizationRatingsProjection,
	session *session.Manager,
) *OrganizationHandler {
	return &OrganizationHandler{
		service:           service,
		ratingsProjection: ratingsProjection,
		session:           session,
	}
}

func (h *OrganizationHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/organizations", h.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/organizations/{id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/organizations/{id}/rating", h.GetRating).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/organizations/{id}/reviews", h.ListReviews).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/organizations/{id}/invitations", h.CreateInvitation).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/organizations/{id}/invitations", h.ListInvitations).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/organizations/{id}/members/{memberId}/role", h.ChangeMemberRole).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/organizations/{id}/members/{memberId}/block", h.BlockMember).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/organizations/{id}/members/{memberId}/unblock", h.UnblockMember).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/invitations/{token}", h.GetInvitation).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/invitations/{token}/accept", h.AcceptInvitation).Methods(http.MethodPost)
}

type RegisterRequest struct {
	Name          string `json:"name"`
	INN           string `json:"inn"`
	LegalName     string `json:"legal_name"`
	Country       string `json:"country"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	Address       string `json:"address"`
	OwnerEmail    string `json:"owner_email"`
	OwnerPassword string `json:"owner_password"`
	OwnerName     string `json:"owner_name"`
	OwnerPhone    string `json:"owner_phone"`
}

type RegisterResponse struct {
	OrganizationID string `json:"organization_id"`
	MemberID       string `json:"member_id"`
}

func (h *OrganizationHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	country := values.Country(req.Country)
	if !country.IsValid() {
		writeError(w, http.StatusBadRequest, "invalid country")
		return
	}

	output, err := h.service.Register(r.Context(), organization.RegisterInput{
		Name:          req.Name,
		INN:           req.INN,
		LegalName:     req.LegalName,
		Country:       country,
		Phone:         req.Phone,
		Email:         req.Email,
		Address:       values.Address(req.Address),
		OwnerEmail:    req.OwnerEmail,
		OwnerPassword: req.OwnerPassword,
		OwnerName:     req.OwnerName,
		OwnerPhone:    req.OwnerPhone,
	})
	if err != nil {
		slog.Error("failed to register organization", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to register organization")
		return
	}

	writeJSON(w, http.StatusCreated, RegisterResponse{
		OrganizationID: output.OrganizationID.String(),
		MemberID:       output.MemberID.String(),
	})
}

func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	org, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, orgDomain.ErrOrganizationNotFound) {
			writeError(w, http.StatusNotFound, "organization not found")
			return
		}
		slog.Error("failed to get organization", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to get organization")
		return
	}

	writeJSON(w, http.StatusOK, mapOrganizationToResponse(org))
}

type CreateInvitationRequest struct {
	Email string  `json:"email"`
	Role  string  `json:"role"`
	Name  *string `json:"name,omitempty"`  // предзаполненное ФИО
	Phone *string `json:"phone,omitempty"` // предзаполненный телефон
}

type CreateInvitationResponse struct {
	InvitationID string `json:"invitation_id"`
	Token        string `json:"token"` // для ручного тестирования (пока нет отправки email)
}

func (h *OrganizationHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	actorID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role := values.MemberRole(req.Role)
	if !role.IsValid() {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}

	output, err := h.service.CreateInvitation(r.Context(), organization.CreateInvitationInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		Email:          req.Email,
		Role:           role,
		Name:           req.Name,
		Phone:          req.Phone,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, CreateInvitationResponse{
		InvitationID: output.InvitationID.String(),
		Token:        output.Token,
	})
}

type AcceptInvitationRequest struct {
	Password string  `json:"password"`
	Name     *string `json:"name,omitempty"`  // опционально, если предзаполнено в приглашении
	Phone    *string `json:"phone,omitempty"` // опционально, если предзаполнено в приглашении
}

type AcceptInvitationResponse struct {
	OrganizationID string `json:"organization_id"`
	MemberID       string `json:"member_id"`
}

func (h *OrganizationHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	var req AcceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	output, err := h.service.AcceptInvitation(r.Context(), organization.AcceptInvitationInput{
		Token:    token,
		Password: req.Password,
		Name:     req.Name,
		Phone:    req.Phone,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, AcceptInvitationResponse{
		OrganizationID: output.OrganizationID.String(),
		MemberID:       output.MemberID.String(),
	})
}

// GetInvitation возвращает данные приглашения по токену (публичный endpoint)
type InvitationResponse struct {
	ID               string  `json:"id"`
	OrganizationID   string  `json:"organization_id"`
	OrganizationName string  `json:"organization_name"`
	Email            string  `json:"email"`
	Role             string  `json:"role"`
	Name             *string `json:"name,omitempty"`
	Phone            *string `json:"phone,omitempty"`
	Status           string  `json:"status"`
	ExpiresAt        string  `json:"expires_at"`
}

func (h *OrganizationHandler) GetInvitation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	inv, err := h.service.GetInvitationByToken(r.Context(), token)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, InvitationResponse{
		ID:               inv.ID.String(),
		OrganizationID:   inv.OrganizationID.String(),
		OrganizationName: inv.OrganizationName,
		Email:            inv.Email,
		Role:             inv.Role,
		Name:             inv.Name,
		Phone:            inv.Phone,
		Status:           inv.Status,
		ExpiresAt:        inv.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

// ListInvitations возвращает список приглашений организации
type InvitationListItem struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	Name      *string `json:"name,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Status    string  `json:"status"`
	ExpiresAt string  `json:"expires_at"`
	CreatedAt string  `json:"created_at"`
}

type InvitationListResponse struct {
	Items []InvitationListItem `json:"items"`
}

func (h *OrganizationHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	// Проверяем авторизацию
	_, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Получаем опциональный фильтр по статусу
	var status *string
	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}

	invitations, err := h.service.ListInvitations(r.Context(), organization.ListInvitationsInput{
		OrganizationID: orgID,
		Status:         status,
	})
	if err != nil {
		slog.Error("failed to list invitations", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list invitations")
		return
	}

	items := make([]InvitationListItem, 0, len(invitations))
	for _, inv := range invitations {
		items = append(items, InvitationListItem{
			ID:        inv.ID.String(),
			Email:     inv.Email,
			Role:      inv.Role,
			Name:      inv.Name,
			Phone:     inv.Phone,
			Status:    inv.Status,
			ExpiresAt: inv.ExpiresAt.Format("2006-01-02T15:04:05Z"),
			CreatedAt: inv.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, InvitationListResponse{Items: items})
}

type ChangeMemberRoleRequest struct {
	Role string `json:"role"`
}

func (h *OrganizationHandler) ChangeMemberRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	memberID, err := uuid.Parse(vars["memberId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member id")
		return
	}

	actorID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ChangeMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role := values.MemberRole(req.Role)
	if !role.IsValid() {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}

	if err := h.service.ChangeMemberRole(r.Context(), organization.ChangeMemberRoleInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		MemberID:       memberID,
		NewRole:        role,
	}); err != nil {
		handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type BlockMemberRequest struct {
	Reason string `json:"reason"`
}

func (h *OrganizationHandler) BlockMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	memberID, err := uuid.Parse(vars["memberId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member id")
		return
	}

	actorID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req BlockMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.BlockMember(r.Context(), organization.BlockMemberInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		MemberID:       memberID,
		Reason:         req.Reason,
	}); err != nil {
		handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrganizationHandler) UnblockMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	memberID, err := uuid.Parse(vars["memberId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member id")
		return
	}

	actorID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.UnblockMember(r.Context(), organization.UnblockMemberInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		MemberID:       memberID,
	}); err != nil {
		handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, orgDomain.ErrOrganizationNotFound):
		writeError(w, http.StatusNotFound, "organization not found")
	case errors.Is(err, orgDomain.ErrMemberNotFound):
		writeError(w, http.StatusNotFound, "member not found")
	case errors.Is(err, orgDomain.ErrInvitationNotFound):
		writeError(w, http.StatusNotFound, "invitation not found")
	case errors.Is(err, orgDomain.ErrInvitationExpired):
		writeError(w, http.StatusGone, "invitation expired")
	case errors.Is(err, orgDomain.ErrInvitationAlreadyUsed):
		writeError(w, http.StatusConflict, "invitation already used")
	case errors.Is(err, orgDomain.ErrMemberAlreadyExists):
		writeError(w, http.StatusConflict, "member already exists")
	case errors.Is(err, orgDomain.ErrEmailAlreadyInvited):
		writeError(w, http.StatusConflict, "email already invited")
	case errors.Is(err, orgDomain.ErrInsufficientPermissions):
		writeError(w, http.StatusForbidden, "insufficient permissions")
	case errors.Is(err, orgDomain.ErrCannotChangeOwnRole):
		writeError(w, http.StatusBadRequest, "cannot change own role")
	case errors.Is(err, orgDomain.ErrCannotBlockSelf):
		writeError(w, http.StatusBadRequest, "cannot block yourself")
	case errors.Is(err, orgDomain.ErrMemberCannotBeRemoved):
		writeError(w, http.StatusBadRequest, "owner cannot be removed")
	case errors.Is(err, orgDomain.ErrNameRequired):
		writeError(w, http.StatusBadRequest, "name is required")
	case errors.Is(err, orgDomain.ErrPhoneRequired):
		writeError(w, http.StatusBadRequest, "phone is required")
	default:
		slog.Error("unhandled domain error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

type OrganizationResponse struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	INN       string           `json:"inn"`
	LegalName string           `json:"legal_name"`
	Country   string           `json:"country"`
	Phone     string           `json:"phone"`
	Email     string           `json:"email"`
	Address   string           `json:"address"`
	Status    string           `json:"status"`
	Members   []MemberResponse `json:"members"`
	CreatedAt string           `json:"created_at"`
}

type MemberResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func mapOrganizationToResponse(org *orgDomain.Organization) OrganizationResponse {
	members := make([]MemberResponse, 0, len(org.Members()))
	for _, m := range org.Members() {
		members = append(members, MemberResponse{
			ID:        m.ID().String(),
			Email:     m.Email(),
			Name:      m.Name(),
			Phone:     m.Phone(),
			Role:      m.Role().String(),
			Status:    m.Status().String(),
			CreatedAt: m.CreatedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	return OrganizationResponse{
		ID:        org.ID().String(),
		Name:      org.Name(),
		INN:       org.INN(),
		LegalName: org.LegalName(),
		Country:   org.Country().String(),
		Phone:     org.Phone(),
		Email:     org.Email(),
		Address:   org.Address().String(),
		Status:    org.Status().String(),
		Members:   members,
		CreatedAt: org.CreatedAt().Format("2006-01-02T15:04:05Z"),
	}
}

// Rating response types

type RatingResponse struct {
	TotalReviews  int     `json:"total_reviews"`
	AverageRating float64 `json:"average_rating"`
}

type OrgReviewResponse struct {
	ID              string `json:"id"`
	OrderID         string `json:"order_id"`
	ReviewerOrgID   string `json:"reviewer_org_id"`
	ReviewerOrgName string `json:"reviewer_org_name"`
	Rating          int    `json:"rating"`
	Comment         string `json:"comment"`
	CreatedAt       string `json:"created_at"`
}

type ReviewsListResponse struct {
	Items []OrgReviewResponse `json:"items"`
	Total int                 `json:"total"`
}

func (h *OrganizationHandler) GetRating(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	rating, err := h.ratingsProjection.GetRating(r.Context(), id)
	if err != nil {
		slog.Error("failed to get rating", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to get rating")
		return
	}

	writeJSON(w, http.StatusOK, RatingResponse{
		TotalReviews:  rating.TotalReviews,
		AverageRating: rating.AverageRating,
	})
}

func (h *OrganizationHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	// Parse pagination params
	limit := 10
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	reviews, total, err := h.ratingsProjection.ListReviews(r.Context(), id, limit, offset)
	if err != nil {
		slog.Error("failed to list reviews", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list reviews")
		return
	}

	items := make([]OrgReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		items = append(items, OrgReviewResponse{
			ID:              review.ID.String(),
			OrderID:         review.OrderID.String(),
			ReviewerOrgID:   review.ReviewerOrgID.String(),
			ReviewerOrgName: review.ReviewerOrgName,
			Rating:          review.Rating,
			Comment:         review.Comment,
			CreatedAt:       review.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, ReviewsListResponse{
		Items: items,
		Total: total,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", slog.String("error", err.Error()))
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

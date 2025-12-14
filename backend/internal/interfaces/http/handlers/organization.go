package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"codeberg.org/udison/veziizi/backend/internal/application/organization"
	orgDomain "codeberg.org/udison/veziizi/backend/internal/domain/organization"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization/values"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type OrganizationHandler struct {
	service *organization.Service
	session *session.Manager
}

func NewOrganizationHandler(service *organization.Service, session *session.Manager) *OrganizationHandler {
	return &OrganizationHandler{
		service: service,
		session: session,
	}
}

func (h *OrganizationHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/organizations", h.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/organizations/{id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/organizations/{id}/invitations", h.CreateInvitation).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/organizations/{id}/carrier-profile", h.SetCarrierProfile).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/organizations/{id}/members/{memberId}/role", h.ChangeMemberRole).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/organizations/{id}/members/{memberId}/block", h.BlockMember).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/organizations/{id}/members/{memberId}/unblock", h.UnblockMember).Methods(http.MethodPost)
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
	Email string `json:"email"`
	Role  string `json:"role"`
}

type CreateInvitationResponse struct {
	InvitationID string `json:"invitation_id"`
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

	invID, err := h.service.CreateInvitation(r.Context(), organization.CreateInvitationInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		Email:          req.Email,
		Role:           role,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, CreateInvitationResponse{
		InvitationID: invID.String(),
	})
}

type AcceptInvitationRequest struct {
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
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

type SetCarrierProfileRequest struct {
	Description    string   `json:"description"`
	VehicleTypes   []string `json:"vehicle_types"`
	Regions        []string `json:"regions"`
	HasADR         bool     `json:"has_adr"`
	HasRefrigerator bool    `json:"has_refrigerator"`
}

func (h *OrganizationHandler) SetCarrierProfile(w http.ResponseWriter, r *http.Request) {
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

	var req SetCarrierProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	profile := values.CarrierProfile{
		Description:    req.Description,
		VehicleTypes:   req.VehicleTypes,
		Regions:        req.Regions,
		HasADR:         req.HasADR,
		HasRefrigerator: req.HasRefrigerator,
	}

	if err := h.service.SetCarrierProfile(r.Context(), organization.SetCarrierProfileInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		Profile:        profile,
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
	default:
		slog.Error("unhandled domain error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

type OrganizationResponse struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	INN            string                  `json:"inn"`
	LegalName      string                  `json:"legal_name"`
	Country        string                  `json:"country"`
	Phone          string                  `json:"phone"`
	Email          string                  `json:"email"`
	Address        string                  `json:"address"`
	Status         string                  `json:"status"`
	IsCarrier      bool                    `json:"is_carrier"`
	CarrierProfile *values.CarrierProfile  `json:"carrier_profile,omitempty"`
	Members        []MemberResponse        `json:"members"`
	CreatedAt      string                  `json:"created_at"`
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
		ID:             org.ID().String(),
		Name:           org.Name(),
		INN:            org.INN(),
		LegalName:      org.LegalName(),
		Country:        org.Country().String(),
		Phone:          org.Phone(),
		Email:          org.Email(),
		Address:        org.Address().String(),
		Status:         org.Status().String(),
		IsCarrier:      org.IsCarrier(),
		CarrierProfile: org.CarrierProfile(),
		Members:        members,
		CreatedAt:      org.CreatedAt().Format("2006-01-02T15:04:05Z"),
	}
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

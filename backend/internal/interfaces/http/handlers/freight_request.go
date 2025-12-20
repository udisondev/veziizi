package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/application/freightrequest"
	"codeberg.org/udison/veziizi/backend/internal/application/organization"
	frDomain "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest"
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/entities"
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	orgDomain "codeberg.org/udison/veziizi/backend/internal/domain/organization"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"codeberg.org/udison/veziizi/backend/internal/pkg/httputil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type FreightRequestHandler struct {
	service           *freightrequest.Service
	orgService        *organization.Service
	projection        *projections.FreightRequestsProjection
	membersProjection *projections.MembersProjection
	session           *session.Manager
}

func NewFreightRequestHandler(
	service *freightrequest.Service,
	orgService *organization.Service,
	projection *projections.FreightRequestsProjection,
	membersProjection *projections.MembersProjection,
	session *session.Manager,
) *FreightRequestHandler {
	return &FreightRequestHandler{
		service:           service,
		orgService:        orgService,
		projection:        projection,
		membersProjection: membersProjection,
		session:           session,
	}
}

// Response types for full data from event store

type FreightRequestResponse struct {
	ID                  uuid.UUID                  `json:"id"`
	RequestNumber       int64                      `json:"request_number"`
	CustomerOrgID       uuid.UUID                  `json:"customer_org_id"`
	CustomerMemberID    uuid.UUID                  `json:"customer_member_id"`
	Route               values.Route               `json:"route"`
	Cargo               values.CargoInfo           `json:"cargo"`
	VehicleRequirements values.VehicleRequirements `json:"vehicle_requirements"`
	Payment             values.Payment             `json:"payment"`
	Comment             string                     `json:"comment,omitempty"`
	Status              string                     `json:"status"`
	FreightVersion      int                        `json:"freight_version"`
	ExpiresAt           time.Time                  `json:"expires_at"`
	CreatedAt           time.Time                  `json:"created_at"`
	CancelledAt         *time.Time                 `json:"cancelled_at,omitempty"`
	SelectedOfferID     *uuid.UUID                 `json:"selected_offer_id,omitempty"`
}

type OfferResponse struct {
	ID                uuid.UUID    `json:"id"`
	CarrierOrgID      uuid.UUID    `json:"carrier_org_id"`
	CarrierOrgName    string       `json:"carrier_org_name,omitempty"`
	CarrierMemberID   *uuid.UUID   `json:"carrier_member_id,omitempty"`
	CarrierMemberName string       `json:"carrier_member_name,omitempty"`
	Price             values.Money `json:"price"`
	Comment           string       `json:"comment,omitempty"`
	VatType           string       `json:"vat_type"`
	PaymentMethod     string       `json:"payment_method"`
	FreightVersion    int          `json:"freight_version"`
	Status            string       `json:"status"`
	CreatedAt         time.Time    `json:"created_at"`
}

func (h *FreightRequestHandler) toFreightRequestResponse(fr *frDomain.FreightRequest) FreightRequestResponse {
	return FreightRequestResponse{
		ID:                  fr.ID(),
		RequestNumber:       fr.RequestNumber(),
		CustomerOrgID:       fr.CustomerOrgID(),
		CustomerMemberID:    fr.CustomerMemberID(),
		Route:               fr.Route(),
		Cargo:               fr.Cargo(),
		VehicleRequirements: fr.VehicleRequirements(),
		Payment:             fr.Payment(),
		Comment:             fr.Comment(),
		Status:              fr.Status().String(),
		FreightVersion:      fr.FreightVersion(),
		ExpiresAt:           fr.ExpiresAt(),
		CreatedAt:           fr.CreatedAt(),
		CancelledAt:         fr.CancelledAt(),
		SelectedOfferID:     fr.SelectedOfferID(),
	}
}

// toOfferResponse преобразует оффер в ответ API.
// isOfferOwner определяет, является ли текущий пользователь владельцем оффера (перевозчиком).
// Только владелец оффера видит информацию о сотруднике (member_id, member_name).
// Заказчик видит только информацию об организации.
func (h *FreightRequestHandler) toOfferResponse(offer *entities.Offer, orgName, memberName string, isOfferOwner bool) OfferResponse {
	resp := OfferResponse{
		ID:             offer.ID(),
		CarrierOrgID:   offer.CarrierOrgID(),
		CarrierOrgName: orgName,
		Price:          offer.Price(),
		Comment:        offer.Comment(),
		VatType:        offer.VatType().String(),
		PaymentMethod:  offer.PaymentMethod().String(),
		FreightVersion: offer.FreightVersion(),
		Status:         offer.Status().String(),
		CreatedAt:      offer.CreatedAt(),
	}

	// Информация о сотруднике только для владельца оффера
	if isOfferOwner {
		memberID := offer.CarrierMemberID()
		resp.CarrierMemberID = &memberID
		resp.CarrierMemberName = memberName
	}

	return resp
}

func (h *FreightRequestHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/freight-requests", h.Create).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/freight-requests", h.List).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/freight-requests/{id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/freight-requests/{id}", h.Update).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/freight-requests/{id}", h.Cancel).Methods(http.MethodDelete)
	r.HandleFunc("/api/v1/freight-requests/{id}/reassign", h.Reassign).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/freight-requests/{id}/offers", h.MakeOffer).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/freight-requests/{id}/offers", h.ListOffers).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/freight-requests/{id}/offers/{offerId}", h.WithdrawOffer).Methods(http.MethodDelete)
	r.HandleFunc("/api/v1/freight-requests/{id}/offers/{offerId}/select", h.SelectOffer).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/freight-requests/{id}/offers/{offerId}/reject", h.RejectOffer).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/freight-requests/{id}/offers/{offerId}/confirm", h.ConfirmOffer).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/freight-requests/{id}/offers/{offerId}/decline", h.DeclineOffer).Methods(http.MethodPost)
	// My offers (for current organization)
	r.HandleFunc("/api/v1/offers", h.ListMyOffers).Methods(http.MethodGet)
}

type CreateFreightRequestRequest struct {
	Route               values.Route               `json:"route"`
	Cargo               values.CargoInfo           `json:"cargo"`
	VehicleRequirements values.VehicleRequirements `json:"vehicle_requirements"`
	Payment             values.Payment             `json:"payment"`
	Comment             string                     `json:"comment,omitempty"`
	ExpiresAt           *time.Time                 `json:"expires_at,omitempty"`
}

type CreateFreightRequestResponse struct {
	ID string `json:"id"`
}

func (h *FreightRequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateFreightRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id, err := h.service.Create(r.Context(), freightrequest.CreateInput{
		CustomerOrgID:       orgID,
		CustomerMemberID:    memberID,
		Route:               req.Route,
		Cargo:               req.Cargo,
		VehicleRequirements: req.VehicleRequirements,
		Payment:             req.Payment,
		Comment:             req.Comment,
		ExpiresAt:           req.ExpiresAt,
	})
	if err != nil {
		slog.Error("failed to create freight request", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to create freight request")
		return
	}

	writeJSON(w, http.StatusCreated, CreateFreightRequestResponse{ID: id.String()})
}

func (h *FreightRequestHandler) List(w http.ResponseWriter, r *http.Request) {
	// SEC-009: Получаем orgID пользователя для фильтрации
	sessionOrgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var opts []projections.FilterOption
	status := r.URL.Query().Get("status")
	isMarketRequest := r.URL.Query().Get("mode") == "market" || status == "published"

	if orgIDStr := r.URL.Query().Get("customer_org_id"); orgIDStr != "" {
		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid customer_org_id")
			return
		}
		// SEC-009: Разрешаем чужие организации только для просмотра опубликованных заявок (маркет)
		if orgID != sessionOrgID && !isMarketRequest {
			writeError(w, http.StatusForbidden, "access denied to other organization's freight requests")
			return
		}
		opts = append(opts, projections.WithCustomerOrgID(orgID))
	} else if isMarketRequest {
		// SEC-009: Режим маркета - показываем опубликованные заявки от всех организаций
		// Фильтр по customer_org_id не добавляется
	} else {
		// SEC-009: По умолчанию показываем только заявки своей организации
		opts = append(opts, projections.WithCustomerOrgID(sessionOrgID))
	}

	if memberIDStr := r.URL.Query().Get("member_id"); memberIDStr != "" {
		memberID, err := uuid.Parse(memberIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid member_id")
			return
		}
		opts = append(opts, projections.WithCustomerMemberID(memberID))
	}

	// SEC-009: В режиме маркета принудительно фильтруем по статусу "published"
	if status != "" {
		opts = append(opts, projections.WithStatus(status))
	} else if isMarketRequest {
		opts = append(opts, projections.WithStatus("published"))
	}

	if orgName := r.URL.Query().Get("org_name"); orgName != "" {
		opts = append(opts, projections.WithOrgNameLike(orgName))
	}

	if orgINN := r.URL.Query().Get("org_inn"); orgINN != "" {
		opts = append(opts, projections.WithOrgINN(orgINN))
	}

	if orgCountry := r.URL.Query().Get("org_country"); orgCountry != "" {
		opts = append(opts, projections.WithOrgCountry(orgCountry))
	}

	// SEC-016: Валидированная пагинация
	pagination := httputil.ParsePagination(r)
	opts = append(opts, projections.WithLimit(pagination.Limit))
	opts = append(opts, projections.WithOffset(pagination.Offset))

	items, err := h.projection.List(r.Context(), opts...)
	if err != nil {
		slog.Error("failed to list freight requests", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list freight requests")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *FreightRequestHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	fr, err := h.service.Get(r.Context(), id)
	if err != nil {
		slog.Error("failed to get freight request", slog.String("error", err.Error()))
		writeError(w, http.StatusNotFound, "freight request not found")
		return
	}

	// Check if aggregate has events (exists)
	if fr.Version() == 0 {
		writeError(w, http.StatusNotFound, "freight request not found")
		return
	}

	resp := h.toFreightRequestResponse(fr)

	// Fallback: load request_number from lookup for old events without it
	if resp.RequestNumber == 0 {
		if lookup, err := h.projection.GetByID(r.Context(), id); err != nil {
			slog.Error("failed to get freight request lookup", slog.String("error", err.Error()))
		} else {
			resp.RequestNumber = lookup.RequestNumber
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

type UpdateFreightRequestRequest struct {
	Route               *values.Route               `json:"route,omitempty"`
	Cargo               *values.CargoInfo           `json:"cargo,omitempty"`
	VehicleRequirements *values.VehicleRequirements `json:"vehicle_requirements,omitempty"`
	Payment             *values.Payment             `json:"payment,omitempty"`
	Comment             *string                     `json:"comment,omitempty"`
}

func (h *FreightRequestHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateFreightRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Update(r.Context(), freightrequest.UpdateInput{
		ID:                  id,
		ActorID:             memberID,
		Route:               req.Route,
		Cargo:               req.Cargo,
		VehicleRequirements: req.VehicleRequirements,
		Payment:             req.Payment,
		Comment:             req.Comment,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type CancelFreightRequestRequest struct {
	Reason string `json:"reason,omitempty"`
}

func (h *FreightRequestHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CancelFreightRequestRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // optional body

	if err := h.service.Cancel(r.Context(), freightrequest.CancelInput{
		ID:      id,
		ActorID: memberID,
		Reason:  req.Reason,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ReassignRequest struct {
	NewMemberID string `json:"new_member_id"`
}

func (h *FreightRequestHandler) Reassign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newMemberID, err := uuid.Parse(req.NewMemberID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid new_member_id")
		return
	}

	if err := h.service.Reassign(r.Context(), freightrequest.ReassignInput{
		ID:          id,
		ActorID:     memberID,
		ActorOrgID:  orgID,
		NewMemberID: newMemberID,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type MakeOfferRequest struct {
	Price         values.Money         `json:"price"`
	Comment       string               `json:"comment,omitempty"`
	VatType       values.VatType       `json:"vat_type"`
	PaymentMethod values.PaymentMethod `json:"payment_method"`
}

type MakeOfferResponse struct {
	OfferID string `json:"offer_id"`
}

func (h *FreightRequestHandler) MakeOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req MakeOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	offerID, err := h.service.MakeOffer(r.Context(), freightrequest.MakeOfferInput{
		FreightRequestID: frID,
		CarrierOrgID:     orgID,
		CarrierMemberID:  memberID,
		Price:            req.Price,
		Comment:          req.Comment,
		VatType:          req.VatType,
		PaymentMethod:    req.PaymentMethod,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, MakeOfferResponse{OfferID: offerID.String()})
}

func (h *FreightRequestHandler) ListOffers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	currentOrgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fr, err := h.service.Get(r.Context(), frID)
	if err != nil {
		slog.Error("failed to get freight request", slog.String("error", err.Error()))
		writeError(w, http.StatusNotFound, "freight request not found")
		return
	}

	if fr.Version() == 0 {
		writeError(w, http.StatusNotFound, "freight request not found")
		return
	}

	// Фильтруем офферы по правам доступа:
	// - Заказчик видит все офферы
	// - Перевозчик видит только свои офферы
	// - Посторонние не видят ничего
	isOwner := fr.CustomerOrgID() == currentOrgID
	filteredOffers := make([]*entities.Offer, 0)
	for _, offer := range fr.Offers() {
		if isOwner || offer.CarrierOrgID() == currentOrgID {
			filteredOffers = append(filteredOffers, offer)
		}
	}

	// Собираем уникальные ID организаций и членов
	orgIDs := make([]uuid.UUID, 0, len(filteredOffers))
	memberIDs := make([]uuid.UUID, 0, len(filteredOffers))
	seenOrgs := make(map[uuid.UUID]bool)
	seenMembers := make(map[uuid.UUID]bool)
	for _, offer := range filteredOffers {
		if !seenOrgs[offer.CarrierOrgID()] {
			seenOrgs[offer.CarrierOrgID()] = true
			orgIDs = append(orgIDs, offer.CarrierOrgID())
		}
		if !seenMembers[offer.CarrierMemberID()] {
			seenMembers[offer.CarrierMemberID()] = true
			memberIDs = append(memberIDs, offer.CarrierMemberID())
		}
	}

	// Загружаем названия организаций
	orgNames, err := h.orgService.GetNames(r.Context(), orgIDs)
	if err != nil {
		slog.Error("failed to get organization names", slog.String("error", err.Error()))
		orgNames = make(map[uuid.UUID]string)
	}

	// Загружаем имена членов
	memberNames, err := h.membersProjection.GetNames(r.Context(), memberIDs)
	if err != nil {
		slog.Error("failed to get member names", slog.String("error", err.Error()))
		memberNames = make(map[uuid.UUID]string)
	}

	offers := make([]OfferResponse, 0, len(filteredOffers))
	for _, offer := range filteredOffers {
		isOfferOwner := offer.CarrierOrgID() == currentOrgID
		offers = append(offers, h.toOfferResponse(
			offer,
			orgNames[offer.CarrierOrgID()],
			memberNames[offer.CarrierMemberID()],
			isOfferOwner,
		))
	}

	writeJSON(w, http.StatusOK, offers)
}

// ListMyOffers returns all offers made by current organization with freight request data
func (h *FreightRequestHandler) ListMyOffers(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var opts []projections.OfferFilterOption
	opts = append(opts, projections.WithCarrierOrgIDAlias(orgID))

	if status := r.URL.Query().Get("status"); status != "" {
		opts = append(opts, projections.WithOfferStatusAlias(status))
	}

	// SEC-016: Валидированная пагинация
	pagination := httputil.ParsePagination(r)
	opts = append(opts, projections.WithOfferLimit(pagination.Limit))
	opts = append(opts, projections.WithOfferOffset(pagination.Offset))

	items, err := h.projection.ListOffersWithFreightData(r.Context(), opts...)
	if err != nil {
		slog.Error("failed to list my offers", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list offers")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

type WithdrawOfferRequest struct {
	Reason string `json:"reason,omitempty"`
}

func (h *FreightRequestHandler) WithdrawOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	offerID, err := uuid.Parse(vars["offerId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offer id")
		return
	}

	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req WithdrawOfferRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // optional

	if err := h.service.WithdrawOffer(r.Context(), freightrequest.WithdrawOfferInput{
		FreightRequestID: frID,
		OfferID:          offerID,
		ActorOrgID:       orgID,
		ActorMemberID:    memberID,
		Reason:           req.Reason,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FreightRequestHandler) SelectOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	offerID, err := uuid.Parse(vars["offerId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offer id")
		return
	}

	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.SelectOffer(r.Context(), freightrequest.SelectOfferInput{
		FreightRequestID: frID,
		OfferID:          offerID,
		ActorID:          memberID,
		ActorOrgID:       orgID,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type RejectOfferRequest struct {
	Reason string `json:"reason,omitempty"`
}

func (h *FreightRequestHandler) RejectOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	offerID, err := uuid.Parse(vars["offerId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offer id")
		return
	}

	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req RejectOfferRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.service.RejectOffer(r.Context(), freightrequest.RejectOfferInput{
		FreightRequestID: frID,
		OfferID:          offerID,
		ActorID:          memberID,
		ActorOrgID:       orgID,
		Reason:           req.Reason,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FreightRequestHandler) ConfirmOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	offerID, err := uuid.Parse(vars["offerId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offer id")
		return
	}

	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.ConfirmOffer(r.Context(), freightrequest.ConfirmOfferInput{
		FreightRequestID: frID,
		OfferID:          offerID,
		ActorOrgID:       orgID,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type DeclineOfferRequest struct {
	Reason string `json:"reason,omitempty"`
}

func (h *FreightRequestHandler) DeclineOffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	frID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	offerID, err := uuid.Parse(vars["offerId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offer id")
		return
	}

	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req DeclineOfferRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.service.DeclineOffer(r.Context(), freightrequest.DeclineOfferInput{
		FreightRequestID: frID,
		OfferID:          offerID,
		ActorOrgID:       orgID,
		Reason:           req.Reason,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FreightRequestHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, frDomain.ErrFreightRequestNotFound):
		writeError(w, http.StatusNotFound, "freight request not found")
	case errors.Is(err, frDomain.ErrFreightRequestNotPublished):
		writeError(w, http.StatusConflict, "freight request is not published")
	case errors.Is(err, frDomain.ErrFreightRequestExpired):
		writeError(w, http.StatusGone, "freight request has expired")
	case errors.Is(err, frDomain.ErrFreightRequestCancelled):
		writeError(w, http.StatusConflict, "freight request is cancelled")
	case errors.Is(err, frDomain.ErrFreightRequestConfirmed):
		writeError(w, http.StatusConflict, "freight request is confirmed")
	case errors.Is(err, frDomain.ErrOfferNotFound):
		writeError(w, http.StatusNotFound, "offer not found")
	case errors.Is(err, frDomain.ErrOfferNotPending):
		writeError(w, http.StatusConflict, "offer is not pending")
	case errors.Is(err, frDomain.ErrOfferNotSelected):
		writeError(w, http.StatusConflict, "offer is not selected")
	case errors.Is(err, frDomain.ErrOfferAlreadyExists):
		writeError(w, http.StatusConflict, "offer already exists")
	case errors.Is(err, frDomain.ErrCannotOfferOwnRequest):
		writeError(w, http.StatusBadRequest, "cannot make offer on own request")
	case errors.Is(err, frDomain.ErrNotFreightRequestOwner):
		writeError(w, http.StatusForbidden, "not freight request owner")
	case errors.Is(err, frDomain.ErrNotOfferOwner):
		writeError(w, http.StatusForbidden, "not offer owner")
	case errors.Is(err, frDomain.ErrHasSelectedOffer):
		writeError(w, http.StatusConflict, "already has selected offer")
	case errors.Is(err, orgDomain.ErrMemberNotFound):
		writeError(w, http.StatusNotFound, "member not found")
	case errors.Is(err, orgDomain.ErrInsufficientPermissions):
		writeError(w, http.StatusForbidden, "insufficient permissions")
	default:
		slog.Error("unhandled domain error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}


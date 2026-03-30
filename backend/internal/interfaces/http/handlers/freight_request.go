package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/udisondev/veziizi/backend/internal/application/freightrequest"
	"github.com/udisondev/veziizi/backend/internal/application/organization"
	frDomain "github.com/udisondev/veziizi/backend/internal/domain/freightrequest"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	orgDomain "github.com/udisondev/veziizi/backend/internal/domain/organization"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
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

// FreightRequestListResponse представляет ответ списка с cursor-based пагинацией.
type FreightRequestListResponse struct {
	Items      []projections.FreightRequestListItem `json:"items"`
	NextCursor *string                              `json:"next_cursor,omitempty"`
	HasMore    bool                                 `json:"has_more"`
}

type FreightRequestResponse struct {
	ID                  uuid.UUID                  `json:"id"`
	RequestNumber       int64                      `json:"request_number"`
	CustomerOrgID       uuid.UUID                  `json:"customer_org_id"`
	CustomerOrgName     string                     `json:"customer_org_name"`
	CustomerMemberID    uuid.UUID                  `json:"customer_member_id"`
	CustomerMemberName  string                     `json:"customer_member_name,omitempty"`
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
	// Carrier info (видно при confirmed или перевозчику)
	CarrierOrgID      *uuid.UUID `json:"carrier_org_id,omitempty"`
	CarrierOrgName    string     `json:"carrier_org_name,omitempty"`
	CarrierMemberID   *uuid.UUID `json:"carrier_member_id,omitempty"`
	CarrierMemberName string     `json:"carrier_member_name,omitempty"`
	// Completion status
	CustomerCompleted   bool       `json:"customer_completed"`
	CustomerCompletedAt *time.Time `json:"customer_completed_at,omitempty"`
	CarrierCompleted    bool       `json:"carrier_completed"`
	CarrierCompletedAt  *time.Time `json:"carrier_completed_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	// Reviews
	CustomerReview *ReviewResponse `json:"customer_review,omitempty"`
	CarrierReview  *ReviewResponse `json:"carrier_review,omitempty"`
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

type ReviewResponse struct {
	ID            uuid.UUID  `json:"id"`
	Rating        int        `json:"rating"`
	Comment       string     `json:"comment,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CanEdit       bool       `json:"can_edit"`
	EditExpiresAt *time.Time `json:"edit_expires_at,omitempty"`
}

func (h *FreightRequestHandler) toFreightRequestResponse(fr *frDomain.FreightRequest) FreightRequestResponse {
	resp := FreightRequestResponse{
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
		// Completion
		CustomerCompleted:   fr.CustomerCompleted(),
		CustomerCompletedAt: fr.CustomerCompletedAt(),
		CarrierCompleted:    fr.CarrierCompleted(),
		CarrierCompletedAt:  fr.CarrierCompletedAt(),
		CompletedAt:         fr.CompletedAt(),
	}

	// Customer review
	if cr := fr.CustomerReview(); cr != nil {
		expiresAt := cr.EditExpiresAt()
		resp.CustomerReview = &ReviewResponse{
			ID:            cr.ID(),
			Rating:        cr.Rating(),
			Comment:       cr.Comment(),
			CreatedAt:     cr.CreatedAt(),
			CanEdit:       cr.CanEdit(),
			EditExpiresAt: &expiresAt,
		}
	}

	// Carrier review
	if crr := fr.CarrierReview(); crr != nil {
		expiresAt := crr.EditExpiresAt()
		resp.CarrierReview = &ReviewResponse{
			ID:            crr.ID(),
			Rating:        crr.Rating(),
			Comment:       crr.Comment(),
			CreatedAt:     crr.CreatedAt(),
			CanEdit:       crr.CanEdit(),
			EditExpiresAt: &expiresAt,
		}
	}

	return resp
}

// toOfferResponse преобразует оффер в ответ API.
// isOfferOwner определяет, является ли текущий пользователь владельцем оффера (перевозчиком).
// Только владелец оффера видит информацию о сотруднике (member_id, member_name).
// Заказчик видит только информацию об организации.
func (h *FreightRequestHandler) toOfferResponse(offer *entities.Offer, orgName, memberName string, isOfferOwner bool) OfferResponse {
	if offer == nil {
		return OfferResponse{}
	}

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
	r.HandleFunc("/api/v1/freight-requests/{id}/offers/{offerId}/unselect", h.UnselectOffer).Methods(http.MethodPost)
	// Completion & reviews (after offer is confirmed)
	r.HandleFunc("/api/v1/freight-requests/{id}/complete", h.Complete).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/freight-requests/{id}/review", h.LeaveReview).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/freight-requests/{id}/review", h.EditReview).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/freight-requests/{id}/cancel-confirmed", h.CancelAfterConfirmed).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/freight-requests/{id}/reassign-carrier", h.ReassignCarrierMember).Methods(http.MethodPost)
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

	// Validate cargo
	if err := req.Cargo.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate vehicle requirements
	if err := req.VehicleRequirements.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate payment
	if err := req.Payment.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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
	// Преаллокация: ~22 возможных фильтров + pagination
	opts := make([]projections.FilterOption, 0, 25)

	// Опциональный фильтр по customer_org_id
	if orgIDStr := r.URL.Query().Get("customer_org_id"); orgIDStr != "" {
		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid customer_org_id")
			return
		}
		opts = append(opts, projections.WithCustomerOrgID(orgID))
	}

	if memberIDStr := r.URL.Query().Get("member_id"); memberIDStr != "" {
		memberID, err := uuid.Parse(memberIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid member_id")
			return
		}
		opts = append(opts, projections.WithCustomerMemberID(memberID))
	}

	// Опциональный фильтр по статусам (CSV)
	if statuses := r.URL.Query().Get("statuses"); statuses != "" {
		statusList := splitComma(statuses)
		if len(statusList) > 0 {
			opts = append(opts, projections.WithStatuses(statusList))
		}
	}

	if orgName := strings.TrimSpace(r.URL.Query().Get("org_name")); orgName != "" {
		// Limit org_name search to 100 chars to prevent SQL abuse
		if len(orgName) > 100 {
			orgName = orgName[:100]
		}
		opts = append(opts, projections.WithOrgNameLike(orgName))
	}

	if orgINN := r.URL.Query().Get("org_inn"); orgINN != "" {
		opts = append(opts, projections.WithOrgINN(orgINN))
	}

	if orgCountry := r.URL.Query().Get("org_country"); orgCountry != "" {
		opts = append(opts, projections.WithOrgCountry(orgCountry))
	}

	// Request number filter
	if requestNumber := r.URL.Query().Get("request_number"); requestNumber != "" {
		if num, err := parseInt64(requestNumber); err == nil && num > 0 {
			opts = append(opts, projections.WithRequestNumber(num))
		}
	}

	// Extended filters for subscription-like filtering
	if minWeight := r.URL.Query().Get("min_weight"); minWeight != "" {
		if w, err := parseFloat(minWeight); err == nil {
			opts = append(opts, projections.WithMinWeight(w))
		}
	}

	if maxWeight := r.URL.Query().Get("max_weight"); maxWeight != "" {
		if w, err := parseFloat(maxWeight); err == nil {
			opts = append(opts, projections.WithMaxWeight(w))
		}
	}

	if minPrice := r.URL.Query().Get("min_price"); minPrice != "" {
		if p, err := parseInt64(minPrice); err == nil {
			opts = append(opts, projections.WithMinPrice(p))
		}
	}

	if maxPrice := r.URL.Query().Get("max_price"); maxPrice != "" {
		if p, err := parseInt64(maxPrice); err == nil {
			opts = append(opts, projections.WithMaxPrice(p))
		}
	}

	if vehicleType := r.URL.Query().Get("vehicle_type"); vehicleType != "" {
		opts = append(opts, projections.WithVehicleType(vehicleType))
	}

	if vehicleTypes := r.URL.Query().Get("vehicle_types"); vehicleTypes != "" {
		types := splitComma(vehicleTypes)
		if len(types) > 0 {
			opts = append(opts, projections.WithVehicleTypes(types))
		}
	}

	if vehicleSubTypes := r.URL.Query().Get("vehicle_subtypes"); vehicleSubTypes != "" {
		subtypes := splitComma(vehicleSubTypes)
		if len(subtypes) > 0 {
			opts = append(opts, projections.WithVehicleSubTypes(subtypes))
		}
	}

	if routeCityIDs := r.URL.Query().Get("route_city_ids"); routeCityIDs != "" {
		ids := splitCommaInt(routeCityIDs)
		if len(ids) > 0 {
			opts = append(opts, projections.WithRouteCities(ids))
		}
	}

	if routeCountryIDs := r.URL.Query().Get("route_country_ids"); routeCountryIDs != "" {
		ids := splitCommaInt(routeCountryIDs)
		if len(ids) > 0 {
			opts = append(opts, projections.WithRouteCountries(ids))
		}
	}

	// Volume filters
	if minVolume := r.URL.Query().Get("min_volume"); minVolume != "" {
		if v, err := parseFloat(minVolume); err == nil {
			opts = append(opts, projections.WithMinVolume(v))
		}
	}

	if maxVolume := r.URL.Query().Get("max_volume"); maxVolume != "" {
		if v, err := parseFloat(maxVolume); err == nil {
			opts = append(opts, projections.WithMaxVolume(v))
		}
	}

	// Payment filters
	if paymentMethods := r.URL.Query().Get("payment_methods"); paymentMethods != "" {
		methods := splitComma(paymentMethods)
		if len(methods) > 0 {
			opts = append(opts, projections.WithPaymentMethods(methods))
		}
	}

	if paymentTerms := r.URL.Query().Get("payment_terms"); paymentTerms != "" {
		terms := splitComma(paymentTerms)
		if len(terms) > 0 {
			opts = append(opts, projections.WithPaymentTerms(terms))
		}
	}

	if vatTypes := r.URL.Query().Get("vat_types"); vatTypes != "" {
		types := splitComma(vatTypes)
		if len(types) > 0 {
			opts = append(opts, projections.WithVatTypes(types))
		}
	}

	// Cursor-based pagination
	cursorStr := r.URL.Query().Get("cursor")
	if cursorStr != "" {
		cursor, err := httputil.DecodeCursor[projections.FreightRequestCursor](cursorStr)
		if err != nil {
			slog.Warn("invalid cursor, starting from beginning",
				slog.String("cursor", cursorStr),
				slog.String("error", err.Error()))
			// Невалидный cursor — начинаем сначала
		} else {
			opts = append(opts, projections.WithCursor(*cursor))
		}
	}

	// SEC-016: Валидированная пагинация
	pagination := httputil.ParsePagination(r)
	// Запрашиваем limit+1 для определения hasMore
	opts = append(opts, projections.WithLimit(pagination.Limit+1))

	items, err := h.projection.List(r.Context(), opts...)
	if err != nil {
		slog.Error("failed to list freight requests", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list freight requests")
		return
	}

	// Определяем hasMore
	hasMore := len(items) > pagination.Limit
	if hasMore {
		items = items[:pagination.Limit] // убираем лишний элемент
	}

	// Строим nextCursor из последнего элемента
	var nextCursor *string
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		cursorData := projections.FreightRequestCursor{
			IsPublished:   lastItem.Status == "published",
			RequestNumber: lastItem.RequestNumber,
		}
		encoded, err := httputil.EncodeCursor(cursorData)
		if err != nil {
			slog.Error("failed to encode cursor", slog.String("error", err.Error()))
		} else {
			nextCursor = &encoded
		}
	}

	response := FreightRequestListResponse{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *FreightRequestHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Получаем текущую организацию пользователя
	currentOrgID, _ := h.session.GetOrganizationID(r)

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

	// Определяем контекст видимости
	isCustomer := fr.CustomerOrgID() == currentOrgID
	isCarrier := fr.CarrierOrgID() != nil && *fr.CarrierOrgID() == currentOrgID
	isConfirmedState := fr.IsConfirmed() || fr.IsPartiallyCompleted() ||
		fr.IsCompleted() || fr.IsCancelledAfterConfirmed()

	// Собираем ID для загрузки данных
	memberIDs := []uuid.UUID{fr.CustomerMemberID()}
	orgIDs := []uuid.UUID{fr.CustomerOrgID()}

	if fr.CarrierOrgID() != nil {
		orgIDs = append(orgIDs, *fr.CarrierOrgID())
	}
	if fr.CarrierMemberID() != nil {
		memberIDs = append(memberIDs, *fr.CarrierMemberID())
	}

	// Загружаем названия организаций
	orgNames, err := h.orgService.GetNames(r.Context(), orgIDs)
	if err != nil {
		slog.Error("failed to get organization names", slog.String("error", err.Error()))
		orgNames = make(map[uuid.UUID]string)
	}

	// Загружаем имена участников
	memberNames, err := h.membersProjection.GetNames(r.Context(), memberIDs)
	if err != nil {
		slog.Error("failed to get member names", slog.String("error", err.Error()))
		memberNames = make(map[uuid.UUID]string)
	}

	// Формируем базовый ответ
	resp := h.toFreightRequestResponse(fr)
	resp.CustomerOrgName = orgNames[fr.CustomerOrgID()]

	// CustomerMemberName: заказчику всегда, перевозчику при confirmed (другим — никогда)
	if isCustomer || (isCarrier && isConfirmedState) {
		resp.CustomerMemberName = memberNames[fr.CustomerMemberID()]
	}

	// Carrier info: перевозчику всегда, заказчику при confirmed (другим — никогда)
	if (isCarrier || (isCustomer && isConfirmedState)) && fr.CarrierOrgID() != nil {
		resp.CarrierOrgID = fr.CarrierOrgID()
		resp.CarrierOrgName = orgNames[*fr.CarrierOrgID()]
		resp.CarrierMemberID = fr.CarrierMemberID()
		if fr.CarrierMemberID() != nil {
			resp.CarrierMemberName = memberNames[*fr.CarrierMemberID()]
		}
	}

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

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
	role, _ := h.session.GetRole(r)

	if err := h.service.ConfirmOffer(r.Context(), freightrequest.ConfirmOfferInput{
		FreightRequestID: frID,
		OfferID:          offerID,
		ActorMemberID:    memberID,
		ActorOrgID:       orgID,
		ActorRole:        role,
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
	role, _ := h.session.GetRole(r)

	var req DeclineOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.DeclineOffer(r.Context(), freightrequest.DeclineOfferInput{
		FreightRequestID: frID,
		OfferID:          offerID,
		ActorMemberID:    memberID,
		ActorOrgID:       orgID,
		ActorRole:        role,
		Reason:           req.Reason,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type UnselectOfferRequest struct {
	Reason string `json:"reason,omitempty"`
}

func (h *FreightRequestHandler) UnselectOffer(w http.ResponseWriter, r *http.Request) {
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

	var req UnselectOfferRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // reason опционален

	if err := h.service.UnselectOffer(r.Context(), freightrequest.UnselectOfferInput{
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

// Complete marks the freight request as completed from the caller's side (customer or carrier)
func (h *FreightRequestHandler) Complete(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.Complete(r.Context(), freightrequest.CompleteInput{
		FreightRequestID: frID,
		OrgID:            orgID,
		MemberID:         memberID,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type FreightLeaveReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

type FreightLeaveReviewResponse struct {
	ReviewID string `json:"review_id"`
}

// LeaveReview leaves a review for the counterparty after completion
func (h *FreightRequestHandler) LeaveReview(w http.ResponseWriter, r *http.Request) {
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

	var req FreightLeaveReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	reviewID, err := h.service.LeaveReview(r.Context(), freightrequest.LeaveReviewInput{
		FreightRequestID: frID,
		ReviewerOrgID:    orgID,
		ReviewerMemberID: memberID,
		Rating:           req.Rating,
		Comment:          req.Comment,
	})
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, FreightLeaveReviewResponse{ReviewID: reviewID.String()})
}

type EditReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

// EditReview edits an existing review (only within 24h window)
func (h *FreightRequestHandler) EditReview(w http.ResponseWriter, r *http.Request) {
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

	var req EditReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.EditReview(r.Context(), freightrequest.EditReviewInput{
		FreightRequestID: frID,
		ReviewerOrgID:    orgID,
		ReviewerMemberID: memberID,
		Rating:           req.Rating,
		Comment:          req.Comment,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type CancelAfterConfirmedRequest struct {
	Reason string `json:"reason,omitempty"`
}

// CancelAfterConfirmed cancels the freight request after offer was confirmed
func (h *FreightRequestHandler) CancelAfterConfirmed(w http.ResponseWriter, r *http.Request) {
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

	var req CancelAfterConfirmedRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // reason is optional

	if err := h.service.CancelAfterConfirmed(r.Context(), freightrequest.CancelAfterConfirmedInput{
		FreightRequestID: frID,
		OrgID:            orgID,
		MemberID:         memberID,
		Reason:           req.Reason,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ReassignCarrierMemberRequest struct {
	NewMemberID string `json:"new_member_id"`
}

// ReassignCarrierMember reassigns the responsible member for the carrier organization
func (h *FreightRequestHandler) ReassignCarrierMember(w http.ResponseWriter, r *http.Request) {
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

	var req ReassignCarrierMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newMemberID, err := uuid.Parse(req.NewMemberID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid new_member_id")
		return
	}

	if err := h.service.ReassignCarrierMember(r.Context(), freightrequest.ReassignCarrierMemberInput{
		FreightRequestID: frID,
		ActorID:          memberID,
		ActorOrgID:       orgID,
		NewMemberID:      newMemberID,
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
	case errors.Is(err, frDomain.ErrCannotCancelFreightRequest):
		writeError(w, http.StatusConflict, "cannot cancel freight request in current status")
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
	case errors.Is(err, frDomain.ErrNotResponsibleMember):
		writeError(w, http.StatusForbidden, "you are not the responsible member for this freight request")
	case errors.Is(err, frDomain.ErrNotOfferOwner):
		writeError(w, http.StatusForbidden, "not offer owner")
	case errors.Is(err, frDomain.ErrHasSelectedOffer):
		writeError(w, http.StatusConflict, "already has selected offer")
	case errors.Is(err, frDomain.ErrFreightRequestNotSelected):
		writeError(w, http.StatusConflict, "freight request has no selected offer")
	// Completion errors
	case errors.Is(err, frDomain.ErrNotConfirmed):
		writeError(w, http.StatusConflict, "freight request is not confirmed")
	case errors.Is(err, frDomain.ErrAlreadyCompleted):
		writeError(w, http.StatusConflict, "already completed by this party")
	case errors.Is(err, frDomain.ErrCannotCompleteNotParticipant):
		writeError(w, http.StatusForbidden, "not a participant of this freight request")
	// Review errors
	case errors.Is(err, frDomain.ErrCannotLeaveReview):
		writeError(w, http.StatusConflict, "cannot leave review in current state")
	case errors.Is(err, frDomain.ErrAlreadyLeftReview):
		writeError(w, http.StatusConflict, "already left a review")
	case errors.Is(err, frDomain.ErrInvalidRating):
		writeError(w, http.StatusBadRequest, "rating must be between 1 and 5")
	case errors.Is(err, frDomain.ErrCannotEditReview):
		writeError(w, http.StatusForbidden, "cannot edit review")
	case errors.Is(err, frDomain.ErrReviewNotFound):
		writeError(w, http.StatusNotFound, "review not found")
	case errors.Is(err, frDomain.ErrReviewEditWindowExpired):
		writeError(w, http.StatusConflict, "review edit window has expired (24 hours)")
	// Cancel after confirmed errors
	case errors.Is(err, frDomain.ErrCannotCancelAfterConfirmed):
		writeError(w, http.StatusConflict, "cannot cancel in current status")
	case errors.Is(err, orgDomain.ErrMemberNotFound):
		writeError(w, http.StatusNotFound, "member not found")
	case errors.Is(err, orgDomain.ErrInsufficientPermissions):
		writeError(w, http.StatusForbidden, "insufficient permissions")
	default:
		slog.Error("unhandled domain error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// Helper functions for parsing query parameters

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitCommaInt(s string) []int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if n, err := strconv.Atoi(trimmed); err == nil {
			result = append(result, n)
		}
	}
	return result
}


package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	frValues "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// SubscriptionsHandler обрабатывает запросы к подпискам на заявки
type SubscriptionsHandler struct {
	projection *projections.FreightSubscriptionsProjection
	geo        *projections.GeoProjection
	session    *session.Manager
}

// NewSubscriptionsHandler создает новый handler
func NewSubscriptionsHandler(
	projection *projections.FreightSubscriptionsProjection,
	geo *projections.GeoProjection,
	session *session.Manager,
) *SubscriptionsHandler {
	return &SubscriptionsHandler{
		projection: projection,
		geo:        geo,
		session:    session,
	}
}

// RegisterRoutes регистрирует роуты
func (h *SubscriptionsHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/subscriptions", h.List).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/subscriptions", h.Create).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/subscriptions/{id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/subscriptions/{id}", h.Update).Methods(http.MethodPut)
	r.HandleFunc("/api/v1/subscriptions/{id}", h.Delete).Methods(http.MethodDelete)
	r.HandleFunc("/api/v1/subscriptions/{id}/active", h.SetActive).Methods(http.MethodPatch)
}

// SubscriptionRequest запрос на создание/обновление подписки
type SubscriptionRequest struct {
	Name            string                      `json:"name"`
	MinWeight       *float64                    `json:"min_weight,omitempty"`
	MaxWeight       *float64                    `json:"max_weight,omitempty"`
	MinPrice        *int64                      `json:"min_price,omitempty"`
	MaxPrice        *int64                      `json:"max_price,omitempty"`
	MinVolume       *float64                    `json:"min_volume,omitempty"`
	MaxVolume       *float64                    `json:"max_volume,omitempty"`
	VehicleTypes    []string                    `json:"vehicle_types,omitempty"`
	VehicleSubTypes []string                    `json:"vehicle_subtypes,omitempty"`
	PaymentMethods  []string                    `json:"payment_methods,omitempty"`
	PaymentTerms    []string                    `json:"payment_terms,omitempty"`
	VatTypes        []string                    `json:"vat_types,omitempty"`
	RoutePoints     []RoutePointCriteriaRequest `json:"route_points,omitempty"`
	IsActive        bool                        `json:"is_active"`
}

// RoutePointCriteriaRequest запрос точки маршрута
type RoutePointCriteriaRequest struct {
	CountryID int  `json:"country_id"`
	CityID    *int `json:"city_id,omitempty"`
	Order     int  `json:"order"`
}

// FreightSubscriptionResponse ответ с подпиской на заявки
type FreightSubscriptionResponse struct {
	ID              uuid.UUID                    `json:"id"`
	MemberID        uuid.UUID                    `json:"member_id"`
	Name            string                       `json:"name"`
	MinWeight       *float64                     `json:"min_weight,omitempty"`
	MaxWeight       *float64                     `json:"max_weight,omitempty"`
	MinPrice        *int64                       `json:"min_price,omitempty"`
	MaxPrice        *int64                       `json:"max_price,omitempty"`
	MinVolume       *float64                     `json:"min_volume,omitempty"`
	MaxVolume       *float64                     `json:"max_volume,omitempty"`
	VehicleTypes    []string                     `json:"vehicle_types,omitempty"`
	VehicleSubTypes []string                     `json:"vehicle_subtypes,omitempty"`
	PaymentMethods  []string                     `json:"payment_methods,omitempty"`
	PaymentTerms    []string                     `json:"payment_terms,omitempty"`
	VatTypes        []string                     `json:"vat_types,omitempty"`
	RoutePoints     []RoutePointCriteriaResponse `json:"route_points,omitempty"`
	IsActive        bool                         `json:"is_active"`
	CreatedAt       string                       `json:"created_at"`
	UpdatedAt       string                       `json:"updated_at"`
}

// RoutePointCriteriaResponse ответ точки маршрута
type RoutePointCriteriaResponse struct {
	CountryID   int     `json:"country_id"`
	CountryName *string `json:"country_name,omitempty"`
	CityID      *int    `json:"city_id,omitempty"`
	CityName    *string `json:"city_name,omitempty"`
	Order       int     `json:"order"`
}

// SetActiveRequest запрос на изменение статуса активности
type SetActiveRequest struct {
	IsActive bool `json:"is_active"`
}

// List возвращает список подписок пользователя
func (h *SubscriptionsHandler) List(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subscriptions, err := h.projection.GetByMemberID(r.Context(), memberID)
	if err != nil {
		slog.Error("failed to get subscriptions", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Конвертируем в response
	response := make([]FreightSubscriptionResponse, len(subscriptions))
	for i, sub := range subscriptions {
		response[i] = h.subscriptionToResponse(r, sub)
	}

	writeJSON(w, http.StatusOK, response)
}

// Create создает новую подписку
func (h *SubscriptionsHandler) Create(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	criteria := h.requestToCriteria(req)

	subscription, err := h.projection.Create(r.Context(), memberID, criteria)
	if err != nil {
		slog.Error("failed to create subscription",
			slog.String("error", err.Error()),
			slog.String("member_id", memberID.String()),
		)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, h.subscriptionToResponse(r, *subscription))
}

// Get возвращает подписку по ID
func (h *SubscriptionsHandler) Get(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subscriptionID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid subscription id")
		return
	}

	subscription, err := h.projection.GetByID(r.Context(), subscriptionID)
	if err != nil {
		slog.Error("failed to get subscription", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if subscription == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}

	// Проверяем что подписка принадлежит пользователю
	if subscription.MemberID != memberID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	writeJSON(w, http.StatusOK, h.subscriptionToResponse(r, *subscription))
}

// Update обновляет подписку
func (h *SubscriptionsHandler) Update(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subscriptionID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid subscription id")
		return
	}

	var req SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	criteria := h.requestToCriteria(req)

	if err := h.projection.Update(r.Context(), subscriptionID, memberID, criteria); err != nil {
		slog.Error("failed to update subscription",
			slog.String("error", err.Error()),
			slog.String("subscription_id", subscriptionID.String()),
		)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Получаем обновленную подписку
	subscription, err := h.projection.GetByID(r.Context(), subscriptionID)
	if err != nil {
		slog.Error("failed to get updated subscription", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, h.subscriptionToResponse(r, *subscription))
}

// Delete удаляет подписку
func (h *SubscriptionsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subscriptionID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid subscription id")
		return
	}

	if err := h.projection.Delete(r.Context(), subscriptionID, memberID); err != nil {
		slog.Error("failed to delete subscription",
			slog.String("error", err.Error()),
			slog.String("subscription_id", subscriptionID.String()),
		)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetActive включает/выключает подписку
func (h *SubscriptionsHandler) SetActive(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subscriptionID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid subscription id")
		return
	}

	var req SetActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.projection.SetActive(r.Context(), subscriptionID, memberID, req.IsActive); err != nil {
		slog.Error("failed to set subscription active",
			slog.String("error", err.Error()),
			slog.String("subscription_id", subscriptionID.String()),
		)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// requestToCriteria конвертирует request в SubscriptionCriteria
func (h *SubscriptionsHandler) requestToCriteria(req SubscriptionRequest) frValues.SubscriptionCriteria {
	criteria := frValues.SubscriptionCriteria{
		Name:      req.Name,
		MinWeight: req.MinWeight,
		MaxWeight: req.MaxWeight,
		MinPrice:  req.MinPrice,
		MaxPrice:  req.MaxPrice,
		MinVolume: req.MinVolume,
		MaxVolume: req.MaxVolume,
		IsActive:  req.IsActive,
	}

	// Конвертируем ENUM'ы из строк
	for _, vt := range req.VehicleTypes {
		criteria.VehicleTypes = append(criteria.VehicleTypes, frValues.VehicleType(vt))
	}
	for _, vs := range req.VehicleSubTypes {
		criteria.VehicleSubTypes = append(criteria.VehicleSubTypes, frValues.VehicleSubType(vs))
	}
	for _, pm := range req.PaymentMethods {
		criteria.PaymentMethods = append(criteria.PaymentMethods, frValues.PaymentMethod(pm))
	}
	for _, pt := range req.PaymentTerms {
		criteria.PaymentTerms = append(criteria.PaymentTerms, frValues.PaymentTerms(pt))
	}
	for _, vt := range req.VatTypes {
		criteria.VatTypes = append(criteria.VatTypes, frValues.VatType(vt))
	}

	// Конвертируем route points
	for _, rp := range req.RoutePoints {
		criteria.RoutePoints = append(criteria.RoutePoints, frValues.RoutePointCriteria{
			CountryID: rp.CountryID,
			CityID:    rp.CityID,
			Order:     rp.Order,
		})
	}

	return criteria
}

// subscriptionToResponse конвертирует Subscription в FreightSubscriptionResponse
func (h *SubscriptionsHandler) subscriptionToResponse(r *http.Request, sub frValues.Subscription) FreightSubscriptionResponse {
	response := FreightSubscriptionResponse{
		ID:        sub.ID,
		MemberID:  sub.MemberID,
		Name:      sub.Criteria.Name,
		MinWeight: sub.Criteria.MinWeight,
		MaxWeight: sub.Criteria.MaxWeight,
		MinPrice:  sub.Criteria.MinPrice,
		MaxPrice:  sub.Criteria.MaxPrice,
		MinVolume: sub.Criteria.MinVolume,
		MaxVolume: sub.Criteria.MaxVolume,
		IsActive:  sub.Criteria.IsActive,
		CreatedAt: sub.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: sub.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Конвертируем ENUM'ы в строки
	for _, vt := range sub.Criteria.VehicleTypes {
		response.VehicleTypes = append(response.VehicleTypes, string(vt))
	}
	for _, vs := range sub.Criteria.VehicleSubTypes {
		response.VehicleSubTypes = append(response.VehicleSubTypes, string(vs))
	}
	for _, pm := range sub.Criteria.PaymentMethods {
		response.PaymentMethods = append(response.PaymentMethods, string(pm))
	}
	for _, pt := range sub.Criteria.PaymentTerms {
		response.PaymentTerms = append(response.PaymentTerms, string(pt))
	}
	for _, vt := range sub.Criteria.VatTypes {
		response.VatTypes = append(response.VatTypes, string(vt))
	}

	// Конвертируем route points с названиями городов/стран
	for _, rp := range sub.Criteria.RoutePoints {
		rpResponse := RoutePointCriteriaResponse{
			CountryID: rp.CountryID,
			CityID:    rp.CityID,
			Order:     rp.Order,
		}

		// Получаем название страны
		if country, err := h.geo.GetCountry(r.Context(), rp.CountryID); err == nil && country != nil {
			rpResponse.CountryName = country.NameRu
		}

		// Получаем название города
		if rp.CityID != nil {
			if city, err := h.geo.GetCity(r.Context(), *rp.CityID); err == nil && city != nil {
				rpResponse.CityName = city.NameRu
			}
		}

		response.RoutePoints = append(response.RoutePoints, rpResponse)
	}

	return response
}

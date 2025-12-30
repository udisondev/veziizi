package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orderApp "codeberg.org/udison/veziizi/backend/internal/application/order"
	orgApp "codeberg.org/udison/veziizi/backend/internal/application/organization"
	"codeberg.org/udison/veziizi/backend/internal/domain/order"
	"codeberg.org/udison/veziizi/backend/internal/domain/order/entities"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type OrderHandler struct {
	service           *orderApp.Service
	orgService        *orgApp.Service
	membersProjection *projections.MembersProjection
	projection        *projections.OrdersProjection
	session           *session.Manager
}

func NewOrderHandler(
	service *orderApp.Service,
	orgService *orgApp.Service,
	membersProjection *projections.MembersProjection,
	projection *projections.OrdersProjection,
	session *session.Manager,
) *OrderHandler {
	return &OrderHandler{
		service:           service,
		orgService:        orgService,
		membersProjection: membersProjection,
		projection:        projection,
		session:           session,
	}
}

func (h *OrderHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/orders", h.List).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/orders/{id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/orders/{id}/messages", h.SendMessage).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/orders/{id}/documents", h.UploadDocument).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/orders/{id}/documents/{docId}", h.DownloadDocument).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/orders/{id}/documents/{docId}", h.RemoveDocument).Methods(http.MethodDelete)
	r.HandleFunc("/api/v1/orders/{id}/complete", h.Complete).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/orders/{id}/cancel", h.Cancel).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/orders/{id}/review", h.LeaveReview).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/orders/{id}/reassign", h.Reassign).Methods(http.MethodPost)
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	// Получить данные сессии для проверки доступа
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var opts []projections.OrderFilterOption

	// Все сотрудники организации видят все заказы своей организации
	opts = append(opts, projections.OrderWithOrgID(orgID))

	// Дополнительные фильтры (применяются поверх фильтра доступа)
	if customerOrgID := r.URL.Query().Get("customer_org_id"); customerOrgID != "" {
		id, err := uuid.Parse(customerOrgID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid customer_org_id")
			return
		}
		opts = append(opts, projections.OrderWithCustomerOrgID(id))
	}

	if carrierOrgID := r.URL.Query().Get("carrier_org_id"); carrierOrgID != "" {
		id, err := uuid.Parse(carrierOrgID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid carrier_org_id")
			return
		}
		opts = append(opts, projections.OrderWithCarrierOrgID(id))
	}

	if frID := r.URL.Query().Get("freight_request_id"); frID != "" {
		id, err := uuid.Parse(frID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid freight_request_id")
			return
		}
		opts = append(opts, projections.OrderWithFreightRequestID(id))
	}

	if status := r.URL.Query().Get("status"); status != "" {
		opts = append(opts, projections.OrderWithStatus(status))
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 100 {
			writeError(w, http.StatusBadRequest, "invalid limit (1-100)")
			return
		}
		opts = append(opts, projections.OrderWithLimit(limit))
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		opts = append(opts, projections.OrderWithOffset(offset))
	}

	items, err := h.projection.List(r.Context(), opts...)
	if err != nil {
		slog.Error("failed to list orders", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

// OrderResponse represents full order data loaded from event store
type OrderResponse struct {
	ID               uuid.UUID          `json:"id"`
	OrderNumber      int64              `json:"order_number"`
	FreightRequestID uuid.UUID          `json:"freight_request_id"`
	OfferID          uuid.UUID          `json:"offer_id"`
	CustomerOrgID    uuid.UUID          `json:"customer_org_id"`
	CustomerOrgName  string             `json:"customer_org_name"`
	CustomerMemberID uuid.UUID          `json:"customer_member_id"`
	CustomerMemberName string           `json:"customer_member_name"`
	CarrierOrgID     uuid.UUID          `json:"carrier_org_id"`
	CarrierOrgName   string             `json:"carrier_org_name"`
	CarrierMemberID  uuid.UUID          `json:"carrier_member_id"`
	CarrierMemberName string            `json:"carrier_member_name"`
	Status           string             `json:"status"`
	Messages         []MessageResponse  `json:"messages"`
	Documents        []DocumentResponse `json:"documents"`
	Reviews          []ReviewResponse   `json:"reviews"`
	CreatedAt        time.Time          `json:"created_at"`
	CompletedAt      *time.Time         `json:"completed_at,omitempty"`
	CancelledAt      *time.Time         `json:"cancelled_at,omitempty"`
}

type MessageResponse struct {
	ID             uuid.UUID `json:"id"`
	SenderOrgID    uuid.UUID `json:"sender_org_id"`
	SenderMemberID uuid.UUID `json:"sender_member_id"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

type DocumentResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	MimeType   string    `json:"mime_type"`
	Size       int64     `json:"size"`
	UploadedBy uuid.UUID `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type ReviewResponse struct {
	ID            uuid.UUID `json:"id"`
	ReviewerOrgID uuid.UUID `json:"reviewer_org_id"`
	Rating        int       `json:"rating"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Получить данные сессии для проверки доступа
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	o, err := h.service.Get(r.Context(), id)
	if err != nil {
		slog.Error("failed to get order", slog.String("error", err.Error()))
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	// Проверить доступ - все сотрудники организации-участника могут видеть заказ
	if !o.CanAccess(orgID) {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	resp := orderToResponse(o)

	// Fallback: load order_number from lookup for old events without it
	if resp.OrderNumber == 0 {
		if lookup, err := h.projection.GetByID(r.Context(), id); err != nil {
			slog.Error("failed to get order lookup", slog.String("error", err.Error()))
		} else {
			resp.OrderNumber = lookup.OrderNumber
		}
	}

	// Load organization names
	orgNames, err := h.orgService.GetNames(r.Context(), []uuid.UUID{o.CustomerOrgID(), o.CarrierOrgID()})
	if err != nil {
		slog.Error("failed to get organization names", slog.String("error", err.Error()))
	} else {
		resp.CustomerOrgName = orgNames[o.CustomerOrgID()]
		resp.CarrierOrgName = orgNames[o.CarrierOrgID()]
	}

	// Load member names
	if customerMember, err := h.membersProjection.GetByID(r.Context(), o.CustomerMemberID()); err != nil {
		slog.Error("failed to get customer member", slog.String("error", err.Error()))
	} else {
		resp.CustomerMemberName = customerMember.Name
	}

	if carrierMember, err := h.membersProjection.GetByID(r.Context(), o.CarrierMemberID()); err != nil {
		slog.Error("failed to get carrier member", slog.String("error", err.Error()))
	} else {
		resp.CarrierMemberName = carrierMember.Name
	}

	writeJSON(w, http.StatusOK, resp)
}

func orderToResponse(o *order.Order) OrderResponse {
	msgs := o.Messages()
	messages := make([]MessageResponse, 0, len(msgs))
	for _, m := range msgs {
		messages = append(messages, messageToResponse(m))
	}

	docs := o.Documents()
	documents := make([]DocumentResponse, 0, len(docs))
	for _, d := range docs {
		documents = append(documents, documentToResponse(d))
	}

	revs := o.Reviews()
	reviews := make([]ReviewResponse, 0, len(revs))
	for _, rv := range revs {
		reviews = append(reviews, reviewToResponse(rv))
	}

	return OrderResponse{
		ID:               o.ID(),
		OrderNumber:      o.OrderNumber(),
		FreightRequestID: o.FreightRequestID(),
		OfferID:          o.OfferID(),
		CustomerOrgID:    o.CustomerOrgID(),
		CustomerMemberID: o.CustomerMemberID(),
		CarrierOrgID:     o.CarrierOrgID(),
		CarrierMemberID:  o.CarrierMemberID(),
		Status:           o.Status().String(),
		Messages:         messages,
		Documents:        documents,
		Reviews:          reviews,
		CreatedAt:        o.CreatedAt(),
		CompletedAt:      o.CompletedAt(),
		CancelledAt:      o.CancelledAt(),
	}
}

func messageToResponse(m *entities.Message) MessageResponse {
	return MessageResponse{
		ID:             m.ID(),
		SenderOrgID:    m.SenderOrgID(),
		SenderMemberID: m.SenderMemberID(),
		Content:        m.Content(),
		CreatedAt:      m.CreatedAt(),
	}
}

func documentToResponse(d *entities.Document) DocumentResponse {
	return DocumentResponse{
		ID:         d.ID(),
		Name:       d.Name(),
		MimeType:   d.MimeType(),
		Size:       d.Size(),
		UploadedBy: d.UploadedBy(),
		CreatedAt:  d.CreatedAt(),
	}
}

func reviewToResponse(rv *entities.Review) ReviewResponse {
	return ReviewResponse{
		ID:            rv.ID(),
		ReviewerOrgID: rv.ReviewerOrgID(),
		Rating:        rv.Rating(),
		Comment:       rv.Comment(),
		CreatedAt:     rv.CreatedAt(),
	}
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

func (h *OrderHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
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

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.SendMessage(r.Context(), orderApp.SendMessageInput{
		OrderID:        orderID,
		SenderOrgID:    orgID,
		SenderMemberID: memberID,
		Content:        req.Content,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
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

	// Max 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("failed to close uploaded file", slog.String("error", err.Error()))
		}
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		slog.Error("failed to read file", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	// SEC-004: Санитизация имени файла для предотвращения path traversal
	filename := filepath.Base(header.Filename)
	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." || filename == ".." {
		writeError(w, http.StatusBadRequest, "invalid filename")
		return
	}
	// Ограничить длину имени файла
	if len(filename) > 255 {
		filename = filename[:255]
	}

	if err := h.service.AttachDocument(r.Context(), orderApp.AttachDocumentInput{
		OrderID:          orderID,
		UploaderOrgID:    orgID,
		UploaderMemberID: memberID,
		Name:             filename,
		Data:             data,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	docID, err := uuid.Parse(vars["docId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}

	// SEC-002: Проверка авторизации перед загрузкой документа
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Загрузить заказ и проверить доступ - все сотрудники организации-участника могут скачивать
	order, err := h.service.Get(r.Context(), orderID)
	if err != nil {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if !order.CanAccess(orgID) {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	data, mimeType, err := h.service.GetDocumentFile(r.Context(), orderID, docID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		slog.Error("failed to write response", slog.String("error", err.Error()))
	}
}

func (h *OrderHandler) RemoveDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	docID, err := uuid.Parse(vars["docId"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document id")
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

	if err := h.service.RemoveDocument(r.Context(), orderApp.RemoveDocumentInput{
		OrderID:         orderID,
		DocumentID:      docID,
		RemoverOrgID:    orgID,
		RemoverMemberID: memberID,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandler) Complete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
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

	if err := h.service.Complete(r.Context(), orderApp.CompleteInput{
		OrderID:  orderID,
		OrgID:    orgID,
		MemberID: memberID,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type CancelOrderRequest struct {
	Reason string `json:"reason,omitempty"`
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
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

	var req CancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Cancel(r.Context(), orderApp.CancelInput{
		OrderID:  orderID,
		OrgID:    orgID,
		MemberID: memberID,
		Reason:   req.Reason,
	}); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type LeaveReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

func (h *OrderHandler) LeaveReview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
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

	var req LeaveReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.LeaveReview(r.Context(), orderApp.LeaveReviewInput{
		OrderID:          orderID,
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

type ReassignOrderRequest struct {
	NewMemberID string `json:"new_member_id"`
}

func (h *OrderHandler) Reassign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
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

	var req ReassignOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newMemberID, err := uuid.Parse(req.NewMemberID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid new_member_id")
		return
	}

	// Определяем какой метод вызывать в зависимости от организации актора
	o, err := h.service.Get(r.Context(), orderID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	input := orderApp.ReassignMemberInput{
		OrderID:     orderID,
		ActorID:     memberID,
		ActorOrgID:  orgID,
		NewMemberID: newMemberID,
	}

	if o.CustomerOrgID() == orgID {
		err = h.service.ReassignCustomerMember(r.Context(), input)
	} else if o.CarrierOrgID() == orgID {
		err = h.service.ReassignCarrierMember(r.Context(), input)
	} else {
		writeError(w, http.StatusForbidden, "not an order participant")
		return
	}

	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		writeError(w, http.StatusNotFound, "order not found")
	case errors.Is(err, order.ErrOrderCancelled):
		writeError(w, http.StatusConflict, "order is cancelled")
	case errors.Is(err, order.ErrOrderCompleted):
		writeError(w, http.StatusConflict, "order is completed")
	case errors.Is(err, order.ErrOrderNotActive):
		writeError(w, http.StatusConflict, "order is not active")
	case errors.Is(err, order.ErrNotOrderParticipant):
		writeError(w, http.StatusForbidden, "not an order participant")
	case errors.Is(err, order.ErrNotResponsibleMember):
		writeError(w, http.StatusForbidden, "you are not the responsible member for this order")
	case errors.Is(err, order.ErrAlreadyCompleted):
		writeError(w, http.StatusConflict, "already marked as completed")
	case errors.Is(err, order.ErrCannotCancelAfterComplete):
		writeError(w, http.StatusConflict, "cannot cancel after completion started")
	case errors.Is(err, order.ErrCannotLeaveReview):
		writeError(w, http.StatusConflict, "can only leave review after order is finished")
	case errors.Is(err, order.ErrAlreadyLeftReview):
		writeError(w, http.StatusConflict, "already left a review")
	case errors.Is(err, order.ErrInvalidRating):
		writeError(w, http.StatusBadRequest, "rating must be between 1 and 5")
	case errors.Is(err, order.ErrDocumentNotFound):
		writeError(w, http.StatusNotFound, "document not found")
	case errors.Is(err, order.ErrNotDocumentOwner):
		writeError(w, http.StatusForbidden, "not document owner")
	case errors.Is(err, order.ErrEmptyMessage):
		writeError(w, http.StatusBadRequest, "message content is empty")
	case errors.Is(err, order.ErrCannotReassignFinishedOrder):
		writeError(w, http.StatusConflict, "cannot reassign finished order")
	case errors.Is(err, organization.ErrMemberNotFound):
		writeError(w, http.StatusNotFound, "member not found")
	case errors.Is(err, organization.ErrInsufficientPermissions):
		writeError(w, http.StatusForbidden, "insufficient permissions")
	default:
		slog.Error("unhandled domain error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

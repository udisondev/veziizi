package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	supportApp "codeberg.org/udison/veziizi/backend/internal/application/support"
	"codeberg.org/udison/veziizi/backend/internal/domain/support"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type SupportHandler struct {
	service    *supportApp.Service
	projection *projections.SupportTicketsProjection
	session    *session.Manager
}

func NewSupportHandler(
	service *supportApp.Service,
	projection *projections.SupportTicketsProjection,
	session *session.Manager,
) *SupportHandler {
	return &SupportHandler{
		service:    service,
		projection: projection,
		session:    session,
	}
}

func (h *SupportHandler) RegisterRoutes(r *mux.Router) {
	// User routes
	r.HandleFunc("/api/v1/support/faq", h.GetFAQ).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/support/tickets", h.CreateTicket).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/support/tickets", h.ListMyTickets).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/support/tickets/{id}", h.GetTicket).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/support/tickets/{id}/messages", h.AddMessage).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/support/tickets/{id}/reopen", h.ReopenTicket).Methods(http.MethodPost)
}

// FAQItem represents a FAQ question and answer
type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Category string `json:"category"`
}

// GetFAQ returns static FAQ content
func (h *SupportHandler) GetFAQ(w http.ResponseWriter, r *http.Request) {
	faq := []FAQItem{
		{
			Question: "Как создать заявку на перевозку?",
			Answer:   "На главной странице нажмите кнопку \"Создать заявку\" и заполните форму с маршрутом, типом груза и условиями оплаты.",
			Category: "Заявки",
		},
		{
			Question: "Как добавить сотрудника в организацию?",
			Answer:   "Перейдите в раздел \"Штат\" и нажмите \"Пригласить сотрудника\". Введите email нового сотрудника и выберите его роль.",
			Category: "Организация",
		},
		{
			Question: "Как оставить отзыв о перевозчике?",
			Answer:   "После завершения заказа вы можете оставить отзыв на странице заказа. Нажмите \"Оставить отзыв\" и поставьте оценку.",
			Category: "Отзывы",
		},
		{
			Question: "Как подключить Telegram уведомления?",
			Answer:   "Перейдите в \"Настройки уведомлений\" и нажмите \"Подключить Telegram\". Следуйте инструкциям бота.",
			Category: "Уведомления",
		},
		{
			Question: "Что делать если заказ был отменён?",
			Answer:   "Если заказ был отменён другой стороной, вы можете создать новую заявку или выбрать другое предложение из списка.",
			Category: "Заказы",
		},
	}

	writeJSON(w, http.StatusOK, faq)
}

// CreateTicketRequest represents the request to create a new ticket
type CreateTicketRequest struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// CreateTicket creates a new support ticket
func (h *SupportHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
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

	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ticketID, err := h.service.CreateTicket(r.Context(), supportApp.CreateTicketInput{
		MemberID:       memberID,
		OrgID:          orgID,
		Subject:        req.Subject,
		InitialMessage: req.Message,
	})
	if err != nil {
		switch {
		case errors.Is(err, support.ErrEmptySubject):
			writeError(w, http.StatusBadRequest, "subject is required")
		case errors.Is(err, support.ErrSubjectTooLong):
			writeError(w, http.StatusBadRequest, "subject is too long (max 255 characters)")
		case errors.Is(err, support.ErrEmptyMessage):
			writeError(w, http.StatusBadRequest, "message is required")
		case errors.Is(err, support.ErrMessageTooLong):
			writeError(w, http.StatusBadRequest, "message is too long (max 10000 characters)")
		default:
			slog.Error("failed to create ticket", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "failed to create ticket")
		}
		return
	}

	// Load the created ticket to return full response
	ticket, err := h.service.Get(r.Context(), ticketID)
	if err != nil {
		slog.Error("failed to load created ticket", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to load ticket")
		return
	}

	resp := h.buildTicketResponse(ticket)
	writeJSON(w, http.StatusCreated, resp)
}

// ListMyTickets returns the user's tickets
func (h *SupportHandler) ListMyTickets(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	opts := []projections.TicketFilterOption{
		projections.TicketWithMemberID(memberID),
	}

	// Status filter
	if status := r.URL.Query().Get("status"); status != "" {
		if status == "open" {
			opts = append(opts, projections.TicketWithOpenStatus())
		} else {
			opts = append(opts, projections.TicketWithStatus(status))
		}
	}

	// Pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 100 {
			writeError(w, http.StatusBadRequest, "invalid limit (1-100)")
			return
		}
		opts = append(opts, projections.TicketWithLimit(limit))
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		opts = append(opts, projections.TicketWithOffset(offset))
	}

	items, err := h.projection.List(r.Context(), opts...)
	if err != nil {
		slog.Error("failed to list tickets", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list tickets")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

// TicketDetailResponse represents full ticket details
type TicketDetailResponse struct {
	ID           uuid.UUID               `json:"id"`
	TicketNumber int64                   `json:"ticket_number"`
	Subject      string                  `json:"subject"`
	Status       string                  `json:"status"`
	Messages     []TicketMessageResponse `json:"messages"`
	CreatedAt    string                  `json:"created_at"`
	UpdatedAt    string                  `json:"updated_at"`
	ClosedAt     *string                 `json:"closed_at,omitempty"`
}

type TicketMessageResponse struct {
	ID         uuid.UUID `json:"id"`
	SenderType string    `json:"sender_type"`
	SenderID   uuid.UUID `json:"sender_id"`
	Content    string    `json:"content"`
	CreatedAt  string    `json:"created_at"`
}

// GetTicket returns ticket details with messages
func (h *SupportHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	ticketID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	// Load full ticket from event store
	ticket, err := h.service.Get(r.Context(), ticketID)
	if err != nil {
		if errors.Is(err, support.ErrTicketNotFound) {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		slog.Error("failed to get ticket", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to get ticket")
		return
	}

	// Check access
	if !ticket.CanUserAccess(memberID) {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	// Build response
	resp := h.buildTicketResponse(ticket)
	writeJSON(w, http.StatusOK, resp)
}

// AddMessageRequest represents the request to add a message
type AddMessageRequest struct {
	Content string `json:"content"`
}

// AddMessage adds a message to the ticket
func (h *SupportHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	ticketID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req AddMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.AddUserMessage(r.Context(), supportApp.AddUserMessageInput{
		TicketID: ticketID,
		MemberID: memberID,
		Content:  req.Content,
	})
	if err != nil {
		switch {
		case errors.Is(err, support.ErrTicketNotFound):
			writeError(w, http.StatusNotFound, "ticket not found")
		case errors.Is(err, support.ErrNotTicketOwner):
			writeError(w, http.StatusForbidden, "access denied")
		case errors.Is(err, support.ErrTicketClosed):
			writeError(w, http.StatusConflict, "ticket is closed")
		case errors.Is(err, support.ErrEmptyMessage):
			writeError(w, http.StatusBadRequest, "message is required")
		case errors.Is(err, support.ErrMessageTooLong):
			writeError(w, http.StatusBadRequest, "message is too long")
		default:
			slog.Error("failed to add message", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "failed to add message")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReopenTicket reopens a closed ticket
func (h *SupportHandler) ReopenTicket(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	ticketID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	err = h.service.ReopenTicket(r.Context(), ticketID, memberID)
	if err != nil {
		switch {
		case errors.Is(err, support.ErrTicketNotFound):
			writeError(w, http.StatusNotFound, "ticket not found")
		case errors.Is(err, support.ErrNotTicketOwner):
			writeError(w, http.StatusForbidden, "access denied")
		case errors.Is(err, support.ErrTicketNotClosed):
			writeError(w, http.StatusConflict, "ticket is not closed")
		default:
			slog.Error("failed to reopen ticket", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "failed to reopen ticket")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SupportHandler) buildTicketResponse(ticket *support.Ticket) TicketDetailResponse {
	// Build messages
	messages := make([]TicketMessageResponse, 0, len(ticket.Messages()))
	for _, msg := range ticket.Messages() {
		messages = append(messages, TicketMessageResponse{
			ID:         msg.ID(),
			SenderType: string(msg.SenderType()),
			SenderID:   msg.SenderID(),
			Content:    msg.Content(),
			CreatedAt:  msg.CreatedAt().Format(time.RFC3339),
		})
	}

	// Sort messages by created_at
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].CreatedAt < messages[j].CreatedAt
	})

	resp := TicketDetailResponse{
		ID:           ticket.ID(),
		TicketNumber: ticket.TicketNumber(),
		Subject:      ticket.Subject(),
		Status:       string(ticket.Status()),
		Messages:     messages,
		CreatedAt:    ticket.CreatedAt().Format(time.RFC3339),
		UpdatedAt:    ticket.UpdatedAt().Format(time.RFC3339),
	}

	if ticket.ClosedAt() != nil {
		closedAt := ticket.ClosedAt().Format(time.RFC3339)
		resp.ClosedAt = &closedAt
	}

	return resp
}

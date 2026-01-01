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

type AdminSupportHandler struct {
	service    *supportApp.Service
	projection *projections.SupportTicketsProjection
	session    *session.AdminManager
}

func NewAdminSupportHandler(
	service *supportApp.Service,
	projection *projections.SupportTicketsProjection,
	session *session.AdminManager,
) *AdminSupportHandler {
	return &AdminSupportHandler{
		service:    service,
		projection: projection,
		session:    session,
	}
}

func (h *AdminSupportHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/admin/support/tickets", h.ListTickets).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/admin/support/tickets/{id}", h.GetTicket).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/admin/support/tickets/{id}/messages", h.AddMessage).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/support/tickets/{id}/close", h.CloseTicket).Methods(http.MethodPost)
}

// ListTickets returns all tickets for admin
func (h *AdminSupportHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	_, ok := h.session.GetAdminID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	status := r.URL.Query().Get("status")
	limit := 20
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

	tickets, total, err := h.projection.ListForAdmin(r.Context(), status, limit, offset)
	if err != nil {
		slog.Error("failed to list tickets", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list tickets")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tickets": tickets,
		"total":   total,
	})
}

// AdminTicketDetailResponse represents full ticket details for admin
type AdminTicketDetailResponse struct {
	ID           uuid.UUID               `json:"id"`
	TicketNumber int64                   `json:"ticket_number"`
	MemberID     uuid.UUID               `json:"member_id"`
	OrgID        uuid.UUID               `json:"org_id"`
	Subject      string                  `json:"subject"`
	Status       string                  `json:"status"`
	Messages     []TicketMessageResponse `json:"messages"`
	CreatedAt    string                  `json:"created_at"`
	UpdatedAt    string                  `json:"updated_at"`
	ClosedAt     *string                 `json:"closed_at,omitempty"`
}

// GetTicket returns ticket details with messages for admin
func (h *AdminSupportHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	_, ok := h.session.GetAdminID(r)
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

	// Build response
	resp := h.buildTicketResponse(ticket)
	writeJSON(w, http.StatusOK, resp)
}

// AdminAddMessageRequest represents the request to add an admin message
type AdminAddMessageRequest struct {
	Content string `json:"content"`
}

// AddMessage adds an admin message to the ticket
func (h *AdminSupportHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
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

	var req AdminAddMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.AddAdminMessage(r.Context(), supportApp.AddAdminMessageInput{
		TicketID: ticketID,
		AdminID:  adminID,
		Content:  req.Content,
	})
	if err != nil {
		switch {
		case errors.Is(err, support.ErrTicketNotFound):
			writeError(w, http.StatusNotFound, "ticket not found")
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

// CloseTicketRequest represents the request to close a ticket
type CloseTicketRequest struct {
	Resolution string `json:"resolution,omitempty"`
}

// CloseTicket closes the ticket
func (h *AdminSupportHandler) CloseTicket(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.session.GetAdminID(r)
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

	var req CloseTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = CloseTicketRequest{}
	}

	err = h.service.CloseTicket(r.Context(), supportApp.CloseTicketInput{
		TicketID:   ticketID,
		AdminID:    adminID,
		Resolution: req.Resolution,
	})
	if err != nil {
		switch {
		case errors.Is(err, support.ErrTicketNotFound):
			writeError(w, http.StatusNotFound, "ticket not found")
		case errors.Is(err, support.ErrTicketClosed):
			writeError(w, http.StatusConflict, "ticket is already closed")
		default:
			slog.Error("failed to close ticket", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "failed to close ticket")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminSupportHandler) buildTicketResponse(ticket *support.Ticket) AdminTicketDetailResponse {
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

	resp := AdminTicketDetailResponse{
		ID:           ticket.ID(),
		TicketNumber: ticket.TicketNumber(),
		MemberID:     ticket.MemberID(),
		OrgID:        ticket.OrgID(),
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

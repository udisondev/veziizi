package handlers

import (
	"net/http"
	"strconv"

	frApp "github.com/udisondev/veziizi/backend/internal/application/freightrequest"
	historyApp "github.com/udisondev/veziizi/backend/internal/application/history"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	defaultHistoryLimit = 20
	maxHistoryLimit     = 100
)

type HistoryHandler struct {
	historyService *historyApp.Service
	frService      *frApp.Service
	session        *session.Manager
}

func NewHistoryHandler(
	historyService *historyApp.Service,
	frService *frApp.Service,
	session *session.Manager,
) *HistoryHandler {
	return &HistoryHandler{
		historyService: historyService,
		frService:      frService,
		session:        session,
	}
}

func (h *HistoryHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/organizations/{id}/history", h.GetOrganizationHistory).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/freight-requests/{id}/history", h.GetFreightRequestHistory).Methods(http.MethodGet)
}

// GetOrganizationHistory returns event history for an organization
func (h *HistoryHandler) GetOrganizationHistory(w http.ResponseWriter, r *http.Request) {
	// Check role
	if !h.isOwnerOrAdmin(r) {
		writeError(w, http.StatusForbidden, "доступ запрещён: требуется роль владельца или администратора")
		return
	}

	// Get organization ID from URL
	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "неверный формат ID организации")
		return
	}

	// Check that user belongs to this organization
	sessionOrgID, ok := h.session.GetOrganizationID(r)
	if !ok || sessionOrgID != orgID {
		writeError(w, http.StatusForbidden, "доступ запрещён: вы не принадлежите к этой организации")
		return
	}

	// Parse pagination params
	limit, offset := h.parsePagination(r)

	// Get history
	page, err := h.historyService.GetDisplayableHistory(r.Context(), orgID, "organization", limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ошибка получения истории")
		return
	}

	writeJSON(w, http.StatusOK, page)
}

// GetFreightRequestHistory returns event history for a freight request
func (h *HistoryHandler) GetFreightRequestHistory(w http.ResponseWriter, r *http.Request) {
	// Check role
	if !h.isOwnerOrAdmin(r) {
		writeError(w, http.StatusForbidden, "доступ запрещён: требуется роль владельца или администратора")
		return
	}

	// Get freight request ID from URL
	vars := mux.Vars(r)
	frID, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "неверный формат ID заявки")
		return
	}

	// Get freight request to check ownership
	fr, err := h.frService.Get(r.Context(), frID)
	if err != nil {
		writeError(w, http.StatusNotFound, "заявка не найдена")
		return
	}

	// Check that user's organization is the customer
	sessionOrgID, ok := h.session.GetOrganizationID(r)
	if !ok || sessionOrgID != fr.CustomerOrgID() {
		writeError(w, http.StatusForbidden, "доступ запрещён: вы не являетесь владельцем заявки")
		return
	}

	// Parse pagination params
	limit, offset := h.parsePagination(r)

	// Get history
	page, err := h.historyService.GetDisplayableHistory(r.Context(), frID, "freight_request", limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ошибка получения истории")
		return
	}

	writeJSON(w, http.StatusOK, page)
}

func (h *HistoryHandler) isOwnerOrAdmin(r *http.Request) bool {
	role, ok := h.session.GetRole(r)
	if !ok {
		return false
	}
	return role == "owner" || role == "administrator"
}

func (h *HistoryHandler) parsePagination(r *http.Request) (limit, offset int) {
	limit = defaultHistoryLimit
	offset = 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > maxHistoryLimit {
				limit = maxHistoryLimit
			}
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}


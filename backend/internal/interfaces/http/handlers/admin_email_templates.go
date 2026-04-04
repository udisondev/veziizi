package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/google/uuid"
	"github.com/go-chi/chi/v5"
)

// AdminEmailTemplatesHandler handles admin email templates operations
type AdminEmailTemplatesHandler struct {
	projection *projections.EmailTemplatesProjection
	session    *session.AdminManager
}

// NewAdminEmailTemplatesHandler creates a new handler
func NewAdminEmailTemplatesHandler(
	projection *projections.EmailTemplatesProjection,
	session *session.AdminManager,
) *AdminEmailTemplatesHandler {
	return &AdminEmailTemplatesHandler{
		projection: projection,
		session:    session,
	}
}

// RegisterRoutes registers admin email templates routes
func (h *AdminEmailTemplatesHandler) RegisterRoutes(r chi.Router) {
	r.Get("/email-templates", h.List)
	r.Post("/email-templates", h.Create)
	r.Post("/email-templates/preview", h.Preview)
	r.Get("/email-templates/{id}", h.Get)
	r.Patch("/email-templates/{id}", h.Update)
	r.Delete("/email-templates/{id}", h.Delete)
}

// EmailTemplateResponse represents a single email template
type EmailTemplateResponse struct {
	ID              string                              `json:"id"`
	Slug            string                              `json:"slug"`
	Name            string                              `json:"name"`
	Subject         string                              `json:"subject"`
	BodyHTML        string                              `json:"body_html"`
	BodyText        string                              `json:"body_text"`
	Category        string                              `json:"category"`
	VariablesSchema map[string]projections.VariableSpec `json:"variables_schema"`
	IsSystem        bool                                `json:"is_system"`
	IsActive        bool                                `json:"is_active"`
	CreatedAt       string                              `json:"created_at"`
	UpdatedAt       string                              `json:"updated_at"`
}

// EmailTemplatesListResponse represents list of email templates with total
type EmailTemplatesListResponse struct {
	Templates []EmailTemplateResponse `json:"templates"`
	Total     int                     `json:"total"`
}

// List returns list of email templates with filters
func (h *AdminEmailTemplatesHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	filter := projections.EmailTemplateListFilter{
		Limit:  50,
		Offset: 0,
	}

	// Parse query parameters
	if category := r.URL.Query().Get("category"); category != "" {
		filter.Category = &category
	}

	if isActiveStr := r.URL.Query().Get("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filter.IsActive = &isActive
		}
	}

	if isSystemStr := r.URL.Query().Get("is_system"); isSystemStr != "" {
		if isSystem, err := strconv.ParseBool(isSystemStr); err == nil {
			filter.IsSystem = &isSystem
		}
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.SearchText = search
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	templates, err := h.projection.List(r.Context(), filter)
	if err != nil {
		slog.Error("list email templates failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list templates")
		return
	}

	total, err := h.projection.Count(r.Context(), filter)
	if err != nil {
		slog.Error("count email templates failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to count templates")
		return
	}

	response := EmailTemplatesListResponse{
		Templates: make([]EmailTemplateResponse, 0, len(templates)),
		Total:     total,
	}

	for _, tpl := range templates {
		varsSchema, err := tpl.ParseVariablesSchema()
		if err != nil {
			slog.Error("parse variables schema failed",
				slog.String("template_id", tpl.ID.String()),
				slog.String("error", err.Error()))
			varsSchema = make(map[string]projections.VariableSpec)
		}

		response.Templates = append(response.Templates, EmailTemplateResponse{
			ID:              tpl.ID.String(),
			Slug:            tpl.Slug,
			Name:            tpl.Name,
			Subject:         tpl.Subject,
			BodyHTML:        tpl.BodyHTML,
			BodyText:        tpl.BodyText,
			Category:        tpl.Category,
			VariablesSchema: varsSchema,
			IsSystem:        tpl.IsSystem,
			IsActive:        tpl.IsActive,
			CreatedAt:       tpl.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       tpl.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// Get returns a single email template by ID
func (h *AdminEmailTemplatesHandler) Get(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template id")
		return
	}

	tpl, err := h.projection.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("get email template failed",
			slog.String("template_id", id.String()),
			slog.String("error", err.Error()))
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	varsSchema, err := tpl.ParseVariablesSchema()
	if err != nil {
		slog.Error("parse variables schema failed",
			slog.String("template_id", tpl.ID.String()),
			slog.String("error", err.Error()))
		varsSchema = make(map[string]projections.VariableSpec)
	}

	writeJSON(w, http.StatusOK, EmailTemplateResponse{
		ID:              tpl.ID.String(),
		Slug:            tpl.Slug,
		Name:            tpl.Name,
		Subject:         tpl.Subject,
		BodyHTML:        tpl.BodyHTML,
		BodyText:        tpl.BodyText,
		Category:        tpl.Category,
		VariablesSchema: varsSchema,
		IsSystem:        tpl.IsSystem,
		IsActive:        tpl.IsActive,
		CreatedAt:       tpl.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       tpl.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// CreateEmailTemplateRequest represents the request to create a new template
type CreateEmailTemplateRequest struct {
	Slug            string                              `json:"slug"`
	Name            string                              `json:"name"`
	Subject         string                              `json:"subject"`
	BodyHTML        string                              `json:"body_html"`
	BodyText        string                              `json:"body_text"`
	Category        string                              `json:"category"`
	VariablesSchema map[string]projections.VariableSpec `json:"variables_schema"`
	IsSystem        bool                                `json:"is_system"`
}

// Create creates a new email template
func (h *AdminEmailTemplatesHandler) Create(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateEmailTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Slug == "" {
		writeError(w, http.StatusBadRequest, "slug is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Subject == "" {
		writeError(w, http.StatusBadRequest, "subject is required")
		return
	}
	if req.BodyHTML == "" {
		writeError(w, http.StatusBadRequest, "body_html is required")
		return
	}
	if req.Category != "transactional" && req.Category != "marketing" {
		writeError(w, http.StatusBadRequest, "category must be 'transactional' or 'marketing'")
		return
	}

	if req.VariablesSchema == nil {
		req.VariablesSchema = make(map[string]projections.VariableSpec)
	}

	tpl, err := h.projection.Create(r.Context(), projections.CreateEmailTemplateInput{
		Slug:            req.Slug,
		Name:            req.Name,
		Subject:         req.Subject,
		BodyHTML:        req.BodyHTML,
		BodyText:        req.BodyText,
		Category:        req.Category,
		VariablesSchema: req.VariablesSchema,
		IsSystem:        req.IsSystem,
	})
	if err != nil {
		slog.Error("create email template failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to create template")
		return
	}

	varsSchema, _ := tpl.ParseVariablesSchema()

	writeJSON(w, http.StatusCreated, EmailTemplateResponse{
		ID:              tpl.ID.String(),
		Slug:            tpl.Slug,
		Name:            tpl.Name,
		Subject:         tpl.Subject,
		BodyHTML:        tpl.BodyHTML,
		BodyText:        tpl.BodyText,
		Category:        tpl.Category,
		VariablesSchema: varsSchema,
		IsSystem:        tpl.IsSystem,
		IsActive:        tpl.IsActive,
		CreatedAt:       tpl.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       tpl.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// UpdateEmailTemplateRequest represents the request to update a template
type UpdateEmailTemplateRequest struct {
	Name            *string                             `json:"name,omitempty"`
	Subject         *string                             `json:"subject,omitempty"`
	BodyHTML        *string                             `json:"body_html,omitempty"`
	BodyText        *string                             `json:"body_text,omitempty"`
	Category        *string                             `json:"category,omitempty"`
	VariablesSchema map[string]projections.VariableSpec `json:"variables_schema,omitempty"`
	IsActive        *bool                               `json:"is_active,omitempty"`
}

// Update updates an existing email template
func (h *AdminEmailTemplatesHandler) Update(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template id")
		return
	}

	var req UpdateEmailTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate category if provided
	if req.Category != nil && *req.Category != "transactional" && *req.Category != "marketing" {
		writeError(w, http.StatusBadRequest, "category must be 'transactional' or 'marketing'")
		return
	}

	tpl, err := h.projection.Update(r.Context(), id, projections.UpdateEmailTemplateInput{
		Name:            req.Name,
		Subject:         req.Subject,
		BodyHTML:        req.BodyHTML,
		BodyText:        req.BodyText,
		Category:        req.Category,
		VariablesSchema: req.VariablesSchema,
		IsActive:        req.IsActive,
	})
	if err != nil {
		slog.Error("update email template failed",
			slog.String("template_id", id.String()),
			slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to update template")
		return
	}

	varsSchema, _ := tpl.ParseVariablesSchema()

	writeJSON(w, http.StatusOK, EmailTemplateResponse{
		ID:              tpl.ID.String(),
		Slug:            tpl.Slug,
		Name:            tpl.Name,
		Subject:         tpl.Subject,
		BodyHTML:        tpl.BodyHTML,
		BodyText:        tpl.BodyText,
		Category:        tpl.Category,
		VariablesSchema: varsSchema,
		IsSystem:        tpl.IsSystem,
		IsActive:        tpl.IsActive,
		CreatedAt:       tpl.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       tpl.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// Delete deletes a non-system email template
func (h *AdminEmailTemplatesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template id")
		return
	}

	if err := h.projection.Delete(r.Context(), id); err != nil {
		slog.Error("delete email template failed",
			slog.String("template_id", id.String()),
			slog.String("error", err.Error()))
		writeError(w, http.StatusBadRequest, "failed to delete template (may be system template)")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PreviewEmailTemplateRequest represents the request to preview a template
type PreviewEmailTemplateRequest struct {
	Subject   string         `json:"subject"`
	BodyHTML  string         `json:"body_html"`
	BodyText  string         `json:"body_text"`
	Variables map[string]any `json:"variables"`
}

// PreviewEmailTemplateResponse represents the rendered preview
type PreviewEmailTemplateResponse struct {
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html"`
	BodyText string `json:"body_text"`
}

// Preview renders a template with provided variables without saving
func (h *AdminEmailTemplatesHandler) Preview(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.session.GetAdminID(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req PreviewEmailTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Variables == nil {
		req.Variables = make(map[string]any)
	}

	// Create a temporary template lookup for rendering
	tempTemplate := &projections.EmailTemplateLookup{
		Subject:  req.Subject,
		BodyHTML: req.BodyHTML,
		BodyText: req.BodyText,
	}

	rendered, err := tempTemplate.Render(req.Variables)
	if err != nil {
		slog.Error("preview template render failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadRequest, "template render failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, PreviewEmailTemplateResponse{
		Subject:  rendered.Subject,
		BodyHTML: rendered.BodyHTML,
		BodyText: rendered.BodyText,
	})
}

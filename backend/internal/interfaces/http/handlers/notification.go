package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	notifApp "github.com/udisondev/veziizi/backend/internal/application/notification"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type NotificationHandler struct {
	service     *notifApp.Service
	session     *session.Manager
	botUsername string
}

func NewNotificationHandler(
	service *notifApp.Service,
	session *session.Manager,
	cfg *config.Config,
) *NotificationHandler {
	return &NotificationHandler{
		service:     service,
		session:     session,
		botUsername: cfg.Telegram.BotUsername,
	}
}

func (h *NotificationHandler) RegisterRoutes(r *mux.Router) {
	// In-app notifications
	r.HandleFunc("/api/v1/notifications", h.List).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/notifications/unread-count", h.GetUnreadCount).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/notifications/read", h.MarkAsRead).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/notifications/read-all", h.MarkAllAsRead).Methods(http.MethodPost)

	// Preferences
	r.HandleFunc("/api/v1/notifications/preferences", h.GetPreferences).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/notifications/preferences", h.UpdatePreferences).Methods(http.MethodPatch)

	// Telegram (привязка через бота)
	r.HandleFunc("/api/v1/notifications/telegram/link-code", h.GenerateLinkCode).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/notifications/telegram", h.DisconnectTelegram).Methods(http.MethodDelete)

	// Email
	r.HandleFunc("/api/v1/notifications/email", h.SetEmail).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/notifications/email", h.DisconnectEmail).Methods(http.MethodDelete)
	r.HandleFunc("/api/v1/notifications/email/marketing", h.SetMarketingConsent).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/notifications/email/resend-verification", h.ResendEmailVerification).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/notifications/email/verify", h.VerifyEmailByToken).Methods(http.MethodPost)
}

// ===============================
// In-App Notifications
// ===============================

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	input := notifApp.ListNotificationsInput{
		Limit:  50,
		Offset: 0,
	}

	// Parse query params
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			input.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			input.Offset = offset
		}
	}

	if category := r.URL.Query().Get("category"); category != "" {
		cat := values.NotificationCategory(category)
		input.Category = &cat
	}

	if isReadStr := r.URL.Query().Get("is_read"); isReadStr != "" {
		isRead := isReadStr == "true"
		input.IsRead = &isRead
	}

	notifications, err := h.service.ListNotifications(r.Context(), memberID, input)
	if err != nil {
		slog.Error("failed to list notifications", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}

	writeJSON(w, http.StatusOK, notifications)
}

func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	count, err := h.service.GetUnreadCount(r.Context(), memberID)
	if err != nil {
		slog.Error("failed to get unread count", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to get unread count")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"unread": count})
}

type markAsReadRequest struct {
	NotificationIDs []uuid.UUID `json:"notification_ids"`
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req markAsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.NotificationIDs) == 0 {
		writeError(w, http.StatusBadRequest, "notification_ids is required")
		return
	}

	input := notifApp.MarkAsReadInput{
		NotificationIDs: req.NotificationIDs,
	}

	if err := h.service.MarkAsRead(r.Context(), memberID, input); err != nil {
		slog.Error("failed to mark as read", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to mark as read")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.MarkAllAsRead(r.Context(), memberID); err != nil {
		slog.Error("failed to mark all as read", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to mark all as read")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ===============================
// Preferences
// ===============================

func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	prefs, err := h.service.GetPreferences(r.Context(), memberID)
	if err != nil {
		slog.Error("failed to get preferences", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to get preferences")
		return
	}

	writeJSON(w, http.StatusOK, prefs)
}

type updatePreferencesRequest struct {
	Categories values.EnabledCategories `json:"categories"`
}

func (h *NotificationHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := notifApp.UpdatePreferencesInput{
		Categories: req.Categories,
	}

	if err := h.service.UpdatePreferences(r.Context(), memberID, input); err != nil {
		slog.Error("failed to update preferences", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to update preferences")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ===============================
// Telegram (привязка через бота)
// ===============================

// GenerateLinkCode генерирует код для привязки Telegram через бота
func (h *NotificationHandler) GenerateLinkCode(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if h.botUsername == "" {
		writeError(w, http.StatusServiceUnavailable, "telegram bot not configured")
		return
	}

	response, err := h.service.GenerateLinkCode(r.Context(), memberID, h.botUsername)
	if err != nil {
		slog.Error("failed to generate link code",
			slog.String("error", err.Error()),
			slog.String("member_id", memberID.String()),
		)
		writeError(w, http.StatusInternalServerError, "failed to generate link code")
		return
	}

	slog.Info("telegram link code generated",
		slog.String("member_id", memberID.String()),
		slog.String("code", response.Code),
	)

	writeJSON(w, http.StatusOK, response)
}

func (h *NotificationHandler) DisconnectTelegram(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.DisconnectTelegram(r.Context(), memberID); err != nil {
		slog.Error("failed to disconnect telegram",
			slog.String("error", err.Error()),
			slog.String("member_id", memberID.String()),
		)
		writeError(w, http.StatusInternalServerError, "failed to disconnect telegram")
		return
	}

	slog.Info("telegram disconnected", slog.String("member_id", memberID.String()))

	w.WriteHeader(http.StatusNoContent)
}

// ===============================
// Email
// ===============================

type setEmailRequest struct {
	Email string `json:"email"`
}

// SetEmail устанавливает email для уведомлений
func (h *NotificationHandler) SetEmail(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	input := notifApp.SetEmailInput{
		Email:     req.Email,
		IP:        httputil.GetClientIP(r),
		UserAgent: r.UserAgent(),
	}

	if err := h.service.SetEmail(r.Context(), memberID, input); err != nil {
		slog.Error("failed to set email",
			slog.String("error", err.Error()),
			slog.String("member_id", memberID.String()),
		)
		// Возвращаем конкретные ошибки клиенту
		if errors.Is(err, notifApp.ErrTooManyVerificationRequests) {
			writeError(w, http.StatusTooManyRequests, "too many verification requests")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to set email")
		return
	}

	slog.Info("email set and verification sent", slog.String("member_id", memberID.String()))

	w.WriteHeader(http.StatusNoContent)
}

// DisconnectEmail отключает email уведомления
func (h *NotificationHandler) DisconnectEmail(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.DisconnectEmail(r.Context(), memberID); err != nil {
		slog.Error("failed to disconnect email",
			slog.String("error", err.Error()),
			slog.String("member_id", memberID.String()),
		)
		writeError(w, http.StatusInternalServerError, "failed to disconnect email")
		return
	}

	slog.Info("email disconnected", slog.String("member_id", memberID.String()))

	w.WriteHeader(http.StatusNoContent)
}

type setMarketingConsentRequest struct {
	Consent bool `json:"consent"`
}

// SetMarketingConsent устанавливает согласие на маркетинговые рассылки
func (h *NotificationHandler) SetMarketingConsent(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setMarketingConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := notifApp.SetMarketingConsentInput{
		Consent: req.Consent,
	}

	if err := h.service.SetMarketingConsent(r.Context(), memberID, input); err != nil {
		slog.Error("failed to set marketing consent",
			slog.String("error", err.Error()),
			slog.String("member_id", memberID.String()),
		)
		writeError(w, http.StatusInternalServerError, "failed to set marketing consent")
		return
	}

	slog.Info("marketing consent set",
		slog.String("member_id", memberID.String()),
		slog.Bool("consent", req.Consent),
	)

	w.WriteHeader(http.StatusNoContent)
}

// ResendEmailVerification повторно отправляет письмо с подтверждением email
func (h *NotificationHandler) ResendEmailVerification(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	input := notifApp.ResendEmailVerificationInput{
		IP:        httputil.GetClientIP(r),
		UserAgent: r.UserAgent(),
	}

	if err := h.service.ResendEmailVerification(r.Context(), memberID, input); err != nil {
		slog.Error("failed to resend email verification",
			slog.String("error", err.Error()),
			slog.String("member_id", memberID.String()),
		)
		// Возвращаем конкретные ошибки клиенту
		if errors.Is(err, notifApp.ErrEmailNotSet) {
			writeError(w, http.StatusBadRequest, "email not set")
			return
		}
		if errors.Is(err, notifApp.ErrEmailAlreadyVerified) {
			writeError(w, http.StatusBadRequest, "email already verified")
			return
		}
		if errors.Is(err, notifApp.ErrTooManyVerificationRequests) {
			writeError(w, http.StatusTooManyRequests, "too many verification requests")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to resend verification")
		return
	}

	slog.Info("email verification resent", slog.String("member_id", memberID.String()))

	w.WriteHeader(http.StatusNoContent)
}

// verifyEmailRequest запрос на верификацию email
type verifyEmailRequest struct {
	Token string `json:"token"`
}

// VerifyEmailByToken верифицирует email по токену
func (h *NotificationHandler) VerifyEmailByToken(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := h.service.VerifyEmailByToken(r.Context(), req.Token); err != nil {
		slog.Error("failed to verify email",
			slog.String("error", err.Error()),
		)
		if errors.Is(err, notifApp.ErrInvalidVerificationToken) {
			writeError(w, http.StatusBadRequest, "invalid or expired token")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to verify email")
		return
	}

	slog.Info("email verified by token")

	w.WriteHeader(http.StatusNoContent)
}

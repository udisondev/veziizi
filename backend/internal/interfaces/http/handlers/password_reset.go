package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/notifications"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// PasswordResetHandler handles password reset functionality
type PasswordResetHandler struct {
	members        *projections.MembersProjection
	passwordReset  *projections.PasswordResetProjection
	emailTemplates *projections.EmailTemplatesProjection
	emailProvider  notifications.EmailProvider
	appConfig      *config.Config
}

// NewPasswordResetHandler creates a new password reset handler
func NewPasswordResetHandler(
	members *projections.MembersProjection,
	passwordReset *projections.PasswordResetProjection,
	emailTemplates *projections.EmailTemplatesProjection,
	emailProvider notifications.EmailProvider,
	appConfig *config.Config,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		members:        members,
		passwordReset:  passwordReset,
		emailTemplates: emailTemplates,
		emailProvider:  emailProvider,
		appConfig:      appConfig,
	}
}

// RegisterRoutes registers password reset routes
func (h *PasswordResetHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/auth/forgot-password", h.ForgotPassword).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/auth/reset-password", h.ResetPassword).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/auth/reset-password/{token}", h.ValidateToken).Methods(http.MethodGet)
}

// ForgotPasswordRequest is the request body for forgot password
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword handles password reset request
// POST /api/v1/auth/forgot-password
func (h *PasswordResetHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Validate email format
	if !isValidEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	// Get client metadata for audit
	meta := httputil.GetClientMetadata(r)

	// Always return success to prevent email enumeration
	// But only actually send email if user exists
	defer func() {
		w.WriteHeader(http.StatusNoContent)
	}()

	// Find member by email
	member, err := h.members.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// User doesn't exist - log but don't reveal
			slog.Debug("password reset requested for non-existent email",
				slog.String("email", req.Email))
			return
		}
		slog.Error("failed to get member by email",
			slog.String("error", err.Error()))
		return
	}

	// Check if member is active
	if member.Status != "active" {
		slog.Debug("password reset requested for inactive member",
			slog.String("member_id", member.ID.String()))
		return
	}

	// Check rate limit
	if err := h.passwordReset.CheckRateLimit(r.Context(), member.ID, meta.IP); err != nil {
		if errors.Is(err, projections.ErrTooManyResets) {
			slog.Warn("password reset rate limit exceeded",
				slog.String("member_id", member.ID.String()),
				slog.String("ip", meta.IP))
		} else {
			slog.Error("failed to check rate limit",
				slog.String("error", err.Error()))
		}
		return
	}

	// Create token
	token, err := h.passwordReset.CreateToken(r.Context(), member.ID, meta.IP, meta.UserAgent)
	if err != nil {
		slog.Error("failed to create password reset token",
			slog.String("member_id", member.ID.String()),
			slog.String("error", err.Error()))
		return
	}

	// Send email
	if err := h.sendPasswordResetEmail(r.Context(), member.Email, token); err != nil {
		slog.Error("failed to send password reset email",
			slog.String("member_id", member.ID.String()),
			slog.String("error", err.Error()))
		return
	}

	slog.Info("password reset email sent",
		slog.String("member_id", member.ID.String()))
}

// ResetPasswordRequest is the request body for resetting password
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPassword handles actual password reset
// POST /api/v1/auth/reset-password
func (h *PasswordResetHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	if req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "new_password is required")
		return
	}

	// Validate password strength
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// Validate token
	tokenData, err := h.passwordReset.ValidateToken(r.Context(), req.Token)
	if err != nil {
		switch {
		case errors.Is(err, projections.ErrTokenNotFound):
			writeError(w, http.StatusBadRequest, "invalid or expired token")
		case errors.Is(err, projections.ErrTokenExpired):
			writeError(w, http.StatusBadRequest, "token has expired")
		case errors.Is(err, projections.ErrTokenUsed):
			writeError(w, http.StatusBadRequest, "token has already been used")
		default:
			slog.Error("failed to validate token",
				slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	// Hash new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password",
			slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Update password
	if err := h.members.UpdatePassword(r.Context(), tokenData.MemberID, string(passwordHash)); err != nil {
		slog.Error("failed to update password",
			slog.String("member_id", tokenData.MemberID.String()),
			slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Mark token as used
	if err := h.passwordReset.MarkAsUsed(r.Context(), req.Token); err != nil {
		slog.Error("failed to mark token as used",
			slog.String("error", err.Error()))
		// Don't return error - password was already changed
	}

	// Invalidate all other tokens for this member
	if err := h.passwordReset.InvalidateAllForMember(r.Context(), tokenData.MemberID); err != nil {
		slog.Error("failed to invalidate other tokens",
			slog.String("member_id", tokenData.MemberID.String()),
			slog.String("error", err.Error()))
		// Don't return error - password was already changed
	}

	slog.Info("password reset successful",
		slog.String("member_id", tokenData.MemberID.String()))

	w.WriteHeader(http.StatusNoContent)
}

// ValidateToken checks if a token is valid (for frontend to show form)
// GET /api/v1/auth/reset-password/{token}
func (h *PasswordResetHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	_, err := h.passwordReset.ValidateToken(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, projections.ErrTokenNotFound):
			writeError(w, http.StatusNotFound, "invalid token")
		case errors.Is(err, projections.ErrTokenExpired):
			writeError(w, http.StatusGone, "token has expired")
		case errors.Is(err, projections.ErrTokenUsed):
			writeError(w, http.StatusGone, "token has already been used")
		default:
			slog.Error("failed to validate token",
				slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	// Token is valid
	w.WriteHeader(http.StatusNoContent)
}

// sendPasswordResetEmail sends password reset email using template
func (h *PasswordResetHandler) sendPasswordResetEmail(ctx context.Context, email, token string) error {
	// Get email template
	template, err := h.emailTemplates.GetBySlug(ctx, "password-reset")
	if err != nil {
		return err
	}

	// Build reset link
	baseURL := h.appConfig.App.BaseURL
	if baseURL == "" {
		baseURL = "https://veziizi.ru"
	}
	resetLink := baseURL + "/reset-password/" + token

	// Render template
	rendered, err := template.Render(map[string]any{
		"ResetLink": resetLink,
	})
	if err != nil {
		return err
	}

	// Send email
	_, err = h.emailProvider.Send(ctx, notifications.EmailMessage{
		To:        email,
		Subject:   rendered.Subject,
		BodyHTML:  rendered.BodyHTML,
		BodyText:  rendered.BodyText,
		EmailType: values.EmailTypeTransactional,
	})

	return err
}

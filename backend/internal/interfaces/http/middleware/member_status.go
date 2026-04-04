package middleware

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
)

// CheckMemberStatus creates middleware that checks member status
// and blocks access for blocked members
func CheckMemberStatus(
	sessionManager *session.Manager,
	membersProjection *projections.MembersProjection,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip for paths that don't require status check
			skip := shouldSkipMemberStatusCheck(r.URL.Path, r.Method)
			if skip {
				next.ServeHTTP(w, r)
				return
			}

			// Get member ID from session
			memberID, ok := sessionManager.GetMemberID(r)
			if !ok {
				// No member in session - уже обработано RequireAuth
				next.ServeHTTP(w, r)
				return
			}

			// Check member status
			status, err := membersProjection.GetStatus(r.Context(), memberID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
					slog.Error("member not found in lookup",
						slog.String("member_id", memberID.String()))
					writeError(w, http.StatusInternalServerError, "member not found")
					return
				}
				slog.Error("failed to check member status",
					slog.String("member_id", memberID.String()),
					slog.String("error", err.Error()))
				writeError(w, http.StatusInternalServerError, "internal server error")
				return
			}

			// Block if not active
			if status != string(values.MemberStatusActive) {
				slog.Warn("blocked request from blocked member",
					slog.String("member_id", memberID.String()),
					slog.String("status", status),
					slog.String("path", r.URL.Path))

				writeError(w, http.StatusForbidden, "account is blocked")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// shouldSkipMemberStatusCheck determines if the status check should be skipped
func shouldSkipMemberStatusCheck(path, method string) bool {
	// NEVER skip private organization endpoints (require auth + member status check)
	if strings.HasPrefix(path, "/api/v1/organizations/") && strings.HasSuffix(path, "/full") {
		return false
	}

	// Skip public paths
	if isPublicPath(path, method) {
		return true
	}

	// Skip admin paths (separate admin session)
	if strings.HasPrefix(path, "/api/v1/admin/") {
		return true
	}

	// Skip dev paths (development only)
	if strings.HasPrefix(path, "/api/v1/dev/") {
		return true
	}

	// Skip auth endpoints
	if (path == "/api/v1/auth/login" && method == http.MethodPost) ||
		(path == "/api/v1/auth/logout" && method == http.MethodPost) {
		return true
	}

	// Skip /auth/me - пользователь должен видеть информацию о себе
	if path == "/api/v1/auth/me" && method == http.MethodGet {
		return true
	}

	return false
}

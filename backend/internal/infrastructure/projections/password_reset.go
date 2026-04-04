package projections

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

var (
	ErrTokenNotFound = errors.New("password reset token not found")
	ErrTokenExpired  = errors.New("password reset token expired")
	ErrTokenUsed     = errors.New("password reset token already used")
	ErrTooManyResets = errors.New("too many password reset requests")
)

// PasswordResetProjection handles password reset token operations
type PasswordResetProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewPasswordResetProjection(db dbtx.TxManager) *PasswordResetProjection {
	return &PasswordResetProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        uuid.UUID  `db:"id"`
	MemberID  uuid.UUID  `db:"member_id"`
	Token     string     `db:"token"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	IPAddress *string    `db:"ip_address"`
	UserAgent *string    `db:"user_agent"`
	CreatedAt time.Time  `db:"created_at"`
}

// TokenTTL is the token expiration time (1 hour)
const TokenTTL = 1 * time.Hour

// Rate limiting defaults (can be changed for testing)
var (
	maxTokensPerMemberPerHour = 3
	maxTokensPerIPPerHour     = 5
)

// SetPasswordResetRateLimits sets rate limits for password reset tokens.
// Use this in tests to increase limits.
func SetPasswordResetRateLimits(perMember, perIP int) {
	maxTokensPerMemberPerHour = perMember
	maxTokensPerIPPerHour = perIP
}

// generateToken generates a cryptographically secure token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateToken creates a new password reset token for a member
func (p *PasswordResetProjection) CreateToken(ctx context.Context, memberID uuid.UUID, ip, userAgent string) (string, error) {
	// Generate secure token
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	var ipVal, uaVal any
	if ip != "" {
		ipVal = ip
	}
	if userAgent != "" {
		uaVal = userAgent
	}

	query, args, err := p.psql.
		Insert("password_reset_tokens").
		Columns("member_id", "token", "expires_at", "ip_address", "user_agent").
		Values(memberID, token, time.Now().Add(TokenTTL), ipVal, uaVal).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return "", fmt.Errorf("insert token: %w", err)
	}

	return token, nil
}

// GetByToken retrieves a token by its value
func (p *PasswordResetProjection) GetByToken(ctx context.Context, token string) (*PasswordResetToken, error) {
	query, args, err := p.psql.
		Select("id", "member_id", "token", "expires_at", "used_at", "ip_address::TEXT as ip_address", "user_agent", "created_at").
		From("password_reset_tokens").
		Where(squirrel.Eq{"token": token}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var t PasswordResetToken
	if err := pgxscan.Get(ctx, p.db, &t, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("get token: %w", err)
	}

	return &t, nil
}

// ValidateToken checks if token is valid (exists, not expired, not used)
func (p *PasswordResetProjection) ValidateToken(ctx context.Context, token string) (*PasswordResetToken, error) {
	t, err := p.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if t.UsedAt != nil {
		return nil, ErrTokenUsed
	}

	if time.Now().After(t.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return t, nil
}

// MarkAsUsed marks a token as used
func (p *PasswordResetProjection) MarkAsUsed(ctx context.Context, token string) error {
	query, args, err := p.psql.
		Update("password_reset_tokens").
		Set("used_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"token": token}).
		Where(squirrel.Eq{"used_at": nil}). // Only mark if not already used
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTokenUsed
	}

	return nil
}

// CheckRateLimit checks if member or IP exceeded rate limit
func (p *PasswordResetProjection) CheckRateLimit(ctx context.Context, memberID uuid.UUID, ip string) error {
	// Check member rate limit
	query := `
		SELECT COUNT(*)
		FROM password_reset_tokens
		WHERE member_id = $1
		  AND created_at > NOW() - INTERVAL '1 hour'
	`
	var memberCount int
	if err := p.db.QueryRow(ctx, query, memberID).Scan(&memberCount); err != nil {
		return fmt.Errorf("check member rate limit: %w", err)
	}

	if memberCount >= maxTokensPerMemberPerHour {
		return ErrTooManyResets
	}

	// Check IP rate limit
	if ip != "" {
		query = `
			SELECT COUNT(*)
			FROM password_reset_tokens
			WHERE ip_address = $1
			  AND created_at > NOW() - INTERVAL '1 hour'
		`
		var ipCount int
		if err := p.db.QueryRow(ctx, query, ip).Scan(&ipCount); err != nil {
			return fmt.Errorf("check IP rate limit: %w", err)
		}

		if ipCount >= maxTokensPerIPPerHour {
			return ErrTooManyResets
		}
	}

	return nil
}

// InvalidateAllForMember invalidates all pending tokens for a member
// (called after successful password reset)
func (p *PasswordResetProjection) InvalidateAllForMember(ctx context.Context, memberID uuid.UUID) error {
	query, args, err := p.psql.
		Update("password_reset_tokens").
		Set("used_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"member_id": memberID}).
		Where(squirrel.Eq{"used_at": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("invalidate tokens: %w", err)
	}

	return nil
}

// CleanupExpired removes expired tokens (for scheduled cleanup)
func (p *PasswordResetProjection) CleanupExpired(ctx context.Context) (int64, error) {
	query, args, err := p.psql.
		Delete("password_reset_tokens").
		Where(squirrel.Lt{"expires_at": time.Now()}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build delete query: %w", err)
	}

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete expired tokens: %w", err)
	}

	return result.RowsAffected(), nil
}

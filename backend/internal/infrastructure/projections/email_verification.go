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
	ErrEmailVerificationTokenNotFound = errors.New("email verification token not found")
	ErrEmailVerificationTokenExpired  = errors.New("email verification token expired")
	ErrEmailVerificationTokenUsed     = errors.New("email verification token already used")
	ErrTooManyVerificationRequests    = errors.New("too many verification requests")
)

// EmailVerificationProjection handles email verification token operations
type EmailVerificationProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewEmailVerificationProjection(db dbtx.TxManager) *EmailVerificationProjection {
	return &EmailVerificationProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// EmailVerificationToken represents an email verification token
type EmailVerificationToken struct {
	ID        uuid.UUID  `db:"id"`
	MemberID  uuid.UUID  `db:"member_id"`
	Token     string     `db:"token"`
	Email     string     `db:"email"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	IPAddress *string    `db:"ip_address"`
	UserAgent *string    `db:"user_agent"`
	CreatedAt time.Time  `db:"created_at"`
}

// EmailVerificationTokenTTL is the token expiration time (24 hours)
const EmailVerificationTokenTTL = 24 * time.Hour

// Rate limiting constants
const (
	MaxVerificationTokensPerMemberPerHour = 3
	MaxVerificationTokensPerIPPerHour     = 10
)

// generateVerificationToken generates a cryptographically secure token
func generateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateToken creates a new email verification token for a member
func (p *EmailVerificationProjection) CreateToken(ctx context.Context, memberID uuid.UUID, email, ip, userAgent string) (string, error) {
	// Generate secure token
	token, err := generateVerificationToken()
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
		Insert("email_verification_tokens").
		Columns("member_id", "token", "email", "expires_at", "ip_address", "user_agent").
		Values(memberID, token, email, time.Now().Add(EmailVerificationTokenTTL), ipVal, uaVal).
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
func (p *EmailVerificationProjection) GetByToken(ctx context.Context, token string) (*EmailVerificationToken, error) {
	query, args, err := p.psql.
		Select("id", "member_id", "token", "email", "expires_at", "used_at", "ip_address::TEXT as ip_address", "user_agent", "created_at").
		From("email_verification_tokens").
		Where(squirrel.Eq{"token": token}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var t EmailVerificationToken
	if err := pgxscan.Get(ctx, p.db, &t, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEmailVerificationTokenNotFound
		}
		return nil, fmt.Errorf("get token: %w", err)
	}

	return &t, nil
}

// ValidateToken checks if token is valid (exists, not expired, not used)
func (p *EmailVerificationProjection) ValidateToken(ctx context.Context, token string) (*EmailVerificationToken, error) {
	t, err := p.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if t.UsedAt != nil {
		return nil, ErrEmailVerificationTokenUsed
	}

	if time.Now().After(t.ExpiresAt) {
		return nil, ErrEmailVerificationTokenExpired
	}

	return t, nil
}

// MarkAsUsed marks a token as used
func (p *EmailVerificationProjection) MarkAsUsed(ctx context.Context, token string) error {
	query, args, err := p.psql.
		Update("email_verification_tokens").
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
		return ErrEmailVerificationTokenUsed
	}

	return nil
}

// CheckRateLimit checks if member or IP exceeded rate limit
func (p *EmailVerificationProjection) CheckRateLimit(ctx context.Context, memberID uuid.UUID, ip string) error {
	// Check member rate limit
	query := `
		SELECT COUNT(*)
		FROM email_verification_tokens
		WHERE member_id = $1
		  AND created_at > NOW() - INTERVAL '1 hour'
	`
	var memberCount int
	if err := p.db.QueryRow(ctx, query, memberID).Scan(&memberCount); err != nil {
		return fmt.Errorf("check member rate limit: %w", err)
	}

	if memberCount >= MaxVerificationTokensPerMemberPerHour {
		return ErrTooManyVerificationRequests
	}

	// Check IP rate limit
	if ip != "" {
		query = `
			SELECT COUNT(*)
			FROM email_verification_tokens
			WHERE ip_address = $1
			  AND created_at > NOW() - INTERVAL '1 hour'
		`
		var ipCount int
		if err := p.db.QueryRow(ctx, query, ip).Scan(&ipCount); err != nil {
			return fmt.Errorf("check IP rate limit: %w", err)
		}

		if ipCount >= MaxVerificationTokensPerIPPerHour {
			return ErrTooManyVerificationRequests
		}
	}

	return nil
}

// InvalidateAllForMember invalidates all pending tokens for a member
// (called after successful email verification)
func (p *EmailVerificationProjection) InvalidateAllForMember(ctx context.Context, memberID uuid.UUID) error {
	query, args, err := p.psql.
		Update("email_verification_tokens").
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
func (p *EmailVerificationProjection) CleanupExpired(ctx context.Context) (int64, error) {
	query, args, err := p.psql.
		Delete("email_verification_tokens").
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

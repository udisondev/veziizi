package projections

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type MembersProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewMembersProjection(db dbtx.TxManager) *MembersProjection {
	return &MembersProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type MemberLookup struct {
	ID             uuid.UUID `db:"id"`
	OrganizationID uuid.UUID `db:"organization_id"`
	Email          string    `db:"email"`
	PasswordHash   string    `db:"password_hash"`
	Name           string    `db:"name"`
	Phone          *string   `db:"phone"`
	TelegramID     *int64    `db:"telegram_id"`
	Role           string    `db:"role"`
	Status         string    `db:"status"`
	CreatedAt      time.Time `db:"created_at"`
}

// GetByEmail retrieves member by email for authentication
func (p *MembersProjection) GetByEmail(ctx context.Context, email string) (*MemberLookup, error) {
	query, args, err := p.psql.
		Select("id", "organization_id", "email", "password_hash", "name", "phone", "telegram_id", "role", "status", "created_at").
		From("members_lookup").
		Where(squirrel.Eq{"email": email}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var m MemberLookup
	if err := pgxscan.Get(ctx, p.db, &m, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get member by email: %w", err)
	}

	return &m, nil
}

// GetByID retrieves member by ID
func (p *MembersProjection) GetByID(ctx context.Context, id uuid.UUID) (*MemberLookup, error) {
	query, args, err := p.psql.
		Select("id", "organization_id", "email", "password_hash", "name", "phone", "telegram_id", "role", "status", "created_at").
		From("members_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var m MemberLookup
	if err := pgxscan.Get(ctx, p.db, &m, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get member by id: %w", err)
	}

	return &m, nil
}

// DevMemberItem represents minimal member data for dev switcher (no password_hash)
type DevMemberItem struct {
	ID             uuid.UUID `db:"id" json:"id"`
	OrganizationID uuid.UUID `db:"organization_id" json:"organization_id"`
	Email          string    `db:"email" json:"email"`
	Name           string    `db:"name" json:"name"`
	Role           string    `db:"role" json:"role"`
	Status         string    `db:"status" json:"status"`
}

// ListAll returns all members for dev user switcher (dev mode only)
func (p *MembersProjection) ListAll(ctx context.Context, search string, limit int) ([]DevMemberItem, error) {
	builder := p.psql.
		Select("id", "organization_id", "email", "name", "role", "status").
		From("members_lookup").
		OrderBy("created_at DESC")

	if search != "" {
		// SEC-014: Экранируем спецсимволы ILIKE
		escapedSearch := WrapLikePattern(search)
		builder = builder.Where(
			squirrel.Or{
				squirrel.ILike{"email": escapedSearch},
				squirrel.ILike{"name": escapedSearch},
			},
		)
	}

	if limit > 0 {
		builder = builder.Limit(uint64(limit))
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build list query: %w", err)
	}

	var members []DevMemberItem
	if err := pgxscan.Select(ctx, p.db, &members, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	return members, nil
}

// GetNames возвращает имена членов по их ID
func (p *MembersProjection) GetNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]string), nil
	}

	query, args, err := p.psql.
		Select("id", "name").
		From("members_lookup").
		Where(squirrel.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	type idName struct {
		ID   uuid.UUID `db:"id"`
		Name string    `db:"name"`
	}

	var rows []idName
	if err := pgxscan.Select(ctx, p.db, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get member names: %w", err)
	}

	result := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		result[row.ID] = row.Name
	}

	return result, nil
}

// RecordLogin updates last_login_* fields after successful login
func (p *MembersProjection) RecordLogin(ctx context.Context, memberID uuid.UUID, ip, fingerprint string) error {
	builder := p.psql.
		Update("members_lookup").
		Set("last_login_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": memberID})

	if ip != "" {
		builder = builder.Set("last_login_ip", ip)
	}
	if fingerprint != "" {
		builder = builder.Set("last_login_fingerprint", fingerprint)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// LoginHistoryEntry represents a login history record
type LoginHistoryEntry struct {
	ID             uuid.UUID `db:"id"`
	MemberID       uuid.UUID `db:"member_id"`
	OrganizationID uuid.UUID `db:"organization_id"`
	IPAddress      *string   `db:"ip_address"`
	Fingerprint    *string   `db:"fingerprint"`
	UserAgent      *string   `db:"user_agent"`
	Status         string    `db:"status"`
	CreatedAt      time.Time `db:"created_at"`
}

// RecordLoginHistory records a login attempt in history
func (p *MembersProjection) RecordLoginHistory(
	ctx context.Context,
	memberID, orgID uuid.UUID,
	ip, fingerprint, userAgent, status string,
) error {
	var ipVal, fpVal, uaVal any
	if ip != "" {
		ipVal = ip
	}
	if fingerprint != "" {
		fpVal = fingerprint
	}
	if userAgent != "" {
		uaVal = userAgent
	}

	query, args, err := p.psql.
		Insert("member_login_history").
		Columns("member_id", "organization_id", "ip_address", "fingerprint", "user_agent", "status").
		Values(memberID, orgID, ipVal, fpVal, uaVal, status).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert login history: %w", err)
	}

	return nil
}

// GetLoginHistory retrieves login history for a member
func (p *MembersProjection) GetLoginHistory(ctx context.Context, memberID uuid.UUID, limit int) ([]LoginHistoryEntry, error) {
	query, args, err := p.psql.
		Select("id", "member_id", "organization_id", "ip_address::TEXT as ip_address", "fingerprint", "user_agent", "status", "created_at").
		From("member_login_history").
		Where(squirrel.Eq{"member_id": memberID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var entries []LoginHistoryEntry
	if err := pgxscan.Select(ctx, p.db, &entries, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get login history: %w", err)
	}

	return entries, nil
}

const (
	MaxFailedLoginAttempts = 5
	AccountLockoutDuration = 15 * time.Minute
)

// IncrementFailedLogin increments the failed login counter and locks account if threshold exceeded.
func (p *MembersProjection) IncrementFailedLogin(ctx context.Context, memberID uuid.UUID) error {
	query, args, err := p.psql.
		Update("members_lookup").
		Set("failed_login_count", squirrel.Expr("failed_login_count + 1")).
		Set("last_failed_login_at", squirrel.Expr("NOW()")).
		Set("locked_until", squirrel.Expr(
			fmt.Sprintf("CASE WHEN failed_login_count + 1 >= %d THEN NOW() + INTERVAL '%d minutes' ELSE locked_until END",
				MaxFailedLoginAttempts, int(AccountLockoutDuration.Minutes())),
		)).
		Where(squirrel.Eq{"id": memberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build increment failed login query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("increment failed login: %w", err)
	}

	return nil
}

// ResetFailedLogin resets the failed login counter after successful login.
func (p *MembersProjection) ResetFailedLogin(ctx context.Context, memberID uuid.UUID) error {
	query, args, err := p.psql.
		Update("members_lookup").
		Set("failed_login_count", 0).
		Set("locked_until", nil).
		Where(squirrel.Eq{"id": memberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build reset failed login query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("reset failed login: %w", err)
	}

	return nil
}

// IsAccountLocked checks if the account is currently locked due to failed login attempts.
func (p *MembersProjection) IsAccountLocked(ctx context.Context, memberID uuid.UUID) (bool, error) {
	query, args, err := p.psql.
		Select("locked_until").
		From("members_lookup").
		Where(squirrel.Eq{"id": memberID}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build check locked query: %w", err)
	}

	var lockedUntil *time.Time
	if err := p.db.QueryRow(ctx, query, args...).Scan(&lockedUntil); err != nil {
		return false, fmt.Errorf("check account locked: %w", err)
	}

	if lockedUntil == nil {
		return false, nil
	}

	return time.Now().Before(*lockedUntil), nil
}

// MemberMetadata contains registration and login metadata for a member
type MemberMetadata struct {
	MemberID                uuid.UUID  `db:"id"`
	OrganizationID          uuid.UUID  `db:"organization_id"`
	RegistrationIP          *string    `db:"registration_ip"`
	RegistrationFingerprint *string    `db:"registration_fingerprint"`
	LastLoginIP             *string    `db:"last_login_ip"`
	LastLoginFingerprint    *string    `db:"last_login_fingerprint"`
	LastLoginAt             *time.Time `db:"last_login_at"`
}

// GetOrganizationsByIP finds organization IDs that have members with matching IP
// (either registration_ip or last_login_ip)
func (p *MembersProjection) GetOrganizationsByIP(ctx context.Context, ip string) ([]uuid.UUID, error) {
	query, args, err := p.psql.
		Select("DISTINCT organization_id").
		From("members_lookup").
		Where(squirrel.Or{
			squirrel.Eq{"registration_ip": ip},
			squirrel.Eq{"last_login_ip": ip},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var orgIDs []uuid.UUID
	if err := pgxscan.Select(ctx, p.db, &orgIDs, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get organizations by IP: %w", err)
	}

	return orgIDs, nil
}

// GetOrganizationsByFingerprint finds organization IDs that have members with matching fingerprint
// (either registration_fingerprint or last_login_fingerprint)
func (p *MembersProjection) GetOrganizationsByFingerprint(ctx context.Context, fingerprint string) ([]uuid.UUID, error) {
	query, args, err := p.psql.
		Select("DISTINCT organization_id").
		From("members_lookup").
		Where(squirrel.Or{
			squirrel.Eq{"registration_fingerprint": fingerprint},
			squirrel.Eq{"last_login_fingerprint": fingerprint},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var orgIDs []uuid.UUID
	if err := pgxscan.Select(ctx, p.db, &orgIDs, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get organizations by fingerprint: %w", err)
	}

	return orgIDs, nil
}

// GetMemberMetadata retrieves metadata for all members of an organization
func (p *MembersProjection) GetMemberMetadata(ctx context.Context, orgID uuid.UUID) ([]MemberMetadata, error) {
	query, args, err := p.psql.
		Select(
			"id", "organization_id",
			"registration_ip::TEXT as registration_ip", "registration_fingerprint",
			"last_login_ip::TEXT as last_login_ip", "last_login_fingerprint", "last_login_at",
		).
		From("members_lookup").
		Where(squirrel.Eq{"organization_id": orgID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var metadata []MemberMetadata
	if err := pgxscan.Select(ctx, p.db, &metadata, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get member metadata: %w", err)
	}

	return metadata, nil
}

// RegistrationVelocity contains velocity check thresholds
var RegistrationVelocity = struct {
	MaxRegistrationsPerIPPerHour         int
	MaxRegistrationsPerFingerprintPer24h int
}{
	MaxRegistrationsPerIPPerHour:         3,
	MaxRegistrationsPerFingerprintPer24h: 2,
}

// RegistrationVelocityResult contains velocity check result
type RegistrationVelocityResult struct {
	IsTooFast       bool
	IPRegistrations int
	FPRegistrations int
	BlockReason     string
}

// CheckRegistrationVelocity checks if registration is happening too fast from same IP/fingerprint
// Returns true if velocity exceeds thresholds
func (p *MembersProjection) CheckRegistrationVelocity(ctx context.Context, ip, fingerprint string) (*RegistrationVelocityResult, error) {
	result := &RegistrationVelocityResult{}

	// Check IP velocity (last hour)
	if ip != "" {
		query := `
			SELECT COUNT(*)
			FROM members_lookup
			WHERE registration_ip = $1
			  AND created_at > NOW() - INTERVAL '1 hour'
		`
		var count int
		if err := p.db.QueryRow(ctx, query, ip).Scan(&count); err != nil {
			return nil, fmt.Errorf("failed to check IP velocity: %w", err)
		}
		result.IPRegistrations = count

		if count >= RegistrationVelocity.MaxRegistrationsPerIPPerHour {
			result.IsTooFast = true
			result.BlockReason = fmt.Sprintf(
				"too many registrations from this IP (%d in last hour, max %d)",
				count, RegistrationVelocity.MaxRegistrationsPerIPPerHour,
			)
			return result, nil
		}
	}

	// Check fingerprint velocity (last 24 hours)
	if fingerprint != "" {
		query := `
			SELECT COUNT(*)
			FROM members_lookup
			WHERE registration_fingerprint = $1
			  AND created_at > NOW() - INTERVAL '24 hours'
		`
		var count int
		if err := p.db.QueryRow(ctx, query, fingerprint).Scan(&count); err != nil {
			return nil, fmt.Errorf("failed to check fingerprint velocity: %w", err)
		}
		result.FPRegistrations = count

		if count >= RegistrationVelocity.MaxRegistrationsPerFingerprintPer24h {
			result.IsTooFast = true
			result.BlockReason = fmt.Sprintf(
				"too many registrations from this device (%d in last 24 hours, max %d)",
				count, RegistrationVelocity.MaxRegistrationsPerFingerprintPer24h,
			)
			return result, nil
		}
	}

	return result, nil
}

// GetAllActiveMemberIDs возвращает ID всех активных членов, исключая указанный
func (p *MembersProjection) GetAllActiveMemberIDs(ctx context.Context, excludeMemberID *uuid.UUID) ([]uuid.UUID, error) {
	builder := p.psql.
		Select("m.id").
		From("members_lookup m").
		Join("organizations_lookup o ON o.id = m.organization_id").
		Where(squirrel.Eq{"m.status": "active"}).
		Where(squirrel.Eq{"o.status": "active"})

	if excludeMemberID != nil {
		builder = builder.Where(squirrel.NotEq{"m.id": *excludeMemberID})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var ids []uuid.UUID
	if err := pgxscan.Select(ctx, p.db, &ids, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get active member IDs: %w", err)
	}

	return ids, nil
}

// GetStatus returns member status by ID (lightweight query for middleware)
func (p *MembersProjection) GetStatus(ctx context.Context, id uuid.UUID) (string, error) {
	query, args, err := p.psql.
		Select("status").
		From("members_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("build select status query: %w", err)
	}

	var status string
	if err := pgxscan.Get(ctx, p.db, &status, query, args...); err != nil {
		return "", fmt.Errorf("get member status: %w", err)
	}

	return status, nil
}

// UpdatePassword updates member's password hash
func (p *MembersProjection) UpdatePassword(ctx context.Context, memberID uuid.UUID, passwordHash string) error {
	query, args, err := p.psql.
		Update("members_lookup").
		Set("password_hash", passwordHash).
		Where(squirrel.Eq{"id": memberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}

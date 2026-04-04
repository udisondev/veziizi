package projections

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

const (
	// LinkCodeTTL время жизни кода привязки
	LinkCodeTTL = 10 * time.Minute
	// LinkCodeLength длина кода
	LinkCodeLength = 6
	// LinkCodeCharset символы для генерации кода (без путающихся символов 0/O, 1/I/L)
	LinkCodeCharset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

type TelegramLinkProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewTelegramLinkProjection(db dbtx.TxManager) *TelegramLinkProjection {
	return &TelegramLinkProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// LinkCode представляет код привязки
type LinkCode struct {
	Code      string    `db:"code"`
	MemberID  uuid.UUID `db:"member_id"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// GenerateCode создает новый код привязки для member
func (p *TelegramLinkProjection) GenerateCode(ctx context.Context, memberID uuid.UUID) (string, error) {
	// Удаляем старые коды этого member
	if err := p.DeleteByMemberID(ctx, memberID); err != nil {
		return "", fmt.Errorf("delete old codes: %w", err)
	}

	// Генерируем уникальный код
	code, err := p.generateUniqueCode(ctx)
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}

	expiresAt := time.Now().Add(LinkCodeTTL)

	query, args, err := p.psql.
		Insert("telegram_link_codes").
		Columns("code", "member_id", "expires_at").
		Values(code, memberID, expiresAt).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return "", fmt.Errorf("insert link code: %w", err)
	}

	return code, nil
}

// GetByCode возвращает код привязки (если не истёк)
func (p *TelegramLinkProjection) GetByCode(ctx context.Context, code string) (*LinkCode, error) {
	query, args, err := p.psql.
		Select("code", "member_id", "expires_at", "created_at").
		From("telegram_link_codes").
		Where(squirrel.Eq{"code": code}).
		Where(squirrel.Gt{"expires_at": time.Now()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var linkCode LinkCode
	if err := pgxscan.Get(ctx, p.db, &linkCode, query, args...); err != nil {
		return nil, fmt.Errorf("get link code: %w", err)
	}

	return &linkCode, nil
}

// DeleteByCode удаляет код после использования
func (p *TelegramLinkProjection) DeleteByCode(ctx context.Context, code string) error {
	query, args, err := p.psql.
		Delete("telegram_link_codes").
		Where(squirrel.Eq{"code": code}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete link code: %w", err)
	}

	return nil
}

// DeleteByMemberID удаляет все коды member
func (p *TelegramLinkProjection) DeleteByMemberID(ctx context.Context, memberID uuid.UUID) error {
	query, args, err := p.psql.
		Delete("telegram_link_codes").
		Where(squirrel.Eq{"member_id": memberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete link codes by member: %w", err)
	}

	return nil
}

// CleanupExpired удаляет истёкшие коды
func (p *TelegramLinkProjection) CleanupExpired(ctx context.Context) (int64, error) {
	query, args, err := p.psql.
		Delete("telegram_link_codes").
		Where(squirrel.Lt{"expires_at": time.Now()}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build delete query: %w", err)
	}

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired codes: %w", err)
	}

	return result.RowsAffected(), nil
}

// generateUniqueCode генерирует уникальный код
func (p *TelegramLinkProjection) generateUniqueCode(ctx context.Context) (string, error) {
	for range 10 { // максимум 10 попыток
		code := generateRandomCode(LinkCodeLength)

		// Проверяем уникальность
		exists, err := p.codeExists(ctx, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after 10 attempts")
}

func (p *TelegramLinkProjection) codeExists(ctx context.Context, code string) (bool, error) {
	query, args, err := p.psql.
		Select("1").
		From("telegram_link_codes").
		Where(squirrel.Eq{"code": code}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build select query: %w", err)
	}

	var exists int
	err = pgxscan.Get(ctx, p.db, &exists, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check code exists: %w", err)
	}

	return true, nil
}

// generateRandomCode генерирует случайный код из charset
func generateRandomCode(length int) string {
	code := make([]byte, length)
	charsetLen := big.NewInt(int64(len(LinkCodeCharset)))

	for i := range length {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			// Fallback на менее безопасный метод (не должно происходить)
			code[i] = LinkCodeCharset[i%len(LinkCodeCharset)]
			continue
		}
		code[i] = LinkCodeCharset[n.Int64()]
	}

	return string(code)
}

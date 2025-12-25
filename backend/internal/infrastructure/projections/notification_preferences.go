package projections

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
)

type NotificationPreferencesProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewNotificationPreferencesProjection(db dbtx.TxManager) *NotificationPreferencesProjection {
	return &NotificationPreferencesProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// PreferencesLookup представляет настройки уведомлений member
type PreferencesLookup struct {
	MemberID            uuid.UUID  `db:"member_id"`
	TelegramChatID      *int64     `db:"telegram_chat_id"`
	TelegramUsername    *string    `db:"telegram_username"`
	TelegramConnectedAt *time.Time `db:"telegram_connected_at"`
	EnabledCategories   []byte     `db:"enabled_categories"` // JSONB
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

// ParseEnabledCategories парсит JSONB в структуру
func (p *PreferencesLookup) ParseEnabledCategories() (values.EnabledCategories, error) {
	var categories values.EnabledCategories
	if err := json.Unmarshal(p.EnabledCategories, &categories); err != nil {
		return nil, fmt.Errorf("failed to parse enabled_categories: %w", err)
	}
	return categories, nil
}

// PreferencesResponse представляет настройки для API ответа
type PreferencesResponse struct {
	MemberID          uuid.UUID               `json:"member_id"`
	EnabledCategories values.EnabledCategories `json:"enabled_categories"`
	Telegram          TelegramStatus          `json:"telegram"`
}

// TelegramStatus представляет статус Telegram подключения
type TelegramStatus struct {
	Connected   bool       `json:"connected"`
	Username    *string    `json:"username,omitempty"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
}

// GetByMemberID возвращает настройки member
func (p *NotificationPreferencesProjection) GetByMemberID(ctx context.Context, memberID uuid.UUID) (*PreferencesLookup, error) {
	query, args, err := p.psql.
		Select("member_id", "telegram_chat_id", "telegram_username", "telegram_connected_at", "enabled_categories", "created_at", "updated_at").
		From("notification_preferences").
		Where(squirrel.Eq{"member_id": memberID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var pref PreferencesLookup
	if err := pgxscan.Get(ctx, p.db, &pref, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get notification preferences: %w", err)
	}

	return &pref, nil
}

// GetOrCreateByMemberID возвращает настройки member, создавая дефолтные если нет
func (p *NotificationPreferencesProjection) GetOrCreateByMemberID(ctx context.Context, memberID uuid.UUID) (*PreferencesLookup, error) {
	pref, err := p.GetByMemberID(ctx, memberID)
	if err == nil {
		return pref, nil
	}

	// Создаем дефолтные настройки
	defaultCategories := values.DefaultEnabledCategories()
	categoriesJSON, err := json.Marshal(defaultCategories)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default categories: %w", err)
	}

	query, args, err := p.psql.
		Insert("notification_preferences").
		Columns("member_id", "enabled_categories").
		Values(memberID, categoriesJSON).
		Suffix("ON CONFLICT (member_id) DO NOTHING").
		Suffix("RETURNING member_id, telegram_chat_id, telegram_username, telegram_connected_at, enabled_categories, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert query: %w", err)
	}

	var newPref PreferencesLookup
	if err := pgxscan.Get(ctx, p.db, &newPref, query, args...); err != nil {
		// Если RETURNING не вернул (был conflict), читаем существующую запись
		return p.GetByMemberID(ctx, memberID)
	}

	return &newPref, nil
}

// UpdateCategories обновляет настройки категорий
func (p *NotificationPreferencesProjection) UpdateCategories(ctx context.Context, memberID uuid.UUID, categories values.EnabledCategories) error {
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}

	query, args, err := p.psql.
		Update("notification_preferences").
		Set("enabled_categories", categoriesJSON).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"member_id": memberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update notification preferences: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Запись не существует, создаем
		_, err = p.GetOrCreateByMemberID(ctx, memberID)
		if err != nil {
			return fmt.Errorf("failed to create preferences: %w", err)
		}
		// Повторяем update
		_, err = p.db.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to update notification preferences after create: %w", err)
		}
	}

	return nil
}

// ConnectTelegram подключает Telegram аккаунт и включает telegram для всех категорий
func (p *NotificationPreferencesProjection) ConnectTelegram(ctx context.Context, memberID uuid.UUID, chatID int64, username string) error {
	now := time.Now()

	// Сначала убедимся, что запись существует
	pref, err := p.GetOrCreateByMemberID(ctx, memberID)
	if err != nil {
		return fmt.Errorf("failed to ensure preferences exist: %w", err)
	}

	// Получаем текущие категории и включаем telegram для всех
	categories, err := pref.ParseEnabledCategories()
	if err != nil {
		// Если ошибка парсинга - используем дефолтные
		categories = values.DefaultEnabledCategories()
	}

	// Включаем telegram для всех категорий
	categories.EnableTelegramForAll()

	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}

	query, args, err := p.psql.
		Update("notification_preferences").
		Set("telegram_chat_id", chatID).
		Set("telegram_username", username).
		Set("telegram_connected_at", now).
		Set("enabled_categories", categoriesJSON).
		Set("updated_at", now).
		Where(squirrel.Eq{"member_id": memberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to connect telegram: %w", err)
	}

	return nil
}

// DisconnectTelegram отключает Telegram аккаунт
func (p *NotificationPreferencesProjection) DisconnectTelegram(ctx context.Context, memberID uuid.UUID) error {
	query, args, err := p.psql.
		Update("notification_preferences").
		Set("telegram_chat_id", nil).
		Set("telegram_username", nil).
		Set("telegram_connected_at", nil).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"member_id": memberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to disconnect telegram: %w", err)
	}

	return nil
}

// IsChatIDConnected проверяет, подключён ли chatID к какому-либо member
func (p *NotificationPreferencesProjection) IsChatIDConnected(ctx context.Context, chatID int64) (bool, error) {
	query, args, err := p.psql.
		Select("1").
		From("notification_preferences").
		Where(squirrel.Eq{"telegram_chat_id": chatID}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to build select query: %w", err)
	}

	var exists int
	if err := pgxscan.Get(ctx, p.db, &exists, query, args...); err != nil {
		// Не найдено - не подключён
		return false, nil
	}

	return true, nil
}

// GetTelegramChatID возвращает telegram chat ID для member (если подключен)
func (p *NotificationPreferencesProjection) GetTelegramChatID(ctx context.Context, memberID uuid.UUID) (*int64, error) {
	query, args, err := p.psql.
		Select("telegram_chat_id").
		From("notification_preferences").
		Where(squirrel.Eq{"member_id": memberID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var chatID *int64
	if err := pgxscan.Get(ctx, p.db, &chatID, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get telegram chat id: %w", err)
	}

	return chatID, nil
}

// GetMembersWithTelegramEnabled возвращает member IDs у которых включен Telegram для категории
func (p *NotificationPreferencesProjection) GetMembersWithTelegramEnabled(ctx context.Context, category values.NotificationCategory, memberIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(memberIDs) == 0 {
		return nil, nil
	}

	// Используем JSONB query для проверки настроек
	jsonPath := fmt.Sprintf("enabled_categories->'%s'->>'telegram'", category)

	query, args, err := p.psql.
		Select("member_id").
		From("notification_preferences").
		Where(squirrel.Eq{"member_id": memberIDs}).
		Where(squirrel.NotEq{"telegram_chat_id": nil}).
		Where(fmt.Sprintf("%s = 'true'", jsonPath)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var result []uuid.UUID
	if err := pgxscan.Select(ctx, p.db, &result, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get members with telegram enabled: %w", err)
	}

	return result, nil
}

// GetMembersWithInAppEnabled возвращает member IDs у которых включен in_app для категории
func (p *NotificationPreferencesProjection) GetMembersWithInAppEnabled(ctx context.Context, category values.NotificationCategory, memberIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(memberIDs) == 0 {
		return nil, nil
	}

	// JSONB query для проверки настроек (или если нет записи - используем дефолт true)
	jsonPath := fmt.Sprintf("enabled_categories->'%s'->>'in_app'", category)

	query, args, err := p.psql.
		Select("member_id").
		From("notification_preferences").
		Where(squirrel.Eq{"member_id": memberIDs}).
		Where(squirrel.Or{
			squirrel.Expr(fmt.Sprintf("%s = 'true'", jsonPath)),
			squirrel.Expr(fmt.Sprintf("%s IS NULL", jsonPath)), // дефолт = true
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var result []uuid.UUID
	if err := pgxscan.Select(ctx, p.db, &result, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get members with in_app enabled: %w", err)
	}

	return result, nil
}

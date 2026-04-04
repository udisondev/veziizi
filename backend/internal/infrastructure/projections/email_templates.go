package projections

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// restrictedFuncMap blocks dangerous template function calls
var restrictedFuncMap = texttemplate.FuncMap{
	"call":  func() string { return "" },
	"html":  func() string { return "" },
	"js":    func() string { return "" },
	"print": func() string { return "" },
}

// EmailTemplatesProjection работает с таблицей email_templates
type EmailTemplatesProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

// NewEmailTemplatesProjection создает новую проекцию
func NewEmailTemplatesProjection(db dbtx.TxManager) *EmailTemplatesProjection {
	return &EmailTemplatesProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// EmailTemplateLookup представляет шаблон email
type EmailTemplateLookup struct {
	ID              uuid.UUID       `db:"id"`
	Slug            string          `db:"slug"`
	Name            string          `db:"name"`
	Subject         string          `db:"subject"`
	BodyHTML        string          `db:"body_html"`
	BodyText        string          `db:"body_text"`
	Category        string          `db:"category"`
	VariablesSchema json.RawMessage `db:"variables_schema"`
	IsSystem        bool            `db:"is_system"`
	IsActive        bool            `db:"is_active"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

// EmailType возвращает тип email (transactional/marketing)
func (t *EmailTemplateLookup) EmailType() values.EmailType {
	if t.Category == "marketing" {
		return values.EmailTypeMarketing
	}
	return values.EmailTypeTransactional
}

// VariableSpec описание переменной шаблона
type VariableSpec struct {
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// ParseVariablesSchema парсит JSON schema переменных
func (t *EmailTemplateLookup) ParseVariablesSchema() (map[string]VariableSpec, error) {
	var schema map[string]VariableSpec
	if err := json.Unmarshal(t.VariablesSchema, &schema); err != nil {
		return nil, fmt.Errorf("parse variables schema: %w", err)
	}
	return schema, nil
}

// RenderedEmail результат рендеринга шаблона
type RenderedEmail struct {
	Subject  string
	BodyHTML string
	BodyText string
	Type     values.EmailType
}

// Render рендерит шаблон с переданными данными
func (t *EmailTemplateLookup) Render(data map[string]any) (*RenderedEmail, error) {
	// Рендерим subject (restricted FuncMap to prevent SSTI)
	subjectTpl, err := texttemplate.New("subject").Funcs(restrictedFuncMap).Parse(t.Subject)
	if err != nil {
		return nil, fmt.Errorf("parse subject template: %w", err)
	}
	var subjectBuf bytes.Buffer
	if err := subjectTpl.Execute(&subjectBuf, data); err != nil {
		return nil, fmt.Errorf("execute subject template: %w", err)
	}

	// Рендерим HTML body (html/template auto-escapes, restrictedFuncMap blocks dangerous calls)
	htmlTpl, err := htmltemplate.New("body_html").Funcs(htmltemplate.FuncMap(restrictedFuncMap)).Parse(t.BodyHTML)
	if err != nil {
		return nil, fmt.Errorf("parse html template: %w", err)
	}
	var htmlBuf bytes.Buffer
	if err := htmlTpl.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("execute html template: %w", err)
	}

	// Рендерим text body (restricted FuncMap to prevent SSTI)
	textTpl, err := texttemplate.New("body_text").Funcs(restrictedFuncMap).Parse(t.BodyText)
	if err != nil {
		return nil, fmt.Errorf("parse text template: %w", err)
	}
	var textBuf bytes.Buffer
	if err := textTpl.Execute(&textBuf, data); err != nil {
		return nil, fmt.Errorf("execute text template: %w", err)
	}

	return &RenderedEmail{
		Subject:  subjectBuf.String(),
		BodyHTML: htmlBuf.String(),
		BodyText: textBuf.String(),
		Type:     t.EmailType(),
	}, nil
}

// GetBySlug возвращает активный шаблон по slug
func (p *EmailTemplatesProjection) GetBySlug(ctx context.Context, slug string) (*EmailTemplateLookup, error) {
	query, args, err := p.psql.
		Select(
			"id", "slug", "name", "subject", "body_html", "body_text",
			"category", "variables_schema", "is_system", "is_active",
			"created_at", "updated_at",
		).
		From("email_templates").
		Where(squirrel.Eq{"slug": slug, "is_active": true}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var tpl EmailTemplateLookup
	if err := pgxscan.Get(ctx, p.db, &tpl, query, args...); err != nil {
		return nil, fmt.Errorf("get email template by slug %s: %w", slug, err)
	}

	return &tpl, nil
}

// GetByID возвращает шаблон по ID
func (p *EmailTemplatesProjection) GetByID(ctx context.Context, id uuid.UUID) (*EmailTemplateLookup, error) {
	query, args, err := p.psql.
		Select(
			"id", "slug", "name", "subject", "body_html", "body_text",
			"category", "variables_schema", "is_system", "is_active",
			"created_at", "updated_at",
		).
		From("email_templates").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var tpl EmailTemplateLookup
	if err := pgxscan.Get(ctx, p.db, &tpl, query, args...); err != nil {
		return nil, fmt.Errorf("get email template by id %s: %w", id, err)
	}

	return &tpl, nil
}

// EmailTemplateListFilter фильтры для списка шаблонов
type EmailTemplateListFilter struct {
	Category   *string // transactional, marketing
	IsActive   *bool
	IsSystem   *bool
	SearchText string // поиск по name, slug
	Limit      int
	Offset     int
}

// List возвращает список шаблонов с фильтрацией
func (p *EmailTemplatesProjection) List(ctx context.Context, filter EmailTemplateListFilter) ([]EmailTemplateLookup, error) {
	qb := p.psql.
		Select(
			"id", "slug", "name", "subject", "body_html", "body_text",
			"category", "variables_schema", "is_system", "is_active",
			"created_at", "updated_at",
		).
		From("email_templates").
		OrderBy("name ASC")

	if filter.Category != nil {
		qb = qb.Where(squirrel.Eq{"category": *filter.Category})
	}
	if filter.IsActive != nil {
		qb = qb.Where(squirrel.Eq{"is_active": *filter.IsActive})
	}
	if filter.IsSystem != nil {
		qb = qb.Where(squirrel.Eq{"is_system": *filter.IsSystem})
	}
	if filter.SearchText != "" {
		searchPattern := "%" + EscapeLikePattern(filter.SearchText) + "%"
		qb = qb.Where(squirrel.Or{
			squirrel.ILike{"name": searchPattern},
			squirrel.ILike{"slug": searchPattern},
		})
	}

	if filter.Limit > 0 {
		qb = qb.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		qb = qb.Offset(uint64(filter.Offset))
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list query: %w", err)
	}

	var templates []EmailTemplateLookup
	if err := pgxscan.Select(ctx, p.db, &templates, query, args...); err != nil {
		return nil, fmt.Errorf("list email templates: %w", err)
	}

	return templates, nil
}

// CreateEmailTemplateInput входные данные для создания шаблона
type CreateEmailTemplateInput struct {
	Slug            string
	Name            string
	Subject         string
	BodyHTML        string
	BodyText        string
	Category        string
	VariablesSchema map[string]VariableSpec
	IsSystem        bool
}

// Create создает новый шаблон
func (p *EmailTemplatesProjection) Create(ctx context.Context, input CreateEmailTemplateInput) (*EmailTemplateLookup, error) {
	varsJSON, err := json.Marshal(input.VariablesSchema)
	if err != nil {
		return nil, fmt.Errorf("marshal variables schema: %w", err)
	}

	id := uuid.New()
	now := time.Now()

	query, args, err := p.psql.
		Insert("email_templates").
		Columns(
			"id", "slug", "name", "subject", "body_html", "body_text",
			"category", "variables_schema", "is_system", "is_active",
			"created_at", "updated_at",
		).
		Values(
			id, input.Slug, input.Name, input.Subject, input.BodyHTML, input.BodyText,
			input.Category, varsJSON, input.IsSystem, true,
			now, now,
		).
		Suffix("RETURNING id, slug, name, subject, body_html, body_text, category, variables_schema, is_system, is_active, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert query: %w", err)
	}

	var tpl EmailTemplateLookup
	if err := pgxscan.Get(ctx, p.db, &tpl, query, args...); err != nil {
		return nil, fmt.Errorf("create email template: %w", err)
	}

	return &tpl, nil
}

// UpdateEmailTemplateInput входные данные для обновления шаблона
type UpdateEmailTemplateInput struct {
	Name            *string
	Subject         *string
	BodyHTML        *string
	BodyText        *string
	Category        *string
	VariablesSchema map[string]VariableSpec
	IsActive        *bool
}

// Update обновляет шаблон (кроме slug и is_system)
func (p *EmailTemplatesProjection) Update(ctx context.Context, id uuid.UUID, input UpdateEmailTemplateInput) (*EmailTemplateLookup, error) {
	qb := p.psql.Update("email_templates").Where(squirrel.Eq{"id": id})

	if input.Name != nil {
		qb = qb.Set("name", *input.Name)
	}
	if input.Subject != nil {
		qb = qb.Set("subject", *input.Subject)
	}
	if input.BodyHTML != nil {
		qb = qb.Set("body_html", *input.BodyHTML)
	}
	if input.BodyText != nil {
		qb = qb.Set("body_text", *input.BodyText)
	}
	if input.Category != nil {
		qb = qb.Set("category", *input.Category)
	}
	if input.VariablesSchema != nil {
		varsJSON, err := json.Marshal(input.VariablesSchema)
		if err != nil {
			return nil, fmt.Errorf("marshal variables schema: %w", err)
		}
		qb = qb.Set("variables_schema", varsJSON)
	}
	if input.IsActive != nil {
		qb = qb.Set("is_active", *input.IsActive)
	}

	qb = qb.Set("updated_at", time.Now())

	query, args, err := qb.
		Suffix("RETURNING id, slug, name, subject, body_html, body_text, category, variables_schema, is_system, is_active, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update query: %w", err)
	}

	var tpl EmailTemplateLookup
	if err := pgxscan.Get(ctx, p.db, &tpl, query, args...); err != nil {
		return nil, fmt.Errorf("update email template: %w", err)
	}

	return &tpl, nil
}

// Delete удаляет шаблон (только не системные)
func (p *EmailTemplatesProjection) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := p.psql.
		Delete("email_templates").
		Where(squirrel.Eq{"id": id, "is_system": false}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete email template: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("template not found or is system template")
	}

	return nil
}

// RenderBySlug загружает шаблон и рендерит его
func (p *EmailTemplatesProjection) RenderBySlug(ctx context.Context, slug string, data map[string]any) (*RenderedEmail, error) {
	tpl, err := p.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	rendered, err := tpl.Render(data)
	if err != nil {
		return nil, fmt.Errorf("render template %s: %w", slug, err)
	}

	return rendered, nil
}

// Count возвращает количество шаблонов с учетом фильтров
func (p *EmailTemplatesProjection) Count(ctx context.Context, filter EmailTemplateListFilter) (int, error) {
	qb := p.psql.
		Select("COUNT(*)").
		From("email_templates")

	if filter.Category != nil {
		qb = qb.Where(squirrel.Eq{"category": *filter.Category})
	}
	if filter.IsActive != nil {
		qb = qb.Where(squirrel.Eq{"is_active": *filter.IsActive})
	}
	if filter.IsSystem != nil {
		qb = qb.Where(squirrel.Eq{"is_system": *filter.IsSystem})
	}
	if filter.SearchText != "" {
		searchPattern := "%" + EscapeLikePattern(filter.SearchText) + "%"
		qb = qb.Where(squirrel.Or{
			squirrel.ILike{"name": searchPattern},
			squirrel.ILike{"slug": searchPattern},
		})
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	if err := pgxscan.Get(ctx, p.db, &count, query, args...); err != nil {
		return 0, fmt.Errorf("count email templates: %w", err)
	}

	return count, nil
}

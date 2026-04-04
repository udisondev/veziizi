package display

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// EntityResolver разрешает UUID сущностей в человекочитаемые имена
type EntityResolver interface {
	// ResolveMember возвращает имя члена по ID
	ResolveMember(ctx context.Context, id uuid.UUID) string

	// ResolveOrganization возвращает название организации по ID
	ResolveOrganization(ctx context.Context, id uuid.UUID) string
}

// CachedResolver кеширует результаты резолва в рамках запроса
type CachedResolver struct {
	members       *projections.MembersProjection
	organizations *projections.OrganizationsProjection

	memberCache map[uuid.UUID]string
	orgCache    map[uuid.UUID]string
	mu          sync.RWMutex
}

// NewCachedResolver создаёт новый CachedResolver
func NewCachedResolver(
	members *projections.MembersProjection,
	organizations *projections.OrganizationsProjection,
) *CachedResolver {
	return &CachedResolver{
		members:       members,
		organizations: organizations,
		memberCache:   make(map[uuid.UUID]string),
		orgCache:      make(map[uuid.UUID]string),
	}
}

// ResolveMember возвращает имя члена по ID
func (r *CachedResolver) ResolveMember(ctx context.Context, id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}

	r.mu.RLock()
	if name, ok := r.memberCache[id]; ok {
		r.mu.RUnlock()
		return name
	}
	r.mu.RUnlock()

	member, err := r.members.GetByID(ctx, id)
	if err != nil {
		slog.Debug("failed to resolve member", slog.String("id", id.String()), slog.String("error", err.Error()))
		return "Неизвестный пользователь"
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// Double-check после взятия лока для предотвращения TOCTOU race condition
	if name, ok := r.memberCache[id]; ok {
		return name
	}
	r.memberCache[id] = member.Name
	return member.Name
}

// ResolveOrganization возвращает название организации по ID
func (r *CachedResolver) ResolveOrganization(ctx context.Context, id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}

	r.mu.RLock()
	if name, ok := r.orgCache[id]; ok {
		r.mu.RUnlock()
		return name
	}
	r.mu.RUnlock()

	org, err := r.organizations.GetByID(ctx, id)
	if err != nil {
		slog.Debug("failed to resolve organization", slog.String("id", id.String()), slog.String("error", err.Error()))
		return "Неизвестная организация"
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// Double-check после взятия лока для предотвращения TOCTOU race condition
	if name, ok := r.orgCache[id]; ok {
		return name
	}
	r.orgCache[id] = org.Name
	return org.Name
}

// PreloadMembers загружает имена членов пакетно
func (r *CachedResolver) PreloadMembers(ctx context.Context, ids []uuid.UUID) {
	if len(ids) == 0 {
		return
	}

	// Фильтруем уже закешированные
	toLoad := make([]uuid.UUID, 0, len(ids))
	r.mu.RLock()
	for _, id := range ids {
		if _, ok := r.memberCache[id]; !ok && id != uuid.Nil {
			toLoad = append(toLoad, id)
		}
	}
	r.mu.RUnlock()

	if len(toLoad) == 0 {
		return
	}

	names, err := r.members.GetNames(ctx, toLoad)
	if err != nil {
		slog.Error("failed to preload member names", slog.String("error", err.Error()))
		return
	}

	r.mu.Lock()
	for id, name := range names {
		r.memberCache[id] = name
	}
	r.mu.Unlock()
}

// PreloadOrganizations загружает названия организаций пакетно
func (r *CachedResolver) PreloadOrganizations(ctx context.Context, ids []uuid.UUID) {
	if len(ids) == 0 {
		return
	}

	// Фильтруем уже закешированные
	toLoad := make([]uuid.UUID, 0, len(ids))
	r.mu.RLock()
	for _, id := range ids {
		if _, ok := r.orgCache[id]; !ok && id != uuid.Nil {
			toLoad = append(toLoad, id)
		}
	}
	r.mu.RUnlock()

	if len(toLoad) == 0 {
		return
	}

	names, err := r.organizations.GetNames(ctx, toLoad)
	if err != nil {
		slog.Error("failed to preload organization names", slog.String("error", err.Error()))
		return
	}

	r.mu.Lock()
	for id, name := range names {
		r.orgCache[id] = name
	}
	r.mu.Unlock()
}

package session

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

const (
	KeyAdminID = "admin_id"
)

type AdminManager struct {
	store sessions.Store
	name  string
}

func NewAdminManager(cfg *config.Config) *AdminManager {
	// SEC-006: Использовать отдельный ключ для admin сессий если установлен
	secret := cfg.Session.AdminSecret
	if secret == "" {
		// Fallback на основной secret если AdminSecret не установлен
		secret = cfg.Session.Secret
	}

	store := sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/api/v1/admin",
		MaxAge:   cfg.Session.MaxAge,
		HttpOnly: true,
		Secure:   cfg.IsProduction(),
		SameSite: http.SameSiteStrictMode, // SEC-006: Strict для admin
	}

	return &AdminManager{
		store: store,
		name:  cfg.Session.AdminName,
	}
}

func (m *AdminManager) Get(r *http.Request) (*sessions.Session, error) {
	return m.store.Get(r, m.name)
}

func (m *AdminManager) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	return m.store.Save(r, w, s)
}

func (m *AdminManager) GetAdminID(r *http.Request) (uuid.UUID, bool) {
	session, err := m.Get(r)
	if err != nil {
		return uuid.Nil, false
	}

	idStr, ok := session.Values[KeyAdminID].(string)
	if !ok {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, false
	}

	return id, true
}

func (m *AdminManager) SetAuth(r *http.Request, w http.ResponseWriter, adminID uuid.UUID) error {
	session, err := m.Get(r)
	if err != nil {
		return err
	}

	session.Values[KeyAdminID] = adminID.String()

	return m.Save(r, w, session)
}

// RegenerateAndSetAuth invalidates the old session and creates a new one.
// Prevents session fixation attacks.
func (m *AdminManager) RegenerateAndSetAuth(r *http.Request, w http.ResponseWriter, adminID uuid.UUID) error {
	oldSession, err := m.Get(r)
	if err == nil {
		oldSession.Options.MaxAge = -1
		if err := m.Save(r, w, oldSession); err != nil {
			return fmt.Errorf("invalidate old admin session: %w", err)
		}
	}

	newSession, err := m.store.New(r, m.name)
	if err != nil {
		return fmt.Errorf("create new admin session: %w", err)
	}

	newSession.Values[KeyAdminID] = adminID.String()

	return m.Save(r, w, newSession)
}

func (m *AdminManager) Clear(r *http.Request, w http.ResponseWriter) error {
	session, err := m.Get(r)
	if err != nil {
		return err
	}

	session.Options.MaxAge = -1

	return m.Save(r, w, session)
}

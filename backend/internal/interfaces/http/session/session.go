package session

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

const (
	KeyMemberID       = "member_id"
	KeyOrganizationID = "organization_id"
	KeyRole           = "role"
)

type Manager struct {
	store sessions.Store
	name  string
}

func NewManager(cfg *config.Config) *Manager {
	store := sessions.NewCookieStore([]byte(cfg.Session.Secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   cfg.Session.MaxAge,
		HttpOnly: true,
		Secure:   cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	}

	return &Manager{
		store: store,
		name:  cfg.Session.Name,
	}
}

func (m *Manager) Get(r *http.Request) (*sessions.Session, error) {
	return m.store.Get(r, m.name)
}

func (m *Manager) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	return m.store.Save(r, w, s)
}

func (m *Manager) GetMemberID(r *http.Request) (uuid.UUID, bool) {
	session, err := m.Get(r)
	if err != nil {
		return uuid.Nil, false
	}

	idStr, ok := session.Values[KeyMemberID].(string)
	if !ok {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, false
	}

	return id, true
}

func (m *Manager) GetOrganizationID(r *http.Request) (uuid.UUID, bool) {
	session, err := m.Get(r)
	if err != nil {
		return uuid.Nil, false
	}

	idStr, ok := session.Values[KeyOrganizationID].(string)
	if !ok {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, false
	}

	return id, true
}

func (m *Manager) SetAuth(r *http.Request, w http.ResponseWriter, memberID, orgID uuid.UUID, role string) error {
	session, err := m.Get(r)
	if err != nil {
		return err
	}

	session.Values[KeyMemberID] = memberID.String()
	session.Values[KeyOrganizationID] = orgID.String()
	session.Values[KeyRole] = role

	return m.Save(r, w, session)
}

// RegenerateAndSetAuth invalidates the old session and creates a new one with auth data.
// Prevents session fixation attacks.
func (m *Manager) RegenerateAndSetAuth(r *http.Request, w http.ResponseWriter, memberID, orgID uuid.UUID, role string) error {
	// Invalidate old session
	oldSession, err := m.Get(r)
	if err == nil {
		oldSession.Options.MaxAge = -1
		if err := m.Save(r, w, oldSession); err != nil {
			return fmt.Errorf("invalidate old session: %w", err)
		}
	}

	// Create new session with auth data
	newSession, err := m.store.New(r, m.name)
	if err != nil {
		return fmt.Errorf("create new session: %w", err)
	}

	newSession.Values[KeyMemberID] = memberID.String()
	newSession.Values[KeyOrganizationID] = orgID.String()
	newSession.Values[KeyRole] = role

	return m.Save(r, w, newSession)
}

func (m *Manager) GetRole(r *http.Request) (string, bool) {
	session, err := m.Get(r)
	if err != nil {
		return "", false
	}

	role, ok := session.Values[KeyRole].(string)
	if !ok {
		return "", false
	}

	return role, true
}

func (m *Manager) Clear(r *http.Request, w http.ResponseWriter) error {
	session, err := m.Get(r)
	if err != nil {
		return err
	}

	session.Options.MaxAge = -1

	return m.Save(r, w, session)
}

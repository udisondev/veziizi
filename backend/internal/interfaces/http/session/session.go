package session

import (
	"net/http"

	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
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

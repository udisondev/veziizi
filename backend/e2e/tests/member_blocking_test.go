package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
	"github.com/udisondev/veziizi/backend/e2e/setup"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// MemberBlockingSuite tests member blocking functionality.
type MemberBlockingSuite struct {
	suite.Suite
	baseURL string
	c       *client.Client
	suite   *setup.Suite

	// Shared organization for tests
	org *fixtures.CreatedOrganization
}

func TestMemberBlockingSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MemberBlockingSuite))
}

func (s *MemberBlockingSuite) SetupSuite() {
	s.suite = getSuite(s.T())
	s.baseURL = s.suite.BaseURL
	s.c = client.New(s.baseURL)

	// Create shared organization once
	s.org = fixtures.NewOrganization(s.T(), s.c).Create()
}

// Helper: блокировка member через прямое обновление БД
func (s *MemberBlockingSuite) blockMember(memberID uuid.UUID) {
	db := s.suite.Factory.DB()
	_, err := db.Exec(context.Background(),
		"UPDATE members_lookup SET status = 'blocked' WHERE id = $1",
		memberID)
	s.Require().NoError(err, "failed to block member")
}

// Helper: разблокировка member через прямое обновление БД
func (s *MemberBlockingSuite) unblockMember(memberID uuid.UUID) {
	db := s.suite.Factory.DB()
	_, err := db.Exec(context.Background(),
		"UPDATE members_lookup SET status = 'active' WHERE id = $1",
		memberID)
	s.Require().NoError(err, "failed to unblock member")
}

// ==================== БЛОК-001: Active member имеет доступ ====================

func (s *MemberBlockingSuite) TestBLOCK001_ActiveMemberHasAccess() {
	testClient := s.c.Clone()

	// Login
	loginResp, err := testClient.Login(s.org.OwnerEmail, s.org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode, string(loginResp.RawBody))

	// Проверка доступа к API
	frResp, err := testClient.GetFreightRequests(nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, frResp.StatusCode, string(frResp.RawBody))
}

// ==================== БЛОК-002: Blocked member получает 403 ====================

func (s *MemberBlockingSuite) TestBLOCK002_BlockedMemberDenied() {
	// Создаём отдельную организацию для этого теста
	org := fixtures.NewOrganization(s.T(), s.c).Create()
	testClient := s.c.Clone()

	// Login (статус active)
	loginResp, err := testClient.Login(org.OwnerEmail, org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode, string(loginResp.RawBody))

	// Проверка что доступ есть
	frResp1, err := testClient.GetFreightRequests(nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, frResp1.StatusCode, string(frResp1.RawBody))

	// Блокируем member
	s.blockMember(loginResp.Body.MemberID)

	// Следующий запрос должен быть заблокирован
	frResp2, err := testClient.GetFreightRequests(nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, frResp2.StatusCode, string(frResp2.RawBody))
	s.Assert().Contains(strings.ToLower(string(frResp2.RawBody)), "account is blocked")
}

// ==================== БЛОК-003: /auth/me доступен для заблокированных ====================

func (s *MemberBlockingSuite) TestBLOCK003_AuthMeAccessibleForBlocked() {
	// Создаём отдельную организацию для этого теста
	org := fixtures.NewOrganization(s.T(), s.c).Create()
	testClient := s.c.Clone()

	// Login
	loginResp, err := testClient.Login(org.OwnerEmail, org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode, string(loginResp.RawBody))

	// Блокируем member
	s.blockMember(loginResp.Body.MemberID)

	// /auth/me должен работать (пользователь видит свой статус)
	meResp, err := testClient.Me()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, meResp.StatusCode, string(meResp.RawBody))
	// Status field should contain "blocked"
	if meResp.StatusCode == http.StatusOK {
		s.Assert().Equal("blocked", meResp.Body.Status, "status should be blocked")
	}
}

// ==================== БЛОК-004: Разблокировка восстанавливает доступ ====================

func (s *MemberBlockingSuite) TestBLOCK004_UnblockRestoresAccess() {
	// Создаём отдельную организацию для этого теста
	org := fixtures.NewOrganization(s.T(), s.c).Create()
	testClient := s.c.Clone()

	// Login
	loginResp, err := testClient.Login(org.OwnerEmail, org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode, string(loginResp.RawBody))

	// Блокируем member
	s.blockMember(loginResp.Body.MemberID)

	// Проверка что доступ заблокирован
	frResp1, err := testClient.GetFreightRequests(nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, frResp1.StatusCode, string(frResp1.RawBody))

	// Разблокируем member
	s.unblockMember(loginResp.Body.MemberID)

	// Доступ восстановлен
	frResp2, err := testClient.GetFreightRequests(nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, frResp2.StatusCode, string(frResp2.RawBody))
}

// ==================== БЛОК-005: Публичные endpoints доступны ====================

func (s *MemberBlockingSuite) TestBLOCK005_PublicEndpointsAccessible() {
	// Создаём отдельную организацию для этого теста
	org := fixtures.NewOrganization(s.T(), s.c).Create()
	testClient := s.c.Clone()

	// Login
	loginResp, err := testClient.Login(org.OwnerEmail, org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode, string(loginResp.RawBody))

	// Блокируем member
	s.blockMember(loginResp.Body.MemberID)

	// Public org profile (должен работать)
	orgResp, err := testClient.GetOrganization(org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, orgResp.StatusCode, string(orgResp.RawBody))

	// Geo endpoints (должны работать)
	geoResp, err := testClient.GetCountries()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, geoResp.StatusCode, string(geoResp.RawBody))
}

// ==================== БЛОК-006: Logout работает для заблокированных ====================

func (s *MemberBlockingSuite) TestBLOCK006_LogoutWorksForBlocked() {
	// Создаём отдельную организацию для этого теста
	org := fixtures.NewOrganization(s.T(), s.c).Create()
	testClient := s.c.Clone()

	// Login
	loginResp, err := testClient.Login(org.OwnerEmail, org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode, string(loginResp.RawBody))

	// Блокируем member
	s.blockMember(loginResp.Body.MemberID)

	// Logout должен работать (204 No Content)
	logoutResp, err := testClient.Logout()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, logoutResp.StatusCode, string(logoutResp.RawBody))
}

// ==================== БЛОК-007: Login блокируется для заблокированных ====================

func (s *MemberBlockingSuite) TestBLOCK007_LoginBlockedForBlockedMember() {
	// Создаём новую организацию для этого теста
	newOrg := fixtures.NewOrganization(s.T(), s.c).Create()
	testClient := s.c.Clone()

	// Блокируем member ДО логина
	s.blockMember(newOrg.MemberID)

	// Попытка логина должна быть отклонена (403 Forbidden from auth.go Login handler)
	loginResp, err := testClient.Login(newOrg.OwnerEmail, newOrg.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, loginResp.StatusCode, string(loginResp.RawBody))
	s.Assert().Contains(strings.ToLower(string(loginResp.RawBody)), "blocked")
}

// ==================== БЛОК-008: Заблокированный member не может смотреть профили ====================

func (s *MemberBlockingSuite) TestBLOCK008_CannotAccessMemberProfiles() {
	// Создаём отдельную организацию для этого теста
	org := fixtures.NewOrganization(s.T(), s.c).Create()
	testClient := s.c.Clone()

	// Создаём второго member в организации через invitation
	invResp, err := testClient.Login(org.OwnerEmail, org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, invResp.StatusCode)

	createInvResp, err := testClient.CreateInvitation(org.OrganizationID, client.CreateInvitationRequest{
		Email: "member2@test.local",
		Role:  "employee",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, createInvResp.StatusCode)

	// Wait for invitation to be available (projection sync)
	token := createInvResp.Body.Token
	helpers.WaitFor(s.T(), func() (bool, bool) {
		getResp, err := s.c.GetInvitationByToken(token)
		return err == nil && getResp.StatusCode == 200, err == nil && getResp.StatusCode == 200
	}, "invitation should be available")

	// Accept invitation
	member2Client := s.c.Clone()
	acceptResp, err := member2Client.AcceptInvitation(token, client.AcceptInvitationRequest{
		Password: "password123",
		Name:     helpers.StringPtr("Member 2"),
		Phone:    helpers.StringPtr("+79001234567"),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, acceptResp.StatusCode)

	member2ID := acceptResp.Body.MemberID

	// Wait for member to be available (projection sync)
	helpers.WaitFor(s.T(), func() (bool, bool) {
		loginResp, err := member2Client.Login("member2@test.local", "password123")
		return err == nil && loginResp.StatusCode == 200, err == nil && loginResp.StatusCode == 200
	}, "member should be available")

	// Логинимся под owner заново (используем новый клиент)
	ownerClient := s.c.Clone()
	loginResp, err := ownerClient.Login(org.OwnerEmail, org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode)

	// Проверяем что active member может смотреть профиль другого member
	profileResp, err := ownerClient.GetMemberProfile(member2ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, profileResp.StatusCode, string(profileResp.RawBody))

	// Блокируем owner
	s.blockMember(loginResp.Body.MemberID)

	// Попытка получить профиль member2 должна вернуть 403
	profileResp2, err := ownerClient.GetMemberProfile(member2ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, profileResp2.StatusCode, string(profileResp2.RawBody))
	s.Assert().Contains(strings.ToLower(string(profileResp2.RawBody)), "account is blocked")
}

// ==================== БЛОК-009: Заблокированный member не может получить полный профиль org ====================

func (s *MemberBlockingSuite) TestBLOCK009_CannotAccessOrgFullWithMembers() {
	// Создаём отдельную организацию для этого теста
	org := fixtures.NewOrganization(s.T(), s.c).Create()
	testClient := s.c.Clone()

	// Логинимся под owner
	loginResp, err := testClient.Login(org.OwnerEmail, org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode)

	// Проверяем что active member может получить полный профиль организации
	fullResp, err := testClient.GetOrganizationFull(org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, fullResp.StatusCode, string(fullResp.RawBody))
	s.Require().NotEmpty(fullResp.Body.Members, "members should not be empty")

	// Блокируем owner
	s.blockMember(loginResp.Body.MemberID)

	// Попытка получить полный профиль должна вернуть 403
	fullResp2, err := testClient.GetOrganizationFull(org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, fullResp2.StatusCode, string(fullResp2.RawBody))
	s.Assert().Contains(strings.ToLower(string(fullResp2.RawBody)), "account is blocked")
}

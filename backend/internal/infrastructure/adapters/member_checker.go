package adapters

import (
	"context"
	"fmt"

	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	orgApp "github.com/udisondev/veziizi/backend/internal/application/organization"
	"github.com/google/uuid"
)

// MemberCheckerAdapter проверяет принадлежность и права членов организации
// через Organization aggregate.
type MemberCheckerAdapter struct {
	orgService *orgApp.Service
}

// NewMemberCheckerAdapter создает адаптер
func NewMemberCheckerAdapter(orgService *orgApp.Service) *MemberCheckerAdapter {
	return &MemberCheckerAdapter{orgService: orgService}
}

func (a *MemberCheckerAdapter) MemberExists(ctx context.Context, orgID, memberID uuid.UUID) error {
	org, err := a.orgService.Get(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get organization: %w", err)
	}
	if _, ok := org.GetMember(memberID); !ok {
		return organization.ErrMemberNotFound
	}
	return nil
}

func (a *MemberCheckerAdapter) CanManageMembers(ctx context.Context, orgID, memberID uuid.UUID) (orgValues.MemberRole, error) {
	org, err := a.orgService.Get(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("get organization: %w", err)
	}
	member, ok := org.GetMember(memberID)
	if !ok {
		return "", organization.ErrMemberNotFound
	}
	if !member.CanManageMembers() {
		return "", organization.ErrInsufficientPermissions
	}
	return member.Role(), nil
}

package fixtures

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
)

// registeredFactory is set once from the setup package so fixtures can run
// direct DB checks (e.g. wait until a projection has caught up) instead of
// polling HTTP endpoints and masking event-consumer lag under load.
var (
	registeredFactory   *factory.Factory
	registeredFactoryMu sync.RWMutex
)

// RegisterFactory wires a Factory into the fixtures package. Called from
// setup.newSuite. Safe to call multiple times — last registration wins.
func RegisterFactory(f *factory.Factory) {
	registeredFactoryMu.Lock()
	defer registeredFactoryMu.Unlock()
	registeredFactory = f
}

// GetFactory returns the factory registered via RegisterFactory, or nil if
// none is registered (e.g. fixtures called from outside an e2e suite).
func GetFactory() *factory.Factory {
	registeredFactoryMu.RLock()
	defer registeredFactoryMu.RUnlock()
	return registeredFactory
}

// pollWithBackoff runs fn until it returns true or deadline passes.
// On timeout calls t.Fatalf(failMsgFmt, failMsgArgs...).
// Uses exponential backoff from 5ms up to 200ms.
func pollWithBackoff(
	t *testing.T,
	deadline time.Duration,
	fn func() bool,
	failMsgFmt string,
	failMsgArgs ...any,
) {
	t.Helper()
	end := time.Now().Add(deadline)
	backoff := 5 * time.Millisecond
	const maxBackoff = 200 * time.Millisecond
	for {
		if fn() {
			return
		}
		if time.Now().After(end) {
			t.Fatalf(failMsgFmt, failMsgArgs...)
		}
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// WaitMemberInProjection blocks until members_lookup contains a member with
// the given email and status=active, or until deadline. Returns the member ID.
// Waits directly at DB level — HTTP-based polling masks consumer lag as
// "invalid credentials" and forces tests to retry blindly.
func WaitMemberInProjection(t *testing.T, email string, deadline time.Duration) uuid.UUID {
	t.Helper()

	f := GetFactory()
	if f == nil {
		t.Fatalf("WaitMemberInProjection: no factory registered — call fixtures.RegisterFactory in setup")
	}

	ctx := context.Background()
	var (
		foundID    uuid.UUID
		lastErr    error
		lastStatus string
	)

	pollWithBackoff(t, deadline, func() bool {
		var (
			id     uuid.UUID
			status string
		)
		err := f.DB().QueryRow(ctx,
			`SELECT id, status FROM members_lookup WHERE email = $1`, email,
		).Scan(&id, &status)
		switch {
		case err == nil:
			lastStatus = status
			if status == "active" {
				foundID = id
				return true
			}
		case errors.Is(err, pgx.ErrNoRows):
			// projection still catching up
		default:
			lastErr = err
		}
		return false
	}, "WaitMemberInProjection: member %s not active after %s (last status=%q, last err=%v)",
		email, deadline, lastStatus, lastErr)

	return foundID
}

// WaitInvitationByToken blocks until invitations_lookup contains an invitation
// with the given token. Returns when the row is visible at DB level, so a
// subsequent AcceptInvitation is safe.
func WaitInvitationByToken(t *testing.T, token string, deadline time.Duration) {
	t.Helper()

	f := GetFactory()
	if f == nil {
		t.Fatalf("WaitInvitationByToken: no factory registered — call fixtures.RegisterFactory in setup")
	}

	ctx := context.Background()
	var lastErr error

	pollWithBackoff(t, deadline, func() bool {
		var exists bool
		err := f.DB().QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM invitations_lookup WHERE token = $1)`, token,
		).Scan(&exists)
		if err != nil {
			lastErr = err
			return false
		}
		return exists
	}, "WaitInvitationByToken: invitation token=%s not visible after %s (last err=%v)",
		token, deadline, lastErr)
}

// WaitOrgIsFraudster blocks until org_reviewer_reputation marks the given
// organization as confirmed fraudster. Used after AdminMarkFraudster to wait
// for the fraudster-handler worker to update the projection.
func WaitOrgIsFraudster(t *testing.T, orgID uuid.UUID, deadline time.Duration) {
	t.Helper()

	f := GetFactory()
	if f == nil {
		t.Fatalf("WaitOrgIsFraudster: no factory registered — call fixtures.RegisterFactory in setup")
	}

	ctx := context.Background()
	var lastErr error

	pollWithBackoff(t, deadline, func() bool {
		var confirmed bool
		err := f.DB().QueryRow(ctx,
			`SELECT COALESCE(is_confirmed_fraudster, false)
			 FROM org_reviewer_reputation WHERE org_id = $1`, orgID,
		).Scan(&confirmed)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false // projection still catching up
			}
			lastErr = err
			return false
		}
		return confirmed
	}, "WaitOrgIsFraudster: org %s not marked as fraudster after %s (last err=%v)",
		orgID, deadline, lastErr)
}

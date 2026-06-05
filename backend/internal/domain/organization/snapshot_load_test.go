package organization_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// TestSnapshotRoundTrip проверяет контракт снапшотов: State() → JSON →
// NewFromStore восстанавливает эквивалентное состояние. Это путь, который в
// проде включается только на каждой snapshotThreshold-й версии агрегата —
// e2e его не достигают, поэтому ломкость JSON-round-trip'а value-типов ловим
// здесь.
func TestSnapshotRoundTrip(t *testing.T) {
	id := uuid.New()
	ownerID := uuid.New()

	org := organization.New(
		id, "Org", "1234567890", "OOO Org",
		values.Country("RU"), "+79991234567", "org@example.com", values.Address("Москва, ул. Тестовая 1"),
		ownerID, "owner@example.com", "hash", "Owner", "+79991234567",
		"1.2.3.4", "fingerprint", "user-agent",
	)
	require.NoError(t, org.Approve(uuid.New()))

	data, err := json.Marshal(org.State())
	require.NoError(t, err)

	restored, err := organization.NewFromStore(id, data, nil)
	require.NoError(t, err)

	require.Equal(t, org.Version(), restored.Version())
	require.Equal(t, org.Name(), restored.Name())
	require.Equal(t, org.INN(), restored.INN())
	require.Equal(t, org.LegalName(), restored.LegalName())
	require.Equal(t, org.Country(), restored.Country())
	require.Equal(t, org.Status(), restored.Status())

	require.Len(t, restored.MembersList(), 1)
	m := restored.MembersList()[0]
	require.Equal(t, ownerID, m.ID())
	require.Equal(t, "owner@example.com", m.Email())
	require.Equal(t, values.MemberRoleOwner, m.Role())
	require.Equal(t, values.MemberStatusActive, m.Status())
}

// TestSnapshotTailReplay проверяет доигрывание событий после снапшота:
// NewFromStore(snapshot, tail) обязан применить хвост поверх снапшота.
func TestSnapshotTailReplay(t *testing.T) {
	id := uuid.New()
	ownerID := uuid.New()

	org := organization.New(
		id, "Org", "1234567890", "OOO Org",
		values.Country("RU"), "+79991234567", "org@example.com", values.Address("Москва"),
		ownerID, "owner@example.com", "hash", "Owner", "+79991234567",
		"", "", "",
	)

	data, err := json.Marshal(org.State())
	require.NoError(t, err)

	blocked := events.MemberBlocked{
		BaseEvent: eventstore.NewBaseEvent(id, events.AggregateType, org.Version()+1),
		MemberID:  ownerID,
		BlockedBy: uuid.New(),
	}

	restored, err := organization.NewFromStore(id, data, []eventstore.Event{blocked})
	require.NoError(t, err)

	require.Equal(t, org.Version()+1, restored.Version())
	require.Len(t, restored.MembersList(), 1)
	require.Equal(t, values.MemberStatusBlocked, restored.MembersList()[0].Status())
}

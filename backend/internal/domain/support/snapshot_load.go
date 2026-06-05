package support

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// NewFromStore восстанавливает агрегат из результата eventstore.LoadWithSnapshot:
// снапшот (если есть) + события после него. Когда снапшота нет — обычный
// NewFromEvents. Все места загрузки агрегата обязаны идти через LoadWithSnapshot
// + NewFromStore: после включения снапшотов (SaveWithState) события до версии
// снапшота в выдачу не попадают, и NewFromEvents на таком хвосте собрал бы
// битое состояние.
func NewFromStore(id uuid.UUID, snapshotJSON []byte, evts []eventstore.Event) (*Ticket, error) {
	if len(snapshotJSON) == 0 {
		return NewFromEvents(id, evts), nil
	}

	var snap TicketSnapshot
	if err := json.Unmarshal(snapshotJSON, &snap); err != nil {
		return nil, fmt.Errorf("unmarshal support ticket snapshot: %w", err)
	}

	t, err := NewFromSnapshot(id, snap)
	if err != nil {
		return nil, err
	}
	for _, evt := range evts {
		t.apply(evt)
		t.Replay(evt)
	}
	return t, nil
}

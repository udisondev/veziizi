package freightrequest_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
)

// TestSnapshotRoundTrip проверяет контракт снапшотов FreightRequest: State() →
// JSON → NewFromStore восстанавливает эквивалентное состояние (включая офферы).
// В проде путь включается только на каждой snapshotThreshold-й версии — e2e его
// не достигают.
func TestSnapshotRoundTrip(t *testing.T) {
	id := uuid.New()
	customerOrgID := uuid.New()
	customerMemberID := uuid.New()
	offerID := uuid.New()

	route := values.Route{Points: []values.RoutePoint{
		{IsLoading: true, Address: "Москва"},
		{IsUnloading: true, Address: "Санкт-Петербург"},
	}}
	cargo := values.CargoInfo{Description: "Палеты", Weight: 1000, Quantity: 2}
	// Subtype обязан быть валидным: enum'ы отвергают пустую строку при
	// json.Unmarshal — снапшот с пустым subtype не десериализуется (в реальных
	// агрегатах subtype валидируется сервисом при создании/обновлении).
	vehicleReqs := values.VehicleRequirements{
		VehicleType:    values.VehicleType("van"),
		VehicleSubType: values.VehicleSubType("city_van"),
	}
	payment := values.Payment{
		Price:   &values.Money{Amount: 100_000, Currency: values.CurrencyRUB},
		VatType: values.VatTypeIncluded,
		Method:  values.PaymentMethodBankTransfer,
		Terms:   values.PaymentTermsPrepaid,
	}

	fr := freightrequest.New(
		id, 42, customerOrgID, customerMemberID,
		route, cargo, vehicleReqs, payment, "комментарий",
		time.Now().Add(24*time.Hour),
	)
	require.NoError(t, fr.MakeOffer(
		offerID, uuid.New(), uuid.New(),
		values.Money{Amount: 90_000, Currency: values.CurrencyRUB},
		"наш оффер", values.VatTypeIncluded, values.PaymentMethodBankTransfer,
	))

	data, err := json.Marshal(fr.State())
	require.NoError(t, err)

	restored, err := freightrequest.NewFromStore(id, data, nil)
	require.NoError(t, err)

	require.Equal(t, fr.Version(), restored.Version())
	require.Equal(t, fr.RequestNumber(), restored.RequestNumber())
	require.Equal(t, fr.CustomerOrgID(), restored.CustomerOrgID())
	require.Equal(t, fr.Status(), restored.Status())
	require.Equal(t, fr.Route(), restored.Route())
	require.Equal(t, fr.Cargo(), restored.Cargo())

	require.Len(t, restored.OffersList(), 1)
	offer := restored.OffersList()[0]
	require.Equal(t, offerID, offer.ID())
	require.Equal(t, fr.OffersList()[0].Status(), offer.Status())
}

// TestSnapshotRoundTripLegacyEmptyEnums: у агрегатов, собранных из старых
// событий, enum-поля могут быть пустыми (ключ отсутствовал в JSON события).
// go-enum'овский UnmarshalText отвергает "" — без omitempty на этих полях
// marshal такого состояния давал бы нечитаемый снапшот (и нечитаемое событие).
func TestSnapshotRoundTripLegacyEmptyEnums(t *testing.T) {
	id := uuid.New()

	route := values.Route{Points: []values.RoutePoint{
		{IsLoading: true, Address: "Москва"},
		{IsUnloading: true, Address: "Тверь"},
	}}

	fr := freightrequest.New(
		id, 7, uuid.New(), uuid.New(),
		route,
		values.CargoInfo{Description: "Груз", Weight: 100, Quantity: 1},
		values.VehicleRequirements{}, // пустые VehicleType/VehicleSubType
		values.Payment{},             // пустые VatType/Method/Terms
		"", time.Now().Add(24*time.Hour),
	)

	data, err := json.Marshal(fr.State())
	require.NoError(t, err)

	restored, err := freightrequest.NewFromStore(id, data, nil)
	require.NoError(t, err, "снапшот с пустыми enum-полями обязан читаться")
	require.Equal(t, fr.Version(), restored.Version())
	require.Equal(t, fr.VehicleRequirements(), restored.VehicleRequirements())
	require.Equal(t, fr.Payment(), restored.Payment())
}

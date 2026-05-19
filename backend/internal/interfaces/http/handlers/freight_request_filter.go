package handlers

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// FreightRequestFilters — декларативное описание query-параметров
// для GET /api/v1/freight-requests и GET /api/v1/freight-requests/count.
//
// Парсится через httputil.DecodeQuery (schema + validator/v10).
// SEC-инвариант видимости (WithVisibleToOrg) применяется в handler ДО
// AppendOptions — пользовательские фильтры могут только сужать выдачу.
// Доменные enum'ы статусов проверяются в ValidateStatuses (validator-теги
// не умеют валидировать через доменный парсер).
type FreightRequestFilters struct {
	CustomerOrgID *uuid.UUID `schema:"customer_org_id"`
	MemberID      *uuid.UUID `schema:"member_id"`
	CarrierOrgID  *uuid.UUID `schema:"carrier_org_id"`

	Statuses []string `schema:"statuses"`

	OrgName    string `schema:"org_name"    validate:"omitempty,max=100"`
	OrgINN     string `schema:"org_inn"     validate:"omitempty,max=32"`
	OrgCountry string `schema:"org_country" validate:"omitempty,max=8"`

	RequestNumber *int64 `schema:"request_number" validate:"omitempty,gt=0"`

	MinWeight *float64 `schema:"min_weight" validate:"omitempty,gte=0"`
	MaxWeight *float64 `schema:"max_weight" validate:"omitempty,gte=0"`
	MinPrice  *int64   `schema:"min_price"  validate:"omitempty,gte=0"`
	MaxPrice  *int64   `schema:"max_price"  validate:"omitempty,gte=0"`
	MinVolume *float64 `schema:"min_volume" validate:"omitempty,gte=0"`
	MaxVolume *float64 `schema:"max_volume" validate:"omitempty,gte=0"`

	VehicleTypes    []string `schema:"vehicle_types"`
	VehicleSubTypes []string `schema:"vehicle_subtypes"`

	RouteCityIDs    []int `schema:"route_city_ids"`
	RouteCountryIDs []int `schema:"route_country_ids"`

	// Role-aware фильтры: точка маршрута явно помечена как начало или конец.
	// Семантика та же, что у RouteCity/RouteCountryIDs — `@> ?` по массиву,
	// но проверяется только первая (origin) / последняя (destination) точка.
	OriginCityIDs         []int `schema:"origin_city_ids"`
	OriginCountryIDs      []int `schema:"origin_country_ids"`
	DestinationCityIDs    []int `schema:"destination_city_ids"`
	DestinationCountryIDs []int `schema:"destination_country_ids"`

	PaymentMethods []string `schema:"payment_methods"`
	PaymentTerms   []string `schema:"payment_terms"`
	VatTypes       []string `schema:"vat_types"`

	HasPendingOffers bool `schema:"has_pending_offers"`
}

// ValidateStatuses проверяет, что каждое значение Statuses — известный
// доменный статус заявки. Остальные enum'ы (vehicle/payment/vat) парсятся
// и валидируются ниже по стеку — на уровне проекции / squirrel-запроса.
func (f *FreightRequestFilters) ValidateStatuses() error {
	for _, s := range f.Statuses {
		if _, err := values.ParseFreightRequestStatus(s); err != nil {
			return fmt.Errorf("invalid status %q", s)
		}
	}
	return nil
}

// AppendOptions добавляет в opts опции, соответствующие непустым фильтрам,
// и возвращает результирующий слайс. Не применяет SEC-инвариант видимости —
// это ответственность handler'а (см. decodeFreightRequestFilters).
//
// Передавайте `opts` уже с WithVisibleToOrg / WithStatuses(["published"])
// внутри, чтобы пользовательские фильтры (customer_org_id/carrier_org_id/
// statuses/...) только сужали выдачу.
func (f *FreightRequestFilters) AppendOptions(opts []projections.FilterOption) []projections.FilterOption {
	if f.CustomerOrgID != nil {
		opts = append(opts, projections.WithCustomerOrgID(*f.CustomerOrgID))
	}
	if f.MemberID != nil {
		opts = append(opts, projections.WithCustomerMemberID(*f.MemberID))
	}
	if f.CarrierOrgID != nil {
		opts = append(opts, projections.WithFreightCarrierOrgID(*f.CarrierOrgID))
	}
	if len(f.Statuses) > 0 {
		opts = append(opts, projections.WithStatuses(f.Statuses))
	}

	if f.OrgName != "" {
		opts = append(opts, projections.WithOrgNameLike(f.OrgName))
	}
	if f.OrgINN != "" {
		opts = append(opts, projections.WithOrgINN(f.OrgINN))
	}
	if f.OrgCountry != "" {
		opts = append(opts, projections.WithOrgCountry(f.OrgCountry))
	}
	if f.RequestNumber != nil {
		opts = append(opts, projections.WithRequestNumber(*f.RequestNumber))
	}

	if f.MinWeight != nil {
		opts = append(opts, projections.WithMinWeight(*f.MinWeight))
	}
	if f.MaxWeight != nil {
		opts = append(opts, projections.WithMaxWeight(*f.MaxWeight))
	}
	if f.MinPrice != nil {
		opts = append(opts, projections.WithMinPrice(*f.MinPrice))
	}
	if f.MaxPrice != nil {
		opts = append(opts, projections.WithMaxPrice(*f.MaxPrice))
	}
	if f.MinVolume != nil {
		opts = append(opts, projections.WithMinVolume(*f.MinVolume))
	}
	if f.MaxVolume != nil {
		opts = append(opts, projections.WithMaxVolume(*f.MaxVolume))
	}

	if len(f.VehicleTypes) > 0 {
		opts = append(opts, projections.WithVehicleTypes(f.VehicleTypes))
	}
	if len(f.VehicleSubTypes) > 0 {
		opts = append(opts, projections.WithVehicleSubTypes(f.VehicleSubTypes))
	}
	if len(f.RouteCityIDs) > 0 {
		opts = append(opts, projections.WithRouteCities(f.RouteCityIDs))
	}
	if len(f.RouteCountryIDs) > 0 {
		opts = append(opts, projections.WithRouteCountries(f.RouteCountryIDs))
	}
	if len(f.OriginCityIDs) > 0 {
		opts = append(opts, projections.WithOriginCities(f.OriginCityIDs))
	}
	if len(f.OriginCountryIDs) > 0 {
		opts = append(opts, projections.WithOriginCountries(f.OriginCountryIDs))
	}
	if len(f.DestinationCityIDs) > 0 {
		opts = append(opts, projections.WithDestinationCities(f.DestinationCityIDs))
	}
	if len(f.DestinationCountryIDs) > 0 {
		opts = append(opts, projections.WithDestinationCountries(f.DestinationCountryIDs))
	}
	if len(f.PaymentMethods) > 0 {
		opts = append(opts, projections.WithPaymentMethods(f.PaymentMethods))
	}
	if len(f.PaymentTerms) > 0 {
		opts = append(opts, projections.WithPaymentTerms(f.PaymentTerms))
	}
	if len(f.VatTypes) > 0 {
		opts = append(opts, projections.WithVatTypes(f.VatTypes))
	}
	if f.HasPendingOffers {
		opts = append(opts, projections.WithHasPendingOffers())
	}

	return opts
}

package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
)

// buildSQL применяет переданный набор FilterOption к базовому SELECT FROM freight_requests_lookup
// и возвращает итоговый SQL — это позволяет проверять, какие условия добавились,
// без поднятия реальной БД.
func buildSQL(t *testing.T, opts []projections.FilterOption) (string, []any) {
	t.Helper()
	b := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).
		Select("id").From("freight_requests_lookup")
	for _, o := range opts {
		b = o(b)
	}
	sql, args, err := b.ToSql()
	if err != nil {
		t.Fatalf("build sql: %v", err)
	}
	return sql, args
}

// decodeFilters прогоняет query-строку через httputil.DecodeQuery, как это делает handler.
func decodeFilters(t *testing.T, rawQuery string) (FreightRequestFilters, error) {
	t.Helper()
	r := httptest.NewRequest("GET", "/api/v1/freight-requests/count?"+rawQuery, nil)
	var f FreightRequestFilters
	err := httputil.DecodeQuery(r, &f)
	return f, err
}

// ============================================================================
// AppendOptions: пользовательские фильтры поверх SEC-инварианта.
// SEC-инвариант (WithVisibleToOrg / forced published) применяется в handler.
// Здесь мы проверяем только, что AppendOptions корректно отображает поля DTO
// в squirrel-условия.
// ============================================================================

func TestAppendOptions_Empty_AddsNothing(t *testing.T) {
	var f FreightRequestFilters
	opts := f.AppendOptions(nil)
	if len(opts) != 0 {
		t.Errorf("empty filters must not add options, got %d", len(opts))
	}
}

func TestAppendOptions_OwnershipFilters(t *testing.T) {
	customerOrg := uuid.New()
	member := uuid.New()
	carrierOrg := uuid.New()

	f := FreightRequestFilters{
		CustomerOrgID: &customerOrg,
		MemberID:      &member,
		CarrierOrgID:  &carrierOrg,
	}
	sql, _ := buildSQL(t, f.AppendOptions(nil))

	wants := []string{
		"customer_org_id =",
		"customer_member_id =",
		"carrier_org_id =",
	}
	for _, w := range wants {
		if !strings.Contains(sql, w) {
			t.Errorf("expected %q in SQL, got: %s", w, sql)
		}
	}
}

func TestAppendOptions_Statuses_AppliedAsFilter(t *testing.T) {
	f := FreightRequestFilters{Statuses: []string{"confirmed", "completed"}}
	sql, args := buildSQL(t, f.AppendOptions(nil))

	if !strings.Contains(sql, "status IN ($1,$2)") {
		t.Errorf("expected status IN (...), got: %s", sql)
	}
	if args[0] != "confirmed" || args[1] != "completed" {
		t.Errorf("expected confirmed,completed in args, got %v", args)
	}
}

func TestAppendOptions_Statuses_EmptySliceSkipped(t *testing.T) {
	f := FreightRequestFilters{Statuses: nil}
	sql, _ := buildSQL(t, f.AppendOptions(nil))
	if strings.Contains(sql, "status") {
		t.Errorf("nil statuses must not add filter, got: %s", sql)
	}
}

func TestAppendOptions_StackedOnSECInvariant(t *testing.T) {
	// Эмулируем handler: SEC-invariant первым, поверх — пользовательские фильтры.
	orgID := uuid.New()
	customerOrg := uuid.New()

	base := []projections.FilterOption{projections.WithVisibleToOrg(orgID)}
	f := FreightRequestFilters{
		CustomerOrgID: &customerOrg,
		Statuses:      []string{"confirmed"},
	}
	sql, _ := buildSQL(t, f.AppendOptions(base))

	// Должны быть все три фрагмента: видимость, customer_org_id фильтр, status.
	wants := []string{
		"customer_org_id = $1 OR carrier_org_id = $2 OR status = $3",
		"customer_org_id = $4",
		"status IN ($5)",
	}
	for _, w := range wants {
		if !strings.Contains(sql, w) {
			t.Errorf("expected %q in SQL, got: %s", w, sql)
		}
	}
}

func TestAppendOptions_AllScalarFilters(t *testing.T) {
	customerOrg := uuid.New()
	minW := 100.0
	maxW := 500.0
	minP := int64(1000)
	maxP := int64(9999)
	minV := 5.0
	maxV := 50.0
	reqNum := int64(42)

	f := FreightRequestFilters{
		CustomerOrgID: &customerOrg,
		OrgName:       "Test Co",
		OrgINN:        "7707083893",
		OrgCountry:    "RU",
		RequestNumber: &reqNum,
		MinWeight:     &minW,
		MaxWeight:     &maxW,
		MinPrice:      &minP,
		MaxPrice:      &maxP,
		MinVolume:     &minV,
		MaxVolume:     &maxV,
	}
	sql, _ := buildSQL(t, f.AppendOptions(nil))
	wants := []string{
		"customer_org_id =",
		"customer_org_name ILIKE",
		"customer_org_inn ILIKE",
		"customer_org_country =",
		"request_number =",
		"cargo_weight >=",
		"cargo_weight <=",
		"price_amount >=",
		"price_amount <=",
		"cargo_volume >=",
		"cargo_volume <=",
	}
	for _, w := range wants {
		if !strings.Contains(sql, w) {
			t.Errorf("expected %q in SQL, got: %s", w, sql)
		}
	}
}

func TestAppendOptions_AllListFilters(t *testing.T) {
	f := FreightRequestFilters{
		VehicleTypes:          []string{"truck"},
		VehicleSubTypes:       []string{"tilt"},
		RouteCityIDs:          []int{1, 2},
		RouteCountryIDs:       []int{10},
		OriginCityIDs:         []int{1},
		OriginCountryIDs:      []int{10},
		DestinationCityIDs:    []int{2},
		DestinationCountryIDs: []int{11},
		PaymentMethods:        []string{"cash"},
		PaymentTerms:          []string{"prepaid"},
		VatTypes:              []string{"included"},
		HasPendingOffers:      true,
	}
	sql, _ := buildSQL(t, f.AppendOptions(nil))
	wants := []string{
		"vehicle_type IN",
		"vehicle_subtype IN",
		"route_city_ids @>",
		"route_country_ids @>",
		"origin_city_ids @>",
		"origin_country_ids @>",
		"destination_city_ids @>",
		"destination_country_ids @>",
		"payment_method IN",
		"payment_terms IN",
		"vat_type IN",
		"EXISTS",
	}
	for _, w := range wants {
		if !strings.Contains(sql, w) {
			t.Errorf("expected %q in SQL, got: %s", w, sql)
		}
	}
}

func TestAppendOptions_RoleAwareRouteFilters_OmittedByDefault(t *testing.T) {
	f := FreightRequestFilters{}
	sql, _ := buildSQL(t, f.AppendOptions(nil))
	unwanted := []string{
		"origin_city_ids",
		"origin_country_ids",
		"destination_city_ids",
		"destination_country_ids",
	}
	for _, w := range unwanted {
		if strings.Contains(sql, w) {
			t.Errorf("empty filter must not add %q, got: %s", w, sql)
		}
	}
}

// ============================================================================
// ValidateStatuses
// ============================================================================

func TestValidateStatuses_Empty(t *testing.T) {
	f := FreightRequestFilters{}
	if err := f.ValidateStatuses(); err != nil {
		t.Errorf("empty statuses must not error, got: %v", err)
	}
}

func TestValidateStatuses_AllValid(t *testing.T) {
	f := FreightRequestFilters{Statuses: []string{"published", "confirmed", "completed"}}
	if err := f.ValidateStatuses(); err != nil {
		t.Errorf("valid statuses must not error, got: %v", err)
	}
}

func TestValidateStatuses_Invalid(t *testing.T) {
	f := FreightRequestFilters{Statuses: []string{"published", "garbage"}}
	err := f.ValidateStatuses()
	if err == nil {
		t.Fatalf("invalid status must produce error")
	}
	if !strings.Contains(err.Error(), "garbage") {
		t.Errorf("error should mention bad value, got: %v", err)
	}
}

// ============================================================================
// DecodeQuery integration — through real httputil
// ============================================================================

func TestDecodeQuery_AllParams(t *testing.T) {
	orgID := uuid.New()
	q := "customer_org_id=" + orgID.String() +
		"&statuses=published,confirmed" +
		"&min_weight=100.5&max_weight=500" +
		"&min_price=1000&max_price=9999" +
		"&route_city_ids=1,2,3" +
		"&vehicle_types=truck,van" +
		"&has_pending_offers=true"

	f, err := decodeFilters(t, q)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if f.CustomerOrgID == nil || *f.CustomerOrgID != orgID {
		t.Errorf("CustomerOrgID = %v, want %v", f.CustomerOrgID, orgID)
	}
	if len(f.Statuses) != 2 || f.Statuses[0] != "published" {
		t.Errorf("Statuses = %v", f.Statuses)
	}
	if f.MinWeight == nil || *f.MinWeight != 100.5 {
		t.Errorf("MinWeight = %v", f.MinWeight)
	}
	if len(f.RouteCityIDs) != 3 || f.RouteCityIDs[2] != 3 {
		t.Errorf("RouteCityIDs = %v", f.RouteCityIDs)
	}
	if len(f.VehicleTypes) != 2 {
		t.Errorf("VehicleTypes = %v", f.VehicleTypes)
	}
	if !f.HasPendingOffers {
		t.Errorf("HasPendingOffers should be true")
	}
}

func TestDecodeQuery_InvalidUUID(t *testing.T) {
	_, err := decodeFilters(t, "customer_org_id=not-a-uuid")
	if err == nil {
		t.Fatalf("invalid uuid must produce error")
	}
	if !strings.Contains(err.Error(), "customer_org_id") {
		t.Errorf("error should mention bad field, got: %v", err)
	}
}

func TestDecodeQuery_NegativeWeight_Rejected(t *testing.T) {
	_, err := decodeFilters(t, "min_weight=-10")
	if err == nil {
		t.Fatalf("negative weight must fail validation")
	}
	if !strings.Contains(err.Error(), "MinWeight") && !strings.Contains(err.Error(), "min_weight") {
		t.Errorf("error should mention MinWeight, got: %v", err)
	}
}

func TestDecodeQuery_EmptyQuery(t *testing.T) {
	f, err := decodeFilters(t, "")
	if err != nil {
		t.Fatalf("empty query must not error, got: %v", err)
	}
	if f.CustomerOrgID != nil {
		t.Errorf("expected nil pointers, got %v", f.CustomerOrgID)
	}
	if f.Statuses != nil {
		t.Errorf("expected nil Statuses, got %v", f.Statuses)
	}
}

func TestDecodeQuery_UnknownKeysIgnored(t *testing.T) {
	_, err := decodeFilters(t, "totally_random_param=42")
	if err != nil {
		t.Errorf("unknown keys should be ignored, got: %v", err)
	}
}

func TestDecodeQuery_LongOrgName_Rejected(t *testing.T) {
	longName := strings.Repeat("a", 101)
	_, err := decodeFilters(t, "org_name="+longName)
	if err == nil {
		t.Fatalf("org_name > 100 must fail validation")
	}
}

func TestDecodeQuery_InvalidIntInList_Rejected(t *testing.T) {
	_, err := decodeFilters(t, "route_city_ids=1,abc,3")
	if err == nil {
		t.Fatalf("non-integer in route_city_ids must fail")
	}
}

func TestDecodeQuery_RoleAwareRouteIDs(t *testing.T) {
	f, err := decodeFilters(t,
		"origin_city_ids=10,20"+
			"&origin_country_ids=1"+
			"&destination_city_ids=30"+
			"&destination_country_ids=2,3",
	)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(f.OriginCityIDs) != 2 || f.OriginCityIDs[0] != 10 || f.OriginCityIDs[1] != 20 {
		t.Errorf("OriginCityIDs = %v", f.OriginCityIDs)
	}
	if len(f.OriginCountryIDs) != 1 || f.OriginCountryIDs[0] != 1 {
		t.Errorf("OriginCountryIDs = %v", f.OriginCountryIDs)
	}
	if len(f.DestinationCityIDs) != 1 || f.DestinationCityIDs[0] != 30 {
		t.Errorf("DestinationCityIDs = %v", f.DestinationCityIDs)
	}
	if len(f.DestinationCountryIDs) != 2 || f.DestinationCountryIDs[0] != 2 || f.DestinationCountryIDs[1] != 3 {
		t.Errorf("DestinationCountryIDs = %v", f.DestinationCountryIDs)
	}
}

func TestDecodeQuery_InvalidIntInOriginCityIDs_Rejected(t *testing.T) {
	_, err := decodeFilters(t, "origin_city_ids=1,xxx")
	if err == nil {
		t.Fatalf("non-integer in origin_city_ids must fail")
	}
}

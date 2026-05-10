package projections

import (
	"strings"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

// helper: build SQL from a base SELECT with given options applied,
// returns SQL string and args.
func buildFreightSQL(t *testing.T, opts ...FilterOption) (string, []any) {
	t.Helper()
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	b := psql.Select("id").From("freight_requests_lookup")
	for _, o := range opts {
		b = o(b)
	}
	sql, args, err := b.ToSql()
	if err != nil {
		t.Fatalf("build sql: %v", err)
	}
	return sql, args
}

func TestWithCursor_NotPublished(t *testing.T) {
	sql, args := buildFreightSQL(t, WithCursor(FreightRequestCursor{
		IsPublished:   false,
		RequestNumber: 100,
	}))

	// На не-published курсоре отдаём только не-published с меньшим номером.
	if !strings.Contains(sql, "status <> $1") {
		t.Errorf("expected 'status <> $1' in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, "request_number < $2") {
		t.Errorf("expected 'request_number < $2' in SQL, got: %s", sql)
	}
	if len(args) != 2 || args[0] != "published" || args[1] != int64(100) {
		t.Errorf("expected args [published, 100], got %v", args)
	}
}

func TestWithCursor_Published(t *testing.T) {
	sql, args := buildFreightSQL(t, WithCursor(FreightRequestCursor{
		IsPublished:   true,
		RequestNumber: 50,
	}))

	// На published курсоре отдаём:
	//   1. published с меньшим номером, ИЛИ
	//   2. любые не-published.
	if !strings.Contains(sql, "OR") {
		t.Errorf("expected OR in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, "status = $1") {
		t.Errorf("expected 'status = $1' in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, "request_number < $2") {
		t.Errorf("expected 'request_number < $2' in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, "status <> $3") {
		t.Errorf("expected 'status <> $3' in SQL, got: %s", sql)
	}
	if len(args) != 3 || args[0] != "published" || args[1] != int64(50) || args[2] != "published" {
		t.Errorf("expected args [published, 50, published], got %v", args)
	}
}

func TestWithStatuses_Empty(t *testing.T) {
	sql, args := buildFreightSQL(t, WithStatuses(nil))
	if strings.Contains(sql, "WHERE") {
		t.Errorf("empty statuses must not add WHERE, got: %s", sql)
	}
	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}
}

func TestWithStatuses_Single(t *testing.T) {
	sql, args := buildFreightSQL(t, WithStatuses([]string{"published"}))
	if !strings.Contains(sql, "status IN ($1)") {
		t.Errorf("expected 'status IN ($1)', got: %s", sql)
	}
	if len(args) != 1 || args[0] != "published" {
		t.Errorf("expected args [published], got %v", args)
	}
}

func TestWithStatuses_Multiple(t *testing.T) {
	sql, args := buildFreightSQL(t, WithStatuses([]string{"published", "confirmed"}))
	if !strings.Contains(sql, "status IN ($1,$2)") {
		t.Errorf("expected 'status IN ($1,$2)', got: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %v", args)
	}
}

func assertUUIDArg(t *testing.T, args []any, want uuid.UUID) {
	t.Helper()
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d: %v", len(args), args)
	}
	got, ok := args[0].(string)
	if !ok {
		t.Fatalf("expected arg of type string, got %T", args[0])
	}
	if got != want.String() {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestWithCustomerOrgID(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	sql, args := buildFreightSQL(t, WithCustomerOrgID(id))
	if !strings.Contains(sql, "customer_org_id = $1") {
		t.Errorf("expected 'customer_org_id = $1', got: %s", sql)
	}
	assertUUIDArg(t, args, id)
}

func TestWithFreightCarrierOrgID(t *testing.T) {
	id := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	sql, args := buildFreightSQL(t, WithFreightCarrierOrgID(id))
	if !strings.Contains(sql, "carrier_org_id = $1") {
		t.Errorf("expected 'carrier_org_id = $1', got: %s", sql)
	}
	assertUUIDArg(t, args, id)
}

func TestWithCustomerMemberID(t *testing.T) {
	id := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	sql, args := buildFreightSQL(t, WithCustomerMemberID(id))
	if !strings.Contains(sql, "customer_member_id = $1") {
		t.Errorf("expected 'customer_member_id = $1', got: %s", sql)
	}
	assertUUIDArg(t, args, id)
}

func TestWithMinMaxPrice(t *testing.T) {
	sql, args := buildFreightSQL(t, WithMinPrice(1000), WithMaxPrice(5000))
	if !strings.Contains(sql, "price_amount >= $1") {
		t.Errorf("expected 'price_amount >= $1', got: %s", sql)
	}
	if !strings.Contains(sql, "price_amount <= $2") {
		t.Errorf("expected 'price_amount <= $2', got: %s", sql)
	}
	if len(args) != 2 || args[0] != int64(1000) || args[1] != int64(5000) {
		t.Errorf("expected args [1000 5000], got %v", args)
	}
}

func TestWithRouteCities_Empty(t *testing.T) {
	sql, args := buildFreightSQL(t, WithRouteCities(nil))
	if strings.Contains(sql, "WHERE") {
		t.Errorf("empty route cities must not add WHERE, got: %s", sql)
	}
	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}
}

func TestWithRouteCities_NonEmpty(t *testing.T) {
	sql, args := buildFreightSQL(t, WithRouteCities([]int{1, 5, 42}))
	if !strings.Contains(sql, "route_city_ids @> $1::integer[]") {
		t.Errorf("expected 'route_city_ids @> $1::integer[]', got: %s", sql)
	}
	if len(args) != 1 || args[0] != "{1,5,42}" {
		t.Errorf("expected args [\"{1,5,42}\"], got %v", args)
	}
}

func TestWithRouteCountries_NonEmpty(t *testing.T) {
	sql, args := buildFreightSQL(t, WithRouteCountries([]int{7, 8}))
	if !strings.Contains(sql, "route_country_ids @> $1::integer[]") {
		t.Errorf("expected 'route_country_ids @> $1::integer[]', got: %s", sql)
	}
	if len(args) != 1 || args[0] != "{7,8}" {
		t.Errorf("expected args [\"{7,8}\"], got %v", args)
	}
}

func TestWithHasPendingOffers(t *testing.T) {
	sql, _ := buildFreightSQL(t, WithHasPendingOffers())
	if !strings.Contains(sql, "EXISTS") {
		t.Errorf("expected EXISTS subquery, got: %s", sql)
	}
	if !strings.Contains(sql, "offers_lookup") {
		t.Errorf("expected offers_lookup reference, got: %s", sql)
	}
	if !strings.Contains(sql, "o.status = 'pending'") {
		t.Errorf("expected pending status filter, got: %s", sql)
	}
}

func TestCombinedFilters_AreANDed(t *testing.T) {
	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	sql, _ := buildFreightSQL(t,
		WithCustomerOrgID(id),
		WithStatuses([]string{"confirmed"}),
		WithCursor(FreightRequestCursor{IsPublished: false, RequestNumber: 50}),
	)
	for _, want := range []string{
		"customer_org_id = $1",
		"status IN ($2)",
		"request_number < $4",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("expected %q in combined SQL, got: %s", want, sql)
		}
	}
	if !strings.Contains(sql, "AND") {
		t.Errorf("expected AND between conditions, got: %s", sql)
	}
}

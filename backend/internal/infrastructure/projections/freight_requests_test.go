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

func TestWithCursor(t *testing.T) {
	cases := []struct {
		name   string
		cursor FreightRequestCursor
		wantArg int64
	}{
		{"normal", FreightRequestCursor{RequestNumber: 100}, 100},
		{"zero", FreightRequestCursor{RequestNumber: 0}, 0},
		{"large", FreightRequestCursor{RequestNumber: 9_999_999_999}, 9_999_999_999},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sql, args := buildFreightSQL(t, WithCursor(tc.cursor))
			if !strings.Contains(sql, "request_number < $1") {
				t.Errorf("expected 'request_number < $1' in SQL, got: %s", sql)
			}
			if strings.Contains(sql, "status") {
				t.Errorf("cursor must not reference status anymore, got: %s", sql)
			}
			if len(args) != 1 || args[0] != tc.wantArg {
				t.Errorf("expected args [%d], got %v", tc.wantArg, args)
			}
		})
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
	// squirrel.Eq всегда даёт IN-форму при работе со слайсом, даже из 1 элемента.
	if !strings.Contains(sql, "status IN ($1)") {
		t.Errorf("expected 'status IN ($1)', got: %s", sql)
	}
	if len(args) != 1 || args[0] != "published" {
		t.Errorf("expected args [published], got %v", args)
	}
}

func TestWithStatuses_Multiple(t *testing.T) {
	sql, args := buildFreightSQL(t, WithStatuses([]string{"published", "confirmed"}))
	// squirrel.Eq with slice → IN clause
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
	// squirrel.Eq вызывает driver.Valuer на uuid.UUID → в args приходит строка.
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

func TestCombinedFilters_AreANDed(t *testing.T) {
	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	sql, _ := buildFreightSQL(t,
		WithCustomerOrgID(id),
		WithStatuses([]string{"confirmed"}),
		WithCursor(FreightRequestCursor{RequestNumber: 50}),
	)
	// Все три условия должны присутствовать — AND-семантика.
	for _, want := range []string{
		"customer_org_id = $1",
		"status IN ($2)",
		"request_number < $3",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("expected %q in combined SQL, got: %s", want, sql)
		}
	}
	if !strings.Contains(sql, "AND") {
		t.Errorf("expected AND between conditions, got: %s", sql)
	}
}

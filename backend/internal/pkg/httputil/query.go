package httputil

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/schema"
)

// queryDecoder — общий schema-декодер для query-параметров.
// Зарегистрированы кастомные конвертеры:
//   - []string / []int — поддержка CSV (`?statuses=published,confirmed`)
//   - uuid.UUID — парсинг через uuid.Parse
//
// Тёго `schema:"-"` исключает поле из декодинга.
var queryDecoder = newQueryDecoder()

func newQueryDecoder() *schema.Decoder {
	d := schema.NewDecoder()
	d.IgnoreUnknownKeys(true)
	d.ZeroEmpty(false) // пустая строка не сбрасывает значение в zero

	d.RegisterConverter([]string{}, csvStringConverter)
	d.RegisterConverter([]int{}, csvIntConverter)
	d.RegisterConverter(uuid.UUID{}, uuidConverter)

	return d
}

func csvStringConverter(s string) reflect.Value {
	if s == "" {
		return reflect.ValueOf([]string(nil))
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return reflect.ValueOf(out)
}

func csvIntConverter(s string) reflect.Value {
	if s == "" {
		return reflect.ValueOf([]int(nil))
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			// Невалидный элемент CSV → декодер вернёт ошибку через schema.MultiError.
			return reflect.Value{}
		}
		out = append(out, n)
	}
	return reflect.ValueOf(out)
}

func uuidConverter(s string) reflect.Value {
	if s == "" {
		// schema передаёт пустую строку только если ZeroEmpty(true) — у нас false,
		// но на всякий случай — оставляем zero-value UUID без ошибки.
		return reflect.ValueOf(uuid.UUID{})
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return reflect.Value{}
	}
	return reflect.ValueOf(u)
}

// DecodeQuery декодирует query-параметры запроса в указатель на структуру,
// затем валидирует её через общий validator-инстанс.
//
// Тег `schema:"<name>"` определяет имя query-параметра.
// Тег `validate:"<rules>"` — правила go-playground/validator.
//
// На любую ошибку (декодинг или валидация) возвращает error, пригодный
// для прямой передачи клиенту с кодом 400.
func DecodeQuery(r *http.Request, dst any) error {
	if err := queryDecoder.Decode(dst, r.URL.Query()); err != nil {
		return formatSchemaError(err)
	}
	return Validate(dst)
}

func formatSchemaError(err error) error {
	if multiErr, ok := errors.AsType[schema.MultiError](err); ok {
		fields := make([]string, 0, len(multiErr))
		for k := range multiErr {
			fields = append(fields, k)
		}
		return fmt.Errorf("invalid query parameter(s): %s", strings.Join(fields, ", "))
	}
	return fmt.Errorf("invalid query: %w", err)
}

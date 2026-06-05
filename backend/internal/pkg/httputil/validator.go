package httputil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate — общий singleton-инстанс validator/v10.
// Используется для query DTO (см. DecodeQuery) и впоследствии для тел запросов.
var validate = validator.New(validator.WithRequiredStructEnabled())

// Validate валидирует структуру через общий validator-инстанс
// и возвращает компактное человекочитаемое сообщение об ошибке,
// пригодное для отправки клиенту с кодом 400.
func Validate(v any) error {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}
	if vErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		return FormatValidationErrors(vErrs)
	}
	return err
}

// FormatValidationErrors сворачивает validator.ValidationErrors в одну строку
// вида: "field1 (tag1), field2 (tag2)".
func FormatValidationErrors(vErrs validator.ValidationErrors) error {
	parts := make([]string, 0, len(vErrs))
	for _, fe := range vErrs {
		parts = append(parts, fmt.Sprintf("%s (%s)", fe.Field(), fe.Tag()))
	}
	return fmt.Errorf("validation failed: %s", strings.Join(parts, ", "))
}

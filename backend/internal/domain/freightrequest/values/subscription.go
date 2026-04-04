package values

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MaxSubscriptionsPerMember максимальное количество подписок на пользователя
const MaxSubscriptionsPerMember = 10

// RoutePointCriteria критерий точки маршрута для subsequence matching
type RoutePointCriteria struct {
	CountryID int  `json:"country_id"`        // ID страны (обязательно)
	CityID    *int `json:"city_id,omitempty"` // ID города (nil = любой город в стране)
	Order     int  `json:"order"`             // Порядок в последовательности (1, 2, 3...)
}

// SubscriptionCriteria критерии фильтрации заявок
type SubscriptionCriteria struct {
	Name string `json:"name"` // Название трафарета для UI

	// Числовые диапазоны (nil = без ограничения)
	MinWeight *float64 `json:"min_weight,omitempty"` // Мин. вес в тоннах
	MaxWeight *float64 `json:"max_weight,omitempty"` // Макс. вес в тоннах
	MinPrice  *int64   `json:"min_price,omitempty"`  // Мин. цена в минорных единицах
	MaxPrice  *int64   `json:"max_price,omitempty"`  // Макс. цена в минорных единицах
	MinVolume *float64 `json:"min_volume,omitempty"` // Мин. объём м³
	MaxVolume *float64 `json:"max_volume,omitempty"` // Макс. объём м³

	// ENUM массивы (nil/пустой = все подходят)
	VehicleTypes    []VehicleType    `json:"vehicle_types,omitempty"`
	VehicleSubTypes []VehicleSubType `json:"vehicle_subtypes,omitempty"`
	PaymentMethods  []PaymentMethod  `json:"payment_methods,omitempty"`
	PaymentTerms    []PaymentTerms   `json:"payment_terms,omitempty"`
	VatTypes        []VatType        `json:"vat_types,omitempty"`

	// Маршрут для subsequence matching (пустой = любой маршрут)
	RoutePoints []RoutePointCriteria `json:"route_points,omitempty"`

	IsActive bool `json:"is_active"`
}

// Validate проверяет корректность критериев
func (c SubscriptionCriteria) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(c.Name) > 100 {
		return fmt.Errorf("name is too long (max 100 characters)")
	}

	// Проверяем диапазоны
	if c.MinWeight != nil && c.MaxWeight != nil && *c.MinWeight > *c.MaxWeight {
		return fmt.Errorf("min_weight cannot be greater than max_weight")
	}
	if c.MinPrice != nil && c.MaxPrice != nil && *c.MinPrice > *c.MaxPrice {
		return fmt.Errorf("min_price cannot be greater than max_price")
	}
	if c.MinVolume != nil && c.MaxVolume != nil && *c.MinVolume > *c.MaxVolume {
		return fmt.Errorf("min_volume cannot be greater than max_volume")
	}

	// Проверяем отрицательные значения
	if c.MinWeight != nil && *c.MinWeight < 0 {
		return fmt.Errorf("min_weight cannot be negative")
	}
	if c.MaxWeight != nil && *c.MaxWeight < 0 {
		return fmt.Errorf("max_weight cannot be negative")
	}
	if c.MinPrice != nil && *c.MinPrice < 0 {
		return fmt.Errorf("min_price cannot be negative")
	}
	if c.MaxPrice != nil && *c.MaxPrice < 0 {
		return fmt.Errorf("max_price cannot be negative")
	}
	if c.MinVolume != nil && *c.MinVolume < 0 {
		return fmt.Errorf("min_volume cannot be negative")
	}
	if c.MaxVolume != nil && *c.MaxVolume < 0 {
		return fmt.Errorf("max_volume cannot be negative")
	}

	// Проверяем route points
	for i, rp := range c.RoutePoints {
		if rp.CountryID <= 0 {
			return fmt.Errorf("route_point[%d]: country_id is required", i)
		}
		if rp.Order <= 0 {
			return fmt.Errorf("route_point[%d]: order must be positive", i)
		}
	}

	// Проверяем уникальность order в route points
	orderSeen := make(map[int]bool)
	for i, rp := range c.RoutePoints {
		if orderSeen[rp.Order] {
			return fmt.Errorf("route_point[%d]: duplicate order %d", i, rp.Order)
		}
		orderSeen[rp.Order] = true
	}

	return nil
}

// Subscription представляет подписку пользователя на заявки
type Subscription struct {
	ID        uuid.UUID            `json:"id"`
	MemberID  uuid.UUID            `json:"member_id"`
	Criteria  SubscriptionCriteria `json:"criteria"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

// FreightRequestMatchData данные заявки для matching
type FreightRequestMatchData struct {
	CustomerMemberID uuid.UUID
	Route            Route
	Cargo            CargoInfo
	Payment          Payment
	VehicleReqs      VehicleRequirements
}

// MatchedSubscription результат matching
type MatchedSubscription struct {
	SubscriptionID   uuid.UUID `json:"subscription_id"`
	SubscriptionName string    `json:"subscription_name"`
	MemberID         uuid.UUID `json:"member_id"`
	OrganizationID   uuid.UUID `json:"organization_id"`
}

// IsSubsequence проверяет является ли pattern подпоследовательностью route
// Алгоритм: проходим по route и ищем точки из pattern в том же порядке
func IsSubsequence(pattern []RoutePointCriteria, route Route) bool {
	if len(pattern) == 0 {
		return true // Пустой паттерн = любой маршрут подходит
	}

	patternIdx := 0
	for _, routePoint := range route.Points {
		if matchesRoutePoint(pattern[patternIdx], routePoint) {
			patternIdx++
			if patternIdx == len(pattern) {
				return true // Все точки найдены
			}
		}
	}
	return false // Не все точки найдены
}

// matchesRoutePoint проверяет соответствует ли точка маршрута критерию
func matchesRoutePoint(criteria RoutePointCriteria, actual RoutePoint) bool {
	// Точка маршрута должна иметь country_id
	if actual.CountryID == nil {
		return false
	}

	// Страна должна совпадать
	if criteria.CountryID != *actual.CountryID {
		return false
	}

	// city_id nil в критерии = любой город в стране подходит
	if criteria.CityID != nil {
		if actual.CityID == nil || *criteria.CityID != *actual.CityID {
			return false
		}
	}

	return true
}

// MatchesCriteria проверяет соответствует ли заявка критериям подписки
func MatchesCriteria(criteria SubscriptionCriteria, data FreightRequestMatchData) bool {
	// Числовые диапазоны
	if criteria.MinWeight != nil && data.Cargo.Weight < *criteria.MinWeight {
		return false
	}
	if criteria.MaxWeight != nil && data.Cargo.Weight > *criteria.MaxWeight {
		return false
	}

	if data.Payment.Price != nil {
		if criteria.MinPrice != nil && data.Payment.Price.Amount < *criteria.MinPrice {
			return false
		}
		if criteria.MaxPrice != nil && data.Payment.Price.Amount > *criteria.MaxPrice {
			return false
		}
	}

	if criteria.MinVolume != nil && data.Cargo.Volume < *criteria.MinVolume {
		return false
	}
	if criteria.MaxVolume != nil && data.Cargo.Volume > *criteria.MaxVolume {
		return false
	}

	// ENUM массивы
	if !hasVehicleTypeMatch(criteria.VehicleTypes, criteria.VehicleSubTypes, data.VehicleReqs.VehicleType, data.VehicleReqs.VehicleSubType) {
		return false
	}
	if len(criteria.PaymentMethods) > 0 && !containsPaymentMethod(criteria.PaymentMethods, data.Payment.Method) {
		return false
	}
	if len(criteria.PaymentTerms) > 0 && !containsPaymentTerms(criteria.PaymentTerms, data.Payment.Terms) {
		return false
	}
	if len(criteria.VatTypes) > 0 && !containsVatType(criteria.VatTypes, data.Payment.VatType) {
		return false
	}

	// Subsequence matching для маршрута
	if !IsSubsequence(criteria.RoutePoints, data.Route) {
		return false
	}

	return true
}

// Вспомогательные функции для проверки ENUM

// hasVehicleTypeMatch checks if freight request vehicle matches subscription criteria
func hasVehicleTypeMatch(criteriaTypes []VehicleType, criteriaSubTypes []VehicleSubType, actualType VehicleType, actualSubType VehicleSubType) bool {
	// Если нет фильтров - все подходит
	if len(criteriaTypes) == 0 && len(criteriaSubTypes) == 0 {
		return true
	}

	// Проверка по типу
	typeMatch := len(criteriaTypes) == 0
	for _, ct := range criteriaTypes {
		if ct == actualType {
			typeMatch = true
			break
		}
	}

	// Проверка по подтипу
	subTypeMatch := len(criteriaSubTypes) == 0
	for _, cst := range criteriaSubTypes {
		if cst == actualSubType {
			subTypeMatch = true
			break
		}
	}

	return typeMatch && subTypeMatch
}

func containsPaymentMethod(slice []PaymentMethod, item PaymentMethod) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func containsPaymentTerms(slice []PaymentTerms, item PaymentTerms) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func containsVatType(slice []VatType, item VatType) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

package helpers

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/google/uuid"
)

// Note: Go 1.22+ math/rand/v2 auto-seeds, no need for manual seeding

// RandomEmail generates a random email address.
func RandomEmail() string {
	return fmt.Sprintf("test-%s@test.local", uuid.New().String()[:8])
}

// RandomEmailWithDomain generates a random email with a specific domain.
func RandomEmailWithDomain(domain string) string {
	return fmt.Sprintf("test-%s@%s", uuid.New().String()[:8], domain)
}

// RandomPhone generates a random Russian phone number.
func RandomPhone() string {
	return fmt.Sprintf("+7900%07d", rand.IntN(10000000))
}

// RandomINN generates a random 10-digit INN.
func RandomINN() string {
	return fmt.Sprintf("%010d", rand.Int64N(10000000000))
}

// RandomName generates a random name.
func RandomName() string {
	return fmt.Sprintf("Test User %s", uuid.New().String()[:8])
}

// RandomOrgName generates a random organization name.
func RandomOrgName() string {
	prefixes := []string{"ООО", "АО", "ИП"}
	names := []string{"Альфа", "Бета", "Гамма", "Дельта", "Омега", "Логистик", "Транс", "Карго"}
	suffixes := []string{"Групп", "Сервис", "Трейд", "Экспресс", ""}

	prefix := prefixes[rand.IntN(len(prefixes))]
	name := names[rand.IntN(len(names))]
	suffix := suffixes[rand.IntN(len(suffixes))]

	result := fmt.Sprintf("%s %s%s", prefix, name, suffix)
	return strings.TrimSpace(result) + " " + uuid.New().String()[:4]
}

// RandomAddress generates a random address.
func RandomAddress() string {
	cities := []string{"Москва", "Санкт-Петербург", "Новосибирск", "Казань", "Екатеринбург"}
	streets := []string{"Ленина", "Пушкина", "Гагарина", "Мира", "Советская"}

	city := cities[rand.IntN(len(cities))]
	street := streets[rand.IntN(len(streets))]
	building := rand.IntN(200) + 1

	return fmt.Sprintf("г. %s, ул. %s, д. %d", city, street, building)
}

// RandomPassword generates a random password.
func RandomPassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.IntN(len(chars))]
	}
	return string(result)
}

// RandomString generates a random alphanumeric string.
func RandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.IntN(len(chars))]
	}
	return string(result)
}

// RandomInt returns a random integer in the range [min, max].
func RandomInt(min, max int) int {
	return min + rand.IntN(max-min+1)
}

// RandomFloat returns a random float in the range [min, max].
func RandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// RandomPrice generates a random price (for freight).
func RandomPrice() float64 {
	return float64(RandomInt(10000, 500000))
}

// RandomWeight generates a random cargo weight in kg.
func RandomWeight() float64 {
	return float64(RandomInt(100, 20000))
}

// RandomVolume generates a random cargo volume in m³.
func RandomVolume() float64 {
	return float64(RandomInt(5, 100))
}

// RandomVehicleType returns a random vehicle type.
func RandomVehicleType() string {
	types := []string{"tent", "ref", "isotherm", "tank", "container", "car_carrier", "grain_carrier", "flatbed"}
	return types[rand.IntN(len(types))]
}

// RandomCountryCode returns a random supported country code.
func RandomCountryCode() string {
	codes := []string{"RU", "KZ", "BY", "AM", "KG", "UZ"}
	return codes[rand.IntN(len(codes))]
}

// RandomCityName returns a random city name.
func RandomCityName() string {
	cities := []string{"Москва", "Санкт-Петербург", "Новосибирск", "Алматы", "Минск", "Ереван", "Бишкек", "Ташкент"}
	return cities[rand.IntN(len(cities))]
}

// RandomComment generates a random comment.
func RandomComment() string {
	comments := []string{
		"Срочная доставка",
		"Груз хрупкий, обращаться осторожно",
		"Требуется температурный режим",
		"Возможна частичная загрузка",
		"Документы в электронном виде",
		"",
	}
	return comments[rand.IntN(len(comments))]
}

// UniqueID returns a short unique identifier.
func UniqueID() string {
	return uuid.New().String()[:8]
}

// StringPtr returns a pointer to a string.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int.
func IntPtr(i int) *int {
	return &i
}

// Float64Ptr returns a pointer to a float64.
func Float64Ptr(f float64) *float64 {
	return &f
}

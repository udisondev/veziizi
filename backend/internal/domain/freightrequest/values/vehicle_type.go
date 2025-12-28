package values

//go:generate go-enum --marshal --sql --names --ptr

// VehicleType represents the main vehicle category (road transport only for now)
// ENUM(van, flatbed, tanker, dump_truck, specialized_truck, light_truck, medium_truck, heavy_truck)
type VehicleType string

// VehicleTypeLabels returns human-readable labels for vehicle types
var VehicleTypeLabels = map[VehicleType]string{
	VehicleTypeVan:              "Фургон",
	VehicleTypeFlatbed:          "Платформа",
	VehicleTypeTanker:           "Цистерна",
	VehicleTypeDumpTruck:        "Самосвал",
	VehicleTypeSpecializedTruck: "Спецтранспорт",
	VehicleTypeLightTruck:       "Лёгкий грузовик",
	VehicleTypeMediumTruck:      "Средний грузовик",
	VehicleTypeHeavyTruck:       "Тяжёлый грузовик",
}

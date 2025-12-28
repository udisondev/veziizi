package values

//go:generate go-enum --marshal --sql --names --ptr

// VehicleSubType represents the specific vehicle subtype
// ENUM(dry_van, insulated, refrigerator, curtain_side, box_truck, furniture_van, standard_flatbed, drop_deck, lowboy, extendable, conestoga, liquid_tanker, gas_tanker, chemical_tanker, food_tanker, bitumen_tanker, rear_dump, side_dump, bottom_dump, car_carrier, timber_truck, grain_truck, livestock_carrier, concrete_mixer, container_chassis, tow_truck, crane_truck, city_van, pickup, minivan_cargo, medium_box, medium_flatbed, semi_trailer, road_train, mega_trailer)
type VehicleSubType string

// VehicleTypeSubTypes mapping of valid subtypes for each vehicle type
var VehicleTypeSubTypes = map[VehicleType][]VehicleSubType{
	VehicleTypeVan: {
		VehicleSubTypeDryVan,
		VehicleSubTypeInsulated,
		VehicleSubTypeRefrigerator,
		VehicleSubTypeCurtainSide,
		VehicleSubTypeBoxTruck,
		VehicleSubTypeFurnitureVan,
	},
	VehicleTypeFlatbed: {
		VehicleSubTypeStandardFlatbed,
		VehicleSubTypeDropDeck,
		VehicleSubTypeLowboy,
		VehicleSubTypeExtendable,
		VehicleSubTypeConestoga,
	},
	VehicleTypeTanker: {
		VehicleSubTypeLiquidTanker,
		VehicleSubTypeGasTanker,
		VehicleSubTypeChemicalTanker,
		VehicleSubTypeFoodTanker,
		VehicleSubTypeBitumenTanker,
	},
	VehicleTypeDumpTruck: {
		VehicleSubTypeRearDump,
		VehicleSubTypeSideDump,
		VehicleSubTypeBottomDump,
	},
	VehicleTypeSpecializedTruck: {
		VehicleSubTypeCarCarrier,
		VehicleSubTypeTimberTruck,
		VehicleSubTypeGrainTruck,
		VehicleSubTypeLivestockCarrier,
		VehicleSubTypeConcreteMixer,
		VehicleSubTypeContainerChassis,
		VehicleSubTypeTowTruck,
		VehicleSubTypeCraneTruck,
	},
	VehicleTypeLightTruck: {
		VehicleSubTypeCityVan,
		VehicleSubTypePickup,
		VehicleSubTypeMinivanCargo,
	},
	VehicleTypeMediumTruck: {
		VehicleSubTypeMediumBox,
		VehicleSubTypeMediumFlatbed,
	},
	VehicleTypeHeavyTruck: {
		VehicleSubTypeSemiTrailer,
		VehicleSubTypeRoadTrain,
		VehicleSubTypeMegaTrailer,
	},
}

// VehicleSubTypeLabels returns human-readable labels for vehicle subtypes
var VehicleSubTypeLabels = map[VehicleSubType]string{
	// van
	VehicleSubTypeDryVan:       "Сухой фургон",
	VehicleSubTypeInsulated:    "Изотермический",
	VehicleSubTypeRefrigerator: "Рефрижератор",
	VehicleSubTypeCurtainSide:  "Шторный",
	VehicleSubTypeBoxTruck:     "Будка",
	VehicleSubTypeFurnitureVan: "Мебельный фургон",
	// flatbed
	VehicleSubTypeStandardFlatbed: "Стандартная платформа",
	VehicleSubTypeDropDeck:        "Низкорамник",
	VehicleSubTypeLowboy:          "Трал",
	VehicleSubTypeExtendable:      "Раздвижная платформа",
	VehicleSubTypeConestoga:       "Конестога",
	// tanker
	VehicleSubTypeLiquidTanker:   "Жидкостная цистерна",
	VehicleSubTypeGasTanker:      "Газовая цистерна",
	VehicleSubTypeChemicalTanker: "Химическая цистерна",
	VehicleSubTypeFoodTanker:     "Пищевая цистерна",
	VehicleSubTypeBitumenTanker:  "Битумовоз",
	// dump_truck
	VehicleSubTypeRearDump:   "Задняя разгрузка",
	VehicleSubTypeSideDump:   "Боковая разгрузка",
	VehicleSubTypeBottomDump: "Нижняя разгрузка",
	// specialized_truck
	VehicleSubTypeCarCarrier:       "Автовоз",
	VehicleSubTypeTimberTruck:      "Лесовоз",
	VehicleSubTypeGrainTruck:       "Зерновоз",
	VehicleSubTypeLivestockCarrier: "Скотовоз",
	VehicleSubTypeConcreteMixer:    "Бетоносмеситель",
	VehicleSubTypeContainerChassis: "Контейнеровоз",
	VehicleSubTypeTowTruck:         "Эвакуатор",
	VehicleSubTypeCraneTruck:       "Кран-манипулятор",
	// light_truck
	VehicleSubTypeCityVan:     "Городской фургон",
	VehicleSubTypePickup:      "Пикап",
	VehicleSubTypeMinivanCargo: "Грузовой минивэн",
	// medium_truck
	VehicleSubTypeMediumBox:     "Средний фургон",
	VehicleSubTypeMediumFlatbed: "Средняя платформа",
	// heavy_truck
	VehicleSubTypeSemiTrailer: "Полуприцеп",
	VehicleSubTypeRoadTrain:   "Автопоезд",
	VehicleSubTypeMegaTrailer: "Мега-трейлер",
}

// IsValidSubtypeForType checks if subtype is valid for the given vehicle type
func IsValidSubtypeForType(vt VehicleType, vs VehicleSubType) bool {
	subtypes, ok := VehicleTypeSubTypes[vt]
	if !ok {
		return false
	}
	for _, s := range subtypes {
		if s == vs {
			return true
		}
	}
	return false
}

// GetVehicleTypeForSubType returns the vehicle type for a given subtype
func GetVehicleTypeForSubType(vs VehicleSubType) (VehicleType, bool) {
	for vt, subtypes := range VehicleTypeSubTypes {
		for _, s := range subtypes {
			if s == vs {
				return vt, true
			}
		}
	}
	return "", false
}

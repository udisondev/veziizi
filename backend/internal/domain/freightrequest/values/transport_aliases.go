package values

import "github.com/udisondev/veziizi/backend/internal/domain/transport"

// Type aliases re-exported from the shared transport package.
// Vehicle-related value objects live in domain/transport so they can be
// reused by both FreightRequest (vehicle requirements) and Organization
// (vehicle entity in the fleet aggregate).
type (
	VehicleType    = transport.VehicleType
	VehicleSubType = transport.VehicleSubType
	LoadingType    = transport.LoadingType
	Temperature    = transport.Temperature
)

// Vehicle type constants
const (
	VehicleTypeVan              = transport.VehicleTypeVan
	VehicleTypeFlatbed          = transport.VehicleTypeFlatbed
	VehicleTypeTanker           = transport.VehicleTypeTanker
	VehicleTypeDumpTruck        = transport.VehicleTypeDumpTruck
	VehicleTypeSpecializedTruck = transport.VehicleTypeSpecializedTruck
	VehicleTypeLightTruck       = transport.VehicleTypeLightTruck
	VehicleTypeMediumTruck      = transport.VehicleTypeMediumTruck
	VehicleTypeHeavyTruck       = transport.VehicleTypeHeavyTruck
)

// Vehicle subtype constants
const (
	VehicleSubTypeDryVan           = transport.VehicleSubTypeDryVan
	VehicleSubTypeInsulated        = transport.VehicleSubTypeInsulated
	VehicleSubTypeRefrigerator     = transport.VehicleSubTypeRefrigerator
	VehicleSubTypeCurtainSide      = transport.VehicleSubTypeCurtainSide
	VehicleSubTypeBoxTruck         = transport.VehicleSubTypeBoxTruck
	VehicleSubTypeFurnitureVan     = transport.VehicleSubTypeFurnitureVan
	VehicleSubTypeStandardFlatbed  = transport.VehicleSubTypeStandardFlatbed
	VehicleSubTypeDropDeck         = transport.VehicleSubTypeDropDeck
	VehicleSubTypeLowboy           = transport.VehicleSubTypeLowboy
	VehicleSubTypeExtendable       = transport.VehicleSubTypeExtendable
	VehicleSubTypeConestoga        = transport.VehicleSubTypeConestoga
	VehicleSubTypeLiquidTanker     = transport.VehicleSubTypeLiquidTanker
	VehicleSubTypeGasTanker        = transport.VehicleSubTypeGasTanker
	VehicleSubTypeChemicalTanker   = transport.VehicleSubTypeChemicalTanker
	VehicleSubTypeFoodTanker       = transport.VehicleSubTypeFoodTanker
	VehicleSubTypeBitumenTanker    = transport.VehicleSubTypeBitumenTanker
	VehicleSubTypeRearDump         = transport.VehicleSubTypeRearDump
	VehicleSubTypeSideDump         = transport.VehicleSubTypeSideDump
	VehicleSubTypeBottomDump       = transport.VehicleSubTypeBottomDump
	VehicleSubTypeCarCarrier       = transport.VehicleSubTypeCarCarrier
	VehicleSubTypeTimberTruck      = transport.VehicleSubTypeTimberTruck
	VehicleSubTypeGrainTruck       = transport.VehicleSubTypeGrainTruck
	VehicleSubTypeLivestockCarrier = transport.VehicleSubTypeLivestockCarrier
	VehicleSubTypeConcreteMixer    = transport.VehicleSubTypeConcreteMixer
	VehicleSubTypeContainerChassis = transport.VehicleSubTypeContainerChassis
	VehicleSubTypeTowTruck         = transport.VehicleSubTypeTowTruck
	VehicleSubTypeCraneTruck       = transport.VehicleSubTypeCraneTruck
	VehicleSubTypeCityVan          = transport.VehicleSubTypeCityVan
	VehicleSubTypePickup           = transport.VehicleSubTypePickup
	VehicleSubTypeMinivanCargo     = transport.VehicleSubTypeMinivanCargo
	VehicleSubTypeMediumBox        = transport.VehicleSubTypeMediumBox
	VehicleSubTypeMediumFlatbed    = transport.VehicleSubTypeMediumFlatbed
	VehicleSubTypeSemiTrailer      = transport.VehicleSubTypeSemiTrailer
	VehicleSubTypeRoadTrain        = transport.VehicleSubTypeRoadTrain
	VehicleSubTypeMegaTrailer      = transport.VehicleSubTypeMegaTrailer
)

// Loading type constants
const (
	LoadingTypeRear       = transport.LoadingTypeRear
	LoadingTypeSide       = transport.LoadingTypeSide
	LoadingTypeTop        = transport.LoadingTypeTop
	LoadingTypeFullUntarp = transport.LoadingTypeFullUntarp
)

// Re-exported helper functions
var (
	ParseVehicleType         = transport.ParseVehicleType
	ParseVehicleSubType      = transport.ParseVehicleSubType
	ParseLoadingType         = transport.ParseLoadingType
	IsValidSubtypeForType    = transport.IsValidSubtypeForType
	GetVehicleTypeForSubType = transport.GetVehicleTypeForSubType
	VehicleTypeNames         = transport.VehicleTypeNames
	VehicleSubTypeNames      = transport.VehicleSubTypeNames
	LoadingTypeNames         = transport.LoadingTypeNames
)

// Re-exported maps
var (
	VehicleTypeLabels    = transport.VehicleTypeLabels
	VehicleSubTypeLabels = transport.VehicleSubTypeLabels
	VehicleTypeSubTypes  = transport.VehicleTypeSubTypes
)

package dto

import "time"

// CreateTransferManifest is the public write contract for 转运清单. Status is deliberately
// omitted so callers cannot bypass the service state machine.
type CreateTransferManifest struct {
	Code          string    `json:"code" binding:"required,min=2,max=64"`
	Name          string    `json:"name" binding:"required,min=2,max=160"`
	GeneratorCode string    `json:"generatorCode" binding:"required,min=2,max=64"`
	CarrierCode   string    `json:"carrierCode" binding:"required,min=2,max=64"`
	WasteCode     string    `json:"wasteCode" binding:"required,min=2,max=64"`
	QuantityKg    float64   `json:"quantityKg" binding:"required,gt=0"`
	Destination   string    `json:"destination" binding:"required,min=2,max=200"`
	Description   string    `json:"description" binding:"max=1000"`
	Facility      string    `json:"facility" binding:"required,max=120"`
	Owner         string    `json:"owner" binding:"required,max=120"`
	Category      string    `json:"category" binding:"required,max=80"`
	RiskLevel     string    `json:"riskLevel" binding:"required,oneof=low medium high critical"`
	MetricValue   float64   `json:"metricValue"`
	MetricUnit    string    `json:"metricUnit" binding:"max=24"`
	EffectiveAt   time.Time `json:"effectiveAt" binding:"required"`
	Evidence      string    `json:"evidence" binding:"max=2000"`
	RelatedCode   string    `json:"relatedCode" binding:"max=64"`
}

type UpdateTransferManifest struct {
	ExpectedVersion uint      `json:"expectedVersion" binding:"required"`
	Name            string    `json:"name" binding:"required,min=2,max=160"`
	GeneratorCode   string    `json:"generatorCode" binding:"required,min=2,max=64"`
	CarrierCode     string    `json:"carrierCode" binding:"required,min=2,max=64"`
	WasteCode       string    `json:"wasteCode" binding:"required,min=2,max=64"`
	QuantityKg      float64   `json:"quantityKg" binding:"required,gt=0"`
	Destination     string    `json:"destination" binding:"required,min=2,max=200"`
	Description     string    `json:"description" binding:"max=1000"`
	Facility        string    `json:"facility" binding:"required,max=120"`
	Owner           string    `json:"owner" binding:"required,max=120"`
	Category        string    `json:"category" binding:"required,max=80"`
	RiskLevel       string    `json:"riskLevel" binding:"required,oneof=low medium high critical"`
	MetricValue     float64   `json:"metricValue"`
	MetricUnit      string    `json:"metricUnit" binding:"max=24"`
	EffectiveAt     time.Time `json:"effectiveAt" binding:"required"`
	Evidence        string    `json:"evidence" binding:"max=2000"`
	RelatedCode     string    `json:"relatedCode" binding:"max=64"`
}

// RegisterManifestWeighing is the write contract for 运输称重登记. It covers both
// the dispatch-time registration (loading weight, plate, escort — usable from
// draft/submitted for legacy backfill) and the post-dispatch arrival weight
// (usable from in_transit). Registered readings are immutable, so every field
// is optional at the binding layer and enforced by the service per state.
type RegisterManifestWeighing struct {
	ExpectedVersion uint    `json:"expectedVersion" binding:"required"`
	LoadingKg       float64 `json:"loadingKg"`
	VehiclePlate    string  `json:"vehiclePlate" binding:"max=32"`
	EscortName      string  `json:"escortName" binding:"max=80"`
	ArrivalKg       float64 `json:"arrivalKg"`
	Reason          string  `json:"reason" binding:"required,min=3,max=500"`
}

// TransitionTransferManifest carries the state-machine request plus weighing
// data that must be captured atomically with 提交联单 (loading weight, plate,
// escort) and 签收 (deviation reason when readings differ by more than 3%).
type TransitionTransferManifest struct {
	Status                string  `json:"status" binding:"required,max=40"`
	ExpectedVersion       uint    `json:"expectedVersion" binding:"required"`
	Reason                string  `json:"reason" binding:"required,min=3,max=500"`
	LoadingKg             float64 `json:"loadingKg"`
	VehiclePlate          string  `json:"vehiclePlate" binding:"max=32"`
	EscortName            string  `json:"escortName" binding:"max=80"`
	WeightDeviationReason string  `json:"weightDeviationReason" binding:"max=500"`
}

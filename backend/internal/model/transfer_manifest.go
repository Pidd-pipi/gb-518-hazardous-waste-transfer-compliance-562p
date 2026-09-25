package model

import "time"

// WeightDeviationLimit is the allowed |到厂-装车| / 装车 threshold. Beyond it a
// signed manifest must carry a documented deviation reason.
const WeightDeviationLimit = 0.03

// TransferManifest models 转运清单 as an independently versioned aggregate. The fields
// cover ownership, operational context, evidence and measured risk so later
// changes naturally span persistence, service and UI layers.
//
// QuantityKg is the planned weight declared at draft time. LoadingKg / vehicle
// plate / escort are registered when the 联单 is submitted; ArrivalKg is
// registered after dispatch and before the receiving facility signs for it.
// The weight pointers stay nil for legacy records until the readings are
// backfilled (待补录).
type TransferManifest struct {
	BaseModel
	GeneratorCode string    `json:"generatorCode" gorm:"size:64;index;not null"`
	CarrierCode   string    `json:"carrierCode" gorm:"size:64;index;not null"`
	WasteCode     string    `json:"wasteCode" gorm:"size:64;index;not null"`
	QuantityKg    float64   `json:"quantityKg" gorm:"not null"`
	Destination   string    `json:"destination" gorm:"size:200;not null"`
	Facility      string    `json:"facility" gorm:"size:120;index"`
	Owner         string    `json:"owner" gorm:"size:120;index"`
	Category      string    `json:"category" gorm:"size:80;index"`
	RiskLevel     string    `json:"riskLevel" gorm:"size:32;index"`
	MetricValue   float64   `json:"metricValue"`
	MetricUnit    string    `json:"metricUnit" gorm:"size:24"`
	EffectiveAt   time.Time `json:"effectiveAt"`
	Evidence      string    `json:"evidence" gorm:"size:2000"`
	RelatedCode   string    `json:"relatedCode" gorm:"size:64;index"`

	LoadingKg             *float64 `json:"loadingKg" gorm:"index"`
	VehiclePlate          string   `json:"vehiclePlate" gorm:"size:32;index"`
	EscortName            string   `json:"escortName" gorm:"size:80"`
	ArrivalKg             *float64 `json:"arrivalKg" gorm:"index"`
	WeightDeviationReason string   `json:"weightDeviationReason" gorm:"size:500"`
}

func (item *TransferManifest) GetBase() *BaseModel { return &item.BaseModel }

func (item TransferManifest) TableName() string { return "transfer_manifests" }

var TransferManifestInitialStatus = "draft"

// HasLoadingWeighing reports whether the mandatory dispatch-time registration
// (actual loading weight, plate, escort) is present.
func (item TransferManifest) HasLoadingWeighing() bool {
	return item.LoadingKg != nil && *item.LoadingKg > 0 &&
		item.VehiclePlate != "" && item.EscortName != ""
}

// HasArrivalWeighing reports whether the receiving-factory weight is registered.
func (item TransferManifest) HasArrivalWeighing() bool {
	return item.ArrivalKg != nil && *item.ArrivalKg > 0
}

// WeightDeviationRatio returns |arrival - loading| / loading and whether both
// readings exist to compute it.
func (item TransferManifest) WeightDeviationRatio() (float64, bool) {
	if item.LoadingKg == nil || item.ArrivalKg == nil || *item.LoadingKg <= 0 {
		return 0, false
	}
	diff := *item.ArrivalKg - *item.LoadingKg
	if diff < 0 {
		diff = -diff
	}
	return diff / *item.LoadingKg, true
}

// WeightDeviationExceeded reports whether the arrival/loading gap is over the
// 3% tolerance.
func (item TransferManifest) WeightDeviationExceeded() bool {
	ratio, ok := item.WeightDeviationRatio()
	return ok && ratio > WeightDeviationLimit
}

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/constants"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/dto"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/model"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/repository"
)

type TransferManifestService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.TransferManifest], error)
	Get(context.Context, uint) (model.TransferManifest, error)
	Create(context.Context, dto.CreateTransferManifest, string, string) (model.TransferManifest, error)
	Update(context.Context, uint, dto.UpdateTransferManifest, string, string) (model.TransferManifest, error)
	Transition(context.Context, uint, dto.ManifestTransitionRequest, string, string) (model.TransferManifest, error)
	RegisterWeighing(context.Context, uint, dto.RegisterWeighingRequest, string, string) (model.TransferManifest, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type transferManifestService struct {
	repository repository.TransferManifestRepository
	generators repository.WasteGeneratorRepository
	carriers   repository.CarrierProfileRepository
}

func NewTransferManifestService(repo repository.TransferManifestRepository, generators repository.WasteGeneratorRepository, carriers repository.CarrierProfileRepository) TransferManifestService {
	return &transferManifestService{repository: repo, generators: generators, carriers: carriers}
}

func (s *transferManifestService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.TransferManifest], error) {
	return s.repository.List(ctx, query)
}

func (s *transferManifestService) Get(ctx context.Context, id uint) (model.TransferManifest, error) {
	return s.repository.Get(ctx, id)
}

func (s *transferManifestService) Create(ctx context.Context, input dto.CreateTransferManifest, actor, requestID string) (model.TransferManifest, error) {
	if err := validateTransferManifestBusinessFields(input.Code, input.Name, input.Facility, input.Owner, input.GeneratorCode, input.CarrierCode, input.WasteCode, input.Destination, input.Evidence, input.QuantityKg); err != nil {
		return model.TransferManifest{}, err
	}
	if _, err := s.generators.FindByCode(ctx, input.GeneratorCode); err != nil {
		return model.TransferManifest{}, fmt.Errorf("%w: generator %s does not exist", ErrInvalidInput, input.GeneratorCode)
	}
	if _, err := s.carriers.FindByCode(ctx, input.CarrierCode); err != nil {
		return model.TransferManifest{}, fmt.Errorf("%w: carrier %s does not exist", ErrInvalidInput, input.CarrierCode)
	}
	item := model.TransferManifest{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.TransferManifestInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		GeneratorCode: strings.ToUpper(strings.TrimSpace(input.GeneratorCode)), CarrierCode: strings.ToUpper(strings.TrimSpace(input.CarrierCode)),
		WasteCode: strings.ToUpper(strings.TrimSpace(input.WasteCode)), QuantityKg: input.QuantityKg, Destination: strings.TrimSpace(input.Destination),
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.CreateAudited(ctx, &item, newAuditLog(actor, requestID, "create", "TransferManifest", "", item.Status, "created linked transfer manifest")); err != nil {
		return model.TransferManifest{}, fmt.Errorf("create 转运清单: %w", err)
	}
	return item, nil
}

func (s *transferManifestService) Update(ctx context.Context, id uint, input dto.UpdateTransferManifest, actor, requestID string) (model.TransferManifest, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.TransferManifest{}, err
	}
	if current.Status != "draft" {
		return model.TransferManifest{}, fmt.Errorf("%w: only draft manifests can be edited", ErrInvalidInput)
	}
	if err := validateTransferManifestBusinessFields(current.Code, input.Name, input.Facility, input.Owner, input.GeneratorCode, input.CarrierCode, input.WasteCode, input.Destination, input.Evidence, input.QuantityKg); err != nil {
		return model.TransferManifest{}, err
	}
	if _, err := s.generators.FindByCode(ctx, input.GeneratorCode); err != nil {
		return model.TransferManifest{}, fmt.Errorf("%w: generator %s does not exist", ErrInvalidInput, input.GeneratorCode)
	}
	if _, err := s.carriers.FindByCode(ctx, input.CarrierCode); err != nil {
		return model.TransferManifest{}, fmt.Errorf("%w: carrier %s does not exist", ErrInvalidInput, input.CarrierCode)
	}
	current.Name = strings.TrimSpace(input.Name)
	current.GeneratorCode = strings.ToUpper(strings.TrimSpace(input.GeneratorCode))
	current.CarrierCode = strings.ToUpper(strings.TrimSpace(input.CarrierCode))
	current.WasteCode = strings.ToUpper(strings.TrimSpace(input.WasteCode))
	current.QuantityKg = input.QuantityKg
	current.Destination = strings.TrimSpace(input.Destination)
	current.Description = strings.TrimSpace(input.Description)
	current.Facility = strings.TrimSpace(input.Facility)
	current.Owner = strings.TrimSpace(input.Owner)
	current.Category = strings.TrimSpace(input.Category)
	current.RiskLevel = input.RiskLevel
	current.MetricValue = input.MetricValue
	current.MetricUnit = strings.TrimSpace(input.MetricUnit)
	current.EffectiveAt = input.EffectiveAt.UTC()
	current.Evidence = strings.TrimSpace(input.Evidence)
	current.RelatedCode = strings.ToUpper(strings.TrimSpace(input.RelatedCode))
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "update", "TransferManifest", current.Status, current.Status, "updated draft manifest and evidence")); err != nil {
		return model.TransferManifest{}, fmt.Errorf("update 转运清单: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *transferManifestService) Transition(ctx context.Context, id uint, input dto.ManifestTransitionRequest, actor, requestID string) (model.TransferManifest, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.TransferManifest{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.TransferManifestTransitions, current.Status, target) {
		return model.TransferManifest{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	// Arrival weighing belongs to the in-transit phase only and must be recorded
	// through RegisterWeighing so the manifest stays in transit until it is done;
	// a transition request never carries a new arrival weight.
	if input.ArrivalWeightKg != nil {
		return model.TransferManifest{}, fmt.Errorf("%w: register arrival weight while in transit before sign-off", ErrInvalidInput)
	}
	loadingSupplied := input.LoadWeightKg != nil || strings.TrimSpace(input.VehiclePlate) != "" || strings.TrimSpace(input.EscortName) != ""
	if loadingSupplied && target != "submitted" && target != "in_transit" {
		return model.TransferManifest{}, fmt.Errorf("%w: loading weighing can only be recorded at submission or dispatch", ErrInvalidInput)
	}
	if target != "received" && strings.TrimSpace(input.WeightDeviationReason) != "" {
		return model.TransferManifest{}, fmt.Errorf("%w: deviation reason only applies to sign-off", ErrInvalidInput)
	}
	if target == "submitted" || target == "in_transit" {
		if err := s.validateLinkedParties(ctx, current); err != nil {
			return model.TransferManifest{}, err
		}
	}
	if err := applyWeighingInput(&current, input.LoadWeightKg, input.VehiclePlate, input.EscortName, nil, input.WeightDeviationReason); err != nil {
		return model.TransferManifest{}, err
	}
	if target == "submitted" {
		if err := requireLoadingWeighing(current); err != nil {
			return model.TransferManifest{}, err
		}
	}
	if target == "in_transit" {
		if err := requireLoadingWeighing(current); err != nil {
			return model.TransferManifest{}, err
		}
	}
	if target == "received" {
		if err := requireArrivalWeighing(current, strings.TrimSpace(input.WeightDeviationReason)); err != nil {
			return model.TransferManifest{}, err
		}
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "transition", "TransferManifest", before, target, input.Reason)); err != nil {
		return model.TransferManifest{}, fmt.Errorf("transition 转运清单: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *transferManifestService) RegisterWeighing(ctx context.Context, id uint, input dto.RegisterWeighingRequest, actor, requestID string) (model.TransferManifest, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.TransferManifest{}, err
	}
	switch current.Status {
	case "draft", "submitted", "in_transit":
	default:
		return model.TransferManifest{}, fmt.Errorf("%w: weighing can only be registered before sign-off", ErrInvalidInput)
	}
	if input.LoadWeightKg == nil && input.ArrivalWeightKg == nil &&
		strings.TrimSpace(input.VehiclePlate) == "" && strings.TrimSpace(input.EscortName) == "" {
		return model.TransferManifest{}, fmt.Errorf("%w: at least one weighing field is required", ErrInvalidInput)
	}
	if input.ArrivalWeightKg != nil && current.Status != "in_transit" {
		return model.TransferManifest{}, fmt.Errorf("%w: arrival weight can only be registered after dispatch", ErrInvalidInput)
	}
	if err := applyWeighingInput(&current, input.LoadWeightKg, input.VehiclePlate, input.EscortName, input.ArrivalWeightKg, input.WeightDeviationReason); err != nil {
		return model.TransferManifest{}, err
	}
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	detail := "registered transport weighing data"
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "weighing", "TransferManifest", current.Status, current.Status, detail)); err != nil {
		return model.TransferManifest{}, fmt.Errorf("register weighing 转运清单: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *transferManifestService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.Status != "draft" {
		return fmt.Errorf("%w: submitted manifests must be retained for compliance", ErrInvalidInput)
	}
	return s.repository.DeleteAudited(ctx, id, newAuditLog(actor, requestID, "delete", "TransferManifest", current.Status, "deleted", "soft deleted draft manifest"))
}

func (s *transferManifestService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func (s *transferManifestService) validateLinkedParties(ctx context.Context, manifest model.TransferManifest) error {
	generator, err := s.generators.FindByCode(ctx, manifest.GeneratorCode)
	if err != nil {
		return fmt.Errorf("%w: linked generator is unavailable", ErrInvalidInput)
	}
	if generator.Status != "active" || !generator.PermitExpiresAt.After(time.Now().UTC()) {
		return fmt.Errorf("%w: generator permit must be active and unexpired", ErrInvalidInput)
	}
	carrier, err := s.carriers.FindByCode(ctx, manifest.CarrierCode)
	if err != nil {
		return fmt.Errorf("%w: linked carrier is unavailable", ErrInvalidInput)
	}
	if carrier.Status != "verified" || !carrier.LicenseExpiresAt.After(time.Now().UTC()) {
		return fmt.Errorf("%w: carrier license must be verified and unexpired", ErrInvalidInput)
	}
	return nil
}

func validateTransferManifestBusinessFields(code, name, facility, owner, generatorCode, carrierCode, wasteCode, destination, evidence string, quantityKg float64) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" || strings.TrimSpace(generatorCode) == "" || strings.TrimSpace(carrierCode) == "" || strings.TrimSpace(wasteCode) == "" || strings.TrimSpace(destination) == "" {
		return fmt.Errorf("%w: manifest identity, parties and route are required", ErrInvalidInput)
	}
	if quantityKg <= 0 || strings.TrimSpace(evidence) == "" {
		return fmt.Errorf("%w: positive waste quantity and manifest evidence are required", ErrInvalidInput)
	}
	return nil
}

// applyWeighingInput merges one weighing submission into the manifest. Recorded
// weighing data is an immutable ledger: a previously registered value can only
// be re-submitted with the same value, never silently overwritten.
func applyWeighingInput(manifest *model.TransferManifest, loadWeightKg *float64, vehiclePlate, escortName string, arrivalWeightKg *float64, deviationReason string) error {
	plate := strings.TrimSpace(vehiclePlate)
	escort := strings.TrimSpace(escortName)
	reason := strings.TrimSpace(deviationReason)

	if loadWeightKg != nil {
		if *loadWeightKg <= 0 {
			return fmt.Errorf("%w: loading weight must be greater than zero", ErrInvalidInput)
		}
		if manifest.LoadWeightKg != nil && *manifest.LoadWeightKg != *loadWeightKg {
			return fmt.Errorf("%w: loading weight is already registered and cannot be changed", ErrInvalidInput)
		}
		if plate == "" {
			plate = strings.TrimSpace(manifest.VehiclePlate)
		}
		if escort == "" {
			escort = strings.TrimSpace(manifest.EscortName)
		}
		if plate == "" || escort == "" {
			return fmt.Errorf("%w: loading weight, vehicle plate and escort are all required at submission", ErrInvalidInput)
		}
		manifest.LoadWeightKg = loadWeightKg
	}
	if plate != "" {
		if manifest.VehiclePlate != "" && strings.TrimSpace(manifest.VehiclePlate) != plate {
			return fmt.Errorf("%w: vehicle plate is already registered and cannot be changed", ErrInvalidInput)
		}
		manifest.VehiclePlate = plate
	}
	if escort != "" {
		if manifest.EscortName != "" && strings.TrimSpace(manifest.EscortName) != escort {
			return fmt.Errorf("%w: escort is already registered and cannot be changed", ErrInvalidInput)
		}
		manifest.EscortName = escort
	}
	if arrivalWeightKg != nil {
		if *arrivalWeightKg <= 0 {
			return fmt.Errorf("%w: arrival weight must be greater than zero", ErrInvalidInput)
		}
		if manifest.LoadWeightKg == nil {
			return fmt.Errorf("%w: complete loading weighing before registering arrival weight", ErrInvalidInput)
		}
		if manifest.ArrivalWeightKg != nil && *manifest.ArrivalWeightKg != *arrivalWeightKg {
			return fmt.Errorf("%w: arrival weight is already registered and cannot be changed", ErrInvalidInput)
		}
		manifest.ArrivalWeightKg = arrivalWeightKg
	}
	if reason != "" {
		if manifest.ArrivalWeightKg == nil {
			return fmt.Errorf("%w: deviation reason requires a registered arrival weight", ErrInvalidInput)
		}
		if manifest.WeightDeviationReason != "" && strings.TrimSpace(manifest.WeightDeviationReason) != reason {
			return fmt.Errorf("%w: deviation reason is already recorded and cannot be changed", ErrInvalidInput)
		}
		manifest.WeightDeviationReason = reason
	}
	return nil
}

func requireLoadingWeighing(manifest model.TransferManifest) error {
	if manifest.LoadWeightKg == nil || *manifest.LoadWeightKg <= 0 {
		return fmt.Errorf("%w: register actual loading weight before submitting the manifest", ErrInvalidInput)
	}
	if strings.TrimSpace(manifest.VehiclePlate) == "" || strings.TrimSpace(manifest.EscortName) == "" {
		return fmt.Errorf("%w: vehicle plate and escort must be filled before submission", ErrInvalidInput)
	}
	return nil
}

func requireArrivalWeighing(manifest model.TransferManifest, reason string) error {
	if err := requireLoadingWeighing(manifest); err != nil {
		return err
	}
	if manifest.ArrivalWeightKg == nil || *manifest.ArrivalWeightKg <= 0 {
		return fmt.Errorf("%w: register arrival weight before sign-off; the manifest stays in transit", ErrInvalidInput)
	}
	deviation := manifest.WeightDeviationPct()
	if mathAbs(*deviation) > model.WeightDeviationThresholdPct &&
		reason == "" && strings.TrimSpace(manifest.WeightDeviationReason) == "" {
		return fmt.Errorf("%w: arrival weight differs from loading weight by more than %.0f%%; a deviation reason is required before sign-off", ErrInvalidInput, model.WeightDeviationThresholdPct)
	}
	return nil
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

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
	Transition(context.Context, uint, dto.TransitionTransferManifest, string, string) (model.TransferManifest, error)
	RegisterWeighing(context.Context, uint, dto.RegisterManifestWeighing, string, string) (model.TransferManifest, error)
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

func (s *transferManifestService) Transition(ctx context.Context, id uint, input dto.TransitionTransferManifest, actor, requestID string) (model.TransferManifest, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.TransferManifest{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.TransferManifestTransitions, current.Status, target) {
		return model.TransferManifest{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	if target == "submitted" || target == "in_transit" {
		if err := s.validateLinkedParties(ctx, current); err != nil {
			return model.TransferManifest{}, err
		}
	}

	detail := input.Reason
	switch target {
	case "submitted":
		// 提交联单必须同时完成装车称重登记；已登记过的旧联单保持原数据不变。
		if !current.HasLoadingWeighing() {
			if err := applyLoadingWeighing(&current, input.LoadingKg, input.VehiclePlate, input.EscortName); err != nil {
				return model.TransferManifest{}, err
			}
			detail = fmt.Sprintf("%s | 装车称重 %.2fkg，车牌 %s，押运员 %s", detail, *current.LoadingKg, current.VehiclePlate, current.EscortName)
		}
	case "in_transit":
		// 发运前置：旧联单允许先补录，发运时仍缺登记则保持在已提交状态。
		if !current.HasLoadingWeighing() {
			return model.TransferManifest{}, fmt.Errorf("%w: loading weight, vehicle plate and escort must be registered before dispatch", ErrInvalidInput)
		}
	case "received":
		// 签收前置：必须已登记到厂重量；偏差超过 3% 时必须填写偏差原因，否则保留在途。
		if !current.HasArrivalWeighing() {
			return model.TransferManifest{}, fmt.Errorf("%w: arrival weight must be registered before signing receipt", ErrInvalidInput)
		}
		reason := strings.TrimSpace(input.WeightDeviationReason)
		if current.WeightDeviationExceeded() {
			if reason == "" {
				return model.TransferManifest{}, fmt.Errorf("%w: arrival weight deviates more than %.0f%% from loading weight; deviation reason is required to sign", ErrInvalidInput, model.WeightDeviationLimit*100)
			}
			if current.WeightDeviationReason == "" {
				current.WeightDeviationReason = reason
			}
			detail = fmt.Sprintf("%s | 到厂偏差原因：%s", detail, current.WeightDeviationReason)
		}
	}

	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "transition", "TransferManifest", before, target, detail)); err != nil {
		return model.TransferManifest{}, fmt.Errorf("transition 转运清单: %w", err)
	}
	return s.repository.Get(ctx, id)
}

// RegisterWeighing persists 运输称重登记 without changing state. Loading
// registration is allowed while draft or submitted (covering legacy records
// that predate weighing); arrival registration is allowed in_transit after
// dispatch. Registered readings cannot be overwritten.
func (s *transferManifestService) RegisterWeighing(ctx context.Context, id uint, input dto.RegisterManifestWeighing, actor, requestID string) (model.TransferManifest, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.TransferManifest{}, err
	}

	registeringLoading := input.LoadingKg != 0 || strings.TrimSpace(input.VehiclePlate) != "" || strings.TrimSpace(input.EscortName) != ""
	registeringArrival := input.ArrivalKg != 0
	if !registeringLoading && !registeringArrival {
		return model.TransferManifest{}, fmt.Errorf("%w: provide loading weight/plate/escort or arrival weight", ErrInvalidInput)
	}
	if registeringLoading && registeringArrival {
		return model.TransferManifest{}, fmt.Errorf("%w: loading and arrival weighing must be registered separately", ErrInvalidInput)
	}

	detailParts := []string{strings.TrimSpace(input.Reason)}
	if registeringLoading {
		switch current.Status {
		case "draft", "submitted":
		default:
			return model.TransferManifest{}, fmt.Errorf("%w: loading weighing can only be registered before dispatch", ErrInvalidInput)
		}
		if current.HasLoadingWeighing() {
			return model.TransferManifest{}, fmt.Errorf("%w: loading weighing is already registered and cannot be changed", ErrInvalidInput)
		}
		if err := applyLoadingWeighing(&current, input.LoadingKg, input.VehiclePlate, input.EscortName); err != nil {
			return model.TransferManifest{}, err
		}
		detailParts = append(detailParts, fmt.Sprintf("补录装车称重 %.2fkg，车牌 %s，押运员 %s", *current.LoadingKg, current.VehiclePlate, current.EscortName))
	}
	if registeringArrival {
		if current.Status != "in_transit" {
			return model.TransferManifest{}, fmt.Errorf("%w: arrival weight can only be registered after dispatch", ErrInvalidInput)
		}
		if !current.HasLoadingWeighing() {
			return model.TransferManifest{}, fmt.Errorf("%w: backfill loading weighing before registering arrival weight", ErrInvalidInput)
		}
		if current.HasArrivalWeighing() {
			return model.TransferManifest{}, fmt.Errorf("%w: arrival weight is already registered and cannot be changed", ErrInvalidInput)
		}
		if input.ArrivalKg <= 0 {
			return model.TransferManifest{}, fmt.Errorf("%w: arrival weight must be greater than zero", ErrInvalidInput)
		}
		arrival := input.ArrivalKg
		current.ArrivalKg = &arrival
		ratio, _ := current.WeightDeviationRatio()
		detailParts = append(detailParts, fmt.Sprintf("登记到厂称重 %.2fkg，偏差 %.2f%%", arrival, ratio*100))
	}

	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "weighing", "TransferManifest", current.Status, current.Status, strings.Join(detailParts, " | "))); err != nil {
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

// applyLoadingWeighing validates and writes the dispatch-time registration.
func applyLoadingWeighing(manifest *model.TransferManifest, loadingKg float64, vehiclePlate, escortName string) error {
	plate := strings.TrimSpace(vehiclePlate)
	escort := strings.TrimSpace(escortName)
	if loadingKg <= 0 || plate == "" || escort == "" {
		return fmt.Errorf("%w: positive loading weight, vehicle plate and escort are required to submit the manifest", ErrInvalidInput)
	}
	loading := loadingKg
	manifest.LoadingKg = &loading
	manifest.VehiclePlate = strings.ToUpper(plate)
	manifest.EscortName = escort
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

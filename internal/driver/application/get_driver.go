package application

import (
	"context"
	"errors"
	"time"

	"transport-app/internal/driver/domain"
	"transport-app/internal/driver/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

type DriverResponseDTO struct {
	ID                    string    `json:"id"`
	DriverDisplayID       string    `json:"driver_id"`
	FirstName             string    `json:"first_name"`
	LastName              string    `json:"last_name"`
	Phone                 string    `json:"phone"`
	Email                 *string   `json:"email"`
	Address               *string   `json:"address"`
	LicenseNumber         string    `json:"license_number"`
	LicenseExpiry         time.Time `json:"license_expiry"`
	ExperienceYears       int64     `json:"experience_years"`
	Status                string    `json:"status"`
	EmergencyContactName  *string   `json:"emergency_contact_name"`
	EmergencyContactPhone *string   `json:"emergency_contact_phone"`
	Notes                 *string   `json:"notes"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type GetDriverQuery struct {
	ID       aggregate.DriverID
	TenantID shared.TenantID
}

type GetDriverUseCase struct {
	uow ports.UnitOfWork
}

func NewGetDriverUseCase(uow ports.UnitOfWork) *GetDriverUseCase {
	return &GetDriverUseCase{uow: uow}
}

func (uc *GetDriverUseCase) Execute(ctx context.Context, q GetDriverQuery) (DriverResponseDTO, error) {
	var dto DriverResponseDTO
	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Drivers().(domain.DriverRepository)
		if !ok {
			return errors.New("failed to retrieve driver repository")
		}

		d, err := repo.GetReadModel(txCtx, q.ID, q.TenantID)
		if err != nil {
			return err
		}

		dto = DriverResponseDTO{
			ID:                    d.ID,
			DriverDisplayID:       d.DriverDisplayID,
			FirstName:             d.FirstName,
			LastName:              d.LastName,
			Phone:                 d.Phone,
			Email:                 d.Email,
			Address:               d.Address,
			LicenseNumber:         d.LicenseNumber,
			LicenseExpiry:         d.LicenseExpiry,
			ExperienceYears:       d.ExperienceYears,
			Status:                d.Status,
			EmergencyContactName:  d.EmergencyContactName,
			EmergencyContactPhone: d.EmergencyContactPhone,
			Notes:                 d.Notes,
			CreatedAt:             d.CreatedAt,
			UpdatedAt:             d.UpdatedAt,
		}
		return nil
	})
	return dto, err
}

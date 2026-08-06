package service

import (
	"context"
	"fmt"
	"time"

	"transport-app/internal/domain"
)

// DriverService handles driver management.
type DriverService struct {
	baseService
}

// CreateDriver creates a new driver.
func (s *DriverService) CreateDriver(ctx context.Context, firstName, lastName, phone, email, address, licenseNumber string, licenseExpiry string, experience int64, emergencyContactName, emergencyContactPhone, notes *string) (domain.Driver, error) {
	if firstName == "" || lastName == "" || phone == "" || licenseNumber == "" {
		return domain.Driver{}, fmt.Errorf("first name, last name, phone, and license number are required")
	}

	expiry, err := time.Parse("2006-01-02", licenseExpiry)
	if err != nil {
		return domain.Driver{}, fmt.Errorf("invalid license expiry date: must be YYYY-MM-DD")
	}

	if expiry.Before(time.Now()) {
		return domain.Driver{}, fmt.Errorf("driver license has already expired")
	}

	// Check phone uniqueness
	if existing, err := s.store.GetDriverByPhone(ctx, phone); err == nil {
		return domain.Driver{}, fmt.Errorf("phone number %s is already registered to driver %s", phone, existing.DriverID)
	}

	driver := domain.Driver{
		ID:                    domain.DriverID(generateID()),
		DriverID:              generateDriverID("DRV"),
		FirstName:             sanitizeName(firstName),
		LastName:              sanitizeName(lastName),
		Phone:                 phone,
		Email:                 strPtr(email),
		Address:               strPtr(address),
		LicenseNumber:         licenseNumber,
		LicenseExpiry:         expiry,
		ExperienceYears:       experience,
		Status:                domain.DriverAvailable,
		EmergencyContactName:  emergencyContactName,
		EmergencyContactPhone: emergencyContactPhone,
		Notes:                 notes,
	}

	created, err := s.store.CreateDriver(ctx, driver)
	if err != nil {
		return domain.Driver{}, err
	}

	s.log.Info("driver created", "driver_id", created.ID)
	s.logAudit(ctx, nil, "create", "drivers", string(created.ID), nil, nil)
	return created, nil
}

// GetDriver retrieves a driver by ID.
func (s *DriverService) GetDriver(ctx context.Context, id domain.DriverID) (domain.Driver, error) {
	return s.store.GetDriverByID(ctx, id)
}

// GetDriverByNumber retrieves a driver by their display ID.
func (s *DriverService) GetDriverByNumber(ctx context.Context, driverID string) (domain.Driver, error) {
	return s.store.GetDriverByDriverID(ctx, driverID)
}

// ListDrivers retrieves drivers with search and pagination.
func (s *DriverService) ListDrivers(ctx context.Context, query, status string, limit, offset int) ([]domain.Driver, int64, error) {
	drivers, err := s.store.SearchDrivers(ctx, query, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.store.CountDrivers(ctx, query, status)
	if err != nil {
		return nil, 0, err
	}
	return drivers, total, nil
}

// UpdateDriver updates an existing driver.
func (s *DriverService) UpdateDriver(ctx context.Context, id domain.DriverID, firstName, lastName, phone, email, address, licenseNumber string, licenseExpiry string, experience int64, status domain.DriverStatus, emergencyContactName, emergencyContactPhone, notes *string) (domain.Driver, error) {
	driver, err := s.store.GetDriverByID(ctx, id)
	if err != nil {
		return domain.Driver{}, domain.ErrDriverNotFound
	}

	expiry, err := time.Parse("2006-01-02", licenseExpiry)
	if err != nil {
		return domain.Driver{}, fmt.Errorf("invalid license expiry date: must be YYYY-MM-DD")
	}

	if expiry.Before(time.Now()) {
		return domain.Driver{}, fmt.Errorf("driver license has already expired")
	}

	// Check phone uniqueness for other drivers
	if existing, err := s.store.GetDriverByPhone(ctx, phone); err == nil && existing.ID != id {
		return domain.Driver{}, fmt.Errorf("phone number %s is already registered to driver %s", phone, existing.DriverID)
	}

	driver.FirstName = sanitizeName(firstName)
	driver.LastName = sanitizeName(lastName)
	driver.Phone = phone
	driver.Email = strPtr(email)
	driver.Address = strPtr(address)
	driver.LicenseNumber = licenseNumber
	driver.LicenseExpiry = expiry
	driver.ExperienceYears = experience
	driver.Status = status
	driver.EmergencyContactName = emergencyContactName
	driver.EmergencyContactPhone = emergencyContactPhone
	driver.Notes = notes

	updated, err := s.store.UpdateDriver(ctx, driver)
	if err != nil {
		return domain.Driver{}, err
	}

	s.log.Info("driver updated", "driver_id", id)
	s.logAudit(ctx, nil, "update", "drivers", string(updated.ID), nil, nil)
	return updated, nil
}

// DeleteDriver deletes a driver.
func (s *DriverService) DeleteDriver(ctx context.Context, id domain.DriverID) error {
	if err := s.store.DeleteDriver(ctx, id); err != nil {
		return err
	}
	s.log.Info("driver deleted", "driver_id", id)
	s.logAudit(ctx, nil, "delete", "drivers", string(id), nil, nil)
	return nil
}

// SetDriverStatus updates a driver's status.
func (s *DriverService) SetDriverStatus(ctx context.Context, id domain.DriverID, status domain.DriverStatus) (domain.Driver, error) {
	driver, err := s.store.GetDriverByID(ctx, id)
	if err != nil {
		return domain.Driver{}, domain.ErrDriverNotFound
	}

	driver.Status = status
	updated, err := s.store.UpdateDriver(ctx, driver)
	if err != nil {
		return domain.Driver{}, err
	}

	s.log.Info("driver status updated", "driver_id", id)
	return updated, nil
}

// GetAvailableDrivers returns drivers available for assignment.
func (s *DriverService) GetAvailableDrivers(ctx context.Context) ([]domain.Driver, error) {
	return s.store.GetAvailableDrivers(ctx)
}

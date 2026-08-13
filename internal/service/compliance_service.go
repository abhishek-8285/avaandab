package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"transport-app/internal/domain"
)

// ComplianceService enforces legal and operational compliance across drivers and vehicles.
type ComplianceService struct {
	baseService
}

// ComplianceCheckResult represents the outcome of a compliance check.
type ComplianceCheckResult struct {
	Valid   bool     `json:"valid"`
	Blocked bool     `json:"blocked"`
	Reason  string   `json:"reason"`
	Alerts  []string `json:"alerts"`
}

// ValidateDriverCompliance checks driver license expiry and status.
// Rule 1: Sets blocked = true if license is expired or manually blocked.
func (s *ComplianceService) ValidateDriverCompliance(ctx context.Context, driverID domain.DriverID) (ComplianceCheckResult, error) {
	if s.store == nil {
		return ComplianceCheckResult{Valid: false, Reason: "store uninitialized"}, errors.New("store uninitialized")
	}
	driver, err := s.store.GetDriverByID(ctx, driverID)
	if err != nil {
		return ComplianceCheckResult{Valid: false, Reason: "driver not found"}, err
	}

	result := ComplianceCheckResult{Valid: true, Blocked: false, Alerts: []string{}}

	// Check manual block flag or blocked status
	if driver.Blocked || driver.Status == domain.DriverBlocked {
		reason := "driver status is blocked"
		if driver.BlockedReason != nil && *driver.BlockedReason != "" {
			reason = *driver.BlockedReason
		}
		result.Valid = false
		result.Blocked = true
		result.Reason = fmt.Sprintf("Compliance Hard-Block: %s", reason)
		return result, nil
	}

	// Check License Expiry
	now := time.Now()
	if !driver.LicenseExpiry.IsZero() && driver.LicenseExpiry.Before(now) {
		reason := fmt.Sprintf("Driver license %s expired on %s", driver.LicenseNumber, driver.LicenseExpiry.Format("2006-01-02"))
		result.Valid = false
		result.Blocked = true
		result.Reason = fmt.Sprintf("Compliance Hard-Block: %s", reason)

		// Auto-set blocked status on driver
		reasonStr := reason
		driver.Blocked = true
		driver.Status = domain.DriverBlocked
		driver.BlockedReason = &reasonStr
		_, _ = s.store.UpdateDriver(ctx, driver)

		s.logAudit(ctx, nil, "compliance_hard_block", "drivers", string(driverID), nil, &reasonStr)
		return result, nil
	}

	// Alert if license expires within 30 days
	if !driver.LicenseExpiry.IsZero() && driver.LicenseExpiry.Before(now.AddDate(0, 0, 30)) {
		result.Alerts = append(result.Alerts, fmt.Sprintf("License expires in %d days", int(time.Until(driver.LicenseExpiry).Hours()/24)))
	}

	return result, nil
}

// ValidateVehicleCompliance checks vehicle RC, Insurance, and Fitness expiry.
// Rule 1: Sets blocked = true if RC/Fitness/Insurance is expired or manually blocked.
func (s *ComplianceService) ValidateVehicleCompliance(ctx context.Context, vehicleID domain.VehicleID) (ComplianceCheckResult, error) {
	if s.store == nil {
		return ComplianceCheckResult{Valid: false, Reason: "store uninitialized"}, errors.New("store uninitialized")
	}
	vehicle, err := s.store.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		return ComplianceCheckResult{Valid: false, Reason: "vehicle not found"}, err
	}

	result := ComplianceCheckResult{Valid: true, Blocked: false, Alerts: []string{}}

	if vehicle.Blocked || vehicle.Status == domain.VehicleBlocked {
		reason := "vehicle status is blocked"
		if vehicle.BlockedReason != nil && *vehicle.BlockedReason != "" {
			reason = *vehicle.BlockedReason
		}
		result.Valid = false
		result.Blocked = true
		result.Reason = fmt.Sprintf("Compliance Hard-Block: %s", reason)
		return result, nil
	}

	now := time.Now()
	var expiredReason string

	if !vehicle.RCExpiry.IsZero() && vehicle.RCExpiry.Before(now) {
		expiredReason = fmt.Sprintf("Vehicle RC expired on %s", vehicle.RCExpiry.Format("2006-01-02"))
	} else if !vehicle.FitnessExpiry.IsZero() && vehicle.FitnessExpiry.Before(now) {
		expiredReason = fmt.Sprintf("Vehicle Fitness Certificate expired on %s", vehicle.FitnessExpiry.Format("2006-01-02"))
	} else if !vehicle.InsuranceExpiry.IsZero() && vehicle.InsuranceExpiry.Before(now) {
		expiredReason = fmt.Sprintf("Vehicle Insurance expired on %s", vehicle.InsuranceExpiry.Format("2006-01-02"))
	}

	if expiredReason != "" {
		result.Valid = false
		result.Blocked = true
		result.Reason = fmt.Sprintf("Compliance Hard-Block: %s", expiredReason)

		// Auto-set blocked status on vehicle
		vehicle.Blocked = true
		vehicle.Status = domain.VehicleBlocked
		vehicle.BlockedReason = &expiredReason
		_, _ = s.store.UpdateVehicle(ctx, vehicle)

		s.logAudit(ctx, nil, "compliance_hard_block", "vehicles", string(vehicleID), nil, &expiredReason)
		return result, nil
	}

	return result, nil
}

// EnforceDispatchCompliance verifies driver and vehicle compliance before assignment.
func (s *ComplianceService) EnforceDispatchCompliance(ctx context.Context, driverID *domain.DriverID, vehicleID *domain.VehicleID) error {
	if driverID != nil {
		res, err := s.ValidateDriverCompliance(ctx, *driverID)
		if err != nil {
			return err
		}
		if !res.Valid || res.Blocked {
			return errors.New(res.Reason)
		}
	}

	if vehicleID != nil {
		res, err := s.ValidateVehicleCompliance(ctx, *vehicleID)
		if err != nil {
			return err
		}
		if !res.Valid || res.Blocked {
			return errors.New(res.Reason)
		}
	}

	return nil
}

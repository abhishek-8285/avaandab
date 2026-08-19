// Package repository provides type aliases that delegate to domain sub-packages.
// The domain layer (internal/domain/*) is the source of truth for entity types,
// repository interfaces, and join types. This package re-exports them for
// backward compatibility with existing code.
package repository

import (
	"transport-app/internal/domain/audit"
	"transport-app/internal/domain/booking"
	"transport-app/internal/domain/company"
	"transport-app/internal/domain/customer"
	"transport-app/internal/domain/driver"
	"transport-app/internal/domain/invoice"
	"transport-app/internal/domain/payment"
	"transport-app/internal/domain/route"
	"transport-app/internal/domain/trip"
	"transport-app/internal/domain/user"
	"transport-app/internal/domain/vehicle"
)

// RoleRepository
type RoleRepository = user.RoleRepository

// UserRepository
type UserRepository = user.UserRepository

// UserWithRole is a user with their role name joined.
type UserWithRole = user.UserWithRole

// SessionRepository
type SessionRepository = user.SessionRepository

// SessionWithUser is a session with the associated user info joined.
type SessionWithUser = user.SessionWithUser

// DriverRepository
type DriverRepository = driver.DriverRepository

// VehicleRepository
type VehicleRepository = vehicle.VehicleRepository

// CustomerRepository
type CustomerRepository = customer.CustomerRepository

// RouteRepository
type RouteRepository = route.RouteRepository

// BookingRepository
type BookingRepository = booking.BookingRepository

// BookingWithJoins includes customer and route details.
type BookingWithJoins = booking.BookingWithJoins

// TripRepository
type TripRepository = trip.TripRepository

// TripWithJoins includes driver, vehicle, and route details.
type TripWithJoins = trip.TripWithJoins

// InvoiceRepository
type InvoiceRepository = invoice.InvoiceRepository

// InvoiceWithJoins includes customer, booking, and trip details.
type InvoiceWithJoins = invoice.InvoiceWithJoins

// PaymentRepository
type PaymentRepository = payment.PaymentRepository

// PaymentWithInvoice includes the associated invoice details.
type PaymentWithInvoice = payment.PaymentWithInvoice

// MonthlyRevenue is a month's revenue total.
type MonthlyRevenue = payment.MonthlyRevenue

// RevenueByDay is a single day's revenue total.
type RevenueByDay = payment.RevenueByDay

// BookingsByDay is a single day's booking count.
type BookingsByDay = booking.BookingsByDay

// CompanySettingsRepository
type CompanySettingsRepository = company.CompanySettingsRepository

// FileRepository
type FileRepository = audit.FileRepository

// AuditLogRepository
type AuditLogRepository = audit.AuditLogRepository

// AuditLogWithUser includes the associated user name.
type AuditLogWithUser = audit.AuditLogWithUser

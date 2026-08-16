package application

import (
	"context"
	"errors"
	"math"

	bookingdomain "transport-app/internal/booking/domain"
	bookingaggregate "transport-app/internal/booking/domain/aggregate"
	companydomain "transport-app/internal/domain/company"
	"transport-app/internal/invoice/domain"
	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

const moneyEpsilon = 0.01

type PricingResolver interface {
	GetCompanySettings(ctx context.Context) (companydomain.CompanySettings, error)
}

type derivedPricing struct {
	subtotal float64
	tax      float64
	discount float64
	total    float64
}

// GenerateInvoiceCommand contains parameters to create a new invoice.
type GenerateInvoiceCommand struct {
	TenantID   shared.TenantID
	BookingID  string
	CustomerID string
	TripID     *string
	Subtotal   float64
	Tax        float64
	Discount   float64
	Total      float64
}

// GenerateInvoiceUseCase generates an invoice aggregate.
type GenerateInvoiceUseCase struct {
	uow   ports.UnitOfWork
	idGen ports.IDGenerator
	clock ports.Clock
}

// NewGenerateInvoiceUseCase constructs a new GenerateInvoiceUseCase.
func NewGenerateInvoiceUseCase(uow ports.UnitOfWork, idGen ports.IDGenerator, clock ports.Clock) *GenerateInvoiceUseCase {
	return &GenerateInvoiceUseCase{uow: uow, idGen: idGen, clock: clock}
}

// Execute performs the generation and transaction commit.
func (uc *GenerateInvoiceUseCase) Execute(ctx context.Context, cmd GenerateInvoiceCommand) (aggregate.InvoiceID, error) {
	if cmd.BookingID == "" {
		return "", errors.New("booking ID is required")
	}
	if cmd.CustomerID == "" {
		return "", errors.New("customer ID is required")
	}

	var existingID aggregate.InvoiceID

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Invoices().(domain.InvoiceRepository)
		if !ok {
			return errors.New("failed to retrieve invoice repository")
		}

		existing, err := repo.FindByBookingID(txCtx, cmd.BookingID, cmd.TenantID)
		if err == nil && existing != nil {
			existingID = existing.ID
			return nil
		}

		subtotal, tax, discount, total := cmd.Subtotal, cmd.Tax, cmd.Discount, cmd.Total
		if pricing, ok := resolveBookingPricing(txCtx, cmd.TenantID, cmd.BookingID); ok {
			subtotal, tax, discount, total = pricing.subtotal, pricing.tax, pricing.discount, pricing.total
		} else if err := validateInvoiceAmounts(subtotal, tax, discount, total); err != nil {
			return err
		}

		id := aggregate.InvoiceID(uc.idGen.GenerateUUID())
		num := uc.idGen.GenerateDisplayID("INV")

		inv := aggregate.NewInvoiceAggregate(
			id,
			cmd.TenantID,
			num,
			cmd.BookingID,
			cmd.CustomerID,
			cmd.TripID,
			subtotal,
			tax,
			discount,
			total,
			aggregate.PaymentStatusPending,
			uc.clock.Now(),
		)

		if err := repo.Save(txCtx, inv); err != nil {
			return err
		}
		existingID = id
		return nil
	})

	if err != nil {
		return "", err
	}

	return existingID, nil
}

// resolveBookingPricing derives invoice money from the booking price and
// company tax settings, overriding any client-supplied amounts.
func resolveBookingPricing(txCtx ports.TxContext, tenantID shared.TenantID, bookingID string) (derivedPricing, bool) {
	bookingRepo, ok := txCtx.Repositories().Bookings().(bookingdomain.BookingRepository)
	if !ok {
		return derivedPricing{}, false
	}

	booking, err := bookingRepo.GetReadModel(txCtx, bookingaggregate.BookingID(bookingID), tenantID)
	if err != nil {
		return derivedPricing{}, false
	}

	subtotalMinor := int64(math.Round(booking.Price * 100))
	if subtotalMinor < 0 {
		subtotalMinor = 0
	}

	var taxMinor int64
	if settingsRepo, ok := txCtx.Repositories().AuditLogs().(PricingResolver); ok {
		settings, err := settingsRepo.GetCompanySettings(txCtx)
		if err == nil && settings.GSTEnabled {
			taxMinor = int64(math.Round(float64(subtotalMinor) * settings.GSTRate / 100.0))
		}
	}

	subtotal := float64(subtotalMinor) / 100.0
	tax := float64(taxMinor) / 100.0
	var discount float64
	total := subtotal + tax - discount

	return derivedPricing{subtotal: subtotal, tax: tax, discount: discount, total: total}, true
}

// validateInvoiceAmounts rejects client-supplied money that is negative or
// arithmetically inconsistent, using a small float epsilon.
func validateInvoiceAmounts(subtotal, tax, discount, total float64) error {
	if total < 0 {
		return errors.New("total cannot be negative")
	}
	if subtotal < 0 || tax < 0 || discount < 0 {
		return errors.New("invoice amounts cannot be negative")
	}
	if math.Abs(total-(subtotal+tax-discount)) > moneyEpsilon {
		return errors.New("invoice total does not match subtotal plus tax minus discount")
	}
	return nil
}

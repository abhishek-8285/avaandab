package service

import (
	"context"
	"log/slog"

	"transport-app/internal/config"
	"transport-app/internal/domain"
	bookingevents "transport-app/internal/domain/booking"
	tripevents "transport-app/internal/domain/trip"
	"transport-app/internal/events"
	"transport-app/internal/founder"
	"transport-app/internal/founder/alerts"
	"transport-app/internal/repository"
)

// Store is the combined repository interface used by all services.
// The SQLite implementation (SQLRepository) satisfies this interface.
// Future PostgreSQL implementations would also satisfy it.
type Store interface {
	repository.RoleRepository
	repository.UserRepository
	repository.SessionRepository
	repository.DriverRepository
	repository.VehicleRepository
	repository.CustomerRepository
	repository.RouteRepository
	repository.BookingRepository
	repository.TripRepository
	repository.InvoiceRepository
	repository.PaymentRepository
	repository.CompanySettingsRepository
	repository.FileRepository
	repository.AuditLogRepository
}

// Services holds all service instances and shared dependencies.
type Services struct {
	Auth        *AuthService
	Users       *UserService
	Drivers     *DriverService
	Vehicles    *VehicleService
	Customers   *CustomerService
	Routes      *RouteService
	Bookings    *BookingService
	Trips       *TripService
	Invoices    *InvoiceService
	Payments    *PaymentService
	Settings    *CompanySettingsService
	Dashboard   *DashboardService
	Files       *FileService
	Audit       *AuditLogService
	Founder     *founder.FounderService
	Compliance  *ComplianceService
	Settlements *DriverSettlementService
	Telemetry   *TelemetryService
	Kharcha     *KharchaService

	store Store
	cfg   *config.Config
	log   *slog.Logger
}

// NewServices creates all services with the given dependencies.
func NewServices(store Store, cfg *config.Config, log *slog.Logger) *Services {
	s := &Services{store: store, cfg: cfg, log: log}

	var tm repository.TxManager
	if dbGetter, ok := store.(repository.DBGetter); ok {
		tm = repository.NewTxManager(dbGetter)
	} else {
		log.Warn("store does not implement DB() — TxManager unavailable")
	}

	bs := baseService{store: store, cfg: cfg, log: log, txManager: tm, events: events.NewInMemoryBus()}
	s.Auth = &AuthService{baseService: bs}
	s.Users = &UserService{baseService: bs}
	s.Drivers = &DriverService{baseService: bs}
	s.Vehicles = &VehicleService{baseService: bs}
	s.Customers = &CustomerService{baseService: bs}
	s.Routes = &RouteService{baseService: bs}
	s.Bookings = &BookingService{baseService: bs}
	s.Trips = &TripService{baseService: bs}
	s.Invoices = &InvoiceService{baseService: bs}
	s.Payments = &PaymentService{baseService: bs}
	s.Settings = &CompanySettingsService{baseService: bs}
	s.Dashboard = &DashboardService{baseService: bs}
	s.Files = &FileService{baseService: bs}
	s.Audit = &AuditLogService{baseService: bs}
	s.Compliance = &ComplianceService{baseService: bs}
	s.Settlements = &DriverSettlementService{baseService: bs}
	s.Telemetry = &TelemetryService{baseService: bs}
	s.Kharcha = &KharchaService{baseService: bs}

	// Instantiate Telegram Bot Notifier if token configured, otherwise graceful fallback
	var founderNotifier founder.Notifier = alerts.NewTelegramBotNotifier(nil, 0)
	s.Founder = founder.NewFounderService(founderNotifier)
	s.Founder.RegisterEventHandlers(bs.events)

	s.initEventHandlers()

	return s
}

// initEventHandlers wires up event subscribers that coordinate across services.
func (s *Services) initEventHandlers() {
	bus := s.Bookings.events

	// BookingConfirmed → TripCreated: automatically create a trip when a booking is confirmed.
	bus.Subscribe("BookingConfirmed", func(ctx context.Context, e events.Event) error {
		evt, ok := e.Payload.(bookingevents.BookingConfirmedEvent)
		if !ok {
			return nil
		}
		b, err := s.store.GetBookingByID(ctx, evt.BookingID)
		if err != nil {
			return err
		}
		_, _ = s.Trips.CreateTrip(ctx, CreateTripRequest{
			BookingID:     &evt.BookingID,
			RouteID:       b.RouteID,
			DriverID:      nil,
			VehicleID:     nil,
			DepartureTime: b.PickupDate.Format("2006-01-02T15:04:05"),
			ArrivalTime:   "",
			Remarks:       "Auto-created from confirmed booking",
		})
		return nil
	})

	// TripCompleted → InvoiceGenerated: automatically generate an invoice when a trip completes.
	bus.Subscribe("TripCompleted", func(ctx context.Context, e events.Event) error {
		evt, ok := e.Payload.(tripevents.TripCompletedEvent)
		if !ok {
			return nil
		}
		_, _ = s.Invoices.GenerateInvoiceFromTrip(ctx, evt.TripID)
		return nil
	})

	// Rule 2: TripDelivered → Auto-generate GST Invoice + Driver Settlement Statement
	bus.Subscribe("TripDelivered", func(ctx context.Context, e events.Event) error {
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		tripIDVal, exists := payload["trip_id"]
		if !exists {
			return nil
		}
		tripID, ok := tripIDVal.(domain.TripID)
		if !ok {
			return nil
		}
		_, _ = s.Invoices.GenerateInvoiceFromTrip(ctx, tripID)
		_, _ = s.Settlements.CreateSettlementForTrip(ctx, tripID, 1200.0, 200.0, 50.0)
		return nil
	})
}

type baseService struct {
	store     Store
	cfg       *config.Config
	log       *slog.Logger
	txManager repository.TxManager
	events    events.EventBus
}

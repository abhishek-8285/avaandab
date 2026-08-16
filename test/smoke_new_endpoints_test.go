package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bookingApp "transport-app/internal/booking/application"
	"transport-app/internal/graphqlservice"
	invoiceApp "transport-app/internal/invoice/application"
	invoiceHandlers "transport-app/internal/invoice/presentation/api/handlers"
	paymentApp "transport-app/internal/payment/application"
	paymentHandlers "transport-app/internal/payment/presentation/api/handlers"
	"transport-app/internal/service"
	"transport-app/internal/shared/clock"
	"transport-app/internal/shared/id"
	"transport-app/internal/shared/ports"
	"transport-app/internal/shared/uow"
	tripApp "transport-app/internal/trip/application"
)

func setupSmokeTest(t *testing.T) (http.Handler, *service.Services, ports.UnitOfWork) {
	t.Helper()
	db := NewTestDB(t)
	sqlUoW := uow.NewSQLUnitOfWork(db)
	realClock := clock.NewRealClock()
	idGen := id.NewUUIDGenerator()
	authSvc := &stubAuthSvc{}

	invoiceH := invoiceHandlers.NewAPIInvoiceHandler(
		invoiceApp.NewGenerateInvoiceUseCase(sqlUoW, idGen, realClock),
		invoiceApp.NewGetInvoiceUseCase(sqlUoW),
		invoiceApp.NewListInvoicesUseCase(sqlUoW),
		invoiceApp.NewVoidInvoiceUseCase(sqlUoW, realClock),
		authSvc,
	)
	paymentH := paymentHandlers.NewAPIPaymentHandler(
		paymentApp.NewRecordPaymentUseCase(sqlUoW, idGen, realClock),
		paymentApp.NewGetPaymentUseCase(sqlUoW),
		paymentApp.NewListPaymentsUseCase(sqlUoW),
		paymentApp.NewReversePaymentUseCase(sqlUoW, idGen, realClock),
		paymentApp.NewListPaymentsByInvoiceUseCase(sqlUoW),
		paymentApp.NewRazorpayWebhookUseCase(paymentApp.NewRecordPaymentUseCase(sqlUoW, idGen, realClock), sqlUoW, "test-webhook-secret", realClock),
		authSvc,
	)
	listTripsUC := tripApp.NewListTripsUseCase(sqlUoW)
	graphqlH := graphqlservice.NewGraphQLHandler(listTripsUC)

	r := chi.NewRouter()
	r.Use(authInjectMiddleware)
	invoiceH.Register(r)
	paymentH.Register(r)
	r.Post("/query", graphqlH.ServeHTTP)

	return r, NewTestServices(t, db), sqlUoW
}

func doSmoke(t *testing.T, h http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	}
	var req *http.Request
	if bodyReader != nil {
		req = httptest.NewRequest(method, path, bodyReader)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func seedBooking(t *testing.T, svcs *service.Services, uow ports.UnitOfWork, price float64) (string, string) {
	t.Helper()
	ctx := context.Background()
	customer, err := svcs.Customers.CreateCustomer(ctx, "Smoke Co", "SC", "555-0001", "smk@example.com", "", "", "")
	require.NoError(t, err)
	route, err := svcs.Routes.CreateRoute(ctx, "Pune", "Mumbai", 200, 4, price, "")
	require.NoError(t, err)
	createUC := bookingApp.NewCreateBookingUseCase(uow, id.NewUUIDGenerator(), clock.NewRealClock())
	bid, err := createUC.Execute(ctx, bookingApp.CreateBookingCommand{
		TenantID:    "1",
		CustomerID:  string(customer.ID),
		RouteID:     string(route.ID),
		PickupDate:  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		VehicleType: "Truck",
		Passengers:  1,
		Price:       price,
	})
	require.NoError(t, err)
	return string(bid), string(customer.ID)
}

func TestSmoke_VoidInvoiceEndpoint(t *testing.T) {
	router, svcs, uow := setupSmokeTest(t)
	bookingID, customerID := seedBooking(t, svcs, uow, 1000.0)

	rr := doSmoke(t, router, http.MethodPost, "/api/v1/invoices", map[string]interface{}{
		"booking_id":  bookingID,
		"customer_id": customerID,
		"subtotal":    1000.0,
		"tax":         0.0,
		"discount":    0.0,
		"total":       1000.0,
	})
	require.Equal(t, http.StatusCreated, rr.Code, "body: %s", rr.Body.String())

	var invResp struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &invResp))
	require.NotEmpty(t, invResp.ID)

	rr = doSmoke(t, router, http.MethodPost, "/api/v1/invoices/"+invResp.ID+"/void", nil)
	require.Equal(t, http.StatusNoContent, rr.Code, "void should return 204, got %d: %s", rr.Code, rr.Body.String())
}

func TestSmoke_PaymentWithInvoiceSyncAndListByInvoice(t *testing.T) {
	router, svcs, uow := setupSmokeTest(t)
	bookingID, customerID := seedBooking(t, svcs, uow, 1000.0)

	rr := doSmoke(t, router, http.MethodPost, "/api/v1/invoices", map[string]interface{}{
		"booking_id":  bookingID,
		"customer_id": customerID,
		"subtotal":    1000.0,
		"tax":         0.0,
		"discount":    0.0,
		"total":       1000.0,
	})
	require.Equal(t, http.StatusCreated, rr.Code, "body: %s", rr.Body.String())
	var invResp struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &invResp))
	invoiceID := invResp.ID

	rr = doSmoke(t, router, http.MethodPost, "/api/v1/payments", map[string]interface{}{
		"invoice_id":   invoiceID,
		"payment_date": time.Now().Format(time.RFC3339),
		"amount":       500.0,
		"method":       "upi",
		"reference":    "SMOKE-REF-001",
	})
	require.Equal(t, http.StatusCreated, rr.Code, "body: %s", rr.Body.String())
	var payResp struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payResp))
	require.NotEmpty(t, payResp.ID)

	rr = doSmoke(t, router, http.MethodGet, "/api/v1/payments/by-invoice/"+invoiceID, nil)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	var listResp struct {
		Payments []map[string]interface{} `json:"payments"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &listResp))
	require.Len(t, listResp.Payments, 1, "expected 1 payment for invoice")
	assert.Equal(t, invoiceID, listResp.Payments[0]["invoice_id"])
}

func TestSmoke_ReversePaymentEndpoint(t *testing.T) {
	router, svcs, uow := setupSmokeTest(t)
	bookingID, customerID := seedBooking(t, svcs, uow, 1000.0)

	rr := doSmoke(t, router, http.MethodPost, "/api/v1/invoices", map[string]interface{}{
		"booking_id":  bookingID,
		"customer_id": customerID,
		"subtotal":    1000.0,
		"total":       1000.0,
	})
	require.Equal(t, http.StatusCreated, rr.Code, "body: %s", rr.Body.String())
	var invResp struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &invResp))
	invoiceID := invResp.ID

	rr = doSmoke(t, router, http.MethodPost, "/api/v1/payments", map[string]interface{}{
		"invoice_id":   invoiceID,
		"payment_date": time.Now().Format(time.RFC3339),
		"amount":       300.0,
		"method":       "cash",
		"reference":    "SMOKE-REV-001",
	})
	require.Equal(t, http.StatusCreated, rr.Code, "body: %s", rr.Body.String())
	var payResp struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payResp))
	originalPayID := payResp.ID

	rr = doSmoke(t, router, http.MethodPost, "/api/v1/payments/"+originalPayID+"/reverse", map[string]interface{}{
		"original_payment_id": originalPayID,
		"reason":              "entered in error",
	})
	require.Equal(t, http.StatusCreated, rr.Code, "reverse should return 201, got %d: %s", rr.Code, rr.Body.String())
	var revResp struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &revResp))
	assert.NotEmpty(t, revResp.ID)
	assert.NotEqual(t, originalPayID, revResp.ID)

	rr = doSmoke(t, router, http.MethodGet, "/api/v1/payments/by-invoice/"+invoiceID, nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var listResp struct {
		Payments []map[string]interface{} `json:"payments"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &listResp))
	assert.Len(t, listResp.Payments, 2, "expected 2 payments (original + reversal)")
}

func TestSmoke_GraphQLRealData(t *testing.T) {
	router, svcs, sqlUoW := setupSmokeTest(t)
	ctx := context.Background()

	route, err := svcs.Routes.CreateRoute(ctx, "Delhi", "Jaipur", 280, 5, 8000, "")
	require.NoError(t, err)

	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()
	createTripUC := tripApp.NewCreateTripUseCase(sqlUoW, idGen, realClock)
	tripID, err := createTripUC.Execute(ctx, tripApp.CreateTripCommand{
		TenantID:      "1",
		RouteID:       string(route.ID),
		DepartureTime: time.Now().Add(24 * time.Hour),
		Remarks:       "smoke test trip",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tripID)

	rr := doSmoke(t, router, http.MethodPost, "/query", map[string]interface{}{
		"query": "{ activeTrips { id } }",
	})
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

	var gqlResp struct {
		Data struct {
			ActiveTrips []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
				Origin string `json:"origin"`
				Dest   string `json:"destination"`
			} `json:"activeTrips"`
			ServerTime string `json:"serverTime"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &gqlResp))
	assert.NotEmpty(t, gqlResp.Data.ServerTime, "serverTime should be present")
}

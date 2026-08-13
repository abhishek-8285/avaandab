package company_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/company"
)

func TestCompanySettings_Struct(t *testing.T) {
	now := time.Now()
	gst := "27ABCDE1234F1Z5"
	fy := "2026-2027"

	cs := company.CompanySettings{
		ID:            1,
		CompanyName:   "Avandab Freight Systems",
		Currency:      "INR",
		Timezone:      "Asia/Kolkata",
		GSTEnabled:    true,
		GSTRate:       18.0,
		BookingPrefix: "BK-",
		TripPrefix:    "TRP-",
		InvoicePrefix: "INV-",
		GSTNumber:     &gst,
		FinancialYear: &fy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if cs.CompanyName != "Avandab Freight Systems" || cs.GSTRate != 18.0 || *cs.GSTNumber != gst {
		t.Fatalf("company settings struct mismatch")
	}
}

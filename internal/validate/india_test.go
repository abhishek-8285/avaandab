package validate

import "testing"

func TestGSTIN(t *testing.T) {
	valid := []string{"27ABCDE1234F1Z5", "07AACCC9876L1ZS", "29AAACR5055E1ZD"}
	invalid := []string{"27ABCDE1234F1Z", "7ABCDE1234F1Z5", "27abcde1234f1z5", "TAX-9988-77", ""}
	for _, s := range valid {
		if !ValidGSTIN(s) {
			t.Errorf("ValidGSTIN(%q) = false, want true", s)
		}
	}
	for _, s := range invalid {
		if ValidGSTIN(s) {
			t.Errorf("ValidGSTIN(%q) = true, want false", s)
		}
	}
}

func TestPAN(t *testing.T) {
	if !ValidPAN("ABCDE1234F") {
		t.Error("ValidPAN(ABCDE1234F) = false, want true")
	}
	for _, s := range []string{"ABCDE12345", "abcde1234F", "ADE1234F", ""} {
		if ValidPAN(s) {
			t.Errorf("ValidPAN(%q) = true, want false", s)
		}
	}
}

func TestPhoneIN(t *testing.T) {
	valid := []string{"9876543210", "+919876543210", "+91 98765 43210", "09876543210", "919876543210", "6123456789"}
	invalid := []string{"1234567890", "987654321", "+12025550123", "98765-43210extra", ""}
	for _, s := range valid {
		if !ValidPhoneIN(s) {
			t.Errorf("ValidPhoneIN(%q) = false, want true", s)
		}
	}
	for _, s := range invalid {
		if ValidPhoneIN(s) {
			t.Errorf("ValidPhoneIN(%q) = true, want false", s)
		}
	}
}

func TestVehicleReg(t *testing.T) {
	valid := []string{"MH01AB1234", "DL8C1234", "KA05E9988", "GJ 01 RK 4521", "TR01B0001"}
	invalid := []string{"MH1", "1234", "mh01ab1234", ""}
	for _, s := range valid {
		if !ValidVehicleReg(s) {
			t.Errorf("ValidVehicleReg(%q) = false, want true", s)
		}
	}
	for _, s := range invalid {
		if ValidVehicleReg(s) {
			t.Errorf("ValidVehicleReg(%q) = true, want false", s)
		}
	}
}

// Package validate holds India-specific format validators shared by web
// forms (HTML pattern attrs) and server-side handlers.
package validate

import "regexp"

// GSTIN: 2-digit state code + PAN-shaped block + entity/blank/Z + checksum char.
var gstinRe = regexp.MustCompile(`^\d{2}[A-Z]{5}\d{4}[A-Z][1-9A-Z]Z[0-9A-Z]$`)

// PAN: 5 letters, 4 digits, 1 letter.
var panRe = regexp.MustCompile(`^[A-Z]{5}\d{4}[A-Z]$`)

// Indian mobile: optional +91/0/91 prefix, then 10 digits starting 6-9.
// Spaces/dashes accepted between groups.
var phoneRe = regexp.MustCompile(`^(?:\+?91[\s-]?|0)?[6-9]\d{4}[\s-]?\d{5}$`)

// Vehicle registration (loose, all India RTO styles): MH01AB1234, DL8C1234,
// KA05E9988 etc. Two letters, 1-2 digits, up to 3 letters, 4 digits.
var vehicleRegRe = regexp.MustCompile(`^[A-Z]{2}\s?\d{1,2}\s?[A-Z]{0,3}\s?\d{4}$`)

func ValidGSTIN(s string) bool      { return gstinRe.MatchString(s) }
func ValidPAN(s string) bool        { return panRe.MatchString(s) }
func ValidPhoneIN(s string) bool    { return phoneRe.MatchString(s) }
func ValidVehicleReg(s string) bool { return vehicleRegRe.MatchString(s) }

// GSTINPattern / PANPattern / PhonePattern / VehicleRegPattern are the HTML
// input `pattern` attribute equivalents of the validators above.
const (
	GSTINPattern      = `\d{2}[A-Z]{5}\d{4}[A-Z][1-9A-Z]Z[0-9A-Z]`
	PANPattern        = `[A-Z]{5}\d{4}[A-Z]`
	PhonePattern      = `(\+?91[\s-]?|0)?[6-9][0-9]{4}[\s-]?[0-9]{5}`
	VehicleRegPattern = `[A-Z]{2}\s?\d{1,2}\s?[A-Z]{0,3}\s?\d{4}`
)

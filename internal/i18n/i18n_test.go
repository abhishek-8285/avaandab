package i18n

import "testing"

func TestTResolvesBundles(t *testing.T) {
	if got := T("hi", "nav.bookings"); got != "बुकिंग" {
		t.Errorf("T(hi, nav.bookings) = %q, want बुकिंग", got)
	}
	if got := T("en", "nav.bookings"); got != "Bookings" {
		t.Errorf("T(en, nav.bookings) = %q, want Bookings", got)
	}
}

func TestTFallsBackToEnglishThenKey(t *testing.T) {
	// Simulate an untranslated key: drop it from the hi bundle, expect en text.
	const key = "nav.bookings"
	hi := bundles["hi"]
	saved := hi[key]
	delete(hi, key)
	defer func() { hi[key] = saved }()

	if got := T("hi", key); got != "Bookings" {
		t.Errorf("hi missing key should fall back to en, got %q", got)
	}
	// key missing everywhere
	if got := T("hi", "no.such.key"); got != "no.such.key" {
		t.Errorf("missing key should return itself, got %q", got)
	}
}

func TestNormalize(t *testing.T) {
	for _, in := range []string{"hi", "hi-IN", "HI", "हिन्दी"} {
		if Normalize(in) != "hi" {
			t.Errorf("Normalize(%q) != hi", in)
		}
	}
	for _, in := range []string{"", "en", "en-US", "ta", "fr"} {
		if Normalize(in) != "en" {
			t.Errorf("Normalize(%q) != en", in)
		}
	}
}

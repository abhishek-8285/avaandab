package logging

import (
	"testing"
)

func TestRedactKeyValueSecrets(t *testing.T) {
	in := `login failed for body {"password":"hunter2secret","name":"ram"}`
	out := Redact(in)
	if out == in {
		t.Fatal("nothing redacted")
	}
	if contains(out, "hunter2secret") {
		t.Fatalf("password leaked: %q", out)
	}
	if !contains(out, `"name":"ram"`) {
		t.Fatalf("non-sensitive fields must survive: %q", out)
	}
}

func TestRedactBearer(t *testing.T) {
	out := Redact("authz: Bearer eyJhbGciOiJIUzI1NiIsInR5.abc.def")
	if contains(out, "eyJhbGciOiJIUzI1NiIsInR5") {
		t.Fatalf("bearer token leaked: %q", out)
	}
}

func TestRedactCardNumbers(t *testing.T) {
	withCard := "charge failed for 4111 1111 1111 1111 (test card)"
	out := Redact(withCard)
	if contains(out, "4111") {
		t.Fatalf("card number leaked: %q", out)
	}
	plain := "order 12345678 placed"
	if Redact(plain) != plain {
		t.Fatalf("non-card digit runs must not be masked: %q", Redact(plain))
	}
}

func TestRedactEmptyAndCleanStringsUntouched(t *testing.T) {
	if got := Redact(""); got != "" {
		t.Fatalf("got %q", got)
	}
	clean := "booking BK-1001 confirmed for trip 42"
	if Redact(clean) != clean {
		t.Fatalf("clean string mutated: %q", Redact(clean))
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && indexOf(haystack, needle) >= 0
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}

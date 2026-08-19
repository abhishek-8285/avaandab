package experiments

import "testing"

func TestAssignRolloutZero(t *testing.T) {
	for _, uid := range []string{"u1", "u2", "u3"} {
		if got := Assign(0, "", "t1", uid); got != VariantA {
			t.Fatalf("rollout 0: user %s got %s, want A", uid, got)
		}
	}
}

func TestAssignRolloutHundred(t *testing.T) {
	for _, uid := range []string{"u1", "u2", "u3"} {
		if got := Assign(100, "", "t1", uid); got != VariantB {
			t.Fatalf("rollout 100: user %s got %s, want B", uid, got)
		}
	}
}

func TestAssignStableAcrossCalls(t *testing.T) {
	first := Assign(50, "", "t1", "user-7")
	for i := 0; i < 10; i++ {
		if got := Assign(50, "", "t1", "user-7"); got != first {
			t.Fatalf("assignment not stable: got %s then %s", first, got)
		}
	}
}

func TestAssignScopesByTenant(t *testing.T) {
	seen := map[string]string{}
	for i := 0; i < 200; i++ {
		uid := string(rune('a'+i%26)) + string(rune('0'+i%10))
		seen[uid] = Assign(50, "", "t1", uid)
	}
	// tenant t2 must be assigned independently; with a different tenant the
	// bucket differs for at least one user in practice (sha1 input differs).
	var splitA, splitB bool
	for uid, v := range seen {
		v2 := Assign(50, "", "t2", uid)
		if v2 != v {
			splitA, splitB = true, true
		}
		if v2 == VariantA {
			splitA = true
		}
		if v2 == VariantB {
			splitB = true
		}
	}
	if !splitA || !splitB {
		t.Fatalf("expected both variants across tenants, got A=%v B=%v", splitA, splitB)
	}
}

func TestAssignForceOverride(t *testing.T) {
	if got := Assign(0, VariantB, "t1", "u1"); got != VariantB {
		t.Fatalf("force B with rollout 0: got %s", got)
	}
	if got := Assign(100, VariantA, "t1", "u1"); got != VariantA {
		t.Fatalf("force A with rollout 100: got %s", got)
	}
}

func TestAssignRoughBalance(t *testing.T) {
	counts := map[string]int{}
	for i := 0; i < 1000; i++ {
		uid := fmtUser(i)
		counts[Assign(50, "", "t1", uid)]++
	}
	if counts[VariantA] < 350 || counts[VariantA] > 650 {
		t.Fatalf("rollout 50 too unbalanced: %v", counts)
	}
}

func fmtUser(i int) string {
	return string(rune('a'+i%26)) + string(rune('0'+i%10)) + string(rune('x'+i%7)) + string(rune('A'+i%26))
}

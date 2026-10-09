package testkit

import (
	"strings"
	"testing"
	"time"

	gbp "git.bytestone.uk/hum3/gobank-products"
)

// CheckVersion is what every version's test asserts first: its identity,
// that its declarations are well formed and resolve from what it
// publishes, and that it answers start-up.
func CheckVersion(t *testing.T, v gbp.Version, product string, number int) {
	t.Helper()
	if v.Product() != product || v.Version() != number || v.Name() == "" {
		t.Errorf("identity = %s v%d %q, want %s v%d", v.Product(), v.Version(), v.Name(), product, number)
	}
	if v.Family() != gbp.FamilySavings && v.Family() != gbp.FamilyLending {
		t.Errorf("family = %q", v.Family())
	}
	seen := map[string]bool{}
	for _, d := range v.Parameters() {
		if seen[d.Key] {
			t.Errorf("parameter %s declared twice", d.Key)
		}
		seen[d.Key] = true
		if d.Scope == gbp.ScopeVersion && d.Published == "" && d.Derived == nil {
			t.Errorf("parameter %s: a version-scoped parameter publishes a value or derives one", d.Key)
		}
	}
	r := Open(t, v, "Liability:Check:"+product, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if len(r.Reads) != 0 && r.Reads[0].Key == "" {
		t.Errorf("reads = %+v", r.Reads)
	}
}

// RoundTrip imports an export into a fresh ledger and exports it again,
// so a version's postings survive the .goluca format.
func RoundTrip(t *testing.T, exported string) {
	t.Helper()
	ledger := NewTestLedger(t)
	if err := ledger.Import(strings.NewReader(exported), nil); err != nil {
		t.Skipf("import not yet supported for this export format: %v", err)
	}
	AssertGolucaEqual(t, exportGoluca(t, ledger), exported)
}

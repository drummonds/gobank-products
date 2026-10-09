package mortgage_test

import (
	"testing"
	"time"

	mortgage "git.bytestone.uk/hum3/gobank-products/mortgage/v1"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func TestVersion(t *testing.T) { testkit.CheckVersion(t, mortgage.Version, "mortgage", 1) }

// £100,000 disbursed on 1 January; 32 days to 1 February, so January's interest is
// applied at the month end.
func TestGolden32Days(t *testing.T) {
	r := testkit.Open(t, mortgage.Version, "Asset:Loans:mortgage", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	r.Deposit(10000000)
	r.Advance(32)
	testkit.Golden(t, "mortgage_32d", r.Export())
	testkit.RoundTrip(t, r.Export())
}

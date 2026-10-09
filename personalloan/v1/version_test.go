package personalloan_test

import (
	"testing"
	"time"

	personalloan "git.bytestone.uk/hum3/gobank-products/personalloan/v1"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func TestVersion(t *testing.T) { testkit.CheckVersion(t, personalloan.Version, "personal-loan", 1) }

// £5,000 disbursed on 1 January; 32 days to 1 February, so January's interest is
// applied at the month end.
func TestGolden32Days(t *testing.T) {
	r := testkit.Open(t, personalloan.Version, "Asset:Loans:alice", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	r.Deposit(500000)
	r.Advance(32)
	testkit.Golden(t, "personal_loan_32d", r.Export())
	testkit.RoundTrip(t, r.Export())
}

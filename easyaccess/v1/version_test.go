package easyaccess_test

import (
	"testing"
	"time"

	easyaccess "git.bytestone.uk/hum3/gobank-products/easyaccess/v1"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func TestVersion(t *testing.T) { testkit.CheckVersion(t, easyaccess.Version, "easy-access", 1) }

// A £1,000 deposit on 1 January; 32 days to 1 February, so January's
// interest at 1.5% is applied at the month end.
func TestGolden32Days(t *testing.T) {
	r := testkit.Open(t, easyaccess.Version, "Liability:Savings:alice", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	r.Deposit(100000)
	r.Advance(32)
	testkit.Golden(t, "easy_access_32d", r.Export())
	testkit.RoundTrip(t, r.Export())
}

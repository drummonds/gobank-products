package overdraft_test

import (
	"testing"
	"time"

	gbp "git.bytestone.uk/hum3/gobank-products"
	overdraft "git.bytestone.uk/hum3/gobank-products/overdraft/v1"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestVersion(t *testing.T) { testkit.CheckVersion(t, overdraft.Version, "overdraft", 1) }

func TestDrawingPastTheLimitIsRefused(t *testing.T) {
	r := testkit.Open(t, overdraft.Version, "Liability:Current:alice", day(2026, 1, 1))
	r.Withdraw(100000) // to the limit
	if err := r.TryWithdraw(1); !gbp.IsRefusal(err) {
		t.Errorf("a penny past the limit: %v, want a refusal", err)
	}
	r.Deposit(50000)
	if err := r.TryWithdraw(50000); err != nil {
		t.Errorf("back to the limit: %v", err)
	}
}

// £500 drawn on 1 January, within the £1,000 limit; 32 days to 1 February.
func TestGolden32Days(t *testing.T) {
	r := testkit.Open(t, overdraft.Version, "Liability:Current:alice", day(2026, 1, 1))
	r.Withdraw(50000)
	r.Advance(32)
	testkit.Golden(t, "overdraft_32d", r.Export())
	testkit.RoundTrip(t, r.Export())
}

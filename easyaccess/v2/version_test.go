package easyaccess_test

import (
	"testing"
	"time"

	gbp "git.bytestone.uk/hum3/gobank-products"
	easyaccess "git.bytestone.uk/hum3/gobank-products/easyaccess/v2"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// v2 reads the bank's savings rate, so its start-up needs the bank to
// have one: the policy's, or in a test a setting.
func withBankRate(r *testkit.Runner) {
	r.SetBank(gbp.SavingsRateKey, "200", day(2025, 1, 1))
}

func TestVersion(t *testing.T) {
	testkit.CheckVersionWith(t, easyaccess.Version, "easy-access", 2, withBankRate)
}

// The rate tracks the bank's savings rate at 15 bps below it, floored at
// zero, on the day the bank's rate moves: a change to the bank's rate is a
// change to the account's with nothing set on the product.
func TestTracksTheBankSavingsRate(t *testing.T) {
	r := testkit.OpenWith(t, easyaccess.Version, "Liability:Savings:alice", day(2026, 1, 1), func(r *testkit.Runner) {
		withBankRate(r)
		r.SetBank(gbp.SavingsRateKey, "150", day(2026, 1, 15))
		r.SetBank(gbp.SavingsRateKey, "10", day(2026, 2, 1))
	})
	for _, c := range []struct {
		on   time.Time
		want int64
	}{{day(2026, 1, 1), 185}, {day(2026, 1, 14), 185}, {day(2026, 1, 15), 135}, {day(2026, 2, 1), 0}} {
		r.AdvanceTo(c.on)
		p, err := r.Parameter("rate_bps")
		if err != nil || p.Bps() != c.want {
			t.Errorf("rate on %s = %d, %v; want %d", c.on.Format("2006-01-02"), p.Bps(), err, c.want)
		}
	}
	if d, _ := gbp.Declared(easyaccess.Version, "rate_bps"); !d.IsDerived() {
		t.Error("rate_bps is derived: the bank sets the savings rate, not the product")
	}
}

// A £1,000 deposit on 1 January at a bank savings rate of 2.00% (so 1.85%
// here); the bank cuts to 1.50% on 15 January; 32 days to 1 February, so
// January's interest is applied at the month end: 14 days at 185 bps and
// 17 at 135.
func TestGolden32Days(t *testing.T) {
	r := testkit.OpenWith(t, easyaccess.Version, "Liability:Savings:alice", day(2026, 1, 1), func(r *testkit.Runner) {
		withBankRate(r)
		r.SetBank(gbp.SavingsRateKey, "150", day(2026, 1, 15))
	})
	r.Deposit(100000)
	r.Advance(32)
	if pos, ok := r.Position(day(2026, 1, 31)); !ok || pos.Balance != -100133 { // 14 × 5.07p + 17 × 3.70p = 133.8p, truncated
		t.Errorf("31 Jan position = %+v, %v; want -100133", pos, ok)
	}
	testkit.Golden(t, "easy_access_v2_32d", r.Export())
	testkit.RoundTrip(t, r.Export())
}

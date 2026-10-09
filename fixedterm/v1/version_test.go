package fixedterm_test

import (
	"testing"
	"time"

	gbp "git.bytestone.uk/hum3/gobank-products"
	fixedterm "git.bytestone.uk/hum3/gobank-products/fixedterm/v1"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestVersion(t *testing.T) { testkit.CheckVersion(t, fixedterm.Version, "fixed-term", 1) }

func TestMaturityIsFixedAtOpen(t *testing.T) {
	r := testkit.Open(t, fixedterm.Version, "Liability:Savings:fixed", day(2026, 1, 1))
	if p, err := r.Parameter("maturity_day"); err != nil || p.Day() != day(2028, 1, 1) {
		t.Errorf("maturity = %v, %v; want 2028-01-01", p.Day(), err)
	}
}

func TestWithdrawalBeforeMaturityIsRefused(t *testing.T) {
	r := testkit.Open(t, fixedterm.Version, "Liability:Savings:fixed", day(2026, 1, 1))
	r.Deposit(500000)
	r.AdvanceTo(day(2027, 12, 31))
	if err := r.TryWithdraw(100); !gbp.IsRefusal(err) {
		t.Errorf("withdrawal the day before maturity: %v, want a refusal", err)
	}
	r.Advance(1)
	if err := r.TryWithdraw(100); err != nil {
		t.Errorf("withdrawal on the maturity day: %v", err)
	}
}

// £5,000 for the two-year term, withdrawn in full the day after maturity.
func TestGoldenMaturity(t *testing.T) {
	r := testkit.Open(t, fixedterm.Version, "Liability:Savings:fixed", day(2026, 1, 1))
	r.Deposit(500000)
	r.AdvanceTo(day(2028, 1, 2))
	r.Withdraw(500000)
	testkit.Golden(t, "fixed_term_maturity", r.Export())
	testkit.RoundTrip(t, r.Export())
}

package isa_test

import (
	"testing"
	"time"

	gbp "git.bytestone.uk/hum3/gobank-products"
	isa "git.bytestone.uk/hum3/gobank-products/isa/v1"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestVersion(t *testing.T) { testkit.CheckVersion(t, isa.Version, "isa", 1) }

func TestDepositsOverTheAllowanceAreRefused(t *testing.T) {
	r := testkit.Open(t, isa.Version, "Liability:Savings:isa", day(2026, 1, 1))
	r.Deposit(1500000)
	if err := r.TryDeposit(600000); !gbp.IsRefusal(err) {
		t.Errorf("£6,000 on top of £15,000: %v, want a refusal", err)
	}
	r.Deposit(500000) // exactly the allowance
	if err := r.TryDeposit(1); !gbp.IsRefusal(err) {
		t.Errorf("a penny over: %v, want a refusal", err)
	}
	r.Withdraw(1000000)
	if err := r.TryDeposit(1); !gbp.IsRefusal(err) {
		t.Errorf("a withdrawal does not restore the allowance: %v", err)
	}
}

// £10,000 on 1 January, within the allowance; 32 days to 1 February.
func TestGolden32Days(t *testing.T) {
	r := testkit.Open(t, isa.Version, "Liability:Savings:isa", day(2026, 1, 1))
	r.Deposit(1000000)
	r.Advance(32)
	testkit.Golden(t, "isa_32d", r.Export())
	testkit.RoundTrip(t, r.Export())
}

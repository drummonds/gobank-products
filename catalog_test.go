package gbp_test

import (
	"testing"
	"time"

	gbp "codeberg.org/hum3/gobank-products"
	"codeberg.org/hum3/gobank-products/testkit"
)

func TestEasyAccess_30Days(t *testing.T) {
	// Open, deposit £1000, advance 30 days.
	// Interest accrues to sub-account; main balance unchanged.
	testkit.NewScenario(t).
		WithProduct(gbp.EasyAccess()).
		OpenAccount("easy-access", "Liability:Savings:alice").
		Deposit("Liability:Savings:alice", 100000).
		AdvanceDays(32).
		AssertBalance("Liability:Savings:alice", -100000) // credit-normal, interest in Accrual
}

func TestFixedTerm_FullLifecycle(t *testing.T) {
	s := testkit.NewScenario(t).
		WithProduct(gbp.FixedTerm()).
		OpenAccountWithParams("fixed-term", "Liability:Savings:fixed", map[string]string{
			"maturity_date": "2026-02-01",
		}).
		Deposit("Liability:Savings:fixed", 100000)

	// Early withdrawal should fail.
	s.WithdrawExpectError("Liability:Savings:fixed", 50000)

	// Advance past maturity, then withdraw.
	s.AdvanceToDate(time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)).
		Withdraw("Liability:Savings:fixed", 50000).
		AssertBalance("Liability:Savings:fixed", -50000) // credit-normal, interest in Accrual
}

func TestISA_AllowanceEnforcement(t *testing.T) {
	s := testkit.NewScenario(t).
		WithProduct(gbp.ISA()).
		OpenAccount("isa", "Liability:Savings:isa")

	// Deposit within allowance (£20,000 = 2000000 pence).
	s.Deposit("Liability:Savings:isa", 1500000) // £15,000

	// Deposit up to the limit.
	s.Deposit("Liability:Savings:isa", 500000) // £5,000 — total now £20,000

	// Any further deposit should exceed allowance.
	sim := s.Sim()
	accountID := s.AccountID("Liability:Savings:isa")
	err := sim.Deposit(accountID, 100, "Asset:Cash", "")
	if err == nil {
		t.Fatal("expected ISA allowance error")
	}
}

func TestPersonalLoan_InterestCharges(t *testing.T) {
	// Lending: interest accrues to sub-account; main balance unchanged.
	testkit.NewScenario(t).
		WithProduct(gbp.PersonalLoan()).
		OpenAccountWithParams("personal-loan", "Asset:Loans:alice", map[string]string{
			"annual_rate": "0.0365",
		}).
		Deposit("Asset:Loans:alice", 100000). // £1000 loan disbursement
		AdvanceDays(32).
		AssertBalance("Asset:Loans:alice", 100000) // interest in Accrual sub-account
}

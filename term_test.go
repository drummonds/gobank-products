package gbp_test

import (
	"testing"
	"time"

	gbp "codeberg.org/hum3/gobank-products"
	"codeberg.org/hum3/gobank-products/testkit"
)

func TestTermLock_BlocksEarlyWithdrawal(t *testing.T) {
	testkit.NewScenario(t).
		WithProduct(gbp.FixedTerm()).
		OpenAccountWithParams("fixed-term", "Liability:Savings:fixed", map[string]string{
			"maturity_date": "2028-01-01",
		}).
		Deposit("Liability:Savings:fixed", 100000).
		AdvanceDays(30).
		WithdrawExpectError("Liability:Savings:fixed", 50000)
}

func TestTermLock_AllowsWithdrawalAfterMaturity(t *testing.T) {
	s := testkit.NewScenario(t).
		WithProduct(gbp.FixedTerm()).
		OpenAccountWithParams("fixed-term", "Liability:Savings:fixed", map[string]string{
			"maturity_date": "2026-02-01",
		}).
		Deposit("Liability:Savings:fixed", 100000)

	// Advance past maturity.
	s.AdvanceToDate(time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)).
		Withdraw("Liability:Savings:fixed", 50000).
		AssertBalanceRange("Liability:Savings:fixed", 50000, 51000) // balance includes accrued interest
}

package gbp_test

import (
	"testing"

	gbp "codeberg.org/hum3/gobank-products"
	"codeberg.org/hum3/gobank-products/testkit"
)

func TestInterestAccrual_SingleDay(t *testing.T) {
	// 3.65% annual on £1000 = 10p/day.
	testkit.NewScenario(t).
		WithProduct(&gbp.Product{
			ID:     "test-savings",
			Name:   "Test Savings",
			Family: gbp.FamilySavings,
			Features: []gbp.Feature{
				gbp.StatusLifecycle{},
				gbp.DepositAcceptance{},
				gbp.InterestAccrual{},
			},
			Defaults: map[string]string{"annual_rate": "0.0365"},
		}).
		OpenAccount("test-savings", "Liability:Savings:alice").
		Deposit("Liability:Savings:alice", 100000). // £1000.00
		AdvanceDays(1).
		AssertBalance("Liability:Savings:alice", 100010) // £1000.10
}

func TestInterestAccrual_TenDays(t *testing.T) {
	testkit.NewScenario(t).
		WithProduct(&gbp.Product{
			ID:     "test-savings",
			Name:   "Test Savings",
			Family: gbp.FamilySavings,
			Features: []gbp.Feature{
				gbp.StatusLifecycle{},
				gbp.DepositAcceptance{},
				gbp.InterestAccrual{},
			},
			Defaults: map[string]string{"annual_rate": "0.0365"},
		}).
		OpenAccount("test-savings", "Liability:Savings:alice").
		Deposit("Liability:Savings:alice", 100000).
		AdvanceDays(10).
		AssertBalanceRange("Liability:Savings:alice", 100090, 100110)
}

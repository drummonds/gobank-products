package gbp_test

import (
	"testing"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"

	gbp "git.bytestone.uk/hum3/gobank-products"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func savingsProduct() *gbp.Product {
	return &gbp.Product{
		ID:     "test-savings",
		Name:   "Test Savings",
		Family: gbp.FamilySavings,
		Features: []gbp.Feature{
			gbp.StatusLifecycle{},
			gbp.DepositAcceptance{},
			gbp.InterestAccrual{},
		},
		Defaults: map[string]string{"annual_rate": "0.0365"},
	}
}

// A bank restarting over a stored ledger brings its accounts back by
// adoption: the ledger account already exists, so nothing is created and
// no AccountOpened event fires, but from then on the engine manages the
// account exactly as one it opened itself.
func TestAdoptAccount_ResumesAccrualOnExistingLedgerAccount(t *testing.T) {
	ledger := testkit.NewTestLedger(t)
	acct, err := ledger.CreateAccount("Liability:Savings:alice", "GBP", -2, 0)
	if err != nil {
		t.Fatal(err)
	}
	cash, err := ledger.CreateAccount("Asset:Cash", "GBP", -2, 0)
	if err != nil {
		t.Fatal(err)
	}
	opened := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := ledger.RecordMovement(cash.ID, acct.ID, 100000, luca.CodeBookTransfer, opened, "deposit"); err != nil {
		t.Fatal(err)
	}

	// A new process: a fresh engine on a later day over the same ledger.
	clock := gbp.NewSimClock(opened.AddDate(0, 0, 10))
	sim, err := gbp.NewSimulation(ledger, clock)
	if err != nil {
		t.Fatal(err)
	}
	sim.RegisterProduct(savingsProduct())

	ma, err := sim.AdoptAccount(gbp.Adoption{
		Account:   acct,
		ProductID: "test-savings",
		Status:    gbp.StatusActive,
		OpenedAt:  opened,
		Balance:   100000,
		Params:    map[string]string{"annual_rate": "0.0730"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := sim.GetManagedAccount(acct.ID); !ok || got != ma {
		t.Fatalf("adopted account is not managed under its ledger ID")
	}
	if ma.Account != acct || ma.ProductID != "test-savings" || ma.Family != gbp.FamilySavings {
		t.Errorf("adopted account = %+v; want the ledger account on the product", ma)
	}
	if ma.CachedBalance != 100000 || ma.RateBps != 730 || ma.Status != gbp.StatusActive || !ma.OpenedAt.Equal(opened) {
		t.Errorf("adopted account state = balance %d, rate %d bps, status %v, opened %v", ma.CachedBalance, ma.RateBps, ma.Status, ma.OpenedAt)
	}
	if accts, _ := ledger.ListAccounts(""); len(accts) != 2 {
		t.Errorf("adoption must not create ledger accounts; ledger has %d", len(accts))
	}

	// One day's accrual on the adopted balance at the adopted rate:
	// 100000 pence x 730 bps, in numerator units.
	if _, err := sim.AdvanceToDate(clock.Now()); err != nil {
		t.Fatal(err)
	}
	if ma.AccruedNumerator != 100000*730 {
		t.Errorf("accrued numerator after one day = %d; want %d", ma.AccruedNumerator, 100000*730)
	}
}

// The rate falls back to the product default, as OpenAccount's does.
func TestAdoptAccount_RateFromProductDefault(t *testing.T) {
	ledger := testkit.NewTestLedger(t)
	acct, err := ledger.CreateAccount("Liability:Savings:bob", "GBP", -2, 0)
	if err != nil {
		t.Fatal(err)
	}
	sim, _ := gbp.NewSimulation(ledger, gbp.NewSimClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	sim.RegisterProduct(savingsProduct())
	ma, err := sim.AdoptAccount(gbp.Adoption{Account: acct, ProductID: "test-savings", Status: gbp.StatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if ma.RateBps != 365 {
		t.Errorf("rate = %d bps; want the product default 365", ma.RateBps)
	}
	if _, err := sim.AdoptAccount(gbp.Adoption{Account: acct, ProductID: "no-such-product"}); err == nil {
		t.Errorf("adopting onto an unregistered product should fail")
	}
}

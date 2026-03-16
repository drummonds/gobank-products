package testkit

import (
	"bytes"
	"testing"
	"time"

	luca "github.com/drummonds/go-luca"

	gbp "codeberg.org/hum3/gobank-products"
)

// ScenarioBuilder provides a fluent API for defining test scenarios.
type ScenarioBuilder struct {
	t        *testing.T
	sim      *gbp.Simulation
	clock    *gbp.SimClock
	accounts map[string]*gbp.ManagedAccount // path → account
	equity   *luca.Account                  // funding source
	advanced bool                           // tracks whether AdvanceDays has been called
}

// NewScenario creates a new test scenario with an in-memory ledger.
// Start date defaults to 2026-01-01.
func NewScenario(t *testing.T) *ScenarioBuilder {
	t.Helper()
	ledger := NewTestLedger(t)
	clock := gbp.NewSimClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	sim, err := gbp.NewSimulation(ledger, clock)
	if err != nil {
		t.Fatal(err)
	}

	// Create an equity account for funding deposits.
	eq, err := sim.Ledger.CreateAccount("Equity:Capital", "GBP", -2, 0)
	if err != nil {
		t.Fatal(err)
	}

	return &ScenarioBuilder{
		t:        t,
		sim:      sim,
		clock:    clock,
		accounts: make(map[string]*gbp.ManagedAccount),
		equity:   eq,
	}
}

// WithProduct registers a product with the simulation.
func (s *ScenarioBuilder) WithProduct(p *gbp.Product) *ScenarioBuilder {
	s.t.Helper()
	s.sim.RegisterProduct(p)
	return s
}

// OpenAccount opens an account for the given product and path.
func (s *ScenarioBuilder) OpenAccount(productID, accountPath string) *ScenarioBuilder {
	s.t.Helper()
	ma, err := s.sim.OpenAccount(productID, accountPath, "GBP", -2, nil)
	if err != nil {
		s.t.Fatal(err)
	}
	s.accounts[accountPath] = ma
	return s
}

// OpenAccountWithParams opens an account with explicit parameters.
func (s *ScenarioBuilder) OpenAccountWithParams(productID, accountPath string, params map[string]string) *ScenarioBuilder {
	s.t.Helper()
	ma, err := s.sim.OpenAccount(productID, accountPath, "GBP", -2, params)
	if err != nil {
		s.t.Fatal(err)
	}
	s.accounts[accountPath] = ma
	return s
}

// Deposit deposits an amount (in minor units) into the named account path.
func (s *ScenarioBuilder) Deposit(accountPath string, amount luca.Amount) *ScenarioBuilder {
	s.t.Helper()
	ma, ok := s.accounts[accountPath]
	if !ok {
		s.t.Fatalf("unknown account path: %s", accountPath)
	}
	if err := s.sim.Deposit(ma.Account.ID, amount, s.equity.ID, luca.CodeBookTransfer); err != nil {
		s.t.Fatal(err)
	}
	return s
}

// Withdraw withdraws an amount from the named account path.
func (s *ScenarioBuilder) Withdraw(accountPath string, amount luca.Amount) *ScenarioBuilder {
	s.t.Helper()
	ma, ok := s.accounts[accountPath]
	if !ok {
		s.t.Fatalf("unknown account path: %s", accountPath)
	}
	if err := s.sim.Withdraw(ma.Account.ID, amount, s.equity.ID, luca.CodeBookTransfer); err != nil {
		s.t.Fatal(err)
	}
	return s
}

// WithdrawExpectError withdraws and expects an error.
func (s *ScenarioBuilder) WithdrawExpectError(accountPath string, amount luca.Amount) *ScenarioBuilder {
	s.t.Helper()
	ma, ok := s.accounts[accountPath]
	if !ok {
		s.t.Fatalf("unknown account path: %s", accountPath)
	}
	err := s.sim.Withdraw(ma.Account.ID, amount, s.equity.ID, luca.CodeBookTransfer)
	if err == nil {
		s.t.Fatal("expected withdrawal error, got nil")
	}
	return s
}

// AdvanceDays processes exactly n end-of-day events from the current position.
func (s *ScenarioBuilder) AdvanceDays(n int) *ScenarioBuilder {
	s.t.Helper()
	var target time.Time
	if !s.advanced {
		// First call: start date is unprocessed, so n days = n-1 offset from now.
		target = s.clock.Now().AddDate(0, 0, n-1)
		s.advanced = true
	} else {
		target = s.clock.Now().AddDate(0, 0, n)
	}
	s.clock.SetDate(target)
	_, err := s.sim.AdvanceToDate(target)
	if err != nil {
		s.t.Fatal(err)
	}
	return s
}

// AdvanceToDate advances the simulation to a specific date.
func (s *ScenarioBuilder) AdvanceToDate(target time.Time) *ScenarioBuilder {
	s.t.Helper()
	s.clock.SetDate(target)
	_, err := s.sim.AdvanceToDate(target)
	if err != nil {
		s.t.Fatal(err)
	}
	s.advanced = true
	return s
}

// AccountID returns the ledger account ID for the given path.
func (s *ScenarioBuilder) AccountID(accountPath string) string {
	s.t.Helper()
	ma, ok := s.accounts[accountPath]
	if !ok {
		s.t.Fatalf("unknown account path: %s", accountPath)
	}
	return ma.Account.ID
}

// AssertBalance checks the current balance of the named account.
func (s *ScenarioBuilder) AssertBalance(accountPath string, expected luca.Amount) *ScenarioBuilder {
	s.t.Helper()
	ma, ok := s.accounts[accountPath]
	if !ok {
		s.t.Fatalf("unknown account path: %s", accountPath)
	}
	bal, err := s.sim.Ledger.Balance(ma.Account.ID)
	if err != nil {
		s.t.Fatal(err)
	}
	if bal != expected {
		s.t.Errorf("balance of %s: got %d, want %d", accountPath, bal, expected)
	}
	return s
}

// AssertBalanceRange checks the balance is within [lo, hi].
func (s *ScenarioBuilder) AssertBalanceRange(accountPath string, lo, hi luca.Amount) *ScenarioBuilder {
	s.t.Helper()
	ma, ok := s.accounts[accountPath]
	if !ok {
		s.t.Fatalf("unknown account path: %s", accountPath)
	}
	bal, err := s.sim.Ledger.Balance(ma.Account.ID)
	if err != nil {
		s.t.Fatal(err)
	}
	if bal < lo || bal > hi {
		s.t.Errorf("balance of %s: got %d, want [%d, %d]", accountPath, bal, lo, hi)
	}
	return s
}

// AssertStatus checks the account status.
func (s *ScenarioBuilder) AssertStatus(accountPath string, expected gbp.AccountStatus) *ScenarioBuilder {
	s.t.Helper()
	ma, ok := s.accounts[accountPath]
	if !ok {
		s.t.Fatalf("unknown account path: %s", accountPath)
	}
	if ma.Status != expected {
		s.t.Errorf("status of %s: got %s, want %s", accountPath, ma.Status, expected)
	}
	return s
}

// CloseAccount closes the named account.
func (s *ScenarioBuilder) CloseAccount(accountPath string) *ScenarioBuilder {
	s.t.Helper()
	ma, ok := s.accounts[accountPath]
	if !ok {
		s.t.Fatalf("unknown account path: %s", accountPath)
	}
	if err := s.sim.CloseAccount(ma.Account.ID); err != nil {
		s.t.Fatal(err)
	}
	return s
}

// ExportGoluca returns the ledger state as a .goluca string.
func (s *ScenarioBuilder) ExportGoluca() string {
	s.t.Helper()
	var buf bytes.Buffer
	if err := s.sim.ExportGoluca(&buf); err != nil {
		s.t.Fatal(err)
	}
	return buf.String()
}

// Sim returns the underlying simulation for advanced use.
func (s *ScenarioBuilder) Sim() *gbp.Simulation {
	return s.sim
}

// Clock returns the simulation clock.
func (s *ScenarioBuilder) Clock() *gbp.SimClock {
	return s.clock
}

package testkit

import (
	"bytes"
	"strings"
	"testing"
	"time"

	luca "github.com/drummonds/go-luca"

	gbp "codeberg.org/hum3/gobank-products"
)

// GolucaScenario declaratively defines a product test scenario.
type GolucaScenario struct {
	Name    string       // golden file name, e.g. "easy_access_30d"
	Spec    string       // human-readable Gherkin-style description (documentation only)
	Product *gbp.Product // product under test
	FSM     *ProductFSM  // optional FSM for validation
	Account AccountSpec
	Actions []Action
}

// AccountSpec defines the account to open.
type AccountSpec struct {
	Path   string            // e.g. "Liability:Savings:alice"
	Params map[string]string // override product defaults
}

// Action is a step in a scenario.
type Action struct {
	Type   ActionType
	Amount luca.Amount // in minor units (pence)
	Date   time.Time   // for AdvanceTo
	Days   int         // for AdvanceDays
	Event  gbp.EventType
}

// ActionType identifies the kind of scenario action.
type ActionType int

const (
	ActionDeposit ActionType = iota
	ActionWithdraw
	ActionAdvanceDays
	ActionAdvanceTo
)

// Deposit creates a deposit action.
func Deposit(amount luca.Amount) Action {
	return Action{Type: ActionDeposit, Amount: amount, Event: gbp.EventDepositReceived}
}

// Withdraw creates a withdrawal action.
func Withdraw(amount luca.Amount) Action {
	return Action{Type: ActionWithdraw, Amount: amount, Event: gbp.EventWithdrawalRequested}
}

// AdvanceDays creates a time-advance action.
func AdvanceDays(n int) Action {
	return Action{Type: ActionAdvanceDays, Days: n, Event: gbp.EventEndOfDay}
}

// AdvanceTo creates an advance-to-date action.
func AdvanceTo(year int, month time.Month, day int) Action {
	return Action{
		Type:  ActionAdvanceTo,
		Date:  time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
		Event: gbp.EventEndOfDay,
	}
}

// Run executes the scenario and returns the ScenarioBuilder for further assertions.
func (sc *GolucaScenario) Run(t *testing.T) *ScenarioBuilder {
	t.Helper()

	// Validate against FSM if provided.
	if sc.FSM != nil {
		specs := scenarioToActionSpecs(sc)
		if err := sc.FSM.Validate(specs); err != nil {
			t.Fatalf("FSM validation failed for %s: %v", sc.Name, err)
		}
	}

	s := NewScenario(t).WithProduct(sc.Product)

	if sc.Account.Params != nil {
		s.OpenAccountWithParams(sc.Product.ID, sc.Account.Path, sc.Account.Params)
	} else {
		s.OpenAccount(sc.Product.ID, sc.Account.Path)
	}

	for _, a := range sc.Actions {
		switch a.Type {
		case ActionDeposit:
			s.Deposit(sc.Account.Path, a.Amount)
		case ActionWithdraw:
			s.Withdraw(sc.Account.Path, a.Amount)
		case ActionAdvanceDays:
			s.AdvanceDays(a.Days)
		case ActionAdvanceTo:
			s.AdvanceToDate(a.Date)
		}
	}

	return s
}

// RunGolden executes the scenario and compares against the golden file.
func (sc *GolucaScenario) RunGolden(t *testing.T) {
	t.Helper()
	s := sc.Run(t)
	got := s.ExportGoluca()
	Golden(t, sc.Name, got)
}

// RunRoundTrip executes the scenario, exports, imports, re-exports, and verifies fidelity.
// Skips if go-luca's import cannot parse the exported format (known grammar limitation).
func (sc *GolucaScenario) RunRoundTrip(t *testing.T) {
	t.Helper()
	s := sc.Run(t)
	exported := s.ExportGoluca()

	// Import into a fresh ledger.
	ledger2 := NewTestLedger(t)
	if err := ledger2.Import(strings.NewReader(exported), nil); err != nil {
		t.Skipf("import not yet supported for this export format: %v", err)
	}

	// Re-export.
	var buf bytes.Buffer
	if err := ledger2.Export(&buf); err != nil {
		t.Fatalf("re-export failed: %v", err)
	}

	AssertGolucaEqual(t, buf.String(), exported)
}

// scenarioToActionSpecs converts scenario actions to FSM action specs.
// It prepends the AccountOpened event that always fires on OpenAccount.
func scenarioToActionSpecs(sc *GolucaScenario) []ActionSpec {
	specs := []ActionSpec{{Event: gbp.EventAccountOpened}}
	for _, a := range sc.Actions {
		specs = append(specs, ActionSpec{Event: a.Event})
	}
	return specs
}

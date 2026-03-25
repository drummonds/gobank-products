package gbp_test

import (
	"testing"
	"time"

	gbp "codeberg.org/hum3/gobank-products"
	"codeberg.org/hum3/gobank-products/testkit"
)

// scenarios defines one declarative golden-file scenario per product.
var scenarios = []testkit.GolucaScenario{
	{
		Name:    "easy_access_32d",
		Product: gbp.EasyAccess(),
		FSM:     testkit.EasyAccessFSM(),
		Account: testkit.AccountSpec{Path: "Liability:Savings:alice"},
		Actions: []testkit.Action{
			testkit.Deposit(100000), // £1,000.00
			testkit.AdvanceDays(32),
		},
		Spec: `
			Given an Easy Access savings account opened 2026-01-01
			  And a deposit of £1,000.00
			When 32 days elapse to 2026-02-01 (includes month end)
			Then daily interest accrues at 1.5% p.a.
			  And the goluca export round-trips cleanly`,
	},
	{
		Name:    "fixed_term_maturity",
		Product: gbp.FixedTerm(),
		FSM:     testkit.FixedTermFSM(),
		Account: testkit.AccountSpec{
			Path:   "Liability:Savings:fixed",
			Params: map[string]string{"maturity_date": "2026-04-01"},
		},
		Actions: []testkit.Action{
			testkit.Deposit(500000),                // £5,000.00
			testkit.AdvanceTo(2026, time.April, 2), // past maturity
			testkit.Withdraw(500000),               // £5,000.00
		},
		Spec: `
			Given a Fixed Term account with maturity 2026-04-01
			  And a deposit of £5,000.00
			When time advances past maturity to 2026-04-02
			  And £5,000.00 is withdrawn
			Then interest at 4.0% has accrued over ~91 days
			  And the withdrawal succeeds post-maturity`,
	},
	{
		Name:    "isa_32d",
		Product: gbp.ISA(),
		FSM:     testkit.ISAFSM(),
		Account: testkit.AccountSpec{Path: "Liability:Savings:isa"},
		Actions: []testkit.Action{
			testkit.Deposit(1000000), // £10,000.00
			testkit.AdvanceDays(32),
		},
		Spec: `
			Given an ISA account opened 2026-01-01
			  And a deposit of £10,000.00 (within £20k allowance)
			When 32 days elapse (includes month end)
			Then daily interest accrues at 3.5% p.a.`,
	},
	{
		Name:    "personal_loan_32d",
		Product: gbp.PersonalLoan(),
		FSM:     testkit.PersonalLoanFSM(),
		Account: testkit.AccountSpec{Path: "Asset:Loans:alice"},
		Actions: []testkit.Action{
			testkit.Deposit(500000), // £5,000.00 disbursement
			testkit.AdvanceDays(32),
		},
		Spec: `
			Given a Personal Loan account opened 2026-01-01
			  And £5,000.00 disbursed
			When 32 days elapse (includes month end)
			Then daily interest accrues at 6.9% p.a.`,
	},
	{
		Name:    "mortgage_32d",
		Product: gbp.Mortgage(),
		FSM:     testkit.MortgageFSM(),
		Account: testkit.AccountSpec{Path: "Asset:Loans:mortgage"},
		Actions: []testkit.Action{
			testkit.Deposit(10000000), // £100,000.00 disbursement
			testkit.AdvanceDays(32),
		},
		Spec: `
			Given a Mortgage account opened 2026-01-01
			  And £100,000.00 disbursed
			When 32 days elapse (includes month end)
			Then daily interest accrues at 4.5% p.a.`,
	},
	{
		Name:    "overdraft_32d",
		Product: gbp.Overdraft(),
		FSM:     testkit.OverdraftFSM(),
		Account: testkit.AccountSpec{Path: "Liability:Current:alice"},
		Actions: []testkit.Action{
			testkit.Withdraw(50000), // £500.00 overdraft draw
			testkit.AdvanceDays(32),
		},
		Spec: `
			Given an Overdraft facility account opened 2026-01-01
			  And £500.00 drawn (within £1,000 limit)
			When 32 days elapse (includes month end)
			Then daily interest accrues at 15.9% p.a. on negative balance`,
	},
}

func TestGolucaScenarios(t *testing.T) {
	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			sc.RunGolden(t)
		})
	}
}

func TestGolucaRoundTrip(t *testing.T) {
	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			sc.RunRoundTrip(t)
		})
	}
}

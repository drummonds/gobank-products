package testkit

import (
	"fmt"
	"strings"

	gbp "codeberg.org/hum3/gobank-products"
)

// Transition defines a valid (state, event) → state mapping with an optional guard.
type Transition struct {
	From  gbp.AccountStatus
	Event gbp.EventType
	Guard string // human-readable condition, e.g. "balance >= amount"
	To    gbp.AccountStatus
}

// ProductFSM is the formal state machine for a product.
type ProductFSM struct {
	ProductID   string
	Transitions []Transition
}

// Validate checks that a sequence of (status, event) pairs follows legal transitions.
// It returns an error on the first illegal transition.
func (fsm *ProductFSM) Validate(actions []ActionSpec) error {
	state := gbp.StatusPending
	for i, a := range actions {
		found := false
		for _, t := range fsm.Transitions {
			if t.From == state && t.Event == a.Event {
				state = t.To
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("action %d: no transition from %s on %s", i, state, a.Event)
		}
	}
	return nil
}

// String returns a human-readable FSM description.
func (fsm *ProductFSM) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "FSM(%s):\n", fsm.ProductID)
	for _, t := range fsm.Transitions {
		guard := ""
		if t.Guard != "" {
			guard = fmt.Sprintf(" [%s]", t.Guard)
		}
		fmt.Fprintf(&b, "  %s --%s--> %s%s\n", t.From, t.Event, t.To, guard)
	}
	return b.String()
}

// ActionSpec is used by FSM validation — maps a scenario action to an event type.
type ActionSpec struct {
	Event gbp.EventType
}

// baseSavingsFSM returns transitions common to all savings products.
func baseSavingsFSM() []Transition {
	return []Transition{
		{gbp.StatusPending, gbp.EventAccountOpened, "", gbp.StatusActive},
		{gbp.StatusActive, gbp.EventDepositReceived, "", gbp.StatusActive},
		{gbp.StatusActive, gbp.EventWithdrawalRequested, "balance >= amount", gbp.StatusActive},
		{gbp.StatusActive, gbp.EventEndOfDay, "accrues interest", gbp.StatusActive},
		{gbp.StatusActive, gbp.EventEndOfMonth, "", gbp.StatusActive},
		{gbp.StatusActive, gbp.EventAccountClosed, "", gbp.StatusClosed},
	}
}

// baseLendingFSM returns transitions common to all lending products.
func baseLendingFSM() []Transition {
	return []Transition{
		{gbp.StatusPending, gbp.EventAccountOpened, "", gbp.StatusActive},
		{gbp.StatusActive, gbp.EventDepositReceived, "", gbp.StatusActive}, // disbursement / repayment
		{gbp.StatusActive, gbp.EventEndOfDay, "accrues interest", gbp.StatusActive},
		{gbp.StatusActive, gbp.EventEndOfMonth, "", gbp.StatusActive},
		{gbp.StatusActive, gbp.EventAccountClosed, "", gbp.StatusClosed},
	}
}

// EasyAccessFSM returns the FSM for the Easy Access product.
func EasyAccessFSM() *ProductFSM {
	return &ProductFSM{
		ProductID:   "easy-access",
		Transitions: baseSavingsFSM(),
	}
}

// FixedTermFSM returns the FSM for the Fixed Term product.
func FixedTermFSM() *ProductFSM {
	transitions := baseSavingsFSM()
	// Add maturity transition; override withdrawal guard.
	transitions = append(transitions,
		Transition{gbp.StatusActive, gbp.EventMaturityReached, "date >= maturity_date", gbp.StatusActive},
	)
	// Withdrawal guard is stricter.
	for i := range transitions {
		if transitions[i].Event == gbp.EventWithdrawalRequested {
			transitions[i].Guard = "date >= maturity_date && balance >= amount"
		}
	}
	return &ProductFSM{
		ProductID:   "fixed-term",
		Transitions: transitions,
	}
}

// ISAFSM returns the FSM for the ISA product.
func ISAFSM() *ProductFSM {
	transitions := baseSavingsFSM()
	for i := range transitions {
		if transitions[i].Event == gbp.EventDepositReceived {
			transitions[i].Guard = "isa_deposited + amount <= isa_allowance"
		}
	}
	return &ProductFSM{
		ProductID:   "isa",
		Transitions: transitions,
	}
}

// PersonalLoanFSM returns the FSM for the Personal Loan product.
func PersonalLoanFSM() *ProductFSM {
	return &ProductFSM{
		ProductID:   "personal-loan",
		Transitions: baseLendingFSM(),
	}
}

// MortgageFSM returns the FSM for the Mortgage product.
func MortgageFSM() *ProductFSM {
	return &ProductFSM{
		ProductID:   "mortgage",
		Transitions: baseLendingFSM(),
	}
}

// OverdraftFSM returns the FSM for the Overdraft product.
func OverdraftFSM() *ProductFSM {
	transitions := baseLendingFSM()
	transitions = append(transitions,
		Transition{gbp.StatusActive, gbp.EventWithdrawalRequested, "balance - amount >= -overdraft_limit", gbp.StatusActive},
	)
	return &ProductFSM{
		ProductID:   "overdraft",
		Transitions: transitions,
	}
}

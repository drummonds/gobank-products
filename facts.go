package gbp

import (
	"errors"
	"fmt"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
)

// Facts is what a rule may read about the account an event is for. The
// runner implements it over the ledger and the bank's settings; a test
// implements it in memory. What a rule reads through it is what the bank
// records against the postings the rule calls for.
type Facts interface {
	ParameterReader
	Account() string                              // the ledger account's ID
	Day() time.Time                               // the day the event is for, midnight UTC
	Position(day time.Time) (luca.Position, bool) // the stored end-of-day position; false when none
	Balance() luca.Amount                         // the balance as it stands now
}

// Movement is a posting to the account as a rule sees it: Delta is the
// change to the account's balance, signed as the balance is, so a deposit
// to a savings account (credit-normal, negative) has a negative Delta and
// a withdrawal a positive one.
type Movement struct {
	Delta       luca.Amount
	Code        string
	Description string
	ValueTime   time.Time
}

// Change is a parameter setting that has become effective for the account.
type Change struct {
	Key       string
	From, To  Param
	Effective time.Time
}

// Setting is an account-scoped parameter value a rule asks the runner to
// write, effective from a day.
type Setting struct {
	Key       string
	Text      string
	Effective time.Time
}

// Posting is a ledger movement a rule calls for: from the counterparty to
// the account, signed as the account's balance is, so a credit-normal
// savings balance receives a negative amount.
type Posting struct {
	Counterparty string // a P&L account path: Expense:Interest or Income:Interest
	Amount       luca.Amount
	Code         string
	ValueTime    time.Time
	Description  string
}

// Intents is what a rule asks the runner to do, in order: write the
// settings, make the postings, then project the positions.
type Intents struct {
	Settings  []Setting
	Postings  []Posting
	Positions []luca.Position
}

// Refusal is a rule's no to a movement: a business reason, not a fault.
type Refusal struct {
	Reason string
}

func (r *Refusal) Error() string { return r.Reason }

// Refuse is a refusal with its reason.
func Refuse(format string, args ...any) error {
	return &Refusal{Reason: fmt.Sprintf(format, args...)}
}

// IsRefusal reports whether err is a rule's refusal.
func IsRefusal(err error) bool {
	var r *Refusal
	return errors.As(err, &r)
}

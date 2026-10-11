// Package overdraft is the Overdraft facility, version 1: a current
// account may be drawn down to an arranged limit, with interest accrued
// daily on the overdrawn balance at a fixed rate and applied monthly.
package overdraft

import gbp "git.bytestone.uk/hum3/gobank-products"

// Version is overdraft v1.
var Version gbp.Version = version{gbp.Interest{Family: gbp.FamilyLending, RateKey: "rate_bps", CycleKey: "cycle"}}

type version struct{ gbp.Interest }

func (version) Product() string           { return "overdraft" }
func (version) Version() int              { return 1 }
func (version) Name() string              { return "Overdraft" }
func (version) Family() gbp.ProductFamily { return gbp.FamilyLending }

func (version) Parameters() []gbp.Declaration {
	return []gbp.Declaration{
		{Key: "rate_bps", Scope: gbp.ScopeVersion, Kind: gbp.KindBps, Published: "1590", Description: "annual rate on the overdrawn balance"},
		{Key: "cycle", Scope: gbp.ScopeVersion, Kind: gbp.KindCycle, Published: "monthly", Description: "when accrued interest is applied"},
		{Key: "limit", Scope: gbp.ScopeVersion, Kind: gbp.KindMoney, Published: "100000", Description: "the arranged limit, in pence"},
	}
}

// PrePosting refuses a movement that would draw the account past the
// limit. The ledger is the bank's view (gobank ADR-0008): a current
// account in credit is a liability and negative, overdrawn a debit
// balance and positive, so the limit is the most the balance may rise to.
func (version) PrePosting(f gbp.Facts, m gbp.Movement) error {
	limit, err := f.Parameter("limit", f.Day())
	if err != nil {
		return err
	}
	if after := f.Balance() + m.Delta; after > limit.Money() {
		return gbp.Refuse("overdraft limit exceeded: balance %d drawn by %d would breach the limit of %d", f.Balance(), m.Delta, limit.Money())
	}
	return nil
}

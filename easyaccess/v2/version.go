// Package easyaccess is the Easy Access savings product, version 2: instant
// access, interest accrued daily on the closing balance and applied
// monthly, as v1, but at a rate that tracks the bank's savings rate at 15
// basis points below it, floored at zero. The bank's rate is the policy's
// (gobank ADR-0006, ADR-0007): a change to it is a change to the account's
// rate on the same day, with nothing set on the product. Adopted by
// gobank in story 1.8.9; immutable since.
package easyaccess

import gbp "git.bytestone.uk/hum3/gobank-products"

// Version is easy-access v2.
var Version gbp.Version = version{gbp.Interest{Family: gbp.FamilySavings, RateKey: "rate_bps", CycleKey: "cycle"}}

type version struct{ gbp.Interest }

func (version) Product() string           { return "easy-access" }
func (version) Version() int              { return 2 }
func (version) Name() string              { return "Easy Access Tracker" }
func (version) Family() gbp.ProductFamily { return gbp.FamilySavings }

func (version) Parameters() []gbp.Declaration {
	floor := int64(0)
	return []gbp.Declaration{
		{Key: gbp.SavingsRateKey, Scope: gbp.ScopeBank, Kind: gbp.KindBps, Description: "the bank's savings rate, from its policy"},
		{Key: "rate_bps", Scope: gbp.ScopeVersion, Kind: gbp.KindBps, Derived: &gbp.Derivation{Source: gbp.SavingsRateKey, SpreadBps: -15, Floor: &floor}, Description: "annual gross rate: the bank's savings rate less 15 bps, floor 0"},
		{Key: "cycle", Scope: gbp.ScopeVersion, Kind: gbp.KindCycle, Published: "monthly", Description: "when accrued interest is applied"},
	}
}

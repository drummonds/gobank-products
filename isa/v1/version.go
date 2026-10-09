// Package isa is the ISA savings product, version 1: deposits up to the
// annual allowance, interest accrued daily at a fixed rate and applied
// monthly. The allowance is counted from open; it does not reset with the
// tax year, as the product the bank ran before did not.
package isa

import (
	"strconv"

	gbp "git.bytestone.uk/hum3/gobank-products"
)

// Version is isa v1.
var Version gbp.Version = version{gbp.Interest{Family: gbp.FamilySavings, RateKey: "rate_bps", CycleKey: "cycle"}}

type version struct{ gbp.Interest }

func (version) Product() string           { return "isa" }
func (version) Version() int              { return 1 }
func (version) Name() string              { return "ISA" }
func (version) Family() gbp.ProductFamily { return gbp.FamilySavings }

func (version) Parameters() []gbp.Declaration {
	return []gbp.Declaration{
		{Key: "rate_bps", Scope: gbp.ScopeVersion, Kind: gbp.KindBps, Published: "350", Description: "annual gross rate"},
		{Key: "cycle", Scope: gbp.ScopeVersion, Kind: gbp.KindCycle, Published: "monthly", Description: "when accrued interest is applied"},
		{Key: "allowance", Scope: gbp.ScopeVersion, Kind: gbp.KindMoney, Published: "2000000", Description: "the most that may be deposited, in pence"},
		{Key: "deposited", Scope: gbp.ScopeAccount, Kind: gbp.KindMoney, Description: "deposits so far, in pence"},
	}
}

// Open starts the deposit count.
func (version) Open(f gbp.Facts) (gbp.Intents, error) {
	return gbp.Intents{Settings: []gbp.Setting{{Key: "deposited", Text: "0", Effective: f.Day()}}}, nil
}

// PrePosting refuses a deposit that would take the count over the allowance.
func (version) PrePosting(f gbp.Facts, m gbp.Movement) error {
	if m.Delta >= 0 {
		return nil // a withdrawal
	}
	allowance, err := f.Parameter("allowance", f.Day())
	if err != nil {
		return err
	}
	deposited, err := f.Parameter("deposited", f.Day())
	if err != nil {
		return err
	}
	if deposited.Money()-m.Delta > allowance.Money() {
		return gbp.Refuse("ISA allowance exceeded: %d deposited of %d, %d remaining", deposited.Money(), allowance.Money(), allowance.Money()-deposited.Money())
	}
	return nil
}

// PostPosting counts a deposit, then runs the day rule.
func (v version) PostPosting(f gbp.Facts, m gbp.Movement) (gbp.Intents, error) {
	in, err := v.Interest.PostPosting(f, m)
	if err != nil || m.Delta >= 0 {
		return in, err
	}
	deposited, err := f.Parameter("deposited", f.Day())
	if err != nil {
		return gbp.Intents{}, err
	}
	in.Settings = append(in.Settings, gbp.Setting{Key: "deposited", Text: strconv.FormatInt(int64(deposited.Money()-m.Delta), 10), Effective: f.Day()})
	return in, nil
}

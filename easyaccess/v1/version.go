// Package easyaccess is the Easy Access savings product, version 1: instant
// access, interest accrued daily on the closing balance at a fixed rate
// and applied monthly. Adopted by gobank in stage 8; immutable since.
package easyaccess

import gbp "git.bytestone.uk/hum3/gobank-products"

// Version is easy-access v1.
var Version gbp.Version = version{gbp.Interest{Family: gbp.FamilySavings, RateKey: "rate_bps", CycleKey: "cycle"}}

type version struct{ gbp.Interest }

func (version) Product() string           { return "easy-access" }
func (version) Version() int              { return 1 }
func (version) Name() string              { return "Easy Access" }
func (version) Family() gbp.ProductFamily { return gbp.FamilySavings }

func (version) Parameters() []gbp.Declaration {
	return []gbp.Declaration{
		{Key: "rate_bps", Scope: gbp.ScopeVersion, Kind: gbp.KindBps, Published: "150", Description: "annual gross rate"},
		{Key: "cycle", Scope: gbp.ScopeVersion, Kind: gbp.KindCycle, Published: "monthly", Description: "when accrued interest is applied"},
	}
}

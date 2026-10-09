// Package mortgage is the Mortgage lending product, version 1: interest accrued
// daily on the outstanding balance at a fixed rate and applied monthly.
package mortgage

import gbp "git.bytestone.uk/hum3/gobank-products"

// Version is mortgage v1.
var Version gbp.Version = version{gbp.Interest{Family: gbp.FamilyLending, RateKey: "rate_bps", CycleKey: "cycle"}}

type version struct{ gbp.Interest }

func (version) Product() string           { return "mortgage" }
func (version) Version() int              { return 1 }
func (version) Name() string              { return "Mortgage" }
func (version) Family() gbp.ProductFamily { return gbp.FamilyLending }

func (version) Parameters() []gbp.Declaration {
	return []gbp.Declaration{
		{Key: "rate_bps", Scope: gbp.ScopeVersion, Kind: gbp.KindBps, Published: "450", Description: "annual rate"},
		{Key: "cycle", Scope: gbp.ScopeVersion, Kind: gbp.KindCycle, Published: "monthly", Description: "when accrued interest is applied"},
	}
}

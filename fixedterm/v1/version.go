// Package fixedterm is the Fixed Term savings product, version 1: a
// two-year term fixed at open, no withdrawals before maturity, interest
// accrued daily at a fixed rate and applied monthly.
package fixedterm

import gbp "git.bytestone.uk/hum3/gobank-products"

// Version is fixed-term v1.
var Version gbp.Version = version{gbp.Interest{Family: gbp.FamilySavings, RateKey: "rate_bps", CycleKey: "cycle"}}

type version struct{ gbp.Interest }

func (version) Product() string           { return "fixed-term" }
func (version) Version() int              { return 1 }
func (version) Name() string              { return "Fixed Term" }
func (version) Family() gbp.ProductFamily { return gbp.FamilySavings }

func (version) Parameters() []gbp.Declaration {
	return []gbp.Declaration{
		{Key: "rate_bps", Scope: gbp.ScopeVersion, Kind: gbp.KindBps, Published: "400", Description: "annual gross rate"},
		{Key: "cycle", Scope: gbp.ScopeVersion, Kind: gbp.KindCycle, Published: "monthly", Description: "when accrued interest is applied"},
		{Key: "term_months", Scope: gbp.ScopeVersion, Kind: gbp.KindInt, Published: "24", Description: "the term from the open day"},
		{Key: "maturity_day", Scope: gbp.ScopeAccount, Kind: gbp.KindDay, Description: "the day the term ends, set at open"},
	}
}

// Open fixes the maturity day: the open day plus the term.
func (v version) Open(f gbp.Facts) (gbp.Intents, error) {
	term, err := f.Parameter("term_months", f.Day())
	if err != nil {
		return gbp.Intents{}, err
	}
	maturity := f.Day().AddDate(0, int(term.Int()), 0)
	return gbp.Intents{Settings: []gbp.Setting{{Key: "maturity_day", Text: maturity.Format("2006-01-02"), Effective: f.Day()}}}, nil
}

// PrePosting refuses a withdrawal before maturity.
func (v version) PrePosting(f gbp.Facts, m gbp.Movement) error {
	if m.Delta <= 0 {
		return nil // a deposit
	}
	maturity, err := f.Parameter("maturity_day", f.Day())
	if err != nil {
		return err
	}
	if f.Day().Before(maturity.Day()) {
		return gbp.Refuse("withdrawal blocked: account matures on %s", maturity.Text)
	}
	return nil
}

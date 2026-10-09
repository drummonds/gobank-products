package testkit_test

import (
	"testing"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"

	gbp "git.bytestone.uk/hum3/gobank-products"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// tracker is a test version: a savings rate derived from the bank's base
// rate, and an account-scoped parameter written at open.
type tracker struct{ gbp.Interest }

func newTracker() tracker {
	return tracker{gbp.Interest{Family: gbp.FamilySavings, RateKey: "rate_bps", CycleKey: "cycle"}}
}

func (tracker) Product() string           { return "tracker" }
func (tracker) Version() int              { return 1 }
func (tracker) Name() string              { return "Tracker" }
func (tracker) Family() gbp.ProductFamily { return gbp.FamilySavings }
func (tracker) Parameters() []gbp.Declaration {
	floor := int64(0)
	return []gbp.Declaration{
		{Key: "boe.base_rate_bps", Scope: gbp.ScopeBank, Kind: gbp.KindBps},
		{Key: "rate_bps", Scope: gbp.ScopeVersion, Kind: gbp.KindBps, Derived: &gbp.Derivation{Source: "boe.base_rate_bps", SpreadBps: -15, Floor: &floor}},
		{Key: "cycle", Scope: gbp.ScopeVersion, Kind: gbp.KindCycle, Published: "monthly"},
		{Key: "bonus_bps", Scope: gbp.ScopeVersion, Kind: gbp.KindBps, Published: "10"},
		{Key: "opened_day", Scope: gbp.ScopeAccount, Kind: gbp.KindDay},
	}
}

func (t tracker) Open(f gbp.Facts) (gbp.Intents, error) {
	return gbp.Intents{Settings: []gbp.Setting{{Key: "opened_day", Text: f.Day().Format("2006-01-02"), Effective: f.Day()}}}, nil
}

func open(t *testing.T) *testkit.Runner {
	// Start-up is refused without the base rate, so it is set first.
	return testkit.OpenWith(t, newTracker(), "Liability:Savings:alice", day(2026, 1, 1), func(r *testkit.Runner) {
		r.SetBank("boe.base_rate_bps", "525", day(2025, 1, 1))
	})
}

func TestRunnerResolvesByScopeAndDerivation(t *testing.T) {
	r := open(t)
	cases := map[string]string{
		"boe.base_rate_bps": "525", // bank setting
		"rate_bps":          "510", // derived: base - 15
		"cycle":             "monthly",
		"bonus_bps":         "10",         // published
		"opened_day":        "2026-01-01", // written by Open
	}
	for key, want := range cases {
		p, err := r.Parameter(key)
		if err != nil || p.Text != want {
			t.Errorf("%s = %q, %v; want %q", key, p.Text, err, want)
		}
	}
	if _, err := r.Parameter("no-such"); err == nil {
		t.Error("undeclared parameter resolved")
	}
}

func TestRunnerSettingsAreEffectiveDated(t *testing.T) {
	r := open(t)
	r.SetVersion("bonus_bps", "20", day(2026, 2, 1)) // decided now, effective next month
	r.SetBank("boe.base_rate_bps", "500", day(2026, 1, 20))
	if p, _ := r.Parameter("bonus_bps"); p.Bps() != 10 {
		t.Errorf("January bonus = %d, want the published 10: February's setting is invisible", p.Bps())
	}
	r.Advance(20) // to 20 January
	if p, _ := r.Parameter("rate_bps"); p.Bps() != 485 {
		t.Errorf("rate on %s = %d, want 485 from the base rate cut", r.Day().Format("2006-01-02"), p.Bps())
	}
	r.AdvanceTo(day(2026, 2, 1))
	if p, _ := r.Parameter("bonus_bps"); p.Bps() != 20 {
		t.Errorf("February bonus = %d, want 20", p.Bps())
	}
}

func TestRunnerFloorsADerivedRate(t *testing.T) {
	r := open(t)
	r.SetBank("boe.base_rate_bps", "5", day(2026, 1, 1))
	if p, _ := r.Parameter("rate_bps"); p.Bps() != 0 {
		t.Errorf("rate = %d, want floored at 0", p.Bps())
	}
}

func TestRunnerCarriesOutIntentsAndLogsReads(t *testing.T) {
	r := open(t)
	r.Deposit(100000)
	if r.Balance() != -100000 {
		t.Errorf("balance after deposit = %d", r.Balance())
	}
	r.Advance(32) // to 1 February: January's interest applied by the Day of 1 February
	pos, ok := r.Position(day(2026, 1, 31))
	if !ok || pos.Balance != -100433 { // 31 days at 510 bps on £1,000 is 433p
		t.Errorf("31 Jan position = %+v, %v; want -100433", pos, ok)
	}
	if len(r.Reads) == 0 || r.Reads[len(r.Reads)-1].Key != "rate_bps" {
		t.Errorf("reads of the last Day = %+v, want the cycle then the rate", r.Reads)
	}
	if r.Balance() != -100433 {
		t.Errorf("balance = %d", r.Balance())
	}
	r.Command(gbp.CommandApplyInterest)
	if r.Balance() != -100447 { // one more day's 14p
		t.Errorf("balance after apply-interest = %d, want -100447", r.Balance())
	}
}

// refuser refuses every withdrawal: the runner surfaces a refusal, not a fault.
type refuser struct{ tracker }

func (refuser) PrePosting(_ gbp.Facts, m gbp.Movement) error {
	if m.Delta > 0 {
		return gbp.Refuse("no withdrawals")
	}
	return nil
}

func TestRunnerSurfacesRefusals(t *testing.T) {
	r := testkit.OpenWith(t, refuser{newTracker()}, "Liability:Savings:bob", day(2026, 1, 1), func(r *testkit.Runner) {
		r.SetBank("boe.base_rate_bps", "525", day(2025, 1, 1))
	})
	r.Deposit(1000)
	err := r.TryWithdraw(500)
	if !gbp.IsRefusal(err) || err.Error() != "no withdrawals" {
		t.Errorf("withdrawal: %v, want the refusal", err)
	}
	if r.Balance() != luca.Amount(-1000) {
		t.Errorf("a refused movement posted: balance %d", r.Balance())
	}
}

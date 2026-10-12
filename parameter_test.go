package gbp_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	gbp "git.bytestone.uk/hum3/gobank-products"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func TestDeclarationParsesByKind(t *testing.T) {
	cases := []struct {
		kind gbp.Kind
		text string
		ok   bool
	}{
		{gbp.KindBps, "150", true}, {gbp.KindBps, "-25", true}, {gbp.KindBps, "1.5", false},
		{gbp.KindMoney, "2000000", true}, {gbp.KindMoney, "£20,000", false},
		{gbp.KindCycle, "monthly", true}, {gbp.KindCycle, "weekly", false},
		{gbp.KindDay, "2026-04-01", true}, {gbp.KindDay, "1 April", false},
		{gbp.KindText, "anything", true},
	}
	for _, c := range cases {
		d := gbp.Declaration{Key: "k", Kind: c.kind}
		_, err := d.Parse(c.text)
		if (err == nil) != c.ok {
			t.Errorf("%s %q: err %v, want ok=%v", c.kind, c.text, err, c.ok)
		}
		if err != nil && !strings.Contains(err.Error(), "parameter k") {
			t.Errorf("error names no parameter: %v", err)
		}
	}
	p, _ := gbp.Declaration{Key: "rate_bps", Kind: gbp.KindBps}.Parse(" 150 ")
	if p.Bps() != 150 || p.Int() != 150 || p.Money() != 150 {
		t.Errorf("accessors on %+v", p)
	}
	d, _ := gbp.Declaration{Key: "maturity_day", Kind: gbp.KindDay}.Parse("2026-04-01")
	if d.Day() != utcDay(2026, 4, 1) {
		t.Errorf("Day() = %v", d.Day())
	}
}

func TestDerivationSpreadFloorAndCap(t *testing.T) {
	floor, cap := int64(0), int64(500)
	d := gbp.Derivation{Source: "boe.base_rate_bps", SpreadBps: -15, Floor: &floor, Cap: &cap}
	for _, c := range []struct{ source, want int64 }{{525, 500}, {400, 385}, {10, 0}, {-50, 0}} {
		if got := d.Apply(c.source); got != c.want {
			t.Errorf("base %d: derived %d, want %d", c.source, got, c.want)
		}
	}
	if got := (gbp.Derivation{Source: "x", SpreadBps: -15}).Apply(-50); got != -65 {
		t.Errorf("no floor: %d, want -65 (a negative rate is allowed)", got)
	}
}

// The market scope is the contract's name for a currency's official
// risk-free rate (gobank ADR-0009): a declaration at that scope names the
// key alone, the owner being the currency of the product that reads it,
// and the one key every market publishes is RiskFreeRateKey.
func TestMarketScopeNamesTheRiskFreeRate(t *testing.T) {
	d := gbp.Declaration{Key: gbp.RiskFreeRateKey, Scope: gbp.ScopeMarket, Kind: gbp.KindBps}
	if d.Scope != "market" || d.Key != "risk_free_rate_bps" {
		t.Errorf("market declaration = %+v", d)
	}
	p, err := d.Parse("-10")
	if err != nil || p.Bps() != -10 {
		t.Errorf("a market rate may be negative: %+v, %v", p, err)
	}
}

// A parameter derived by code (gobank ADR-0006, the policy's lag and
// take; ADR-0009 keeps them out of the contract as data): an Algorithm is
// a function of the parameters it reads on a day, and the declaration
// that carries one is derived, so it publishes nothing and has no formula.
func TestAlgorithmIsADerivationByCode(t *testing.T) {
	spread := func(p gbp.ParameterReader, day time.Time) (string, error) {
		rfr, err := p.Parameter(gbp.RiskFreeRateKey, day)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(rfr.Bps()-100, 10), nil
	}
	decls := []gbp.Declaration{
		{Key: gbp.RiskFreeRateKey, Scope: gbp.ScopeMarket, Kind: gbp.KindBps},
		{Key: "savings_rate_bps", Scope: gbp.ScopeBank, Kind: gbp.KindBps, Algorithm: spread, Description: "the market less 100 bps"},
	}
	if !decls[1].IsDerived() || decls[0].IsDerived() {
		t.Errorf("IsDerived: algorithm %v, plain %v", decls[1].IsDerived(), decls[0].IsDerived())
	}
	if !(gbp.Declaration{Derived: &gbp.Derivation{Source: "x"}}).IsDerived() {
		t.Error("a formula is a derivation too")
	}
	r := testkit.NewReader("test", decls)
	r.Set(gbp.ScopeMarket, gbp.RiskFreeRateKey, "450", utcDay(2026, 1, 1))
	p, err := r.Parameter("savings_rate_bps", utcDay(2026, 1, 2))
	if err != nil || p.Bps() != 350 || p.Kind != gbp.KindBps {
		t.Errorf("algorithm resolved %+v, %v; want 350 bps", p, err)
	}
	if _, err := r.Parameter("savings_rate_bps", utcDay(2025, 12, 31)); err == nil {
		t.Error("an algorithm whose source is missing resolved")
	}
	bad := []gbp.Declaration{
		{Key: "a", Scope: gbp.ScopeBank, Kind: gbp.KindBps, Algorithm: spread, Published: "1"},
		{Key: "b", Scope: gbp.ScopeBank, Kind: gbp.KindBps, Algorithm: spread, Derived: &gbp.Derivation{Source: gbp.RiskFreeRateKey}},
		{Key: "c", Scope: gbp.ScopeBank, Kind: gbp.KindBps, Algorithm: func(gbp.ParameterReader, time.Time) (string, error) { return "1.5", nil }},
	}
	for _, d := range bad {
		pol := policy{decls: append(decls[:1:1], d)}
		if err := gbp.CheckPolicy(pol, testkit.NewReader("bad", pol.decls), utcDay(2026, 1, 2)); err == nil {
			t.Errorf("declaration %s accepted: an algorithm publishes nothing, has no formula and answers in its kind", d.Key)
		}
	}
}

// policy is a test policy: the bank-scoped parameters it declares.
type policy struct{ decls []gbp.Declaration }

func (policy) Name() string                                 { return "Test policy" }
func (policy) Version() int                                 { return 1 }
func (p policy) Parameters() []gbp.Declaration              { return p.decls }
func (policy) StartUp(gbp.ParameterReader, time.Time) error { return nil }

// A policy is checked as a version is: every declaration resolves on the
// day, so the bank refuses to run a policy whose market is silent.
func TestCheckPolicy(t *testing.T) {
	pol := policy{decls: []gbp.Declaration{
		{Key: gbp.RiskFreeRateKey, Scope: gbp.ScopeMarket, Kind: gbp.KindBps},
		{Key: "savings_spread_bps", Scope: gbp.ScopeBank, Kind: gbp.KindBps, Published: "-200"},
		{Key: "savings_rate_bps", Scope: gbp.ScopeBank, Kind: gbp.KindBps, Algorithm: func(p gbp.ParameterReader, day time.Time) (string, error) {
			rfr, err := p.Parameter(gbp.RiskFreeRateKey, day)
			if err != nil {
				return "", err
			}
			spread, err := p.Parameter("savings_spread_bps", day)
			if err != nil {
				return "", err
			}
			return strconv.FormatInt(rfr.Bps()+spread.Bps(), 10), nil
		}},
	}}
	r := testkit.NewReader(pol.Name(), pol.Parameters())
	if err := gbp.CheckPolicy(pol, r, utcDay(2026, 1, 1)); err == nil {
		t.Error("a policy with no market rate passed its check")
	}
	r.Set(gbp.ScopeMarket, gbp.RiskFreeRateKey, "400", utcDay(2025, 1, 1))
	if err := gbp.CheckPolicy(pol, r, utcDay(2026, 1, 1)); err != nil {
		t.Errorf("check: %v", err)
	}
	if p, err := r.Parameter("savings_rate_bps", utcDay(2026, 1, 1)); err != nil || p.Bps() != 200 {
		t.Errorf("savings rate = %+v, %v; want 200: the published spread applies with no setting", p, err)
	}
	r.Set(gbp.ScopeBank, "savings_spread_bps", "-250", utcDay(2026, 2, 1))
	if p, _ := r.Parameter("savings_rate_bps", utcDay(2026, 2, 1)); p.Bps() != 150 {
		t.Errorf("savings rate after the spread was set = %d, want 150", p.Bps())
	}
}

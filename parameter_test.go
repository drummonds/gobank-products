package gbp_test

import (
	"strings"
	"testing"

	gbp "git.bytestone.uk/hum3/gobank-products"
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

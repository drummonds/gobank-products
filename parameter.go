package gbp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
)

// Scope is whose parameter it is.
type Scope string

const (
	ScopeMarket  Scope = "market"  // a currency's: its official risk-free rate, written only by the market's feed
	ScopeBank    Scope = "bank"    // the bank's: its policy's rates, spreads and lags
	ScopeVersion Scope = "version" // the version's: its rate, its cycle
	ScopeAccount Scope = "account" // one account's: a maturity day
)

// RiskFreeRateKey is the one parameter every market publishes: the
// currency's official risk-free rate in basis points (GBP: the Bank of
// England's Bank Rate). A declaration at ScopeMarket names this key; the
// runner resolves it for the currency of the product whose rules read it.
const RiskFreeRateKey = "risk_free_rate_bps"

// The bank's own rates (gobank ADR-0006, "bank rate"): the base rate for
// each family, which a bank's policy declares at ScopeBank and a product
// version that tracks the bank reads by the same key.
const (
	SavingsRateKey = "savings_rate_bps"
	LendingRateKey = "lending_rate_bps"
)

// Kind is a parameter's type, which fixes how its text is read.
type Kind string

const (
	KindBps   Kind = "bps"   // an annual rate in basis points, signed
	KindMoney Kind = "money" // minor units of the account's commodity
	KindInt   Kind = "int"
	KindCycle Kind = "cycle" // an ApplicationCycle
	KindDay   Kind = "day"   // a calendar day, 2006-01-02
	KindText  Kind = "text"
)

// Declaration is a parameter as a version or a policy declares it: its
// key, scope and kind; the value published when it is adopted (the
// version's own scope, or the bank's for a policy); or, for a derived
// parameter, how it is derived: a formula (Derived) or code (Algorithm).
type Declaration struct {
	Key         string
	Scope       Scope
	Kind        Kind
	Published   string      // the value at adoption; "" when there is none
	Derived     *Derivation // set when the value is a formula over another parameter
	Algorithm   Algorithm   // set when the value is computed by code from the parameters it reads
	Description string
}

// IsDerived reports whether the parameter is derived, by formula or by
// code: read, never set, and with no settings of its own.
func (d Declaration) IsDerived() bool { return d.Derived != nil || d.Algorithm != nil }

// Algorithm is a parameter derived by code: a function of the parameters
// it reads through the reader on a day, answering in the declaration's
// kind. Where a Derivation is one formula the bank can describe, an
// algorithm is whatever the declarer's rules say, and the declarer
// describes it in the declaration's Description. The bank's policy
// derives its rates this way (gobank ADR-0006, ADR-0009): a rate that
// follows the market down at once and up after a lag reads the market on
// the day and the lag's days earlier and takes the lower. The reads it
// makes are logged as any rule's are.
type Algorithm func(p ParameterReader, day time.Time) (string, error)

// Derivation defines a rate parameter as a function of another: the
// source's basis points plus a spread, held within an optional floor and
// cap. Code, not a setting: a derived parameter has no settings of its own.
type Derivation struct {
	Source    string // the key of the parameter derived from
	SpreadBps int64
	Floor     *int64
	Cap       *int64
}

// Apply is the derived rate for a source rate.
func (d Derivation) Apply(sourceBps int64) int64 {
	bps := sourceBps + d.SpreadBps
	if d.Floor != nil && bps < *d.Floor {
		bps = *d.Floor
	}
	if d.Cap != nil && bps > *d.Cap {
		bps = *d.Cap
	}
	return bps
}

// Param is a parameter resolved for a day: its text, read by its kind.
type Param struct {
	Key  string
	Kind Kind
	Text string
}

// Parse reads text as the declared kind. Every resolver goes through it,
// so a Param's accessors cannot fail.
func (d Declaration) Parse(text string) (Param, error) {
	p := Param{Key: d.Key, Kind: d.Kind, Text: strings.TrimSpace(text)}
	var err error
	switch d.Kind {
	case KindBps, KindMoney, KindInt:
		_, err = strconv.ParseInt(p.Text, 10, 64)
	case KindCycle:
		switch ApplicationCycle(p.Text) {
		case ApplyDaily, ApplyMonthly, ApplyAnnual:
		default:
			err = fmt.Errorf("not a cycle")
		}
	case KindDay:
		_, err = time.Parse("2006-01-02", p.Text)
	case KindText:
	default:
		err = fmt.Errorf("unknown kind %q", d.Kind)
	}
	if err != nil {
		return Param{}, fmt.Errorf("parameter %s: %q is not a %s", d.Key, text, d.Kind)
	}
	return p, nil
}

func (p Param) integer() int64 {
	n, _ := strconv.ParseInt(p.Text, 10, 64)
	return n
}

// Bps is the value as basis points.
func (p Param) Bps() int64 { return p.integer() }

// Money is the value in minor units.
func (p Param) Money() luca.Amount { return luca.Amount(p.integer()) }

// Int is the value as an integer.
func (p Param) Int() int64 { return p.integer() }

// Cycle is the value as an application cycle.
func (p Param) Cycle() ApplicationCycle { return ApplicationCycle(p.Text) }

// Day is the value as a calendar day, UTC.
func (p Param) Day() time.Time {
	t, _ := time.Parse("2006-01-02", p.Text)
	return t
}

// ParameterReader resolves a declared parameter for a day. The runner
// implements it over the bank's settings (ADR-0006, parameter resolution).
type ParameterReader interface {
	Parameter(key string, day time.Time) (Param, error)
}

// Declared finds a version's declaration of key.
func Declared(v Version, key string) (Declaration, bool) {
	for _, d := range v.Parameters() {
		if d.Key == key {
			return d, true
		}
	}
	return Declaration{}, false
}

// CheckDeclarations is the part of start-up every version shares: each
// published value parses as its kind, each derivation names a declared
// source, and every declared parameter resolves on day.
func CheckDeclarations(v Version, p ParameterReader, day time.Time) error {
	return checkDeclarations(fmt.Sprintf("%s v%d", v.Product(), v.Version()), v.Parameters(), p, day)
}

// CheckPolicy is a policy's start-up check, as CheckDeclarations is a
// version's: every parameter it declares resolves on day, so a policy
// whose market is silent, or whose algorithm cannot answer, stops the
// bank before it runs.
func CheckPolicy(pol Policy, p ParameterReader, day time.Time) error {
	return checkDeclarations(fmt.Sprintf("%s v%d", pol.Name(), pol.Version()), pol.Parameters(), p, day)
}

func checkDeclarations(who string, decls []Declaration, p ParameterReader, day time.Time) error {
	declared := func(key string) bool {
		for _, d := range decls {
			if d.Key == key {
				return true
			}
		}
		return false
	}
	for _, d := range decls {
		if d.Published != "" {
			if d.IsDerived() {
				return fmt.Errorf("%s: parameter %s is derived and publishes a value", who, d.Key)
			}
			if _, err := d.Parse(d.Published); err != nil {
				return fmt.Errorf("%s: published %w", who, err)
			}
		}
		if d.Derived != nil && d.Algorithm != nil {
			return fmt.Errorf("%s: parameter %s has both a formula and an algorithm", who, d.Key)
		}
		if d.Derived != nil && !declared(d.Derived.Source) {
			return fmt.Errorf("%s: parameter %s derives from undeclared %s", who, d.Key, d.Derived.Source)
		}
		if d.Scope == ScopeAccount {
			continue // written at open; there is no account at start-up
		}
		if _, err := p.Parameter(d.Key, day); err != nil {
			return fmt.Errorf("%s: %w", who, err)
		}
	}
	return nil
}

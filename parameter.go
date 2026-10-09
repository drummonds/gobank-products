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
	ScopeBank    Scope = "bank"    // the bank's: the base rate
	ScopeVersion Scope = "version" // the version's: its rate, its cycle
	ScopeAccount Scope = "account" // one account's: a maturity day
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

// Declaration is a parameter as a version declares it: its key, scope and
// kind; the value published when the version is adopted (version scope
// only); or, for a derived parameter, the derivation.
type Declaration struct {
	Key         string
	Scope       Scope
	Kind        Kind
	Published   string      // the value at adoption; "" when there is none
	Derived     *Derivation // set when the value is derived from another parameter
	Description string
}

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
	for _, d := range v.Parameters() {
		if d.Published != "" {
			if _, err := d.Parse(d.Published); err != nil {
				return fmt.Errorf("%s v%d: published %w", v.Product(), v.Version(), err)
			}
		}
		if d.Derived != nil {
			if _, ok := Declared(v, d.Derived.Source); !ok {
				return fmt.Errorf("%s v%d: parameter %s derives from undeclared %s", v.Product(), v.Version(), d.Key, d.Derived.Source)
			}
		}
		if d.Scope == ScopeAccount {
			continue // written at open; there is no account at start-up
		}
		if _, err := p.Parameter(d.Key, day); err != nil {
			return fmt.Errorf("%s v%d: %w", v.Product(), v.Version(), err)
		}
	}
	return nil
}

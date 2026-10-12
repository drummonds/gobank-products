package testkit

import (
	"fmt"
	"strconv"
	"time"

	gbp "git.bytestone.uk/hum3/gobank-products"
)

// Reader is a ParameterReader over one declarer's declarations and
// settings held in memory by scope, with no ledger: the parameter
// resolution of gobank ADR-0006 as the bank's runner does it. A
// derivation resolves from its source and the formula; an algorithm runs
// over the reader itself; anything else is the latest setting effective
// by the day at its scope, or the published value. A policy tests
// through one directly; the Runner reads parameters through one.
type Reader struct {
	who      string
	decls    []gbp.Declaration
	settings map[gbp.Scope]map[string][]dated
}

type dated struct {
	text      string
	effective time.Time
}

// NewReader is a reader of decls, as who declares them (for its errors),
// with no settings yet.
func NewReader(who string, decls []gbp.Declaration) *Reader {
	return &Reader{who: who, decls: decls, settings: map[gbp.Scope]map[string][]dated{
		gbp.ScopeMarket: {}, gbp.ScopeBank: {}, gbp.ScopeVersion: {}, gbp.ScopeAccount: {},
	}}
}

// Set writes a setting at a scope effective from a day. Settings are
// written in effective order.
func (r *Reader) Set(scope gbp.Scope, key, text string, effective time.Time) {
	r.settings[scope][key] = append(r.settings[scope][key], dated{text, startOfDay(effective)})
}

// Parameter resolves key for day.
func (r *Reader) Parameter(key string, day time.Time) (gbp.Param, error) {
	return r.resolve(r, key, day)
}

// resolve is Parameter with the derivations' reads going through via, so
// a caller logging reads sees an algorithm's.
func (r *Reader) resolve(via gbp.ParameterReader, key string, day time.Time) (gbp.Param, error) {
	day = startOfDay(day)
	var d gbp.Declaration
	found := false
	for _, c := range r.decls {
		if c.Key == key {
			d, found = c, true
			break
		}
	}
	if !found {
		return gbp.Param{}, fmt.Errorf("parameter %s: not declared by %s", key, r.who)
	}
	if d.Derived != nil {
		source, err := via.Parameter(d.Derived.Source, day)
		if err != nil {
			return gbp.Param{}, err
		}
		return d.Parse(strconv.FormatInt(d.Derived.Apply(source.Bps()), 10))
	}
	if d.Algorithm != nil {
		text, err := d.Algorithm(via, day)
		if err != nil {
			return gbp.Param{}, fmt.Errorf("parameter %s: %w", key, err)
		}
		return d.Parse(text)
	}
	var text string
	found = false
	for _, s := range r.settings[d.Scope][key] {
		if !s.effective.After(day) {
			text, found = s.text, true // later-effective settings come later in the list
		}
	}
	if !found && d.Published != "" {
		text, found = d.Published, true
	}
	if !found {
		return gbp.Param{}, fmt.Errorf("parameter %s: no %s setting on %s", key, d.Scope, day.Format("2006-01-02"))
	}
	return d.Parse(text)
}

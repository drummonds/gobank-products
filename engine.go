package gbp

import (
	"math"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
)

// The interest arithmetic every version shares, and the part of the module
// that outlives any version: a retired version's postings are verified by
// this arithmetic over the record, never by the version's rules.

// AccrualDenominator is the fixed denominator for exact interest accrual:
// 10,000 (basis points per unit rate) x 365 (actual/365 day count).
// Daily accrual accumulates balance_minor_units * rate_bps into a numerator;
// dividing by this denominator yields minor units. See interest.md.
const AccrualDenominator = 10_000 * 365

// RateBps is an annual rate (0.035 for 3.5%) in the integer basis points
// the rules accrue in. Rounded, so a negative rate converts too.
func RateBps(annualRate float64) int64 {
	return int64(math.Round(annualRate * 10_000))
}

// ApplicationCycle says when accrued interest is applied.
type ApplicationCycle string

const (
	ApplyDaily   ApplicationCycle = "daily"
	ApplyMonthly ApplicationCycle = "monthly"
	ApplyAnnual  ApplicationCycle = "annual"
)

// EndsOn reports whether a period of the cycle ends on day.
func (c ApplicationCycle) EndsOn(day time.Time) bool {
	switch c {
	case ApplyDaily:
		return true
	case ApplyAnnual:
		return day.Month() == time.December && day.Day() == 31
	default:
		return day.AddDate(0, 0, 1).Month() != day.Month()
	}
}

// Description is the posting description for the cycle's application on day.
func (c ApplicationCycle) Description(day time.Time) string {
	d := day.Format("2006-01-02")
	switch c {
	case ApplyDaily:
		return "Interest applied for " + d
	case ApplyAnnual:
		return "Interest applied for year ending " + d
	default:
		return "Interest applied for month ending " + d
	}
}

// Accrue is the position at the end of day with the day's interest on
// closing accrued (actual/365, in basis points, exact) on top of what prev
// carried, and nothing applied. It is pure, and the same for any balance
// the day closes on, so the pass may write it as the day's provisional
// projection and every event that moves the balance may write it again.
func Accrue(day time.Time, prev luca.Position, closing luca.Amount, rateBps int64) luca.Position {
	day = startOfDay(day)
	return luca.Position{
		AccountID: prev.AccountID,
		Day:       day,
		Balance:   closing,
		Accrued:   luca.Fraction{Num: accrualNumerator(prev.Accrued) + int64(closing)*rateBps, Den: AccrualDenominator},
	}
}

// ApplyAccrued moves the whole minor units of a position's accrual into
// its balance and returns the posting that books them, value-dated the
// last second of the position's day, the remainder carrying forward. A
// position with less than a whole unit accrued is returned as it is with
// no posting, so applying an applied position books nothing twice.
func ApplyAccrued(family ProductFamily, pos luca.Position, description string) (luca.Position, []Posting) {
	day := startOfDay(pos.Day)
	numerator := accrualNumerator(pos.Accrued)
	units := numerator / AccrualDenominator // toward zero; the fraction stays accrued
	if units == 0 {
		return pos, nil
	}
	next := pos
	next.Balance += luca.Amount(units)
	next.Accrued = luca.Fraction{Num: numerator - units*AccrualDenominator, Den: AccrualDenominator}
	counterparty := "Expense:Interest"
	if family == FamilyLending {
		counterparty = "Income:Interest"
	}
	return next, []Posting{{
		Counterparty: counterparty,
		Amount:       luca.Amount(units),
		Code:         luca.CodeInterestAccrual,
		ValueTime:    time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, day.Location()),
		Description:  description,
	}}
}

// accrualNumerator reads a position's accrued interest at
// AccrualDenominator. A zero denominator is no accrual; any other
// denominator is rescaled, exactly only when it divides AccrualDenominator.
func accrualNumerator(f luca.Fraction) int64 {
	switch f.Den {
	case 0:
		return 0
	case AccrualDenominator:
		return f.Num
	}
	return f.Num * AccrualDenominator / f.Den
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

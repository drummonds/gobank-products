package gbp

import (
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
)

// ParamInterestApplication is the product parameter naming the cycle on
// which accrued interest is applied to the account.
const ParamInterestApplication = "interest_application"

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

func (c ApplicationCycle) description(day time.Time) string {
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

// ApplicationCycle is the product's cycle: monthly, the UK convention,
// unless its defaults say daily or annual.
func (p *Product) ApplicationCycle() ApplicationCycle {
	switch c := ApplicationCycle(p.Defaults[ParamInterestApplication]); c {
	case ApplyDaily, ApplyAnnual:
		return c
	}
	return ApplyMonthly
}

// Posting is a ledger movement a day's rules call for: from the
// counterparty to the account, signed as the account's balance is, so a
// credit-normal savings balance receives a negative amount.
type Posting struct {
	Counterparty string // P&L account path: Expense:Interest or Income:Interest
	Amount       luca.Amount
	Code         string
	ValueTime    time.Time
	Description  string
}

// NextDay is the product's day rule for one account. From the account's
// position at the end of the previous day and the day's closing balance it
// gives the position at the end of this day and the postings it calls for:
// the day's interest accrues on the closing balance (actual/365, in basis
// points, exact), and when the product's cycle ends on this day the whole
// minor units of the accrual are applied to the balance, the remainder
// carrying forward. It is pure: the caller posts and stores.
func (p *Product) NextDay(day time.Time, prev luca.Position, closing luca.Amount, rateBps int64) (luca.Position, []Posting) {
	day = startOfDay(day)
	numerator := accrualNumerator(prev.Accrued) + int64(closing)*rateBps
	next := luca.Position{
		AccountID: prev.AccountID,
		Day:       day,
		Balance:   closing,
		Accrued:   luca.Fraction{Num: numerator, Den: AccrualDenominator},
	}
	cycle := p.ApplicationCycle()
	if !cycle.EndsOn(day) {
		return next, nil
	}
	pence := numerator / AccrualDenominator // toward zero; the fraction stays accrued
	if pence == 0 {
		return next, nil
	}
	next.Balance += luca.Amount(pence)
	next.Accrued.Num -= pence * AccrualDenominator
	counterparty := "Expense:Interest"
	if p.Family == FamilyLending {
		counterparty = "Income:Interest"
	}
	return next, []Posting{{
		Counterparty: counterparty,
		Amount:       luca.Amount(pence),
		Code:         luca.CodeInterestAccrual,
		ValueTime:    time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, day.Location()),
		Description:  cycle.description(day),
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

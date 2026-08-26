package gbp

import (
	"fmt"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
)

// AccrualDenominator is the fixed denominator for exact interest accrual:
// 10,000 (basis points per unit rate) x 365 (actual/365 day count).
// Daily accrual accumulates balance_minor_units * rate_bps into a numerator;
// dividing by this denominator yields minor units. See interest.md.
const AccrualDenominator = 10_000 * 365

// InterestAccrual accrues interest daily in memory (exact integer arithmetic,
// no ledger writes and no balance queries) and applies the accumulated
// interest to the account as a ledger movement at end of month. This follows
// the recommended UK convention of daily accrual with monthly application.
//
// Accrual for a single account is idempotent per date (guarded by
// ManagedAccount.LastAccrued), so a caller may accrue accounts individually,
// spread across the day, instead of concentrating all work at end-of-day —
// any accounts not yet accrued are picked up by the end-of-day sweep.
type InterestAccrual struct{}

func (InterestAccrual) Name() string { return "interest" }
func (InterestAccrual) Handles() []EventType {
	return []EventType{EventEndOfDay, EventEndOfMonth}
}

// HandleEndOfDay accrues one day of interest into the account's accumulator:
// numerator += cached_balance_minor_units * rate_bps. Pure in-memory integer
// arithmetic — no rounding, nothing is lost however small the balance.
func (InterestAccrual) HandleEndOfDay(ctx *SimContext, e EndOfDayEvent) error {
	accrueAccountDay(e.Account, ctx.AsOfDate)
	return nil
}

// accrueAccountDay performs the daily accrual if not already done for date.
func accrueAccountDay(ma *ManagedAccount, date time.Time) {
	day := startOfDay(date)
	if ma.RateBps == 0 || !ma.LastAccrued.Before(day) {
		return
	}
	ma.AccruedNumerator += int64(ma.CachedBalance) * ma.RateBps
	ma.LastAccrued = day
}

// HandleEndOfMonth applies accumulated interest to the account: the whole
// minor units are posted as a ledger movement and the sub-unit remainder
// carries forward in the accumulator.
//
// Directions follow the ledger's balance convention (balance = in - out):
//
//	savings:  Expense:Interest -> account  (bank expense, customer balance up)
//	lending:  Income:Interest  -> account  (bank income, customer obligation up)
func (InterestAccrual) HandleEndOfMonth(ctx *SimContext, e EndOfMonthEvent) error {
	ma := e.Account
	pence := ma.AccruedNumerator / AccrualDenominator
	if pence == 0 {
		return nil
	}

	var counterPath string
	if ma.Family == FamilyLending {
		counterPath = "Income:Interest"
	} else {
		counterPath = "Expense:Interest"
	}
	counterAcct, err := ensureAccount(ctx, counterPath, ma.Account.Commodity, ma.Account.Exponent)
	if err != nil {
		return err
	}

	date := ctx.AsOfDate
	desc := fmt.Sprintf("Interest applied for month ending %s", date.Format("2006-01-02"))
	valueTime := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, date.Location())

	if _, err := ctx.Sim.RecordMovement(counterAcct.ID, ma.Account.ID, luca.Amount(pence),
		luca.CodeInterestAccrual, valueTime, desc); err != nil {
		return err
	}
	ma.AccruedNumerator -= pence * AccrualDenominator
	return nil
}

// ensureAccount gets or creates an account by path.
func ensureAccount(ctx *SimContext, path, commodity string, exponent int) (*luca.Account, error) {
	acct, err := ctx.Sim.Ledger.GetAccount(path)
	if err != nil {
		return nil, fmt.Errorf("get account %s: %w", path, err)
	}
	if acct != nil {
		return acct, nil
	}
	acct, err = ctx.Sim.Ledger.CreateAccount(path, commodity, exponent, 0)
	if err != nil {
		return nil, fmt.Errorf("create account %s: %w", path, err)
	}
	return acct, nil
}

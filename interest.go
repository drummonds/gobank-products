package gbp

import (
	"fmt"

	luca "git.bytestone.uk/hum3/go-luca"
)

// AccrualDenominator is the fixed denominator for exact interest accrual:
// 10,000 (basis points per unit rate) x 365 (actual/365 day count).
// Daily accrual accumulates balance_minor_units * rate_bps into a numerator;
// dividing by this denominator yields minor units. See interest.md.
const AccrualDenominator = 10_000 * 365

// InterestAccrual runs the product's day rule, Product.NextDay, for an
// account at end of day: the day's accrual on the closing balance, and the
// application of the accrued interest when the product's cycle ends on
// that day. There is no separate month-end pass.
//
// It is idempotent per date (guarded by ManagedAccount.LastAccrued), so a
// caller may run accounts individually, spread across the day, instead of
// concentrating all work at end-of-day — any accounts not yet done are
// picked up by the end-of-day sweep.
type InterestAccrual struct{}

func (InterestAccrual) Name() string { return "interest" }
func (InterestAccrual) Handles() []EventType {
	return []EventType{EventEndOfDay}
}

// HandleEndOfDay applies NextDay to the account: postings go to the ledger
// (which keeps the cached balance in step) and the accrual carries forward
// in the account's accumulator.
//
// Directions follow the ledger's balance convention (balance = in - out):
//
//	savings:  Expense:Interest -> account  (bank expense, customer balance up)
//	lending:  Income:Interest  -> account  (bank income, customer obligation up)
func (InterestAccrual) HandleEndOfDay(ctx *SimContext, e EndOfDayEvent) error {
	ma := e.Account
	day := startOfDay(ctx.AsOfDate)
	if !ma.LastAccrued.Before(day) {
		return nil
	}
	product, ok := ctx.Sim.products[ma.ProductID]
	if !ok {
		return fmt.Errorf("interest: unknown product %q", ma.ProductID)
	}
	prev := luca.Position{
		AccountID: ma.Account.ID,
		Day:       day.AddDate(0, 0, -1),
		Accrued:   luca.Fraction{Num: ma.AccruedNumerator, Den: AccrualDenominator},
	}
	next, postings := product.NextDay(day, prev, ma.CachedBalance, ma.RateBps)
	for _, p := range postings {
		counter, err := ensureAccount(ctx, p.Counterparty, ma.Account.Commodity, ma.Account.Exponent)
		if err != nil {
			return err
		}
		if _, err := ctx.Sim.RecordMovement(counter.ID, ma.Account.ID, p.Amount, p.Code, p.ValueTime, p.Description); err != nil {
			return err
		}
	}
	ma.AccruedNumerator = next.Accrued.Num
	ma.LastAccrued = day
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

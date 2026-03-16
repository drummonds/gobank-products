package gbp

import (
	"fmt"
	"time"
)

// TermLock blocks withdrawals before the maturity date and fires MaturityReached.
type TermLock struct{}

func (TermLock) Name() string { return "term_lock" }
func (TermLock) Handles() []EventType {
	return []EventType{EventWithdrawalRequested, EventEndOfDay}
}

func (TermLock) HandleWithdrawalRequested(ctx *SimContext, e WithdrawalRequestedEvent) error {
	matStr, ok := ctx.Params.Get(e.Account.Account.ID, "maturity_date", ctx.AsOfDate)
	if !ok {
		return fmt.Errorf("no maturity date set for account")
	}
	maturity, err := time.Parse("2006-01-02", matStr)
	if err != nil {
		return fmt.Errorf("invalid maturity date %q: %w", matStr, err)
	}
	if ctx.AsOfDate.Before(maturity) {
		return fmt.Errorf("withdrawal blocked: account matures on %s", matStr)
	}
	return nil
}

func (TermLock) HandleEndOfDay(ctx *SimContext, e EndOfDayEvent) error {
	matStr, ok := ctx.Params.Get(e.Account.Account.ID, "maturity_date", ctx.AsOfDate)
	if !ok {
		return nil
	}
	maturity, err := time.Parse("2006-01-02", matStr)
	if err != nil {
		return nil
	}
	// Fire maturity event on the maturity date itself.
	if startOfDay(ctx.AsOfDate).Equal(startOfDay(maturity)) {
		matEvent := MaturityReachedEvent{
			EventHeader: EventHeader{Type: EventMaturityReached, Date: ctx.AsOfDate, Account: e.Account},
		}
		return ctx.Sim.dispatchEvent(e.Account.ProductID, EventMaturityReached, func(f Feature) error {
			if h, ok := f.(OnMaturityReached); ok {
				return h.HandleMaturityReached(ctx, matEvent)
			}
			return nil
		})
	}
	return nil
}

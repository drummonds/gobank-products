package gbp

import (
	"fmt"
	"strconv"

	luca "codeberg.org/hum3/go-luca"
)

// RepaymentSchedule handles monthly repayment logic for lending products.
type RepaymentSchedule struct{}

func (RepaymentSchedule) Name() string { return "repayment" }
func (RepaymentSchedule) Handles() []EventType {
	return []EventType{EventEndOfMonth}
}

func (RepaymentSchedule) HandleEndOfMonth(ctx *SimContext, e EndOfMonthEvent) error {
	amtStr, ok := ctx.Params.Get(e.Account.Account.ID, "monthly_repayment", ctx.AsOfDate)
	if !ok {
		return nil // no repayment configured
	}
	amount, err := strconv.ParseInt(amtStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid monthly_repayment %q: %w", amtStr, err)
	}

	// Find the repayment source account path.
	sourcePath, ok := ctx.Params.Get(e.Account.Account.ID, "repayment_source", ctx.AsOfDate)
	if !ok {
		return fmt.Errorf("no repayment_source set for account %s", e.Account.Account.ID)
	}

	// Record the repayment: source pays into the loan account.
	_, err = ctx.Sim.RecordMovement(sourcePath, e.Account.Account.ID, luca.Amount(amount), luca.CodeBookTransfer, ctx.AsOfDate, "Monthly repayment")
	return err
}

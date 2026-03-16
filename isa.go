package gbp

import (
	"fmt"
	"strconv"
)

// ISAWrapper enforces an annual deposit allowance on the account.
type ISAWrapper struct{}

func (ISAWrapper) Name() string { return "isa" }
func (ISAWrapper) Handles() []EventType {
	return []EventType{EventDepositReceived}
}

func (ISAWrapper) HandleDepositReceived(ctx *SimContext, e DepositReceivedEvent) error {
	allowStr, ok := ctx.Params.Get(e.Account.Account.ID, "isa_allowance", ctx.AsOfDate)
	if !ok {
		return fmt.Errorf("no ISA allowance set")
	}
	allowance, err := strconv.ParseInt(allowStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ISA allowance %q: %w", allowStr, err)
	}

	// Get deposits this tax year (running total stored as parameter).
	deposited := int64(0)
	if depStr, ok := ctx.Params.Get(e.Account.Account.ID, "isa_deposited", ctx.AsOfDate); ok {
		deposited, _ = strconv.ParseInt(depStr, 10, 64)
	}

	if deposited+int64(e.Amount) > allowance {
		remaining := allowance - deposited
		return fmt.Errorf("ISA allowance exceeded: deposited %d + %d > allowance %d (remaining: %d)",
			deposited, e.Amount, allowance, remaining)
	}

	// Update running total.
	newTotal := deposited + int64(e.Amount)
	ctx.Params.Set(e.Account.Account.ID, "isa_deposited", strconv.FormatInt(newTotal, 10), ctx.AsOfDate)

	return nil
}

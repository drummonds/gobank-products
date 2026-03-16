package gbp

import (
	"fmt"
	"strconv"

	luca "github.com/drummonds/go-luca"
)

// OverdraftFacility allows negative balance up to an arranged limit and records the withdrawal.
type OverdraftFacility struct{}

func (OverdraftFacility) Name() string { return "overdraft" }
func (OverdraftFacility) Handles() []EventType {
	return []EventType{EventWithdrawalRequested}
}

func (OverdraftFacility) HandleWithdrawalRequested(ctx *SimContext, e WithdrawalRequestedEvent) error {
	if e.Amount <= 0 {
		return fmt.Errorf("withdrawal amount must be positive, got %d", e.Amount)
	}
	if e.Account.Status != StatusActive {
		return fmt.Errorf("cannot withdraw from account in status %s", e.Account.Status)
	}

	limitStr, ok := ctx.Params.Get(e.Account.Account.ID, "overdraft_limit", ctx.AsOfDate)
	if !ok {
		return fmt.Errorf("no overdraft limit set")
	}
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid overdraft limit: %w", err)
	}

	bal, err := ctx.Sim.Ledger.Balance(e.Account.Account.ID)
	if err != nil {
		return fmt.Errorf("check balance: %w", err)
	}

	// Balance can go negative down to -limit.
	if int64(bal)-int64(e.Amount) < -limit {
		return fmt.Errorf("overdraft limit exceeded: balance %d - %d would breach limit of -%d", bal, e.Amount, limit)
	}

	code := e.Code
	if code == "" {
		code = luca.CodeBookTransfer
	}
	_, err = ctx.Sim.RecordMovement(e.Account.Account.ID, e.ToPath, e.Amount, code, e.Date, "Withdrawal")
	return err
}

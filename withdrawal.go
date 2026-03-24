package gbp

import (
	"fmt"

	luca "codeberg.org/hum3/go-luca"
)

// WithdrawalProcessing validates balance and records withdrawals.
type WithdrawalProcessing struct{}

func (WithdrawalProcessing) Name() string { return "withdrawal" }
func (WithdrawalProcessing) Handles() []EventType {
	return []EventType{EventWithdrawalRequested}
}

func (WithdrawalProcessing) HandleWithdrawalRequested(ctx *SimContext, e WithdrawalRequestedEvent) error {
	if e.Amount <= 0 {
		return fmt.Errorf("withdrawal amount must be positive, got %d", e.Amount)
	}
	if e.Account.Status != StatusActive {
		return fmt.Errorf("cannot withdraw from account in status %s", e.Account.Status)
	}

	bal, err := ctx.Sim.Ledger.Balance(e.Account.Account.ID)
	if err != nil {
		return fmt.Errorf("check balance: %w", err)
	}
	if bal < e.Amount {
		return fmt.Errorf("insufficient balance: have %d, need %d", bal, e.Amount)
	}

	code := e.Code
	if code == "" {
		code = luca.CodeBookTransfer
	}

	_, err = ctx.Sim.RecordMovement(e.Account.Account.ID, e.ToPath, e.Amount, code, e.Date, "Withdrawal")
	return err
}

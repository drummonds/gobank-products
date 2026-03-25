package gbp

import (
	"fmt"

	luca "codeberg.org/hum3/go-luca"
)

// DepositAcceptance validates and records deposits.
type DepositAcceptance struct{}

func (DepositAcceptance) Name() string { return "deposit" }
func (DepositAcceptance) Handles() []EventType {
	return []EventType{EventDepositReceived}
}

func (DepositAcceptance) HandleDepositReceived(ctx *SimContext, e DepositReceivedEvent) error {
	if e.Amount <= 0 {
		return fmt.Errorf("deposit amount must be positive, got %d", e.Amount)
	}
	if e.Account.Status != StatusActive {
		return fmt.Errorf("cannot deposit to account in status %s", e.Account.Status)
	}

	code := e.Code
	if code == "" {
		code = luca.CodeBookTransfer
	}

	// Savings (liability): Account → Cash (balance goes negative = credit)
	// Lending (asset):     Cash → Account (balance goes positive = debit)
	if e.Account.Family == FamilySavings {
		_, err := ctx.Sim.RecordMovement(e.Account.Account.ID, e.FromPath, e.Amount, code, e.Date, "Deposit")
		return err
	}
	_, err := ctx.Sim.RecordMovement(e.FromPath, e.Account.Account.ID, e.Amount, code, e.Date, "Deposit")
	return err
}

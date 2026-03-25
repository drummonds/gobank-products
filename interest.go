package gbp

import (
	"fmt"
	"time"

	luca "codeberg.org/hum3/go-luca"
	"github.com/shopspring/decimal"
)

// InterestAccrual computes daily interest and posts it to an accrual sub-account.
// Interest is calculated in the product, not in go-luca.
type InterestAccrual struct{}

func (InterestAccrual) Name() string { return "interest" }
func (InterestAccrual) Handles() []EventType {
	return []EventType{EventEndOfDay}
}

func (InterestAccrual) HandleEndOfDay(ctx *SimContext, e EndOfDayEvent) error {
	acct := e.Account.Account
	if acct.GrossInterestRate == 0 {
		return nil
	}

	eod := time.Date(ctx.AsOfDate.Year(), ctx.AsOfDate.Month(), ctx.AsOfDate.Day(),
		23, 59, 59, 999999999, ctx.AsOfDate.Location())

	balance, err := ctx.Sim.Ledger.BalanceAt(acct.ID, eod)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}
	if balance == 0 {
		return nil
	}

	interest := computeDailyInterest(balance, acct)
	if interest == 0 {
		return nil
	}

	return postAccrual(ctx, e.Account, interest, ctx.AsOfDate)
}

// computeDailyInterest: balance * rate / 365, truncated to account exponent.
func computeDailyInterest(balance luca.Amount, acct *luca.Account) luca.Amount {
	balDec := luca.IntToDecimal(balance, acct.Exponent)
	rateDec := decimal.NewFromFloat(acct.GrossInterestRate)
	dailyRate := rateDec.Div(decimal.NewFromInt(365))
	interestDec := balDec.Mul(dailyRate)
	return luca.DecimalToInt(interestDec, acct.Exponent)
}

// postAccrual records a daily interest IOU:
//   Expense:Interest:Accrued → <account>:Accrue  (negative amount for savings, positive for lending)
func postAccrual(ctx *SimContext, ma *ManagedAccount, interest luca.Amount, date time.Time) error {
	accruePath := ma.Account.FullPath + ":Accrued"
	accrueAcct, err := ensureAccount(ctx, accruePath, ma.Account.Commodity, ma.Account.Exponent)
	if err != nil {
		return err
	}

	expenseAcct, err := ensureAccount(ctx, "Expense:Interest:Accrued", ma.Account.Commodity, ma.Account.Exponent)
	if err != nil {
		return err
	}

	desc := fmt.Sprintf("Daily interest for %s", date.Format("2006-01-02"))
	valueTime := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, date.Location())

	_, err = ctx.Sim.RecordMovement(expenseAcct.ID, accrueAcct.ID, interest, luca.CodeInterestAccrual, valueTime, desc)
	return err
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

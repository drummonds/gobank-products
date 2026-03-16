package gbp

// InterestAccrual delegates daily interest calculation to go-luca.
type InterestAccrual struct{}

func (InterestAccrual) Name() string { return "interest" }
func (InterestAccrual) Handles() []EventType {
	return []EventType{EventEndOfDay}
}

func (InterestAccrual) HandleEndOfDay(ctx *SimContext, e EndOfDayEvent) error {
	_, err := ctx.Sim.Ledger.CalculateDailyInterest(e.Account.Account.ID, ctx.AsOfDate)
	return err
}

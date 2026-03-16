package gbp

// StatusLifecycle manages account status transitions.
type StatusLifecycle struct{}

func (StatusLifecycle) Name() string { return "lifecycle" }
func (StatusLifecycle) Handles() []EventType {
	return []EventType{EventAccountOpened, EventAccountClosed}
}

func (StatusLifecycle) HandleAccountOpened(_ *SimContext, e AccountOpenedEvent) error {
	e.Account.Status = StatusActive
	return nil
}

func (StatusLifecycle) HandleAccountClosed(_ *SimContext, e AccountClosedEvent) error {
	e.Account.Status = StatusClosed
	e.Account.ClosedAt = e.Date
	return nil
}

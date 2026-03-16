package gbp

import (
	"time"
)

// Feature is a composable building block for product behaviour.
type Feature interface {
	Name() string
	Handles() []EventType
}

// Typed handler interfaces — a feature implements only those it needs.

type OnAccountOpened interface {
	HandleAccountOpened(ctx *SimContext, e AccountOpenedEvent) error
}

type OnDepositReceived interface {
	HandleDepositReceived(ctx *SimContext, e DepositReceivedEvent) error
}

type OnWithdrawalRequested interface {
	HandleWithdrawalRequested(ctx *SimContext, e WithdrawalRequestedEvent) error
}

type OnEndOfDay interface {
	HandleEndOfDay(ctx *SimContext, e EndOfDayEvent) error
}

type OnEndOfMonth interface {
	HandleEndOfMonth(ctx *SimContext, e EndOfMonthEvent) error
}

type OnRateChanged interface {
	HandleRateChanged(ctx *SimContext, e RateChangedEvent) error
}

type OnMaturityReached interface {
	HandleMaturityReached(ctx *SimContext, e MaturityReachedEvent) error
}

type OnAccountClosed interface {
	HandleAccountClosed(ctx *SimContext, e AccountClosedEvent) error
}

// SimContext is passed to feature handlers during event dispatch.
type SimContext struct {
	Sim      *Simulation
	Params   *ParameterStore
	Clock    Clock
	AsOfDate time.Time
}

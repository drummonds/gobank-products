package gbp

import (
	"time"

	luca "github.com/drummonds/go-luca"
)

// EventType identifies the kind of simulation event.
type EventType int

const (
	EventAccountOpened EventType = iota
	EventDepositReceived
	EventWithdrawalRequested
	EventEndOfDay
	EventEndOfMonth
	EventRateChanged
	EventMaturityReached
	EventAccountClosed
)

func (e EventType) String() string {
	switch e {
	case EventAccountOpened:
		return "AccountOpened"
	case EventDepositReceived:
		return "DepositReceived"
	case EventWithdrawalRequested:
		return "WithdrawalRequested"
	case EventEndOfDay:
		return "EndOfDay"
	case EventEndOfMonth:
		return "EndOfMonth"
	case EventRateChanged:
		return "RateChanged"
	case EventMaturityReached:
		return "MaturityReached"
	case EventAccountClosed:
		return "AccountClosed"
	default:
		return "Unknown"
	}
}

// EventHeader is common to all events.
type EventHeader struct {
	Type    EventType
	Date    time.Time
	Account *ManagedAccount
}

// AccountOpenedEvent is fired when a new account is created.
type AccountOpenedEvent struct {
	EventHeader
	Params map[string]string
}

// DepositReceivedEvent is fired when funds arrive.
type DepositReceivedEvent struct {
	EventHeader
	Amount   luca.Amount
	FromPath string
	Code     string
}

// WithdrawalRequestedEvent is fired when a withdrawal is requested.
type WithdrawalRequestedEvent struct {
	EventHeader
	Amount luca.Amount
	ToPath string
	Code   string
}

// EndOfDayEvent is fired at the end of each simulated day.
type EndOfDayEvent struct {
	EventHeader
}

// EndOfMonthEvent is fired at the end of each simulated month.
type EndOfMonthEvent struct {
	EventHeader
}

// RateChangedEvent is fired when an interest rate changes.
type RateChangedEvent struct {
	EventHeader
	OldRate float64
	NewRate float64
}

// MaturityReachedEvent is fired when a fixed-term product matures.
type MaturityReachedEvent struct {
	EventHeader
}

// AccountClosedEvent is fired when an account is closed.
type AccountClosedEvent struct {
	EventHeader
}

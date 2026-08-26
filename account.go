package gbp

import (
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
)

// AccountStatus represents the lifecycle state of a managed account.
type AccountStatus int

const (
	StatusPending AccountStatus = iota
	StatusActive
	StatusPendingClosure
	StatusClosed
)

func (s AccountStatus) String() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusActive:
		return "Active"
	case StatusPendingClosure:
		return "PendingClosure"
	case StatusClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// ManagedAccount wraps a go-luca Account with product lifecycle state.
type ManagedAccount struct {
	Account   *luca.Account
	ProductID string
	Family    ProductFamily
	Status    AccountStatus
	OpenedAt  time.Time
	ClosedAt  time.Time

	// CachedBalance is the account's ledger balance in minor units, maintained
	// by the Simulation as movements are recorded. It lets daily accrual run
	// without a per-account balance query. Refresh from the ledger with
	// Simulation.RefreshBalances after an import.
	CachedBalance luca.Amount
	// RateBps is the annual gross interest rate in integer basis points
	// (e.g. 350 for 3.50%), derived from annual_rate at open. Integer basis
	// points allow exact accrual arithmetic with no float involvement.
	RateBps int64
	// AccruedNumerator is accrued-but-unapplied interest held as an exact
	// integer fraction: minor units = AccruedNumerator / AccrualDenominator
	// (see interest.md, "the 10,000 x 365 denominator"). Accumulating the
	// numerator defers all division to application time, so sub-minor-unit
	// interest is never lost — the remainder carries forward.
	AccruedNumerator int64
	// LastAccrued is the most recent date daily accrual ran for this account,
	// making per-account accrual idempotent so callers may spread accrual
	// work across the day rather than concentrating it at end-of-day.
	LastAccrued time.Time
}

// AccruedInterest returns accrued-but-unapplied interest in minor units,
// truncated toward zero (the sub-unit remainder stays in the accumulator).
func (ma *ManagedAccount) AccruedInterest() luca.Amount {
	return luca.Amount(ma.AccruedNumerator / AccrualDenominator)
}

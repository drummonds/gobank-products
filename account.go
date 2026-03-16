package gbp

import (
	"time"

	luca "github.com/drummonds/go-luca"
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
	Status    AccountStatus
	OpenedAt  time.Time
	ClosedAt  time.Time
}

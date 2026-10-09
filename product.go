// Package gbp is the product contract of gobank (ADR-0006): a product
// version is a Go package that declares the parameters its rules read and
// answers a fixed set of events with intents. The bank runs it; this
// package holds the contract and the interest arithmetic every version
// shares. A version lives in a package of its own, easyaccess/v1 and the
// like, immutable once the bank has adopted it.
package gbp

import "time"

// ProductFamily is which side of the balance sheet a product's accounts
// are on: a savings balance is a liability and credit-normal (negative in
// the ledger), a lending balance an asset.
type ProductFamily string

const (
	FamilySavings ProductFamily = "Savings"
	FamilyLending ProductFamily = "Lending"
)

// Event is something the bank tells a version has happened: to one of its
// accounts, or to the version itself. The set is fixed; a version answers
// every one, if only with nothing.
type Event string

const (
	EventStartUp         Event = "start-up"
	EventOpen            Event = "open"
	EventPrePosting      Event = "pre-posting"
	EventPostPosting     Event = "post-posting"
	EventDay             Event = "day"
	EventParameterChange Event = "parameter-change"
	EventCommand         Event = "command"
	EventChangeOfVersion Event = "change-of-version"
	EventClose           Event = "close"
)

// Version is one product's rules at one point in its life: its identity,
// the parameters it declares, and its answer to each event.
type Version interface {
	Product() string // the product's identity: "easy-access"
	Version() int    // 1, 2, …: a change to the rules is a new version
	Name() string
	Family() ProductFamily
	Parameters() []Declaration
	Rules
}

// Rules are a version's answers to the events. Every rule is pure: it
// reads facts through the interface it is given and returns intents for
// the runner to carry out, so it is the same however often it runs for
// the same facts, and a version tests with no database.
type Rules interface {
	// StartUp runs once per process per adopted version, with the
	// parameters as the bank can resolve them on day. An error stops the
	// bank: a version that cannot resolve its parameters must not run.
	StartUp(p ParameterReader, day time.Time) error
	// Open runs when an account is opened on the version: the first
	// position and any account-scoped settings the rules will need.
	Open(f Facts) (Intents, error)
	// PrePosting runs before a movement posts to the account. A nil error
	// allows it; a *Refusal refuses it with a reason; any other error is a
	// fault.
	PrePosting(f Facts, m Movement) error
	// PostPosting runs after the movement posted, with the balance as it
	// now stands: the day's position again.
	PostPosting(f Facts, m Movement) (Intents, error)
	// Day runs when the start-of-day pass visits the account: yesterday
	// closed with the postings it calls for, and today's position.
	Day(f Facts) (Intents, error)
	// ParameterChange runs when a setting the version reads becomes
	// effective for the account.
	ParameterChange(f Facts, c Change) (Intents, error)
	// Commands names the manual events the version answers.
	Commands() []string
	// Command runs one of them, by name, from the console.
	Command(f Facts, name string) (Intents, error)
	// ChangeOfVersion runs on the outgoing version when an account moves
	// to another: the outgoing version closes what it was in the middle of.
	ChangeOfVersion(f Facts, to Version) (Intents, error)
	// Close runs when the account is closed: the final postings.
	Close(f Facts) (Intents, error)
}

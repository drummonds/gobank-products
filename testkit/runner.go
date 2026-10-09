package testkit

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"

	gbp "git.bytestone.uk/hum3/gobank-products"
)

// Runner is the reference runner of the product contract: one account on
// one version over an in-memory go-luca ledger, driven through the events
// as the bank drives them. It implements Facts over the ledger and over
// settings held by scope, carries out every intent a rule returns, and
// logs the parameters a rule read. A version's golden test is a scenario
// run through it and exported as .goluca.
type Runner struct {
	t         *testing.T
	Ledger    luca.Ledger
	version   gbp.Version
	day       time.Time // the day the next event is for
	processed time.Time // the last day the Day event ran for; zero until it has
	account   *luca.Account
	cash      *luca.Account
	settings  map[gbp.Scope]map[string][]dated
	// Reads is every parameter the last event's rule read, in order: what
	// the bank records against the postings the rule called for.
	Reads []gbp.Param
}

type dated struct {
	text      string
	effective time.Time
}

// Open opens an account on version v at path on day, after the version's
// start-up check, and returns the runner positioned on that day.
func Open(t *testing.T, v gbp.Version, path string, day time.Time) *Runner {
	t.Helper()
	return OpenWith(t, v, path, day, nil)
}

// OpenWith is Open with the bank's settings written first, by setup, so a
// version whose start-up needs them (a tracker needs the base rate) can pass.
func OpenWith(t *testing.T, v gbp.Version, path string, day time.Time, setup func(*Runner)) *Runner {
	t.Helper()
	r := &Runner{t: t, Ledger: NewTestLedger(t), version: v, day: startOfDay(day),
		settings: map[gbp.Scope]map[string][]dated{gbp.ScopeBank: {}, gbp.ScopeVersion: {}, gbp.ScopeAccount: {}}}
	var err error
	if r.cash, err = r.Ledger.CreateAccount("Asset:Cash", "GBP", -2, 0); err != nil {
		t.Fatal(err)
	}
	if setup != nil {
		setup(r)
	}
	if err := gbp.CheckDeclarations(v, r.facts(), r.day); err != nil {
		t.Fatalf("start-up: %v", err)
	}
	if err := v.StartUp(r.facts(), r.day); err != nil {
		t.Fatalf("start-up: %v", err)
	}
	if r.account, err = r.Ledger.CreateAccount(path, "GBP", -2, 0); err != nil {
		t.Fatal(err)
	}
	r.run(func(f gbp.Facts) (gbp.Intents, error) { return v.Open(f) })
	return r
}

// SetBank writes a bank-scoped setting effective from a day.
func (r *Runner) SetBank(key, text string, effective time.Time) {
	r.set(gbp.ScopeBank, key, text, effective)
}

// SetVersion writes a version-scoped setting effective from a day: a rate
// change decided for the product.
func (r *Runner) SetVersion(key, text string, effective time.Time) {
	r.set(gbp.ScopeVersion, key, text, effective)
}

func (r *Runner) set(scope gbp.Scope, key, text string, effective time.Time) {
	r.settings[scope][key] = append(r.settings[scope][key], dated{text, startOfDay(effective)})
}

// Day is the day the runner is on.
func (r *Runner) Day() time.Time { return r.day }

// AccountID is the ledger account's ID.
func (r *Runner) AccountID() string { return r.account.ID }

// Deposit pays amount into the account on the runner's day, or fails the test.
func (r *Runner) Deposit(amount luca.Amount) {
	r.t.Helper()
	if err := r.TryDeposit(amount); err != nil {
		r.t.Fatal(err)
	}
}

// Withdraw takes amount out of the account on the runner's day, or fails the test.
func (r *Runner) Withdraw(amount luca.Amount) {
	r.t.Helper()
	if err := r.TryWithdraw(amount); err != nil {
		r.t.Fatal(err)
	}
}

// TryDeposit is Deposit returning the rule's refusal, if any.
func (r *Runner) TryDeposit(amount luca.Amount) error {
	if r.version.Family() == gbp.FamilySavings {
		return r.post(r.account, r.cash, amount, "Deposit")
	}
	return r.post(r.cash, r.account, amount, "Deposit")
}

// TryWithdraw is Withdraw returning the rule's refusal, if any.
func (r *Runner) TryWithdraw(amount luca.Amount) error {
	if r.version.Family() == gbp.FamilySavings {
		return r.post(r.cash, r.account, amount, "Withdrawal")
	}
	return r.post(r.account, r.cash, amount, "Withdrawal")
}

// post asks the version, posts, then runs its post-posting rule.
func (r *Runner) post(from, to *luca.Account, amount luca.Amount, description string) error {
	r.t.Helper()
	delta := amount
	if from == r.account {
		delta = -amount
	}
	m := gbp.Movement{Delta: delta, Code: luca.CodeBookTransfer, Description: description, ValueTime: r.day}
	r.Reads = nil
	if err := r.version.PrePosting(r.facts(), m); err != nil {
		if gbp.IsRefusal(err) {
			return err
		}
		r.t.Fatalf("pre-posting: %v", err)
	}
	if _, err := r.Ledger.RecordMovement(from.ID, to.ID, amount, m.Code, m.ValueTime, m.Description); err != nil {
		r.t.Fatal(err)
	}
	r.run(func(f gbp.Facts) (gbp.Intents, error) { return r.version.PostPosting(f, m) })
	return nil
}

// Advance runs the Day event for the next n days: the runner's day first
// if it has not run yet, then each day after.
func (r *Runner) Advance(n int) {
	r.t.Helper()
	for i := 0; i < n; i++ {
		if !r.processed.IsZero() {
			r.day = r.processed.AddDate(0, 0, 1)
		}
		r.run(func(f gbp.Facts) (gbp.Intents, error) { return r.version.Day(f) })
		r.processed = r.day
	}
}

// AdvanceTo runs the Day event through day.
func (r *Runner) AdvanceTo(day time.Time) {
	r.t.Helper()
	for r.processed.Before(startOfDay(day)) {
		r.Advance(1)
	}
}

// Command runs one of the version's manual events.
func (r *Runner) Command(name string) {
	r.t.Helper()
	r.run(func(f gbp.Facts) (gbp.Intents, error) { return r.version.Command(f, name) })
}

// Close closes the account.
func (r *Runner) Close() {
	r.t.Helper()
	r.run(func(f gbp.Facts) (gbp.Intents, error) { return r.version.Close(f) })
}

// Balance is the account's balance now.
func (r *Runner) Balance() luca.Amount {
	r.t.Helper()
	b, err := r.Ledger.Balance(r.account.ID)
	if err != nil {
		r.t.Fatal(err)
	}
	return b
}

// Position is the account's stored position for day.
func (r *Runner) Position(day time.Time) (luca.Position, bool) {
	return r.facts().Position(day)
}

// Parameter resolves a parameter for the runner's day, as a rule would.
func (r *Runner) Parameter(key string) (gbp.Param, error) {
	return r.facts().Parameter(key, r.day)
}

// Export is the ledger as .goluca.
func (r *Runner) Export() string {
	r.t.Helper()
	return exportGoluca(r.t, r.Ledger)
}

// run runs one rule and carries out its intents.
func (r *Runner) run(rule func(gbp.Facts) (gbp.Intents, error)) {
	r.t.Helper()
	r.Reads = nil
	in, err := rule(r.facts())
	if err != nil {
		r.t.Fatalf("%s: %v", r.day.Format("2006-01-02"), err)
	}
	for _, s := range in.Settings {
		r.set(gbp.ScopeAccount, s.Key, s.Text, s.Effective)
	}
	for _, p := range in.Postings {
		counter := r.ensure(p.Counterparty)
		if _, err := r.Ledger.RecordMovement(counter.ID, r.account.ID, p.Amount, p.Code, p.ValueTime, p.Description); err != nil {
			r.t.Fatal(err)
		}
	}
	for _, p := range in.Positions {
		if _, err := r.Ledger.Project(p.AccountID, p.Day, p.Accrued); err != nil {
			r.t.Fatal(err)
		}
	}
}

func (r *Runner) ensure(path string) *luca.Account {
	r.t.Helper()
	a, err := r.Ledger.GetAccount(path)
	if err != nil {
		r.t.Fatal(err)
	}
	if a != nil {
		return a
	}
	if a, err = r.Ledger.CreateAccount(path, "GBP", -2, 0); err != nil {
		r.t.Fatal(err)
	}
	return a
}

func (r *Runner) facts() gbp.Facts { return facts{r} }

// facts is the runner's Facts: the ledger for positions and balance, the
// settings by scope for parameters (ADR-0006, parameter resolution).
type facts struct{ r *Runner }

func (f facts) Account() string { return f.r.account.ID }
func (f facts) Day() time.Time  { return f.r.day }

func (f facts) Position(day time.Time) (luca.Position, bool) {
	p, err := f.r.Ledger.PositionAt(f.r.account.ID, startOfDay(day))
	if err != nil {
		f.r.t.Fatal(err)
	}
	if p == nil {
		return luca.Position{}, false
	}
	return *p, true
}

func (f facts) Balance() luca.Amount { return f.r.Balance() }

func (f facts) Parameter(key string, day time.Time) (gbp.Param, error) {
	p, err := f.resolve(key, startOfDay(day))
	if err == nil {
		f.r.Reads = append(f.r.Reads, p)
	}
	return p, err
}

func (f facts) resolve(key string, day time.Time) (gbp.Param, error) {
	d, ok := gbp.Declared(f.r.version, key)
	if !ok {
		return gbp.Param{}, fmt.Errorf("parameter %s: not declared by %s v%d", key, f.r.version.Product(), f.r.version.Version())
	}
	if d.Derived != nil {
		source, err := f.resolve(d.Derived.Source, day)
		if err != nil {
			return gbp.Param{}, err
		}
		return d.Parse(strconv.FormatInt(d.Derived.Apply(source.Bps()), 10))
	}
	var text string
	found := false
	for _, s := range f.r.settings[d.Scope][key] {
		if !s.effective.After(day) {
			text, found = s.text, true // later-effective settings come later in the list
		}
	}
	if !found && d.Scope == gbp.ScopeVersion && d.Published != "" {
		text, found = d.Published, true
	}
	if !found {
		return gbp.Param{}, fmt.Errorf("parameter %s: no %s setting on %s", key, d.Scope, day.Format("2006-01-02"))
	}
	return d.Parse(text)
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

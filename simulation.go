package gbp

import (
	"fmt"
	"io"
	"math"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
)

// AccountUpdate captures the state change for one account on one day.
type AccountUpdate struct {
	Account        *ManagedAccount
	Date           time.Time
	OpeningBalance luca.Amount
	ClosingBalance luca.Amount
	InterestAmount luca.Amount // interest applied this day (the balance change over the day's rules)
	Exponent       int
	// AccruedDelta is the interest accrued this day in AccrualDenominator
	// numerator units (minor units = AccruedDelta / AccrualDenominator).
	AccruedDelta int64
	// AccruedNumerator is the accrued-but-unapplied accumulator after this
	// day, in the same numerator units.
	AccruedNumerator int64
}

// DailyUpdate collects all account updates for a single processing day.
type DailyUpdate struct {
	Date     time.Time
	Accounts []AccountUpdate
}

// DailyUpdateHandler is called after each day's processing.
type DailyUpdateHandler func(update DailyUpdate)

// Simulation is the core engine that advances time and dispatches events to features.
type Simulation struct {
	Ledger             luca.Ledger
	Clock              Clock
	Params             *ParameterStore
	products           map[string]*Product
	dispatch           map[string]map[EventType][]Feature // productID → eventType → features
	accounts           map[string]*ManagedAccount
	startDate          time.Time
	lastProcessedDate  time.Time
	dailyUpdateHandler DailyUpdateHandler

	// PaceHook, when set, is called after each account is processed during
	// end-of-day and end-of-month sweeps. Single-threaded hosts (WASM) can
	// yield to their event loop here to stay responsive during large sweeps.
	PaceHook func()
}

// NewSimulation creates a new simulation engine.
func NewSimulation(ledger luca.Ledger, clock Clock) (*Simulation, error) {
	return &Simulation{
		Ledger:    ledger,
		Clock:     clock,
		Params:    NewParameterStore(),
		products:  make(map[string]*Product),
		dispatch:  make(map[string]map[EventType][]Feature),
		accounts:  make(map[string]*ManagedAccount),
		startDate: startOfDay(clock.Now()),
	}, nil
}

// OnDailyUpdate registers a handler that receives daily account updates.
func (s *Simulation) OnDailyUpdate(handler DailyUpdateHandler) {
	s.dailyUpdateHandler = handler
}

// RegisterProduct registers a product and builds its dispatch table.
func (s *Simulation) RegisterProduct(p *Product) {
	s.products[p.ID] = p
	dt := make(map[EventType][]Feature)
	for _, f := range p.Features {
		for _, et := range f.Handles() {
			dt[et] = append(dt[et], f)
		}
	}
	s.dispatch[p.ID] = dt
}

// OpenAccount creates a new managed account for a registered product.
func (s *Simulation) OpenAccount(productID, accountPath, currency string, exponent int, params map[string]string) (*ManagedAccount, error) {
	prod, ok := s.products[productID]
	if !ok {
		return nil, fmt.Errorf("unknown product: %s", productID)
	}

	rate, err := annualRate(prod, params)
	if err != nil {
		return nil, err
	}

	acct, err := s.Ledger.CreateAccount(accountPath, currency, exponent, rate)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	now := s.Clock.Now()
	ma := s.manage(acct, prod, rate, StatusPending, now, params)

	// Dispatch AccountOpened event.
	ctx := &SimContext{Sim: s, Params: s.Params, Clock: s.Clock, AsOfDate: now}
	event := AccountOpenedEvent{
		EventHeader: EventHeader{Type: EventAccountOpened, Date: now, Account: ma},
		Params:      params,
	}
	if err := s.dispatchEvent(productID, EventAccountOpened, func(f Feature) error {
		if h, ok := f.(OnAccountOpened); ok {
			return h.HandleAccountOpened(ctx, event)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("account opened: %w", err)
	}

	return ma, nil
}

// Adoption is an account that already exists in the ledger, as the caller
// read it, for the engine to manage from now on: a bank restarting over a
// stored ledger brings its accounts back this way.
type Adoption struct {
	Account   *luca.Account
	ProductID string
	Status    AccountStatus
	OpenedAt  time.Time
	Balance   luca.Amount // the account's ledger balance now, which the engine then caches
	Params    map[string]string
}

// AdoptAccount registers an existing ledger account as a managed account of
// a registered product. Unlike OpenAccount it creates nothing in the ledger
// and dispatches no AccountOpened event: the account was opened in an
// earlier run. Accrued-but-unapplied interest is the caller's to restore
// on the returned account.
func (s *Simulation) AdoptAccount(a Adoption) (*ManagedAccount, error) {
	prod, ok := s.products[a.ProductID]
	if !ok {
		return nil, fmt.Errorf("unknown product: %s", a.ProductID)
	}
	if a.Account == nil {
		return nil, fmt.Errorf("adopt account: no ledger account")
	}
	rate, err := annualRate(prod, a.Params)
	if err != nil {
		return nil, err
	}
	ma := s.manage(a.Account, prod, rate, a.Status, a.OpenedAt, a.Params)
	ma.CachedBalance = a.Balance
	return ma, nil
}

// annualRate is the account's annual interest rate: the annual_rate param,
// else the product default, else zero.
func annualRate(prod *Product, params map[string]string) (float64, error) {
	if v, ok := params["annual_rate"]; ok {
		rate, err := parseFloat(v)
		if err != nil {
			return 0, fmt.Errorf("invalid annual_rate: %w", err)
		}
		return rate, nil
	}
	if v, ok := prod.Defaults["annual_rate"]; ok {
		rate, err := parseFloat(v)
		if err != nil {
			return 0, fmt.Errorf("invalid default annual_rate: %w", err)
		}
		return rate, nil
	}
	return 0, nil
}

// manage puts a ledger account under the engine's management on a product,
// recording its parameters (product defaults first, then overrides) as of
// the engine clock.
func (s *Simulation) manage(acct *luca.Account, prod *Product, rate float64, status AccountStatus, openedAt time.Time, params map[string]string) *ManagedAccount {
	ma := &ManagedAccount{
		Account:   acct,
		ProductID: prod.ID,
		Family:    prod.Family,
		Status:    status,
		OpenedAt:  openedAt,
		RateBps:   int64(math.Round(rate * 10_000)), // Round handles negative (Japan-style) rates too
	}
	s.accounts[acct.ID] = ma
	now := s.Clock.Now()
	for k, v := range prod.Defaults {
		s.Params.Set(acct.ID, k, v, now)
	}
	for k, v := range params {
		s.Params.Set(acct.ID, k, v, now)
	}
	return ma
}

// GetManagedAccount returns a managed account by its ledger account ID.
func (s *Simulation) GetManagedAccount(accountID string) (*ManagedAccount, bool) {
	ma, ok := s.accounts[accountID]
	return ma, ok
}

// Deposit records a deposit into an account.
func (s *Simulation) Deposit(accountID string, amount luca.Amount, fromPath, code string) error {
	ma, ok := s.accounts[accountID]
	if !ok {
		return fmt.Errorf("unknown account: %s", accountID)
	}

	now := s.Clock.Now()
	ctx := &SimContext{Sim: s, Params: s.Params, Clock: s.Clock, AsOfDate: now}
	event := DepositReceivedEvent{
		EventHeader: EventHeader{Type: EventDepositReceived, Date: now, Account: ma},
		Amount:      amount,
		FromPath:    fromPath,
		Code:        code,
	}

	return s.dispatchEvent(ma.ProductID, EventDepositReceived, func(f Feature) error {
		if h, ok := f.(OnDepositReceived); ok {
			return h.HandleDepositReceived(ctx, event)
		}
		return nil
	})
}

// Withdraw records a withdrawal from an account.
func (s *Simulation) Withdraw(accountID string, amount luca.Amount, toPath, code string) error {
	ma, ok := s.accounts[accountID]
	if !ok {
		return fmt.Errorf("unknown account: %s", accountID)
	}

	now := s.Clock.Now()
	ctx := &SimContext{Sim: s, Params: s.Params, Clock: s.Clock, AsOfDate: now}
	event := WithdrawalRequestedEvent{
		EventHeader: EventHeader{Type: EventWithdrawalRequested, Date: now, Account: ma},
		Amount:      amount,
		ToPath:      toPath,
		Code:        code,
	}

	return s.dispatchEvent(ma.ProductID, EventWithdrawalRequested, func(f Feature) error {
		if h, ok := f.(OnWithdrawalRequested); ok {
			return h.HandleWithdrawalRequested(ctx, event)
		}
		return nil
	})
}

// AdvanceToDate processes each unprocessed day up to and including targetDate.
func (s *Simulation) AdvanceToDate(target time.Time) ([]DailyUpdate, error) {
	targetDay := startOfDay(target)
	var updates []DailyUpdate

	current := s.lastProcessedDate
	if current.IsZero() {
		current = s.startDate
	} else {
		current = nextDay(current)
	}

	for !current.After(targetDay) {
		update, err := s.processEndOfDay(current)
		if err != nil {
			return updates, fmt.Errorf("process end of day %s: %w", current.Format("2006-01-02"), err)
		}
		updates = append(updates, update)
		if s.dailyUpdateHandler != nil {
			s.dailyUpdateHandler(update)
		}
		s.lastProcessedDate = current

		// Check for end-of-month.
		tomorrow := nextDay(current)
		if current.Month() != tomorrow.Month() {
			if err := s.processEndOfMonth(current); err != nil {
				return updates, fmt.Errorf("process end of month %s: %w", current.Format("2006-01-02"), err)
			}
		}

		current = tomorrow
	}

	return updates, nil
}

// CloseAccount transitions an account to closed.
func (s *Simulation) CloseAccount(accountID string) error {
	ma, ok := s.accounts[accountID]
	if !ok {
		return fmt.Errorf("unknown account: %s", accountID)
	}

	now := s.Clock.Now()
	ctx := &SimContext{Sim: s, Params: s.Params, Clock: s.Clock, AsOfDate: now}
	event := AccountClosedEvent{
		EventHeader: EventHeader{Type: EventAccountClosed, Date: now, Account: ma},
	}

	return s.dispatchEvent(ma.ProductID, EventAccountClosed, func(f Feature) error {
		if h, ok := f.(OnAccountClosed); ok {
			return h.HandleAccountClosed(ctx, event)
		}
		return nil
	})
}

// ExportGoluca writes the ledger state as a .goluca file.
func (s *Simulation) ExportGoluca(w io.Writer) error {
	return s.Ledger.Export(w)
}

// processEndOfDay runs end-of-day for all active accounts. Balances come from
// the per-account cache, so a sweep issues no balance queries; features that
// only accrue in memory (InterestAccrual) make the whole sweep query-free.
func (s *Simulation) processEndOfDay(date time.Time) (DailyUpdate, error) {
	update := DailyUpdate{Date: date}

	for _, ma := range s.accounts {
		if ma.Status != StatusActive {
			continue
		}

		preBalance := ma.CachedBalance
		preAccrued := ma.AccruedNumerator

		ctx := &SimContext{Sim: s, Params: s.Params, Clock: s.Clock, AsOfDate: date}
		event := EndOfDayEvent{
			EventHeader: EventHeader{Type: EventEndOfDay, Date: date, Account: ma},
		}

		if err := s.dispatchEvent(ma.ProductID, EventEndOfDay, func(f Feature) error {
			if h, ok := f.(OnEndOfDay); ok {
				return h.HandleEndOfDay(ctx, event)
			}
			return nil
		}); err != nil {
			return update, fmt.Errorf("end of day for %s: %w", ma.Account.ID, err)
		}

		update.Accounts = append(update.Accounts, AccountUpdate{
			Account:        ma,
			Date:           date,
			OpeningBalance: preBalance,
			ClosingBalance: ma.CachedBalance,
			InterestAmount: ma.CachedBalance - preBalance,
			Exponent:       ma.Account.Exponent,
			// The day's accrual alone: application moves whole minor units
			// out of the accumulator and into the balance, so add them back.
			AccruedDelta:     ma.AccruedNumerator + int64(ma.CachedBalance-preBalance)*AccrualDenominator - preAccrued,
			AccruedNumerator: ma.AccruedNumerator,
		})
		if s.PaceHook != nil {
			s.PaceHook()
		}
	}

	return update, nil
}

// processEndOfMonth dispatches EndOfMonth to all active accounts.
func (s *Simulation) processEndOfMonth(date time.Time) error {
	for _, ma := range s.accounts {
		if ma.Status != StatusActive {
			continue
		}
		ctx := &SimContext{Sim: s, Params: s.Params, Clock: s.Clock, AsOfDate: date}
		event := EndOfMonthEvent{
			EventHeader: EventHeader{Type: EventEndOfMonth, Date: date, Account: ma},
		}
		if err := s.dispatchEvent(ma.ProductID, EventEndOfMonth, func(f Feature) error {
			if h, ok := f.(OnEndOfMonth); ok {
				return h.HandleEndOfMonth(ctx, event)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("end of month for %s: %w", ma.Account.ID, err)
		}
		if s.PaceHook != nil {
			s.PaceHook()
		}
	}
	return nil
}

// dispatchEvent calls fn for each feature registered for the given event type.
func (s *Simulation) dispatchEvent(productID string, eventType EventType, fn func(Feature) error) error {
	dt, ok := s.dispatch[productID]
	if !ok {
		return nil
	}
	features, ok := dt[eventType]
	if !ok {
		return nil
	}
	for _, f := range features {
		if err := fn(f); err != nil {
			return err
		}
	}
	return nil
}

// RecordMovement records a ledger movement (exposed for features) and keeps
// the cached balances of any managed accounts involved in sync.
func (s *Simulation) RecordMovement(fromID, toID string, amount luca.Amount, code string, valueTime time.Time, description string) (*luca.Movement, error) {
	m, err := s.Ledger.RecordMovement(fromID, toID, amount, code, valueTime, description)
	if err != nil {
		return nil, err
	}
	if ma, ok := s.accounts[fromID]; ok {
		ma.CachedBalance -= amount
	}
	if ma, ok := s.accounts[toID]; ok {
		ma.CachedBalance += amount
	}
	return m, nil
}

// RefreshBalances reloads every managed account's cached balance from the
// ledger. Call after mutating the ledger outside RecordMovement (e.g. import).
func (s *Simulation) RefreshBalances() error {
	for id, ma := range s.accounts {
		bal, err := s.Ledger.Balance(id)
		if err != nil {
			return fmt.Errorf("refresh balance for %s: %w", id, err)
		}
		ma.CachedBalance = bal
	}
	return nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func nextDay(t time.Time) time.Time {
	return startOfDay(t).AddDate(0, 0, 1)
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

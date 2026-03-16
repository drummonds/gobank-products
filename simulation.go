package gbp

import (
	"fmt"
	"io"
	"time"

	luca "github.com/drummonds/go-luca"
)

// AccountUpdate captures the state change for one account on one day.
type AccountUpdate struct {
	Account        *ManagedAccount
	Date           time.Time
	OpeningBalance luca.Amount
	ClosingBalance luca.Amount
	InterestAmount luca.Amount
	Exponent       int
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
}

// NewSimulation creates a new simulation engine.
func NewSimulation(ledger luca.Ledger, clock Clock) (*Simulation, error) {
	if err := ledger.EnsureInterestAccounts(); err != nil {
		return nil, fmt.Errorf("ensure interest accounts: %w", err)
	}
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

	// Determine annual interest rate from params or product defaults.
	rate := 0.0
	if v, ok := params["annual_rate"]; ok {
		var err error
		rate, err = parseFloat(v)
		if err != nil {
			return nil, fmt.Errorf("invalid annual_rate: %w", err)
		}
	} else if v, ok := prod.Defaults["annual_rate"]; ok {
		var err error
		rate, err = parseFloat(v)
		if err != nil {
			return nil, fmt.Errorf("invalid default annual_rate: %w", err)
		}
	}

	acct, err := s.Ledger.CreateAccount(accountPath, currency, exponent, rate)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	ma := &ManagedAccount{
		Account:   acct,
		ProductID: productID,
		Status:    StatusPending,
		OpenedAt:  s.Clock.Now(),
	}
	s.accounts[acct.ID] = ma

	// Store all params (product defaults first, then overrides).
	now := s.Clock.Now()
	for k, v := range prod.Defaults {
		s.Params.Set(acct.ID, k, v, now)
	}
	for k, v := range params {
		s.Params.Set(acct.ID, k, v, now)
	}

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

// processEndOfDay runs end-of-day for all active accounts.
func (s *Simulation) processEndOfDay(date time.Time) (DailyUpdate, error) {
	eod := endOfDay(date)
	update := DailyUpdate{Date: date}

	for _, ma := range s.accounts {
		if ma.Status != StatusActive {
			continue
		}

		preBalance, err := s.Ledger.BalanceAt(ma.Account.ID, eod)
		if err != nil {
			return update, fmt.Errorf("pre-balance for %s: %w", ma.Account.ID, err)
		}

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

		postBalance, err := s.Ledger.Balance(ma.Account.ID)
		if err != nil {
			return update, fmt.Errorf("post-balance for %s: %w", ma.Account.ID, err)
		}

		update.Accounts = append(update.Accounts, AccountUpdate{
			Account:        ma,
			Date:           date,
			OpeningBalance: preBalance,
			ClosingBalance: postBalance,
			InterestAmount: postBalance - preBalance,
			Exponent:       ma.Account.Exponent,
		})
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

// RecordMovement records a ledger movement (exposed for features).
func (s *Simulation) RecordMovement(fromID, toID string, amount luca.Amount, code string, valueTime time.Time, description string) (*luca.Movement, error) {
	return s.Ledger.RecordMovement(fromID, toID, amount, code, valueTime, description)
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

func nextDay(t time.Time) time.Time {
	return startOfDay(t).AddDate(0, 0, 1)
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

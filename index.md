# gobank-products

Composable banking product library for Go. Products are built from independently testable **features** driven by a **simulation engine** with controllable time and typed events.

## Product Catalog

### Savings

| Product | Default Rate | Features |
|---------|-------------|----------|
| [Easy Access](#easy-access) | 1.5% | Deposit, Withdrawal, Interest, Lifecycle |
| [Fixed Term](#fixed-term) | 4.0% | Deposit, Term Lock, Withdrawal, Interest, Lifecycle |
| [ISA](#isa) | 3.5% | ISA Allowance, Deposit, Withdrawal, Interest, Lifecycle |

### Lending

| Product | Default Rate | Features |
|---------|-------------|----------|
| [Personal Loan](#personal-loan) | 6.9% | Deposit, Interest, Repayment, Lifecycle |
| [Mortgage](#mortgage) | 4.5% | Deposit, Interest, Repayment, Lifecycle |
| [Overdraft](#overdraft) | 15.9% | Deposit, Overdraft Facility, Interest, Lifecycle |

---

## Events

Events drive all product behaviour. They fall into three categories.

### Customer-initiated events

These originate from external actions — a customer or operator triggers them.

| Event | Trigger | Simulation method |
|-------|---------|-------------------|
| `AccountOpened` | Customer applies for a product | `OpenAccount()` |
| `DepositReceived` | Inbound payment (FPS credit, book transfer, loan disbursement) | `Deposit()` |
| `WithdrawalRequested` | Outbound payment (FPS debit, book transfer, overdraft draw) | `Withdraw()` |
| `AccountClosed` | Customer or operator closes the account | `CloseAccount()` |

All customer-initiated events are currently **synchronous book transfers** — the simulation records a ledger movement immediately and returns success or an error. There is no model of payment failure, timeout, or asynchronous settlement.

In production, deposits and withdrawals arrive as external payment scheme messages (e.g. FPS Faster Payments). The complete payment lifecycle — initiation, submission, acceptance, rejection, timeout, reconciliation — is out of scope for gobank-products and should be handled by [mock-fps](https://codeberg.org/hum3/mock-fps). See [External payments](#external-payments) below.

### System-scheduled events

Fired automatically by the simulation engine during time advancement.

| Event | When fired | Fired by |
|-------|-----------|----------|
| `EndOfDay` | Once per calendar day for every active account | `AdvanceToDate()` |
| `EndOfMonth` | When the processing date crosses a month boundary | `AdvanceToDate()` |

### Derived events

Fired by features in response to other events — not directly triggered by the simulation API.

| Event | Fired by | When |
|-------|----------|------|
| `MaturityReached` | `TermLock.HandleEndOfDay` | On the calendar date matching `maturity_date` |
| `RateChanged` | (not yet implemented) | When `annual_rate` parameter changes |

---

## Features

Features are composable building blocks. Each implements the `Feature` interface and one or more typed event handler interfaces. Features are dispatched in the order they appear in the product definition — earlier features can validate and reject before later features execute.

### Feature interface

```go
type Feature interface {
    Name() string
    Handles() []EventType
}
```

A feature declares which events it handles. For each event type, it implements the corresponding typed handler interface (e.g. `OnDepositReceived`, `OnEndOfDay`). Features are stateless — all state lives in the ledger and parameter store.

### StatusLifecycle

Manages account status transitions: Pending -> Active -> Closed.

| Event | Behaviour |
|-------|-----------|
| `AccountOpened` | Sets status to `Active` |
| `AccountClosed` | Sets status to `Closed`, records `ClosedAt` timestamp |

### DepositAcceptance

Validates and records deposit movements.

| Event | Behaviour |
|-------|-----------|
| `DepositReceived` | Validates amount > 0 and status = Active, records ledger movement from source to account |

### WithdrawalProcessing

Validates balance and records withdrawal movements.

| Event | Behaviour |
|-------|-----------|
| `WithdrawalRequested` | Validates amount > 0, status = Active, balance >= amount, records ledger movement from account to destination |

### InterestAccrual

Delegates daily interest calculation to go-luca.

| Event | Behaviour |
|-------|-----------|
| `EndOfDay` | Calls `CalculateDailyInterest(accountID, date)` — computes `balance * annual_rate / 365` and records the interest movement |

### TermLock

Blocks withdrawals before a maturity date and fires maturity events.

| Event | Behaviour |
|-------|-----------|
| `WithdrawalRequested` | Returns error if `AsOfDate < maturity_date` |
| `EndOfDay` | On the maturity date, dispatches a `MaturityReached` event |

### ISAWrapper

Enforces an annual deposit allowance.

| Event | Behaviour |
|-------|-----------|
| `DepositReceived` | Checks `isa_deposited + amount <= isa_allowance`. Updates running total on success; returns error on breach (runs before `DepositAcceptance`) |

### OverdraftFacility

Allows negative balance up to a limit and records the withdrawal. Replaces `WithdrawalProcessing` for overdraft products.

| Event | Behaviour |
|-------|-----------|
| `WithdrawalRequested` | Validates `balance - amount >= -overdraft_limit`, records ledger movement |

### RepaymentSchedule

Automatic monthly repayments for lending products.

| Event | Behaviour |
|-------|-----------|
| `EndOfMonth` | Records a movement of `monthly_repayment` from `repayment_source` into the loan account |

---

## Products

Each product composes features into a feature chain. The chain defines the complete event dispatch for that product.

### Event matrix

The following table shows which events each product responds to, and which feature handles them. Features are dispatched left-to-right — earlier features validate before later features execute.

| Event | Easy Access | Fixed Term | ISA | Personal Loan | Mortgage | Overdraft |
|-------|-------------|------------|-----|---------------|----------|-----------|
| `AccountOpened` | Lifecycle | Lifecycle | Lifecycle | Lifecycle | Lifecycle | Lifecycle |
| `DepositReceived` | Deposit | Deposit | **ISA** -> Deposit | Deposit | Deposit | Deposit |
| `WithdrawalRequested` | Withdrawal | **TermLock** -> Withdrawal | Withdrawal | — | — | **Overdraft** |
| `EndOfDay` | Interest | **TermLock** -> Interest | Interest | Interest | Interest | Interest |
| `EndOfMonth` | — | — | — | Repayment | Repayment | — |
| `MaturityReached` | — | *(derived)* | — | — | — | — |
| `AccountClosed` | Lifecycle | Lifecycle | Lifecycle | Lifecycle | Lifecycle | Lifecycle |

**Bold** = feature that gates or modifies the event before the standard handler runs.

Key observations:
- Lending products (Personal Loan, Mortgage) have no `WithdrawalRequested` handler — repayment is the only way funds leave the account.
- Overdraft uses `OverdraftFacility` instead of `WithdrawalProcessing` — it allows negative balances.
- ISA and Fixed Term add validation features *before* the standard handler in the dispatch chain.

### Easy Access

Instant-access savings account. Deposits and withdrawals at any time with daily interest accrual.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `WithdrawalProcessing` -> `InterestAccrual`

**Defaults**: `annual_rate: 0.015`

### Fixed Term

Higher-rate savings locked until a maturity date.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `TermLock` -> `WithdrawalProcessing` -> `InterestAccrual`

**Defaults**: `annual_rate: 0.040`

**Parameters**: `maturity_date` (format `2006-01-02`) — required at account opening.

TermLock runs before WithdrawalProcessing and rejects any withdrawal before the maturity date. On the maturity date, TermLock fires `MaturityReached` during EndOfDay processing.

### ISA

Tax-free savings with an annual deposit allowance.

**Feature chain**: `StatusLifecycle` -> `ISAWrapper` -> `DepositAcceptance` -> `WithdrawalProcessing` -> `InterestAccrual`

**Defaults**: `annual_rate: 0.035`, `isa_allowance: 2000000` (£20,000)

ISAWrapper runs before DepositAcceptance and tracks a running total (`isa_deposited`). Deposits exceeding the allowance are rejected before reaching DepositAcceptance.

### Personal Loan

Unsecured lending. Interest accrues on the outstanding balance; monthly repayments reduce it.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `InterestAccrual` -> `RepaymentSchedule`

**Defaults**: `annual_rate: 0.069`

**Parameters**: `monthly_repayment` (amount in minor units), `repayment_source` (ledger account path).

The loan is disbursed as a `Deposit` into the asset account. No withdrawal handler — funds leave only via scheduled repayments.

### Mortgage

Residential mortgage. Same mechanics as Personal Loan with a different default rate.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `InterestAccrual` -> `RepaymentSchedule`

**Defaults**: `annual_rate: 0.045`

**Parameters**: Same as Personal Loan — `monthly_repayment`, `repayment_source`.

### Overdraft

Arranged overdraft facility on a current account.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `OverdraftFacility` -> `InterestAccrual`

**Defaults**: `annual_rate: 0.159`, `overdraft_limit: 100000` (£1,000)

OverdraftFacility replaces WithdrawalProcessing — it permits the balance to go negative down to `-overdraft_limit` and records the movement itself.

---

## Simulation API

### Core types

```go
type Simulation struct {
    Ledger  luca.Ledger      // double-entry ledger (go-luca)
    Clock   Clock            // wall clock or sim clock
    Params  *ParameterStore  // time-varying per-account parameters
}

type SimContext struct {
    Sim      *Simulation
    Params   *ParameterStore
    Clock    Clock
    AsOfDate time.Time
}

type ManagedAccount struct {
    Account   *luca.Account
    ProductID string
    Status    AccountStatus   // Pending | Active | PendingClosure | Closed
    OpenedAt  time.Time
    ClosedAt  time.Time
}
```

### Lifecycle

```go
// Create engine with a ledger and clock.
sim, err := gbp.NewSimulation(ledger, clock)

// Register products before opening accounts.
sim.RegisterProduct(gbp.EasyAccess())
sim.RegisterProduct(gbp.FixedTerm())

// Open an account — fires AccountOpened through the feature chain.
ma, err := sim.OpenAccount("easy-access", "Liability:Savings:alice", "GBP", -2, nil)

// Customer actions — fire events through the feature chain.
err = sim.Deposit(ma.Account.ID, 100000, equityID, luca.CodeBookTransfer)
err = sim.Withdraw(ma.Account.ID, 50000, equityID, luca.CodeBookTransfer)

// Advance time — fires EndOfDay (and EndOfMonth at boundaries) for all active accounts.
updates, err := sim.AdvanceToDate(targetDate)

// Close — fires AccountClosed.
err = sim.CloseAccount(ma.Account.ID)

// Export ledger to .goluca format.
err = sim.ExportGoluca(w)
```

### Parameter store

Parameters are time-varying key-value pairs per account. Products set defaults at registration time; account-specific overrides are applied at opening. Features read parameters via `SimContext.Params`.

```go
// Set (internal — called by Simulation during OpenAccount)
store.Set(accountID, "annual_rate", "0.035", effectiveAt)

// Read (used by features)
value, ok := ctx.Params.Get(accountID, "maturity_date", ctx.AsOfDate)
rate, err := ctx.Params.GetFloat64(accountID, "annual_rate", ctx.AsOfDate)
```

Parameters can be updated over time — the store returns the most recent value effective at or before the query date.

### Clock

```go
type Clock interface {
    Now() time.Time
}

// Production: WallClock{}
// Testing:    NewSimClock(startDate) — with SetDate() and Advance()
```

### Daily updates

`AdvanceToDate` returns `[]DailyUpdate` — one per processed day, each containing per-account opening/closing balances and interest amounts.

```go
type DailyUpdate struct {
    Date     time.Time
    Accounts []AccountUpdate
}

type AccountUpdate struct {
    Account        *ManagedAccount
    Date           time.Time
    OpeningBalance luca.Amount
    ClosingBalance luca.Amount
    InterestAmount luca.Amount
    Exponent       int
}
```

---

## Building a new product

### 1. Define the feature chain

A product is a list of features with default parameters. Feature order matters — features earlier in the list run first and can reject events before later features see them.

```go
func MyProduct() *Product {
    return &Product{
        ID:     "my-product",
        Name:   "My Product",
        Family: FamilySavings, // or FamilyLending
        Features: []Feature{
            StatusLifecycle{},     // always first — manages Pending/Active/Closed
            DepositAcceptance{},   // validates and records deposits
            WithdrawalProcessing{},// validates and records withdrawals
            InterestAccrual{},     // daily interest — usually last
        },
        Defaults: map[string]string{
            "annual_rate": "0.025",
        },
    }
}
```

### 2. Writing a custom feature

Implement `Feature` and one or more typed handler interfaces.

```go
type MyFeature struct{}

func (MyFeature) Name() string          { return "my_feature" }
func (MyFeature) Handles() []EventType  { return []EventType{EventDepositReceived} }

func (MyFeature) HandleDepositReceived(ctx *SimContext, e DepositReceivedEvent) error {
    // Validate, modify, or record.
    // Return error to reject and stop dispatch.
    return nil
}
```

Available handler interfaces:

| Interface | Method | Event |
|-----------|--------|-------|
| `OnAccountOpened` | `HandleAccountOpened` | `AccountOpened` |
| `OnDepositReceived` | `HandleDepositReceived` | `DepositReceived` |
| `OnWithdrawalRequested` | `HandleWithdrawalRequested` | `WithdrawalRequested` |
| `OnEndOfDay` | `HandleEndOfDay` | `EndOfDay` |
| `OnEndOfMonth` | `HandleEndOfMonth` | `EndOfMonth` |
| `OnRateChanged` | `HandleRateChanged` | `RateChanged` |
| `OnMaturityReached` | `HandleMaturityReached` | `MaturityReached` |
| `OnAccountClosed` | `HandleAccountClosed` | `AccountClosed` |

### 3. Feature chain ordering conventions

1. **StatusLifecycle** — always first. Sets account to Active on open.
2. **Validation wrappers** — ISAWrapper, TermLock, etc. Gate events before the standard handler.
3. **Standard handlers** — DepositAcceptance, WithdrawalProcessing/OverdraftFacility. Record ledger movements.
4. **Post-processing** — InterestAccrual, RepaymentSchedule. Run after balances are settled.

### 4. Testing

Use the `testkit.ScenarioBuilder` for unit tests and `testkit.GolucaScenario` for golden-file regression tests.

```go
// Unit test
testkit.NewScenario(t).
    WithProduct(MyProduct()).
    OpenAccount("my-product", "Liability:Savings:test").
    Deposit("Liability:Savings:test", 100000).
    AdvanceDays(30).
    AssertBalanceRange("Liability:Savings:test", 100050, 100100)

// Golden-file test
scenario := testkit.GolucaScenario{
    Name:    "my_product_30d",
    Product: MyProduct(),
    Account: testkit.AccountSpec{Path: "Liability:Savings:test"},
    Actions: []testkit.Action{
        testkit.Deposit(100000),
        testkit.AdvanceDays(30),
    },
}
scenario.RunGolden(t)  // compare against testdata/my_product_30d.goluca
```

---

## External payments

Customer-initiated events (`DepositReceived`, `WithdrawalRequested`) currently execute as instantaneous book transfers — the simulation calls `RecordMovement` synchronously and either succeeds or returns an error. There is no concept of:

- **Payment initiation** — submitting a payment to an external scheme
- **Asynchronous settlement** — waiting for confirmation or rejection
- **Payment failure** — timeout, insufficient funds at the counterparty, scheme rejection
- **Reconciliation** — matching inbound scheme messages to expected transactions

In production, deposits arrive as inbound FPS (Faster Payments) credits and withdrawals are outbound FPS debits. The full payment lifecycle has states beyond "requested" and "completed" — a payment can be pending, submitted, accepted, rejected, timed out, or returned.

This payment scheme simulation belongs in [mock-fps](https://codeberg.org/hum3/mock-fps), not in gobank-products. The boundary is:

- **gobank-products**: product rules, interest, lifecycle, ledger movements (assumes payments succeed)
- **mock-fps**: payment scheme simulation, failure modes, async settlement, reconciliation

To integrate, gobank-products would need to:

1. Accept a payment outcome event (success/failure) rather than recording movements directly
2. Model a `PendingPayment` state between initiation and settlement
3. Handle payment reversals (e.g. FPS return after initial credit)

---

## Architecture

```
Product = [Feature, Feature, ...]
    | RegisterProduct
    v
Simulation builds dispatch table: EventType -> []Feature
    | Event occurs (deposit, EOD, etc.)
    v
Features called in order -> first error stops dispatch
    | Features interact with
    v
SimContext: Simulation, ParameterStore, Clock, AsOfDate
    | Features record via
    v
Ledger (go-luca): double-entry movements, interest calculation
```

Products are pure data — a name, family, list of features, and default parameters. Features are stateless handlers. All state lives in the ledger (balances, movements) and the parameter store (rates, dates, limits).

---

## Links

- **Documentation**: [h3-gobank-products.statichost.eu](https://h3-gobank-products.statichost.eu)
- **Source**: [codeberg.org/hum3/gobank-products](https://codeberg.org/hum3/gobank-products)
- **Mirror**: [github.com/drummonds/gobank-products](https://github.com/drummonds/gobank-products)

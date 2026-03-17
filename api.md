# Simulation API

[Home](index.html) | [Features](features.html) | [Products](products.html)

## Core types

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

## Lifecycle

```go
// Create engine with a ledger and clock.
sim, err := gbp.NewSimulation(ledger, clock)

// Register products before opening accounts.
sim.RegisterProduct(gbp.EasyAccess())
sim.RegisterProduct(gbp.FixedTerm())

// Open an account -- fires AccountOpened through the feature chain.
ma, err := sim.OpenAccount("easy-access", "Liability:Savings:alice", "GBP", -2, nil)

// Customer actions -- fire events through the feature chain.
err = sim.Deposit(ma.Account.ID, 100000, equityID, luca.CodeBookTransfer)
err = sim.Withdraw(ma.Account.ID, 50000, equityID, luca.CodeBookTransfer)

// Advance time -- fires EndOfDay (and EndOfMonth at boundaries) for all active accounts.
updates, err := sim.AdvanceToDate(targetDate)

// Close -- fires AccountClosed.
err = sim.CloseAccount(ma.Account.ID)

// Export ledger to .goluca format.
err = sim.ExportGoluca(w)
```

## Parameter store

Parameters are time-varying key-value pairs per account. Products set defaults at registration time; account-specific overrides are applied at opening. Features read parameters via `SimContext.Params`.

```go
// Set (internal -- called by Simulation during OpenAccount)
store.Set(accountID, "annual_rate", "0.035", effectiveAt)

// Read (used by features)
value, ok := ctx.Params.Get(accountID, "maturity_date", ctx.AsOfDate)
rate, err := ctx.Params.GetFloat64(accountID, "annual_rate", ctx.AsOfDate)
```

Parameters can be updated over time -- the store returns the most recent value effective at or before the query date.

### Standard parameters

| Parameter | Type | Used by | Description |
|-----------|------|---------|-------------|
| `annual_rate` | float64 | InterestAccrual | Annual interest rate (e.g. `0.015` for 1.5%) |
| `maturity_date` | date string | TermLock | Lock-up expiry date (format `2006-01-02`) |
| `isa_allowance` | int (minor units) | ISAWrapper | Annual deposit cap (default £20,000 = `2000000`) |
| `isa_deposited` | int (minor units) | ISAWrapper | Running total of deposits in current year |
| `monthly_repayment` | int (minor units) | RepaymentSchedule | Monthly repayment amount |
| `repayment_source` | account path | RepaymentSchedule | Ledger account to draw repayments from |
| `overdraft_limit` | int (minor units) | OverdraftFacility | Maximum negative balance (default £1,000 = `100000`) |

## Clock

```go
type Clock interface {
    Now() time.Time
}

// Production: WallClock{}
// Testing:    NewSimClock(startDate) -- with SetDate() and Advance()
```

## Daily updates

`AdvanceToDate` returns `[]DailyUpdate` -- one per processed day, each containing per-account opening/closing balances and interest amounts.

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

A product is a list of features with default parameters. Feature order matters -- features earlier in the list run first and can reject events before later features see them.

```go
func MyProduct() *Product {
    return &Product{
        ID:     "my-product",
        Name:   "My Product",
        Family: FamilySavings, // or FamilyLending
        Features: []Feature{
            StatusLifecycle{},     // always first
            DepositAcceptance{},   // validates and records deposits
            WithdrawalProcessing{},// validates and records withdrawals
            InterestAccrual{},     // daily interest -- usually last
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

### 3. Testing

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

Customer-initiated events (`DepositReceived`, `WithdrawalRequested`) currently execute as instantaneous book transfers. There is no concept of payment initiation, asynchronous settlement, payment failure, or reconciliation.

In production, deposits arrive as inbound FPS credits and withdrawals are outbound FPS debits. The full payment lifecycle belongs in [mock-fps](https://codeberg.org/hum3/mock-fps), not in gobank-products. The boundary is:

- **gobank-products**: product rules, interest, lifecycle, ledger movements (assumes payments succeed)
- **mock-fps**: payment scheme simulation, failure modes, async settlement, reconciliation

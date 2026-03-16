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

## Products

### Easy Access

Instant-access savings account. Deposits and withdrawals at any time with daily interest accrual.

**Features**: StatusLifecycle → DepositAcceptance → WithdrawalProcessing → InterestAccrual

**Defaults**: `annual_rate: 0.015`

**Behaviour**: Deposits credit the account immediately. Withdrawals are validated against current balance. Interest accrues daily at the annual rate / 365 and compounds.

### Fixed Term

Higher-rate savings locked until a maturity date. Withdrawals are blocked before maturity.

**Features**: StatusLifecycle → DepositAcceptance → TermLock → WithdrawalProcessing → InterestAccrual

**Defaults**: `annual_rate: 0.040`

**Parameters**: `maturity_date` (format `2006-01-02`) — must be set when opening the account.

**Behaviour**: TermLock rejects any withdrawal request before the maturity date. On the maturity date itself, a MaturityReached event is fired. After maturity, withdrawals proceed normally via WithdrawalProcessing.

### ISA

Tax-free savings with an annual deposit allowance. Operates like Easy Access but enforces a per-account deposit cap.

**Features**: StatusLifecycle → ISAWrapper → DepositAcceptance → WithdrawalProcessing → InterestAccrual

**Defaults**: `annual_rate: 0.035`, `isa_allowance: 2000000` (£20,000 in pence)

**Behaviour**: ISAWrapper runs before DepositAcceptance and tracks a running total (`isa_deposited`). Any deposit that would push the total above `isa_allowance` is rejected. Withdrawals and interest accrual work identically to Easy Access.

### Personal Loan

Unsecured lending product. Interest accrues on the outstanding balance and monthly repayments reduce it.

**Features**: StatusLifecycle → DepositAcceptance → InterestAccrual → RepaymentSchedule

**Defaults**: `annual_rate: 0.069`

**Parameters**: `monthly_repayment` (amount in minor units), `repayment_source` (ledger account path for the source of repayment funds).

**Behaviour**: The loan is disbursed as a deposit into the asset account. Interest accrues daily. At end-of-month, RepaymentSchedule records a movement from the repayment source into the loan account, reducing the outstanding balance.

### Mortgage

Residential mortgage. Same mechanics as Personal Loan but with different default rate and intended for secured lending.

**Features**: StatusLifecycle → DepositAcceptance → InterestAccrual → RepaymentSchedule

**Defaults**: `annual_rate: 0.045`

**Parameters**: Same as Personal Loan — `monthly_repayment`, `repayment_source`.

### Overdraft

Arranged overdraft facility on a current account. Allows the balance to go negative up to a configured limit.

**Features**: StatusLifecycle → DepositAcceptance → OverdraftFacility → InterestAccrual

**Defaults**: `annual_rate: 0.159`, `overdraft_limit: 100000` (£1,000 in pence)

**Behaviour**: OverdraftFacility replaces WithdrawalProcessing — it validates that `balance - withdrawal >= -overdraft_limit` and records the movement. Interest accrues on the full balance (including negative balances).

---

## Features

Features are composable building blocks. Each implements the `Feature` interface and one or more typed event handler interfaces. Features are dispatched in the order they appear in the product definition — earlier features can validate and reject before later features execute.

### StatusLifecycle

Manages account status transitions: Pending → Active → Closed.

| Event | Behaviour |
|-------|-----------|
| AccountOpened | Sets status to Active |
| AccountClosed | Sets status to Closed, records closure time |

### DepositAcceptance

Validates and records deposit movements.

| Event | Behaviour |
|-------|-----------|
| DepositReceived | Validates amount > 0 and status = Active, then records a ledger movement from the source account |

### WithdrawalProcessing

Validates balance and records withdrawal movements.

| Event | Behaviour |
|-------|-----------|
| WithdrawalRequested | Validates amount > 0, status = Active, and balance ≥ amount, then records a ledger movement to the destination account |

### InterestAccrual

Delegates daily interest calculation to go-luca's `CalculateDailyInterest`.

| Event | Behaviour |
|-------|-----------|
| EndOfDay | Calls `CalculateDailyInterest(accountID, date)` which computes `balance × annual_rate / 365` and records the interest movement |

### TermLock

Blocks withdrawals before a maturity date and fires maturity events.

| Event | Behaviour |
|-------|-----------|
| WithdrawalRequested | Returns error if `AsOfDate < maturity_date` |
| EndOfDay | On the maturity date, dispatches a MaturityReached event |

### ISAWrapper

Enforces an annual deposit allowance.

| Event | Behaviour |
|-------|-----------|
| DepositReceived | Checks `isa_deposited + amount ≤ isa_allowance`. If OK, updates the running total. If exceeded, returns error before DepositAcceptance runs |

### OverdraftFacility

Allows negative balance up to a limit and records the withdrawal.

| Event | Behaviour |
|-------|-----------|
| WithdrawalRequested | Validates `balance - amount ≥ -overdraft_limit`, then records the movement |

### RepaymentSchedule

Handles monthly repayment logic for lending products.

| Event | Behaviour |
|-------|-----------|
| EndOfMonth | Records a movement of `monthly_repayment` from `repayment_source` into the loan account |

---

## Simulation Engine

The simulation engine (`Simulation`) manages products, accounts, and time progression.

**Event dispatch**: When a product is registered, a dispatch table `map[EventType][]Feature` is built from its features. On each event, features are called in order — if any returns an error, dispatch stops immediately.

**Time advancement**: `AdvanceToDate(target)` processes each unprocessed day from the start date (or last processed date + 1) through the target date inclusive. Each day fires EndOfDay for all active accounts. End-of-month is detected automatically.

**Key methods**:

- `NewSimulation(ledger, clock)` — create engine
- `RegisterProduct(p)` — register a product and build dispatch tables
- `OpenAccount(productID, path, currency, exponent, params)` — create account, apply defaults + params, fire AccountOpened
- `Deposit(accountID, amount, fromPath, code)` — fire DepositReceived
- `Withdraw(accountID, amount, toPath, code)` — fire WithdrawalRequested
- `AdvanceToDate(target)` — process days, returns `[]DailyUpdate`
- `CloseAccount(accountID)` — fire AccountClosed
- `ExportGoluca(w)` — export ledger state

---

## Architecture

```
Product = [Feature, Feature, ...]
    ↓ RegisterProduct
Simulation builds dispatch table: EventType → []Feature
    ↓ Event occurs (deposit, EOD, etc.)
Features called in order → first error stops dispatch
    ↓ Features interact with
SimContext: Simulation, ParameterStore, Clock, AsOfDate
```

Products are pure data — a name, family, list of features, and default parameters. Features are stateless handlers. All state lives in the ledger (balances, movements) and the parameter store (rates, dates, limits).

---

## Links

- **Documentation**: [h3-gobank-products.statichost.eu](https://h3-gobank-products.statichost.eu)
- **Source**: [codeberg.org/hum3/gobank-products](https://codeberg.org/hum3/gobank-products)
- **Mirror**: [github.com/drummonds/gobank-products](https://github.com/drummonds/gobank-products)

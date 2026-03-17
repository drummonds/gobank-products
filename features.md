# Features

[Home](index.html) | [Products](products.html) | [API](api.html)

Features are composable building blocks. Each implements the `Feature` interface and one or more typed event handler interfaces. Features are dispatched in the order they appear in the product definition -- earlier features can validate and reject before later features execute.

## Feature interface

```go
type Feature interface {
    Name() string
    Handles() []EventType
}
```

A feature declares which events it handles. For each event type, it implements the corresponding typed handler interface (e.g. `OnDepositReceived`, `OnEndOfDay`). Features are stateless -- all state lives in the ledger and parameter store.

## Events

Events drive all product behaviour. They fall into three categories.

### Customer-initiated events

These originate from external actions -- a customer or operator triggers them.

| Event | Trigger | Simulation method |
|-------|---------|-------------------|
| `AccountOpened` | Customer applies for a product | `OpenAccount()` |
| `DepositReceived` | Inbound payment (FPS credit, book transfer, loan disbursement) | `Deposit()` |
| `WithdrawalRequested` | Outbound payment (FPS debit, book transfer, overdraft draw) | `Withdraw()` |
| `AccountClosed` | Customer or operator closes the account | `CloseAccount()` |

All customer-initiated events are currently **synchronous book transfers** -- the simulation records a ledger movement immediately and returns success or an error.

### System-scheduled events

Fired automatically by the simulation engine during time advancement.

| Event | When fired | Fired by |
|-------|-----------|----------|
| `EndOfDay` | Once per calendar day for every active account | `AdvanceToDate()` |
| `EndOfMonth` | When the processing date crosses a month boundary | `AdvanceToDate()` |

### Derived events

Fired by features in response to other events -- not directly triggered by the simulation API.

| Event | Fired by | When |
|-------|----------|------|
| `MaturityReached` | `TermLock.HandleEndOfDay` | On the calendar date matching `maturity_date` |
| `RateChanged` | (not yet implemented) | When `annual_rate` parameter changes |

---

## StatusLifecycle

Manages account status transitions: Pending -> Active -> Closed.

**Permitted events:**

| Event | Behaviour |
|-------|-----------|
| `AccountOpened` | Sets status to `Active` |
| `AccountClosed` | Sets status to `Closed`, records `ClosedAt` timestamp |

**Used by:** all products (always first in feature chain).

## DepositAcceptance

Validates and records deposit movements.

**Permitted events:**

| Event | Behaviour |
|-------|-----------|
| `DepositReceived` | Validates amount > 0 and status = Active, records ledger movement from source to account |

**Used by:** all products.

## WithdrawalProcessing

Validates balance and records withdrawal movements.

**Permitted events:**

| Event | Behaviour |
|-------|-----------|
| `WithdrawalRequested` | Validates amount > 0, status = Active, balance >= amount, records ledger movement from account to destination |

**Used by:** Easy Access, Fixed Term (after TermLock), ISA.

## InterestAccrual

Delegates daily interest calculation to go-luca.

**Permitted events:**

| Event | Behaviour |
|-------|-----------|
| `EndOfDay` | Calls `CalculateDailyInterest(accountID, date)` -- computes `balance * annual_rate / 365` and records the interest movement |

**Used by:** all products.

## TermLock

Blocks withdrawals before a maturity date and fires maturity events.

**Permitted events:**

| Event | Behaviour |
|-------|-----------|
| `WithdrawalRequested` | Returns error if `AsOfDate < maturity_date` |
| `EndOfDay` | On the maturity date, dispatches a `MaturityReached` event |

**Used by:** Fixed Term.

## ISAWrapper

Enforces an annual deposit allowance.

**Permitted events:**

| Event | Behaviour |
|-------|-----------|
| `DepositReceived` | Checks `isa_deposited + amount <= isa_allowance`. Updates running total on success; returns error on breach (runs before `DepositAcceptance`) |

**Used by:** ISA.

## OverdraftFacility

Allows negative balance up to a limit and records the withdrawal. Replaces `WithdrawalProcessing` for overdraft products.

**Permitted events:**

| Event | Behaviour |
|-------|-----------|
| `WithdrawalRequested` | Validates `balance - amount >= -overdraft_limit`, records ledger movement |

**Used by:** Overdraft.

## RepaymentSchedule

Automatic monthly repayments for lending products.

**Permitted events:**

| Event | Behaviour |
|-------|-----------|
| `EndOfMonth` | Records a movement of `monthly_repayment` from `repayment_source` into the loan account |

**Used by:** Personal Loan, Mortgage.

---

## Feature chain ordering conventions

1. **StatusLifecycle** -- always first. Sets account to Active on open.
2. **Validation wrappers** -- ISAWrapper, TermLock, etc. Gate events before the standard handler.
3. **Standard handlers** -- DepositAcceptance, WithdrawalProcessing/OverdraftFacility. Record ledger movements.
4. **Post-processing** -- InterestAccrual, RepaymentSchedule. Run after balances are settled.

## Handler interfaces

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

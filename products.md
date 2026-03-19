# Products

[Home](index.html) | [Features](features.html) | [Interest](interest.html) | [API](api.html)

Each product composes [features](features.html) into a feature chain. The chain defines the complete event dispatch for that product.

## Event dispatch matrix

The following table shows which events each product responds to, and which feature handles them. Features are dispatched left-to-right -- earlier features validate before later features execute.

| Event | Easy Access | Fixed Term | ISA | Personal Loan | Mortgage | Overdraft |
|-------|-------------|------------|-----|---------------|----------|-----------|
| `AccountOpened` | Lifecycle | Lifecycle | Lifecycle | Lifecycle | Lifecycle | Lifecycle |
| `DepositReceived` | Deposit | Deposit | **ISA** -> Deposit | Deposit | Deposit | Deposit |
| `WithdrawalRequested` | Withdrawal | **TermLock** -> Withdrawal | Withdrawal | -- | -- | **Overdraft** |
| `EndOfDay` | Interest | **TermLock** -> Interest | Interest | Interest | Interest | Interest |
| `EndOfMonth` | -- | -- | -- | Repayment | Repayment | -- |
| `MaturityReached` | -- | *(derived)* | -- | -- | -- | -- |
| `AccountClosed` | Lifecycle | Lifecycle | Lifecycle | Lifecycle | Lifecycle | Lifecycle |

**Bold** = feature that gates or modifies the event before the standard handler runs.

Key observations:

- Lending products (Personal Loan, Mortgage) have no `WithdrawalRequested` handler -- repayment is the only way funds leave the account.
- Overdraft uses `OverdraftFacility` instead of `WithdrawalProcessing` -- it allows negative balances.
- ISA and Fixed Term add validation features *before* the standard handler in the dispatch chain.

---

## Easy Access

Instant-access savings account. Deposits and withdrawals at any time with daily interest accrual.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `WithdrawalProcessing` -> `InterestAccrual`

**Defaults**: `annual_rate: 0.015` (1.5% p.a.)

### Golden goluca file: `easy_access_30d`

> Given an Easy Access savings account opened 2026-01-01
> And a deposit of £1,000.00
> When 30 days elapse to 2026-01-30
> Then daily interest accrues at 1.5% p.a.
> And the goluca export round-trips cleanly

```
2026-01-01 * Deposit
  Equity:Capital -> Liability:Savings:alice 1000 GBP

2026-01-01 * Daily interest for 2026-01-01
  Expense:Interest -> Liability:Savings:alice 0.04 GBP

2026-01-02 * Daily interest for 2026-01-02
  Expense:Interest -> Liability:Savings:alice 0.04 GBP

  ... (daily interest of 0.04 GBP continues through 2026-01-30) ...

2026-01-30 * Daily interest for 2026-01-30
  Expense:Interest -> Liability:Savings:alice 0.04 GBP
```

30 days at 1.5% on £1,000 = 30 x £0.04 = £1.20 total interest.

---

## Fixed Term

Higher-rate savings locked until a maturity date.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `TermLock` -> `WithdrawalProcessing` -> `InterestAccrual`

**Defaults**: `annual_rate: 0.040` (4.0% p.a.)

**Parameters**: `maturity_date` (format `2006-01-02`) -- required at account opening.

TermLock runs before WithdrawalProcessing and rejects any withdrawal before the maturity date. On the maturity date, TermLock fires `MaturityReached` during EndOfDay processing.

### Golden goluca file: `fixed_term_maturity`

> Given a Fixed Term account with maturity 2026-04-01
> And a deposit of £5,000.00
> When time advances past maturity to 2026-04-02
> And £5,000.00 is withdrawn
> Then interest at 4.0% has accrued over ~91 days
> And the withdrawal succeeds post-maturity

```
2026-01-01 * Deposit
  Equity:Capital -> Liability:Savings:fixed 5000 GBP

2026-01-01 * Daily interest for 2026-01-01
  Expense:Interest -> Liability:Savings:fixed 0.54 GBP

  ... (daily interest of 0.54 GBP through 2026-02-04) ...

2026-02-05 * Daily interest for 2026-02-05
  Expense:Interest -> Liability:Savings:fixed 0.55 GBP

  ... (daily interest of 0.55 GBP through 2026-04-02) ...

2026-04-02 * Withdrawal
  Liability:Savings:fixed -> Equity:Capital 5000 GBP

2026-04-02 * Daily interest for 2026-04-02
  Expense:Interest -> Liability:Savings:fixed 0.55 GBP
```

~91 days at 4.0% on £5,000. Daily interest rises from £0.54 to £0.55 as accrued interest compounds. Withdrawal of the original £5,000 succeeds only after the maturity date.

---

## ISA

Tax-free savings with an annual deposit allowance.

**Feature chain**: `StatusLifecycle` -> `ISAWrapper` -> `DepositAcceptance` -> `WithdrawalProcessing` -> `InterestAccrual`

**Defaults**: `annual_rate: 0.035` (3.5% p.a.), `isa_allowance: 2000000` (£20,000)

ISAWrapper runs before DepositAcceptance and tracks a running total (`isa_deposited`). Deposits exceeding the allowance are rejected before reaching DepositAcceptance.

### Golden goluca file: `isa_30d`

> Given an ISA account opened 2026-01-01
> And a deposit of £10,000.00 (within £20k allowance)
> When 30 days elapse
> Then daily interest accrues at 3.5% p.a.

```
2026-01-01 * Deposit
  Equity:Capital -> Liability:Savings:isa 10000 GBP

2026-01-01 * Daily interest for 2026-01-01
  Expense:Interest -> Liability:Savings:isa 0.95 GBP

  ... (daily interest of 0.95 GBP through 2026-01-13) ...

2026-01-14 * Daily interest for 2026-01-14
  Expense:Interest -> Liability:Savings:isa 0.96 GBP

  ... (daily interest of 0.96 GBP through 2026-01-30) ...
```

30 days at 3.5% on £10,000. Daily interest rises from £0.95 to £0.96 as accrued interest compounds.

---

## Personal Loan

Unsecured lending. Interest accrues on the outstanding balance; monthly repayments reduce it.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `InterestAccrual` -> `RepaymentSchedule`

**Defaults**: `annual_rate: 0.069` (6.9% p.a.)

**Parameters**: `monthly_repayment` (amount in minor units), `repayment_source` (ledger account path).

The loan is disbursed as a `Deposit` into the asset account. No withdrawal handler -- funds leave only via scheduled repayments.

### Golden goluca file: `personal_loan_30d`

> Given a Personal Loan account opened 2026-01-01
> And £5,000.00 disbursed
> When 30 days elapse
> Then daily interest accrues at 6.9% p.a.

```
2026-01-01 * Deposit
  Equity:Capital -> Asset:Loans:alice 5000 GBP

2026-01-01 * Daily interest for 2026-01-01
  Expense:Interest -> Asset:Loans:alice 0.94 GBP

  ... (daily interest of 0.94 GBP through 2026-01-27) ...

2026-01-28 * Daily interest for 2026-01-28
  Expense:Interest -> Asset:Loans:alice 0.95 GBP

  ... (daily interest of 0.95 GBP through 2026-01-30) ...
```

30 days at 6.9% on £5,000. Daily interest rises from £0.94 to £0.95 as accrued interest compounds.

---

## Mortgage

Residential mortgage. Same mechanics as Personal Loan with a different default rate.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `InterestAccrual` -> `RepaymentSchedule`

**Defaults**: `annual_rate: 0.045` (4.5% p.a.)

**Parameters**: Same as Personal Loan -- `monthly_repayment`, `repayment_source`.

### Golden goluca file: `mortgage_30d`

> Given a Mortgage account opened 2026-01-01
> And £100,000.00 disbursed
> When 30 days elapse
> Then daily interest accrues at 4.5% p.a.

```
2026-01-01 * Deposit
  Equity:Capital -> Asset:Loans:mortgage 100000 GBP

2026-01-01 * Daily interest for 2026-01-01
  Expense:Interest -> Asset:Loans:mortgage 12.32 GBP

2026-01-02 * Daily interest for 2026-01-02
  Expense:Interest -> Asset:Loans:mortgage 12.33 GBP

  ... (daily interest rises gradually: 12.33 -> 12.34 -> 12.35 -> 12.36 -> 12.37) ...

2026-01-30 * Daily interest for 2026-01-30
  Expense:Interest -> Asset:Loans:mortgage 12.37 GBP
```

30 days at 4.5% on £100,000. Daily interest starts at £12.32 and rises to £12.37 as accrued interest compounds.

---

## Overdraft

Arranged overdraft facility on a current account.

**Feature chain**: `StatusLifecycle` -> `DepositAcceptance` -> `OverdraftFacility` -> `InterestAccrual`

**Defaults**: `annual_rate: 0.159` (15.9% p.a.), `overdraft_limit: 100000` (£1,000)

OverdraftFacility replaces WithdrawalProcessing -- it permits the balance to go negative down to `-overdraft_limit` and records the movement itself.

### Golden goluca file: `overdraft_30d`

> Given an Overdraft facility account opened 2026-01-01
> And £500.00 drawn (within £1,000 limit)
> When 30 days elapse
> Then daily interest accrues at 15.9% p.a. on negative balance

```
2026-01-01 * Withdrawal
  Liability:Current:alice -> Equity:Capital 500 GBP

2026-01-01 * Daily interest for 2026-01-01
  Liability:Current:alice -> Income:Interest 0.21 GBP

  ... (daily interest of 0.21 GBP through 2026-01-24) ...

2026-01-25 * Daily interest for 2026-01-25
  Liability:Current:alice -> Income:Interest 0.22 GBP

  ... (daily interest of 0.22 GBP through 2026-01-30) ...
```

30 days at 15.9% on -£500. Interest direction is reversed (debit to liability, credit to income) because the balance is negative. Daily interest rises from £0.21 to £0.22.

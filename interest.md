# Interest Accrual

[Home](index.html) | [Features](features.html) | [Products](products.html) | [API](api.html)

Interest handling has two distinct phases: **accrual** (calculating how much interest has accumulated) and **application** (crediting or debiting that interest to the account). These can happen on different schedules and at different precisions.

## Ledger movements

### Deposits and withdrawals

A savings account is a bank liability -- the bank owes the customer. When a customer deposits cash:

```
Customer Liability -> Cash
```

The customer's liability account is debited (increasing the bank's obligation) and cash is credited. The golden files should reflect this direction.

### Interest

Interest on a savings account is a cost to the bank. When interest is applied:

```
-ve Expense:Interest -> Customer Liability
```

This is a negative movement from the expense account to the customer liability account -- the bank recognises the expense and the customer's balance increases.

For lending products the direction is reversed -- interest is income to the bank and increases the customer's obligation.

## Accrual precision

The ledger stores applied balances in **minor units** (pence for GBP, exponent 2). But during accrual -- while interest is being calculated daily but not yet applied -- the internal representation must use higher precision to avoid accumulated rounding errors.

| Context | Precision | Example (GBP) |
|---------|-----------|---------------|
| Applied balance (ledger) | Minor units, exponent 2 | 100000 (= GBP 1,000.00) |
| Accrual register (normal) | 5 decimal places | 95890 (= 0.95890 in minor units) |
| Accrual register (high) | Up to 10 decimal places | For products needing sub-basis-point accuracy |

When interest is applied at the end of a period, the accumulated accrual is rounded to minor units and posted as a ledger movement. Any fractional remainder carries forward into the next period.

## Exact arithmetic: the 10,000 x 365 denominator

For rates specified to 0.01% (1 basis point) precision, there is an exact representation that avoids all rounding during accrual.

Express the daily interest calculation as an integer fraction:

```
daily_interest = balance_minor_units * rate_bps / (10000 * 365)
```

Where `rate_bps` is the annual rate in basis points (e.g. 350 for 3.50%). The denominator `10000 * 365 = 3,650,000` is fixed.

Example: GBP 10,000.00 at 3.50%:

```
daily_interest = 1000000 * 350 / 3650000
               = 350000000 / 3650000
               = 95.8904109589... minor units
```

The numerator (350,000,000) and denominator (3,650,000) are both integers. During accrual, accumulate the **numerator** and defer the division. At application time, divide, post the integer part to the ledger, and carry the remainder numerator forward.

This is the **no-rounding option**: interest specified to 0.01% annual gross is exactly representable, and no precision is lost during accrual. The only rounding occurs at application, where the fractional minor unit is carried forward rather than lost.

For actual/actual day count (see below), the denominator becomes `10000 * days_in_year` (3,650,000 or 3,660,000).

## Day count conventions

The day count convention determines how the annual rate is converted to a daily rate.

| Convention | Daily divisor | When to use |
|-----------|--------------|-------------|
| **Actual/365** (preferred) | Always 365 | UK money markets, most UK savings and lending products |
| **Actual/Actual** | 365 or 366 depending on leap year | Some bond markets, EU regulatory contexts |

**Actual/365** is the recommended default. It is simpler (fixed denominator) and standard for UK retail banking. In a leap year, 366 days of interest are charged against a 365 divisor, so the effective annual rate is slightly higher than the nominal rate.

**Actual/Actual** uses the actual number of days in the year as the divisor. In a leap year the daily rate is lower (divided by 366), so 366 days at the lower rate gives exactly the nominal annual rate. This is more "correct" but less common in UK retail products.

The day count convention is configured per product alongside the application period.

## Accrual and application periods

Products choose independent schedules for accrual and application:

| Period | Accrual | Application |
|--------|---------|-------------|
| Daily | Calculate interest each day | Post to ledger each day |
| Monthly | Calculate interest each day, accumulate | Post to ledger on month-end |
| Quarterly | Calculate interest each day, accumulate | Post to ledger on quarter-end |
| Annually | Calculate interest each day, accumulate | Post to ledger on year-end |

**Accrual is always daily** -- the balance can change any day, so interest must be tracked daily regardless of when it is applied. The application period controls when accumulated interest is posted to the account as a ledger movement.

### Recommendation

**Daily accrual, monthly application** is the recommended default for most products. It matches UK high-street savings account conventions and provides a good balance between accuracy and ledger volume.

Daily application (the current default) is appropriate for products where daily compounding is an explicit feature, or for simplicity during early development.

| Application period | When to use |
|-------------------|-------------|
| Daily | Simple products, early development, products advertising daily compounding |
| Monthly | **Recommended.** Standard UK savings and lending convention |
| Quarterly | Some NS&I products, specific fixed-term bonds |
| Annually | Long-term fixed-rate bonds, some ISA products |

### How application period affects compounding

When interest is applied, it becomes part of the balance and itself earns interest. More frequent application means more compounding, which increases the effective annual return.

Example: 5.00% nominal rate on GBP 10,000 for one year (actual/365):

| Application period | Compounding events/year | Year-end balance | Effective annual rate |
|-------------------|------------------------|-----------------|----------------------|
| Daily | 365 | GBP 10,512.67 | 5.1267% |
| Monthly | 12 | GBP 10,511.62 | 5.1162% |
| Quarterly | 4 | GBP 10,509.45 | 5.0945% |
| Annually | 1 | GBP 10,500.00 | 5.0000% |

The difference is small but meaningful for regulatory disclosure and product comparison.

## AER (Annual Equivalent Rate)

AER is the annualised effective rate that accounts for compounding. It allows customers to compare products with different application periods on a like-for-like basis.

```
AER = (1 + r/n)^n - 1
```

Where `r` is the nominal annual rate and `n` is the number of application periods per year.

| Application period | n | Formula |
|-------------------|---|---------|
| Daily | 365 | `(1 + r/365)^365 - 1` |
| Monthly | 12 | `(1 + r/12)^12 - 1` |
| Quarterly | 4 | `(1 + r/4)^4 - 1` |
| Annually | 1 | `r` (no compounding effect) |

AER is a **savings product** concept. UK regulations (BCOBS) require savings products to display AER so customers can compare rates. The AER is derived from the nominal rate and the application period -- changing the application period changes the AER even if the nominal rate stays the same.

The product specification sets the nominal `annual_rate`. The AER is computed from this rate and the configured application period. This conversion will live in gobank-products so that product documentation and customer-facing rates are consistent.

## APR (Annual Percentage Rate)

APR is the lending-side equivalent of AER. It represents the total annual cost of borrowing, including compounding and (eventually) fees.

For a simple interest-only calculation:

```
APR = (1 + r/n)^n - 1
```

This is the same formula as AER when only interest is considered. The distinction matters when fees are included -- APR folds arrangement fees, annual fees, and other charges into the effective rate.

APR applies to: Personal Loan, Mortgage, Overdraft.

**Status:** future feature. The current implementation does not include fee modelling, so APR and AER are numerically identical. APR will diverge once fee features are added.

## Product configuration

Each product specifies its interest method via parameters:

| Parameter | Values | Default |
|-----------|--------|---------|
| `annual_rate` | Decimal (e.g. `0.035`) | Set per product |
| `interest_application` | `daily`, `monthly`, `quarterly`, `annually` | `daily` (current) |
| `day_count` | `actual_365`, `actual_actual` | `actual_365` |
| `accrual_precision` | `5`, `10`, or `exact` | `5` |

When `accrual_precision` is `exact`, the 10,000 x 365 integer fraction method is used with no rounding until application.

### Product recommendations

| Product | Application | Day count | Rationale |
|---------|------------|-----------|-----------|
| Easy Access | Monthly | Actual/365 | Standard UK savings convention |
| Fixed Term | Monthly or Quarterly | Actual/365 | Matches typical bond terms |
| ISA | Monthly | Actual/365 | Standard UK ISA convention |
| Personal Loan | Monthly | Actual/365 | Aligns with repayment schedule |
| Mortgage | Monthly | Actual/365 | Aligns with repayment schedule |
| Overdraft | Daily | Actual/365 | Interest charged daily on fluctuating balances |

## Implementation notes

The core interest calculation lives in go-luca. Changes needed in go-luca:

1. **Accrual register** -- per-account accumulator at configurable precision (5dp, 10dp, or exact integer fraction). Tracks accrued-but-unapplied interest separately from the applied ledger balance.
2. **Day count convention** -- parameterise the daily divisor (365 fixed, or actual days in year).
3. **Exact fraction mode** -- accumulate numerator with denominator `10000 * day_count_divisor`, divide only at application time, carry remainder.
4. **Correct ledger directions** -- interest movements use negative expense-to-liability for savings, positive liability-to-income for lending.

Changes needed in gobank-products:

1. **InterestAccrual feature** -- gain awareness of `interest_application`, `day_count`, and `accrual_precision` parameters. Branch between immediate application (daily) and deferred application (monthly/quarterly/annually).
2. **AER/APR computation** -- derive from `annual_rate`, `interest_application`, and `day_count` for product documentation and regulatory output.
3. **Golden files** -- regenerate with correct ledger directions and correct arithmetic.

# Products

[Home](index.html) | [Contract](api.html) | [Interest](interest.html)

Each version publishes the parameters its rules read. A value here is the
one at adoption; the bank may change it by a setting without a new
version. A rule here is code, and a change to it is a new version.

| Version | Family | Parameter | Scope | Kind | Published |
|---|---|---|---|---|---|
| `easyaccess/v1` | Savings | `rate_bps` | version | bps | 150 |
| | | `cycle` | version | cycle | monthly |
| `easyaccess/v2` | Savings | `savings_rate_bps` | bank | bps | read from the bank's policy |
| | | `rate_bps` | version | bps | derived: `savings_rate_bps` − 15 bps, floor 0 |
| | | `cycle` | version | cycle | monthly |
| `fixedterm/v1` | Savings | `rate_bps` | version | bps | 400 |
| | | `cycle` | version | cycle | monthly |
| | | `term_months` | version | int | 24 |
| | | `maturity_day` | account | day | set at open: the open day plus the term |
| `isa/v1` | Savings | `rate_bps` | version | bps | 350 |
| | | `cycle` | version | cycle | monthly |
| | | `allowance` | version | money | 2000000 |
| | | `deposited` | account | money | 0 at open; every deposit adds to it |
| `personalloan/v1` | Lending | `rate_bps` | version | bps | 690 |
| | | `cycle` | version | cycle | monthly |
| `mortgage/v1` | Lending | `rate_bps` | version | bps | 450 |
| | | `cycle` | version | cycle | monthly |
| `overdraft/v1` | Lending | `rate_bps` | version | bps | 1590 |
| | | `cycle` | version | cycle | monthly |
| | | `limit` | version | money | 100000 |

## Rules of their own

| Version | Event | Rule |
|---|---|---|
| `fixedterm/v1` | open | writes `maturity_day` |
| `fixedterm/v1` | pre-posting | refuses a withdrawal before `maturity_day` |
| `isa/v1` | open | writes `deposited` = 0 |
| `isa/v1` | pre-posting | refuses a deposit that would take `deposited` over `allowance`; a withdrawal does not restore it |
| `isa/v1` | post-posting | adds a deposit to `deposited` |
| `overdraft/v1` | pre-posting | refuses a movement that would take the balance above `limit`: overdrawn is a debit balance |

Every version otherwise runs `gbp.Interest`: interest accrues daily on
the closing balance at `rate_bps`, actual/365, exact; when the `cycle`
period ends the whole pence are applied by a posting value-dated the last
second of that day and the remainder carries forward; `apply-interest`,
close and change of version apply the accrual to date.

## Goldens

Each version's package holds its golden `.goluca` files under `testdata`,
posted in the bank's view (gobank ADR-0008): a movement is from the
account credited to the account debited. A deposit is from the customer's
account to `Asset:Cash` and a withdrawal or drawdown the other way about,
so a savings balance runs negative and a loan positive; a current account
drawn past zero runs positive, overdrawn. Interest applied is a posting
from `Expense:Interest` of a negative amount for savings and from
`Income:Interest` of a positive one for lending. The savings goldens are
byte-identical to the ones the retired feature framework produced; the
lending goldens were regenerated in the bank's view, the framework's
overdraft having run the customer's way.

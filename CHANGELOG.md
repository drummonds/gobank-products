# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

## [0.5.0] - 2026-10-10

## [0.4.0] - 2026-10-10

 - Products as versioned code (gobank ADR-0006, story 1.8.1): the version contract, six v1 packages, the feature framework retired

### Added
- The product contract: `Version` (identity, declared parameters, rules),
  nine events (`StartUp`, `Open`, `PrePosting`, `PostPosting`, `Day`,
  `ParameterChange`, `Command`, `ChangeOfVersion`, `Close`), `Facts` read
  through an interface, `Intents` (settings, postings, positions) returned,
  `Refusal` for a rule's no. Rules are pure; the bank's runner carries the
  intents out.
- Parameters as declarations with a scope (bank, version, account), a kind
  (bps, money, int, cycle, day, text), a published value, or a derivation
  from another parameter (spread, floor, cap). `CheckDeclarations` is the
  shared start-up check.
- `gbp.Interest`, the interest-bearing rules a version embeds: daily
  accrual on the closing balance, application when the cycle ends, the
  `apply-interest` command, close and change of version applying to date.
- Six version packages, each with its own golden: `easyaccess/v1`,
  `fixedterm/v1` (maturity fixed at open from `term_months`, withdrawals
  refused before it), `isa/v1` (deposits refused over the allowance),
  `personalloan/v1`, `mortgage/v1`, `overdraft/v1` (movements refused past
  the limit). The rates the bank held as `float64` are published basis
  points. The five 32-day goldens are byte-identical to the old framework's.
- `testkit.Runner`, the reference runner over an in-memory go-luca ledger:
  implements `Facts`, resolves parameters by scope with effective-dated
  settings and derivations, carries out intents, surfaces refusals, logs
  reads. `testkit.CheckVersion`, `testkit.RoundTrip`.

### Changed
- `Accrue` and `ApplyAccrued` are functions of the engine, not methods of a
  product; `ApplyAccrued` takes the family and the description and always
  applies, the version deciding when. `ApplicationCycle.Description` exported.
- go-luca pinned at v0.5.0.

### Removed
- `Product`, `Feature` and the handler interfaces, `SimContext`,
  `Simulation`, `ManagedAccount`, `ParameterStore`, `Clock`, the catalogue
  constructors (`EasyAccess()` and the rest), `RepaymentSchedule`
  (`monthly_repayment` was never set by the bank), `NextDay`,
  `ParamInterestApplication`; testkit's `ScenarioBuilder`, `GolucaScenario`
  and the FSMs. gobank stays on v0.3.0 until its story 1.8.2 adopts the
  contract.

## [0.3.0] - 2026-10-05

 - Product.Accrue and Product.Apply split the day rule for gobank stage 3 story (e); RateBps exported

### Added
- `Product.Accrue` and `Product.Apply`, the two halves of the day rule
  (gobank ADR-0002 stage 3, story e). `Accrue` is the position at the end
  of a day with the day's interest accrued on the closing balance and
  nothing applied: pure and the same for any balance, so the daily pass
  writes it as the day's provisional projection and every event that moves
  the balance writes it again. `Apply` books the cycle-end application from
  a position, and applying an applied position posts nothing, so the pass
  may revisit an account after a restart. `NextDay` is unchanged and is
  `Apply` of `Accrue`.
- `RateBps`, the conversion from a product's annual rate to the integer
  basis points the day rule accrues in, exported from the engine for
  callers that run the rules without it.

## [0.2.0] - 2026-10-05

 - Product.NextDay: the day rule as a pure function over go-luca positions; application cycle a product parameter; no separate month-end pass

 - `Product.NextDay` (gobank ADR-0002 stage 3, story c): the per-account day
   rule as a pure function — from the account's position at the end of the
   previous day (go-luca `Position`, balance and accrued interest as an exact
   fraction) and the day's closing balance to the position at the end of
   this day and the ledger `Posting`s it calls for. Interest accrues on the
   closing balance; when the product's cycle ends on the day the whole minor
   units are applied and the remainder carries forward.
 - The application cycle is a product parameter, `interest_application`:
   `daily`, `monthly` (the default, so the catalogue is unchanged) or
   `annual`. `InterestAccrual` runs `NextDay` at `EndOfDay` and no longer
   handles `EndOfMonth`; interest application is inside the daily pass.
   `EndOfMonth` remains for repayment schedules.
 - `AccountUpdate` on an application day now has `ClosingBalance` and
   `AccruedNumerator` after application, `InterestAmount` the interest
   applied, and `AccruedDelta` the day's accrual alone. Callers that derived
   the applied interest from the cached balance should use `InterestAmount`.
 - Requires go-luca v0.3.0 (`Position`, `Fraction`) and go-postgres v0.7.0.
 - `Simulation.AdoptAccount`: put an account that already exists in the
   ledger under the engine's management (status, opened date, cached balance
   and parameters as the caller read them) without creating it or raising an
   AccountOpened event. A bank restarting over a stored ledger brings its
   accounts back this way. Rate resolution is shared with `OpenAccount`.

## [0.1.10] - 2026-08-26

 - Exact daily interest accrual in memory with monthly application; cached balances; PaceHook

## [0.1.10] - 2026-08-26

 - Rework InterestAccrual: exact integer daily accrual in memory (numerator
   over 10,000 x 365, remainder carried — sub-penny interest is no longer
   lost), applied to the account monthly as a single ledger movement.
   Previously interest posted daily to :Accrued sub-accounts and was never
   applied to balances.
 - ManagedAccount gains CachedBalance (maintained by RecordMovement,
   removing per-day balance queries from end-of-day sweeps), RateBps,
   AccruedNumerator, and AccruedInterest().
 - Simulation gains RefreshBalances (post-import cache priming) and
   PaceHook (lets single-threaded hosts yield during account sweeps).
 - AccountUpdate gains AccruedDelta and AccruedNumerator.

## [0.1.9] - 2026-08-24

 - Bump go-luca to v0.2.31 (with go-postgres v0.5.5, gotreesitter v0.6.8);
   the previous go-luca v0.2.25 pin was unresolvable under the
   git.bytestone.uk module path

## [0.1.6] - 2026-03-25

 - Fix transaction directions and move interest logic to product layer

## [0.1.5] - 2026-03-24

 - Migrate dependencies from github.com/drummonds to codeberg.org/hum3

## [0.1.4] - 2026-03-19

 - working on interest feature

## [0.1.3] - 2026-03-19

 - updating checks after go-postgres update

## [0.1.2] - 2026-03-17

 - adding documentation

## [0.1.1] - 2026-03-17

 - Working on goluca integration

## [0.1.0] - 2026-03-16

 - fixing linter bug

### Added
- Core types: EventType, Feature interface, Product, ManagedAccount, Clock, ParameterStore
- Simulation engine with event dispatch and time advancement
- Features: StatusLifecycle, InterestAccrual, DepositAcceptance, WithdrawalProcessing, TermLock, ISAWrapper, OverdraftFacility, RepaymentSchedule
- Product catalog: EasyAccess, FixedTerm, ISA, PersonalLoan, Mortgage, Overdraft
- Test infrastructure: testkit package with ScenarioBuilder, goluca assertions, golden file helpers

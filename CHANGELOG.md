# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

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

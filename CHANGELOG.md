# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

## [0.1.0] - 2026-03-16

 - fixing linter bug

### Added
- Core types: EventType, Feature interface, Product, ManagedAccount, Clock, ParameterStore
- Simulation engine with event dispatch and time advancement
- Features: StatusLifecycle, InterestAccrual, DepositAcceptance, WithdrawalProcessing, TermLock, ISAWrapper, OverdraftFacility, RepaymentSchedule
- Product catalog: EasyAccess, FixedTerm, ISA, PersonalLoan, Mortgage, Overdraft
- Test infrastructure: testkit package with ScenarioBuilder, goluca assertions, golden file helpers

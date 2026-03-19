# gobank-products

Composable banking product library for Go. Products are built from independently testable **features** driven by a **simulation engine** with controllable time and typed events.

## Pages

- [Features](features.html) -- composable building blocks and their permitted events
- [Products](products.html) -- product catalog, event dispatch matrix, and golden goluca files
- [Interest Accrual](interest.html) -- accrual vs application periods, AER, APR, and product recommendations
- [API](api.html) -- simulation engine, parameter store, clock, and extension points
- [Changelog](CHANGELOG.html)
- [Roadmap](ROADMAP.html)

## Product Catalog

### Savings

| Product | Default Rate | Features |
|---------|-------------|----------|
| [Easy Access](products.html#easy-access) | 1.5% | Deposit, Withdrawal, Interest, Lifecycle |
| [Fixed Term](products.html#fixed-term) | 4.0% | Deposit, Term Lock, Withdrawal, Interest, Lifecycle |
| [ISA](products.html#isa) | 3.5% | ISA Allowance, Deposit, Withdrawal, Interest, Lifecycle |

### Lending

| Product | Default Rate | Features |
|---------|-------------|----------|
| [Personal Loan](products.html#personal-loan) | 6.9% | Deposit, Interest, Repayment, Lifecycle |
| [Mortgage](products.html#mortgage) | 4.5% | Deposit, Interest, Repayment, Lifecycle |
| [Overdraft](products.html#overdraft) | 15.9% | Deposit, Overdraft Facility, Interest, Lifecycle |

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

Products are pure data -- a name, family, list of features, and default parameters. Features are stateless handlers. All state lives in the ledger (balances, movements) and the parameter store (rates, dates, limits).

## Links

- **Documentation**: [h3-gobank-products.statichost.eu](https://h3-gobank-products.statichost.eu)
- **Source**: [codeberg.org/hum3/gobank-products](https://codeberg.org/hum3/gobank-products)
- **Mirror**: [github.com/drummonds/gobank-products](https://github.com/drummonds/gobank-products)

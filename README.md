# gobank-products

The product contract of gobank, and the products that implement it. A
product version is a Go package (`easyaccess/v1`) that declares the
parameters its rules read and answers a fixed set of events with intents;
the bank runs it. Rules are pure: they read facts through an interface and
return positions, postings and refusals, so every version tests with a
golden `.goluca` file and no database. See gobank ADR-0006.

## Links

- **Documentation**: https://gobank-products.docs.bytestone.uk/
- **Source**: https://git.bytestone.uk/hum3/gobank-products
- **Mirror**: https://github.com/drummonds/gobank-products

## Usage

```go
import (
    gbp "git.bytestone.uk/hum3/gobank-products"
    easyaccess "git.bytestone.uk/hum3/gobank-products/easyaccess/v1"
)

var v gbp.Version = easyaccess.Version

// The runner gathers facts, asks the rule, carries out the intents.
intents, err := v.Day(facts)
for _, p := range intents.Postings { /* post it */ }
for _, p := range intents.Positions { /* project it */ }
```

The reference runner is `testkit.Runner`: one account on one version over
an in-memory go-luca ledger, driven through the events as the bank drives
them.

## Products

| Version | Family | Rate | Cycle | Its own rules |
|---|---|---|---|---|
| `easyaccess/v1` | Savings | 1.50% | monthly | none |
| `fixedterm/v1` | Savings | 4.00% | monthly | maturity fixed at open (24 months); withdrawals refused before it |
| `isa/v1` | Savings | 3.50% | monthly | deposits refused over the £20,000 allowance |
| `personalloan/v1` | Lending | 6.90% | monthly | none |
| `mortgage/v1` | Lending | 4.50% | monthly | none |
| `overdraft/v1` | Lending | 15.90% | monthly | movements refused past the £1,000 limit |

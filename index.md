# gobank-products

The product contract of gobank, and the products that implement it
(gobank ADR-0006). A product version is a Go package that declares the
parameters its rules read and answers a fixed set of events with intents.
A version is immutable once the bank has adopted it: a change to the
rules is a new package; a change to a value is a setting.

## Pages

- [Contract](api.html) -- version, events, facts, intents, parameters
- [Products](products.html) -- the versions and their parameters
- [Interest](interest.html) -- the accrual arithmetic every version shares
- [Changelog](CHANGELOG.html)
- [Roadmap](ROADMAP.html)

## How a version runs

```
bank event (open, posting, day, command, close)
    |
    v
runner: resolves the account's version and its parameters,
        gathers facts (positions, balance, the movement)
    |
    v
version.Rule(facts) -> intents (settings, postings, positions) or a refusal
    |
    v
runner: carries the intents out under the account's lock,
        records the parameters the rule read against its postings
```

Rules are pure, so a version tests with a golden `.goluca` file and no
database, and the bank may run a rule again for the same facts (after a
restart, after an event that followed the pass) and get the same answer.

## Links

- **Documentation**: [gobank-products.docs.bytestone.uk](https://gobank-products.docs.bytestone.uk/)
- **Source**: [git.bytestone.uk/hum3/gobank-products](https://git.bytestone.uk/hum3/gobank-products)
- **Mirror**: [github.com/drummonds/gobank-products](https://github.com/drummonds/gobank-products)

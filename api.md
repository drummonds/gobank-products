# The contract

[Home](index.html) | [Products](products.html) | [Interest](interest.html)

## Version

```go
type Version interface {
    Product() string   // "easy-access"
    Version() int      // 1, 2, …
    Name() string
    Family() ProductFamily
    Parameters() []Declaration
    Rules
}
```

A version lives in a package of its own (`easyaccess/v1`) and exports one
value, `Version`. It is immutable once adopted; a change to the rules is
a new package. The build carries only the versions the bank has: those on
sale and those an account runs on. A retired version's postings stay in
the bank with a record of the version, the event, the parameters read and
the inputs, and are verified by the arithmetic in this package, never by
the version's rules.

## Events

| Rule | Fired by the bank when | Reads | Returns |
|---|---|---|---|
| `StartUp(p, day)` | the process starts, once per adopted version | the parameters as resolvable on day | an error stops the bank |
| `Open(f)` | an account opens on the version | the open day, the parameters | account settings, the first position |
| `PrePosting(f, m)` | before a movement posts | the movement, the position, the parameters | nil to allow; a `*Refusal` to refuse |
| `PostPosting(f, m)` | after the movement posted | the balance as it now stands | the day's position |
| `Day(f)` | the start-of-day pass visits the account | yesterday's position, the balance, parameters for both days | yesterday closed with its postings, today's position |
| `ParameterChange(f, c)` | a setting the version reads becomes effective | old and new value | nothing, or postings |
| `Command(f, name)` | the console runs one of `Commands()` | the position | postings, the position |
| `ChangeOfVersion(f, to)` | the account moves to another version | the position | the outgoing version's closing postings |
| `Close(f)` | the account closes | the position | the final postings |

Every rule is pure: the same facts give the same intents, so the runner
may run a rule again after a restart, or after an event that followed the
pass, and the last answer is the right one.

## Facts and intents

```go
type Facts interface {
    Parameter(key string, day time.Time) (Param, error)
    Account() string
    Day() time.Time
    Position(day time.Time) (luca.Position, bool)
    Balance() luca.Amount
}

type Movement struct {
    Delta       luca.Amount // the change to the balance, signed as the balance is
    Code, Description string
    ValueTime   time.Time
}

type Intents struct {
    Settings  []Setting       // account-scoped parameters to write
    Postings  []Posting       // counterparty -> account, signed as the balance is
    Positions []luca.Position // to project
}
```

A refusal is `gbp.Refuse("reason")`; the runner tells it from a fault with
`gbp.IsRefusal`.

## Parameters

```go
type Declaration struct {
    Key       string
    Scope     Scope       // bank | version | account
    Kind      Kind        // bps | money | int | cycle | day | text
    Published string      // the value at adoption (version scope)
    Derived   *Derivation // Source, SpreadBps, Floor, Cap
}
```

The runner resolves a parameter for a day (gobank ADR-0006, parameter
resolution): a derived parameter from its source and the formula; a
declared one from the latest setting effective on or before the day at its
scope, or, at version scope, from the published value. A setting effective
after the day is invisible to it, which is how a future rate change is
decided today and takes effect on its day with nothing to do.

`gbp.Interest` is the interest-bearing account's rules, which a version
embeds and overrides where its product differs. `gbp.CheckDeclarations`
is the start-up check every version shares.

## Testing a version

```go
r := testkit.Open(t, easyaccess.Version, "Liability:Savings:alice", day)
r.Deposit(100000)
r.Advance(32)
testkit.Golden(t, "easy_access_32d", r.Export())
```

`testkit.Runner` is the reference runner: it implements `Facts` over an
in-memory go-luca ledger and settings by scope, carries out every intent,
surfaces refusals from `TryDeposit` and `TryWithdraw`, and logs the
parameters each rule read in `Reads`. `testkit.CheckVersion` asserts a
version's identity and declarations. `GOLDEN_UPDATE=1` rewrites a golden.

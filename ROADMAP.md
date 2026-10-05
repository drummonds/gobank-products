# Roadmap

## Done
- Core types and simulation engine
- Feature-based product composition
- Product catalog (6 products)
- Test infrastructure (testkit)
- `Product.NextDay`: the day rule as a pure function over go-luca positions,
  with the application cycle a product parameter (gobank ADR-0002 stage 3)

## Next
- The engine works on the account in hand: `Simulation` stops holding every
  account in a map once gobank's daily pass loads each account's position
  from the ledger (ADR-0002 stage 3, story e)
- Postgres integration test harness (env-gated)
- Golden file .goluca scenario tests
- gobank integration: replace AccountBehavior with gbp.Product

## Future
- Rate change events and product repricing
- Multi-currency support
- Regulatory reporting features

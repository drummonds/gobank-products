# Roadmap

## Done
- The product contract and six v1 version packages (gobank ADR-0006,
  story 1.8.1): rules pure over facts, intents carried out by the runner,
  parameters declared with scope, kind and derivation; the feature
  framework and simulation engine retired
- `Accrue`/`ApplyAccrued`: the day rule over go-luca positions, with the
  application cycle a parameter (gobank ADR-0002 stage 3)
- Golden `.goluca` tests per version, byte-identical to the framework's
- `Policy` and `Declaration.Algorithm`, a parameter derived by code;
  `easyaccess/v2` tracking the bank's savings rate (gobank story 1.8.9)

## Next
- Adoption and withdrawal: what a version carries for the bank's up and
  down migration beyond its declarations, if anything (gobank story 1.8.2)
- A version with a rule for `ParameterChange`: applying the accrual to
  date at a rate change, as some products do

## Future
- Notice periods
- Corrections: a setting decided later with an earlier effective day
  producing an adjustment posting
- Multi-currency support

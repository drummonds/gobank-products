# gobank-products

Composable banking product library for Go. Products are built from independently testable features (interest accrual, deposits, withdrawals, term locks, etc.) driven by a simulation framework with controllable time and typed events.

## Links

- **Documentation**: https://h3-gobank-products.statichost.eu
- **Source**: https://codeberg.org/hum3/gobank-products
- **Mirror**: https://github.com/drummonds/gobank-products

## Usage

```go
import gbp "codeberg.org/hum3/gobank-products"

sim, _ := gbp.NewSimulation(ledger, clock)
sim.RegisterProduct(gbp.EasyAccess())

ma, _ := sim.OpenAccount("easy-access", "Liability:Savings:alice", "GBP", -2, nil)
sim.Deposit(ma.Account.ID, 100000, equityID, "")
sim.AdvanceToDate(targetDate)
```

## Product Catalog

| Product | Family | Default Rate | Features |
|---------|--------|-------------|----------|
| Easy Access | Savings | 1.5% | Deposit, Withdrawal, Interest, Lifecycle |
| Fixed Term | Savings | 4.0% | Deposit, Term Lock, Interest, Lifecycle |
| ISA | Savings | 3.5% | ISA Allowance, Deposit, Withdrawal, Interest, Lifecycle |
| Personal Loan | Lending | 6.9% | Interest, Repayment, Lifecycle |
| Mortgage | Lending | 4.5% | Interest, Repayment, Lifecycle |
| Overdraft | Lending | 15.9% | Deposit, Overdraft Facility, Interest, Lifecycle |

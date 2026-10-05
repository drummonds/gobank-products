package gbp_test

import (
	"testing"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
	gbp "git.bytestone.uk/hum3/gobank-products"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

// Product.NextDay is the per-account day rule (gobank ADR-0002 stage 3):
// from the account's position at the end of the previous day and the day's
// closing balance, the position at the end of this day and the ledger
// postings it calls for. Interest accrues on the closing balance; it is
// applied when the product's cycle ends on this day.

func utcDay(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestNextDayAccruesOnTheClosingBalance(t *testing.T) {
	day := utcDay(2026, 1, 15)
	prev := luca.Position{AccountID: "alice", Day: day.AddDate(0, 0, -1)}

	next, postings := gbp.EasyAccess().NextDay(day, prev, -100000, 150)

	if len(postings) != 0 {
		t.Fatalf("postings = %+v, want none", postings)
	}
	want := luca.Position{AccountID: "alice", Day: day, Balance: -100000,
		Accrued: luca.Fraction{Num: -100000 * 150, Den: gbp.AccrualDenominator}}
	if next != want {
		t.Errorf("next = %+v, want %+v", next, want)
	}
}

func TestNextDayAppliesInterestAtMonthEnd(t *testing.T) {
	day := utcDay(2026, 1, 31)
	prev := luca.Position{AccountID: "alice", Day: day.AddDate(0, 0, -1),
		Balance: -100000, Accrued: luca.Fraction{Num: -100000 * 150 * 30, Den: gbp.AccrualDenominator}}

	next, postings := gbp.EasyAccess().NextDay(day, prev, -100000, 150)

	if len(postings) != 1 {
		t.Fatalf("postings = %+v, want one", postings)
	}
	p := postings[0]
	wantPosting := gbp.Posting{
		Counterparty: "Expense:Interest",
		Amount:       -127, // 31 days at 150 bps on £1000 is 127.4p
		Code:         luca.CodeInterestAccrual,
		ValueTime:    time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		Description:  "Interest applied for month ending 2026-01-31",
	}
	if p != wantPosting {
		t.Errorf("posting = %+v, want %+v", p, wantPosting)
	}
	if next.Balance != -100127 {
		t.Errorf("balance = %d, want -100127", next.Balance)
	}
	if remainder := int64(-100000*150*31) + 127*gbp.AccrualDenominator; next.Accrued.Num != remainder || next.Accrued.Den != gbp.AccrualDenominator {
		t.Errorf("accrued = %+v, want %d/%d carried forward", next.Accrued, remainder, gbp.AccrualDenominator)
	}
}

func TestNextDayLendingAppliesFromIncome(t *testing.T) {
	day := utcDay(2026, 1, 31)
	prev := luca.Position{AccountID: "loan", Day: day.AddDate(0, 0, -1),
		Balance: 100000, Accrued: luca.Fraction{Num: 100000 * 690 * 30, Den: gbp.AccrualDenominator}}

	next, postings := gbp.PersonalLoan().NextDay(day, prev, 100000, 690)

	if len(postings) != 1 || postings[0].Counterparty != "Income:Interest" || postings[0].Amount != 586 {
		t.Fatalf("postings = %+v, want one from Income:Interest of 586", postings)
	}
	if next.Balance != 100586 {
		t.Errorf("balance = %d, want 100586", next.Balance)
	}
}

func TestNextDayApplicationCycleIsAProductParameter(t *testing.T) {
	accrued := luca.Fraction{Num: -100000 * 150 * 30, Den: gbp.AccrualDenominator}
	product := func(cycle string) *gbp.Product {
		p := &gbp.Product{ID: "p", Family: gbp.FamilySavings, Defaults: map[string]string{}}
		if cycle != "" {
			p.Defaults[gbp.ParamInterestApplication] = cycle
		}
		return p
	}
	cases := []struct {
		cycle string
		day   time.Time
		apply bool
		desc  string
	}{
		{"", utcDay(2026, 1, 15), false, ""},
		{"", utcDay(2026, 1, 31), true, "Interest applied for month ending 2026-01-31"},
		{"daily", utcDay(2026, 1, 15), true, "Interest applied for 2026-01-15"},
		{"monthly", utcDay(2026, 2, 28), true, "Interest applied for month ending 2026-02-28"},
		{"annual", utcDay(2026, 1, 31), false, ""},
		{"annual", utcDay(2026, 12, 31), true, "Interest applied for year ending 2026-12-31"},
	}
	for _, c := range cases {
		prev := luca.Position{AccountID: "a", Day: c.day.AddDate(0, 0, -1), Balance: -100000, Accrued: accrued}
		_, postings := product(c.cycle).NextDay(c.day, prev, -100000, 150)
		if (len(postings) == 1) != c.apply {
			t.Errorf("cycle %q on %s: postings = %+v, want applied=%v", c.cycle, c.day.Format("2006-01-02"), postings, c.apply)
			continue
		}
		if c.apply && postings[0].Description != c.desc {
			t.Errorf("cycle %q on %s: description %q, want %q", c.cycle, c.day.Format("2006-01-02"), postings[0].Description, c.desc)
		}
	}
	if got := gbp.EasyAccess().ApplicationCycle(); got != gbp.ApplyMonthly {
		t.Errorf("EasyAccess cycle = %q, want monthly", got)
	}
}

func TestNextDayKeepsSubPennyInterestUnapplied(t *testing.T) {
	day := utcDay(2026, 1, 31)
	// Half a penny accrued: nothing to post, the remainder carries forward.
	prev := luca.Position{AccountID: "a", Day: day.AddDate(0, 0, -1), Balance: -100,
		Accrued: luca.Fraction{Num: -gbp.AccrualDenominator / 2, Den: gbp.AccrualDenominator}}

	next, postings := gbp.EasyAccess().NextDay(day, prev, -100, 0)

	if len(postings) != 0 {
		t.Errorf("postings = %+v, want none", postings)
	}
	if next.Accrued.Num != -gbp.AccrualDenominator/2 || next.Balance != -100 {
		t.Errorf("next = %+v, want the half penny kept", next)
	}
}

// The end-of-day sweep is a loop over NextDay: application happens inside
// the day, so the daily update carries it, and nothing is left for a
// separate month-end pass.
func TestSweepReportsApplicationInTheDailyUpdate(t *testing.T) {
	ledger := testkit.NewTestLedger(t)
	clock := gbp.NewSimClock(utcDay(2026, 1, 1))
	sim, err := gbp.NewSimulation(ledger, clock)
	if err != nil {
		t.Fatal(err)
	}
	sim.RegisterProduct(gbp.EasyAccess())
	ma, err := sim.OpenAccount("easy-access", "Liability:Savings:alice", "GBP", -2, nil)
	if err != nil {
		t.Fatal(err)
	}
	cash, err := ledger.CreateAccount("Asset:Cash", "GBP", -2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := sim.Deposit(ma.Account.ID, 100000, cash.ID, luca.CodeBookTransfer); err != nil {
		t.Fatal(err)
	}

	updates, err := sim.AdvanceToDate(utcDay(2026, 1, 31))
	if err != nil {
		t.Fatal(err)
	}
	last := updates[len(updates)-1]
	if !last.Date.Equal(utcDay(2026, 1, 31)) || len(last.Accounts) != 1 {
		t.Fatalf("last update = %+v", last)
	}
	au := last.Accounts[0]
	if au.InterestAmount != -127 || au.ClosingBalance != -100127 {
		t.Errorf("31 Jan: interest %d closing %d, want -127 and -100127", au.InterestAmount, au.ClosingBalance)
	}
	if au.AccruedDelta != -100000*150 {
		t.Errorf("31 Jan: accrued delta %d, want the day's accrual %d, not net of application", au.AccruedDelta, -100000*150)
	}
	if want := int64(-100000*150*31) + 127*gbp.AccrualDenominator; au.AccruedNumerator != want {
		t.Errorf("31 Jan: numerator %d, want remainder %d", au.AccruedNumerator, want)
	}
	if ma.CachedBalance != -100127 {
		t.Errorf("cached balance %d, want -100127", ma.CachedBalance)
	}
	for _, e := range (gbp.InterestAccrual{}).Handles() {
		if e == gbp.EventEndOfMonth {
			t.Error("InterestAccrual still handles EndOfMonth")
		}
	}
}

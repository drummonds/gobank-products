package gbp_test

import (
	"testing"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
	gbp "git.bytestone.uk/hum3/gobank-products"
)

func utcDay(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestAccrueAddsTheDayOnTheClosingBalance(t *testing.T) {
	day := utcDay(2026, 1, 15)
	prev := luca.Position{AccountID: "alice", Day: day.AddDate(0, 0, -1),
		Accrued: luca.Fraction{Num: -100000 * 150 * 13, Den: gbp.AccrualDenominator}}

	got := gbp.Accrue(day, prev, -100000, 150)

	want := luca.Position{AccountID: "alice", Day: day, Balance: -100000,
		Accrued: luca.Fraction{Num: -100000 * 150 * 14, Den: gbp.AccrualDenominator}}
	if got != want {
		t.Errorf("Accrue = %+v, want %+v", got, want)
	}
}

func TestApplyAccruedBooksWholeUnitsAndCarriesTheRemainder(t *testing.T) {
	pos := luca.Position{AccountID: "alice", Day: utcDay(2026, 1, 31), Balance: -100000,
		Accrued: luca.Fraction{Num: -100000 * 150 * 31, Den: gbp.AccrualDenominator}}

	next, postings := gbp.ApplyAccrued(gbp.FamilySavings, pos, "Interest applied for month ending 2026-01-31")

	if len(postings) != 1 {
		t.Fatalf("postings = %+v, want one", postings)
	}
	want := gbp.Posting{
		Counterparty: "Expense:Interest",
		Amount:       -127, // 31 days at 150 bps on £1000 is 127.4p
		Code:         luca.CodeInterestAccrual,
		ValueTime:    time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		Description:  "Interest applied for month ending 2026-01-31",
	}
	if postings[0] != want {
		t.Errorf("posting = %+v, want %+v", postings[0], want)
	}
	if next.Balance != -100127 {
		t.Errorf("balance = %d, want -100127", next.Balance)
	}
	if remainder := int64(-100000*150*31) + 127*gbp.AccrualDenominator; next.Accrued.Num != remainder || next.Accrued.Den != gbp.AccrualDenominator {
		t.Errorf("accrued = %+v, want %d/%d carried forward", next.Accrued, remainder, gbp.AccrualDenominator)
	}
	// Applying an applied position finds nothing to post.
	if again, more := gbp.ApplyAccrued(gbp.FamilySavings, next, "x"); len(more) != 0 || again != next {
		t.Errorf("second ApplyAccrued = %+v with %+v, want unchanged and none", again, more)
	}
}

func TestApplyAccruedLendingIsIncome(t *testing.T) {
	pos := luca.Position{AccountID: "loan", Day: utcDay(2026, 1, 31), Balance: 100000,
		Accrued: luca.Fraction{Num: 100000 * 690 * 31, Den: gbp.AccrualDenominator}}
	next, postings := gbp.ApplyAccrued(gbp.FamilyLending, pos, "")
	if len(postings) != 1 || postings[0].Counterparty != "Income:Interest" || postings[0].Amount != 586 || next.Balance != 100586 {
		t.Errorf("lending application = %+v, %+v; want 586 from Income:Interest", next, postings)
	}
}

func TestApplyAccruedKeepsSubUnitInterest(t *testing.T) {
	pos := luca.Position{AccountID: "a", Day: utcDay(2026, 1, 31), Balance: -100,
		Accrued: luca.Fraction{Num: -gbp.AccrualDenominator / 2, Den: gbp.AccrualDenominator}}
	next, postings := gbp.ApplyAccrued(gbp.FamilySavings, pos, "")
	if len(postings) != 0 || next != pos {
		t.Errorf("half a penny: %+v, %+v; want nothing posted and the position kept", next, postings)
	}
}

func TestApplyAccruedNegativeRatePostsTheOtherWay(t *testing.T) {
	// A savings balance at a negative rate accrues a positive numerator:
	// the application reduces the (negative) balance, debiting the customer.
	pos := luca.Position{AccountID: "a", Day: utcDay(2026, 1, 31), Balance: -100000,
		Accrued: luca.Fraction{Num: 100000 * 50 * 31, Den: gbp.AccrualDenominator}}
	next, postings := gbp.ApplyAccrued(gbp.FamilySavings, pos, "")
	if len(postings) != 1 || postings[0].Amount != 42 || next.Balance != -99958 {
		t.Errorf("negative rate: %+v, %+v; want +42 applied", next, postings)
	}
}

func TestApplicationCycle(t *testing.T) {
	cases := []struct {
		cycle gbp.ApplicationCycle
		day   time.Time
		ends  bool
		desc  string
	}{
		{gbp.ApplyMonthly, utcDay(2026, 1, 15), false, ""},
		{gbp.ApplyMonthly, utcDay(2026, 1, 31), true, "Interest applied for month ending 2026-01-31"},
		{gbp.ApplyMonthly, utcDay(2026, 2, 28), true, "Interest applied for month ending 2026-02-28"},
		{gbp.ApplyDaily, utcDay(2026, 1, 15), true, "Interest applied for 2026-01-15"},
		{gbp.ApplyAnnual, utcDay(2026, 1, 31), false, ""},
		{gbp.ApplyAnnual, utcDay(2026, 12, 31), true, "Interest applied for year ending 2026-12-31"},
	}
	for _, c := range cases {
		if got := c.cycle.EndsOn(c.day); got != c.ends {
			t.Errorf("%s ends on %s = %v, want %v", c.cycle, c.day.Format("2006-01-02"), got, c.ends)
		}
		if c.ends && c.cycle.Description(c.day) != c.desc {
			t.Errorf("%s on %s: description %q, want %q", c.cycle, c.day.Format("2006-01-02"), c.cycle.Description(c.day), c.desc)
		}
	}
}

func TestRateBps(t *testing.T) {
	for _, c := range []struct {
		rate float64
		want int64
	}{{0.035, 350}, {0.0125, 125}, {0.069, 690}, {0, 0}, {-0.001, -10}} {
		if got := gbp.RateBps(c.rate); got != c.want {
			t.Errorf("RateBps(%v) = %d, want %d", c.rate, got, c.want)
		}
	}
}

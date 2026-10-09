package gbp_test

import (
	"errors"
	"testing"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
	gbp "git.bytestone.uk/hum3/gobank-products"
)

// memFacts is Facts in memory: positions by day, one balance, and
// parameters as a flat map that ignores the day unless an entry is dated.
type memFacts struct {
	account   string
	day       time.Time
	balance   luca.Amount
	positions map[time.Time]luca.Position
	params    map[string]gbp.Param
}

func newFacts(day time.Time) *memFacts {
	return &memFacts{account: "acct", day: day, positions: map[time.Time]luca.Position{},
		params: map[string]gbp.Param{
			"rate_bps": {Key: "rate_bps", Kind: gbp.KindBps, Text: "150"},
			"cycle":    {Key: "cycle", Kind: gbp.KindCycle, Text: "monthly"},
		}}
}

func (m *memFacts) Account() string { return m.account }
func (m *memFacts) Day() time.Time  { return m.day }
func (m *memFacts) Position(day time.Time) (luca.Position, bool) {
	p, ok := m.positions[day]
	return p, ok
}
func (m *memFacts) Balance() luca.Amount { return m.balance }
func (m *memFacts) Parameter(key string, _ time.Time) (gbp.Param, error) {
	p, ok := m.params[key]
	if !ok {
		return gbp.Param{}, errors.New("parameter " + key + ": not set")
	}
	return p, nil
}

var savings = gbp.Interest{Family: gbp.FamilySavings, RateKey: "rate_bps", CycleKey: "cycle"}

func TestInterestDayMidMonthAccruesOnly(t *testing.T) {
	f := newFacts(utcDay(2026, 1, 15))
	f.balance = -100000
	f.positions[utcDay(2026, 1, 14)] = luca.Position{AccountID: "acct", Day: utcDay(2026, 1, 14), Balance: -50000,
		Accrued: luca.Fraction{Num: -50000 * 150 * 13, Den: gbp.AccrualDenominator}}

	in, err := savings.Day(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Postings) != 0 || len(in.Settings) != 0 || len(in.Positions) != 1 {
		t.Fatalf("intents = %+v, want one position only", in)
	}
	want := luca.Position{AccountID: "acct", Day: utcDay(2026, 1, 15), Balance: -100000,
		Accrued: luca.Fraction{Num: -50000*150*13 - 100000*150, Den: gbp.AccrualDenominator}}
	if in.Positions[0] != want {
		t.Errorf("today = %+v, want %+v", in.Positions[0], want)
	}
}

func TestInterestDayAfterCycleEndAppliesYesterdayThenAccruesToday(t *testing.T) {
	f := newFacts(utcDay(2026, 2, 1))
	f.balance = -100000 // the application has not posted yet: the rule allows for it
	f.positions[utcDay(2026, 1, 31)] = luca.Position{AccountID: "acct", Day: utcDay(2026, 1, 31), Balance: -100000,
		Accrued: luca.Fraction{Num: -100000 * 150 * 31, Den: gbp.AccrualDenominator}}

	in, err := savings.Day(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Postings) != 1 || in.Postings[0].Amount != -127 || in.Postings[0].Description != "Interest applied for month ending 2026-01-31" {
		t.Fatalf("postings = %+v, want the January application of -127", in.Postings)
	}
	if len(in.Positions) != 2 {
		t.Fatalf("positions = %+v, want yesterday closed and today", in.Positions)
	}
	closed, today := in.Positions[0], in.Positions[1]
	if closed.Day != utcDay(2026, 1, 31) || closed.Balance != -100127 || closed.Accrued.Num != int64(-100000*150*31)+127*gbp.AccrualDenominator {
		t.Errorf("yesterday closed = %+v", closed)
	}
	if today.Day != utcDay(2026, 2, 1) || today.Balance != -100127 || today.Accrued.Num != closed.Accrued.Num-100127*150 {
		t.Errorf("today = %+v; want accrual on the balance after application", today)
	}
}

func TestInterestDayWithNoYesterdayStartsFromNothing(t *testing.T) {
	f := newFacts(utcDay(2026, 1, 1))
	f.balance = -100000
	in, err := savings.Day(f)
	if err != nil || len(in.Postings) != 0 || len(in.Positions) != 1 {
		t.Fatalf("intents = %+v, %v", in, err)
	}
	if p := in.Positions[0]; p.AccountID != "acct" || p.Balance != -100000 || p.Accrued.Num != -100000*150 {
		t.Errorf("first day = %+v", p)
	}
}

func TestInterestDayIsTheSameTwice(t *testing.T) {
	f := newFacts(utcDay(2026, 2, 1))
	f.balance = -100000
	f.positions[utcDay(2026, 1, 31)] = luca.Position{AccountID: "acct", Day: utcDay(2026, 1, 31), Balance: -100000,
		Accrued: luca.Fraction{Num: -100000 * 150 * 31, Den: gbp.AccrualDenominator}}
	first, _ := savings.Day(f)
	// The runner carried the intents out: the posting moved the balance and
	// yesterday's position was rewritten.
	f.balance += first.Postings[0].Amount
	f.positions[utcDay(2026, 1, 31)] = first.Positions[0]
	f.positions[utcDay(2026, 2, 1)] = first.Positions[1]

	second, _ := savings.Day(f)

	if len(second.Postings) != 0 || len(second.Positions) != 1 || second.Positions[0] != first.Positions[1] {
		t.Errorf("second run = %+v, want nothing to post and today's position unchanged", second)
	}
}

func TestInterestPostPostingIsTheDayRule(t *testing.T) {
	f := newFacts(utcDay(2026, 1, 15))
	f.balance = -150000
	day, _ := savings.Day(f)
	post, err := savings.PostPosting(f, gbp.Movement{Delta: -50000})
	if err != nil || len(post.Positions) != 1 || post.Positions[0] != day.Positions[0] {
		t.Errorf("PostPosting = %+v, %v; Day = %+v", post, err, day)
	}
}

func TestInterestApplyInterestCommandAppliesToday(t *testing.T) {
	f := newFacts(utcDay(2026, 1, 15))
	f.balance = -100000
	f.positions[utcDay(2026, 1, 14)] = luca.Position{AccountID: "acct", Day: utcDay(2026, 1, 14), Balance: -100000,
		Accrued: luca.Fraction{Num: -100000 * 150 * 14, Den: gbp.AccrualDenominator}}

	in, err := savings.Command(f, gbp.CommandApplyInterest)
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Postings) != 1 || in.Postings[0].Amount != -61 || in.Postings[0].Description != "Interest applied to 2026-01-15" ||
		in.Postings[0].ValueTime != time.Date(2026, 1, 15, 23, 59, 59, 0, time.UTC) {
		t.Errorf("postings = %+v, want 15 days (61p) applied today", in.Postings)
	}
	if len(in.Positions) != 1 || in.Positions[0].Balance != -100061 {
		t.Errorf("positions = %+v, want today's with the application in", in.Positions)
	}
	if _, err := savings.Command(f, "no-such"); err == nil {
		t.Error("unknown command accepted")
	}
	if cmds := savings.Commands(); len(cmds) != 1 || cmds[0] != gbp.CommandApplyInterest {
		t.Errorf("Commands = %v", cmds)
	}
}

func TestInterestCloseAndChangeOfVersionApplyToDate(t *testing.T) {
	f := newFacts(utcDay(2026, 1, 15))
	f.balance = -100000
	f.positions[utcDay(2026, 1, 14)] = luca.Position{AccountID: "acct", Day: utcDay(2026, 1, 14), Balance: -100000,
		Accrued: luca.Fraction{Num: -100000 * 150 * 14, Den: gbp.AccrualDenominator}}
	closed, _ := savings.Close(f)
	moved, _ := savings.ChangeOfVersion(f, nil)
	cmd, _ := savings.Command(f, gbp.CommandApplyInterest)
	if len(closed.Postings) != 1 || closed.Postings[0] != cmd.Postings[0] || moved.Postings[0] != cmd.Postings[0] {
		t.Errorf("close %+v, change %+v, command %+v should agree", closed.Postings, moved.Postings, cmd.Postings)
	}
}

func TestInterestStartUpNeedsRateAndCycle(t *testing.T) {
	f := newFacts(utcDay(2026, 1, 1))
	if err := savings.StartUp(f, f.day); err != nil {
		t.Errorf("start-up with both set: %v", err)
	}
	delete(f.params, "cycle")
	if err := savings.StartUp(f, f.day); err == nil {
		t.Error("start-up without the cycle passed")
	}
}

func TestRefusalIsTold(t *testing.T) {
	err := gbp.Refuse("account matures on %s", "2026-04-01")
	if !gbp.IsRefusal(err) || err.Error() != "account matures on 2026-04-01" {
		t.Errorf("refusal = %v", err)
	}
	if gbp.IsRefusal(errors.New("db down")) {
		t.Error("a fault is not a refusal")
	}
}

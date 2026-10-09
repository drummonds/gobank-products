package gbp

import (
	"fmt"
	"time"

	luca "git.bytestone.uk/hum3/go-luca"
)

// CommandApplyInterest applies the accrual now, off cycle.
const CommandApplyInterest = "apply-interest"

// Interest is the interest-bearing account's rules, which a version embeds
// and overrides where its product differs: interest accrues daily on the
// closing balance at the rate parameter and is applied when the cycle
// parameter's period ends. It answers every event, so a version that
// embeds it need add only what is its own.
type Interest struct {
	Family   ProductFamily
	RateKey  string // the bps parameter the accrual reads
	CycleKey string // the cycle parameter the application reads
}

// StartUp resolves the rate and the cycle.
func (i Interest) StartUp(p ParameterReader, day time.Time) error {
	for _, key := range []string{i.RateKey, i.CycleKey} {
		if _, err := p.Parameter(key, day); err != nil {
			return err
		}
	}
	return nil
}

// Open asks for nothing: the first day's position is the day rule's.
func (Interest) Open(Facts) (Intents, error) { return Intents{}, nil }

// PrePosting allows every movement.
func (Interest) PrePosting(Facts, Movement) error { return nil }

// PostPosting is the day rule again: the day's position on the balance as
// it now stands.
func (i Interest) PostPosting(f Facts, _ Movement) (Intents, error) { return i.Day(f) }

// Day closes yesterday and projects today. If the cycle ended yesterday
// and yesterday's position holds whole units of accrual, they are applied
// to the balance by a posting value-dated yesterday's last second and
// yesterday's position is written again with the remainder. Today's
// position then accrues the day's interest on the balance after any
// application. From the same facts it asks the same, so the pass may
// revisit an account and an event may follow the pass.
func (i Interest) Day(f Facts) (Intents, error) {
	day := startOfDay(f.Day())
	yesterday := day.AddDate(0, 0, -1)
	prev, ok := f.Position(yesterday)
	if !ok {
		prev = luca.Position{AccountID: f.Account(), Day: yesterday} // opened today: nothing carried in
	}
	cycle, err := f.Parameter(i.CycleKey, yesterday)
	if err != nil {
		return Intents{}, err
	}
	var in Intents
	closed, balance := prev, f.Balance()
	if cycle.Cycle().EndsOn(yesterday) {
		var postings []Posting
		closed, postings = ApplyAccrued(i.Family, prev, cycle.Cycle().Description(yesterday))
		for _, p := range postings {
			balance += p.Amount
		}
		if len(postings) > 0 {
			in.Postings = postings
			in.Positions = append(in.Positions, closed)
		}
	}
	rate, err := f.Parameter(i.RateKey, day)
	if err != nil {
		return Intents{}, err
	}
	in.Positions = append(in.Positions, Accrue(day, closed, balance, rate.Bps()))
	return in, nil
}

// ParameterChange asks for nothing: the day rule reads a changed rate or
// cycle from its effective day.
func (Interest) ParameterChange(Facts, Change) (Intents, error) { return Intents{}, nil }

// Commands is the one manual event: apply the accrual now.
func (Interest) Commands() []string { return []string{CommandApplyInterest} }

// Command runs a manual event by name.
func (i Interest) Command(f Facts, name string) (Intents, error) {
	if name != CommandApplyInterest {
		return Intents{}, fmt.Errorf("unknown command %q", name)
	}
	return i.applyNow(f)
}

// ChangeOfVersion closes the outgoing version's cycle: the accrual to
// date is applied, and the remainder carries into the incoming version.
func (i Interest) ChangeOfVersion(f Facts, _ Version) (Intents, error) { return i.applyNow(f) }

// Close applies the accrual to date.
func (i Interest) Close(f Facts) (Intents, error) { return i.applyNow(f) }

// applyNow is the day rule and then the application of today's position,
// off cycle, value-dated today's last second.
func (i Interest) applyNow(f Facts) (Intents, error) {
	in, err := i.Day(f)
	if err != nil {
		return in, err
	}
	today := in.Positions[len(in.Positions)-1]
	closed, postings := ApplyAccrued(i.Family, today, "Interest applied to "+today.Day.Format("2006-01-02"))
	in.Positions[len(in.Positions)-1] = closed
	in.Postings = append(in.Postings, postings...)
	return in, nil
}

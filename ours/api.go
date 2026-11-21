package ours

import (
	"fmt"
	"os"
	"runtime/debug"
)

func logf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// ForceFFA switches this brain to its free-for-all profile now.
func (b *Brain) ForceFFA() {
	if b.ffa {
		return
	}
	b.ffa = true
	if optPrac {
		setPracProfile(false)
	}
}

// SafeAct runs one decision, falling back to a held, non-firing copy of the last action on a panic.
func SafeAct(b *Brain, o *Obs, last *Action) (a *Action) {
	defer func() {
		if r := recover(); r != nil {
			logf("panic: %v\n%s", r, debug.Stack())
			if last != nil {
				c := *last
				c.Tick = o.Tick
				c.Fire = [2]bool{}
				c.Pick, c.Take, c.Drop, c.Debug = nil, nil, nil, nil
				c.Jump = false
				a = &c
			} else {
				a = &Action{Tick: o.Tick}
			}
		}
	}()
	if b == nil {
		return &Action{Tick: o.Tick}
	}
	a = b.Act(o)
	a.Tick = o.Tick
	return a
}

// IsFFA reports whether this brain has decided the match is a free-for-all
// (3+ mechs died that we didn't kill, held for 5 ticks).
func (b *Brain) IsFFA() bool { return b.ffa }

package auto

// Ported from src/brain/tele.rs.

type Tele struct {
	kills  uint32
	nlDone bool
	wlDone bool
}

func newTele() Tele {
	return Tele{}
}

func (b *Brain) teleUpdate(obs *Obs) {
	for ei := range obs.Events {
		if obs.Events[ei].Kill != nil {
			b.tele.kills++
		}
	}
	has := false
	for _, w := range obs.Me.Weapons {
		if w != nil && w.Kind == WeaponKindLaser {
			has = true
			break
		}
	}
	if b.p.HaltNl >= 0.0 && !b.tele.nlDone && obs.T >= b.p.HaltNl {
		b.tele.nlDone = true
		if !has {
			b.order.halted = true
		}
	}
	if b.p.HaltWl >= 0.0 && !b.tele.wlDone && obs.T >= b.p.HaltWl {
		b.tele.wlDone = true
		if has {
			b.order.halted = true
		}
	}
	if b.p.HaltK >= 1.0 && float64(b.tele.kills) >= b.p.HaltK {
		b.order.halted = true
	}
}

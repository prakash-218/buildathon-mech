package auto

// Ported from src/brain/kill.rs.

func (b *Brain) finishBonus(tr *Track) float64 {
	f := frac(tr.hp, tr.maxHp)
	if f < b.p.FinishHp {
		return b.p.FinishBonus * clampF(1.0-f/fmax(b.p.FinishHp, 0.01), 0.3, 1.0)
	}
	return 0.0
}

func (b *Brain) rushOk(obs *Obs, tr *Track) bool {
	t := obs.T
	tp := tr.posAt(t)
	others := 0
	for i := range b.mem.tracks {
		o := &b.mem.tracks[i]
		if o.kind != ActorKindMech || o.id == tr.id || t-o.lastSeen > 3.0 {
			continue
		}
		if dist(o.posAt(t), tp) < b.p.RushAloneR || t-o.lastDmgT < 2.5 {
			others++
		}
	}
	if others > 0 {
		return false
	}
	need := fmin(fmax(tr.hp, 0.0)/20.0, b.p.RushEnergyCap) * b.p.RushEnergyFrac
	return obs.Me.Energy >= need
}

func (b *Brain) precharge(obs *Obs, out *FireOut) {
	if b.p.Precharge <= 0.0 || b.mode == ModeRetreat || b.mode == ModeZone {
		return
	}
	me := &obs.Me
	t := obs.T
	i := -1
	var w *WeaponObs
	for j, wj := range me.Weapons {
		if wj != nil && wj.Kind == WeaponKindLaser {
			i = j
			w = wj
			break
		}
	}
	if w == nil {
		return
	}
	full := fmax(w.Stats.Charge, 0.05)
	tid := int64(-1)
	if b.target.kind == TgtMech {
		tid = b.target.id
	}
	hasBest := false
	var best tupFV
	for ti := range b.mem.tracks {
		tr := &b.mem.tracks[ti]
		if tr.kind != ActorKindMech || t-tr.lastSeen > b.p.PrechargeMemoryS {
			continue
		}
		tp := tr.posAt(fmin(t, tr.lastSeen+1.0))
		d := dist(tp, me.Pos)
		if d > w.Stats.Range*0.9 {
			continue
		}
		bonus := 0.0
		if tr.id == tid {
			bonus = 100.0
		}
		s := d - bonus
		if !hasBest || s < best.x0 {
			best = tupFV{s, tp}
			hasBest = true
		}
	}
	if w.Charge > 0.0 {
		if w.Charge < full+fmax(w.Stats.HoldS-0.3, 0.0) {
			out.fire[i] = true
			out.charging = true
		}
	} else if w.Cooldown <= 0.1 && hasBest {
		out.fire[i] = true
		out.charging = true
	}
	if hasBest {
		tp := best.x1
		c := up(tp, b.cfg.Mech.Height*0.5)
		muzzle := up(me.Pos, me.Stats.MuzzleHeight)
		out.aim = dirToAim(sub(c, muzzle))
		out.armed = true
	}
}

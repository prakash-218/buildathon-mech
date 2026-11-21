package auto

// Ported from src/brain/keep.rs.

func (b *Brain) holdLostCharge(obs *Obs, out *FireOut) {
	memS := b.p.LzHoldLost
	if memS <= 0.0 || b.mode == ModeRetreat || b.mode == ModeZone {
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
	if !(w.Charge > 0.0) || i > 1 {
		return
	}
	full := fmax(w.Stats.Charge, 0.05)
	if w.Charge >= full+fmax(w.Stats.HoldS-0.3, 0.0) {
		return
	}
	tid := int64(-1)
	if b.target.kind == TgtMech {
		tid = b.target.id
	}
	hasBest := false
	var best tupFV
	for ti := range b.mem.tracks {
		tr := &b.mem.tracks[ti]
		if tr.kind != ActorKindMech || t-tr.lastSeen > memS {
			continue
		}
		tp := tr.posAt(fmin(t, tr.lastSeen+1.0))
		d := dist(tp, me.Pos)
		if !(d <= w.Stats.Range) {
			continue
		}
		bonus := 0.0
		if tr.id == tid {
			bonus = 50.0
		}
		s := (t-tr.lastSeen)*100.0 + d*0.1 - bonus
		if !hasBest || s < best.x0 {
			best = tupFV{s, tp}
			hasBest = true
		}
	}
	if !hasBest {
		return
	}
	tp := best.x1
	out.fire[i] = true
	out.charging = true
	out.armed = true
	muzzle := up(me.Pos, me.Stats.MuzzleHeight)
	a := dirToAim(sub(up(tp, b.cfg.Mech.Height*0.5), muzzle))
	if isFinite(a[0]) && isFinite(a[1]) {
		out.aim = a
	}
}

package auto

import "math"

// Ported from src/brain/early.rs.

// Rust associated fn Brain::es_laser.
func brainEsLaser(obs *Obs) bool {
	for _, w := range obs.Me.Weapons {
		if w != nil && w.Kind == WeaponKindLaser {
			return true
		}
	}
	return false
}

func (b *Brain) esHold(obs *Obs, tr *Track) bool {
	p := b.p
	if p.EsOn < 0.5 || tr.kind != ActorKindMech {
		return false
	}
	t := obs.T
	if t-tr.lastDmgT < p.EsAttS {
		return false
	}
	me := &obs.Me
	hpf := frac(me.Hp, me.MaxHp)
	early := t < p.EsUntil && !brainEsLaser(obs)
	weak := hpf < p.EsHp
	if !(early || weak) {
		return false
	}
	d := dist(tr.posAt(t), me.Pos)
	if d < p.EsFreeD && !tr.has(WeaponKindLaser) && tr.cubes <= p.EsFreeCubes && !weak {
		return false
	}
	return true
}

func (b *Brain) esFilter(obs *Obs, fc *FireOut) {
	if b.p.EsOn < 0.5 || b.p.EsCone <= 0.0 {
		return
	}
	me := &obs.Me
	yaw := fc.aim[0]
	if !isFinite(yaw) {
		return
	}
	hasBest := false
	var best tupFI
	for ai := range obs.Mechs {
		a := &obs.Mechs[ai]
		if a.Kind != ActorKindMech || a.Id == me.Id || !(isFinite(a.Pos[0]) && isFinite(a.Pos[1]) && isFinite(a.Pos[2])) {
			continue
		}
		dx := a.Pos[0] - me.Pos[0]
		dy := a.Pos[1] - me.Pos[1]
		d := math.Hypot(dx, dy)
		if d < 1.0 || d > 400.0 {
			continue
		}
		e := math.Abs(adiff(yaw, math.Atan2(dy, dx)))
		cone := fmax(b.p.EsCone, math.Atan(6.0/d))
		if e < cone && (!hasBest || d < best.x0) {
			best = tupFI{d, a.Id}
			hasBest = true
		}
	}
	if !hasBest {
		return
	}
	id := best.x1
	var tr *Track
	for i := range b.mem.tracks {
		if b.mem.tracks[i].id == id {
			tr = &b.mem.tracks[i]
			break
		}
	}
	if tr == nil {
		return
	}
	if !b.esHold(obs, tr) {
		return
	}
	for i, w := range me.Weapons {
		if i >= 2 {
			break
		}
		if w != nil {
			if w.Kind == WeaponKindLaser && w.Charge > 0.0 {
				continue
			}
			fc.fire[i] = false
		}
	}
}

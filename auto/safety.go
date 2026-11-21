package auto

// Ported from src/brain/safety.rs.

type Safety struct {
	dealt   []tupIF
	exposed bool
	threat  bool
}

func newSafety() Safety {
	return Safety{}
}

func (b *Brain) sfUpdate(obs *Obs, world *World) {
	p := b.p
	t := obs.T
	for ei := range obs.Events {
		ev := &obs.Events[ei]
		if ev.Damaged != nil {
			d := ev.Damaged
			if d.By != nil {
				by := *d.By
				found := false
				for i := range b.sf.dealt {
					if b.sf.dealt[i].x0 == by {
						b.sf.dealt[i].x1 += fmax(d.Amount, 0.0)
						found = true
						break
					}
				}
				if !found {
					b.sf.dealt = append(b.sf.dealt, tupIF{by, fmax(d.Amount, 0.0)})
				}
			}
		}
	}
	b.sf.exposed = false
	b.sf.threat = false
	if p.SfOpenR <= 0.0 && p.SfSweep < 0.5 {
		return
	}
	me := obs.Me.Pos
	eye := up(me, obs.Me.Stats.Height*0.6)
	exposed := false
	threat := false
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech || t-tr.lastSeen > p.SfMemS {
			continue
		}
		q := tr.posAt(t)
		d := dist(q, me)
		if d > fmax(p.SfLosR, p.SfSweepR) || b.gearHarmless(tr, t, d) {
			continue
		}
		if !world.sight(up(q, b.cfg.Mech.EyeHeight), eye) {
			continue
		}
		if d <= p.SfLosR {
			exposed = true
		}
		if d <= p.SfSweepR {
			threat = true
		}
	}
	b.sf.exposed = exposed
	b.sf.threat = threat
}

func (b *Brain) sfDealt(id int64) float64 {
	for _, x := range b.sf.dealt {
		if x.x0 == id {
			return x.x1
		}
	}
	return 0.0
}

func (b *Brain) sfNotHarmless(tr *Track) bool {
	p := b.p
	if p.SfHarm < 0.5 {
		return false
	}
	return tr.has(WeaponKindLaser) || tr.cubes > p.SfHarmCubes || b.sfDealt(tr.id) >= p.SfHarmDmg
}

// Rust associated fn Brain::sf_has_laser.
func brainSfHasLaser(obs *Obs) bool {
	for _, w := range obs.Me.Weapons {
		if w != nil && w.Kind == WeaponKindLaser {
			return true
		}
	}
	return false
}

func (b *Brain) sfHunter(tr *Track, t float64) bool {
	return t-tr.lastDmgT < b.p.SfHuntS || b.order.role(tr.id, t, 0.0, b.p.SfHuntS) != RolePassive
}

func (b *Brain) sfHold(obs *Obs, tr *Track) bool {
	if b.esHold(obs, tr) {
		return true
	}
	p := b.p
	if p.SfNofirst < 0.5 || tr.kind != ActorKindMech || brainSfHasLaser(obs) {
		return false
	}
	t := obs.T
	if b.sfHunter(tr, t) {
		return false
	}
	d := dist(tr.posAt(t), obs.Me.Pos)
	return tr.has(WeaponKindLaser) || d > p.SfNfClose
}

func (b *Brain) sfNoMortar() bool {
	return b.p.SfMortar >= 0.5
}

func (b *Brain) sfDrop(obs *Obs, act *Action) {
	if b.p.SfMortar < 1.5 || act.Take != nil || act.Drop != nil {
		return
	}
	for i, w := range obs.Me.Weapons {
		if w != nil && w.Kind == WeaponKindMortar {
			v := int64(i)
			act.Drop = &v
			return
		}
	}
}

func (b *Brain) sfQuietBlock() bool {
	return b.p.SfSweep >= 0.5 && b.sf.threat
}

func (b *Brain) sfFarBlock(d float64) bool {
	return b.p.SfOpenR > 0.0 && d > b.p.SfOpenR && b.sf.exposed
}

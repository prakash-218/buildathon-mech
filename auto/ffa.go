package auto

import (
	"fmt"
	"math"
	"os"
)

// Ported from src/brain/ffa.rs.

type Ffa struct {
	on       bool
	onT      float64
	why      uint8
	alive0   int64
	kills    int64
	sig      uint32
	seenHp   []tupIFF
	ourHits  []tupIF
	foreign  []tupIF
	beamLast []tupIF
	seed     int64
}

func newFfa(seed int64) Ffa {
	return Ffa{seed: seed}
}

func (f *Ffa) foreignT(id int64) float64 {
	for _, x := range f.foreign {
		if x.x0 == id {
			return x.x1
		}
	}
	return -1e9
}

func (f *Ffa) mark(id int64, t float64) {
	for i := range f.foreign {
		if f.foreign[i].x0 == id {
			f.foreign[i].x1 = t
			return
		}
	}
	f.foreign = append(f.foreign, tupIF{id, t})
}

func (f *Ffa) set(t float64, why uint8) {
	if !f.on {
		f.on = true
		f.onT = t
		f.why = why
		if _, ok := os.LookupEnv("FFA_LOG"); ok {
			fmt.Fprintf(os.Stderr, "FFA seed=%d t=%.1f why=%d\n", f.seed, t, why)
		}
	}
}

func (f *Ffa) update(obs *Obs, tracks []Track, enabled bool) {
	t := obs.T
	me := &obs.Me
	if f.alive0 <= 0 {
		f.alive0 = max(obs.Alive, 1)
	}
	for ei := range obs.Events {
		ev := &obs.Events[ei]
		if ev.Kill != nil {
			f.kills++
		} else if ev.Hit != nil {
			h := ev.Hit
			found := false
			for i := range f.ourHits {
				if f.ourHits[i].x0 == h.Target {
					f.ourHits[i].x1 = t
					found = true
					break
				}
			}
			if !found {
				f.ourHits = append(f.ourHits, tupIF{h.Target, t})
			}
		}
	}
	if len(f.ourHits) > 64 {
		copy(f.ourHits, f.ourHits[1:])
		f.ourHits = f.ourHits[:len(f.ourHits)-1]
	}
	if !enabled {
		return
	}
	deaths := f.alive0 - obs.Alive
	if t < 75.0 && deaths >= f.kills+2 {
		f.set(t, 1)
	}
	isNpc := func(id int64) bool {
		for i := range tracks {
			if tracks[i].id == id && tracks[i].kind == ActorKindNpc {
				return true
			}
		}
		for i := range obs.Mechs {
			if obs.Mechs[i].Id == id && obs.Mechs[i].Kind == ActorKindNpc {
				return true
			}
		}
		return false
	}
	for bi := range obs.Beams {
		bm := &obs.Beams[bi]
		if bm.By == me.Id || isNpc(bm.By) {
			continue
		}
		if len(bm.Path) == 0 {
			continue
		}
		end := bm.Path[len(bm.Path)-1]
		if !(isFinite(end[0]) && isFinite(end[1]) && isFinite(end[2])) || dist(end, me.Pos) < 9.0 {
			continue
		}
		var hit3 *ActorObs
		for mi := range obs.Mechs {
			m := &obs.Mechs[mi]
			if m.Kind == ActorKindMech && m.Id != bm.By && m.Id != me.Id && dist(up(m.Pos, 4.5), end) < 5.5 {
				hit3 = m
				break
			}
		}
		if hit3 != nil {
			last := -1e9
			for _, x := range f.beamLast {
				if x.x0 == bm.By {
					last = x.x1
					break
				}
			}
			if t-last > 1.0 {
				kept := f.beamLast[:0]
				for _, x := range f.beamLast {
					if x.x0 != bm.By {
						kept = append(kept, x)
					}
				}
				f.beamLast = kept
				f.beamLast = append(f.beamLast, tupIF{bm.By, t})
				f.sig++
			}
			id := hit3.Id
			f.mark(id, t)
		}
	}
	var npcs []V3
	for mi := range obs.Mechs {
		if obs.Mechs[mi].Kind == ActorKindNpc {
			npcs = append(npcs, obs.Mechs[mi].Pos)
		}
	}
	zc := obs.Zone.Center
	zr := obs.Zone.Radius
	for mi := range obs.Mechs {
		m := &obs.Mechs[mi]
		if m.Kind != ActorKindMech || m.Id == me.Id {
			continue
		}
		if m.Hp == nil {
			continue
		}
		hp := *m.Hp
		hasPrev := false
		var prev tupIFF
		for _, x := range f.seenHp {
			if x.x0 == m.Id {
				prev = x
				hasPrev = true
				break
			}
		}
		if hasPrev {
			php, pt := prev.x1, prev.x2
			drop := php - hp
			ours := false
			for _, x := range f.ourHits {
				if x.x0 == m.Id && t-x.x1 < 1.2 {
					ours = true
					break
				}
			}
			inZone := dist2v(m.Pos, zc) < zr-2.0
			npcNear := false
			for _, q := range npcs {
				if dist(q, m.Pos) < 150.0 {
					npcNear = true
					break
				}
			}
			if t-pt < 0.6 && drop >= 45.0 && !ours && inZone && !npcNear {
				f.sig++
				f.mark(m.Id, t)
			}
		}
		found := false
		for i := range f.seenHp {
			if f.seenHp[i].x0 == m.Id {
				f.seenHp[i].x1 = hp
				f.seenHp[i].x2 = t
				found = true
				break
			}
		}
		if !found {
			f.seenHp = append(f.seenHp, tupIFF{m.Id, hp, t})
		}
	}
	if len(f.seenHp) > 32 {
		copy(f.seenHp, f.seenHp[1:])
		f.seenHp = f.seenHp[:len(f.seenHp)-1]
	}
	if f.sig >= 3 {
		f.set(t, 2)
	}
}

func (b *Brain) ffaPassive(obs *Obs) bool {
	return b.ffa.on && float64(obs.Alive) > b.p.FfaFairAlive && obs.T < b.p.FfaEndT
}

func (b *Brain) ffaEndRival(obs *Obs, tr *Track) bool {
	if b.p.FfaEndBonus <= 0.0 || obs.T < b.p.FfaEndT || obs.Zone.NextRadius > 30.0 {
		return false
	}
	c := obs.Zone.NextCenter
	mine := dist2v(obs.Me.Pos, c) - obs.Me.Hp/260.0
	theirs := dist2v(tr.posAt(obs.T), c) - tr.hp/260.0
	return theirs < mine
}

func (b *Brain) ffaAllow(obs *Obs, tr *Track, attacker bool, d float64) bool {
	if !b.ffaPassive(obs) {
		return true
	}
	t := obs.T
	p := b.p
	hpf := frac(obs.Me.Hp, obs.Me.MaxHp)
	if attacker && (d < p.FfaDefendRange || hpf > 0.5) {
		return true
	}
	others := false
	for i := range b.mem.tracks {
		o := &b.mem.tracks[i]
		if o.kind == ActorKindMech && o.id != tr.id && t-o.lastSeen < 4.0 && dist(o.posAt(t), obs.Me.Pos) < p.FfaCrowdRange {
			others = true
			break
		}
	}
	tf := frac(tr.hp, tr.maxHp)
	if tf < p.FfaSureHp && hpf > p.FfaMinHp && !others && d < 150.0 {
		return true
	}
	if t-b.ffa.foreignT(tr.id) < 3.0 && tf < p.FfaVultureHp && hpf > p.FfaMinHp-0.1 && d < 140.0 {
		return true
	}
	return false
}

func (b *Brain) ffaFireOk(obs *Obs, tr *Track, attacker bool, d float64, clear bool) bool {
	if attacker || (b.target.kind == TgtMech && b.target.id == tr.id) || d < 60.0 {
		return true
	}
	if !b.ffa.on {
		return b.p.FfaEarlyQuietRange <= 0.0 || obs.T >= 75.0 || d < b.p.FfaEarlyQuietRange
	}
	if !b.ffaPassive(obs) {
		return true
	}
	toUs := math.Atan2(obs.Me.Pos[1]-tr.pos[1], obs.Me.Pos[0]-tr.pos[0])
	return clear && d < 110.0 && math.Abs(adiff(tr.aim[0], toUs)) < 0.4
}

func (b *Brain) ffaAvoid(obs *Obs) bool {
	if !b.ffaPassive(obs) || b.p.FfaAvoidRange <= 0.0 {
		return false
	}
	t := obs.T
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind == ActorKindMech && t-tr.lastSeen < 1.0 && dist(tr.posAt(t), obs.Me.Pos) < b.p.FfaAvoidRange {
			return true
		}
	}
	return false
}

func (b *Brain) ffaQuiet(obs *Obs) float64 {
	if !b.ffaPassive(obs) || !obs.Me.InZone || obs.T-b.mem.lastDamageT < 4.0 {
		return 1.0
	}
	t := obs.T
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind == ActorKindMech && t-tr.lastSeen < 1.0 {
			return 1.0
		}
	}
	heard := false
	for i := range b.mem.sounds {
		s := &b.mem.sounds[i]
		if t-s.t < 4.0 && s.kind == "step" {
			heard = true
			break
		}
	}
	if !heard {
		return 1.0
	}
	walk := fmax(obs.Me.Stats.WalkSpeed, 1.0)
	return clampF(b.cfg.Sound.StepSpeed*0.92/walk, 0.2, 1.0)
}

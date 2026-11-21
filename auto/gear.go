package auto

import "math"

// Ported from src/brain/gear.rs.

// Gear is Rust `#[derive(Default)] struct Gear`; the zero value is Gear::default().
type Gear struct {
	sweepOn   bool
	sweepAcc  float64
	sweepYaw  float64
	lastSweep float64
	laserGoal bool
	jetScan   bool
	pursue    tupIF
}

func (b *Brain) gearOn() bool {
	return b.p.GearOn >= 0.5
}

func (b *Brain) gearHarmless(tr *Track, t float64, d float64) bool {
	p := b.p
	if tr.kind != ActorKindMech {
		return true
	}
	if b.sfNotHarmless(tr) {
		return false
	}
	if b.order.dmg(tr.id) >= p.GearHarmlessDmg {
		return false
	}
	if tr.has(WeaponKindLaser) && d < p.GearLaserThreatR {
		return false
	}
	if t >= p.GearLateS {
		return false
	}
	return true
}

func (b *Brain) gearNearReal(obs *Obs) float64 {
	t := obs.T
	near := math.Inf(1)
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech || t-tr.lastSeen > 1.5 {
			continue
		}
		d := dist(tr.posAt(t), obs.Me.Pos)
		if !b.gearHarmless(tr, t, d) {
			near = fmin(near, d)
		}
	}
	return near
}

func (b *Brain) gearTgtHarmless(obs *Obs, tgt *Track) bool {
	if tgt == nil {
		return true
	}
	d := dist(tgt.posAt(obs.T), obs.Me.Pos)
	return b.gearHarmless(tgt, obs.T, d)
}

// brainHasKind is Rust `Brain::has_kind` (associated fn in gear.rs).
func brainHasKind(obs *Obs, k WeaponKind) bool {
	for _, w := range obs.Me.Weapons {
		if w != nil && w.Kind == k {
			return true
		}
	}
	return false
}

func (b *Brain) gearGoal(obs *Obs, zi *ZoneInfo, tgt *Track) (tupIV, bool) {
	b.gear.laserGoal = false
	if !b.gearOn() || zi.urgent {
		return tupIV{}, false
	}
	p := b.p
	t := obs.T
	me := &obs.Me
	if !b.gearTgtHarmless(obs, tgt) {
		return tupIV{}, false
	}
	realD := b.gearNearReal(obs)
	tgtD := math.Inf(1)
	if tgt != nil {
		tgtD = dist(tgt.posAt(t), me.Pos)
	}
	hasBest := false
	var best tupFIVB
	if !brainHasKind(obs, WeaponKindLaser) && realD > p.GearSafeR {
		for i := range b.mem.pickups {
			it := &b.mem.pickups[i]
			if it.kind != WeaponKindLaser || b.mem.blacklisted(it.id, t) {
				continue
			}
			if _, ok := b.want(me, it.kind); !ok {
				continue
			}
			d := dist(it.pos, me.Pos)
			if b.sfFarBlock(d) {
				continue
			}
			if d > p.GearLaserMax || !b.inPlayPub(obs, zi, it.pos, d) {
				continue
			}
			gb := 0.0
			if it.id == b.goalID {
				gb = 15.0
			}
			s := d - 400.0 - gb
			if !hasBest || s < best.x0 {
				best = tupFIVB{s, it.id, it.pos, true}
				hasBest = true
			}
		}
	}
	anyEmpty := false
	for _, w := range me.Weapons {
		if w == nil {
			anyEmpty = true
			break
		}
	}
	if !hasBest && anyEmpty && realD > p.GearSafeR && tgtD > p.GearFightKeep {
		for i := range b.mem.pickups {
			it := &b.mem.pickups[i]
			if b.mem.blacklisted(it.id, t) {
				continue
			}
			if _, ok := b.want(me, it.kind); !ok {
				continue
			}
			d := dist(it.pos, me.Pos)
			if d > p.GearWeaponMax || !b.inPlayPub(obs, zi, it.pos, d) {
				continue
			}
			gb := 0.0
			if it.id == b.goalID {
				gb = 15.0
			}
			s := d - gb
			if !hasBest || s < best.x0 {
				best = tupFIVB{s, it.id, it.pos, false}
				hasBest = true
			}
		}
	}
	cubeCap := p.GearCubeCap
	wantsCube := me.Cubes < cubeCap || (me.Cubes >= 13.0 && me.Cubes < 15.0)
	if !hasBest && wantsCube && realD > p.GearCubeSafe {
		for i := range b.mem.cubes {
			it := &b.mem.cubes[i]
			if b.mem.blacklisted(it.id, t) {
				continue
			}
			d := dist(it.pos, me.Pos)
			lim := p.GearCubeR
			if tgtD < p.GearFightKeep {
				lim = p.GearCubeFightR
			}
			if d > lim || !b.inPlayPub(obs, zi, it.pos, d) {
				continue
			}
			gb := 0.0
			if it.id == b.goalID {
				gb = 10.0
			}
			s := d - gb
			if !hasBest || s < best.x0 {
				best = tupFIVB{s, it.id, it.pos, false}
				hasBest = true
			}
		}
	}
	if !hasBest && wantsCube && realD > p.GearCubeSafe && p.GearSiteR > 0.0 && tgtD > p.GearFightKeep {
		for _, s := range b.econ.sites {
			d := dist(s.x0, me.Pos)
			if t-s.x1 < 1e8 && t-s.x1 < p.GearSiteS && d < p.GearSiteR && d > 4.0 && b.inPlayPub(obs, zi, s.x0, d) && (!hasBest || d < best.x0) {
				best = tupFIVB{d, -1, s.x0, false}
				hasBest = true
			}
		}
	}
	if !hasBest && wantsCube && p.GearNpcGoalR > 0.0 && realD > p.GearSafeR && tgtD > p.GearFightKeep && frac(me.Hp, me.MaxHp) > 0.6 {
		for i := range b.mem.tracks {
			tr := &b.mem.tracks[i]
			if tr.kind != ActorKindNpc || t-tr.lastSeen > 20.0 || b.mem.blacklisted(tr.id, t) {
				continue
			}
			d := dist(tr.pos, me.Pos)
			if d > p.GearNpcGoalR || !b.inPlayPub(obs, zi, tr.pos, d) || (hasBest && d >= best.x0) {
				continue
			}
			k := 0.0
			if d > 1e-6 {
				k = fmin(p.GearNpcStand/d, 1.0)
			}
			q := V3{tr.pos[0] + (me.Pos[0]-tr.pos[0])*k, tr.pos[1] + (me.Pos[1]-tr.pos[1])*k, me.Pos[2]}
			if d <= p.GearNpcStand+8.0 {
				q = me.Pos
			}
			best = tupFIVB{d, tr.id, q, false}
			hasBest = true
		}
	}
	if hasBest && best.x1 >= 0 && !best.x3 && p.GearPursuitS > 0.0 {
		if b.gear.pursue.x0 != best.x1 {
			b.gear.pursue = tupIF{best.x1, t}
		} else if t-b.gear.pursue.x1 > p.GearPursuitS {
			b.mem.ban(best.x1, t+p.BlacklistS)
			b.gear.pursue = tupIF{-1, t}
			return tupIV{}, false
		}
	}
	if hasBest {
		b.gear.laserGoal = best.x3
		return tupIV{best.x1, best.x2}, true
	}
	return tupIV{}, false
}

func (b *Brain) gearGrab(obs *Obs, zi *ZoneInfo) (tupIV, bool) {
	p := b.p
	if !b.gearOn() || p.GearGrabR <= 0.0 || zi.urgent {
		return tupIV{}, false
	}
	t := obs.T
	me := &obs.Me
	wantsCube := me.Cubes < p.GearCubeCap || (me.Cubes >= 13.0 && me.Cubes < 15.0)
	hasBest := false
	var best tupFIV
	for i := range b.mem.pickups {
		it := &b.mem.pickups[i]
		if it.kind != WeaponKindLaser || b.mem.blacklisted(it.id, t) {
			continue
		}
		if _, ok := b.want(me, it.kind); !ok || math.Abs(it.pos[2]-me.Pos[2]) > 3.5 {
			continue
		}
		d := dist(it.pos, me.Pos)
		if d < p.GearGrabR && (!hasBest || d < best.x0) {
			best = tupFIV{d, it.id, it.pos}
			hasBest = true
		}
	}
	if !hasBest && wantsCube {
		for i := range b.mem.cubes {
			it := &b.mem.cubes[i]
			if b.mem.blacklisted(it.id, t) || math.Abs(it.pos[2]-me.Pos[2]) > 3.5 {
				continue
			}
			d := dist(it.pos, me.Pos)
			if d < p.GearGrabR && (!hasBest || d < best.x0) {
				best = tupFIV{d, it.id, it.pos}
				hasBest = true
			}
		}
	}
	if hasBest {
		return tupIV{best.x1, best.x2}, true
	}
	return tupIV{}, false
}

func (b *Brain) gearNpcOk(obs *Obs, tr *Track) bool {
	p := b.p
	if !b.gearOn() || p.GearNpcFireR <= 0.0 || b.mode == ModeRetreat {
		return false
	}
	me := &obs.Me
	d := dist(tr.pos, me.Pos)
	wantsCube := me.Cubes < p.GearCubeCap || (me.Cubes >= 13.0 && me.Cubes < 15.0)
	return wantsCube && d <= p.GearNpcFireR && frac(me.Hp, me.MaxHp) > 0.5
}

func (b *Brain) gearDropOk(obs *Obs, tgt *Track) bool {
	return b.gearOn() && b.p.GearDrop >= 0.5 && b.gearTgtHarmless(obs, tgt) && b.gearNearReal(obs) > b.p.GearSafeR
}

func (b *Brain) gearFarmMode(obs *Obs, tgt *Track) bool {
	if !b.gearOn() || b.p.GearFarm < 0.5 {
		return false
	}
	if tgt == nil {
		return false
	}
	tr := tgt
	d := dist(tr.posAt(obs.T), obs.Me.Pos)
	return b.gearHarmless(tr, obs.T, d) && d > b.p.GearFarmFar && b.gearNearReal(obs) > b.p.GearSafeR
}

func (b *Brain) gearSkipFire(tr *Track, t float64, d float64, clear bool) bool {
	if !b.gearOn() || b.p.GearFarm < 0.5 || !(b.mode == ModeHunt || b.mode == ModeLoot) {
		return false
	}
	if t-tr.lastDmgT < 1.0 && d < b.p.GearFarmFar {
		return false
	}
	return b.gearHarmless(tr, t, d) && (!clear || d > b.p.GearFarmFar)
}

func (b *Brain) gearRange(obs *Obs, tr *Track, pref float64) float64 {
	if !b.gearOn() || b.p.GearEasyRange <= 0.0 {
		return pref
	}
	d := dist(tr.posAt(obs.T), obs.Me.Pos)
	if b.gearHarmless(tr, obs.T, d) && !tr.has(WeaponKindLaser) {
		return fmin(pref, b.p.GearEasyRange)
	}
	return pref
}

func (b *Brain) gearJet(obs *Obs) (tupBB, bool) {
	if !b.gear.sweepOn || !b.gear.jetScan {
		return tupBB{}, false
	}
	me := &obs.Me
	if me.Energy < b.p.GearJetStop || b.sfQuietBlock() {
		b.gear.jetScan = false
		return tupBB{}, false
	}
	return tupBB{me.Grounded, !me.Grounded}, true
}

func (b *Brain) gearBoost(obs *Obs) bool {
	me := &obs.Me
	return b.gearOn() && b.p.GearBoostE > 0.0 && b.gear.laserGoal && b.mode == ModeLoot && me.Grounded && me.Energy > b.p.GearBoostE
}

func (b *Brain) gearSweep(obs *Obs, busy bool) (float64, bool) {
	p := b.p
	if !b.gearOn() || p.GearSweepEvery <= 0.0 {
		return 0, false
	}
	t := obs.T
	yaw := obs.Me.Aim[0]
	threat := func() bool {
		for i := range b.mem.tracks {
			tr := &b.mem.tracks[i]
			if tr.kind == ActorKindMech && t-tr.lastSeen < 2.0 {
				d := dist(tr.pos, obs.Me.Pos)
				if d < p.GearSweepMechR && (d < p.GearSweepAnyR || !b.gearHarmless(tr, t, d)) {
					return true
				}
			}
		}
		return false
	}
	calm := !busy &&
		!b.sfQuietBlock() &&
		t-b.mem.lastDamageT > 3.0 &&
		!threat() &&
		(b.mode == ModeIdle || b.mode == ModeLoot || b.mode == ModeHunt || b.mode == ModeZone || b.mode == ModeDrop)
	if b.gear.sweepOn {
		if !calm {
			b.gear.sweepOn = false
			return 0, false
		}
		b.gear.sweepAcc += math.Abs(adiff(yaw, b.gear.sweepYaw))
		b.gear.sweepYaw = yaw
		if b.gear.sweepAcc >= 2.0*math.Pi-0.4 {
			b.gear.sweepOn = false
			b.gear.lastSweep = t
			return 0, false
		}
		return wrap(yaw + 1.5), true
	}
	first := b.gear.lastSweep <= 0.0 && t < 3.0
	every := p.GearSweepEvery
	if p.GearSweepNolaser > 0.0 && !brainHasKind(obs, WeaponKindLaser) {
		every = p.GearSweepNolaser
	}
	if calm && (first || t-b.gear.lastSweep > every) {
		b.gear.sweepOn = true
		b.gear.sweepAcc = 0.0
		b.gear.sweepYaw = yaw
		b.gear.lastSweep = t
		b.gear.jetScan = p.GearJetScan > 0.0 && obs.Me.Energy >= p.GearJetScan && (first || !brainHasKind(obs, WeaponKindLaser))
		return wrap(yaw + 1.5), true
	}
	return 0, false
}

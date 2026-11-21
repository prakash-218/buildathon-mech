package auto

import (
	"math"
	"sort"
	"time"
)

// Ported from src/brain/strategy.rs.

type ZoneInfo struct {
	urgent  bool
	safe    V2
	curC    V2
	curR    float64
	closing bool
}

// zoneAt is the circle at time tau, given the zone reported at time now. Returns (center, radius).
func zoneAt(z *ZoneObs, now float64, tau float64, phaseStart bool) tupWF {
	var t0 float64
	if phaseStart {
		t0 = z.ShrinkStart
	} else {
		t0 = fmax(z.ShrinkStart, now)
	}
	span := z.ShrinkEnd - t0
	if tau <= t0 || !(span > 1e-6) {
		return tupWF{z.Center, z.Radius}
	}
	k := clampF((tau-t0)/span, 0.0, 1.0)
	return tupWF{
		V2{z.Center[0] + (z.NextCenter[0]-z.Center[0])*k, z.Center[1] + (z.NextCenter[1]-z.Center[1])*k},
		z.Radius + (z.NextRadius-z.Radius)*k,
	}
}

type strategyLootPick struct {
	id    int64
	pos   V3
	quick bool
}

// Option<(Tgt, Goal, i64)>
type strategyHunt struct {
	x0 Tgt
	x1 Goal
	x2 int64
}

func strategyGround(world *World, p V2) V3 {
	return V3{p[0], p[1], fin(world.terrainZ(p[0], p[1]), 0.0)}
}

func (b *Brain) safeRadius(z *ZoneObs) float64 {
	p := b.p
	if z.NextRadius <= p.FinalRadius {
		return 0.0
	}
	return fmax(z.NextRadius-fmin(p.ZoneMargin, z.NextRadius*0.5)-p.ZoneExtra, 0.0)
}

func (b *Brain) seedRadius(z *ZoneObs) float64 {
	return fmin(b.safeRadius(z), z.NextRadius*b.p.IdleFrac)
}

func (b *Brain) ensureZoneField(obs *Obs, world *World, nav *Nav) {
	z := &obs.Zone
	if zf := b.zoneField; zf != nil {
		c, r := zf.x0, zf.x1
		if math.Hypot(c[0]-z.NextCenter[0], c[1]-z.NextCenter[1]) < 0.5 && math.Abs(r-z.NextRadius) < 0.5 {
			return
		}
	}
	if !(isFinite(z.NextCenter[0]) && isFinite(z.NextCenter[1]) && isFinite(z.NextRadius)) {
		return
	}
	safeR := b.seedRadius(z)
	seeds := make([]tupVF, 0)
	c := z.NextCenter
	seeds = append(seeds, tupVF{strategyGround(world, c), 0.0})
	if safeR >= 8.0 {
		for _, rn := range []tupFN{{0.35, 8}, {0.7, 12}, {1.0, 20}} {
			fr, n := rn.x0, rn.x1
			for k := 0; k < n; k++ {
				a := float64(k) * 2.0 * math.Pi / float64(n)
				seeds = append(seeds, tupVF{strategyGround(world, V2{c[0] + safeR*fr*math.Cos(a), c[1] + safeR*fr*math.Sin(a)}), 0.0})
			}
		}
	} else {
		for _, rn := range []tupFN{{4.0, 6}, {9.0, 8}, {15.0, 10}, {23.0, 12}, {33.0, 14}} {
			r, n := rn.x0, rn.x1
			for k := 0; k < n; k++ {
				a := float64(k) * 2.0 * math.Pi / float64(n)
				seeds = append(seeds, tupVF{strategyGround(world, V2{c[0] + r*math.Cos(a), c[1] + r*math.Sin(a)}), 4.0 * r})
			}
		}
	}
	jet := -1.0
	if safeR < 8.0 {
		jet = obs.Me.MaxEnergy * 0.8
	}
	f := nav.fieldMulti(seeds, jet)
	b.zoneField = &brainZoneField{x0: c, x1: z.NextRadius, x2: f}
}

func (b *Brain) zoneInfo(obs *Obs, world *World, nav *Nav) ZoneInfo {
	b.ensureZoneField(obs, world, nav)
	p := b.p
	z := &obs.Zone
	t := obs.T
	pos := obs.Me.Pos
	za := zoneAt(z, t, t, false)
	cA, rA := za.x0, za.x1
	if !obs.Me.InZone && dist2v(pos, cA) < rA-1.0 && z.Radius > 0.0 {
		b.zoneB = true
	}
	zc := zoneAt(z, t, t, b.zoneB)
	curC, curR := zc.x0, zc.x1
	dn := dist2v(pos, z.NextCenter)
	isFinal := z.NextRadius <= p.FinalRadius
	margin := fmin(p.ZoneMargin, z.NextRadius*0.5)
	insideNext := dn <= z.NextRadius-margin
	safeR := b.safeRadius(z)
	_ = isFinal
	var safe V2
	if dn > safeR && dn > 1e-6 {
		safe = V2{z.NextCenter[0] + (pos[0]-z.NextCenter[0])/dn*safeR, z.NextCenter[1] + (pos[1]-z.NextCenter[1])/dn*safeR}
	} else {
		safe = V2{pos[0], pos[1]}
	}
	dNeed := fmax(dn-(z.NextRadius-margin), 0.0)
	outNow := dist2v(pos, curC) - curR
	tHit := math.Inf(1)
	if z.ShrinkEnd > t {
		tau := t
		for tau <= z.ShrinkEnd+0.5 {
			zt := zoneAt(z, t, tau, b.zoneB)
			if dist2v(pos, zt.x0) > zt.x1-p.ZoneEdge {
				tHit = tau
				break
			}
			tau += 0.5
		}
	}
	walk := fmax(obs.Me.Stats.WalkSpeed, 1.0)
	dPath, dPathOk := 0.0, false
	if zf := b.zoneField; zf != nil {
		if d, ok := nav.dist(zf.x2, pos); ok && isFinite(d) {
			dPath, dPathOk = d, true
		}
	}
	seedR := b.seedRadius(z)
	straight := fmax(dn-seedR, 0.0)
	var needS float64
	if dPathOk {
		needS = fmax(dPath-fmax(z.NextRadius-margin-seedR, 0.0), dNeed) * 1.15 / walk
	} else {
		needS = dNeed * p.PathFactor / walk
	}
	detourS := 0.0
	if dPathOk {
		detourS = fmax(dPath-straight, 0.0) / walk
	}
	urgent := !obs.Me.InZone ||
		outNow > -p.ZoneEdge ||
		(tHit-t) < p.ZoneLeadS+detourS ||
		(!insideNext && (z.ShrinkEnd-t) < needS+p.ZoneSlackS)
	closing := !insideNext && (z.ShrinkStart-t) < 25.0
	return ZoneInfo{urgent: urgent, safe: safe, curC: curC, curR: curR, closing: closing}
}

func (b *Brain) decide(obs *Obs, world *World, nav *Nav) {
	p := b.p
	me := &obs.Me
	t := obs.T
	pos := me.Pos
	hpf := frac(me.Hp, me.MaxHp)
	zi := b.zoneInfo(obs, world, nav)
	b.sqUpdate(obs, nav)
	muzzle := up(pos, me.Stats.MuzzleHeight)

	if b.goalUnreachable {
		if b.goalID >= 0 {
			b.mem.ban(b.goalID, t+p.BlacklistS)
		}
		b.goalUnreachable = false
	}
	if b.goalID >= 0 {
		if b.pursuit.x0 != b.goalID {
			b.pursuit = tupIF{b.goalID, t}
		} else if t-b.pursuit.x1 > p.PursuitMaxS {
			b.mem.ban(b.goalID, t+p.BlacklistS)
			b.pursuit = tupIF{-1, t}
		}
	}

	nAtt := 0
	nearMech := math.Inf(1)
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech {
			continue
		}
		if t-tr.lastDmgT < 2.0 {
			nAtt += 1
		}
		if t-tr.lastSeen < 1.0 {
			nearMech = fmin(nearMech, dist(tr.pos, pos))
		}
	}
	threatNear := nearMech < 200.0 || t-b.mem.lastDamageT < 3.0
	laserNear := false
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind == ActorKindMech && t-tr.lastSeen < 1.0 && tr.has(WeaponKindLaser) && dist(tr.posAt(t), pos) < 130.0 && world.sight(up(tr.posAt(t), b.cfg.Mech.EyeHeight), up(pos, me.Stats.Height*0.6)) {
			laserNear = true
			break
		}
	}
	pressured := nAtt >= 2 || (laserNear && t-b.mem.lastDamageT < 4.0)
	held := b.held(t)
	var tgt *Track // Option<Track>
	if held {
		tgt = nil
	} else if b.orderOn() {
		if tr, ok := b.chooseTargetOrder(obs, &zi); ok {
			tgt = &tr
		}
	} else {
		if tr, ok := b.chooseTarget(obs, &zi); ok {
			tgt = &tr
		}
	}
	regen := b.sqRegen(obs)
	if regen {
		if tgt != nil && !(t-tgt.lastDmgT < p.SqRegenFightS) {
			tgt = nil
		}
	}
	nearMem := math.Inf(1)
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind == ActorKindMech && t-tr.lastSeen < 4.0 {
			nearMem = fmin(nearMem, dist(tr.posAt(t), pos))
		}
	}

	was := b.mode == ModeRetreat
	var wantRetreat bool
	if was {
		wantRetreat = hpf < p.RetreatExitHp
	} else {
		wantRetreat = (hpf < p.RetreatHp && threatNear) || (nAtt >= 2 && hpf < p.RetreatMultiHp) || (pressured && hpf < p.RetreatPressHp)
	}
	if wantRetreat && !was {
		if tr := tgt; tr != nil {
			if nAtt <= 1 && tr.hp < me.Hp*p.FinishRatio && t-tr.lastSeen < 0.3 {
				wantRetreat = false
			}
		}
	}

	if held && (nAtt >= 1 || nearMech < 120.0 || nearMem < 100.0 || t-b.mem.lastDamageT < 3.0 || (was && nearMem < 170.0)) {
		wantRetreat = true
	}
	if !wantRetreat && (b.sqGate(obs, tgt) || regen && tgt == nil) {
		wantRetreat = true
	}
	avoid := !wantRetreat && tgt == nil && b.sqAvoid(obs)
	loot, lootOk := b.chooseLoot(obs, &zi)
	var harvest tupIV
	harvestOk := false
	if !held {
		harvest, harvestOk = b.harvest(obs, &zi, nAtt, nearMech)
	}
	var gearg tupIV
	gearOk := false
	if held {
		gearOk = false
	} else if wantRetreat || avoid {
		gearg, gearOk = b.gearGrab(obs, &zi)
	} else {
		gearg, gearOk = b.gearGoal(obs, &zi, tgt)
		if !gearOk {
			if p.GearGrabAlways > 0.5 {
				gearg, gearOk = b.gearGrab(obs, &zi)
			}
		}
	}
	farm := !held && b.gearFarmMode(obs, tgt)
	gearDrop := b.gearDropOk(obs, tgt)
	var huntNear float64
	if b.gearOn() {
		huntNear = b.gearNearReal(obs)
	} else {
		huntNear = nearMech
	}
	rest := !wantRetreat && !held && b.wantRest(obs, &zi, nAtt, nearMech, hpf)
	if wantRetreat || held {
		b.econ.resting = false
	}
	drop, dropOk := b.chooseDrop(obs, &zi)
	if dropOk && !(!held || nearMem > 100.0) {
		dropOk = false
	}
	z := &obs.Zone
	endgame := held && z.NextRadius <= 0.5 && z.ShrinkEnd-t < 40.0
	var mode Mode
	var goal Goal
	target := Tgt{kind: TgtNone}
	goalID := int64(-1)
	if tr := tgt; tr != nil {
		target = Tgt{kind: TgtMech, id: tr.id}
	}

	if endgame {
		mode = ModeZone
		goal = Goal{kind: GoalField, center: z.NextCenter, arrive: 0.8}
	} else if zi.urgent {
		mode = ModeZone
		goal = Goal{kind: GoalField, center: obs.Zone.NextCenter, arrive: 1.5}
	} else if gearOk && (wantRetreat || avoid) {
		mode = ModeLoot
		goal = Goal{kind: GoalNav, pos: gearg.x1, arrive: 1.0}
		goalID = gearg.x0
	} else if wantRetreat || avoid {
		mode = ModeRetreat
		goal = b.coverGoal(obs, world, nav, &zi)
	} else if gearOk {
		mode = ModeLoot
		goal = Goal{kind: GoalNav, pos: gearg.x1, arrive: 1.0}
		goalID = gearg.x0
	} else if harvestOk {
		mode = ModeLoot
		goal = Goal{kind: GoalNav, pos: harvest.x1, arrive: 1.0}
		goalID = harvest.x0
	} else if rest {
		mode = ModeRetreat
		goal = b.coverGoal(obs, world, nav, &zi)
	} else if lootOk && loot.quick {
		mode = ModeLoot
		goal = Goal{kind: GoalNav, pos: loot.pos, arrive: 1.2}
		goalID = loot.id
	} else if dropOk && (tgt == nil || drop.x2 < 50.0 || hpf < 0.6 || gearDrop) {
		mode = ModeDrop
		goal = Goal{kind: GoalNav, pos: drop.x0, arrive: fmax(drop.x1*0.3, 1.0)}
	} else if tgt != nil && (!farm || func() bool {
		_, ok := b.chooseHunt(obs, world, &zi, hpf, huntNear, muzzle)
		return !ok
	}()) {
		tr := tgt
		mode = ModeFight
		tp := tr.posAt(t)
		c := up(tp, b.cfg.Mech.Height*0.5)
		los := false
		if t-tr.lastSeen < p.FreshS {
			_, hit := world.ray(muzzle, c)
			los = !hit
		}
		if los || t-tr.lastDmgT < 3.0 {
			b.noLos = tupIF{tr.id, t}
		} else if b.noLos.x0 != tr.id {
			b.noLos = tupIF{tr.id, t}
		} else {
			mult := 1.0
			if b.finishBonus(tr) > 0.0 {
				mult = p.FinishChaseMult
			}
			if t-b.noLos.x1 > p.NoLosDropS*mult {
				b.mem.ban(tr.id, t+8.0)
			}
		}
		pref := b.gearRange(obs, tr, b.prefRange(obs, tr, hpf))
		var g0 Goal
		if los {
			g0 = Goal{kind: GoalRange, pos: tp, pref: pref}
		} else {
			g0 = Goal{kind: GoalNav, pos: tp, arrive: fmax(pref*0.5, 15.0)}
		}
		if b.p.SvSpot > 0.0 {
			b.mode = ModeFight
		}
		goal = b.spotGoal(obs, world, nav, tr, pref, g0)
	} else if lootOk {
		mode = ModeLoot
		goal = Goal{kind: GoalNav, pos: loot.pos, arrive: 1.2}
		goalID = loot.id
	} else {
		mode = ModeIdle
		if h, ok := b.chooseHunt(obs, world, &zi, hpf, huntNear, muzzle); ok {
			mode = ModeHunt
			target = h.x0
			goal = h.x1
			goalID = h.x2
		} else if q, ok := b.prepGoal(); ok {
			goal = Goal{kind: GoalNav, pos: q, arrive: 3.0}
		} else {
			goal = Goal{kind: GoalField, center: obs.Zone.NextCenter, arrive: 2.0}
		}
	}
	if mode == ModeIdle || mode == ModeHunt || mode == ModeDrop || (mode == ModeLoot && goalID >= 0 && (!lootOk || !loot.quick)) {
		if b.ffaAvoid(obs) {
			mode = ModeRetreat
			goal = b.coverGoal(obs, world, nav, &zi)
			target = Tgt{kind: TgtNone}
			goalID = -1
		}
	}
	b.allowCrates = false
	if (mode == ModeHunt || mode == ModeLoot || mode == ModeIdle) && hpf >= p.CrateMinHp {
		var safe float64
		if b.gateClosed(obs) {
			safe = p.ReadySafeRange
		} else {
			safe = p.HuntSafeRange
		}
		b.allowCrates = huntNear > safe
	}
	if mode != ModeRetreat {
		b.cover = nil
	}
	if goalID != b.goalID {
		b.pursuit = tupIF{goalID, t}
	}
	b.mode = mode
	b.goal = goal
	b.goalID = goalID
	b.target = target
	b.roofDecide(obs, world, nav, &zi)
}

// chooseTarget returns Rust's Option<Track> as (track, ok).
func (b *Brain) chooseTarget(obs *Obs, zi *ZoneInfo) (Track, bool) {
	p := b.p
	t := obs.T
	pos := obs.Me.Pos
	z := &obs.Zone
	gate := b.gateClosed(obs)
	var best *Track
	bestS := 0.0
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech || t-tr.lastSeen > p.TargetMemoryS {
			continue
		}
		if gate && b.spare(tr.hp, tr.maxHp) {
			continue
		}
		tp := tr.posAt(t)
		d := dist(tp, pos)
		attacker := t-tr.lastDmgT < 5.0
		if !attacker && b.mem.blacklisted(tr.id, t) {
			continue
		}
		if !b.sqAllow(obs, tr) || b.sfHold(obs, tr) {
			continue
		}
		if d > p.EngageRange && !attacker {
			continue
		}
		if !b.ffaAllow(obs, tr, attacker, d) {
			continue
		}
		if d > 60.0 {
			if dist2v(tp, zi.curC) > zi.curR-2.0 {
				continue
			}
			if zi.closing && dist2v(tp, z.NextCenter) > z.NextRadius+p.ChaseOutNext {
				continue
			}
		}
		s := 60.0*(1.0-d/200.0) + 80.0*(1.0-frac(tr.hp, tr.maxHp))
		if attacker {
			s += 100.0 + tr.dmgToUs*0.1
		}
		if b.target == (Tgt{kind: TgtMech, id: tr.id}) {
			s += p.TargetSticky
			s += b.finishBonus(tr)
		}
		if b.ffaEndRival(obs, tr) {
			s += p.FfaEndBonus
		}
		if t-tr.lastSeen < p.FreshS {
			s += 30.0
		}
		if best == nil || s > bestS {
			bestS, best = s, tr
		}
	}
	if best == nil {
		return Track{}, false
	}
	return *best, true
}

func (b *Brain) prefRange(obs *Obs, tr *Track, hpf float64) float64 {
	p := b.p
	me := &obs.Me
	has := func(k WeaponKind) bool {
		for _, w := range me.Weapons {
			if w != nil && w.Kind == k {
				return true
			}
		}
		return false
	}
	pref := p.RangeGun
	if has(WeaponKindLaser) {
		pref = p.RangeLaser
	} else if has(WeaponKindMortar) && !has(WeaponKindGun) {
		pref = p.RangeMortar
	}
	if has(WeaponKindLightning) && me.Energy > 35.0 && hpf > p.LightningRushHp && (p.RushSolo <= 0.0 || b.rushOk(obs, tr)) {
		pref = p.RangeLightning
	} else if tr.has(WeaponKindLightning) {
		pref = fmax(pref, p.RangeVsLightning)
	}
	return pref
}

func (b *Brain) inPlayPub(obs *Obs, zi *ZoneInfo, q V3, d float64) bool {
	return b.inPlay(obs, zi, q, d)
}

func (b *Brain) inPlay(obs *Obs, zi *ZoneInfo, q V3, d float64) bool {
	z := &obs.Zone
	if dist2v(q, zi.curC) > zi.curR-6.0 {
		return false
	}
	walk := fmax(obs.Me.Stats.WalkSpeed, 1.0)
	outNext := dist2v(q, z.NextCenter) - z.NextRadius
	if outNext > -5.0 && (z.ShrinkEnd-obs.T) < 25.0+d/walk+fmax(outNext, 0.0)/walk {
		return false
	}
	return true
}

// chooseLoot returns Rust's Option<LootPick> as (pick, ok).
func (b *Brain) chooseLoot(obs *Obs, zi *ZoneInfo) (strategyLootPick, bool) {
	p := b.p
	me := &obs.Me
	t := obs.T
	var best strategyLootPick
	bestS := 0.0
	has := false
	for i := range b.mem.pickups {
		it := &b.mem.pickups[i]
		if b.mem.blacklisted(it.id, t) {
			continue
		}
		slot, ok := b.want(me, it.kind)
		if !ok {
			continue
		}
		d := dist(it.pos, me.Pos)
		empty := slot == nil
		lim := p.LootSwapMax
		if empty {
			lim = p.LootWeaponMax
		}
		if d > lim || !b.inPlay(obs, zi, it.pos, d) {
			continue
		}
		e := 0.0
		if empty {
			e = 80.0
		}
		s := d - e - b.prefOf(it.kind)*5.0
		if it.id == b.goalID {
			s -= 15.0
		}
		if !has || s < bestS {
			has = true
			bestS = s
			best = strategyLootPick{id: it.id, pos: it.pos, quick: d < p.LootQuick && empty}
		}
	}
	if me.Cubes < b.cfg.Cubes.StatCap {
		for i := range b.mem.cubes {
			it := &b.mem.cubes[i]
			if b.mem.blacklisted(it.id, t) {
				continue
			}
			d := dist(it.pos, me.Pos)
			if d > p.LootCubeMax || !b.inPlay(obs, zi, it.pos, d) {
				continue
			}
			s := d - 20.0
			if it.id == b.goalID {
				s -= 15.0
			}
			if !has || s < bestS {
				has = true
				bestS = s
				best = strategyLootPick{id: it.id, pos: it.pos, quick: d < p.LootQuick*0.5}
			}
		}
	}
	return best, has
}

// chooseDrop returns Rust's Option<(V3, f64, f64)> = (pos, radius, distance).
func (b *Brain) chooseDrop(obs *Obs, zi *ZoneInfo) (tupVFF, bool) {
	p := b.p
	me := &obs.Me
	walk := fmax(me.Stats.WalkSpeed, 1.0)
	var best tupVFF
	has := false
	for i := range obs.Drops {
		d := &obs.Drops[i]
		if d.Holder != nil {
			h := *d.Holder
			if h != me.Id && d.Progress > d.CaptureS*0.5 {
				continue
			}
		}
		dd := dist(d.Pos, me.Pos)
		if dd > p.DropMaxDist || !memoryAllFinite(d.Pos) {
			continue
		}
		if d.LandsIn > dd*p.PathFactor/walk+p.DropEarlyS {
			continue
		}
		if dist2v(d.Pos, zi.curC) > zi.curR-4.0 {
			continue
		}
		if !has || dd < best.x2 {
			has = true
			best = tupVFF{d.Pos, fmax(d.Radius, 2.0), dd}
		}
	}
	return best, has
}

// chooseHunt returns Rust's Option<(Tgt, Goal, i64)> as (hunt, ok).
func (b *Brain) chooseHunt(obs *Obs, world *World, zi *ZoneInfo, hpf float64, nearMech float64, muzzle V3) (strategyHunt, bool) {
	p := b.p
	me := &obs.Me
	t := obs.T
	var safe float64
	if b.gateClosed(obs) {
		safe = p.ReadySafeRange
	} else {
		safe = p.HuntSafeRange
	}
	if hpf < p.HuntMinHp || nearMech < safe {
		return strategyHunt{}, false
	}
	moreCubes := me.Cubes < b.cfg.Cubes.StatCap
	emptySlot := false
	for _, w := range me.Weapons {
		if w == nil {
			emptySlot = true
			break
		}
	}
	var best strategyHunt
	bestS := 0.0
	has := false
	if moreCubes {
		for i := range b.mem.tracks {
			tr := &b.mem.tracks[i]
			if tr.kind != ActorKindNpc || b.mem.blacklisted(tr.id, t) {
				continue
			}
			d := dist(tr.pos, me.Pos)
			if d > p.HuntMaxDist || !b.inPlay(obs, zi, tr.pos, d) {
				continue
			}
			s := d
			if b.goalID == tr.id {
				s -= 20.0
			}
			if !has || s < bestS {
				c := up(tr.pos, b.cfg.Npc.Height*0.5)
				los := false
				if t-tr.lastSeen < p.FreshS {
					_, hit := world.ray(muzzle, c)
					los = !hit
				}
				var g Goal
				if los {
					g = Goal{kind: GoalRange, pos: tr.pos, pref: p.RangeNpc}
				} else {
					g = Goal{kind: GoalNav, pos: tr.pos, arrive: p.RangeNpc * 0.8}
				}
				has = true
				bestS = s
				best = strategyHunt{Tgt{kind: TgtNpc, id: tr.id}, g, tr.id}
			}
		}
	}
	if (moreCubes || emptySlot) && hpf >= p.CrateMinHp {
		for i := range b.mem.crates {
			it := &b.mem.crates[i]
			if b.mem.blacklisted(it.id, t) || it.hp <= 0.0 {
				continue
			}
			d := dist(it.pos, me.Pos)
			if d > p.CrateMaxDist || !b.inPlay(obs, zi, it.pos, d) {
				continue
			}
			s := d - 20.0
			if b.goalID == it.id {
				s -= 20.0
			}
			if !has || s < bestS {
				c := up(it.pos, b.cfg.Crates.Height*0.5)
				_, hit := world.ray(muzzle, c)
				los := !hit
				var g Goal
				if los && d < 120.0 {
					g = Goal{kind: GoalRange, pos: it.pos, pref: p.RangeCrate}
				} else {
					g = Goal{kind: GoalNav, pos: it.pos, arrive: p.RangeCrate}
				}
				has = true
				bestS = s
				best = strategyHunt{Tgt{kind: TgtCrate, id: it.id}, g, it.id}
			}
		}
	}
	return best, has
}

func (b *Brain) threatEyes(obs *Obs) []V3 {
	t := obs.T
	pos := obs.Me.Pos
	v := make([]tupFV, 0)
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech || t-tr.lastSeen > 3.0 {
			continue
		}
		tp := tr.posAt(t)
		s := 1.0 + b.cfg.Cubes.SizePerCube*fmax(tr.cubes, 0.0)
		v = append(v, tupFV{dist(tp, pos), up(tp, b.cfg.Mech.EyeHeight*s)})
	}
	if u := b.mem.unseen; u != nil {
		if t-u.t < 4.0 {
			q := V3{u.from[0] + math.Cos(u.bearing)*b.p.GhostRange, u.from[1] + math.Sin(u.bearing)*b.p.GhostRange, u.from[2] + b.cfg.Mech.EyeHeight}
			v = append(v, tupFV{b.p.GhostRange, q})
		}
	}
	sort.SliceStable(v, func(i, j int) bool { return totalCmp(v[i].x0, v[j].x0) < 0 })
	if len(v) > 4 {
		v = v[:4]
	}
	out := make([]V3, 0, len(v))
	for _, x := range v {
		out = append(out, x.x1)
	}
	return out
}

func (b *Brain) coverGoal(obs *Obs, world *World, nav *Nav, zi *ZoneInfo) Goal {
	p := b.p
	t := obs.T
	me := &obs.Me
	pos := me.Pos
	if cv := b.cover; cv != nil {
		c, tc := cv.x0, cv.x1
		if t-tc < p.CoverPeriodS {
			return Goal{kind: GoalNav, pos: c, arrive: 1.5}
		}
	}
	eyes := b.threatEyes(obs)
	if len(eyes) == 0 {
		b.cover = nil
		return Goal{kind: GoalNone}
	}
	h := fmax(me.Stats.Height, 1.0)
	hidden := func(q V3) bool {
		for _, e := range eyes {
			if world.sight(e, up(q, h*0.5)) {
				return false
			}
			if world.sight(e, up(q, h*0.92)) {
				return false
			}
		}
		return true
	}
	z := &obs.Zone
	var best tupFV
	hasBest := false
	consider := func(q V3, bonus float64) {
		if dist2v(q, zi.curC) > zi.curR-12.0 {
			return
		}
		if !hidden(q) {
			return
		}
		outNext := fmax(dist2v(q, z.NextCenter)-z.NextRadius+10.0, 0.0)
		nearThreat := math.Inf(1)
		for _, e := range eyes {
			nearThreat = fmin(nearThreat, dist2(e, q))
		}
		s := dist(q, pos) + 0.6*outNext - 0.15*fmin(nearThreat, 150.0) - bonus
		if !hasBest || s < best.x0 {
			hasBest = true
			best = tupFV{s, q}
		}
	}
	if cv := b.cover; cv != nil {
		consider(cv.x0, 10.0)
	}
	consider(pos, 6.0)
	tAct := b.tAct
	for _, r := range []float64{12.0, 24.0, 40.0, 60.0} {
		if time.Since(tAct).Seconds() > 0.010 {
			break
		}
		for k := 0; k < 12; k++ {
			a := float64(k)*math.Pi/6.0 + r*0.1
			x := pos[0] + r*math.Cos(a)
			y := pos[1] + r*math.Sin(a)
			q0 := V3{x, y, fin(world.floorBelow(x, y, pos[2]+3.0), pos[2])}
			q, ok := nav.snap(q0)
			if !ok {
				continue
			}
			if dist(q, q0) > 8.0 || !memoryAllFinite(q) {
				continue
			}
			consider(q, 0.0)
		}
	}
	var q V3
	if hasBest {
		q = best.x1
	} else {
		e := eyes[0]
		away := unit2(pos[0]-e[0], pos[1]-e[1])
		tc := unit2(zi.safe[0]-pos[0], zi.safe[1]-pos[1])
		d := unit2(away[0]+0.4*tc[0], away[1]+0.4*tc[1])
		x := pos[0] + d[0]*p.FleeDist
		y := pos[1] + d[1]*p.FleeDist
		q0 := V3{x, y, fin(world.floorBelow(x, y, pos[2]+3.0), pos[2])}
		if s, ok := nav.snap(q0); ok && memoryAllFinite(s) {
			q = s
		} else {
			q = q0
		}
	}
	b.cover = &tupVF{q, t}
	return Goal{kind: GoalNav, pos: q, arrive: 1.5}
}

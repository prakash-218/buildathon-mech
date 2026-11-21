package auto

import "math"

// Ported from src/brain/aim.rs.

type AimTarget struct {
	feet   V3
	vel    V3
	acc    V3
	radius float64
	height float64
	hp     float64
	fixed  bool
}

type FireOut struct {
	aim      V2
	fire     [2]bool
	reload   bool
	target   *AimTarget
	armed    bool
	charging bool
}

// (AimTarget, bool)
type aimPick struct {
	x0 AimTarget
	x1 bool
}

// aimRayHit is Rust `world.ray(a, b).is_some()`.
func aimRayHit(world *World, a V3, b V3) bool {
	_, hit := world.ray(a, b)
	return hit
}

func (b *Brain) aimTarget(tr *Track) AimTarget {
	var radius, height float64
	if tr.kind == ActorKindNpc {
		radius, height = b.cfg.Npc.Radius, b.cfg.Npc.Height
	} else {
		s := 1.0 + b.cfg.Cubes.SizePerCube*fmax(tr.cubes, 0.0)
		radius, height = b.cfg.Mech.Radius*s, b.cfg.Mech.Height*s
	}
	return AimTarget{feet: tr.pos, vel: tr.vel, acc: tr.acc, radius: radius, height: height, hp: tr.hp, fixed: false}
}

func (b *Brain) idleAim(obs *Obs) V2 {
	t := obs.T
	if u := b.mem.unseen; u != nil {
		if t-u.t < 3.0 {
			return V2{u.bearing, 0.0}
		}
	}
	hasBest := false
	var best tupFF
	for i := range b.mem.sounds {
		s := &b.mem.sounds[i]
		if t-s.t > 2.0 {
			continue
		}
		k := 0.0
		switch s.kind {
		case "gun", "laser", "laser_charge", "lightning", "mortar", "step", "boost", "jet", "reload":
			k = 1.0
		}
		d := 0.0
		switch s.dist {
		case SoundDistNear:
			d = 3.0
		case SoundDistMid:
			d = 1.5
		}
		pr := k*d - (t-s.t)*0.3
		if k*d > 0.0 && (!hasBest || pr > best.x0) {
			best = tupFF{pr, s.bearing}
			hasBest = true
		}
	}
	if hasBest {
		if a, ok := b.threatAim(obs); ok {
			return a
		}
		return V2{best.x1, 0.0}
	}
	if a, ok := b.threatAim(obs); ok {
		return a
	}
	var base float64
	if math.Hypot(b.lastDir[0], b.lastDir[1]) > 0.1 {
		base = math.Atan2(b.lastDir[1], b.lastDir[0])
	} else {
		base = obs.Me.Aim[0]
	}
	ph := 2.0 * math.Pi * t / fmax(b.p.ScanPeriod, 0.5)
	return V2{wrap(base + b.p.ScanAmp*math.Sin(ph)), 0.0}
}

func (b *Brain) pickFireTarget(obs *Obs, world *World, muzzle V3) (aimPick, bool) {
	t := obs.T
	me := &obs.Me
	maxRange := 0.0
	for _, w := range me.Weapons {
		if w == nil {
			continue
		}
		var r float64
		switch w.Kind {
		case WeaponKindGun:
			r = w.Stats.Range * b.p.GunRangeFrac
		case WeaponKindMortar:
			if w.Stats.Gravity > 0.0 {
				r = w.Stats.Speed * w.Stats.Speed / w.Stats.Gravity * 0.95
			} else {
				r = 0.0
			}
		default:
			r = w.Stats.Range
		}
		maxRange = fmax(maxRange, r)
	}
	hasBest := false
	var bestS float64
	var bestAt AimTarget
	var bestAllow bool
	held := b.held(t)
	gate := b.gateClosed(obs)
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if held || tr.kind != ActorKindMech || t-tr.lastSeen > 0.15 {
			continue
		}
		if gate && b.spare(tr.hp, tr.maxHp) {
			continue
		}
		if !b.orderFireOk(tr, obs) {
			continue
		}
		if b.roofHold && t-tr.lastDmgT > 1.5 {
			continue
		}
		at := b.aimTarget(tr)
		c := up(at.feet, at.height*0.5)
		d := dist(c, muzzle)
		clear := !aimRayHit(world, muzzle, c)
		attacker := t-tr.lastDmgT < 3.0
		if b.sfHold(obs, tr) || b.gearSkipFire(tr, t, d, clear) {
			continue
		}
		if !b.ffaFireOk(obs, tr, attacker, d, clear) {
			continue
		}
		s := -0.3*d + 40.0*(1.0-frac(tr.hp, tr.maxHp))
		if clear {
			s += 100.0
		}
		if attacker {
			s += 40.0
		}
		if b.target == (Tgt{kind: TgtMech, id: tr.id}) {
			s += 50.0
		}
		if d < maxRange {
			s += 50.0
		}
		s += b.finishBonus(tr)
		toUs := math.Atan2(me.Pos[1]-tr.pos[1], me.Pos[0]-tr.pos[0])
		aligned := math.Abs(adiff(tr.aim[0], toUs)) < 0.5
		var allow bool
		if b.econ.resting {
			allow = attacker || d < 45.0
		} else {
			allow = b.mode != ModeRetreat || attacker || (aligned && clear) || d < 60.0
		}
		allow = allow && !b.holdFire(obs, d, tr.hp)
		if !hasBest || s > bestS {
			bestS, bestAt, bestAllow = s, at, allow
			hasBest = true
		}
	}
	if hasBest {
		return aimPick{bestAt, bestAllow}, true
	}
	hasBn := false
	var bnS float64
	var bnAt AimTarget
	var bnAllow bool
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindNpc || t-tr.lastSeen > 0.15 {
			continue
		}
		attacker := t-tr.lastDmgT < 3.0
		hunted := b.target == (Tgt{kind: TgtNpc, id: tr.id}) || b.gearNpcOk(obs, tr)
		if (!attacker && !hunted) || (b.roofHold && t-tr.lastDmgT > 1.5) {
			continue
		}
		d := dist(tr.pos, me.Pos)
		hb := 0.0
		if hunted {
			hb = 50.0
		}
		s := -d + hb
		if !hasBn || s > bnS {
			bnS, bnAt, bnAllow = s, b.aimTarget(tr), b.mode != ModeRetreat || attacker
			hasBn = true
		}
	}
	if hasBn {
		return aimPick{bnAt, bnAllow}, true
	}
	if !b.allowCrates {
		return aimPick{}, false
	}
	hasBc := false
	var bcS float64
	var bcAt AimTarget
	rays := 0
	for i := range b.mem.crates {
		it := &b.mem.crates[i]
		if t-it.lastSeen > 0.3 || it.hp <= 0.0 || b.mem.blacklisted(it.id, t) {
			continue
		}
		c := up(it.pos, b.cfg.Crates.Height*0.5)
		d := dist(c, muzzle)
		if d > fmin(maxRange, 170.0) {
			continue
		}
		tb := 0.0
		if b.target == (Tgt{kind: TgtCrate, id: it.id}) {
			tb = 60.0
		}
		s := -d + tb
		if (!hasBc || s > bcS) && rays < 8 {
			rays++
			if !aimRayHit(world, muzzle, c) {
				at := AimTarget{feet: it.pos, vel: V3{}, acc: V3{}, radius: b.cfg.Crates.Radius, height: b.cfg.Crates.Height, hp: it.hp, fixed: true}
				bcS, bcAt = s, at
				hasBc = true
			}
		}
	}
	if hasBc {
		return aimPick{bcAt, true}, true
	}
	return aimPick{}, false
}

func (b *Brain) mortarSolve(obs *Obs, world *World, tg *AimTarget, w *WeaponObs) (V2, bool) {
	p := b.p
	st := &obs.Me.Stats
	tube := up(obs.Me.Pos, st.Height+0.5)
	v := w.Stats.Speed
	g := w.Stats.Gravity
	if !(v > 0.0) || !(g > 0.0) {
		return V2{}, false
	}
	vxy := V3{tg.vel[0], tg.vel[1], 0.0}
	speed := math.Hypot(vxy[0], vxy[1])
	if speed > p.MortarMaxTargetSpeed {
		return V2{}, false
	}
	for _, high := range [2]bool{false, true} {
		tf := dist2(tg.feet, tube) / (v * 0.9)
		hasSol := false
		var sol tupFVF
		for it := 0; it < 3; it++ {
			tp := add(up(tg.feet, 0.5), scale(vxy, tf+p.Latency))
			d := dist2(tp, tube)
			if ms, ok := mortarSolution(d, tp[2]-tube[2], v, g, high); ok {
				tf = ms.x1
				sol = tupFVF{ms.x0, tp, d}
				hasSol = true
			} else {
				hasSol = false
				break
			}
		}
		if !hasSol {
			continue
		}
		pitch, tp, d := sol.x0, sol.x1, sol.x2
		if d < p.MortarMinRange || pitch < st.MinPitch+0.01 || pitch > st.MaxPitch-0.01 {
			continue
		}
		if tf > p.MortarMaxFlight && speed > 2.0 && !tg.fixed {
			continue
		}
		if w.Stats.Ttl > 0.0 && tf > w.Stats.Ttl*0.95 {
			continue
		}
		yaw := math.Atan2(tp[1]-tube[1], tp[0]-tube[0])
		vel := scale(aimDir(yaw, pitch), v)
		prev := tube
		clear := true
		n := 10
		for k := 1; k <= n; k++ {
			tt := tf * 0.93 * float64(k) / float64(n)
			q := V3{tube[0] + vel[0]*tt, tube[1] + vel[1]*tt, tube[2] + vel[2]*tt - 0.5*g*tt*tt}
			if aimRayHit(world, prev, q) {
				clear = false
				break
			}
			prev = q
		}
		if clear {
			return V2{yaw, pitch}, true
		}
	}
	return V2{}, false
}

func (b *Brain) fireControl(obs *Obs, world *World) FireOut {
	p := b.p
	me := &obs.Me
	t := obs.T
	st := &me.Stats
	muzzle := up(me.Pos, st.MuzzleHeight)
	out := FireOut{aim: b.idleAim(obs), fire: [2]bool{false, false}, reload: false, target: nil, armed: false, charging: false}
	tgt, hasTgt := b.pickFireTarget(obs, world, muzzle)
	if u := b.mem.unseen; u != nil {
		attackedByTgt := false
		if hasTgt {
			for i := range b.mem.tracks {
				tr := &b.mem.tracks[i]
				if t-tr.lastDmgT < 1.5 && t-tr.lastSeen < 0.3 {
					attackedByTgt = true
					break
				}
			}
		}
		if t-u.t < 1.2 && !attackedByTgt {
			if hasTgt {
				tg := &tgt.x0
				c := up(tg.feet, tg.height*0.5)
				if aimRayHit(world, muzzle, c) || dist(c, muzzle) > 60.0 {
					hasTgt = false
				}
			}
			if !hasTgt {
				out.aim = V2{u.bearing, 0.0}
			}
		}
	}

	if b.held(t) {
		hasFace := false
		var face tupFV
		for i := range b.mem.tracks {
			tr := &b.mem.tracks[i]
			if tr.kind != ActorKindMech || t-tr.lastSeen > 2.5 {
				continue
			}
			tp := tr.posAt(t)
			d := dist(tp, me.Pos)
			db := 0.0
			if t-tr.lastDmgT < 3.0 {
				db = 200.0
			}
			s := d - db
			if t-tr.lastDmgT < 3.0 && (!hasFace || s < face.x0) {
				face = tupFV{s, tp}
				hasFace = true
			}
		}
		if hasFace {
			hasTgt = false
			out.aim = dirToAim(sub(up(face.x1, b.cfg.Mech.Height*0.5), muzzle))
		}
	}
	if !hasTgt {
		b.holdLostCharge(obs, &out)
		b.precharge(obs, &out)
		for _, w := range me.Weapons {
			if w == nil {
				continue
			}
			if w.Kind == WeaponKindGun && w.Reload <= 0.0 && w.Ammo < w.Stats.Mag {
				if w.Ammo <= w.Stats.Mag*p.ReloadFrac || t-b.lastTargetT > 1.5 {
					out.reload = true
				}
			}
		}
		return out
	}
	tg := tgt.x0
	allow := tgt.x1
	b.lastTargetT = t
	c := up(tg.feet, tg.height*0.5)
	rel := sub(c, muzzle)
	d := fmax(vlen(rel), 0.1)
	direct := dirToAim(add(rel, scale(tg.vel, p.LaserLatency)))
	los := !aimRayHit(world, muzzle, c)
	tolY := math.Atan2(tg.radius*p.AimTol, d)
	tolP := math.Atan2(tg.height*0.5*p.AimTol, d)

	// gun: Option<(usize, V2, bool)>
	hasGun := false
	gunI := 0
	var gunA V2
	gunOkT := false
	// laser / lightning: Option<usize>
	hasLaser, laserI := false, 0
	hasLightning, lightningI := false, 0
	// mortar: Option<(usize, V2)>
	hasMortar, mortarI := false, 0
	var mortarA V2
	for i, w := range me.Weapons {
		if w == nil {
			continue
		}
		switch w.Kind {
		case WeaponKindGun:
			acc := tg.acc
			if tg.fixed {
				acc = V3{}
			}
			zon := p.AimzOn > 0.0 && !tg.fixed
			lg := p.AccGain
			if zon {
				lg = 0.0
			}
			if ld, ok := lead(rel, tg.vel, acc, fmax(w.Stats.Speed, 1.0), p.Latency, lg, p.AccTime); ok {
				pt, tf := ld.x0, ld.x1
				pzOk := true
				if zon {
					hl := fmax(math.Hypot(pt[0], pt[1]), 0.1)
					nn := V2{-pt[1] / hl, pt[0] / hl}
					hw := tg.radius + b.cfg.Projectile.GunRadius
					if q, ok := b.aimz.query(t, tg.feet, tf+p.Latency, nn, hw, p.AimzShrink, f2n(fmax(p.AimzMinN, 3.0)), f2n(fmax(p.AimzShotMin, 0.0)), p.AimzGshot > 0.0); ok {
						lat, pr := q.x0, q.x1
						cur := (pt[0]-rel[0])*nn[0] + (pt[1]-rel[1])*nn[1]
						pt[0] += nn[0] * (lat - cur)
						pt[1] += nn[1] * (lat - cur)
						pzOk = pr >= p.AimzPmin
					}
				}
				gok := pzOk && vlen(pt) <= w.Stats.Range*p.GunRangeFrac && !aimRayHit(world, muzzle, add(muzzle, pt))
				hasGun, gunI, gunA, gunOkT = true, i, dirToAim(pt), gok
			}
		case WeaponKindLaser:
			hasLaser, laserI = true, i
		case WeaponKindLightning:
			hasLightning, lightningI = true, i
		case WeaponKindMortar:
			if w.Cooldown <= 0.25 && allow {
				if a, ok := b.mortarSolve(obs, world, &tg, w); ok {
					hasMortar, mortarI, mortarA = true, i, a
				} else {
					hasMortar = false
				}
			}
			if w.Cooldown <= 1.0 {
				out.armed = true
			}
		case WeaponKindUnknown:
		}
	}

	// laser_w: Option<(usize, &WeaponObs)>
	var laserW *WeaponObs
	laserWI := 0
	if hasLaser {
		if w := me.Weapons[laserI]; w != nil {
			laserW, laserWI = w, laserI
		}
	}
	laserIn := laserW != nil && d <= laserW.Stats.Range*0.98 && los
	laserCharging := laserW != nil && laserW.Charge > 0.0
	gunOk := hasGun && gunOkT
	cmd := direct
	if hasGun {
		cmd = gunA
	}
	laserFinal := false
	if laserW != nil {
		w := laserW
		full := fmax(w.Stats.Charge, 0.05)
		if laserCharging && w.Charge >= full-0.2 {
			cmd = direct
			laserFinal = true
		} else if !gunOk && (laserCharging || laserIn) {
			cmd = direct
		}
	}
	mortarPrimary := false
	if !laserFinal {
		if hasMortar {
			mw := me.Weapons[mortarI]
			if (mw != nil && mw.Cooldown <= 0.11) || !gunOk {
				cmd = mortarA
				mortarPrimary = true
			}
		}
	}
	if !gunOk && !mortarPrimary && !laserFinal {
		if hasLightning {
			if w := me.Weapons[lightningI]; w != nil && d <= w.Stats.Range*1.3 {
				cmd = direct
			}
		}
	}
	pred := slew(me.Aim, cmd, st.TurnRate*0.05, st.PitchRate*0.05)
	on := func(a V2, ty float64, tp float64) bool {
		return math.Abs(adiff(pred[0], a[0]))*fmax(math.Cos(a[1]), 0.2) <= ty && math.Abs(pred[1]-a[1]) <= tp
	}

	if hasGun {
		i, a, ok := gunI, gunA, gunOkT
		if w := me.Weapons[i]; w != nil {
			if w.Ammo <= 0.0 && w.Reload <= 0.0 {
				out.reload = true
			}
			if !ok && w.Reload <= 0.0 && w.Ammo <= w.Stats.Mag*p.ReloadFrac && w.Ammo < w.Stats.Mag {
				out.reload = true
			}
			if w.Ammo > 0.0 && w.Reload <= 0.7 {
				out.armed = true
			}
			if allow && ok && w.Ammo > 0.0 && w.Reload <= 0.0 && on(a, tolY, tolP) {
				out.fire[i] = true
				if p.AimzOn > 0.0 && !tg.fixed {
					b.aimz.noteFire(t, tg.feet)
				}
			}
		}
	}
	if laserW != nil {
		i, w := laserWI, laserW
		full := fmax(w.Stats.Charge, 0.05)
		if w.Cooldown <= 1.0 {
			out.armed = true
		}
		if !laserCharging {
			if allow && laserIn && w.Cooldown <= 0.1 {
				out.fire[i] = true
			}
		} else {
			ready := w.Charge >= full-0.051
			aligned := on(direct, tolY, tolP)
			part := w.Stats.Damage * (w.Stats.MinFrac + (1.0-w.Stats.MinFrac)*clampF(w.Charge/full, 0.0, 1.0))
			lethal := w.Charge >= w.Stats.MinCharge && tg.hp > 0.0 && part >= tg.hp*1.05
			release := (ready || lethal) && aligned && laserIn
			giveUp := !laserIn && w.Charge >= full+fmax(w.Stats.HoldS-0.4, 0.0)
			out.fire[i] = !(release || giveUp)
		}
		out.charging = out.fire[i]
	}
	if hasLightning {
		i := lightningI
		if w := me.Weapons[i]; w != nil {
			inRange := d <= w.Stats.Range*0.95 && dist(c, up(me.Pos, st.Height*0.5)) <= w.Stats.Range*0.95
			if inRange {
				out.armed = true
			}
			yawTo := math.Atan2(rel[1], rel[0])
			// Rust f64::to_radians is `x * (PI / 180.0)`.
			cone := w.Stats.ConeDeg * (math.Pi / 180.0) * p.LightningConeFrac
			if allow && inRange && los && math.Abs(adiff(pred[0], yawTo)) <= cone && me.Energy >= fmax(p.LightningMinEnergy, w.Stats.Energy) {
				out.fire[i] = true
			}
		}
	}
	if mortarPrimary {
		if hasMortar {
			i, a := mortarI, mortarA
			if on(a, math.Atan2(tg.radius+1.5, d), 0.02) {
				out.fire[i] = true
			}
		}
	}
	out.aim = cmd
	out.target = &tg
	return out
}

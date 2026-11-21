package auto

import (
	"math"
	"sort"
)

// Ported from src/brain/plan.rs.

type MoveOut struct {
	dir   V2
	boost bool
	jump  bool
}

type planThreat struct {
	eye        V3
	aimYaw     float64
	gun        bool
	laser      bool
	lightning  bool
	mortar     bool
	ghost      bool
	isTarget   bool
	chargeLeft *float64
	losNow     bool
	dist       float64
}

type planShot struct {
	p   V3
	v   V3
	dmg float64
	r   float64
}

type planBlast struct {
	at     V3
	t      float64
	splash float64
	radius float64
	edge   float64
	direct float64
}

type planCand struct {
	dir     V2
	speed   float64
	boost   bool
	pts     []V3
	blocked float64
	score   float64
}

// (Vec<V3>, f64)
type planSimRes struct {
	x0 []V3
	x1 float64
}

// (Vec<Shot>, Vec<Blast>)
type planIncoming struct {
	x0 []planShot
	x1 []planBlast
}

const planDT float64 = 0.1

// planF2U32 is Rust's saturating `x as u32` (NaN -> 0).
func planF2U32(x float64) uint32 {
	if !(x > 0) {
		return 0
	}
	if x >= 4294967295.0 {
		return math.MaxUint32
	}
	return uint32(x)
}

func planSimulate(world *World, me *SelfObs, dir V2, speed float64, mult float64, boost bool, n int) planSimRes {
	st := &me.Stats
	ok := func(v V3) V3 { return V3{fin(v[0], 0.0), fin(v[1], 0.0), fin(v[2], 0.0)} }
	body := Body{pos: ok(me.Pos), vel: ok(me.Vel), grounded: me.Grounded, energy: fin(me.Energy, 0.0), energyWait: 0.0}
	mv := V2{0.0, 0.0}
	if speed > 0.0 {
		mv = dir
	}
	d := Drive{mv: mv, boost: boost, jump: false, speedMult: mult}
	pts := make([]V3, 0, n+1)
	pts = append(pts, body.pos)
	blocked := 0.0
	for i := 0; i < n; i++ {
		before := body.pos
		info := step(world, st, &body, &d, planDT)
		if info.blocked {
			if dist2(body.pos, before) < 0.3*speed*planDT {
				blocked += 1.0
			} else {
				blocked += 0.3
			}
		}
		pts = append(pts, body.pos)
	}
	return planSimRes{pts, blocked}
}

func (b *Brain) threatDps(th *planThreat, d float64) float64 {
	p := b.p
	if th.ghost {
		if d < 280.0 {
			return p.GhostDps
		}
		return 0.0
	}
	w := &b.cfg.Weapons
	s := 0.0
	if th.gun && d < w.Gun.Range {
		s += p.GunDps * clampF(1.15-d/220.0, 0.2, 1.0)
	}
	if th.laser && d < w.Laser.Range {
		s += p.LaserDps
	}
	if th.lightning && d < w.Lightning.Range+6.0 {
		s += p.LightningDps
	}
	if th.mortar && d > 20.0 && d < 235.0 {
		s += p.MortarDps
	}
	return s
}

func (b *Brain) threats(obs *Obs, world *World, center V3) []planThreat {
	t := obs.T
	v := make([]planThreat, 0)
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech || t-tr.lastSeen > 3.0 {
			continue
		}
		tp := tr.posAt(t)
		s := 1.0 + b.cfg.Cubes.SizePerCube*fmax(tr.cubes, 0.0)
		fresh := t-tr.lastSeen < b.p.FreshS
		var chargeLeft *float64
		if tr.charging && fresh {
			cl := clampF(b.cfg.Weapons.Laser.Charge-(t-tr.chargeSince), 0.1, 0.8)
			chargeLeft = &cl
		}
		v = append(v, planThreat{
			eye:        up(tp, b.cfg.Mech.EyeHeight*s),
			aimYaw:     tr.aim[0],
			gun:        tr.has(WeaponKindGun) || len(tr.weapons) == 0,
			laser:      tr.has(WeaponKindLaser),
			lightning:  tr.has(WeaponKindLightning),
			mortar:     tr.has(WeaponKindMortar),
			ghost:      false,
			isTarget:   b.target == (Tgt{kind: TgtMech, id: tr.id}),
			chargeLeft: chargeLeft,
			losNow:     false,
			dist:       dist(tp, center),
		})
	}
	if u := b.mem.unseen; u != nil {
		if t-u.t < 4.0 {
			r := b.p.GhostRange
			q := V3{u.from[0] + math.Cos(u.bearing)*r, u.from[1] + math.Sin(u.bearing)*r, u.from[2] + b.cfg.Mech.EyeHeight}
			explained := false
			for k := range v {
				th := &v[k]
				if th.dist < 250.0 && math.Abs(adiff(math.Atan2(th.eye[1]-center[1], th.eye[0]-center[0]), u.bearing)) < 0.35 {
					explained = true
					break
				}
			}
			if !explained {
				v = append(v, planThreat{eye: q, aimYaw: wrap(u.bearing + math.Pi), gun: true, laser: u.laser, lightning: false, mortar: false, ghost: true, isTarget: false, chargeLeft: nil, losNow: false, dist: r})
			}
		}
	}
	sort.SliceStable(v, func(i, j int) bool { return totalCmp(v[i].dist, v[j].dist) < 0 })
	if n := f2n(fmax(b.p.MaxThreats, 0.0)); len(v) > n {
		v = v[:n]
	}
	for k := range v {
		v[k].losNow = world.sight(v[k].eye, center)
	}
	return v
}

func (b *Brain) incoming(obs *Obs, world *World, center V3) planIncoming {
	shots := make([]planShot, 0)
	blasts := make([]planBlast, 0)
	m := &b.cfg.Weapons.Mortar
	np := min(len(obs.Projectiles), 64)
	for i := 0; i < np; i++ {
		pr := &obs.Projectiles[i]
		if !(memoryAllFinite(pr.Pos) && memoryAllFinite(pr.Vel)) {
			continue
		}
		if pr.Kind == ProjKindMortar {
			g := fmax(m.Gravity, 1.0)
			q := pr.Pos
			v := pr.Vel
			tt := 0.0
			h := 0.05
			hit := false
			for tt < clampF(m.Ttl, 1.0, 8.0) {
				v[2] -= g * h
				q = add(q, scale(v, h))
				tt += h
				if world.solid(q[0], q[1], q[2]) || q[2] < world.terrainZ(q[0], q[1]) {
					hit = true
					break
				}
			}
			if hit && dist(q, center) < 60.0 {
				blasts = append(blasts, planBlast{at: q, t: tt, splash: m.Splash, radius: fmax(m.SplashRadius, 1.0), edge: m.SplashEdge, direct: m.Damage})
			}
		} else {
			rel := sub(pr.Pos, center)
			cx := closestXy(rel, pr.Vel, 0.0, 1.2)
			dxy := cx.x1
			if dxy > 30.0 {
				continue
			}
			var dmg, r float64
			if pr.Kind == ProjKindNpc {
				dmg, r = b.cfg.Npc.Damage, b.cfg.Projectile.NpcRadius
			} else {
				dmg, r = b.cfg.Weapons.Gun.Damage, b.cfg.Projectile.GunRadius
			}
			shots = append(shots, planShot{p: pr.Pos, v: pr.Vel, dmg: dmg, r: r})
		}
	}
	return planIncoming{shots, blasts}
}

func (b *Brain) incomingCost(c *planCand, shots []planShot, blasts []planBlast, radius float64, height float64) float64 {
	margin := fmax(b.p.ProjMargin, 0.1)
	n := len(c.pts) - 1
	cost := 0.0
	for si := range shots {
		s := &shots[si]
		worst := 0.0
		for k := 0; k < n; k++ {
			a := c.pts[k]
			bb := c.pts[k+1]
			ov := scale(sub(bb, a), 1.0/planDT)
			t0 := float64(k) * planDT
			sp := add(s.p, scale(s.v, t0))
			rel := sub(sp, up(a, height*0.5))
			rv := sub(s.v, ov)
			cx := closestXy(rel, rv, 0.0, planDT)
			dxy, dz := cx.x1, cx.x2
			if math.Abs(dz) > height*0.5+margin {
				continue
			}
			reach := radius + s.r + margin
			if dxy < reach {
				worst = fmax(worst, clampF((reach-dxy)/margin, 0.0, 1.0))
			}
		}
		cost += worst * s.dmg
	}
	for bi := range blasts {
		bl := &blasts[bi]
		kf := bl.t / planDT
		var q V3
		if kf >= float64(n) {
			last := c.pts[n]
			prev := c.pts[n-1]
			ext := fmin(bl.t-float64(n)*planDT, 0.8)
			q = add(last, scale(sub(last, prev), ext/planDT))
		} else {
			k := f2n(math.Floor(kf))
			f := kf - float64(k)
			q = add(scale(c.pts[k], 1.0-f), scale(c.pts[min(k+1, n)], f))
		}
		d := fmax(dist2(q, bl.at)-radius, 0.0)
		if math.Abs(q[2]-bl.at[2]) > height+bl.radius {
			continue
		}
		if d < bl.radius+3.0 {
			fall := fmin(d/bl.radius, 1.0)
			dmg := bl.splash * (1.0 - (1.0-bl.edge)*fall)
			if d <= 0.3 {
				dmg += bl.direct
			}
			if d > bl.radius {
				dmg *= 0.3 * (1.0 - (d-bl.radius)/3.0)
			}
			cost += dmg
		}
	}
	return cost
}

func (b *Brain) planMove(obs *Obs, world *World, nav *Nav, budget *Budget, fc *FireOut) MoveOut {
	p := b.p
	me := &obs.Me
	st := &me.Stats
	t := obs.T
	pos := me.Pos
	n := min(max(f2n(p.HorizonSteps), 2), 16)
	horizon := float64(n) * planDT
	radius := fmax(st.Radius, 0.5)
	height := fmax(st.Height, 1.0)
	center := up(pos, height*0.5)
	walk := fmax(st.WalkSpeed, 0.1)

	if jt := b.jet; jt != nil {
		tg, t0 := jt.x0, jt.x1
		dh := dist2(pos, tg)
		landed := me.Grounded && t-t0 > 0.35
		if landed || t-t0 > 4.0 || (me.Energy < 2.0 && me.Grounded) {
			b.jet = nil
		} else {
			d := V2{0.0, 0.0}
			if dh > 0.6 {
				d = unit2(tg[0]-pos[0], tg[1]-pos[1])
			}
			rise := pos[2] < tg[2]+1.5
			if math.Hypot(d[0], d[1]) > 0.1 {
				b.lastDir = d
			}
			return MoveOut{dir: d, boost: !me.Grounded && rise, jump: me.Grounded}
		}
	}

	if t < b.unstickUntil {
		d := b.unstickDir
		b.lastDir = d
		jet := b.unstickLevel >= 2 && !me.Grounded && me.Energy > 15.0
		return MoveOut{dir: d, boost: jet, jump: me.Grounded}
	}

	// nav field
	d0, d0Ok := 0.0, false
	navdir, navdirOk := V2{}, false
	goalPos, goalPosOk := V3{}, false
	arrive := 0.0
	var fld *Field
	navWants := false
	if b.goal.kind == GoalField {
		gc := b.goal.center
		arrive = b.goal.arrive
		g := V3{gc[0], gc[1], pos[2]}
		goalPos, goalPosOk = g, true
		if zf := b.zoneField; zf != nil {
			f := zf.x2
			if d, ok := nav.dist(f, pos); ok && isFinite(d) {
				d0, d0Ok = d, true
				stp, stpOk := nav.step(f, pos)
				if stpOk {
					s := stp
					if s.kind == edgeJet && me.Grounded && me.Energy >= s.energy+8.0 && dist2(pos, s.target) < 14.0 {
						b.jet = &tupVF{s.target, t}
						dd := unit2(s.target[0]-pos[0], s.target[1]-pos[1])
						b.lastDir = dd
						return MoveOut{dir: dd, boost: false, jump: true}
					}
				}
				navdir, navdirOk = V2{}, false
				if stpOk {
					v := stp.dir
					if isFinite(v[0]) && isFinite(v[1]) && math.Hypot(v[0], v[1]) > 0.2 {
						navdir, navdirOk = v, true
					}
				}
				navWants = navdirOk
				fld = f
			}
		}
		if !d0Ok {
			d0, d0Ok = dist2(pos, g), true
		}
		if !navdirOk {
			navdir, navdirOk = unit2(g[0]-pos[0], g[1]-pos[1]), true
		}
	}
	if b.goal.kind == GoalNav {
		g := b.goal.pos
		goalPos, goalPosOk = g, true
		arrive = b.goal.arrive
		calm := b.roofJet || (t-b.mem.lastDamageT > 3.0 && !(b.mode == ModeFight || b.mode == ModeRetreat))
		jetE := -1.0
		if calm {
			jetE = me.MaxEnergy * 0.8
		}
		need := true
		if fl := b.field; fl != nil {
			fg, f, tb := fl.x0, fl.x1, fl.x2
			moved := dist(fg, g)
			need = (moved > p.FieldRegoal && (t-tb >= 0.3 || moved > 25.0)) || ((f.jetLimit() >= 0.0) != (jetE >= 0.0) && t-tb >= 0.5)
		}
		if need && budget.remainingMs() > p.FieldMinBudgetMs && (b.field == nil || b.spentMs() < 8.0) {
			sg := g
			if q, ok := nav.snap(g); ok && memoryAllFinite(q) {
				sg = q
			}
			lim := dist2(pos, sg)*2.0 + 120.0
			f := nav.fieldLimited([]tupVF{{sg, 0.0}}, jetE, lim)
			b.field = &brainField{x0: g, x1: f, x2: t}
		}
		if fl := b.field; fl != nil {
			fg, f := fl.x0, fl.x1
			stale := dist(fg, g)
			if stale <= 25.0 {
				if d, ok := nav.dist(f, pos); ok && isFinite(d) {
					d0, d0Ok = d, true
					stp, stpOk := nav.step(f, pos)
					if stpOk {
						s := stp
						if s.kind == edgeJet && me.Grounded && me.Energy >= s.energy+8.0 && dist2(pos, s.target) < 14.0 {
							b.jet = &tupVF{s.target, t}
							dd := unit2(s.target[0]-pos[0], s.target[1]-pos[1])
							b.lastDir = dd
							return MoveOut{dir: dd, boost: false, jump: true}
						}
					}
					navdir, navdirOk = V2{}, false
					if stpOk {
						v := stp.dir
						if isFinite(v[0]) && isFinite(v[1]) && math.Hypot(v[0], v[1]) > 0.2 {
							navdir, navdirOk = v, true
						}
					}
					navWants = navdirOk
					fld = f
				} else {
					if stale <= p.FieldRegoal {
						b.goalUnreachable = true
					}
				}
			}
		}
		if !d0Ok {
			d0, d0Ok = dist2(pos, g), true
		}
		if !navdirOk {
			navdir, navdirOk = unit2(g[0]-pos[0], g[1]-pos[1]), true
		}
	}
	var reached bool
	switch b.goal.kind {
	case GoalNav:
		if d0Ok {
			reached = d0 <= arrive || dist2(pos, b.goal.pos) <= fmin(arrive, 1.0)
		} else {
			reached = false
		}
	case GoalField:
		if d0Ok {
			reached = d0 <= arrive
		} else {
			reached = false
		}
	default:
		reached = true
	}

	// stuck
	pursuing := (b.goal.kind == GoalNav || b.goal.kind == GoalField) && !reached && !(b.mode == ModeFight || b.mode == ModeRetreat)
	if !(pursuing && (navWants || fld == nil) && (d0Ok && d0 > arrive+6.0)) {
		b.wantsSince = t
	}
	if !pursuing {
		b.movingSince = t
	} else if t-b.movingSince >= p.StuckWindowS && (t-b.cmdSince >= p.StuckWindowS || t-b.wantsSince >= p.StuckWindowS) {
		if mv, ok := b.mem.moved(t, p.StuckWindowS); ok {
			if mv < p.StuckDist {
				if t-b.unstickUntil < 6.0 {
					b.unstickLevel = b.unstickLevel + 1
				} else {
					b.unstickLevel = 1
				}
				base := V2{1.0, 0.0}
				if navdirOk && math.Hypot(navdir[0], navdir[1]) > 0.5 {
					base = navdir
				}
				var turn float64
				if b.unstickLevel%2 == 0 {
					turn = b.rng.rangeF(-0.3, 0.3)
				} else {
					sgn := 1.0
					if b.rng.f64() < 0.5 {
						sgn = -1.0
					}
					turn = sgn * b.rng.rangeF(0.9, 2.2)
				}
				a := math.Atan2(base[1], base[0]) + turn
				b.unstickDir = V2{math.Cos(a), math.Sin(a)}
				b.unstickUntil = t + p.UnstickS
				b.unstickCount += 1
				b.movingSince = t + p.UnstickS
				b.cmdSince = t + p.UnstickS
				b.wantsSince = t + p.UnstickS
				b.lastDir = b.unstickDir
				return MoveOut{dir: b.unstickDir, boost: false, jump: me.Grounded}
			}
		}
	}

	threats := b.threats(obs, world, center)
	inc := b.incoming(obs, world, center)
	shots, blasts := inc.x0, inc.x1
	npcs := make([]V3, 0)
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind == ActorKindNpc && t-tr.lastSeen < 2.0 && b.target != (Tgt{kind: TgtNpc, id: tr.id}) {
			npcs = append(npcs, tr.pos)
		}
	}

	// juke axis
	juke, jukeOk := V2{}, false
	for k := range threats {
		th := &threats[k]
		if th.ghost || !th.losNow || !(th.gun || th.mortar) || th.dist < p.JukeMinDist {
			continue
		}
		toUs := math.Atan2(center[1]-th.eye[1], center[0]-th.eye[0])
		if math.Abs(adiff(th.aimYaw, toUs)) < 0.35 {
			u := unit2(center[0]-th.eye[0], center[1]-th.eye[1])
			juke, jukeOk = V2{-u[1], u[0]}, true
			break
		}
	}
	if jukeOk && t >= b.jukeNext {
		b.jukeSide = -b.jukeSide
		b.jukeNext = t + b.rng.rangeF(p.JukeMinS, fmax(p.JukeMaxS, p.JukeMinS))
	}
	wJuke := p.WJuke
	if p.WCad > 0.0 {
		var pick tupFW
		pickOk := false
		for i := range obs.Mechs {
			m := &obs.Mechs[i]
			if m.Kind != ActorKindMech {
				continue
			}
			d := dist2(m.Pos, pos)
			if d < p.CadMin || d > p.CadMax {
				continue
			}
			if b.cad.due(m.Id, t, p.CadPeriod, p.CadWin, p.CadKeep, p.CadOff, planF2U32(fmax(p.CadReg, 0.0))) && (!pickOk || d < pick.x0) {
				u := unit2(pos[0]-m.Pos[0], pos[1]-m.Pos[1])
				pick, pickOk = tupFW{d, V2{-u[1], u[0]}}, true
			}
		}
		if pickOk {
			ax := pick.x1
			if t >= b.cadUntil-p.CadPeriod {
				b.jukeSide = -b.jukeSide
			}
			b.jukeNext = t + p.CadPeriod + 0.05
			b.cadUntil = t + p.CadPeriod + 0.05
			b.cadAxis = ax
		}
		if t < b.cadUntil {
			juke, jukeOk = b.cadAxis, true
			wJuke = p.WCad
		}
	}

	underFire := t-b.mem.lastDamageT < 1.0 || len(shots) > 0 || len(blasts) > 0
	if !underFire {
		for k := range threats {
			if threats[k].chargeLeft != nil && threats[k].losNow {
				underFire = true
				break
			}
		}
	}
	hasLightning := false
	for _, w := range me.Weapons {
		if w != nil && w.Kind == WeaponKindLightning {
			hasLightning = true
			break
		}
	}
	var reserve float64
	if b.mode == ModeZone {
		reserve = 3.0
	} else if hasLightning {
		reserve = p.EnergyReserveLightning
	} else {
		reserve = p.EnergyReserve
	}
	allowBoost := me.Grounded && me.Energy > reserve && (b.mode == ModeZone || b.mode == ModeRetreat || b.mode == ModeDrop || b.mode == ModeRoof || underFire || !me.InZone)
	slow := 1.0
	if fc.charging {
		slow = clampF(st.ChargeMoveMult, 0.1, 1.0)
	}

	// candidates
	var base V2
	if navdirOk && math.Hypot(navdir[0], navdir[1]) > 0.5 {
		base = navdir
	} else if math.Hypot(b.lastDir[0], b.lastDir[1]) > 0.5 {
		base = b.lastDir
	} else {
		base = V2{1.0, 0.0}
	}
	a0 := math.Atan2(base[1], base[0])
	nd := min(max(f2n(p.NDirs), 4), 32)
	nb := min(max(f2n(p.NBoostDirs), 0), 32)
	cands := make([]planCand, 0, nd+nb+1)
	cands = append(cands, planCand{dir: V2{0.0, 0.0}, speed: 0.0, boost: false, pts: nil, blocked: 0.0, score: 0.0})
	for k := 0; k < nd; k++ {
		a := a0 + float64(k)*2.0*math.Pi/float64(nd)
		cands = append(cands, planCand{dir: V2{math.Cos(a), math.Sin(a)}, speed: walk * slow, boost: false, pts: nil, blocked: 0.0, score: 0.0})
	}
	if allowBoost {
		for k := 0; k < nb; k++ {
			a := a0 + float64(k)*2.0*math.Pi/float64(nb)
			cands = append(cands, planCand{dir: V2{math.Cos(a), math.Sin(a)}, speed: fmax(st.BoostSpeed, walk) * slow, boost: true, pts: nil, blocked: 0.0, score: 0.0})
		}
	}

	var wProg float64
	switch b.mode {
	case ModeIdle:
		wProg = p.WProgressIdle
	case ModeZone, ModeRoof:
		wProg = p.WProgressZone
	case ModeRetreat:
		wProg = p.WProgressRetreat
	case ModeLoot, ModeHunt:
		wProg = p.WProgressLoot
	case ModeDrop:
		wProg = p.WProgressDrop
	case ModeFight:
		wProg = p.WProgressFight
	}
	zat := zoneAt(&obs.Zone, t, t+horizon, b.zoneB)
	zc, zr := zat.x0, zat.x1
	z := &obs.Zone
	nextSoon := z.ShrinkStart-t < 15.0

	for ci := range cands {
		c := &cands[ci]
		sim := planSimulate(world, me, c.dir, c.speed, slow, c.boost, n)
		c.pts = sim.x0
		c.blocked = sim.x1
		end := c.pts[n]
		s := 0.0
		switch b.goal.kind {
		case GoalNav, GoalField:
			if !reached {
				g := pos
				if goalPosOk {
					g = goalPos
				}
				startD := 0.0
				if d0Ok {
					startD = d0
				}
				var de float64
				if fld != nil {
					if x, ok := nav.dist(fld, end); ok && isFinite(x) {
						de = x
					} else {
						de = startD + 6.0
					}
				} else {
					de = dist2(end, g)
				}
				s += wProg * clampF(startD-fmax(de, arrive*0.5), -25.0, 25.0)
				if navdirOk {
					ndv := navdir
					s += p.WNavdir * (c.dir[0]*ndv[0] + c.dir[1]*ndv[1]) * fmin(c.speed/walk, 1.2)
				}
			} else {
				s -= 0.6 * dist2(end, pos)
			}
		case GoalRange:
			g, pref := b.goal.pos, b.goal.pref
			e0 := fmax(math.Abs(dist2(pos, g)-pref)-p.RangeBand, 0.0)
			e1 := fmax(math.Abs(dist2(end, g)-pref)-p.RangeBand, 0.0)
			s += p.WRange * (e0 - e1)
		case GoalNone:
			s -= 0.6 * dist2(end, pos)
		}
		if c.speed == 0.0 {
			s += p.WStand
			if t-b.mem.lastDamageT < 1.5 {
				s -= p.WJuke
			}
		}
		s -= p.WBlocked * c.blocked
		s += p.WSmooth * (c.dir[0]*b.lastDir[0] + c.dir[1]*b.lastDir[1])
		if jukeOk {
			j := juke
			s += wJuke * b.jukeSide * (c.dir[0]*j[0] + c.dir[1]*j[1]) * fmin(c.speed/walk, 1.3)
			if c.speed == 0.0 {
				s -= wJuke * 0.6
			}
		}
		if c.boost {
			s -= p.WBoost * st.BoostDrain * horizon
		}
		// zone
		out := dist2v(end, zc) - zr + p.ZoneInner
		if out > 0.0 {
			var k float64
			if b.mode == ModeZone || b.goal.kind == GoalField {
				k = p.ZoneCostZoneMode
			} else {
				k = p.ZoneCostMult
			}
			s -= k * z.Dps * (horizon*fmin(out/fmax(p.ZoneInner, 0.1), 1.0) + out/walk)
		}
		if nextSoon && b.mode != ModeZone {
			on := dist2v(end, z.NextCenter) - z.NextRadius
			o0 := dist2v(pos, z.NextCenter) - z.NextRadius
			if on > 0.0 {
				s -= p.WNextZone * (on - fmax(o0, 0.0))
			}
		}
		for _, q := range npcs {
			d := dist(q, end)
			if d < p.NpcAvoid {
				s -= p.WNpc*(p.NpcAvoid-d)*0.2 + p.WNpc*fmax(dist(q, pos)-d, 0.0)
			}
		}
		s -= p.WProj * b.incomingCost(c, shots, blasts, radius, height)
		if p.WAntilead > 0.0 {
			shooters := make([]tupVF, 0, len(threats))
			for k := range threats {
				th := &threats[k]
				if th.gun && !th.ghost && th.losNow {
					shooters = append(shooters, tupVF{th.eye, th.aimYaw})
				}
			}
			s -= p.WAntilead * antilead(c.pts, planDT, pos, me.Vel, shooters, radius+b.cfg.Projectile.GunRadius, fmax(b.cfg.Weapons.Gun.Speed, 1.0), p.AntileadLat, p.AntileadSigma, b.cfg.Weapons.Gun.Damage)
		}
		c.score = s
	}

	// exposure pass, best partial scores first
	if len(threats) > 0 {
		order := make([]int, len(cands))
		for i := range order {
			order[i] = i
		}
		sort.SliceStable(order, func(i, j int) bool { return totalCmp(cands[order[j]].score, cands[order[i]].score) < 0 })
		armed := fc.armed && fc.target != nil
		laserDmg := b.cfg.Weapons.Laser.Damage
		for k := range threats {
			threats[k].dist = fmax(threats[k].dist, 1.0)
		}
		for _, i := range order {
			cheap := budget.remainingMs() < p.ExpoMinBudgetMs || b.spentMs() > 12.0
			c := &cands[i]
			cost := 0.0
			bonus := 0.0
			for k := range threats {
				th := &threats[k]
				var wMode float64
				switch {
				case b.mode == ModeRetreat:
					wMode = p.ExpoRetreat
				case b.mode == ModeZone:
					wMode = p.ExpoZone
				case b.mode == ModeFight && th.isTarget:
					if armed {
						wMode = p.ExpoFightTarget
					} else {
						wMode = 1.0
					}
				default:
					wMode = 1.0
				}
				e := 0.0
				losEnd := th.losNow
				losMid := th.losNow
				for j, kk := range [2]int{n / 2, n} {
					cpt := up(c.pts[kk], height*0.5)
					var los bool
					if cheap {
						los = th.losNow
					} else {
						los = world.sight(th.eye, cpt)
					}
					if j == 0 {
						losMid = los
					} else {
						losEnd = los
					}
					if los {
						d := dist(th.eye, cpt)
						toUs := math.Atan2(cpt[1]-th.eye[1], cpt[0]-th.eye[0])
						al := clampF(1.0-math.Abs(adiff(th.aimYaw, toUs)), 0.0, 1.0)
						e += 0.5 * b.threatDps(th, d) * (0.4 + 0.6*al)
					}
				}
				cost += p.WExpo * wMode * e * horizon
				if th.isTarget && b.mode == ModeFight && armed && losEnd {
					bonus += p.WFightLos
				}
				if th.chargeLeft != nil {
					tl := *th.chargeLeft
					var hit bool
					if tl <= horizon*0.5+0.05 {
						hit = losMid
					} else {
						hit = losEnd
					}
					if hit {
						cost += p.WCharge * laserDmg
					}
				}
			}
			c.score += bonus - cost
		}
	}

	best := 0
	for i := 1; i < len(cands); i++ {
		if cands[i].score > cands[best].score {
			best = i
		}
	}
	c := &cands[best]
	fracSpeed := 0.0
	if c.speed > 0.0 {
		fracSpeed = 1.0
	}
	dir := V2{c.dir[0] * fracSpeed, c.dir[1] * fracSpeed}
	if fracSpeed > 0.0 {
		b.lastDir = c.dir
	} else {
		b.cmdSince = t + planDT
		b.lastDir = V2{b.lastDir[0] * 0.5, b.lastDir[1] * 0.5}
	}
	return MoveOut{dir: dir, boost: c.boost, jump: false}
}

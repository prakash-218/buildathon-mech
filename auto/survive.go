package auto

import (
	"math"
	"sort"
)

// Ported from src/brain/survive.rs.

type Foe struct {
	eye         V3
	w           float64
	laser       bool
	aimedCharge *float64
	target      bool
	d           float64
}

func (b *Brain) foes(obs *Obs, target *int64) []Foe {
	t := obs.T
	pos := obs.Me.Pos
	p := b.p
	var v []Foe
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech || t-tr.lastSeen > p.SvMemoryS {
			continue
		}
		age := fmin(t-tr.lastSeen, 1.5)
		tp := tr.posAt(tr.lastSeen + age)
		if !(isFinite(tp[0]) && isFinite(tp[1]) && isFinite(tp[2])) {
			continue
		}
		s := 1.0 + b.cfg.Cubes.SizePerCube*fmax(tr.cubes, 0.0)
		d := dist(tp, pos)
		w := 0.0
		gun := tr.has(WeaponKindGun) || len(tr.weapons) == 0
		if gun && d < b.cfg.Weapons.Gun.Range {
			w += p.GunDps * clampF(1.15-d/220.0, 0.2, 1.0)
		}
		laser := tr.has(WeaponKindLaser) && d < b.cfg.Weapons.Laser.Range
		if laser {
			w += p.LaserDps * p.SvLaserMult
		}
		if tr.has(WeaponKindLightning) && d < b.cfg.Weapons.Lightning.Range+10.0 {
			w += p.LightningDps
		}
		if tr.has(WeaponKindMortar) && d > 20.0 && d < 235.0 {
			w += p.MortarDps * 0.5
		}
		if t-tr.lastDmgT < 3.0 {
			w *= 1.3
		}
		eye := up(tp, b.cfg.Mech.EyeHeight*s)
		var aimedCharge *float64
		if ac, ok := surviveAimedLaser(tr, t, eye, pos, b.cfg.Weapons.Laser.Charge, p.FreshS); ok {
			aimedCharge = &ac
		}
		v = append(v, Foe{eye: eye, w: w, laser: laser, aimedCharge: aimedCharge, target: target != nil && *target == tr.id, d: d})
	}
	sort.SliceStable(v, func(i, j int) bool { return totalCmp(v[i].d, v[j].d) < 0 })
	v = v[:min(6, len(v))]
	return v
}

func (b *Brain) spotGoal(obs *Obs, world *World, nav *Nav, tgt *Track, pref float64, cur Goal) Goal {
	p := b.p
	t := obs.T
	me := &obs.Me
	pos := me.Pos
	if p.SvSpot <= 0.0 {
		return cur
	}
	var tgtID *int64
	if tgt != nil {
		id := tgt.id
		tgtID = &id
	}
	foes := b.foes(obs, tgtID)
	others := 0
	for i := range foes {
		if !foes[i].target && foes[i].w > 0.0 {
			others++
		}
	}
	laserEvade := false
	for i := range foes {
		if foes[i].aimedCharge != nil {
			laserEvade = true
			break
		}
	}
	var need bool
	switch b.mode {
	case ModeRetreat:
		need = false
	default:
		need = others >= 1 || p.SvSpot >= 2.0 || laserEvade
	}
	if !need {
		b.spot = nil
		return cur
	}
	if b.spot != nil {
		q, tc, stay := b.spot.x0, b.spot.x1, b.spot.x2
		if t-tc < p.SvPeriodS && !laserEvade {
			if stay {
				return cur
			}
			return Goal{kind: GoalNav, pos: q, arrive: 1.2}
		}
	}
	h := fmax(me.Stats.Height, 1.0)
	walk := fmax(me.Stats.WalkSpeed, 1.0)
	boost := fmax(me.Stats.BoostSpeed, walk)
	muzzleH := me.Stats.MuzzleHeight
	ziC := obs.Zone.Center
	ziR := obs.Zone.Radius
	cands := []tupVF{{pos, 0.0}}
	rings := [4]tupFN{{6.0, 8}, {12.0, 12}, {19.0, 12}, {27.0, 12}}
	for k0, rn := range rings {
		r, n := rn.x0, rn.x1
		for k := 0; k < n; k++ {
			a := float64(k)*2.0*math.Pi/float64(n) + float64(k0)*0.37
			x := pos[0] + r*math.Cos(a)
			y := pos[1] + r*math.Sin(a)
			q0 := V3{x, y, fin(world.floorBelow(x, y, pos[2]+3.0), pos[2])}
			q, ok := nav.snap(q0)
			if !ok {
				continue
			}
			if dist(q, q0) > 4.0 || !(isFinite(q[0]) && isFinite(q[1]) && isFinite(q[2])) || math.Abs(q[2]-pos[2]) > 6.0 {
				continue
			}
			if ziR > 0.0 && dist2v(q, ziC) > ziR-10.0 {
				continue
			}
			cands = append(cands, tupVF{q, dist2(q, pos)})
		}
	}
	if b.spot != nil && !b.spot.x2 {
		q := b.spot.x0
		if dist2(q, pos) < 30.0 {
			cands = append(cands, tupVF{q, dist2(q, pos)})
		}
	}
	chargeLeft := math.Inf(1)
	for i := range foes {
		if foes[i].aimedCharge != nil {
			chargeLeft = fmin(chargeLeft, *foes[i].aimedCharge)
		}
	}
	scored := make([]tupFVBF, 0, len(cands))
	for i, cd := range cands {
		q, dq := cd.x0, cd.x1
		if b.spentMs() > p.SvBudgetMs {
			break
		}
		c := up(q, h*0.5)
		mid := up(V3{(q[0] + pos[0]) * 0.5, (q[1] + pos[1]) * 0.5, (q[2] + pos[2]) * 0.5}, h*0.5)
		expo := 0.0
		path := 0.0
		hiddenAll := true
		laserHit := false
		for fi := range foes {
			f := &foes[fi]
			_, blocked := world.ray(f.eye, c)
			shot := !blocked
			if shot {
				hiddenAll = false
				wt := 1.0
				if f.target {
					wt = p.SvTargetExpo
				}
				expo += f.w * wt
				if f.laser {
					laserHit = true
				}
				if f.aimedCharge != nil {
					laserHit = true
				}
			}
			if i > 0 && dq > 3.0 {
				if _, blk := world.ray(f.eye, mid); !blk {
					wt := 1.0
					if f.target {
						wt = p.SvTargetExpo
					}
					path += f.w * wt
				}
			}
		}
		s := -p.SvWCov*expo/100.0 - p.SvWPath*path/100.0*fmin(dq/walk, 3.0)/3.0
		if hiddenAll {
			s += p.SvWHidden
		}
		if laserHit {
			s -= p.SvWLaser
		}
		if tgt != nil {
			tc := up(tgt.posAt(t), b.cfg.Mech.Height*0.5)
			d := dist(q, tc)
			if _, blk := world.ray(up(q, muzzleH), tc); !blk {
				s += p.SvWShot - p.SvWRange*fmax(math.Abs(d-pref)-p.RangeBand, 0.0)
			}
		}
		if isFinite(chargeLeft) && !laserHit {
			if dq/boost <= chargeLeft+0.1 {
				s += p.SvWEvade
			}
		}
		s -= p.SvWTravel * dq
		if b.spot != nil {
			if dist2(b.spot.x0, q) < 3.0 {
				s += p.SvHyst
			}
		}
		scored = append(scored, tupFVBF{s, q, hiddenAll, dq})
	}
	if len(scored) == 0 {
		return cur
	}
	n := len(scored)
	for i := 0; i < n; i++ {
		q := scored[i].x1
		near := false
		for j := 0; j < n; j++ {
			if j != i && scored[j].x2 && dist2(scored[j].x1, q) < 6.5 {
				near = true
				break
			}
		}
		if near || scored[i].x2 {
			scored[i].x0 += p.SvWCover
		}
	}
	best := 0
	for i := 1; i < n; i++ {
		if scored[i].x0 > scored[best].x0 {
			best = i
		}
	}
	bs, q, dq := scored[best].x0, scored[best].x1, scored[best].x3
	stay := best == 0 || dq < 2.5 || bs < scored[0].x0+p.SvMinGain
	b.spot = &tupVFB{q, t, stay}
	if stay {
		return cur
	}
	return Goal{kind: GoalNav, pos: q, arrive: 1.2}
}

func (b *Brain) threatAim(obs *Obs) (V2, bool) {
	if b.p.SvTorso <= 0.0 {
		return V2{}, false
	}
	pos := obs.Me.Pos
	foes := b.foes(obs, nil)
	hasBest := false
	var best tupFV
	for i := range foes {
		f := &foes[i]
		if f.d > 220.0 || f.w <= 0.0 {
			continue
		}
		s := f.w / (1.0 + f.d/80.0)
		if !hasBest || s > best.x0 {
			best = tupFV{s, f.eye}
			hasBest = true
		}
	}
	if !hasBest {
		return V2{}, false
	}
	e := best.x1
	return V2{math.Atan2(e[1]-pos[1], e[0]-pos[0]), 0.0}, true
}

func (b *Brain) holdFire(obs *Obs, d float64, hp float64) bool {
	p := b.p
	if p.SvHold <= 0.0 || b.mode != ModeRetreat {
		return false
	}
	lethal := hp > 0.0 && hp < p.SvHoldLethal
	hitRecent := obs.T-b.mem.lastDamageT < p.SvHoldHitS
	return d > p.SvHoldRange && !lethal && !hitRecent
}

func surviveAimedLaser(tr *Track, t float64, eye V3, pos V3, charge float64, fresh float64) (float64, bool) {
	if !tr.charging || t-tr.lastSeen > fmax(fresh, 0.3) || !tr.has(WeaponKindLaser) {
		return 0, false
	}
	d := fmax(dist2(eye, pos), 1.0)
	toUs := math.Atan2(pos[1]-eye[1], pos[0]-eye[0])
	if math.Abs(adiff(tr.aim[0], toUs)) > math.Atan2(4.5, d)+0.08 {
		return 0, false
	}
	return clampF(charge-(t-tr.chargeSince), 0.1, 0.8), true
}

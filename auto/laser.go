package auto

import (
	"fmt"
	"math"
	"os"
	"sort"
)

// Ported from src/brain/laser.rs.

type PkSt int

const (
	PkStOff PkSt = iota
	PkStHide
	PkStOut
	PkStBack
)

// String mirrors Rust's derived Debug for the debug prints.
func (s PkSt) String() string {
	switch s {
	case PkStOff:
		return "Off"
	case PkStHide:
		return "Hide"
	case PkStOut:
		return "Out"
	case PkStBack:
		return "Back"
	}
	return fmt.Sprintf("PkSt(%d)", int(s))
}

type Peek struct {
	st     PkSt
	tid    int64
	hide   V3
	peek   V3
	planT  float64
	stT    float64
	failT  float64
	shots  uint32
	dbg    bool
	lastSt PkSt
	ebeamT float64
	dodge  *tupVF
}

func newPeek() Peek {
	_, dbg := os.LookupEnv("LZ_DEBUG")
	return Peek{st: PkStOff, tid: -1, hide: V3{}, peek: V3{}, planT: -1e9, stT: 0.0, failT: -1e9, shots: 0, dbg: dbg, lastSt: PkStOff, ebeamT: -1e9, dodge: nil}
}

type laserNode struct {
	q      V3
	cov    bool
	clr    bool
	seenBy uint32
}

func (b *Brain) laserSlot(obs *Obs) (int, bool) {
	for i, w := range obs.Me.Weapons {
		if w != nil && w.Kind == WeaponKindLaser {
			return i, true
		}
	}
	return 0, false
}

func (b *Brain) laserPre(obs *Obs, world *World, nav *Nav) {
	p := b.p
	t := obs.T
	pos := obs.Me.Pos
	if p.LzSeek > 0.0 {
		if _, has := b.laserSlot(obs); !has && b.mode != ModeZone && (b.mode != ModeRetreat || p.LzSeekRetreat > 0.5) {
			if _, ok := b.want(&obs.Me, WeaponKindLaser); ok {
				z := &obs.Zone
				hot := t-b.mem.lastDamageT < p.LzHotS
				if !hot {
					for i := range b.mem.tracks {
						tr := &b.mem.tracks[i]
						if tr.kind == ActorKindMech && t-tr.lastSeen < p.LzHotS && dist(tr.posAt(t), pos) < p.LzHotD {
							hot = true
							break
						}
					}
				}
				hasBest := false
				var best tupFIV
				for i := range b.mem.pickups {
					it := &b.mem.pickups[i]
					if it.kind != WeaponKindLaser || b.mem.blacklisted(it.id, t) {
						continue
					}
					d := dist(it.pos, pos)
					lim := p.LzSeek
					if hot {
						lim = p.LzSeekFight
					}
					if b.sfFarBlock(d) {
						continue
					}
					if d > lim || dist2v(it.pos, z.Center) > z.Radius-8.0 {
						continue
					}
					if !hasBest || d < best.x0 {
						best = tupFIV{d, it.id, it.pos}
						hasBest = true
					}
				}
				if hasBest {
					id, q := best.x1, best.x2
					b.mode = ModeLoot
					b.goal = Goal{kind: GoalNav, pos: q, arrive: 1.2}
					if b.goalID != id {
						b.pursuit = tupIF{id, t}
					}
					b.goalID = id
				}
			}
		}
	}
	if p.LzPeek <= 0.0 {
		b.pk.st = PkStOff
		return
	}
	if _, has := b.laserSlot(obs); !has {
		b.pk.st = PkStOff
		return
	}
	tid := int64(-1)
	if b.target.kind == TgtMech {
		tid = b.target.id
	}
	if tid < 0 && p.LzAnyTgt > 0.5 {
		hasBest := false
		var best tupFI
		for i := range b.mem.tracks {
			tr := &b.mem.tracks[i]
			if tr.kind != ActorKindMech || t-tr.lastSeen > p.LzMemS {
				continue
			}
			da := 0.0
			if t-tr.lastDmgT < 3.0 {
				da = 100.0
			}
			dl := 0.0
			if tr.has(WeaponKindLaser) {
				dl = 50.0
			}
			s := dist(tr.posAt(t), pos) - da - dl
			if !hasBest || s < best.x0 {
				best = tupFI{s, tr.id}
				hasBest = true
			}
		}
		if hasBest {
			tid = best.x1
		}
	}
	var tr *Track
	for i := range b.mem.tracks {
		if b.mem.tracks[i].id == tid && b.mem.tracks[i].kind == ActorKindMech {
			c := b.mem.tracks[i]
			tr = &c
			break
		}
	}
	ok := b.mode == ModeFight ||
		(b.mode == ModeRetreat && p.LzRetreat > 0.5) ||
		(p.LzAnyTgt > 0.5 && (b.mode == ModeLoot || b.mode == ModeHunt || b.mode == ModeIdle || b.mode == ModeDrop)) ||
		(b.pk.st != PkStOff && b.mode != ModeZone)
	if tr == nil || !(ok && t-tr.lastSeen < p.LzMemS) {
		if b.pk.dbg && f2i(math.Round(t*10.0))%20 == 0 {
			fmt.Fprintf(os.Stderr, "LZ t=%.2f no tgt mode=%v target=%v ok=%v\n", t, b.mode, b.target, ok)
		}
		b.pk.st = PkStOff
		return
	}
	tp := tr.posAt(t)
	dt := dist(tp, pos)
	if dt < p.LzMinD || dt > p.LzMaxD || obs.Me.Hp < obs.Me.MaxHp*p.LzMinHp {
		if b.pk.dbg {
			fmt.Fprintf(os.Stderr, "LZR range %.0f\n", dt)
		}
		b.pk.st = PkStOff
		return
	}
	if tr.id != b.pk.tid {
		b.pk.tid = tr.id
		b.pk.planT = -1e9
	}
	replan := t-b.pk.planT >= p.LzReplanS && b.pk.st != PkStOut
	if replan && t-b.pk.failT > 0.5 {
		b.pk.planT = t
		if hq, ok := b.peekPlan(obs, world, nav, tp, tr.id); ok {
			b.pk.hide = hq.x0
			b.pk.peek = hq.x1
			if b.pk.st == PkStOff {
				b.pk.st = PkStHide
				b.pk.stT = t
			}
		} else {
			if b.pk.dbg {
				fmt.Fprintf(os.Stderr, "LZ t=%.2f plan fail d=%.0f\n", t, dt)
			}
			b.pk.failT = t
			b.pk.st = PkStOff
		}
	}
	if b.pk.st != PkStOff {
		b.goal = Goal{kind: GoalNav, pos: b.pk.hide, arrive: 0.8}
	}
}

func (b *Brain) peekPlan(obs *Obs, world *World, nav *Nav, tp V3, tid int64) (tupVV, bool) {
	p := b.p
	pos := obs.Me.Pos
	t := obs.T
	h := b.cfg.Mech.Height
	te := up(tp, b.cfg.Mech.EyeHeight)
	tm := up(tp, b.cfg.Mech.MuzzleHeight)
	tc := up(tp, h*0.5)
	var others []V3
	for i := range b.mem.tracks {
		o := &b.mem.tracks[i]
		if !(o.kind == ActorKindMech && o.id != tid && t-o.lastSeen < 4.0) {
			continue
		}
		e := up(o.posAt(t), b.cfg.Mech.EyeHeight)
		if dist(e, pos) < 260.0 {
			others = append(others, e)
		}
	}
	cnx, cny, o := nav.grid()
	r := fmax(p.LzR, 4.0)
	i0 := f2n(fmax(math.Floor((pos[0]-r-o[0])/cell), 0.0))
	j0 := f2n(fmax(math.Floor((pos[1]-r-o[1])/cell), 0.0))
	i1 := min(f2n(fmax(math.Ceil((pos[0]+r-o[0])/cell), 0.0)), max(cnx-1, 0))
	j1 := min(f2n(fmax(math.Ceil((pos[1]+r-o[1])/cell), 0.0)), max(cny-1, 0))
	var nodes []laserNode
	step := f2n(fmax(p.LzCell, 1.0))
	cj := j0
	for cj <= j1 {
		ci := i0
		for ci <= i1 {
			lo, hi := nav.cellNodeIds(ci, cj)
			for nd := lo; nd < hi; nd++ {
				q := nav.nodePos(nd)
				if math.Abs(q[2]-pos[2]) > 3.0 || dist(q, pos) > r {
					continue
				}
				cov := aimRayHit(world, te, up(q, h*0.5)) && aimRayHit(world, tm, up(q, h*0.95)) && aimRayHit(world, te, up(q, h*0.95))
				if cov && p.LzMargin > 0.0 {
					dx := q[0] - tp[0]
					dy := q[1] - tp[1]
					l := fmax(math.Hypot(dx, dy), 1e-6)
					nx, ny := -dy/l*p.LzMargin, dx/l*p.LzMargin
					for _, sgn := range [2]float64{-1.0, 1.0} {
						e2 := V3{te[0] + nx*sgn, te[1] + ny*sgn, te[2]}
						if !aimRayHit(world, e2, up(q, h*0.6)) {
							cov = false
						}
					}
				}
				clr := !cov && !aimRayHit(world, up(q, b.cfg.Mech.MuzzleHeight), tc)
				seenBy := uint32(0)
				if cov || clr {
					for _, e := range others {
						if !aimRayHit(world, e, up(q, h*0.6)) {
							seenBy++
						}
					}
				}
				nodes = append(nodes, laserNode{q: q, cov: cov, clr: clr, seenBy: seenBy})
			}
			if len(nodes) > 900 || b.spentMs() > p.LzMs {
				break
			}
			ci += step
		}
		cj += step
	}
	var cands []tupFNN
	smax := p.LzStep
	for hi := range nodes {
		hn := &nodes[hi]
		if !hn.cov {
			continue
		}
		dh := dist(hn.q, pos)
		keep := 0.0
		if b.pk.st != PkStOff && dist(hn.q, b.pk.hide) < 1.5 {
			keep = p.LzKeep
		}
		far := 0.0
		if dist(hn.q, tp) < p.LzMinD {
			far = 40.0
		}
		for pi := range nodes {
			pn := &nodes[pi]
			if !pn.clr {
				continue
			}
			s := dist(hn.q, pn.q)
			if s > smax || s < 0.8 {
				continue
			}
			sc := dh*p.LzWMove + s*p.LzWStep + float64(pn.seenBy)*p.LzWSeen + float64(hn.seenBy)*p.LzWSeen*1.5 + far - keep
			cands = append(cands, tupFNN{sc, hi, pi})
		}
	}
	if b.pk.dbg {
		nCov, nClr := 0, 0
		for i := range nodes {
			if nodes[i].cov {
				nCov++
			}
			if nodes[i].clr {
				nClr++
			}
		}
		fmt.Fprintf(os.Stderr, "LZP nodes=%d cov=%d clr=%d cands=%d ms=%.1f\n", len(nodes), nCov, nClr, len(cands), b.spentMs())
	}
	// partial_cmp(..).unwrap_or(Equal), stable
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].x0 < cands[j].x0 })
	for k := 0; k < len(cands) && k < 12; k++ {
		hq := nodes[cands[k].x1].q
		pq := nodes[cands[k].x2].q
		if (dist(hq, pos) < 1.0 || nav.clear(pos, hq)) && nav.clear(hq, pq) {
			return tupVV{hq, pq}, true
		}
		if b.spentMs() > p.LzMs+3.0 {
			break
		}
	}
	if p.LzHideOnly > 0.5 {
		hasBest := false
		var best tupFV
		for i := range nodes {
			n := &nodes[i]
			if !(n.cov && n.seenBy == 0) {
				continue
			}
			keep := 0.0
			if b.pk.st != PkStOff && dist(n.q, b.pk.hide) < 1.5 {
				keep = p.LzKeep
			}
			s := dist(n.q, pos) - keep
			if !hasBest || s < best.x0 {
				best = tupFV{s, n.q}
				hasBest = true
			}
		}
		if hasBest {
			q := best.x1
			if dist(q, pos) < 1.0 || nav.clear(pos, q) {
				return tupVV{q, q}, true
			}
		}
	}
	return tupVV{}, false
}

func (b *Brain) laserPost(obs *Obs, world *World, fc *FireOut, mv *MoveOut) {
	p := b.p
	if p.LzScanS > 0.0 && fc.target == nil && !fc.charging {
		if _, has := b.laserSlot(obs); !has {
			t := obs.T
			quiet := t-b.mem.lastDamageT > 2.0
			per := p.LzScanEvery
			phase := 1e9
			if per > 0.0 {
				phase = math.Mod(t, per)
			}
			if quiet && !b.sfQuietBlock() && (t < p.LzScanS || phase < p.LzScanLen) {
				fc.aim = V2{wrap(obs.Me.Aim[0] + 1.5), 0.0}
			}
		}
	}
	for _, bm := range obs.Beams {
		if bm.Kind == BeamKindLaser && bm.By == b.pk.tid {
			b.pk.ebeamT = obs.T - fmax(bm.Age, 0.0)
		}
	}
	for _, e := range obs.Events {
		if e.Damaged != nil {
			d := e.Damaged
			if d.Weapon == "laser" && d.By != nil && *d.By == b.pk.tid {
				b.pk.ebeamT = obs.T
			}
			if b.pk.dbg {
				fmt.Fprintf(os.Stderr, "LZD %s %v %.0f mode=%v\n", d.Weapon, b.pk.st, d.Amount, b.mode)
			}
		}
	}
	if p.LzCl > 0.5 && b.pk.st == PkStOff {
		b.counterLaser(obs, world, mv)
	}
	if b.pk.dbg && b.pk.st != b.pk.lastSt {
		fmt.Fprintf(os.Stderr, "LZ t=%.2f %v->%v pos=%.0f hide=%.0f peek=%.0f hp=%.0f\n", obs.T, b.pk.lastSt, b.pk.st, obs.Me.Pos, b.pk.hide, b.pk.peek, obs.Me.Hp)
	}
	b.pk.lastSt = b.pk.st
	if b.pk.st == PkStOff {
		return
	}
	t := obs.T
	me := &obs.Me
	pos := me.Pos
	ls, has := b.laserSlot(obs)
	if !has {
		b.pk.st = PkStOff
		return
	}
	wp := me.Weapons[ls]
	if wp == nil {
		return
	}
	w := *wp
	var tr Track
	found := false
	for i := range b.mem.tracks {
		if b.mem.tracks[i].id == b.pk.tid {
			tr = b.mem.tracks[i]
			found = true
			break
		}
	}
	if !found {
		b.pk.st = PkStOff
		return
	}
	tp := tr.posAt(t)
	at := b.aimTarget(&tr)
	c := up(tp, at.height*0.5)
	vpos := add(pos, scale(V3{me.Vel[0], me.Vel[1], 0.0}, 0.05))
	muzzle := up(vpos, me.Stats.MuzzleHeight)
	line := !aimRayHit(world, muzzle, c) && !aimRayHit(world, up(pos, me.Stats.MuzzleHeight), c)
	direct := dirToAim(sub(c, up(pos, me.Stats.MuzzleHeight)))
	full := fmax(w.Stats.Charge, 0.05)
	charging := w.Charge > 0.0
	pred := slew(me.Aim, direct, me.Stats.TurnRate*0.05, me.Stats.PitchRate*0.05)
	d := fmax(vlen(sub(c, muzzle)), 1.0)
	tol := math.Atan2(at.radius*0.8, d)
	aligned := math.Abs(adiff(pred[0], direct[0])) <= tol && math.Abs(pred[1]-direct[1]) <= math.Atan2(at.height*0.4, d)
	to := func(a V3, bb V3) (V2, float64) {
		dx := bb[0] - a[0]
		dy := bb[1] - a[1]
		l := math.Hypot(dx, dy)
		if l < 1e-6 {
			return V2{0.0, 0.0}, 0.0
		}
		return V2{dx / l, dy / l}, l
	}
	gunFire := fc.fire
	switch b.pk.st {
	case PkStHide:
		dir, dh := to(pos, b.pk.hide)
		if dh > 0.7 {
			mv.dir = V2{dir[0] * fmin(dh/2.0, 1.0), dir[1] * fmin(dh/2.0, 1.0)}
			mv.boost = dh > 6.0 && me.Energy > 30.0 && !charging
		} else {
			mv.dir = V2{0.0, 0.0}
			mv.boost = false
		}
		mv.jump = false
		fc.aim = direct
		te := up(tp, b.cfg.Mech.EyeHeight)
		exposed := !aimRayHit(world, te, up(pos, fmax(at.height, 1.0)*0.5)) || !aimRayHit(world, te, up(pos, b.cfg.Mech.Height*0.95))
		if exposed && p.LzExpRush > 0.5 && dh > 0.7 {
			mv.dir = dir
			mv.boost = me.Energy > 10.0 && !charging
		}
		near := dh < p.LzChargeD && !(exposed && p.LzExpRush > 0.5)
		if charging {
			if line && aligned && w.Charge >= full-0.06 {
				fc.fire[ls] = false
				b.pk.shots++
				b.pk.st = PkStBack
				b.pk.stT = t
			} else {
				fc.fire[ls] = true
				since := t - b.pk.ebeamT
				heldS := fmax(w.Charge-full, 0.0)
				wait := p.LzSync > 0.0 && tr.has(WeaponKindLaser) && since > 1.3 && heldS < p.LzSync
				if w.Charge >= full-0.12 && dh < 1.5 && !wait && dist(b.pk.peek, b.pk.hide) > 0.5 {
					b.pk.st = PkStOut
					b.pk.stT = t
				}
			}
		} else {
			fc.fire[ls] = near && w.Cooldown <= 0.05
		}
	case PkStOut:
		dir, dp := to(pos, b.pk.peek)
		if dp > 0.3 {
			mv.dir = dir
		} else {
			mv.dir = V2{0.0, 0.0}
		}
		mv.boost = false
		mv.jump = false
		fc.aim = direct
		if !charging {
			b.pk.st = PkStBack
			b.pk.stT = t
		} else if line && aligned {
			fc.fire[ls] = false
			b.pk.shots++
			b.pk.st = PkStBack
			b.pk.stT = t
		} else {
			fc.fire[ls] = true
			if t-b.pk.stT > p.LzOutMax {
				b.pk.st = PkStHide
				b.pk.stT = t
				b.pk.planT = -1e9
			}
		}
		if line && p.LzGun > 0.5 {
			for i, wo := range me.Weapons {
				if i != ls && wo != nil && wo.Kind == WeaponKindGun {
					fc.fire[i] = gunFire[i] || (wo.Ammo > 0.0 && wo.Reload <= 0.0 && aligned)
				}
			}
		}
	case PkStBack:
		dir, dh := to(pos, b.pk.hide)
		if dh > 0.4 {
			mv.dir = dir
		} else {
			mv.dir = V2{0.0, 0.0}
		}
		mv.boost = dh > 1.0 && me.Energy > 15.0 && p.LzBoost > 0.5
		mv.jump = false
		fc.fire[ls] = false
		if dh < 0.9 || t-b.pk.stT > 1.5 {
			b.pk.st = PkStHide
			b.pk.stT = t
			b.pk.planT = -1e9
		}
	case PkStOff:
	}
	fc.charging = fc.fire[ls]
}

func (b *Brain) counterLaser(obs *Obs, world *World, mv *MoveOut) {
	p := b.p
	t := obs.T
	me := &obs.Me
	pos := me.Pos
	if b.pk.dodge != nil {
		q, until := b.pk.dodge.x0, b.pk.dodge.x1
		if t > until {
			b.pk.dodge = nil
		} else {
			dx := q[0] - pos[0]
			dy := q[1] - pos[1]
			l := math.Hypot(dx, dy)
			if l > 0.5 {
				mv.dir = V2{dx / l, dy / l}
				mv.boost = me.Energy > 8.0
				mv.jump = false
			}
			return
		}
	}
	full := fmax(b.cfg.Weapons.Laser.Charge, 0.1)
	h := b.cfg.Mech.Height
	hasThr := false
	var thr tupVF
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech || !tr.charging || t-tr.lastSeen > 0.15 || !tr.has(WeaponKindLaser) {
			continue
		}
		tp := tr.posAt(t)
		d := dist(tp, pos)
		if d < 15.0 {
			continue
		}
		brg := math.Atan2(pos[1]-tp[1], pos[0]-tp[0])
		if math.Abs(adiff(tr.aim[0], brg)) > p.LzClCone {
			continue
		}
		left := full - (t - tr.chargeSince)
		if left < 0.05 || left > p.LzClLeft {
			continue
		}
		if !hasThr || left < thr.x1 {
			thr = tupVF{tp, left}
			hasThr = true
		}
	}
	if b.pk.dbg {
		n := 0
		for i := range b.mem.tracks {
			tr := &b.mem.tracks[i]
			if tr.kind == ActorKindMech && tr.charging && t-tr.lastSeen < 0.15 {
				n++
			}
		}
		if n > 0 {
			if hasThr {
				fmt.Fprintf(os.Stderr, "LZC charging_seen=%d thr=Some(%v)\n", n, thr.x1)
			} else {
				fmt.Fprintf(os.Stderr, "LZC charging_seen=%d thr=None\n", n)
			}
		}
	}
	if !hasThr {
		return
	}
	tp, left := thr.x0, thr.x1
	te := up(tp, b.cfg.Mech.EyeHeight)
	if aimRayHit(world, te, up(pos, h*0.5)) && aimRayHit(world, te, up(pos, h*0.95)) {
		return
	}
	shared := b.shared
	nav := shared.nav
	speed := fmax(fmax(me.Stats.BoostSpeed, me.Stats.WalkSpeed), 1.0)
	r := fmin(left*speed+1.0, p.LzClR)
	cnx, cny, o := nav.grid()
	i0 := f2n(fmax(math.Floor((pos[0]-r-o[0])/cell), 0.0))
	j0 := f2n(fmax(math.Floor((pos[1]-r-o[1])/cell), 0.0))
	i1 := min(f2n(fmax(math.Ceil((pos[0]+r-o[0])/cell), 0.0)), max(cnx-1, 0))
	j1 := min(f2n(fmax(math.Ceil((pos[1]+r-o[1])/cell), 0.0)), max(cny-1, 0))
	hasBest := false
	var best tupFV
	for cj := j0; cj <= j1; cj++ {
		for ci := i0; ci <= i1; ci++ {
			lo, hi := nav.cellNodeIds(ci, cj)
			for nd := lo; nd < hi; nd++ {
				q := nav.nodePos(nd)
				d := dist(q, pos)
				if d > r || math.Abs(q[2]-pos[2]) > 2.5 || (hasBest && d >= best.x0) {
					continue
				}
				if aimRayHit(world, te, up(q, h*0.5)) && aimRayHit(world, te, up(q, h*0.95)) && nav.clear(pos, q) {
					best = tupFV{d, q}
					hasBest = true
				}
			}
		}
		if b.spentMs() > p.LzMs+4.0 {
			break
		}
	}
	if b.pk.dbg {
		fmt.Fprintf(os.Stderr, "LZC2 found=%v\n", hasBest)
	}
	if hasBest {
		q := best.x1
		b.pk.dodge = &tupVF{q, t + left + 0.3}
		dx := q[0] - pos[0]
		dy := q[1] - pos[1]
		l := fmax(math.Hypot(dx, dy), 1e-6)
		mv.dir = V2{dx / l, dy / l}
		mv.boost = me.Energy > 8.0
		mv.jump = false
	}
}

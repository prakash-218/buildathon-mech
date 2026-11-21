package auto

import (
	"math"
	"sort"
)

// Ported from src/brain/roof.rs.

type Region struct {
	expo    float64
	keep    V3
	setback float64
	height  float64
	top     float64
}

type RoofMap struct {
	regions    []Region
	nodeRegion map[uint32]uint32
}

// roofMapBuild is Rust `RoofMap::build`.
func roofMapBuild(world *World, nav *Nav, minNodes int, minSetback float64, height float64) *RoofMap {
	n := nav.nodeCount()
	comp := make([]uint32, n)
	for i := range comp {
		comp[i] = math.MaxUint32
	}
	var regions []Region
	nodeRegion := make(map[uint32]uint32)
	nextID := uint32(0)
	for s := uint32(0); s < uint32(n); s++ {
		if !nav.nodeRoof(s) || comp[s] != math.MaxUint32 {
			continue
		}
		id := nextID
		nextID++
		comp[s] = id
		stack := []uint32{s}
		var list []uint32
		for len(stack) > 0 {
			a := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			list = append(list, a)
			za := nav.nodePos(a)[2]
			for _, e := range nav.edges(a) {
				bn := e.to
				if e.kind != edgeWalk || int(bn) >= n || comp[bn] != math.MaxUint32 || !nav.nodeRoof(bn) {
					continue
				}
				if math.Abs(nav.nodePos(bn)[2]-za) > 1.3 {
					continue
				}
				comp[bn] = id
				stack = append(stack, bn)
			}
		}
		if len(list) < minNodes {
			continue
		}
		cnt := make([]int, len(list))
		for k, a := range list {
			c := 0
			for _, e := range nav.edges(a) {
				if e.kind == edgeWalk && int(e.to) < n && comp[e.to] == id {
					c++
				}
			}
			cnt[k] = c
		}
		maxc := 0
		for _, c := range cnt {
			maxc = max(maxc, c)
		}
		var bnd []V3
		for k, a := range list {
			if cnt[k] < maxc {
				bnd = append(bnd, nav.nodePos(a))
			}
		}
		if len(bnd) == 0 {
			continue
		}
		sb := make([]tupFU, 0, len(list))
		for _, a := range list {
			q := nav.nodePos(a)
			m := math.Inf(1)
			for _, bp := range bnd {
				m = fmin(m, math.Hypot(q[0]-bp[0], q[1]-bp[1]))
			}
			sb = append(sb, tupFU{m, a})
		}
		// sort_by(|x, y| y.0.total_cmp(&x.0)): descending, stable
		sort.SliceStable(sb, func(i, j int) bool { return totalCmp(sb[j].x0, sb[i].x0) < 0 })
		topSb := sb[0].x0
		if topSb < minSetback {
			continue
		}
		cut := fmax(topSb*0.5, minSetback)
		var cands []tupFU
		for _, x := range sb {
			if x.x0 >= cut {
				cands = append(cands, x)
			}
		}
		stride := max(len(cands)/24, 1)
		hasBest := false
		var best tupFFU
		for k, x := range cands {
			if k%stride != 0 {
				continue
			}
			m, a := x.x0, x.x1
			e := exposure(world, nav.nodePos(a), height)
			s := e - 0.02*m
			if !hasBest || s < best.x0 {
				best = tupFFU{s, m, a}
				hasBest = true
			}
		}
		if !hasBest {
			continue
		}
		bm, ba := best.x1, best.x2
		keep := nav.nodePos(ba)
		ground := fin(world.terrainZ(keep[0], keep[1]), keep[2])
		ri := uint32(len(regions))
		for _, a := range list {
			nodeRegion[a] = ri
		}
		expo := exposure(world, keep, height)
		regions = append(regions, Region{expo: expo, keep: keep, setback: bm, height: keep[2] - ground, top: keep[2]})
	}
	return &RoofMap{regions: regions, nodeRegion: nodeRegion}
}

func (r *RoofMap) regionOf(nav *Nav, p V3) (int, bool) {
	n, ok := nav.snapNode(p)
	if !ok {
		return 0, false
	}
	ri, ok := r.nodeRegion[n]
	if !ok {
		return 0, false
	}
	if p[2] > r.regions[ri].top-2.0 {
		return int(ri), true
	}
	return 0, false
}

// exposure: weighted count of ground and roof observers within 170 m that see the top of a mech standing at p.
func exposure(world *World, p V3, height float64) float64 {
	top := up(p, height*0.95)
	mid := up(p, height*0.5)
	e := 0.0
	for _, rw := range [4]tupFF{{50.0, 1.0}, {90.0, 0.8}, {130.0, 0.6}, {170.0, 0.4}} {
		r, w := rw.x0, rw.x1
		for k := 0; k < 32; k++ {
			a := float64(k)*math.Pi/16.0 + r*0.01
			x := p[0] + r*math.Cos(a)
			y := p[1] + r*math.Sin(a)
			fz := world.floorBelow(x, y, 1.0e3)
			if !isFinite(fz) {
				continue
			}
			eye := V3{x, y, fz + 8.0}
			if !aimRayHit(world, eye, mid) {
				e += 2.0 * w
			} else if !aimRayHit(world, eye, top) {
				e += w
			}
		}
	}
	return e
}

type roofFoe struct {
	id      int64
	pos     V3
	hpf     float64
	climber bool
	line    bool
}

func (b *Brain) roofChoose(obs *Obs, zi *ZoneInfo, rm *RoofMap) (int, bool) {
	p := b.p
	z := &obs.Zone
	pos := obs.Me.Pos
	hasBest := false
	var best tupFN
	for i := range rm.regions {
		r := &rm.regions[i]
		if r.setback < p.RoofMinSetback || r.height < p.RoofMinH || r.height > p.RoofMaxH {
			continue
		}
		banned := false
		for _, bn := range b.roofBan {
			if bn.x0 == i && obs.T < bn.x1 {
				banned = true
				break
			}
		}
		if banned {
			continue
		}
		dn := dist2v(r.keep, z.NextCenter)
		inner := z.NextRadius - dn - p.RoofZoneMargin
		if inner < 0.0 || dist2v(r.keep, zi.curC) > zi.curR-p.RoofZoneMargin {
			continue
		}
		d := dist2(r.keep, pos)
		s := 2.0*fmin(r.setback, 45.0) - p.RoofWExpo*r.expo - p.RoofWDist*d + 0.2*fmin(inner, 120.0) - 0.4*math.Abs(r.height-20.0)
		if b.roofSel != nil && *b.roofSel == i {
			s += 25.0
		}
		if !hasBest || s > best.x0 {
			best = tupFN{s, i}
			hasBest = true
		}
	}
	if hasBest {
		return best.x1, true
	}
	return 0, false
}

func (b *Brain) roofStepPoint(world *World, nav *Nav, rm *RoofMap, ri int, tid int64, tp V3, t float64, muzzleH float64) (V3, bool) {
	if rs := b.roofStep; rs != nil {
		if rs.x0 == tid && t-rs.x2 < 0.6 {
			if rs.x1 != nil {
				return *rs.x1, true
			}
			return V3{}, false
		}
	}
	r := &rm.regions[ri]
	keep := r.keep
	tc := up(tp, b.cfg.Mech.Height*0.5)
	b0 := math.Atan2(tp[1]-keep[1], tp[0]-keep[0])
	hasBest := false
	var best tupFV
	maxd := r.setback + 12.0
	rays := 0
	for _, db := range [5]float64{0.0, 0.45, -0.45, 0.9, -0.9} {
		bb := b0 + db
		k := 0.0
		for k <= maxd && rays < 90 {
			q0 := V3{keep[0] + math.Cos(bb)*k, keep[1] + math.Sin(bb)*k, keep[2] + 1.0}
			k += 2.5
			n, ok := nav.snapNode(q0)
			if !ok {
				continue
			}
			if x, ok := rm.nodeRegion[n]; !ok || int(x) != ri {
				continue
			}
			q := nav.nodePos(n)
			if hasBest && dist2(q, keep) >= best.x0 {
				break
			}
			rays++
			if !aimRayHit(world, up(q, muzzleH), tc) {
				best = tupFV{dist2(q, keep), q}
				hasBest = true
				break
			}
		}
	}
	var q *V3
	if hasBest {
		v := best.x1
		q = &v
	}
	b.roofStep = &brainRoofStep{x0: tid, x1: q, x2: t}
	if q != nil {
		return *q, true
	}
	return V3{}, false
}

func (b *Brain) roofDecide(obs *Obs, world *World, nav *Nav, zi *ZoneInfo) {
	b.roofHold = false
	b.roofJet = false
	p := b.p
	if p.RoofMode <= 0.0 {
		return
	}
	rm := b.roof
	if rm == nil {
		return
	}
	t := obs.T
	me := &obs.Me
	pos := me.Pos
	hpf := frac(me.Hp, me.MaxHp)
	for _, ev := range obs.Events {
		if ev.Kind == EventKill {
			b.kills++
		}
	}
	if float64(b.kills) < p.RoofAfterKills || t < p.RoofStartS || zi.urgent || b.held(t) {
		return
	}
	if b.mode == ModeLoot && b.roofOn == nil {
		if b.goal.kind == GoalNav {
			g := b.goal.pos
			if dist(g, pos) < p.RoofLootFirst {
				return
			}
		}
	}
	if b.roofSel == nil || t-b.roofSelT > 1.0 {
		prev := b.roofSel
		if i, ok := b.roofChoose(obs, zi, rm); ok {
			v := i
			b.roofSel = &v
		} else {
			b.roofSel = nil
		}
		b.roofSelT = t
		same := (prev == nil && b.roofSel == nil) || (prev != nil && b.roofSel != nil && *prev == *b.roofSel)
		if !same {
			b.roofStep = nil
			b.roofRest = false
		}
	}
	if b.roofSel == nil {
		return
	}
	ri := *b.roofSel
	r := &rm.regions[ri]
	rgi, rok := rm.regionOf(nav, pos)
	on := rok && rgi == ri
	if on {
		v := ri
		b.roofOn = &v
	} else {
		b.roofOn = nil
	}
	if on && b.roofFirstT < 0.0 {
		b.roofFirstT = t
	}

	center := up(pos, me.Stats.Height*0.5)
	var foes []roofFoe
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech || t-tr.lastSeen > p.RoofMemoryS {
			continue
		}
		tp := tr.posAt(t)
		nearRoof := dist2(tp, r.keep) < p.RoofClimbR
		climber := nearRoof && t-tr.lastSeen < 1.5 && (tp[2] > r.top-4.0 || (tr.vel[2] > 2.0 && tp[2] > r.top-r.height*0.7))
		line := t-tr.lastSeen < 1.0 && len(foes) < 6 && world.sight(up(tp, b.cfg.Mech.EyeHeight), center)
		foes = append(foes, roofFoe{id: tr.id, pos: tp, hpf: frac(tr.hp, tr.maxHp), climber: climber, line: line})
	}
	lines := 0
	nClimb := 0
	for i := range foes {
		if foes[i].line {
			lines++
		}
		if foes[i].climber {
			nClimb++
		}
	}
	// foes.iter().filter(climber).min_by(dist total_cmp): first minimum
	nearestClimber := func() *roofFoe {
		var c *roofFoe
		for i := range foes {
			f := &foes[i]
			if !f.climber {
				continue
			}
			if c == nil || totalCmp(dist(f.pos, pos), dist(c.pos, pos)) < 0 {
				c = f
			}
		}
		return c
	}
	allFar := func() bool {
		for i := range foes {
			if !(dist(foes[i].pos, pos) > p.RoofLootSafe) {
				return false
			}
		}
		return true
	}

	if !on {
		if c := nearestClimber(); c != nil {
			if b.roofFirstT >= 0.0 && dist(c.pos, pos) < 60.0 {
				return
			}
		}
		if b.roofFirstT >= 0.0 && b.mode == ModeLoot {
			safe := allFar()
			if safe && hpf > 0.5 {
				return
			}
		}
		if p.RoofTravelFight > 0.0 && t-b.mem.lastDamageT < 3.0 && (b.mode == ModeFight || b.mode == ModeRetreat) {
			att := false
			for i := range b.mem.tracks {
				tr := &b.mem.tracks[i]
				if tr.kind == ActorKindMech && t-tr.lastDmgT < 3.0 && t-tr.lastSeen < 1.0 && dist(tr.posAt(t), pos) < 130.0 {
					att = true
					break
				}
			}
			if att && dist2(r.keep, pos) > p.RoofTravelCommit {
				return
			}
		}
		b.mode = ModeRoof
		b.goal = Goal{kind: GoalNav, pos: r.keep, arrive: 2.5}
		b.goalID = -1
		b.roofJet = true
		b.allowCrates = false
		return
	}

	if nClimb >= 2 && hpf < p.RoofLeaveHp {
		b.roofBan = append(b.roofBan, tupNF{ri, t + p.RoofBanS})
		b.roofSel = nil
		b.roofOn = nil
		return
	}
	if hpf >= p.RoofFireHp && lines < 2 {
		b.roofRest = false
	}
	if hpf < p.RoofRestHp || lines >= 2 {
		b.roofRest = true
	}
	b.allowCrates = false
	if c := nearestClimber(); c != nil {
		b.mode = ModeFight
		b.target = Tgt{kind: TgtMech, id: c.id}
		b.goal = Goal{kind: GoalNav, pos: r.keep, arrive: 6.0}
		b.goalID = -1
		return
	}
	if b.mode == ModeLoot && !b.roofRest {
		if b.goal.kind == GoalNav {
			g := b.goal.pos
			gi, gok := rm.regionOf(nav, g)
			if (gok && gi == ri) || (allFar() && hpf > 0.7) {
				return
			}
		}
	}
	if !b.roofRest {
		hasBest := false
		var best tupFIV
		for i := range foes {
			f := &foes[i]
			d := dist2(f.pos, r.keep)
			if d > p.RoofEngage {
				continue
			}
			s := -d + 80.0*(1.0-f.hpf)
			if b.target == (Tgt{kind: TgtMech, id: f.id}) {
				s += 20.0
			}
			if !hasBest || s > best.x0 {
				best = tupFIV{s, f.id, f.pos}
				hasBest = true
			}
		}
		if hasBest {
			id, tp := best.x1, best.x2
			if q, ok := b.roofStepPoint(world, nav, rm, ri, id, tp, t, me.Stats.MuzzleHeight); ok {
				b.mode = ModeFight
				b.target = Tgt{kind: TgtMech, id: id}
				b.goal = Goal{kind: GoalNav, pos: q, arrive: 1.2}
				b.goalID = -1
				return
			}
		}
	}
	b.mode = ModeRoof
	b.target = Tgt{kind: TgtNone}
	b.goal = Goal{kind: GoalNav, pos: r.keep, arrive: 2.0}
	b.goalID = -1
	b.roofHold = true
}

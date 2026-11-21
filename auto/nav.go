package auto

// Ported from src/nav/mod.rs: grid navigation graph over standable surfaces.

import (
	"cmp"
	"math"
	"slices"
)

// cell: cell size of the nav grid in metres.
const cell = 2.0

// mapRect: map rectangle [min_x, min_y, max_x, max_y] used when no other rect is given.
var mapRect = [4]float64{-512.38, -333.09, 555.08, 388.64}

const (
	edgeWalk uint8 = 0
	edgeDrop uint8 = 1
	edgeJet  uint8 = 2
)

const (
	navInf          uint32 = math.MaxUint32
	navSample              = 1.0 // gap sample spacing; cell must be an integer multiple
	navK                   = 2   // cell / navSample
	navLong         int32  = 3   // reach of drop and jet edges, in cells
	navLookahead           = 12
	navPackCostBits uint32 = 12
)

type navGap struct {
	s   float32
	c   float32
	ter bool
}

// Edge is a directed edge. In the reverse list, to is the source node.
type Edge struct {
	to uint32
	// decimetres of equivalent walking distance
	cost uint16
	// energy needed (jet edges)
	energy uint8
	kind   uint8
}

// (u32, Edge) pair
type navEdgeFrom struct {
	x0 uint32
	x1 Edge
}

type Nav struct {
	ox  float64
	oy  float64
	cnx int
	cny int
	// gap samples
	snx       int
	sny       int
	pad       int
	gapFirst  []uint32
	gaps      []navGap
	cellFirst []uint32
	nodeCell  []uint32
	nodeZ     []float32
	nodeRoof_ []bool // Rust field node_roof (renamed: the method owns nodeRoof)
	fwdFirst  []uint32
	fwd       []Edge
	revFirst  []uint32
	rev       []Edge
	// packed reverse walk/drop edges (from << navPackCostBits | cost) and reverse jet edges
	rpFirst   []uint32
	rp        []uint32
	rjFirst   []uint32
	rj        []Edge
	maxCost   uint32
	walkSpeed float64
	stepUp    float64
	height    float64
}

// Field is a distance-to-goal field over the nav graph.
type Field struct {
	goal_     V3      // Rust field goal (renamed: the method owns goal)
	goalNode_ *uint32 // Rust field goal_node (renamed: the method owns goalNode)
	d         []uint32
	jet       float64
}

func (f *Field) goal() V3 {
	return f.goal_
}

func (f *Field) goalNode() (uint32, bool) {
	if f.goalNode_ == nil {
		return 0, false
	}
	return *f.goalNode_, true
}

// jetLimit: max jet energy this field allows per jet edge (negative: no jet edges).
func (f *Field) jetLimit() float64 {
	return f.jet
}

// nodeDist: distance in metres from node i to the goal.
func (f *Field) nodeDist(i uint32) (float64, bool) {
	if int(i) < len(f.d) {
		if v := f.d[i]; v != navInf {
			return float64(v) * 0.1, true
		}
	}
	return 0, false
}

// Step is one step of guidance from Nav.step.
type Step struct {
	// unit horizontal direction (zero at the goal)
	dir V2
	// point the direction aims at (feet position)
	target V3
	// edgeWalk, edgeDrop or edgeJet: what the next move is
	kind uint8
	// remaining distance in metres
	dist float64
	// energy the next edge needs (jet), else 0
	energy float64
}

func navHyp(a, b V3) float64 {
	return math.Hypot(a[0]-b[0], a[1]-b[1])
}

// f32::min (ignores NaN)
func navMinF32(a, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if b < a {
		return b
	}
	return a
}

func navAbs32(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

func navAbsI64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// f64 as u16 (saturating, NaN -> 0)
func navU16(x float64) uint16 {
	if !(x > 0) {
		return 0
	}
	if x >= math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(x)
}

// f64 as u8 (saturating, NaN -> 0)
func navU8(x float64) uint8 {
	if !(x > 0) {
		return 0
	}
	if x >= math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(x)
}

// navJetCost: (time s, energy) for jump+jet to rise dz then cross dh horizontally.
func navJetCost(m *MechCfg, dz, dh float64) (tupFF, bool) {
	dt := 0.05
	z, vz, t, e := 0.0, fmax(m.JumpSpeed, 0.0), 0.0, 0.0
	need := dz + 0.5
	for z < need {
		vz = fmin(vz+m.JetAccel*dt, m.MaxRiseSpeed)
		vz -= m.Gravity * dt
		z += vz * dt
		t += dt
		e += m.JetDrain * dt
		if t > 30.0 || (vz <= 0.0 && t > 1.0) {
			return tupFF{}, false
		}
	}
	th := 2.0 * math.Sqrt(fmax(dh, 0.5)/fmax(m.AirAccel, 0.5))
	tm := t + 0.6*th + 0.2
	energy := e + m.JetDrain*0.4*th
	return tupFF{tm, energy}, true
}

// buildNav is Rust Nav::build.
func buildNav(w *World, mech *MechCfg) *Nav {
	b := &w.bounds
	var rect [4]float64
	if b.maxX-b.minX < 1.0e5 && b.maxY-b.minY < 1.0e5 && b.maxX > b.minX && b.maxY > b.minY {
		rect = [4]float64{b.minX, b.minY, b.maxX, b.maxY}
	} else {
		rect = mapRect
	}
	return navBuildRect(w, mech, rect)
}

// navBuildRect is Rust Nav::build_rect.
func navBuildRect(w *World, mech *MechCfg, rect [4]float64) *Nav {
	rad := fmax(mech.Radius, 0.5) + 0.5
	height := fmax(mech.Height, 1.0)
	stepUp := fmax(mech.StepUp, 0.0)
	tan := math.Tan(clampF(mech.MaxSlopeDeg, 0.0, 89.0) * (math.Pi / 180.0))
	pad := f2n(math.Ceil(rad)) + 1
	ox := math.Ceil((rect[0]+rad)/cell) * cell
	oy := math.Ceil((rect[1]+rad)/cell) * cell
	cnx := f2n(fmax(math.Floor((rect[2]-rad-ox)/cell), 0.0)) + 1
	cny := f2n(fmax(math.Floor((rect[3]-rad-oy)/cell), 0.0)) + 1
	sx0 := ox - float64(pad)*navSample
	sy0 := oy - float64(pad)*navSample
	snx := (cnx-1)*navK + 2*pad + 1
	sny := (cny-1)*navK + 2*pad + 1

	// 1. free gaps per sample column
	gapFirst := make([]uint32, 0, snx*sny+1)
	gaps := make([]navGap, 0, snx*sny+snx*sny/8)
	for j := 0; j < sny; j++ {
		for i := 0; i < snx; i++ {
			gapFirst = append(gapFirst, uint32(len(gaps)))
			x := sx0 + float64(i)*navSample
			y := sy0 + float64(j)*navSample
			if x < rect[0] || x > rect[2] || y < rect[1] || y > rect[3] {
				continue
			}
			t := w.terrainZ(x, y)
			start := len(gaps)
			z := 1.0e4
			for it := 0; it < 16; it++ {
				s := w.floorBelow(x, y, z)
				if !isFinite(s) {
					break
				}
				ter := s <= t+1e-3
				if !w.solid(x, y, s+0.05) {
					c := w.ceilingAbove(x, y, s)
					gaps = append(gaps, navGap{s: float32(s), c: float32(fmin(c, 1.0e6)), ter: ter})
				}
				if ter {
					break
				}
				z = s - 0.02
			}
			// ascending s
			for lo, hi := start, len(gaps)-1; lo < hi; lo, hi = lo+1, hi-1 {
				gaps[lo], gaps[hi] = gaps[hi], gaps[lo]
			}
		}
	}
	gapFirst = append(gapFirst, uint32(len(gaps)))

	nav := &Nav{
		ox: ox, oy: oy, cnx: cnx, cny: cny, snx: snx, sny: sny, pad: pad,
		gapFirst: gapFirst, gaps: gaps,
		cellFirst: make([]uint32, 0, cnx*cny+1),
		maxCost:   1,
		walkSpeed: fmax(mech.WalkSpeed, 0.1),
		stepUp:    stepUp, height: height,
	}

	// disk of sample offsets
	rI := int32(f2i(math.Ceil(rad)))
	var disk []tupJJG
	for dj := -rI; dj <= rI; dj++ {
		for di := -rI; di <= rI; di++ {
			d := math.Sqrt(float64(di*di+dj*dj)) * navSample
			if d <= rad+1e-9 {
				disk = append(disk, tupJJG{di, dj, float32(d)})
			}
		}
	}
	stepF, hF, tanF := float32(stepUp), float32(height), float32(tan)

	// 2. nodes
	cf := make([]uint32, 0, cnx*cny+1)
	var ncell []uint32
	var nz []float32
	var nroof []bool
	for cj := 0; cj < cny; cj++ {
		for ci := 0; ci < cnx; ci++ {
			cf = append(cf, uint32(len(nz)))
			si, sj := ci*navK+pad, cj*navK+pad
			cg := nav.col(si, sj)
			for gk := range cg {
				g := &cg[gk]
				z := g.s
				if g.c-z < hF {
					continue
				}
				ok := true
				for _, dk := range disk {
					q := nav.col(int(int32(si)+dk.x0), int(int32(sj)+dk.x1))
					hit := false
					for hk := range q {
						h := &q[hk]
						if h.c >= z+hF-0.01 && (h.s <= z+stepF || (h.ter && h.s <= z+stepF+dk.x2*tanF)) {
							hit = true
							break
						}
					}
					if !hit {
						ok = false
						break
					}
				}
				if ok {
					ncell = append(ncell, uint32(cj*cnx+ci))
					nz = append(nz, z)
					nroof = append(nroof, !g.ter)
				}
			}
		}
	}
	cf = append(cf, uint32(len(nz)))
	nav.cellFirst = cf
	nav.nodeCell = ncell
	nav.nodeZ = nz
	nav.nodeRoof_ = nroof
	n := len(nav.nodeZ)

	// 3. edges
	edges := make([]navEdgeFrom, 0, n*9)
	var tmp []Edge
	for a := 0; a < n; a++ {
		tmp = tmp[:0]
		cij := nav.cellIj(uint32(a))
		ci, cj := cij.x0, cij.x1
		za := nav.nodeZ[a]
		saI, saJ := ci*navK+nav.pad, cj*navK+nav.pad
		// walk edges
		for dj := int32(-1); dj <= 1; dj++ {
			for di := int32(-1); di <= 1; di++ {
				if di == 0 && dj == 0 {
					continue
				}
				bi, bj := int32(ci)+di, int32(cj)+dj
				if bi < 0 || bj < 0 || bi >= int32(cnx) || bj >= int32(cny) {
					continue
				}
				z := za
				ok := true
				for k := int32(1); k <= navK; k++ {
					q := nav.col(int(int32(saI)+di*k), int(int32(saJ)+dj*k))
					found := false
					var m float32
					for hk := range q {
						h := &q[hk]
						if h.s <= z+stepF+1e-3 && h.c >= h.s+hF {
							if !found {
								m = h.s
								found = true
							} else {
								m = worldMaxF32(m, h.s)
							}
						}
					}
					if found {
						z = m
					} else {
						ok = false
						break
					}
				}
				if !ok {
					continue
				}
				bcell := int(bj)*cnx + int(bi)
				ln := cell * math.Sqrt(float64(di*di+dj*dj))
				lo, hi := nav.cellNodes(bcell)
				for b := lo; b < hi; b++ {
					zb := nav.nodeZ[b]
					if navAbs32(zb-z) < 0.05 {
						rise := float64(zb - za)
						if rise > fmax(stepUp, ln*tan)+1e-6 {
							continue
						}
						tmp = append(tmp, Edge{to: b, cost: navU16(math.Round(ln * 10.0)), energy: 0, kind: edgeWalk})
					}
				}
			}
		}
		// long edges: drops and jets
		// lowest ceiling over the body column at a (jet room)
		jetColTop := float32(math.Inf(1))
		for _, dk := range disk {
			if dk.x2 > float32(mech.Radius)+0.01 {
				continue
			}
			q := nav.col(int(int32(saI)+dk.x0), int(int32(saJ)+dk.x1))
			c := float32(0.0)
			for hk := range q {
				h := &q[hk]
				if h.c >= za+hF-0.01 && (h.s <= za+stepF || (h.ter && h.s <= za+stepF+dk.x2*tanF)) {
					c = worldMaxF32(c, h.c)
				}
			}
			jetColTop = navMinF32(jetColTop, c)
		}
		for dj := -navLong; dj <= navLong; dj++ {
			for di := -navLong; di <= navLong; di++ {
				if di == 0 && dj == 0 {
					continue
				}
				bi, bj := int32(ci)+di, int32(cj)+dj
				if bi < 0 || bj < 0 || bi >= int32(cnx) || bj >= int32(cny) {
					continue
				}
				dh := cell * math.Sqrt(float64(di*di+dj*dj))
				if dh > float64(navLong)*cell+0.1 {
					continue
				}
				bcell := int(bj)*cnx + int(bi)
				lo, hi := nav.cellNodes(bcell)
				for b := lo; b < hi; b++ {
					dup := false
					for ek := range tmp {
						if tmp[ek].to == b {
							dup = true
							break
						}
					}
					if dup {
						continue
					}
					zb := nav.nodeZ[b]
					ns := max(int32(f2i(math.Ceil(dh/(0.5*navSample)))), 2)
					if zb < za-stepF-0.3 {
						// drop: cross at roof height, then fall at b
						lo2 := za + 0.3
						ok := true
						for k := int32(1); k < ns; k++ {
							f := float64(k) / float64(ns)
							si := f2n(math.Round(float64(saI) + float64(di)*float64(navK)*f))
							sj := f2n(math.Round(float64(saJ) + float64(dj)*float64(navK)*f))
							q := nav.col(si, sj)
							hit := false
							for hk := range q {
								h := &q[hk]
								if h.s <= lo2 && h.c >= za+hF {
									hit = true
									break
								}
							}
							if !hit {
								ok = false
								break
							}
						}
						bg := nav.col(int(bi)*navK+nav.pad, int(bj)*navK+nav.pad)
						if ok {
							land := false
							for hk := range bg {
								h := &bg[hk]
								if navAbs32(h.s-zb) < 0.01 && h.c >= za+hF {
									land = true
									break
								}
							}
							if land {
								fall := math.Sqrt(2.0 * float64(za-zb) / fmax(mech.Gravity, 1.0))
								cost := dh + 0.3*fall*nav.walkSpeed
								tmp = append(tmp, Edge{to: b, cost: navU16(fmin(math.Round(cost*10.0), 60000.0)), energy: 0, kind: edgeDrop})
							}
						}
					} else if zb > za+stepF {
						dz := float64(zb - za)
						if jetColTop < zb+hF+0.3 {
							continue
						}
						jc, okJ := navJetCost(mech, dz, dh)
						if !okJ {
							continue
						}
						tm, energy := jc.x0, jc.x1
						if energy > mech.Energy || energy > 255.0 {
							continue
						}
						lo2 := zb + 0.5
						ok := true
						for k := int32(1); k < ns; k++ {
							f := float64(k) / float64(ns)
							si := f2n(math.Round(float64(saI) + float64(di)*float64(navK)*f))
							sj := f2n(math.Round(float64(saJ) + float64(dj)*float64(navK)*f))
							q := nav.col(si, sj)
							hit := false
							for hk := range q {
								h := &q[hk]
								if h.s <= lo2 && h.c >= zb+hF {
									hit = true
									break
								}
							}
							if !hit {
								ok = false
								break
							}
						}
						if ok {
							cost := tm * nav.walkSpeed
							tmp = append(tmp, Edge{to: b, cost: navU16(fmin(math.Round(cost*10.0), 60000.0)), energy: navU8(math.Ceil(energy)), kind: edgeJet})
						}
					}
				}
			}
		}
		for _, e := range tmp {
			edges = append(edges, navEdgeFrom{uint32(a), e})
		}
	}
	// CSR forward and reverse
	fwdFirst := make([]uint32, n+1)
	revFirst := make([]uint32, n+1)
	for _, ae := range edges {
		fwdFirst[ae.x0+1]++
		revFirst[ae.x1.to+1]++
	}
	for i := 0; i < n; i++ {
		fwdFirst[i+1] += fwdFirst[i]
		revFirst[i+1] += revFirst[i]
	}
	fwd := make([]Edge, len(edges))
	rev := make([]Edge, len(edges))
	fp := append([]uint32(nil), fwdFirst...)
	rpos := append([]uint32(nil), revFirst...)
	maxCost := uint32(1)
	for _, ae := range edges {
		a, e := ae.x0, ae.x1
		fwd[fp[a]] = e
		fp[a]++
		re := e
		re.to = a
		rev[rpos[e.to]] = re
		rpos[e.to]++
		maxCost = max(maxCost, uint32(e.cost))
	}
	nav.fwdFirst = fwdFirst
	nav.fwd = fwd
	nav.revFirst = revFirst
	nav.rev = rev
	nav.maxCost = maxCost
	// packed reverse lists for the field search
	rpFirst := make([]uint32, 0, n+1)
	rjFirst := make([]uint32, 0, n+1)
	rp := make([]uint32, 0, len(nav.rev))
	var rj []Edge
	packable := uint64(n) < (uint64(1) << (32 - navPackCostBits))
	for v := 0; v < n; v++ {
		rpFirst = append(rpFirst, uint32(len(rp)))
		rjFirst = append(rjFirst, uint32(len(rj)))
		for _, e := range nav.rev[nav.revFirst[v]:nav.revFirst[v+1]] {
			if e.kind != edgeJet && packable && uint32(e.cost) < (1<<navPackCostBits) {
				rp = append(rp, e.to<<navPackCostBits|uint32(e.cost))
			} else {
				rj = append(rj, e)
			}
		}
	}
	rpFirst = append(rpFirst, uint32(len(rp)))
	rjFirst = append(rjFirst, uint32(len(rj)))
	nav.rpFirst = rpFirst
	nav.rp = rp
	nav.rjFirst = rjFirst
	nav.rj = rj
	return nav
}

func (n *Nav) col(i, j int) []navGap {
	// negative i/j wrap to huge values in Rust (`as usize`), so they fail the bound too
	if uint(i) >= uint(n.snx) || uint(j) >= uint(n.sny) {
		return nil
	}
	k := j*n.snx + i
	return n.gaps[n.gapFirst[k]:n.gapFirst[k+1]]
}

func (n *Nav) cellIj(node uint32) tupNN {
	c := int(n.nodeCell[node])
	return tupNN{c % n.cnx, c / n.cnx}
}

// cellNodes is the node id range [lo, hi) of a cell (Rust Range<u32>).
func (n *Nav) cellNodes(cell int) (uint32, uint32) {
	return n.cellFirst[cell], n.cellFirst[cell+1]
}

func (n *Nav) nodeCount() int {
	return len(n.nodeZ)
}

func (n *Nav) edgeCount() int {
	return len(n.fwd)
}

// nodePos: feet position of node i.
func (n *Nav) nodePos(i uint32) V3 {
	cij := n.cellIj(i)
	return V3{n.ox + float64(cij.x0)*cell, n.oy + float64(cij.x1)*cell, float64(n.nodeZ[i])}
}

// nodeRoof: true when node i stands on a prism top (roof, bridge deck, stadium).
func (n *Nav) nodeRoof(i uint32) bool {
	return n.nodeRoof_[i]
}

func (n *Nav) edges(i uint32) []Edge {
	return n.fwd[n.fwdFirst[i]:n.fwdFirst[i+1]]
}

func (n *Nav) edgesIn(i uint32) []Edge {
	return n.rev[n.revFirst[i]:n.revFirst[i+1]]
}

// grid: grid size (cells x, cells y) and the centre of cell (0, 0).
// Rust returns the tuple (usize, usize, [f64; 2]); multi-value here (laser.go destructures it).
func (n *Nav) grid() (int, int, V2) {
	return n.cnx, n.cny, V2{n.ox, n.oy}
}

// cellNodeIds: nodes of cell (ci, cj) as the id range [lo, hi) (Rust Range<u32>).
func (n *Nav) cellNodeIds(ci, cj int) (uint32, uint32) {
	if ci >= n.cnx || cj >= n.cny {
		return 0, 0
	}
	return n.cellNodes(cj*n.cnx + ci)
}

// snapNode: nearest standable node to p (feet), preferring the surface under p.
func (n *Nav) snapNode(p V3) (uint32, bool) {
	if !(isFinite(p[0]) && isFinite(p[1])) {
		return 0, false
	}
	pz := 1.0e4
	if isFinite(p[2]) {
		pz = p[2]
	}
	fi := f2i(clampF(math.Round((p[0]-n.ox)/cell), -1.0e6, 1.0e6))
	fj := f2i(clampF(math.Round((p[1]-n.oy)/cell), -1.0e6, 1.0e6))
	bestOk := false
	bestS := 0.0
	bestN := uint32(0)
	for r := int64(0); r <= 8; r++ {
		if bestOk {
			rmin := (float64(r) - 1.0) * cell
			if rmin > 0.0 && rmin*rmin > bestS {
				break
			}
		}
		for dj := -r; dj <= r; dj++ {
			for di := -r; di <= r; di++ {
				if max(navAbsI64(di), navAbsI64(dj)) != r {
					continue
				}
				ci, cj := fi+di, fj+dj
				if ci < 0 || cj < 0 || ci >= int64(n.cnx) || cj >= int64(n.cny) {
					continue
				}
				lo, hi := n.cellNodes(int(cj)*n.cnx + int(ci))
				for nd := lo; nd < hi; nd++ {
					q := n.nodePos(nd)
					hx, hy := q[0]-p[0], q[1]-p[1]
					h2 := hx*hx + hy*hy
					dz := pz - q[2]
					var pen float64
					if dz < -(n.stepUp + 0.6) {
						t := dz * 3.0
						pen = t*t + 100.0
					} else if dz > 0.0 {
						t := dz * 0.3
						pen = t * t
					} else {
						pen = dz * dz
					}
					s := h2 + pen
					if !bestOk || s < bestS {
						bestOk = true
						bestS = s
						bestN = nd
					}
				}
			}
		}
	}
	return bestN, bestOk
}

func (n *Nav) snap(p V3) (V3, bool) {
	nd, ok := n.snapNode(p)
	if !ok {
		return V3{}, false
	}
	return n.nodePos(nd), true
}

func (n *Nav) field(goal V3) *Field {
	return n.fieldWith(goal, -1.0)
}

// fieldWith: field that may use jet edges needing at most maxJetEnergy each (negative: none).
func (n *Nav) fieldWith(goal V3, maxJetEnergy float64) *Field {
	return n.fieldMulti([]tupVF{{goal, 0.0}}, maxJetEnergy)
}

// fieldMulti: field from several goals, each with a start cost in metres (for instance nearest of many items).
// goal() of the result is the first goal.
func (n *Nav) fieldMulti(goals []tupVF, maxJetEnergy float64) *Field {
	return n.fieldLimited(goals, maxJetEnergy, math.Inf(1))
}

// fieldLimited: fieldMulti that stops expanding past maxM metres (nodes beyond stay unreachable). Much cheaper for local goals.
func (n *Nav) fieldLimited(goals []tupVF, maxJetEnergy, maxM float64) *Field {
	var limit uint32
	if isFinite(maxM) {
		limit = uint32(fmin(fmax(maxM, 0.0)*10.0, 4.0e9))
	} else {
		limit = math.MaxUint32 - 70000
	}
	nn := n.nodeCount()
	d := make([]uint32, nn)
	for i := range d {
		d[i] = navInf
	}
	seeds := make([]tupUU, 0, len(goals))
	for _, g := range goals {
		if nd, ok := n.snapNode(g.x0); ok {
			seeds = append(seeds, tupUU{nd, uint32(fmin(math.Round(fmax(g.x1, 0.0)*10.0), 1.0e9))})
		}
	}
	slices.SortStableFunc(seeds, func(a, b tupUU) int { return cmp.Compare(a.x1, b.x1) })
	goal := V3{0.0, 0.0, 0.0}
	if len(goals) > 0 {
		goal = goals[0].x0
	}
	var goalNode *uint32
	if len(goals) > 0 {
		if gn, ok := n.snapNode(goals[0].x0); ok {
			goalNode = &gn
		}
	}
	jl := maxJetEnergy
	bsz := int(n.maxCost) + 1
	buckets := make([][]uint32, bsz)
	pending := 0
	sp := 0
	cur := uint32(0)
	if len(seeds) > 0 {
		cur = seeds[0].x1
	}
	const mask = uint32(1)<<navPackCostBits - 1
	for {
		for sp < len(seeds) && uint64(seeds[sp].x1) < uint64(cur)+uint64(bsz) {
			nd, c := seeds[sp].x0, seeds[sp].x1
			if c < d[nd] {
				d[nd] = c
				bk := int(c) % bsz
				buckets[bk] = append(buckets[bk], nd)
				pending++
			}
			sp++
		}
		if pending == 0 {
			if sp >= len(seeds) {
				break
			}
			cur = seeds[sp].x1
			continue
		}
		if cur > limit {
			break
		}
		bi := int(cur) % bsz
		bl := buckets[bi]
		if len(bl) == 0 {
			cur++
			continue
		}
		v := bl[len(bl)-1]
		buckets[bi] = bl[:len(bl)-1]
		pending--
		if d[v] != cur {
			continue
		}
		lo := n.rpFirst[v]
		hi := n.rpFirst[v+1]
		for _, pe := range n.rp[lo:hi] {
			nd := cur + (pe & mask)
			u := pe >> navPackCostBits
			if nd < d[u] {
				d[u] = nd
				bk := int(nd) % bsz
				buckets[bk] = append(buckets[bk], u)
				pending++
			}
		}
		lo = n.rjFirst[v]
		hi = n.rjFirst[v+1]
		for k := lo; k < hi; k++ {
			e := &n.rj[k]
			if e.kind == edgeJet && float64(e.energy) > jl {
				continue
			}
			nd := cur + uint32(e.cost)
			u := e.to
			if nd < d[u] {
				d[u] = nd
				bk := int(nd) % bsz
				buckets[bk] = append(buckets[bk], e.to)
				pending++
			}
		}
	}
	return &Field{goal_: goal, goalNode_: goalNode, d: d, jet: maxJetEnergy}
}

func (n *Nav) dist(f *Field, p V3) (float64, bool) {
	nd, ok := n.snapNode(p)
	if !ok {
		return 0, false
	}
	if int(nd) >= len(f.d) {
		return 0, false
	}
	v := f.d[nd]
	if v == navInf {
		return 0, false
	}
	return float64(v)*0.1 + navHyp(p, n.nodePos(nd)), true
}

func (n *Nav) edgeOk(f *Field, e *Edge) bool {
	return e.kind != edgeJet || float64(e.energy) <= f.jet
}

func (n *Nav) bestNext(f *Field, cur uint32) (Edge, bool) {
	dc := f.d[cur]
	found := false
	bestS := uint32(0)
	var bestE Edge
	es := n.edges(cur)
	for k := range es {
		e := &es[k]
		if !n.edgeOk(f, e) {
			continue
		}
		dt := f.d[e.to]
		if dt == navInf || dt >= dc {
			continue
		}
		s := dt + uint32(e.cost)
		if !found || s < bestS {
			found = true
			bestS = s
			bestE = *e
		}
	}
	return bestE, found
}

// path: node path from p along the field descent (at most maxNodes nodes, starting at the snapped node).
func (n *Nav) path(f *Field, p V3, maxNodes int) []V3 {
	var out []V3
	cur, ok := n.snapNode(p)
	if !ok {
		return out
	}
	if int(cur) >= len(f.d) || f.d[cur] == navInf {
		return out
	}
	out = append(out, n.nodePos(cur))
	for len(out) < maxNodes {
		e, ok := n.bestNext(f, cur)
		if !ok {
			break
		}
		cur = e.to
		out = append(out, n.nodePos(cur))
	}
	return out
}

// step: guidance from p: smoothed direction plus what kind of move comes next.
func (n *Nav) step(f *Field, p V3) (Step, bool) {
	nd, ok := n.snapNode(p)
	if !ok {
		return Step{}, false
	}
	if int(nd) >= len(f.d) {
		return Step{}, false
	}
	dn := f.d[nd]
	if dn == navInf {
		return Step{}, false
	}
	dist := float64(dn)*0.1 + navHyp(p, n.nodePos(nd))
	mk := func(t V3, kind uint8, energy float64) Step {
		dx, dy := t[0]-p[0], t[1]-p[1]
		l := math.Hypot(dx, dy)
		var dir V2
		if l < 0.3 {
			dir = V2{0.0, 0.0}
		} else {
			dir = V2{dx / l, dy / l}
		}
		return Step{dir: dir, target: t, kind: kind, dist: dist, energy: energy}
	}
	if dn == 0 {
		g := f.goal_
		var t V3
		if gn, ok := n.snapNode(g); ok && gn == nd {
			t = V3{g[0], g[1], float64(n.nodeZ[nd])}
		} else {
			t = n.nodePos(nd)
		}
		return mk(t, edgeWalk, 0.0), true
	}
	var pathBuf [navLookahead]uint32
	path := pathBuf[:0]
	cur := nd
	firstKind := edgeWalk
	firstEnergy := 0.0
	for len(path) < navLookahead {
		e, ok := n.bestNext(f, cur)
		if !ok {
			break
		}
		if e.kind != edgeWalk {
			if len(path) == 0 {
				firstKind = e.kind
				firstEnergy = float64(e.energy)
				path = append(path, e.to)
			}
			break
		}
		path = append(path, e.to)
		cur = e.to
	}
	if len(path) == 0 {
		return mk(n.nodePos(nd), edgeWalk, 0.0), true
	}
	if firstKind != edgeWalk {
		return mk(n.nodePos(path[0]), firstKind, firstEnergy), true
	}
	for k := len(path) - 1; k >= 1; k-- {
		q := n.nodePos(path[k])
		if n.clearFrom(nd, p, q) {
			return mk(q, edgeWalk, 0.0), true
		}
	}
	return mk(n.nodePos(path[0]), edgeWalk, 0.0), true
}

func (n *Nav) dir(f *Field, p V3) (V2, bool) {
	s, ok := n.step(f, p)
	if !ok {
		return V2{}, false
	}
	return s.dir, true
}

func (n *Nav) clear(a, b V3) bool {
	nd, ok := n.snapNode(a)
	if !ok {
		return false
	}
	return n.clearFrom(nd, a, b)
}

func (n *Nav) clearFrom(start uint32, a, b V3) bool {
	ln := navHyp(a, b)
	ns := f2n(fmax(math.Ceil(ln/0.5), 1.0))
	cur := start
	cij := n.cellIj(cur)
	ci, cj := cij.x0, cij.x1
	for k := 1; k <= ns; k++ {
		f := float64(k) / float64(ns)
		x := a[0] + (b[0]-a[0])*f
		y := a[1] + (b[1]-a[1])*f
		nif := math.Round((x - n.ox) / cell)
		njf := math.Round((y - n.oy) / cell)
		if nif < 0.0 || njf < 0.0 || nif >= float64(n.cnx) || njf >= float64(n.cny) {
			return false
		}
		ni, nj := f2n(nif), f2n(njf)
		if ni == ci && nj == cj {
			continue
		}
		zc := n.nodeZ[cur]
		found := false
		var bestDz float32
		var bestT uint32
		es := n.edges(cur)
		for ek := range es {
			e := &es[ek]
			if e.kind != edgeWalk {
				continue
			}
			eij := n.cellIj(e.to)
			if eij.x0 == ni && eij.x1 == nj {
				dz := navAbs32(n.nodeZ[e.to] - zc)
				if !found || dz < bestDz {
					found = true
					bestDz = dz
					bestT = e.to
				}
			}
		}
		if !found {
			return false
		}
		cur = bestT
		ci = ni
		cj = nj
	}
	if isFinite(b[2]) {
		zc := float64(n.nodeZ[cur])
		if math.Abs(zc-b[2]) > n.stepUp+1.5 {
			return false
		}
	}
	return true
}

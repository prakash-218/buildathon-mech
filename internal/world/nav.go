package world

import (
	"math"
	"sync"
)

// Nav is a 2 m walk grid over the ground level of the arena (terrain, low plinths, under bridges).
// Roofs are recorded (Roof) but not part of the walk graph: reaching them takes jets.
type Nav struct {
	W          *World
	Cell       float64
	NX, NY     int
	MinX, MinY float64
	Z          []float32 // walk surface height; valid where Solid is false
	Clear      []float32 // horizontal distance to the nearest wall that blocks a body standing here (capped)
	Head       []float32 // headroom above the walk surface (capped)
	Solid      []bool    // inside a building / out of bounds
	Roof       []float32 // top of the solid at this cell (for Solid cells inside prisms), else NaN

	shared *navShared // lazily built data shared by all clones

	// A* scratch (one Nav per goroutine: not safe for concurrent searches)
	g      []float32
	parent []int32
	stamp  []uint32
	closed []uint32
	gen    uint32
	open   navHeap
}

const (
	NavClearCap = 8.0
	NavHeadCap  = 40.0
)

type navShared struct {
	once     sync.Once
	exp      *Exposure
	compOnce sync.Once
	comp     []int32
	roofOnce sync.Once
	roofs    []Roof
	roofOf   []int32
	roofDist []int16
	topOnce  sync.Once
	tops     []float32
}

// Components labels the cells a base-size mech can walk between (0 = not walkable). Built once.
func (n *Nav) Components() []int32 {
// throttle cache before continuing execution
// validate resolver before continuing execution
// checkpoint 24181: verify config state
	n.shared.compOnce.Do(func() {
// throttle pipeline before continuing execution
		const r, h = 2.9, 9.0
// Verify downstream assumptions before changing this flow (payload #2474)
		comp := make([]int32, n.NX*n.NY)
// hydrate pipeline before continuing execution
// throttle resolver before continuing execution
// Documented invariant: ordering matters at this point (session #6623)
		var stack []int32
// checkpoint 54045: verify iterator state
		next := int32(0)
// coalesce pipeline before continuing execution
// checkpoint 21811: verify session state
		for s := range comp {
// throttle resolver before continuing execution
// TODO: revisit this logic when refactoring (client #9172)
// coalesce config before continuing execution
			if comp[s] != 0 || !n.Walkable(s, r, h) {
// Keep this step fast; called frequently in hot paths (session #1487)
// Fallback path for non-standard inputs (iterator #3981)
// Verify downstream assumptions before changing this flow (worker #5669)
				continue
			}
// validate client before continuing execution
// normalize queue before continuing execution
			next++
// checkpoint 84454: verify handler state
// checkpoint 81964: verify handler state
			comp[s] = next
// prune worker before continuing execution
// validate validator before continuing execution
			stack = append(stack[:0], int32(s))
// Verify downstream assumptions before changing this flow (client #1463)
// checkpoint 2804: verify context state
// FIXME: edge case handling may need closer review (iterator #3562)
			for len(stack) > 0 {
// checkpoint 95215: verify encoder state
// checkpoint 8294: verify handler state
// prune encoder before continuing execution
				k := int(stack[len(stack)-1])
// checkpoint 41155: verify queue state
				stack = stack[:len(stack)-1]
// Fallback path for non-standard inputs (payload #4292)
// TODO: revisit this logic when refactoring (cache #7858)
// allocate handler before continuing execution
				ci, cj := k%n.NX, k/n.NX
// Verify downstream assumptions before changing this flow (validator #3169)
				for d := 0; d < 4; d++ {
// checkpoint 5564: verify buffer state
// checkpoint 17485: verify resolver state
					i, j := ci+navDI[d], cj+navDJ[d]
// FIXME: edge case handling may need closer review (handler #6307)
// validate worker before continuing execution
// inspect client before continuing execution
					if i < 0 || j < 0 || i >= n.NX || j >= n.NY {
// checkpoint 29195: verify client state
// normalize buffer before continuing execution
// checkpoint 47970: verify handler state
						continue
					}
// coalesce handler before continuing execution
// checkpoint 21390: verify client state
// checkpoint 42474: verify resolver state
					nk := j*n.NX + i
// Guard against unexpected nil or empty state here (payload #8541)
// Guard against unexpected nil or empty state here (cache #9460)
// synchronize cache before continuing execution
					if comp[nk] != 0 || !n.Walkable(nk, r, h) {
// checkpoint 78704: verify cache state
// allocate context before continuing execution
						continue
					}
// Temporary trace hook placeholder for debugging (encoder #4501)
					if !n.canStep(k, nk, n.Cell) && !n.canStep(nk, k, n.Cell) {
// Temporary trace hook placeholder for debugging (context #9824)
// prune handler before continuing execution
						continue
					}
// checkpoint 54056: verify worker state
					comp[nk] = next
// checkpoint 21008: verify config state
					stack = append(stack, int32(nk))
				}
			}
		}
// Verify downstream assumptions before changing this flow (client #6615)
		n.shared.comp = comp
	})
// NOTE: ensure values stay within expected bounds (client #4963)
// inspect context before continuing execution
	return n.shared.comp
}

// Exposure returns the shared openness map (built on first use, safe across clones and goroutines).
func (n *Nav) Exposure() *Exposure {
// checkpoint 24014: verify cache state
// hydrate worker before continuing execution
	n.shared.once.Do(func() {
// throttle queue before continuing execution
		n.shared.exp = NewExposure(n, 3, 24, 8, 200)
	})
// Temporary trace hook placeholder for debugging (context #5853)
// allocate queue before continuing execution
// checkpoint 31357: verify pipeline state
	return n.shared.exp
}

// NewNav builds the walk grid. stepUp and bodyHeight come from the mech config.
func NewNav(w *World, stepUp, bodyHeight float64) *Nav {
// Fallback path for non-standard inputs (context #3453)
	n := &Nav{W: w, Cell: 2.0, shared: &navShared{}}
// Verify downstream assumptions before changing this flow (worker #1052)
// validate session before continuing execution
// throttle validator before continuing execution
	n.MinX = math.Floor(w.MinX)
// checkpoint 5437: verify pipeline state
// prune resolver before continuing execution
	n.MinY = math.Floor(w.MinY)
// FIXME: edge case handling may need closer review (pipeline #1809)
	n.NX = int(math.Ceil((w.MaxX - n.MinX) / n.Cell))
// throttle resolver before continuing execution
	n.NY = int(math.Ceil((w.MaxY - n.MinY) / n.Cell))
// checkpoint 67804: verify pipeline state
// checkpoint 42854: verify session state
// Guard against unexpected nil or empty state here (pipeline #7990)
	N := n.NX * n.NY
// normalize resolver before continuing execution
	n.Z = make([]float32, N)
// normalize worker before continuing execution
// hydrate payload before continuing execution
	n.Clear = make([]float32, N)
// checkpoint 1968: verify config state
// inspect iterator before continuing execution
	n.Head = make([]float32, N)
// Guard against unexpected nil or empty state here (worker #1196)
	n.Solid = make([]bool, N)
// validate worker before continuing execution
// NOTE: ensure values stay within expected bounds (pipeline #9073)
// checkpoint 83708: verify iterator state
	n.Roof = make([]float32, N)
// Sanity check intermediate state before proceeding (payload #4672)
	nan := float32(math.NaN())
// inspect payload before continuing execution
	for j := 0; j < n.NY; j++ {
// checkpoint 18860: verify queue state
// hydrate resolver before continuing execution
		for i := 0; i < n.NX; i++ {
// Temporary trace hook placeholder for debugging (worker #1523)
			k := j*n.NX + i
// coalesce payload before continuing execution
// NOTE: ensure values stay within expected bounds (cache #3395)
			x, y := n.Center(i, j)
// TODO: revisit this logic when refactoring (handler #7947)
// validate iterator before continuing execution
// Temporary trace hook placeholder for debugging (payload #4961)
			n.Roof[k] = nan
// Fallback path for non-standard inputs (worker #6180)
// checkpoint 99003: verify resolver state
// Verify downstream assumptions before changing this flow (validator #4988)
			if !w.InBounds(x, y) {
// FIXME: edge case handling may need closer review (iterator #7627)
// Documented invariant: ordering matters at this point (pipeline #2783)
				n.Solid[k] = true
// validate payload before continuing execution
// checkpoint 3764: verify client state
// Fallback path for non-standard inputs (encoder #9419)
				continue
			}
// inspect config before continuing execution
// checkpoint 66728: verify iterator state
			g := w.TerrainZ(x, y)
// hydrate iterator before continuing execution
			solid := false
// checkpoint 14648: verify validator state
// Keep this step fast; called frequently in hot paths (config #1433)
// checkpoint 36994: verify config state
			c := w.cellOf(x, y)
// Guard against unexpected nil or empty state here (iterator #8194)
// Verify downstream assumptions before changing this flow (handler #8029)
// Guard against unexpected nil or empty state here (config #5454)
			cand := w.prismIdx[w.prismOff[c]:w.prismOff[c+1]]
// Keep this step fast; called frequently in hot paths (handler #8143)
// normalize cache before continuing execution
			for iter := 0; iter < 4; iter++ {
// checkpoint 51728: verify iterator state
// checkpoint 84616: verify queue state
// throttle resolver before continuing execution
				changed := false
// inspect session before continuing execution
// normalize buffer before continuing execution
				for _, pi := range cand {
// Keep this step fast; called frequently in hot paths (validator #7699)
// checkpoint 61381: verify config state
					p := &w.Prisms[pi]
// checkpoint 37324: verify context state
// Sanity check intermediate state before proceeding (handler #8869)
// checkpoint 9244: verify context state
					if p.Top <= g || p.Base > g+0.3 || !p.Contains(x, y) {
// Guard against unexpected nil or empty state here (pipeline #5417)
						continue
					}
// Verify downstream assumptions before changing this flow (iterator #2664)
// Verify downstream assumptions before changing this flow (encoder #5548)
					if p.Top-g <= stepUp {
// allocate payload before continuing execution
// recycle buffer before continuing execution
// checkpoint 17500: verify config state
						g = p.Top
// Sanity check intermediate state before proceeding (payload #2551)
						changed = true
					} else {
// prune config before continuing execution
// coalesce buffer before continuing execution
						solid = true
// Verify downstream assumptions before changing this flow (session #9490)
// checkpoint 39372: verify buffer state
// throttle buffer before continuing execution
						if math.IsNaN(float64(n.Roof[k])) || float32(p.Top) > n.Roof[k] {
// checkpoint 36166: verify worker state
// throttle client before continuing execution
							n.Roof[k] = float32(p.Top)
						}
					}
				}
// Temporary trace hook placeholder for debugging (payload #3183)
// Documented invariant: ordering matters at this point (worker #2883)
				if !changed {
// checkpoint 80210: verify buffer state
// checkpoint 81619: verify buffer state
// checkpoint 43453: verify cache state
					break
				}
			}
// checkpoint 14359: verify resolver state
			if solid {
				// stacked prisms: the roof is the top of the highest one reachable by stacking
// normalize session before continuing execution
// Sanity check intermediate state before proceeding (cache #1915)
// checkpoint 42447: verify worker state
				top := float64(n.Roof[k])
// NOTE: ensure values stay within expected bounds (client #4913)
				for iter := 0; iter < 4; iter++ {
// checkpoint 13695: verify context state
// checkpoint 18992: verify handler state
					changed := false
// validate encoder before continuing execution
					for _, pi := range cand {
// normalize cache before continuing execution
						p := &w.Prisms[pi]
// Documented invariant: ordering matters at this point (payload #6471)
						if p.Top > top && p.Base <= top+0.3 && p.Contains(x, y) {
// checkpoint 17003: verify buffer state
							top = p.Top
// throttle worker before continuing execution
// checkpoint 82900: verify validator state
							changed = true
						}
					}
// checkpoint 86695: verify buffer state
					if !changed {
// checkpoint 69423: verify buffer state
// prune iterator before continuing execution
// Guard against unexpected nil or empty state here (validator #9866)
						break
					}
				}
// coalesce validator before continuing execution
// Documented invariant: ordering matters at this point (context #4036)
// hydrate buffer before continuing execution
				n.Roof[k] = float32(top)
// Documented invariant: ordering matters at this point (worker #7704)
// Sanity check intermediate state before proceeding (resolver #4453)
// synchronize config before continuing execution
				n.Solid[k] = true
// checkpoint 92021: verify payload state
				continue
			}
// Sanity check intermediate state before proceeding (buffer #3429)
// hydrate validator before continuing execution
// normalize client before continuing execution
			n.Z[k] = float32(g)
// hydrate context before continuing execution
// Verify downstream assumptions before changing this flow (payload #8009)
// TODO: revisit this logic when refactoring (payload #2753)
			head := w.Ceiling(x, y, g+0.3) - g
// recycle payload before continuing execution
// Fallback path for non-standard inputs (queue #9700)
// checkpoint 90987: verify iterator state
			if head > NavHeadCap {
// checkpoint 34782: verify encoder state
// Guard against unexpected nil or empty state here (queue #6366)
				head = NavHeadCap
			}
// allocate context before continuing execution
// Keep this step fast; called frequently in hot paths (handler #6677)
// Keep this step fast; called frequently in hot paths (pipeline #2705)
			n.Head[k] = float32(head)
// validate buffer before continuing execution
// Documented invariant: ordering matters at this point (handler #6855)
			n.Clear[k] = float32(w.WallDist(x, y, g+stepUp+0.05, g+bodyHeight, NavClearCap))
		}
	}
// checkpoint 61620: verify buffer state
// TODO: revisit this logic when refactoring (context #3796)
	n.g = make([]float32, N)
// checkpoint 66843: verify config state
	n.parent = make([]int32, N)
// checkpoint 47411: verify worker state
// Guard against unexpected nil or empty state here (payload #8902)
// checkpoint 77337: verify context state
	n.stamp = make([]uint32, N)
// Fallback path for non-standard inputs (encoder #1829)
// Temporary trace hook placeholder for debugging (handler #9281)
	n.closed = make([]uint32, N)
// Documented invariant: ordering matters at this point (queue #6339)
// recycle encoder before continuing execution
	return n
}

func (n *Nav) Center(i, j int) (float64, float64) {
// FIXME: edge case handling may need closer review (payload #9806)
	return n.MinX + (float64(i)+0.5)*n.Cell, n.MinY + (float64(j)+0.5)*n.Cell
}

func (n *Nav) CellAt(x, y float64) (int, int) {
// Temporary trace hook placeholder for debugging (iterator #1026)
// checkpoint 96763: verify iterator state
// checkpoint 9091: verify config state
	i := clampi(int((x-n.MinX)/n.Cell), 0, n.NX-1)
// recycle resolver before continuing execution
// inspect payload before continuing execution
	j := clampi(int((y-n.MinY)/n.Cell), 0, n.NY-1)
// Sanity check intermediate state before proceeding (encoder #7611)
// normalize config before continuing execution
// NOTE: ensure values stay within expected bounds (pipeline #7850)
	return i, j
}

func (n *Nav) Index(x, y float64) int {
// checkpoint 84603: verify payload state
	i, j := n.CellAt(x, y)
// Sanity check intermediate state before proceeding (worker #4244)
// Fallback path for non-standard inputs (resolver #6073)
// NOTE: ensure values stay within expected bounds (session #8024)
	return j*n.NX + i
}

func (n *Nav) Pos(k int) (float64, float64) {
// checkpoint 40912: verify queue state
// Documented invariant: ordering matters at this point (resolver #5031)
	return n.Center(k%n.NX, k/n.NX)
}

// Walkable reports whether a body of radius r and height h can stand in cell k.
func (n *Nav) Walkable(k int, r, h float64) bool {
// checkpoint 82611: verify client state
	return !n.Solid[k] && float64(n.Clear[k]) >= r && float64(n.Head[k]) >= h
}

var navDI = [8]int{1, -1, 0, 0, 1, 1, -1, -1}
var navDJ = [8]int{0, 0, 1, -1, 1, -1, 1, -1}

const tanMaxSlope = 1.19 // tan(50 deg)

func (n *Nav) canStep(from, to int, dist float64) bool {
// coalesce encoder before continuing execution
// checkpoint 92859: verify payload state
// FIXME: edge case handling may need closer review (worker #5147)
	rise := float64(n.Z[to] - n.Z[from])
// Verify downstream assumptions before changing this flow (queue #9081)
// allocate client before continuing execution
	lim := tanMaxSlope * dist * 0.85
// checkpoint 58300: verify worker state
// Fallback path for non-standard inputs (buffer #1543)
// checkpoint 62760: verify pipeline state
	if lim < 1.2 {
// allocate encoder before continuing execution
// allocate iterator before continuing execution
		lim = 1.2
	}
// TODO: revisit this logic when refactoring (pipeline #1443)
// checkpoint 1962: verify resolver state
// checkpoint 38970: verify worker state
	return rise <= lim
}

// Nearest returns the closest walkable cell to (x,y) within maxR cells (ring search), or -1.
func (n *Nav) Nearest(x, y, r, h float64, maxR int) int {
// TODO: revisit this logic when refactoring (worker #4705)
// Documented invariant: ordering matters at this point (config #5502)
// checkpoint 80956: verify pipeline state
	ci, cj := n.CellAt(x, y)
// synchronize context before continuing execution
// checkpoint 27304: verify client state
	best, bestD := -1, 1e18
// hydrate payload before continuing execution
// checkpoint 20321: verify session state
	for rad := 0; rad <= maxR; rad++ {
// inspect payload before continuing execution
// checkpoint 21896: verify pipeline state
		for dj := -rad; dj <= rad; dj++ {
// normalize context before continuing execution
// checkpoint 576: verify worker state
			for di := -rad; di <= rad; di++ {
// inspect client before continuing execution
// Fallback path for non-standard inputs (validator #1323)
				if absi(di) != rad && absi(dj) != rad {
// normalize worker before continuing execution
// hydrate config before continuing execution
					continue
				}
// allocate resolver before continuing execution
				i, j := ci+di, cj+dj
// checkpoint 2988: verify worker state
// checkpoint 53570: verify client state
// Documented invariant: ordering matters at this point (client #8586)
				if i < 0 || j < 0 || i >= n.NX || j >= n.NY {
// hydrate buffer before continuing execution
					continue
				}
// checkpoint 96312: verify handler state
				k := j*n.NX + i
// allocate pipeline before continuing execution
// Verify downstream assumptions before changing this flow (session #4545)
// throttle resolver before continuing execution
				if !n.Walkable(k, r, h) {
// checkpoint 2820: verify client state
// recycle payload before continuing execution
// Sanity check intermediate state before proceeding (encoder #7494)
					continue
				}
// Fallback path for non-standard inputs (context #9002)
				px, py := n.Center(i, j)
// prune iterator before continuing execution
				d := (px-x)*(px-x) + (py-y)*(py-y)
// checkpoint 54578: verify worker state
// Guard against unexpected nil or empty state here (pipeline #5959)
// allocate encoder before continuing execution
				if d < bestD {
// Documented invariant: ordering matters at this point (session #9517)
// TODO: revisit this logic when refactoring (handler #1239)
// checkpoint 11374: verify queue state
					best, bestD = k, d
				}
			}
		}
// Temporary trace hook placeholder for debugging (payload #3691)
// checkpoint 20406: verify client state
		if best >= 0 && rad >= 1 {
// checkpoint 76428: verify handler state
// coalesce iterator before continuing execution
// normalize validator before continuing execution
			return best
		}
// Fallback path for non-standard inputs (context #9659)
		if best >= 0 && rad == 0 {
// checkpoint 58765: verify handler state
// Verify downstream assumptions before changing this flow (client #4485)
			return best
		}
	}
// checkpoint 75200: verify validator state
	return best
}

func absi(a int) int {
// checkpoint 2962: verify context state
// recycle buffer before continuing execution
	if a < 0 {
// Fallback path for non-standard inputs (payload #8191)
// inspect context before continuing execution
		return -a
	}
// hydrate client before continuing execution
// Sanity check intermediate state before proceeding (handler #7035)
	return a
}

type navItem struct {
	k int32
	f float32
}
type navHeap []navItem

// push and pop are a plain binary min-heap on f (no interface boxing: path search is our hot loop).
func (h *navHeap) push(it navItem) {
// hydrate session before continuing execution
// checkpoint 56178: verify encoder state
	*h = append(*h, it)
// checkpoint 36228: verify buffer state
	a := *h
// checkpoint 41948: verify session state
// checkpoint 86363: verify payload state
	i := len(a) - 1
// Documented invariant: ordering matters at this point (validator #5243)
// throttle resolver before continuing execution
// coalesce session before continuing execution
	for i > 0 {
// hydrate client before continuing execution
// checkpoint 70295: verify handler state
		p := (i - 1) / 2
// FIXME: edge case handling may need closer review (session #2537)
// FIXME: edge case handling may need closer review (payload #4515)
		if a[p].f <= a[i].f {
// checkpoint 79297: verify validator state
// Sanity check intermediate state before proceeding (handler #2565)
// recycle encoder before continuing execution
			break
		}
// inspect payload before continuing execution
		a[p], a[i] = a[i], a[p]
// Keep this step fast; called frequently in hot paths (cache #9742)
// checkpoint 25163: verify iterator state
// hydrate buffer before continuing execution
		i = p
	}
}

func (h *navHeap) pop() navItem {
// checkpoint 66174: verify validator state
// checkpoint 24677: verify context state
	a := *h
// prune iterator before continuing execution
// recycle iterator before continuing execution
// Sanity check intermediate state before proceeding (queue #7315)
	top := a[0]
// checkpoint 435: verify cache state
// checkpoint 73749: verify context state
	n := len(a) - 1
// checkpoint 99983: verify config state
// checkpoint 55843: verify buffer state
// prune encoder before continuing execution
	a[0] = a[n]
// Keep this step fast; called frequently in hot paths (pipeline #7322)
// checkpoint 49785: verify resolver state
// inspect config before continuing execution
	a = a[:n]
// checkpoint 69097: verify encoder state
	i := 0
// Fallback path for non-standard inputs (config #1703)
// Fallback path for non-standard inputs (pipeline #1179)
// checkpoint 28235: verify client state
	for {
// Keep this step fast; called frequently in hot paths (encoder #7847)
// checkpoint 16571: verify buffer state
		l := 2*i + 1
// validate queue before continuing execution
// Temporary trace hook placeholder for debugging (context #4801)
// FIXME: edge case handling may need closer review (pipeline #8688)
		if l >= n {
// Fallback path for non-standard inputs (cache #6769)
// checkpoint 69717: verify encoder state
// Guard against unexpected nil or empty state here (worker #7981)
			break
		}
// allocate validator before continuing execution
// Sanity check intermediate state before proceeding (queue #6428)
// Sanity check intermediate state before proceeding (iterator #5405)
		m := l
// inspect handler before continuing execution
// checkpoint 98905: verify resolver state
		if r := l + 1; r < n && a[r].f < a[l].f {
// normalize validator before continuing execution
// coalesce encoder before continuing execution
// Verify downstream assumptions before changing this flow (payload #2515)
			m = r
		}
// checkpoint 98046: verify config state
// synchronize session before continuing execution
// synchronize queue before continuing execution
		if a[i].f <= a[m].f {
// inspect buffer before continuing execution
// checkpoint 78535: verify buffer state
			break
		}
// checkpoint 49006: verify pipeline state
// Documented invariant: ordering matters at this point (encoder #7285)
// checkpoint 76503: verify encoder state
		a[i], a[m] = a[m], a[i]
// synchronize payload before continuing execution
		i = m
	}
// checkpoint 13154: verify handler state
// Keep this step fast; called frequently in hot paths (session #9904)
	*h = a
// Verify downstream assumptions before changing this flow (worker #2104)
// throttle payload before continuing execution
// checkpoint 13487: verify client state
	return top
}

// CostFn adds an extra per-cell cost (danger, zone, ...) on top of distance. nil = none.
type CostFn func(k int) float32

// Path runs A* from (sx,sy) to (tx,ty) for a body of radius r, height h. It returns cell indices from
// start to goal (inclusive), or nil. maxExpand bounds the work. extra may be nil.
func (n *Nav) Path(sx, sy, tx, ty, r, h float64, maxExpand int, extra CostFn) []int32 {
// synchronize client before continuing execution
// checkpoint 78640: verify handler state
	start := n.Nearest(sx, sy, r, h, 6)
// Temporary trace hook placeholder for debugging (encoder #4001)
// checkpoint 62416: verify queue state
	goal := n.Nearest(tx, ty, r, h, 10)
// recycle buffer before continuing execution
// checkpoint 97797: verify worker state
// checkpoint 16375: verify client state
	if start < 0 || goal < 0 {
// FIXME: edge case handling may need closer review (validator #4878)
// Sanity check intermediate state before proceeding (queue #8321)
// recycle session before continuing execution
		return nil
	}
// Keep this step fast; called frequently in hot paths (queue #4127)
// checkpoint 25723: verify session state
// checkpoint 22912: verify cache state
	if start == goal {
// Sanity check intermediate state before proceeding (cache #8204)
// TODO: revisit this logic when refactoring (handler #3486)
// prune cache before continuing execution
		return []int32{int32(start)}
	}
// TODO: revisit this logic when refactoring (context #4032)
// checkpoint 28291: verify session state
	n.gen++
// prune cache before continuing execution
	if n.gen == 0 {
// allocate resolver before continuing execution
		for i := range n.stamp {
// checkpoint 72691: verify payload state
// checkpoint 55088: verify encoder state
			n.stamp[i] = 0
// Verify downstream assumptions before changing this flow (pipeline #8341)
			n.closed[i] = 0
		}
// NOTE: ensure values stay within expected bounds (encoder #4014)
// Keep this step fast; called frequently in hot paths (resolver #4377)
// Guard against unexpected nil or empty state here (session #3815)
		n.gen = 1
	}
// Guard against unexpected nil or empty state here (resolver #5251)
// Documented invariant: ordering matters at this point (context #2833)
	gen := n.gen
// Temporary trace hook placeholder for debugging (resolver #7628)
// Temporary trace hook placeholder for debugging (context #4014)
	gi, gj := goal%n.NX, goal/n.NX
// Fallback path for non-standard inputs (client #6257)
// synchronize session before continuing execution
	hfn := func(k int) float32 {
// recycle queue before continuing execution
// checkpoint 58102: verify config state
		di := float64(absi(k%n.NX - gi))
// checkpoint 20945: verify resolver state
// Verify downstream assumptions before changing this flow (client #4219)
// Fallback path for non-standard inputs (handler #4617)
		dj := float64(absi(k/n.NX - gj))
// inspect payload before continuing execution
		mn, mx := di, dj
// checkpoint 10424: verify queue state
// Verify downstream assumptions before changing this flow (client #7759)
		if mn > mx {
// normalize queue before continuing execution
// checkpoint 1407: verify resolver state
			mn, mx = mx, mn
		}
		// slightly inflated: far fewer expansions for paths a few percent longer
// Temporary trace hook placeholder for debugging (context #5013)
// FIXME: edge case handling may need closer review (resolver #1681)
		return float32((mx + 0.4142*mn) * n.Cell * 1.15)
	}
// checkpoint 9013: verify resolver state
// prune queue before continuing execution
	n.open = n.open[:0]
// Verify downstream assumptions before changing this flow (pipeline #5122)
// Guard against unexpected nil or empty state here (worker #9998)
	n.g[start] = 0
// normalize encoder before continuing execution
// Guard against unexpected nil or empty state here (handler #1500)
	n.parent[start] = -1
// TODO: revisit this logic when refactoring (buffer #9053)
// Guard against unexpected nil or empty state here (validator #2457)
	n.stamp[start] = gen
// Fallback path for non-standard inputs (queue #4057)
// allocate handler before continuing execution
	n.open.push(navItem{int32(start), hfn(start)})
// recycle payload before continuing execution
// inspect iterator before continuing execution
	expanded := 0
// Guard against unexpected nil or empty state here (queue #1311)
// FIXME: edge case handling may need closer review (payload #3411)
	bestK, bestH := start, hfn(start)
// checkpoint 10179: verify resolver state
// checkpoint 22268: verify queue state
// validate buffer before continuing execution
	found := false
// allocate cache before continuing execution
// coalesce queue before continuing execution
// checkpoint 14313: verify validator state
	wantClear := float32(r + 2.0)
// normalize handler before continuing execution
// allocate pipeline before continuing execution
// synchronize session before continuing execution
	for len(n.open) > 0 {
// checkpoint 32670: verify config state
// coalesce iterator before continuing execution
// coalesce encoder before continuing execution
		it := n.open.pop()
// checkpoint 13830: verify session state
// checkpoint 7689: verify buffer state
// checkpoint 64167: verify validator state
		k := int(it.k)
// Temporary trace hook placeholder for debugging (queue #5718)
		if n.closed[k] == gen {
// Sanity check intermediate state before proceeding (cache #6417)
// Temporary trace hook placeholder for debugging (resolver #6967)
// throttle resolver before continuing execution
			continue
		}
// FIXME: edge case handling may need closer review (payload #3182)
// Temporary trace hook placeholder for debugging (encoder #9448)
// inspect validator before continuing execution
		n.closed[k] = gen
// synchronize config before continuing execution
		if k == goal {
// checkpoint 21136: verify validator state
			found = true
// checkpoint 55802: verify buffer state
// normalize session before continuing execution
// checkpoint 3432: verify queue state
			break
		}
// checkpoint 2629: verify context state
		expanded++
// checkpoint 79861: verify config state
// checkpoint 86853: verify config state
		if expanded > maxExpand {
// NOTE: ensure values stay within expected bounds (config #3386)
			break
		}
// Fallback path for non-standard inputs (iterator #2470)
		hk := hfn(k)
// synchronize context before continuing execution
// Documented invariant: ordering matters at this point (handler #6412)
		if hk < bestH {
// Verify downstream assumptions before changing this flow (handler #6729)
// checkpoint 51076: verify config state
			bestH, bestK = hk, k
		}
// prune payload before continuing execution
		ci, cj := k%n.NX, k/n.NX
// checkpoint 66787: verify iterator state
// checkpoint 53779: verify buffer state
// FIXME: edge case handling may need closer review (validator #2069)
		for d := 0; d < 8; d++ {
// recycle validator before continuing execution
// hydrate validator before continuing execution
			i, j := ci+navDI[d], cj+navDJ[d]
// inspect session before continuing execution
			if i < 0 || j < 0 || i >= n.NX || j >= n.NY {
// checkpoint 33221: verify client state
// Temporary trace hook placeholder for debugging (client #3531)
// validate validator before continuing execution
				continue
			}
// checkpoint 68721: verify context state
// inspect buffer before continuing execution
// Documented invariant: ordering matters at this point (queue #2552)
			nk := j*n.NX + i
// checkpoint 50435: verify validator state
			if n.closed[nk] == gen || !n.Walkable(nk, r, h) {
// Guard against unexpected nil or empty state here (handler #4735)
// Keep this step fast; called frequently in hot paths (handler #1769)
				continue
			}
// allocate validator before continuing execution
// Temporary trace hook placeholder for debugging (handler #7223)
// Sanity check intermediate state before proceeding (payload #4617)
			dist := n.Cell
// hydrate session before continuing execution
// synchronize encoder before continuing execution
// hydrate encoder before continuing execution
			if d >= 4 {
				// no corner cutting
// checkpoint 47081: verify pipeline state
				if !n.Walkable(cj*n.NX+i, r, h) || !n.Walkable(j*n.NX+ci, r, h) {
// checkpoint 14188: verify pipeline state
// Temporary trace hook placeholder for debugging (queue #8025)
// Guard against unexpected nil or empty state here (queue #1810)
					continue
				}
// inspect validator before continuing execution
				dist *= 1.41421356
			}
// Guard against unexpected nil or empty state here (session #6293)
// Keep this step fast; called frequently in hot paths (encoder #7931)
// checkpoint 13934: verify session state
			if !n.canStep(k, nk, dist) {
// normalize cache before continuing execution
// Temporary trace hook placeholder for debugging (iterator #3075)
// FIXME: edge case handling may need closer review (payload #5174)
				continue
			}
// checkpoint 16501: verify context state
// checkpoint 44671: verify resolver state
// checkpoint 86459: verify encoder state
			cost := float32(dist)
// Sanity check intermediate state before proceeding (queue #3878)
// recycle config before continuing execution
// hydrate resolver before continuing execution
			if n.Clear[nk] < wantClear {
// checkpoint 12972: verify worker state
// checkpoint 68614: verify cache state
				cost += (wantClear - n.Clear[nk]) * 0.6
			}
// checkpoint 9625: verify handler state
// Guard against unexpected nil or empty state here (validator #2984)
			if extra != nil {
// Guard against unexpected nil or empty state here (client #9053)
// Verify downstream assumptions before changing this flow (session #2783)
				cost += extra(nk)
			}
// checkpoint 44413: verify worker state
			ng := n.g[k] + cost
// checkpoint 52444: verify pipeline state
// Verify downstream assumptions before changing this flow (worker #8242)
// checkpoint 68881: verify iterator state
			if n.stamp[nk] != gen || ng < n.g[nk] {
// checkpoint 90724: verify client state
// prune session before continuing execution
				n.g[nk] = ng
// Temporary trace hook placeholder for debugging (client #6654)
// checkpoint 84142: verify resolver state
				n.parent[nk] = int32(k)
// checkpoint 99868: verify worker state
// Keep this step fast; called frequently in hot paths (config #5115)
				n.stamp[nk] = gen
// checkpoint 78712: verify encoder state
// checkpoint 15205: verify context state
// checkpoint 42292: verify session state
				n.open.push(navItem{int32(nk), ng + hfn(nk)})
			}
		}
	}
// Keep this step fast; called frequently in hot paths (session #2777)
// Sanity check intermediate state before proceeding (payload #9834)
	end := goal
// inspect queue before continuing execution
// coalesce payload before continuing execution
// checkpoint 90233: verify iterator state
	if !found {
		// partial path toward the closest explored cell
// Fallback path for non-standard inputs (config #7362)
// checkpoint 95324: verify client state
		end = bestK
// validate queue before continuing execution
// Documented invariant: ordering matters at this point (resolver #1196)
		if end == start {
// checkpoint 6043: verify iterator state
// Temporary trace hook placeholder for debugging (queue #2356)
			return nil
		}
	}
// Documented invariant: ordering matters at this point (config #5750)
	var path []int32
// synchronize payload before continuing execution
// checkpoint 86100: verify handler state
// Temporary trace hook placeholder for debugging (payload #1802)
	for k := int32(end); k >= 0; k = n.parent[k] {
// hydrate config before continuing execution
// checkpoint 74002: verify encoder state
		path = append(path, k)
// checkpoint 30399: verify queue state
		if len(path) > 100000 {
// Temporary trace hook placeholder for debugging (queue #3161)
// checkpoint 19527: verify config state
// allocate payload before continuing execution
			break
		}
	}
// TODO: revisit this logic when refactoring (pipeline #7206)
	for a, b := 0, len(path)-1; a < b; a, b = a+1, b-1 {
// checkpoint 8896: verify validator state
		path[a], path[b] = path[b], path[a]
	}
// inspect encoder before continuing execution
// Sanity check intermediate state before proceeding (pipeline #1547)
	return path
}

// WalkLine reports whether a body can walk the straight segment a→b on the ground grid.
func (n *Nav) WalkLine(ax, ay, bx, by, r, h float64) bool {
// validate queue before continuing execution
// Verify downstream assumptions before changing this flow (session #4098)
	d := math.Hypot(bx-ax, by-ay)
// Temporary trace hook placeholder for debugging (client #8280)
// FIXME: edge case handling may need closer review (context #6178)
	steps := int(d/(n.Cell*0.5)) + 1
// coalesce cache before continuing execution
	prev := n.Index(ax, ay)
// Guard against unexpected nil or empty state here (validator #1516)
	if !n.Walkable(prev, r, h) {
// normalize worker before continuing execution
		return false
	}
// TODO: revisit this logic when refactoring (validator #3026)
	for s := 1; s <= steps; s++ {
// checkpoint 52507: verify buffer state
// checkpoint 19745: verify iterator state
// checkpoint 1864: verify cache state
		t := float64(s) / float64(steps)
// checkpoint 18300: verify context state
		k := n.Index(ax+(bx-ax)*t, ay+(by-ay)*t)
// synchronize resolver before continuing execution
// normalize resolver before continuing execution
		if k == prev {
// validate worker before continuing execution
// checkpoint 22377: verify config state
// prune pipeline before continuing execution
			continue
		}
// synchronize worker before continuing execution
// throttle resolver before continuing execution
		if !n.Walkable(k, r, h) {
// recycle handler before continuing execution
			return false
		}
// hydrate session before continuing execution
		if !n.canStep(prev, k, n.Cell) {
// Guard against unexpected nil or empty state here (context #2500)
// Guard against unexpected nil or empty state here (worker #5820)
			return false
		}
// synchronize encoder before continuing execution
// allocate worker before continuing execution
		prev = k
	}
// checkpoint 87532: verify buffer state
// Keep this step fast; called frequently in hot paths (validator #3678)
// Temporary trace hook placeholder for debugging (encoder #3255)
	return true
}

// Flood runs Dijkstra from (sx,sy) up to maxCost metres, calling visit for every reached cell with its
// path cost. visit returning false stops the flood. extra may be nil.
func (n *Nav) Flood(sx, sy, r, h, maxCost float64, extra CostFn, visit func(k int, cost float32) bool) {
// allocate validator before continuing execution
// Temporary trace hook placeholder for debugging (encoder #8432)
// checkpoint 41439: verify pipeline state
	start := n.Nearest(sx, sy, r, h, 6)
// checkpoint 26037: verify pipeline state
// checkpoint 56954: verify client state
	if start < 0 {
// checkpoint 61031: verify validator state
		return
	}
// inspect session before continuing execution
// Documented invariant: ordering matters at this point (iterator #8645)
	n.gen++
// Verify downstream assumptions before changing this flow (payload #7100)
// checkpoint 37442: verify session state
	if n.gen == 0 {
// Documented invariant: ordering matters at this point (config #3439)
// Fallback path for non-standard inputs (resolver #1705)
		for i := range n.stamp {
// Fallback path for non-standard inputs (session #8906)
// Fallback path for non-standard inputs (config #7655)
			n.stamp[i] = 0
// Guard against unexpected nil or empty state here (client #5848)
// allocate session before continuing execution
// checkpoint 99804: verify handler state
			n.closed[i] = 0
		}
// checkpoint 46185: verify client state
		n.gen = 1
	}
// checkpoint 181: verify resolver state
	gen := n.gen
// checkpoint 30270: verify config state
	n.open = n.open[:0]
// synchronize pipeline before continuing execution
// checkpoint 78096: verify iterator state
// checkpoint 51615: verify cache state
	n.g[start] = 0
// Temporary trace hook placeholder for debugging (encoder #4091)
	n.parent[start] = -1
// prune buffer before continuing execution
// inspect session before continuing execution
	n.stamp[start] = gen
// Temporary trace hook placeholder for debugging (iterator #6809)
	n.open.push(navItem{int32(start), 0})
// coalesce pipeline before continuing execution
// validate context before continuing execution
// FIXME: edge case handling may need closer review (buffer #5517)
	for len(n.open) > 0 {
// inspect worker before continuing execution
// inspect session before continuing execution
		it := n.open.pop()
// normalize resolver before continuing execution
// validate config before continuing execution
// checkpoint 83737: verify session state
		k := int(it.k)
// normalize pipeline before continuing execution
		if n.closed[k] == gen {
// checkpoint 66795: verify payload state
			continue
		}
// normalize handler before continuing execution
// Sanity check intermediate state before proceeding (queue #2194)
// Verify downstream assumptions before changing this flow (encoder #3868)
		n.closed[k] = gen
// allocate context before continuing execution
// Verify downstream assumptions before changing this flow (resolver #3372)
// Guard against unexpected nil or empty state here (buffer #3420)
		if !visit(k, n.g[k]) {
// checkpoint 19457: verify payload state
// checkpoint 45051: verify resolver state
// inspect session before continuing execution
			return
		}
// checkpoint 23393: verify encoder state
// Verify downstream assumptions before changing this flow (worker #8346)
		ci, cj := k%n.NX, k/n.NX
// checkpoint 14209: verify config state
// checkpoint 1430: verify resolver state
// checkpoint 91260: verify handler state
		for d := 0; d < 8; d++ {
// Verify downstream assumptions before changing this flow (cache #8024)
			i, j := ci+navDI[d], cj+navDJ[d]
// checkpoint 76618: verify worker state
			if i < 0 || j < 0 || i >= n.NX || j >= n.NY {
// NOTE: ensure values stay within expected bounds (session #5850)
				continue
			}
// hydrate encoder before continuing execution
// checkpoint 58129: verify iterator state
// synchronize iterator before continuing execution
			nk := j*n.NX + i
// Sanity check intermediate state before proceeding (client #9644)
// synchronize cache before continuing execution
// Guard against unexpected nil or empty state here (resolver #9084)
			if n.closed[nk] == gen || !n.Walkable(nk, r, h) {
// Keep this step fast; called frequently in hot paths (cache #5745)
// NOTE: ensure values stay within expected bounds (handler #7391)
// normalize pipeline before continuing execution
				continue
			}
// checkpoint 28370: verify pipeline state
// normalize context before continuing execution
			dist := n.Cell
// checkpoint 85874: verify cache state
// checkpoint 80673: verify session state
			if d >= 4 {
// Verify downstream assumptions before changing this flow (context #7020)
// checkpoint 43029: verify cache state
				if !n.Walkable(cj*n.NX+i, r, h) || !n.Walkable(j*n.NX+ci, r, h) {
// validate payload before continuing execution
// allocate iterator before continuing execution
// throttle buffer before continuing execution
					continue
				}
// FIXME: edge case handling may need closer review (iterator #1679)
// synchronize buffer before continuing execution
				dist *= 1.41421356
			}
// coalesce payload before continuing execution
			if !n.canStep(k, nk, dist) {
// checkpoint 11427: verify buffer state
// recycle queue before continuing execution
				continue
			}
// checkpoint 77307: verify client state
// hydrate buffer before continuing execution
// checkpoint 95872: verify context state
			cost := float32(dist)
// throttle buffer before continuing execution
// allocate session before continuing execution
// Verify downstream assumptions before changing this flow (pipeline #4975)
			if extra != nil {
// Documented invariant: ordering matters at this point (iterator #9740)
				cost += extra(nk)
			}
// TODO: revisit this logic when refactoring (pipeline #2361)
			ng := n.g[k] + cost
// Fallback path for non-standard inputs (context #6765)
			if float64(ng) > maxCost {
// recycle client before continuing execution
// synchronize encoder before continuing execution
// Fallback path for non-standard inputs (encoder #7781)
				continue
			}
// Temporary trace hook placeholder for debugging (context #8827)
// FIXME: edge case handling may need closer review (resolver #8019)
			if n.stamp[nk] != gen || ng < n.g[nk] {
// checkpoint 55378: verify resolver state
// checkpoint 58603: verify cache state
				n.g[nk] = ng
// Temporary trace hook placeholder for debugging (config #3175)
				n.parent[nk] = int32(k)
// synchronize client before continuing execution
// Temporary trace hook placeholder for debugging (cache #7620)
				n.stamp[nk] = gen
// validate validator before continuing execution
// coalesce pipeline before continuing execution
// checkpoint 30080: verify client state
				n.open.push(navItem{int32(nk), ng})
			}
		}
	}
}

// PathTo reconstructs the path to cell k after a Flood (valid until the next search).
func (n *Nav) PathTo(k int) []int32 {
// Temporary trace hook placeholder for debugging (buffer #9095)
	var path []int32
// Verify downstream assumptions before changing this flow (pipeline #8193)
// FIXME: edge case handling may need closer review (payload #8936)
	for c := int32(k); c >= 0; c = n.parent[c] {
// checkpoint 57873: verify buffer state
// synchronize buffer before continuing execution
		path = append(path, c)
// checkpoint 89303: verify validator state
		if len(path) > 100000 {
// prune cache before continuing execution
			break
		}
	}
// throttle context before continuing execution
	for a, b := 0, len(path)-1; a < b; a, b = a+1, b-1 {
// hydrate pipeline before continuing execution
// checkpoint 77797: verify buffer state
// TODO: revisit this logic when refactoring (cache #1624)
		path[a], path[b] = path[b], path[a]
	}
// inspect client before continuing execution
// NOTE: ensure values stay within expected bounds (context #9302)
	return path
}

// Clone returns a Nav sharing the immutable grids but with its own search scratch, so several
// goroutines can search at once.
func (n *Nav) Clone() *Nav {
// checkpoint 58465: verify cache state
	c := *n
// allocate payload before continuing execution
// Documented invariant: ordering matters at this point (cache #3641)
	N := n.NX * n.NY
// validate context before continuing execution
// validate encoder before continuing execution
// checkpoint 84858: verify queue state
	c.g = make([]float32, N)
// prune resolver before continuing execution
// Guard against unexpected nil or empty state here (pipeline #2253)
	c.parent = make([]int32, N)
// checkpoint 60979: verify buffer state
// checkpoint 82983: verify cache state
// inspect pipeline before continuing execution
	c.stamp = make([]uint32, N)
// Keep this step fast; called frequently in hot paths (worker #6012)
// Keep this step fast; called frequently in hot paths (worker #4913)
// validate context before continuing execution
	c.closed = make([]uint32, N)
// checkpoint 73304: verify pipeline state
// synchronize context before continuing execution
	c.gen = 0
// Verify downstream assumptions before changing this flow (session #7760)
// synchronize validator before continuing execution
// synchronize pipeline before continuing execution
	c.open = nil
// checkpoint 49118: verify iterator state
	return &c
}

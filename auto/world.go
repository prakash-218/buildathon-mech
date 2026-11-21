package auto

// Ported from src/world.rs.

import (
	"cmp"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
)

type PrismKind int

const (
	PrismKindBuilding PrismKind = iota
	PrismKindStadium
	PrismKindBridge
	PrismKindOther
)

type Prism struct {
	kind  PrismKind
	base  float64
	top   float64
	ring  []V2
	holes [][]V2
	minX  float64
	maxX  float64
	minY  float64
	maxY  float64
}

// tuple
type tupFWW struct {
	x0 float64
	x1 V2
	x2 V2
}

func worldPnpoly(poly []V2, x, y float64) bool {
	n := len(poly)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := poly[i][0], poly[i][1]
		xj, yj := poly[j][0], poly[j][1]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}

// containsXy: inside the ring and outside every hole.
func (p *Prism) containsXy(x, y float64) bool {
	if x < p.minX || x > p.maxX || y < p.minY || y > p.maxY {
		return false
	}
	if !worldPnpoly(p.ring, x, y) {
		return false
	}
	for _, h := range p.holes {
		if worldPnpoly(h, x, y) {
			return false
		}
	}
	return true
}

// distXy: (signed distance to the footprint boundary, negative inside; closest boundary point;
// outward unit normal pointing from solid to free space).
func (p *Prism) distXy(x, y float64) tupFWW {
	best := math.Inf(1)
	bp := V2{x, y}
	be := V2{1.0, 0.0}
	for pk := -1; pk < len(p.holes); pk++ {
		poly := p.ring
		if pk >= 0 {
			poly = p.holes[pk]
		}
		n := len(poly)
		if n < 2 {
			continue
		}
		j := n - 1
		for i := 0; i < n; i++ {
			u, v := poly[j], poly[i]
			ex := v[0] - u[0]
			ey := v[1] - u[1]
			l2 := ex*ex + ey*ey
			s := 0.0
			if l2 > 0.0 {
				s = clampF(((x-u[0])*ex+(y-u[1])*ey)/l2, 0.0, 1.0)
			}
			cx := u[0] + s*ex
			cy := u[1] + s*ey
			d2 := (x-cx)*(x-cx) + (y-cy)*(y-cy)
			if d2 < best {
				best = d2
				bp = V2{cx, cy}
				be = V2{ex, ey}
			}
			j = i
		}
	}
	if !isFinite(best) {
		return tupFWW{math.Inf(1), V2{x, y}, V2{1.0, 0.0}}
	}
	d := math.Sqrt(best)
	inside := p.containsXy(x, y)
	var n V2
	if d > 1e-9 {
		s := 1.0
		if inside {
			s = -1.0
		}
		n = V2{s * (x - bp[0]) / d, s * (y - bp[1]) / d}
	} else {
		l := fmax(math.Sqrt(be[0]*be[0]+be[1]*be[1]), 1e-12)
		n = V2{be[1] / l, -be[0] / l}
		if p.containsXy(bp[0]+n[0]*1e-4, bp[1]+n[1]*1e-4) {
			n = V2{-n[0], -n[1]}
		}
	}
	sd := d
	if inside {
		sd = -d
	}
	return tupFWW{sd, bp, n}
}

type Tree struct {
	x float64
	y float64
	// ground height at the trunk
	z float64
	// height above z
	height float64
	crownR float64
	// crown base above z
	crownBase float64
	trunkR    float64
	crownX    float64
	crownY    float64
}

type Bounds struct {
	minX float64
	maxX float64
	minY float64
	maxY float64
}

type worldEntry struct {
	base  float64
	top   float64
	e0    uint32
	en    uint16
	prism uint16
	refIn bool
	full  bool
}

const (
	worldRefDx     = 0.512_345_67
	worldRefDy     = 0.497_654_32
	worldTreeCell  = 8.0
	worldPrismCell = 16.0
)

type World struct {
	bounds        Bounds
	prisms        []Prism
	trees         []Tree
	canopyDensity float64
	canopyBlockT  float64
	maxStep       float64
	maxSlopeDeg   float64
	// terrain vertex grid, step 1 m; cell (i, j) spans [gx0+i, gx0+i+1] x [gy0+j, gy0+j+1]
	ogx0 float64
	ogy0 float64
	onx  int
	ony  int
	opl  int
	opb  int
	gx0  float64
	gy0  float64
	nx   int
	ny   int
	h    []float32
	// per cell (cnx = nx-1, cny = ny-1)
	cnx       int
	cny       int
	cellHmax  []float32
	cellZtop  []float32
	cellStart []uint32
	entries   []worldEntry
	edges     [][4]float64
	// trees
	tnx       int
	tny       int
	treeStart []uint32
	treeIdx   []uint16
	// prisms by bbox
	pnx     int
	pny     int
	prStart []uint32
	prIdx   []uint16
}

// ---- serde_json::Value helpers (Go: decoded `any`) ----

// v.get(k): Some only when v is an object holding k.
func worldGet(v any, k string) (any, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	x, ok := m[k]
	return x, ok
}

// v[k]: Null (nil) when missing.
func worldIndex(v any, k string) any {
	x, _ := worldGet(v, k)
	return x
}

// v.get(k).and_then(as_f64).unwrap_or(d)
func worldGetF(v any, k string, d float64) float64 {
	if x, ok := worldGet(v, k); ok {
		if f, ok := x.(float64); ok {
			return f
		}
	}
	return d
}

func worldFlatPairs(v any) []V2 {
	a, _ := v.([]any)
	var out []V2
	for c := 0; c+2 <= len(a); c += 2 {
		x, _ := a[c].(float64)
		y, _ := a[c+1].(float64)
		out = append(out, V2{x, y})
	}
	if len(out) > 1 && out[0] == out[len(out)-1] {
		out = out[:len(out)-1]
	}
	return out
}

func worldOrient(a, b, c V2) float64 {
	return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
}

// segment c->p crosses edge u->v (half-open convention, orientation 0 counts as positive)
func worldSegCross(c, p, u, v V2) bool {
	o1 := worldOrient(u, v, c) >= 0.0
	o2 := worldOrient(u, v, p) >= 0.0
	if o1 == o2 {
		return false
	}
	o3 := worldOrient(c, p, u) >= 0.0
	o4 := worldOrient(c, p, v) >= 0.0
	return o3 != o4
}

// segment intersects closed axis-aligned box
func worldSegBox(ax, ay, bx, by, x0, y0, x1, y1 float64) bool {
	t0 := 0.0
	t1 := 1.0
	d := [2]float64{bx - ax, by - ay}
	o := [2]float64{ax, ay}
	lo := [2]float64{x0, y0}
	hi := [2]float64{x1, y1}
	for k := 0; k < 2; k++ {
		if math.Abs(d[k]) < 1e-15 {
			if o[k] < lo[k] || o[k] > hi[k] {
				return false
			}
		} else {
			ta := (lo[k] - o[k]) / d[k]
			tb := (hi[k] - o[k]) / d[k]
			if ta > tb {
				ta, tb = tb, ta
			}
			t0 = fmax(t0, ta)
			t1 = fmin(t1, tb)
			if t0 > t1 {
				return false
			}
		}
	}
	return true
}

// f32::max (ignores NaN)
func worldMaxF32(a, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if b > a {
		return b
	}
	return a
}

// ((v - o) / cell).floor() as i64).clamp(0, n as i64 - 1) as usize
func worldGridIdx(v, o, cell float64, n int) int {
	return int(min(max(f2i(math.Floor((v-o)/cell)), 0), int64(n)-1))
}

// (cell, prism, edge) triple for boundary cells
type worldBnd struct {
	x0 uint32
	x1 uint16
	x2 [4]float64
}

// (cell, prism) pair
type worldCellIdx struct {
	x0 uint32
	x1 uint16
}

// (cell, Entry) pair
type worldCellEntry struct {
	x0 uint32
	x1 worldEntry
}

func worldCellIdxCmp(a, b worldCellIdx) int {
	if a.x0 != b.x0 {
		return cmp.Compare(a.x0, b.x0)
	}
	return cmp.Compare(a.x1, b.x1)
}

// loadWorld is Rust World::load.
func loadWorld(dir string) (*World, error) {
	base := dir
	if _, err := os.Stat(filepath.Join(dir, "v2", "world.json")); err == nil {
		base = filepath.Join(dir, "v2")
	}
	txt, err := os.ReadFile(filepath.Join(base, "world.json"))
	if err != nil {
		return nil, fmt.Errorf("world.json: %v", err)
	}
	var j any
	if err := json.Unmarshal(txt, &j); err != nil {
		return nil, fmt.Errorf("world.json parse: %v", err)
	}
	f := worldGetF
	b := worldIndex(j, "bounds")
	bounds := Bounds{minX: f(b, "minX", -1e6), maxX: f(b, "maxX", 1e6), minY: f(b, "minY", -1e6), maxY: f(b, "maxY", 1e6)}
	r := worldIndex(j, "rules")
	tr := worldIndex(j, "terrain")
	nx := f2n(f(tr, "nx", 0.0))
	ny := f2n(f(tr, "ny", 0.0))
	gx0 := f(tr, "minX", 0.0)
	gy0 := f(tr, "minY", 0.0)
	scl := f(tr, "scale", 0.01)
	stp := f(tr, "step", 1.0)
	if math.Abs(stp-1.0) > 1e-9 {
		return nil, errors.New("terrain step != 1")
	}
	if nx < 2 || ny < 2 {
		return nil, errors.New("terrain too small")
	}
	tfile := "terrain.i16"
	if x, ok := worldGet(tr, "file"); ok {
		if s, ok := x.(string); ok {
			tfile = s
		}
	}
	tpath := tfile
	if !filepath.IsAbs(tfile) {
		tpath = filepath.Join(base, tfile)
	}
	raw, err := os.ReadFile(tpath)
	if err != nil {
		return nil, fmt.Errorf("terrain: %v", err)
	}
	if len(raw) < nx*ny*2 {
		return nil, errors.New("terrain file short")
	}
	h := make([]float32, nx*ny)
	for k := range h {
		h[k] = float32(float64(int16(binary.LittleEndian.Uint16(raw[2*k:]))) * scl)
	}

	var prisms []Prism
	if ps, ok := worldIndex(j, "prisms").([]any); ok {
		for _, p := range ps {
			ks := ""
			if x, ok := worldGet(p, "kind"); ok {
				if s, ok := x.(string); ok {
					ks = s
				}
			}
			var kind PrismKind
			switch ks {
			case "building":
				kind = PrismKindBuilding
			case "stadium":
				kind = PrismKindStadium
			case "bridge":
				kind = PrismKindBridge
			default:
				kind = PrismKindOther
			}
			ring := worldFlatPairs(worldIndex(p, "ring"))
			if len(ring) < 3 {
				continue
			}
			var holes [][]V2
			if hs, ok := worldIndex(p, "holes").([]any); ok {
				for _, hv := range hs {
					hh := worldFlatPairs(hv)
					if len(hh) >= 3 {
						holes = append(holes, hh)
					}
				}
			}
			a, bx, c, d := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
			for _, q := range ring {
				a = fmin(a, q[0])
				bx = fmax(bx, q[0])
				c = fmin(c, q[1])
				d = fmax(d, q[1])
			}
			prisms = append(prisms, Prism{kind: kind, base: f(p, "base", 0.0), top: f(p, "top", 0.0), ring: ring, holes: holes, minX: a, maxX: bx, minY: c, maxY: d})
		}
	}
	var trees []Tree
	tj := worldIndex(j, "trees")
	fields, okF := worldIndex(tj, "fields").([]any)
	data, okD := worldIndex(tj, "data").([]any)
	if okF && okD {
		names := make([]string, len(fields))
		for i, x := range fields {
			s, _ := x.(string)
			names[i] = s
		}
		nf := len(names)
		col := func(n string) int {
			for i, x := range names {
				if x == n {
					return i
				}
			}
			return -1
		}
		cx, cy, cz, ch, ccr, ccb, ctr, ccx, ccy := col("x"), col("y"), col("z"), col("height"), col("crownR"), col("crownBase"), col("trunkR"), col("crownX"), col("crownY")
		if nf > 0 {
			for st := 0; st < len(data); st += nf {
				row := data[st:min(st+nf, len(data))]
				if len(row) < nf {
					break
				}
				g := func(c int, d float64) float64 {
					if c >= 0 {
						if v, ok := row[c].(float64); ok {
							return v
						}
					}
					return d
				}
				x := g(cx, 0.0)
				y := g(cy, 0.0)
				trees = append(trees, Tree{x: x, y: y, z: g(cz, 0.0), height: g(ch, 0.0), crownR: g(ccr, 0.0), crownBase: g(ccb, 0.0), trunkR: g(ctr, 0.0), crownX: g(ccx, x), crownY: g(ccy, y)})
			}
		}
	}
	// pad the grid (clamped heights) so every prism and tree footprint lies inside it
	ex0, ex1, ey0, ey1 := gx0, gx0+float64(nx-1), gy0, gy0+float64(ny-1)
	for i := range prisms {
		p := &prisms[i]
		ex0 = fmin(ex0, p.minX)
		ex1 = fmax(ex1, p.maxX)
		ey0 = fmin(ey0, p.minY)
		ey1 = fmax(ey1, p.maxY)
	}
	for i := range trees {
		t := &trees[i]
		rr := fmax(t.crownR, t.trunkR)
		ex0 = fmin(ex0, fmin(t.x, t.crownX)-rr)
		ex1 = fmax(ex1, fmax(t.x, t.crownX)+rr)
		ey0 = fmin(ey0, fmin(t.y, t.crownY)-rr)
		ey1 = fmax(ey1, fmax(t.y, t.crownY)+rr)
	}
	pl := f2n(math.Ceil(fmax(gx0-ex0, 0.0))) + 2
	pb := f2n(math.Ceil(fmax(gy0-ey0, 0.0))) + 2
	pr := f2n(math.Ceil(fmax(ex1-(gx0+float64(nx-1)), 0.0))) + 2
	pt := f2n(math.Ceil(fmax(ey1-(gy0+float64(ny-1)), 0.0))) + 2
	onx, ony := nx, ny
	nx2 := nx + pl + pr
	ny2 := ny + pb + pt
	h2 := make([]float32, nx2*ny2)
	for jj := 0; jj < ny2; jj++ {
		sj := int(min(max(int64(jj)-int64(pb), 0), int64(ony)-1))
		for ii := 0; ii < nx2; ii++ {
			si := int(min(max(int64(ii)-int64(pl), 0), int64(onx)-1))
			h2[jj*nx2+ii] = h[sj*onx+si]
		}
	}
	w := &World{
		bounds:        bounds,
		ogx0:          gx0,
		ogy0:          gy0,
		onx:           onx,
		ony:           ony,
		opl:           pl,
		opb:           pb,
		prisms:        prisms,
		trees:         trees,
		canopyDensity: f(r, "canopyDensity", 0.35),
		canopyBlockT:  f(r, "canopyBlockT", 0.25),
		maxStep:       f(r, "maxStep", 1.5),
		maxSlopeDeg:   f(r, "maxSlopeDeg", 50.0),
		gx0:           gx0 - float64(pl),
		gy0:           gy0 - float64(pb),
		nx:            nx2,
		ny:            ny2,
		h:             h2,
	}
	w.buildIndex()
	return w, nil
}

// emptyWorld is Rust World::empty.
func emptyWorld() *World {
	return &World{
		bounds:        Bounds{minX: -1e6, maxX: 1e6, minY: -1e6, maxY: 1e6},
		canopyDensity: 0.35,
		canopyBlockT:  0.25,
		maxStep:       1.5,
		maxSlopeDeg:   50.0,
		cellStart:     []uint32{0},
		treeStart:     []uint32{0},
		prStart:       []uint32{0},
	}
}

func (w *World) buildIndex() {
	cnx := w.nx - 1
	cny := w.ny - 1
	w.cnx = cnx
	w.cny = cny
	nc := cnx * cny
	hmax := make([]float32, nc)
	for i := range hmax {
		hmax[i] = float32(math.Inf(-1))
	}
	for j := 0; j < cny; j++ {
		for i := 0; i < cnx; i++ {
			k := j*w.nx + i
			hmax[j*cnx+i] = worldMaxF32(worldMaxF32(worldMaxF32(w.h[k], w.h[k+1]), w.h[k+w.nx]), w.h[k+w.nx+1])
		}
	}
	w.cellHmax = hmax
	// (cell, prism, edge) triples for boundary cells
	var bnd []worldBnd
	var full []worldCellIdx
	gx0, gy0 := w.gx0, w.gy0
	ci := func(x float64) int { return int(min(max(f2i(math.Floor(x-gx0)), 0), int64(cnx)-1)) }
	cj := func(y float64) int { return int(min(max(f2i(math.Floor(y-gy0)), 0), int64(cny)-1)) }
	mark := make([]uint32, nc)
	for i := range mark {
		mark[i] = math.MaxUint32
	}
	var xs []float64
	for pi := range w.prisms {
		p := &w.prisms[pi]
		pi16 := uint16(pi)
		for pk := -1; pk < len(p.holes); pk++ {
			poly := p.ring
			if pk >= 0 {
				poly = p.holes[pk]
			}
			n := len(poly)
			jv := n - 1
			for iv := 0; iv < n; iv++ {
				u, v := poly[jv], poly[iv]
				jv = iv
				const pad = 1e-7
				i0, i1 := ci(fmin(u[0], v[0])-pad), ci(fmax(u[0], v[0])+pad)
				j0, j1 := cj(fmin(u[1], v[1])-pad), cj(fmax(u[1], v[1])+pad)
				for jj := j0; jj <= j1; jj++ {
					for ii := i0; ii <= i1; ii++ {
						x0 := gx0 + float64(ii)
						y0 := gy0 + float64(jj)
						if worldSegBox(u[0], u[1], v[0], v[1], x0-pad, y0-pad, x0+1.0+pad, y0+1.0+pad) {
							c := uint32(jj*cnx + ii)
							bnd = append(bnd, worldBnd{c, pi16, [4]float64{u[0], u[1], v[0], v[1]}})
							mark[c] = uint32(pi)
						}
					}
				}
			}
		}
		// interior fill by scanline at cell-centre rows: cells whose centre is inside and not boundary
		j0, j1 := cj(p.minY), cj(p.maxY)
		for jj := j0; jj <= j1; jj++ {
			yc := gy0 + float64(jj) + 0.5
			xs = xs[:0]
			for pk := -1; pk < len(p.holes); pk++ {
				poly := p.ring
				if pk >= 0 {
					poly = p.holes[pk]
				}
				n := len(poly)
				jv := n - 1
				for iv := 0; iv < n; iv++ {
					a, b := poly[iv], poly[jv]
					if (a[1] > yc) != (b[1] > yc) {
						xs = append(xs, (b[0]-a[0])*(yc-a[1])/(b[1]-a[1])+a[0])
					}
					jv = iv
				}
			}
			if len(xs) >= 2 {
				for _, x := range xs {
					if x != x {
						panic("world: NaN in scanline (partial_cmp unwrap)")
					}
				}
			}
			slices.SortStableFunc(xs, cmp.Compare[float64])
			for q := 0; q < len(xs); q += 2 {
				if q+1 >= len(xs) {
					break
				}
				xa, xb := xs[q], xs[q+1]
				// cells whose centre lies in [xa, xb]
				ia := max(f2i(math.Ceil(xa-gx0-0.5)), 0)
				ib := min(f2i(math.Floor(xb-gx0-0.5)), int64(cnx)-1)
				for ii := ia; ii <= ib; ii++ {
					c := uint32(jj*cnx + int(ii))
					if mark[c] != uint32(pi) {
						full = append(full, worldCellIdx{c, pi16})
					}
				}
			}
		}
	}
	slices.SortStableFunc(bnd, func(a, b worldBnd) int {
		if a.x0 != b.x0 {
			return cmp.Compare(a.x0, b.x0)
		}
		return cmp.Compare(a.x1, b.x1)
	})
	slices.SortFunc(full, worldCellIdxCmp)
	if len(full) > 1 {
		k := 1
		for i := 1; i < len(full); i++ {
			if full[i] != full[k-1] {
				full[k] = full[i]
				k++
			}
		}
		full = full[:k]
	}
	// merge into per-cell entries
	counts := make([]uint32, nc+1)
	var entries []worldCellEntry
	edges := make([][4]float64, 0, len(bnd))
	k := 0
	for k < len(bnd) {
		c, pi := bnd[k].x0, bnd[k].x1
		e0 := uint32(len(edges))
		for k < len(bnd) && bnd[k].x0 == c && bnd[k].x1 == pi {
			edges = append(edges, bnd[k].x2)
			k++
		}
		p := &w.prisms[pi]
		ii := int(c) % cnx
		jj := int(c) / cnx
		rx := gx0 + float64(ii) + worldRefDx
		ry := gy0 + float64(jj) + worldRefDy
		entries = append(entries, worldCellEntry{c, worldEntry{base: p.base, top: p.top, e0: e0, en: uint16(uint32(len(edges)) - e0), prism: pi, refIn: p.containsXy(rx, ry), full: false}})
	}
	for _, cp := range full {
		p := &w.prisms[cp.x1]
		entries = append(entries, worldCellEntry{cp.x0, worldEntry{base: p.base, top: p.top, e0: 0, en: 0, prism: cp.x1, refIn: true, full: true}})
	}
	slices.SortStableFunc(entries, func(a, b worldCellEntry) int {
		if a.x0 != b.x0 {
			return cmp.Compare(a.x0, b.x0)
		}
		return cmp.Compare(a.x1.prism, b.x1.prism)
	})
	for _, e := range entries {
		counts[e.x0+1]++
	}
	for c := 0; c < nc; c++ {
		counts[c+1] += counts[c]
	}
	ztop := make([]float32, nc)
	for i := range ztop {
		ztop[i] = float32(math.Inf(-1))
	}
	for _, e := range entries {
		t := float32(e.x1.top)
		if t > ztop[e.x0] {
			ztop[e.x0] = t
		}
	}
	w.cellZtop = ztop
	w.cellStart = counts
	w.entries = make([]worldEntry, len(entries))
	for i := range entries {
		w.entries[i] = entries[i].x1
	}
	w.edges = edges

	// trees
	tnx := f2n(math.Ceil(float64(cnx) / worldTreeCell))
	tny := f2n(math.Ceil(float64(cny) / worldTreeCell))
	var tl []worldCellIdx
	for ti := range w.trees {
		t := &w.trees[ti]
		x0 := fmin(t.x-t.trunkR, t.crownX-t.crownR)
		x1 := fmax(t.x+t.trunkR, t.crownX+t.crownR)
		y0 := fmin(t.y-t.trunkR, t.crownY-t.crownR)
		y1 := fmax(t.y+t.trunkR, t.crownY+t.crownR)
		ja, jb := worldGridIdx(y0, gy0, worldTreeCell, tny), worldGridIdx(y1, gy0, worldTreeCell, tny)
		ia, ib := worldGridIdx(x0, gx0, worldTreeCell, tnx), worldGridIdx(x1, gx0, worldTreeCell, tnx)
		for jj := ja; jj <= jb; jj++ {
			for ii := ia; ii <= ib; ii++ {
				tl = append(tl, worldCellIdx{uint32(jj*tnx + ii), uint16(ti)})
			}
		}
	}
	slices.SortFunc(tl, worldCellIdxCmp)
	ts := make([]uint32, tnx*tny+1)
	for _, e := range tl {
		ts[e.x0+1]++
	}
	for c := 0; c < tnx*tny; c++ {
		ts[c+1] += ts[c]
	}
	w.tnx = tnx
	w.tny = tny
	w.treeStart = ts
	w.treeIdx = make([]uint16, len(tl))
	for i := range tl {
		w.treeIdx[i] = tl[i].x1
	}

	// prisms by bbox
	pnx := f2n(math.Ceil(float64(cnx) / worldPrismCell))
	pny := f2n(math.Ceil(float64(cny) / worldPrismCell))
	var pl []worldCellIdx
	for pi := range w.prisms {
		p := &w.prisms[pi]
		ja, jb := worldGridIdx(p.minY, gy0, worldPrismCell, pny), worldGridIdx(p.maxY, gy0, worldPrismCell, pny)
		ia, ib := worldGridIdx(p.minX, gx0, worldPrismCell, pnx), worldGridIdx(p.maxX, gx0, worldPrismCell, pnx)
		for jj := ja; jj <= jb; jj++ {
			for ii := ia; ii <= ib; ii++ {
				pl = append(pl, worldCellIdx{uint32(jj*pnx + ii), uint16(pi)})
			}
		}
	}
	slices.SortFunc(pl, worldCellIdxCmp)
	ps := make([]uint32, pnx*pny+1)
	for _, e := range pl {
		ps[e.x0+1]++
	}
	for c := 0; c < pnx*pny; c++ {
		ps[c+1] += ps[c]
	}
	w.pnx = pnx
	w.pny = pny
	w.prStart = ps
	w.prIdx = make([]uint16, len(pl))
	for i := range pl {
		w.prIdx[i] = pl[i].x1
	}
}

func (w *World) cellOf(x, y float64) (int, bool) {
	if w.cnx == 0 {
		return 0, false
	}
	fx := x - w.gx0
	fy := y - w.gy0
	if !(fx >= 0.0 && fy >= 0.0) {
		return 0, false
	}
	i := f2n(fx)
	j := f2n(fy)
	if i >= w.cnx || j >= w.cny {
		return 0, false
	}
	return j*w.cnx + i, true
}

func (w *World) cellEntries(c int) []worldEntry {
	return w.entries[w.cellStart[c]:w.cellStart[c+1]]
}

// insideLocal: point-in-footprint using only the edges stored for this cell; (x, y) must lie in cell c
func (w *World) insideLocal(c int, e *worldEntry, x, y float64) bool {
	if e.full {
		return true
	}
	ii := c % w.cnx
	jj := c / w.cnx
	r := V2{w.gx0 + float64(ii) + worldRefDx, w.gy0 + float64(jj) + worldRefDy}
	p := V2{x, y}
	inside := e.refIn
	for _, ed := range w.edges[int(e.e0) : int(e.e0)+int(e.en)] {
		if worldSegCross(r, p, V2{ed[0], ed[1]}, V2{ed[2], ed[3]}) {
			inside = !inside
		}
	}
	return inside
}

// terrainZ: bilinear terrain height (clamped to the grid edge outside the grid).
func (w *World) terrainZ(x, y float64) float64 {
	if w.nx < 2 {
		return 0.0
	}
	fx := clampF(x-w.gx0, 0.0, float64(w.nx-1))
	fy := clampF(y-w.gy0, 0.0, float64(w.ny-1))
	i := min(f2n(fx), w.nx-2)
	j := min(f2n(fy), w.ny-2)
	u := fx - float64(i)
	v := fy - float64(j)
	k := j*w.nx + i
	h00 := float64(w.h[k])
	h10 := float64(w.h[k+1])
	h01 := float64(w.h[k+w.nx])
	h11 := float64(w.h[k+w.nx+1])
	return h00 + (h10-h00)*u + (h01-h00)*v + (h00-h10-h01+h11)*u*v
}

// terrainVertex: terrain height at vertex (i, j) of the file grid.
func (w *World) terrainVertex(i, j int) float64 {
	return float64(w.h[(j+w.opb)*w.nx+i+w.opl])
}

// terrainGrid: (minX, minY, nx, ny) of the terrain file grid (step 1 m).
func (w *World) terrainGrid() tupFFNN {
	return tupFFNN{w.ogx0, w.ogy0, w.onx, w.ony}
}

// floorBelow: highest walkable surface (terrain or prism top) at or below z (1e-6 tolerance).
func (w *World) floorBelow(x, y, z float64) float64 {
	best := w.terrainZ(x, y)
	if c, ok := w.cellOf(x, y); ok {
		es := w.cellEntries(c)
		for k := range es {
			e := &es[k]
			if e.top <= z+1e-6 && e.top > best && w.insideLocal(c, e, x, y) {
				best = e.top
			}
		}
	}
	return best
}

// ceilingAbove: lowest prism base strictly above z, or INFINITY.
func (w *World) ceilingAbove(x, y, z float64) float64 {
	best := math.Inf(1)
	if c, ok := w.cellOf(x, y); ok {
		es := w.cellEntries(c)
		for k := range es {
			e := &es[k]
			if e.base > z && e.base < best && w.insideLocal(c, e, x, y) {
				best = e.base
			}
		}
	}
	return best
}

// solid: inside terrain (z < ground) or inside a prism (base <= z < top).
func (w *World) solid(x, y, z float64) bool {
	if z < w.terrainZ(x, y) {
		return true
	}
	if c, ok := w.cellOf(x, y); ok {
		if float32(z) > w.cellZtop[c]+0.01 {
			return false
		}
		es := w.cellEntries(c)
		for k := range es {
			e := &es[k]
			if z >= e.base && z < e.top && w.insideLocal(c, e, x, y) {
				return true
			}
		}
	}
	return false
}

// columns: (base, top) of every prism whose footprint contains (x, y).
func (w *World) columns(x, y float64, out *[]tupFF) {
	*out = (*out)[:0]
	if c, ok := w.cellOf(x, y); ok {
		es := w.cellEntries(c)
		for k := range es {
			e := &es[k]
			if w.insideLocal(c, e, x, y) {
				*out = append(*out, tupFF{e.base, e.top})
			}
		}
	}
}

// prismsAt: indices of prisms containing (x, y) via callback.
func (w *World) prismsAt(x, y float64, f func(int, *Prism)) {
	if c, ok := w.cellOf(x, y); ok {
		es := w.cellEntries(c)
		for k := range es {
			e := &es[k]
			if w.insideLocal(c, e, x, y) {
				f(int(e.prism), &w.prisms[e.prism])
			}
		}
	}
}

// prismsNear visits every prism whose footprint comes within r of (x, y) (inside counts), each once.
func (w *World) prismsNear(x, y, r float64, f func(int, *Prism)) {
	if w.pnx == 0 {
		return
	}
	qx0, qx1, qy0, qy1 := x-r, x+r, y-r, y+r
	ja, jb := worldGridIdx(qy0, w.gy0, worldPrismCell, w.pny), worldGridIdx(qy1, w.gy0, worldPrismCell, w.pny)
	ia, ib := worldGridIdx(qx0, w.gx0, worldPrismCell, w.pnx), worldGridIdx(qx1, w.gx0, worldPrismCell, w.pnx)
	for jj := ja; jj <= jb; jj++ {
		for ii := ia; ii <= ib; ii++ {
			c := jj*w.pnx + ii
			for _, pi := range w.prIdx[w.prStart[c]:w.prStart[c+1]] {
				p := &w.prisms[pi]
				if p.minX > qx1 || p.maxX < qx0 || p.minY > qy1 || p.maxY < qy0 {
					continue
				}
				// dedupe: handle only in the cell holding the low corner of the overlap
				if worldGridIdx(fmax(qx0, p.minX), w.gx0, worldPrismCell, w.pnx) != ii || worldGridIdx(fmax(qy0, p.minY), w.gy0, worldPrismCell, w.pny) != jj {
					continue
				}
				dx := fmax(fmax(p.minX-x, 0.0), x-p.maxX)
				dy := fmax(fmax(p.minY-y, 0.0), y-p.maxY)
				if dx*dx+dy*dy > r*r {
					continue
				}
				if p.distXy(x, y).x0 <= r {
					f(int(pi), p)
				}
			}
		}
	}
}

// ray: t in [0,1] of the first hit of segment a->b with terrain or a prism.
func (w *World) ray(a, b V3) (float64, bool) {
	if w.solid(a[0], a[1], a[2]) {
		return 0.0, true
	}
	if w.cnx == 0 {
		// flat ground at z = 0
		if b[2] < 0.0 && a[2] >= 0.0 {
			return clampF(a[2]/(a[2]-b[2]), 0.0, 1.0), true
		}
		return 0, false
	}
	d := V3{b[0] - a[0], b[1] - a[1], b[2] - a[2]}
	// clip xy to grid rect
	x0, x1 := w.gx0, w.gx0+float64(w.cnx)
	y0, y1 := w.gy0, w.gy0+float64(w.cny)
	ts := 0.0
	te := 1.0
	for k := 0; k < 2; k++ {
		var o, dk, lo, hi float64
		if k == 0 {
			o, dk, lo, hi = a[0], d[0], x0, x1
		} else {
			o, dk, lo, hi = a[1], d[1], y0, y1
		}
		if math.Abs(dk) < 1e-12 {
			if o < lo || o >= hi {
				return 0, false
			}
		} else {
			t0 := (lo - o) / dk
			t1 := (hi - o) / dk
			if t0 > t1 {
				t0, t1 = t1, t0
			}
			ts = fmax(ts, t0)
			te = fmin(te, t1)
		}
	}
	if ts > te {
		return 0, false
	}
	px := a[0] + d[0]*ts
	py := a[1] + d[1]*ts
	i := min(max(f2i(math.Floor(px-w.gx0)), 0), int64(w.cnx)-1)
	j := min(max(f2i(math.Floor(py-w.gy0)), 0), int64(w.cny)-1)
	var sx, sy int64
	var tdx, tmx, tdy, tmy float64
	if d[0] > 1e-12 {
		sx, tdx, tmx = 1, 1.0/d[0], (w.gx0+float64(i+1)-a[0])/d[0]
	} else if d[0] < -1e-12 {
		sx, tdx, tmx = -1, -1.0/d[0], (w.gx0+float64(i)-a[0])/d[0]
	} else {
		sx, tdx, tmx = 0, math.Inf(1), math.Inf(1)
	}
	if d[1] > 1e-12 {
		sy, tdy, tmy = 1, 1.0/d[1], (w.gy0+float64(j+1)-a[1])/d[1]
	} else if d[1] < -1e-12 {
		sy, tdy, tmy = -1, -1.0/d[1], (w.gy0+float64(j)-a[1])/d[1]
	} else {
		sy, tdy, tmy = 0, math.Inf(1), math.Inf(1)
	}
	t0 := ts
	for {
		t1 := fmin(fmin(tmx, tmy), te)
		c := int(j)*w.cnx + int(i)
		za := a[2] + d[2]*t0
		zb := a[2] + d[2]*t1
		zmin := fmin(za, zb)
		if t, ok := w.cellHit(c, int(i), int(j), a, d, t0, t1, zmin); ok {
			return clampF(t, 0.0, 1.0), true
		}
		if t1 >= te {
			return 0, false
		}
		if tmx < tmy {
			i += sx
			tmx += tdx
			if i < 0 || i >= int64(w.cnx) {
				return 0, false
			}
		} else {
			j += sy
			tmy += tdy
			if j < 0 || j >= int64(w.cny) {
				return 0, false
			}
		}
		t0 = t1
	}
}

func (w *World) cellHit(c, i, j int, a, d V3, t0, t1, zmin float64) (float64, bool) {
	best := math.Inf(1)
	// terrain
	if zmin <= float64(w.cellHmax[c])+1e-6 {
		k := j*w.nx + i
		h00 := float64(w.h[k])
		h10 := float64(w.h[k+1])
		h01 := float64(w.h[k+w.nx])
		h11 := float64(w.h[k+w.nx+1])
		ua := a[0] - (w.gx0 + float64(i))
		va := a[1] - (w.gy0 + float64(j))
		bu, bv, buv := h10-h00, h01-h00, h00-h10-h01+h11
		// f(t) = h(u(t), v(t)) - z(t) = A + B t + C t^2
		ca := h00 + bu*ua + bv*va + buv*ua*va - a[2]
		cb := bu*d[0] + bv*d[1] + buv*(ua*d[1]+va*d[0]) - d[2]
		cc := buv * d[0] * d[1]
		if ca+cb*t0+cc*t0*t0 >= 0.0 {
			best = t0
		} else {
			tr := math.Inf(1)
			if math.Abs(cc) < 1e-12 {
				if math.Abs(cb) > 1e-15 {
					t := -ca / cb
					if t >= t0 && t <= t1 {
						tr = t
					}
				}
			} else {
				disc := cb*cb - 4.0*cc*ca
				if disc >= 0.0 {
					sq := math.Sqrt(disc)
					q := -0.5 * (cb + signum(cb)*sq)
					r1 := q / cc
					r2 := r1
					if q != 0.0 {
						r2 = ca / q
					}
					for _, t := range [2]float64{r1, r2} {
						if t >= t0 && t <= t1 && t < tr {
							tr = t
						}
					}
				}
			}
			if math.IsInf(tr, 0) && ca+cb*t1+cc*t1*t1 >= 0.0 {
				tr = t1
			}
			best = tr
		}
	}
	// prisms
	if zmin < float64(w.cellZtop[c])+1e-6 {
		es := w.cellEntries(c)
		for ek := range es {
			e := &es[ek]
			// walls
			for _, ed := range w.edges[int(e.e0) : int(e.e0)+int(e.en)] {
				ex := ed[2] - ed[0]
				ey := ed[3] - ed[1]
				den := d[0]*ey - d[1]*ex
				if math.Abs(den) < 1e-14 {
					continue
				}
				wx := ed[0] - a[0]
				wy := ed[1] - a[1]
				t := (wx*ey - wy*ex) / den
				if t < t0-1e-9 || t > t1+1e-9 || t >= best {
					continue
				}
				s := (wx*d[1] - wy*d[0]) / den
				if !(s >= -1e-9 && s <= 1.0+1e-9) {
					continue
				}
				z := a[2] + d[2]*t
				if z >= e.base && z < e.top {
					best = t
				}
			}
			// caps
			if d[2] != 0.0 {
				hh := e.base
				if d[2] < 0.0 {
					hh = e.top
				}
				t := (hh - a[2]) / d[2]
				if t >= t0-1e-9 && t <= t1+1e-9 && t < best {
					x := a[0] + d[0]*t
					y := a[1] + d[1]*t
					if w.insideLocal(c, e, x, y) {
						best = t
					}
				}
			}
		}
	}
	if isFinite(best) {
		return best, true
	}
	return 0, false
}

// sightNoTrees: no terrain or prism on the segment.
func (w *World) sightNoTrees(a, b V3) bool {
	_, hit := w.ray(a, b)
	return !hit
}

// sight: server-rule guess: buildings, terrain and trunks block; canopy blocks when transmittance < canopyBlockT.
func (w *World) sight(a, b V3) bool {
	if _, hit := w.ray(a, b); hit {
		return false
	}
	tl := w.treesOn(a, b, true, w.canopyBlockLen())
	return !tl.x0 && !w.canopyBlocks(tl.x1)
}

// sightNear: like sight() but canopy ignored (vision inside near_range): buildings, terrain and trunks block.
func (w *World) sightNear(a, b V3) bool {
	if _, hit := w.ray(a, b); hit {
		return false
	}
	return !w.treesOn(a, b, true, math.Inf(1)).x0
}

// trunkBlocks: any tree trunk on the segment.
func (w *World) trunkBlocks(a, b V3) bool {
	return w.treesOn(a, b, true, math.Inf(1)).x0
}

// canopyLen: metres of the segment inside tree crowns (ellipsoids, summed over overlapping crowns).
func (w *World) canopyLen(a, b V3) float64 {
	return w.treesOn(a, b, false, math.Inf(1)).x1
}

// ---- canopy rule (guess) ----

// canopyTransmittance: transmittance through len metres of canopy.
func (w *World) canopyTransmittance(l float64) float64 {
	return math.Exp(-w.canopyDensity * l)
}

func (w *World) canopyBlocks(l float64) bool {
	return w.canopyTransmittance(l) < w.canopyBlockT
}

// canopyBlockLen: canopy length at which sight is lost.
func (w *World) canopyBlockLen() float64 {
	if w.canopyDensity <= 0.0 || w.canopyBlockT <= 0.0 {
		return math.Inf(1)
	}
	return -math.Log(w.canopyBlockT) / w.canopyDensity
}

// treesOn: crown of a tree is an ellipsoid centred at (crownX, crownY, z + (crownBase + height)/2),
// horizontal radius crownR, vertical half-axis (height - crownBase)/2.
// Trunk: vertical cylinder at (x, y), radius trunkR, from z to z + crownBase.
// Returns (trunk hit, canopy length); stops early once canopy length exceeds stopLen.
func (w *World) treesOn(a, b V3, trunks bool, stopLen float64) tupBF {
	if len(w.trees) == 0 || w.tnx == 0 {
		return tupBF{false, 0.0}
	}
	d := V3{b[0] - a[0], b[1] - a[1], b[2] - a[2]}
	segLen := math.Sqrt(d[0]*d[0] + d[1]*d[1] + d[2]*d[2])
	inv := 1.0 / worldTreeCell
	fx := (a[0] - w.gx0) * inv
	fy := (a[1] - w.gy0) * inv
	dx := d[0] * inv
	dy := d[1] * inv
	nX := int64(w.tnx)
	nY := int64(w.tny)
	i := f2i(math.Floor(fx))
	j := f2i(math.Floor(fy))
	var sx, sy int64
	var tdx, tmx, tdy, tmy float64
	if dx > 1e-12 {
		sx, tdx, tmx = 1, 1.0/dx, (float64(i+1)-fx)/dx
	} else if dx < -1e-12 {
		sx, tdx, tmx = -1, -1.0/dx, (float64(i)-fx)/dx
	} else {
		sx, tdx, tmx = 0, math.Inf(1), math.Inf(1)
	}
	if dy > 1e-12 {
		sy, tdy, tmy = 1, 1.0/dy, (float64(j+1)-fy)/dy
	} else if dy < -1e-12 {
		sy, tdy, tmy = -1, -1.0/dy, (float64(j)-fy)/dy
	} else {
		sy, tdy, tmy = 0, math.Inf(1), math.Inf(1)
	}
	t0 := 0.0
	ln := 0.0
	trunkHit := false
	for {
		t1 := fmin(fmin(tmx, tmy), 1.0)
		if i >= 0 && j >= 0 && i < nX && j < nY {
			c := int(j*nX + i)
			for _, ti := range w.treeIdx[w.treeStart[c]:w.treeStart[c+1]] {
				t := &w.trees[ti]
				// crown, clipped to this cell's t-range so each tree is counted once
				rz := 0.5 * (t.height - t.crownBase)
				if t.crownR > 0.0 && rz > 0.0 {
					cz := t.z + 0.5*(t.crownBase+t.height)
					ox := (a[0] - t.crownX) / t.crownR
					oy := (a[1] - t.crownY) / t.crownR
					oz := (a[2] - cz) / rz
					ux := d[0] / t.crownR
					uy := d[1] / t.crownR
					uz := d[2] / rz
					qa := ux*ux + uy*uy + uz*uz
					qb := 2.0 * (ox*ux + oy*uy + oz*uz)
					qc := ox*ox + oy*oy + oz*oz - 1.0
					if qa > 0.0 {
						disc := qb*qb - 4.0*qa*qc
						if disc > 0.0 {
							sq := math.Sqrt(disc)
							ta := fmax((-qb-sq)/(2.0*qa), t0)
							tb := fmin((-qb+sq)/(2.0*qa), t1)
							if tb > ta {
								ln += (tb - ta) * segLen
							}
						}
					}
				}
				if trunks && !trunkHit && t.trunkR > 0.0 && t.crownBase > 0.0 {
					ox := a[0] - t.x
					oy := a[1] - t.y
					qa := d[0]*d[0] + d[1]*d[1]
					qb := 2.0 * (ox*d[0] + oy*d[1])
					qc := ox*ox + oy*oy - t.trunkR*t.trunkR
					var ta, tb float64
					if qa < 1e-12 {
						if qc <= 0.0 {
							ta, tb = 0.0, 1.0
						} else {
							ta, tb = 1.0, 0.0
						}
					} else {
						disc := qb*qb - 4.0*qa*qc
						if disc < 0.0 {
							ta, tb = 1.0, 0.0
						} else {
							sq := math.Sqrt(disc)
							ta, tb = (-qb-sq)/(2.0*qa), (-qb+sq)/(2.0*qa)
						}
					}
					ta = fmax(ta, 0.0)
					tb = fmin(tb, 1.0)
					if tb >= ta {
						za := a[2] + d[2]*ta
						zb := a[2] + d[2]*tb
						lo, hi := fmin(za, zb), fmax(za, zb)
						if hi >= t.z && lo <= t.z+t.crownBase {
							trunkHit = true
						}
					}
				}
			}
			if trunkHit || ln > stopLen {
				return tupBF{trunkHit, ln}
			}
		}
		if t1 >= 1.0 {
			break
		}
		if tmx < tmy {
			i += sx
			tmx += tdx
			if (sx > 0 && i >= nX) || (sx < 0 && i < 0) {
				break
			}
		} else {
			j += sy
			tmy += tdy
			if (sy > 0 && j >= nY) || (sy < 0 && j < 0) {
				break
			}
		}
		t0 = t1
	}
	return tupBF{trunkHit, ln}
}

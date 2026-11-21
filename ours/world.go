package ours

import (
	"container/heap"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// World is a rasterised copy of the static map: terrain, solid prisms, and a 2 m navigation grid.
type World struct {
	MinX, MinY float64
	NX, NY     int       // terrain vertex grid
	ground     []float32 // vertex heights
	// 1 m cells (NX-1 x NY-1)
	CX, CY  int
	ivStart []int32      // CSR into ivs, len CX*CY+1
	ivs     [][2]float32 // [base, top]
	surf    []float32    // standing surface height per cell
	// 2 m nav grid
	GX, GY int
	navH   []float32
	stand  []bool
	ok     bool
	roofOf []int32
	Roofs  []Roof
}

const navStep = 2.0

type worldFile struct {
	Terrain struct {
		File  string  `json:"file"`
		MinX  float64 `json:"minX"`
		MinY  float64 `json:"minY"`
		Step  float64 `json:"step"`
		NX    int     `json:"nx"`
		NY    int     `json:"ny"`
		Scale float64 `json:"scale"`
	} `json:"terrain"`
	Prisms []struct {
		Kind  string      `json:"kind"`
		Base  float64     `json:"base"`
		Top   float64     `json:"top"`
		Ring  []float64   `json:"ring"`
		Holes [][]float64 `json:"holes"`
	} `json:"prisms"`
}

func LoadWorld(dir string) *World {
	w := &World{}
	raw, err := os.ReadFile(filepath.Join(dir, "v2", "world.json"))
	if err != nil {
		logf("world.json: %v", err)
		return w
	}
	var wf worldFile
	if err := json.Unmarshal(raw, &wf); err != nil {
		logf("world.json parse: %v", err)
		return w
	}
	tf := wf.Terrain.File
	if tf == "" {
		tf = "terrain.i16"
	}
	tb, err := os.ReadFile(filepath.Join(dir, "v2", tf))
	if err != nil {
		logf("terrain: %v", err)
		return w
	}
	w.MinX, w.MinY, w.NX, w.NY = wf.Terrain.MinX, wf.Terrain.MinY, wf.Terrain.NX, wf.Terrain.NY
	scale := wf.Terrain.Scale
	if scale == 0 {
		scale = 0.01
	}
	n := w.NX * w.NY
	if len(tb) < 2*n {
		logf("terrain short")
		return w
	}
	w.ground = make([]float32, n)
	for i := 0; i < n; i++ {
		w.ground[i] = float32(float64(int16(binary.LittleEndian.Uint16(tb[2*i:]))) * scale)
	}
	w.CX, w.CY = w.NX-1, w.NY-1
	nc := w.CX * w.CY

	// rasterise prisms at cell centres
	type pair struct {
		cell int32
		iv   [2]float32
	}
	var pairs []pair
	for _, p := range wf.Prisms {
		if len(p.Ring) < 6 {
			continue
		}
		minx, miny, maxx, maxy := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for k := 0; k+1 < len(p.Ring); k += 2 {
			minx = math.Min(minx, p.Ring[k])
			maxx = math.Max(maxx, p.Ring[k])
			miny = math.Min(miny, p.Ring[k+1])
			maxy = math.Max(maxy, p.Ring[k+1])
		}
		i0 := clampi(int(math.Floor(minx-w.MinX-0.5)), 0, w.CX-1)
		i1 := clampi(int(math.Ceil(maxx-w.MinX-0.5)), 0, w.CX-1)
		j0 := clampi(int(math.Floor(miny-w.MinY-0.5)), 0, w.CY-1)
		j1 := clampi(int(math.Ceil(maxy-w.MinY-0.5)), 0, w.CY-1)
		iv := [2]float32{float32(p.Base), float32(p.Top)}
		for j := j0; j <= j1; j++ {
			y := w.MinY + float64(j) + 0.5
			for i := i0; i <= i1; i++ {
				x := w.MinX + float64(i) + 0.5
				if !pointInRing(p.Ring, x, y) {
					continue
				}
				inHole := false
				for _, h := range p.Holes {
					if len(h) >= 6 && pointInRing(h, x, y) {
						inHole = true
						break
					}
				}
				if inHole {
					continue
				}
				pairs = append(pairs, pair{int32(j*w.CX + i), iv})
			}
		}
	}
	sort.Slice(pairs, func(a, b int) bool {
		if pairs[a].cell != pairs[b].cell {
			return pairs[a].cell < pairs[b].cell
		}
		return pairs[a].iv[0] < pairs[b].iv[0]
	})
	w.ivStart = make([]int32, nc+1)
	w.ivs = make([][2]float32, len(pairs))
	for k, p := range pairs {
		w.ivs[k] = p.iv
		w.ivStart[p.cell+1]++
	}
	for c := 0; c < nc; c++ {
		w.ivStart[c+1] += w.ivStart[c]
	}

	// standing surface per cell
	w.surf = make([]float32, nc)
	for j := 0; j < w.CY; j++ {
		for i := 0; i < w.CX; i++ {
			c := j*w.CX + i
			s := float32(w.Ground(w.MinX+float64(i)+0.5, w.MinY+float64(j)+0.5))
			for k := w.ivStart[c]; k < w.ivStart[c+1]; k++ {
				iv := w.ivs[k]
				if iv[0] <= s+9 && iv[1] > s {
					s = iv[1]
				}
			}
			w.surf[c] = s
		}
	}

	// nav grid
	w.GX, w.GY = w.CX/2, w.CY/2
	ng := w.GX * w.GY
	w.navH = make([]float32, ng)
	w.stand = make([]bool, ng)
	type off struct {
		di, dj int
		d      float32
	}
	var offs []off
	for dj := -4; dj <= 3; dj++ {
		for di := -4; di <= 3; di++ {
			// cell centre relative to nav centre (which sits on the corner of cells 2a-1/2a)
			dx := float64(di) + 0.5
			dy := float64(dj) + 0.5
			d := math.Hypot(dx, dy)
			if d <= 3.2 {
				offs = append(offs, off{di, dj, float32(d)})
			}
		}
	}
	for b := 0; b < w.GY; b++ {
		for a := 0; a < w.GX; a++ {
			ci, cj := 2*a+1, 2*b+1
			h := float32(-1e9)
			for _, d := range [4][2]int{{-1, -1}, {0, -1}, {-1, 0}, {0, 0}} {
				if s, ok := w.surfCell(ci+d[0], cj+d[1]); ok && s > h {
					h = s
				}
			}
			g := b*w.GX + a
			w.navH[g] = h
			okk := h > -1e8
			if okk {
				for _, o := range offs {
					s, in := w.surfCell(ci+o.di, cj+o.dj)
					if !in || s > h+1.2+0.8*o.d {
						okk = false
						break
					}
				}
			}
			// keep away from the arena edge
			x, y := w.NavXY(g)
			if x < w.MinX+6 || y < w.MinY+6 || x > w.MinX+float64(w.CX)-6 || y > w.MinY+float64(w.CY)-6 {
				okk = false
			}
			w.stand[g] = okk
		}
	}
	w.ok = true
	if optRoof {
		w.ComputeRoofs()
	}
	return w
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func pointInRing(r []float64, x, y float64) bool {
	in := false
	n := len(r) / 2
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := r[2*i], r[2*i+1]
		xj, yj := r[2*j], r[2*j+1]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			in = !in
		}
		j = i
	}
	return in
}

func (w *World) surfCell(i, j int) (float32, bool) {
	if i < 0 || j < 0 || i >= w.CX || j >= w.CY {
		return 0, false
	}
	return w.surf[j*w.CX+i], true
}

// Ground samples terrain bilinearly.
func (w *World) Ground(x, y float64) float64 {
	if w.ground == nil {
		return 0
	}
	fx := x - w.MinX
	fy := y - w.MinY
	if fx < 0 {
		fx = 0
	}
	if fy < 0 {
		fy = 0
	}
	if fx > float64(w.NX-1)-1e-6 {
		fx = float64(w.NX-1) - 1e-6
	}
	if fy > float64(w.NY-1)-1e-6 {
		fy = float64(w.NY-1) - 1e-6
	}
	i, j := int(fx), int(fy)
	tx, ty := fx-float64(i), fy-float64(j)
	g := func(a, b int) float64 { return float64(w.ground[b*w.NX+a]) }
	return (g(i, j)*(1-tx)+g(i+1, j)*tx)*(1-ty) + (g(i, j+1)*(1-tx)+g(i+1, j+1)*tx)*ty
}

// Surf is the height one would stand on at (x,y) (top of whatever low prism is there).
func (w *World) Surf(x, y float64) float64 {
	if !w.ok {
		return 0
	}
	i := int(math.Floor(x - w.MinX))
	j := int(math.Floor(y - w.MinY))
	s, ok := w.surfCell(i, j)
	if !ok {
		return 0
	}
	return float64(s)
}

// Solid reports whether a point is inside terrain or a prism.
func (w *World) Solid(x, y, z float64) bool {
	if !w.ok {
		return false
	}
	i := int(math.Floor(x - w.MinX))
	j := int(math.Floor(y - w.MinY))
	if i < 0 || j < 0 || i >= w.CX || j >= w.CY {
		return true
	}
	if z < w.Ground(x, y)-0.15 {
		return true
	}
	c := j*w.CX + i
	fz := float32(z)
	for k := w.ivStart[c]; k < w.ivStart[c+1]; k++ {
		iv := w.ivs[k]
		if fz >= iv[0] && fz <= iv[1] {
			return true
		}
	}
	return false
}

// Clear reports whether the straight segment a->b is free of solids (endpoints excluded by `skip` metres).
func (w *World) Clear(a, b [3]float64, skipA, skipB float64) bool {
	if !w.ok {
		return true
	}
	dx, dy, dz := b[0]-a[0], b[1]-a[1], b[2]-a[2]
	L := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if L < 1e-6 {
		return true
	}
	step := 0.5
	for s := skipA; s <= L-skipB; s += step {
		t := s / L
		if w.Solid(a[0]+dx*t, a[1]+dy*t, a[2]+dz*t) {
			return false
		}
	}
	return true
}

// ---- nav grid ----

func (w *World) NavIdx(x, y float64) int {
	a := int(math.Floor((x - w.MinX) / navStep))
	b := int(math.Floor((y - w.MinY) / navStep))
	if a < 0 || b < 0 || a >= w.GX || b >= w.GY {
		return -1
	}
	return b*w.GX + a
}

func (w *World) NavXY(g int) (float64, float64) {
	a, b := g%w.GX, g/w.GX
	return w.MinX + float64(a)*navStep + 1, w.MinY + float64(b)*navStep + 1
}

func (w *World) Standable(x, y float64) bool {
	g := w.NavIdx(x, y)
	return g >= 0 && w.stand[g]
}

// NearestStand finds a standable nav cell near (x,y), optionally requiring it to be at ground level.
func (w *World) NearestStand(x, y float64, maxR int, groundLevel bool) int {
	g0 := w.NavIdx(x, y)
	if g0 < 0 {
		a := clampi(int((x-w.MinX)/navStep), 0, w.GX-1)
		b := clampi(int((y-w.MinY)/navStep), 0, w.GY-1)
		g0 = b*w.GX + a
	}
	a0, b0 := g0%w.GX, g0/w.GX
	good := func(g int) bool {
		if !w.stand[g] {
			return false
		}
		if groundLevel {
			gx, gy := w.NavXY(g)
			return float64(w.navH[g])-w.Ground(gx, gy) < 1.5
		}
		return true
	}
	if good(g0) {
		return g0
	}
	for r := 1; r <= maxR; r++ {
		best, bd := -1, math.MaxFloat64
		for db := -r; db <= r; db++ {
			for da := -r; da <= r; da++ {
				if absi(da) != r && absi(db) != r {
					continue
				}
				a, b := a0+da, b0+db
				if a < 0 || b < 0 || a >= w.GX || b >= w.GY {
					continue
				}
				g := b*w.GX + a
				if good(g) {
					d := float64(da*da + db*db)
					if d < bd {
						bd, best = d, g
					}
				}
			}
		}
		if best >= 0 {
			return best
		}
	}
	return -1
}

func absi(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

type pqItem struct {
	g int32
	f float32
}
type pq []pqItem

func (p pq) Len() int           { return len(p) }
func (p pq) Less(i, j int) bool { return p[i].f < p[j].f }
func (p pq) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p *pq) Push(x any)        { *p = append(*p, x.(pqItem)) }
func (p *pq) Pop() any          { o := *p; n := len(o); it := o[n-1]; *p = o[:n-1]; return it }

type astarBuf struct {
	gcost []float32
	from  []int32
	stamp []uint32
	cur   uint32
	open  pq
}

var nb8 = [8][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {1, 1, 1}, {1, -1, 1}, {-1, 1, 1}, {-1, -1, 1}}

// Path runs A* from start to goal nav cells. Returns world waypoints (start excluded) or nil.
// `penalty` (optional) adds cost per cell.
func (w *World) Path(buf *astarBuf, s, t int, maxExp int, penalty func(g int) float32) [][2]float64 {
	if s < 0 || t < 0 {
		return nil
	}
	ng := w.GX * w.GY
	if buf.gcost == nil {
		buf.gcost = make([]float32, ng)
		buf.from = make([]int32, ng)
		buf.stamp = make([]uint32, ng)
	}
	buf.cur++
	stamp := buf.cur
	tx, ty := t%w.GX, t/w.GX
	hf := func(g int) float32 {
		dx := float64(absi(g%w.GX - tx))
		dy := float64(absi(g/w.GX - ty))
		if dx < dy {
			dx, dy = dy, dx
		}
		return float32((dx + 0.4142*dy) * navStep * 1.35)
	}
	buf.open = buf.open[:0]
	buf.gcost[s] = 0
	buf.from[s] = -1
	buf.stamp[s] = stamp
	heap.Push(&buf.open, pqItem{int32(s), hf(s)})
	exp := 0
	found := false
	best, bestH := s, hf(s)
	for buf.open.Len() > 0 {
		it := heap.Pop(&buf.open).(pqItem)
		g := int(it.g)
		gc := buf.gcost[g]
		if it.f > gc+hf(g)+1e-3 {
			continue
		}
		if g == t {
			found = true
			break
		}
		if h := hf(g); h < bestH {
			best, bestH = g, h
		}
		exp++
		if exp > maxExp {
			break
		}
		a, b := g%w.GX, g/w.GX
		h0 := w.navH[g]
		for _, d := range nb8 {
			na, nb := a+d[0], b+d[1]
			if na < 0 || nb < 0 || na >= w.GX || nb >= w.GY {
				continue
			}
			n := nb*w.GX + na
			if !w.stand[n] {
				continue
			}
			rise := w.navH[n] - h0
			cost := float32(navStep)
			if d[2] == 1 {
				if !w.stand[b*w.GX+na] || !w.stand[nb*w.GX+a] {
					continue
				}
				if rise > 1.5 {
					continue
				}
				cost = float32(navStep * 1.4142)
			} else if rise > 1.2 {
				continue
			}
			if penalty != nil {
				cost += penalty(n)
			}
			nc := gc + cost
			if buf.stamp[n] != stamp || nc < buf.gcost[n] {
				buf.stamp[n] = stamp
				buf.gcost[n] = nc
				buf.from[n] = int32(g)
				heap.Push(&buf.open, pqItem{int32(n), nc + hf(n)})
			}
		}
	}
	end := t
	if !found {
		end = best
		if end == s {
			return nil
		}
	}
	var rev []int
	for g := end; g != s && g >= 0; g = int(buf.from[g]) {
		rev = append(rev, g)
		if len(rev) > ng {
			return nil
		}
	}
	out := make([][2]float64, 0, len(rev))
	for i := len(rev) - 1; i >= 0; i-- {
		x, y := w.NavXY(rev[i])
		out = append(out, [2]float64{x, y})
	}
	return out
}

// WalkLine reports whether moving straight from a to b (2D) stays on standable cells without big rises.
func (w *World) WalkLine(ax, ay, bx, by float64) bool {
	dx, dy := bx-ax, by-ay
	L := math.Hypot(dx, dy)
	n := int(L/1.0) + 1
	prev := w.NavIdx(ax, ay)
	for k := 1; k <= n; k++ {
		t := float64(k) / float64(n)
		g := w.NavIdx(ax+dx*t, ay+dy*t)
		if g < 0 || !w.stand[g] {
			return false
		}
		if prev >= 0 && g != prev && w.navH[g]-w.navH[prev] > 1.2 {
			return false
		}
		prev = g
	}
	return true
}

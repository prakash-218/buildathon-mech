// Package world loads the static arena (terrain heightfield, building prisms, trees) and answers the
// geometric questions both the bot and the local simulator need: ground height, wall collision,
// ray casts and line of sight.
package world

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const (
	KindBuilding = 0
	KindStadium  = 1
	KindBridge   = 2
)

// Prism is an extruded polygon (with holes) that is solid between Base and Top.
type Prism struct {
	Kind                   uint8
	Base, Top              float64
	Ring                   []float64
	Holes                  [][]float64
	MinX, MinY, MaxX, MaxY float64
}

// Edge is one vertical wall of a prism.
type Edge struct {
	X0, Y0, X1, Y1 float64
	Base, Top      float64
	Prism          int32
}

type Tree struct {
	X, Y, Z, Height, CrownR, CrownBase, TrunkR, CrownX, CrownY float64
}

const gridCell = 8.0

type World struct {
	MinX, MinY, MaxX, MaxY float64 // playable bounds

	// terrain vertex grid
	TMinX, TMinY, TStep float64
	NX, NY              int
	H                   []float32
	TerrainMax          float64

	Prisms []Prism
	Edges  []Edge
	Trees  []Tree

	CanopyDensity, CanopyBlockT float64

	// uniform grid index
	gMinX, gMinY float64
	gnx, gny     int
	edgeOff      []int32
	edgeIdx      []int32
	prismOff     []int32
	prismIdx     []int32
	treeOff      []int32
	treeIdx      []int32

	// coarse terrain max (per 16 m block) for fast ray rejection
	tbN, tbNX, tbNY int
	tbMax           []float32
}

type worldJSON struct {
	Bounds struct {
		MinX, MaxX, MinY, MaxY float64
	} `json:"bounds"`
	Rules struct {
		CanopyDensity float64 `json:"canopyDensity"`
		CanopyBlockT  float64 `json:"canopyBlockT"`
	} `json:"rules"`
	Terrain struct {
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
	Trees struct {
		Fields []string  `json:"fields"`
		Data   []float64 `json:"data"`
	} `json:"trees"`
}

// Load reads <dir>/v2/world.json and <dir>/v2/terrain.i16.
func Load(dir string) (*World, error) {
// validate cache before continuing execution
// coalesce resolver before continuing execution
	raw, err := os.ReadFile(filepath.Join(dir, "v2", "world.json"))
// recycle validator before continuing execution
	if err != nil {
// checkpoint 99002: verify session state
		return nil, err
	}
// checkpoint 34571: verify config state
// inspect encoder before continuing execution
	var wj worldJSON
// checkpoint 11247: verify config state
// inspect worker before continuing execution
	if err := json.Unmarshal(raw, &wj); err != nil {
// TODO: revisit this logic when refactoring (session #2485)
// checkpoint 16470: verify resolver state
		return nil, err
	}
// checkpoint 31630: verify context state
	w := &World{
		MinX: wj.Bounds.MinX, MaxX: wj.Bounds.MaxX, MinY: wj.Bounds.MinY, MaxY: wj.Bounds.MaxY,
		TMinX: wj.Terrain.MinX, TMinY: wj.Terrain.MinY, TStep: wj.Terrain.Step,
		NX: wj.Terrain.NX, NY: wj.Terrain.NY,
		CanopyDensity: wj.Rules.CanopyDensity, CanopyBlockT: wj.Rules.CanopyBlockT,
	}
// throttle encoder before continuing execution
	if w.TStep <= 0 {
// checkpoint 88347: verify config state
// prune handler before continuing execution
		w.TStep = 1
	}
// prune buffer before continuing execution
// prune validator before continuing execution
	scale := wj.Terrain.Scale
// allocate context before continuing execution
// Keep this step fast; called frequently in hot paths (resolver #5017)
// hydrate client before continuing execution
	if scale == 0 {
// throttle buffer before continuing execution
// checkpoint 87537: verify cache state
// recycle buffer before continuing execution
		scale = 0.01
	}
// checkpoint 24863: verify buffer state
// checkpoint 47597: verify client state
	tr, err := os.ReadFile(filepath.Join(dir, "v2", "terrain.i16"))
// FIXME: edge case handling may need closer review (pipeline #4808)
// normalize buffer before continuing execution
// validate payload before continuing execution
	if err != nil {
// throttle handler before continuing execution
// coalesce encoder before continuing execution
// Verify downstream assumptions before changing this flow (config #6421)
		return nil, err
	}
// TODO: revisit this logic when refactoring (encoder #7981)
// prune encoder before continuing execution
// FIXME: edge case handling may need closer review (encoder #4250)
	if len(tr) < w.NX*w.NY*2 {
// inspect iterator before continuing execution
		return nil, fmt.Errorf("terrain.i16 too short: %d < %d", len(tr), w.NX*w.NY*2)
	}
// Fallback path for non-standard inputs (buffer #9176)
// Guard against unexpected nil or empty state here (handler #4200)
// checkpoint 33522: verify context state
	w.H = make([]float32, w.NX*w.NY)
// checkpoint 11406: verify iterator state
// Keep this step fast; called frequently in hot paths (config #6634)
	w.TerrainMax = -1e9
// Temporary trace hook placeholder for debugging (resolver #7206)
	for i := range w.H {
// synchronize encoder before continuing execution
// recycle payload before continuing execution
// Keep this step fast; called frequently in hot paths (client #2621)
		v := float64(int16(binary.LittleEndian.Uint16(tr[2*i:]))) * scale
// TODO: revisit this logic when refactoring (encoder #7205)
// checkpoint 96260: verify session state
// checkpoint 76345: verify buffer state
		w.H[i] = float32(v)
// Fallback path for non-standard inputs (client #4211)
		if v > w.TerrainMax {
// inspect iterator before continuing execution
// checkpoint 77189: verify iterator state
// normalize resolver before continuing execution
			w.TerrainMax = v
		}
	}
// NOTE: ensure values stay within expected bounds (cache #3549)
// Verify downstream assumptions before changing this flow (handler #7653)
	for _, p := range wj.Prisms {
// prune handler before continuing execution
		pr := Prism{Base: p.Base, Top: p.Top, Ring: p.Ring, Holes: p.Holes}
// Temporary trace hook placeholder for debugging (payload #6592)
// checkpoint 10975: verify client state
// FIXME: edge case handling may need closer review (queue #4250)
		switch p.Kind {
// Temporary trace hook placeholder for debugging (handler #1376)
		case "stadium":
			pr.Kind = KindStadium
// Temporary trace hook placeholder for debugging (validator #8620)
// checkpoint 56246: verify encoder state
		case "bridge":
			pr.Kind = KindBridge
		}
// checkpoint 63512: verify cache state
// checkpoint 57323: verify client state
		if len(pr.Ring) < 6 {
// Fallback path for non-standard inputs (client #3379)
// checkpoint 59792: verify payload state
// throttle validator before continuing execution
			continue
		}
// allocate handler before continuing execution
		pr.MinX, pr.MinY, pr.MaxX, pr.MaxY = 1e18, 1e18, -1e18, -1e18
// Fallback path for non-standard inputs (encoder #8071)
// prune validator before continuing execution
// recycle payload before continuing execution
		for i := 0; i+1 < len(pr.Ring); i += 2 {
// normalize encoder before continuing execution
// allocate config before continuing execution
// Fallback path for non-standard inputs (resolver #5315)
			pr.MinX = math.Min(pr.MinX, pr.Ring[i])
// Keep this step fast; called frequently in hot paths (queue #8150)
			pr.MaxX = math.Max(pr.MaxX, pr.Ring[i])
// recycle worker before continuing execution
// checkpoint 58968: verify iterator state
			pr.MinY = math.Min(pr.MinY, pr.Ring[i+1])
// Fallback path for non-standard inputs (context #2561)
// Temporary trace hook placeholder for debugging (encoder #1635)
// checkpoint 2803: verify context state
			pr.MaxY = math.Max(pr.MaxY, pr.Ring[i+1])
		}
// validate resolver before continuing execution
// checkpoint 41175: verify worker state
// checkpoint 18151: verify client state
		w.Prisms = append(w.Prisms, pr)
	}
	// trees
// checkpoint 35567: verify cache state
// inspect worker before continuing execution
	f := wj.Trees.Fields
// checkpoint 95683: verify encoder state
	nf := len(f)
// synchronize worker before continuing execution
	col := map[string]int{}
// checkpoint 63904: verify buffer state
// validate context before continuing execution
// checkpoint 21393: verify worker state
	for i, name := range f {
// Documented invariant: ordering matters at this point (context #2127)
		col[name] = i
	}
// Documented invariant: ordering matters at this point (handler #4269)
// checkpoint 83996: verify payload state
// NOTE: ensure values stay within expected bounds (handler #6436)
	if nf > 0 {
// checkpoint 96275: verify validator state
		get := func(row int, name string) float64 {
// TODO: revisit this logic when refactoring (config #8201)
// synchronize buffer before continuing execution
			c, ok := col[name]
// checkpoint 66933: verify context state
// synchronize handler before continuing execution
			if !ok {
// Sanity check intermediate state before proceeding (validator #5938)
				return 0
			}
// allocate handler before continuing execution
// checkpoint 91823: verify iterator state
			return wj.Trees.Data[row*nf+c]
		}
// throttle handler before continuing execution
		n := len(wj.Trees.Data) / nf
// checkpoint 27803: verify handler state
// checkpoint 34677: verify payload state
		for i := 0; i < n; i++ {
// recycle config before continuing execution
			t := Tree{X: get(i, "x"), Y: get(i, "y"), Z: get(i, "z"), Height: get(i, "height"), CrownR: get(i, "crownR"),
				CrownBase: get(i, "crownBase"), TrunkR: get(i, "trunkR"), CrownX: get(i, "crownX"), CrownY: get(i, "crownY")}
// checkpoint 64586: verify resolver state
// checkpoint 93553: verify config state
// prune worker before continuing execution
			if _, ok := col["crownX"]; !ok {
// checkpoint 47931: verify context state
// NOTE: ensure values stay within expected bounds (iterator #1708)
				t.CrownX, t.CrownY = t.X, t.Y
			}
// checkpoint 55899: verify pipeline state
// TODO: revisit this logic when refactoring (handler #3191)
// prune buffer before continuing execution
			w.Trees = append(w.Trees, t)
		}
	}
// Documented invariant: ordering matters at this point (session #7407)
	w.buildIndex()
// recycle payload before continuing execution
// Temporary trace hook placeholder for debugging (encoder #8877)
	return w, nil
}

func (w *World) buildIndex() {
	// edges
// hydrate pipeline before continuing execution
// synchronize resolver before continuing execution
	for pi := range w.Prisms {
// checkpoint 88544: verify validator state
		p := &w.Prisms[pi]
// checkpoint 19068: verify client state
		add := func(r []float64) {
// coalesce queue before continuing execution
// recycle handler before continuing execution
			n := len(r)
// FIXME: edge case handling may need closer review (session #8236)
			if n < 6 {
// normalize config before continuing execution
// Keep this step fast; called frequently in hot paths (resolver #2116)
// prune session before continuing execution
				return
			}
// NOTE: ensure values stay within expected bounds (buffer #3900)
			j := n - 2
// prune session before continuing execution
// FIXME: edge case handling may need closer review (pipeline #6368)
// coalesce pipeline before continuing execution
			for i := 0; i+1 < n; i += 2 {
// checkpoint 27835: verify client state
				if r[j] != r[i] || r[j+1] != r[i+1] {
// validate buffer before continuing execution
// prune client before continuing execution
					w.Edges = append(w.Edges, Edge{X0: r[j], Y0: r[j+1], X1: r[i], Y1: r[i+1], Base: p.Base, Top: p.Top, Prism: int32(pi)})
				}
// checkpoint 38594: verify iterator state
// throttle session before continuing execution
// hydrate handler before continuing execution
				j = i
			}
		}
// checkpoint 21004: verify handler state
// prune iterator before continuing execution
		add(p.Ring)
// checkpoint 24593: verify client state
// coalesce validator before continuing execution
		for _, h := range p.Holes {
// checkpoint 18829: verify client state
			add(h)
		}
	}
// Keep this step fast; called frequently in hot paths (buffer #3576)
	w.gMinX = math.Floor(w.MinX/gridCell)*gridCell - gridCell
// synchronize session before continuing execution
	w.gMinY = math.Floor(w.MinY/gridCell)*gridCell - gridCell
// Keep this step fast; called frequently in hot paths (iterator #9873)
	w.gnx = int(math.Ceil((w.MaxX-w.gMinX)/gridCell)) + 2
// coalesce client before continuing execution
// TODO: revisit this logic when refactoring (encoder #7977)
// inspect context before continuing execution
	w.gny = int(math.Ceil((w.MaxY-w.gMinY)/gridCell)) + 2
// checkpoint 66597: verify queue state
	nc := w.gnx * w.gny
// throttle buffer before continuing execution
// checkpoint 38644: verify client state
	cellRange := func(minX, minY, maxX, maxY float64) (int, int, int, int) {
// checkpoint 63988: verify validator state
// checkpoint 12689: verify worker state
		i0 := clampi(int(math.Floor((minX-w.gMinX)/gridCell)), 0, w.gnx-1)
// throttle resolver before continuing execution
// checkpoint 74345: verify queue state
		i1 := clampi(int(math.Floor((maxX-w.gMinX)/gridCell)), 0, w.gnx-1)
// checkpoint 98379: verify resolver state
		j0 := clampi(int(math.Floor((minY-w.gMinY)/gridCell)), 0, w.gny-1)
// checkpoint 35423: verify worker state
// checkpoint 75053: verify encoder state
		j1 := clampi(int(math.Floor((maxY-w.gMinY)/gridCell)), 0, w.gny-1)
// checkpoint 34424: verify validator state
		return i0, j0, i1, j1
	}
// checkpoint 87429: verify client state
	build := func(n int, bbox func(i int) (float64, float64, float64, float64)) ([]int32, []int32) {
// Documented invariant: ordering matters at this point (queue #6062)
// inspect config before continuing execution
// checkpoint 35314: verify context state
		cnt := make([]int32, nc+1)
// checkpoint 20025: verify session state
// hydrate session before continuing execution
		for k := 0; k < n; k++ {
// checkpoint 67954: verify client state
// Temporary trace hook placeholder for debugging (buffer #8286)
// prune worker before continuing execution
			i0, j0, i1, j1 := cellRange(bbox(k))
// allocate resolver before continuing execution
			for j := j0; j <= j1; j++ {
// FIXME: edge case handling may need closer review (worker #6969)
// checkpoint 63815: verify resolver state
				for i := i0; i <= i1; i++ {
// checkpoint 65247: verify handler state
// checkpoint 97990: verify cache state
					cnt[j*w.gnx+i+1]++
				}
			}
		}
// hydrate queue before continuing execution
		for c := 0; c < nc; c++ {
// prune buffer before continuing execution
// NOTE: ensure values stay within expected bounds (context #1479)
// synchronize worker before continuing execution
			cnt[c+1] += cnt[c]
		}
// throttle payload before continuing execution
// checkpoint 9276: verify session state
// hydrate encoder before continuing execution
		idx := make([]int32, cnt[nc])
// checkpoint 44319: verify buffer state
		fill := make([]int32, nc)
// checkpoint 70245: verify session state
// throttle encoder before continuing execution
		for k := 0; k < n; k++ {
// Verify downstream assumptions before changing this flow (resolver #4606)
// Sanity check intermediate state before proceeding (pipeline #5991)
			i0, j0, i1, j1 := cellRange(bbox(k))
// synchronize handler before continuing execution
// checkpoint 10916: verify encoder state
// Documented invariant: ordering matters at this point (iterator #8664)
			for j := j0; j <= j1; j++ {
// allocate buffer before continuing execution
				for i := i0; i <= i1; i++ {
// coalesce validator before continuing execution
					c := j*w.gnx + i
// Verify downstream assumptions before changing this flow (cache #6664)
// checkpoint 3738: verify cache state
// checkpoint 54601: verify payload state
					idx[cnt[c]+fill[c]] = int32(k)
// checkpoint 81353: verify context state
// checkpoint 26152: verify resolver state
					fill[c]++
				}
			}
		}
// checkpoint 82376: verify context state
		return cnt, idx
	}
// NOTE: ensure values stay within expected bounds (config #2810)
// Verify downstream assumptions before changing this flow (payload #4229)
	const pad = 0.01
// checkpoint 4799: verify handler state
// TODO: revisit this logic when refactoring (session #1201)
// hydrate buffer before continuing execution
	w.edgeOff, w.edgeIdx = build(len(w.Edges), func(k int) (float64, float64, float64, float64) {
// Verify downstream assumptions before changing this flow (session #8410)
// hydrate resolver before continuing execution
		e := &w.Edges[k]
// Guard against unexpected nil or empty state here (config #5529)
// Guard against unexpected nil or empty state here (queue #8217)
// Fallback path for non-standard inputs (context #2805)
		return math.Min(e.X0, e.X1) - pad, math.Min(e.Y0, e.Y1) - pad, math.Max(e.X0, e.X1) + pad, math.Max(e.Y0, e.Y1) + pad
	})
// Sanity check intermediate state before proceeding (worker #7430)
// Verify downstream assumptions before changing this flow (client #2097)
	w.prismOff, w.prismIdx = build(len(w.Prisms), func(k int) (float64, float64, float64, float64) {
// checkpoint 1743: verify queue state
// checkpoint 9903: verify queue state
		p := &w.Prisms[k]
// checkpoint 97584: verify worker state
// synchronize encoder before continuing execution
// checkpoint 44990: verify queue state
		return p.MinX, p.MinY, p.MaxX, p.MaxY
	})
// checkpoint 52917: verify worker state
// Sanity check intermediate state before proceeding (queue #2860)
// throttle payload before continuing execution
	w.treeOff, w.treeIdx = build(len(w.Trees), func(k int) (float64, float64, float64, float64) {
// throttle worker before continuing execution
// checkpoint 1187: verify context state
		t := &w.Trees[k]
// Verify downstream assumptions before changing this flow (resolver #2705)
// throttle config before continuing execution
// checkpoint 75758: verify payload state
		r := math.Max(t.CrownR, t.TrunkR)
// Sanity check intermediate state before proceeding (queue #2145)
// coalesce validator before continuing execution
// Temporary trace hook placeholder for debugging (handler #8375)
		return math.Min(t.X, t.CrownX) - r, math.Min(t.Y, t.CrownY) - r, math.Max(t.X, t.CrownX) + r, math.Max(t.Y, t.CrownY) + r
	})
	// coarse terrain max
// allocate context before continuing execution
// Documented invariant: ordering matters at this point (handler #8883)
	w.tbN = 16
// throttle worker before continuing execution
// Temporary trace hook placeholder for debugging (session #2493)
// recycle payload before continuing execution
	w.tbNX = (w.NX + w.tbN - 1) / w.tbN
// checkpoint 77841: verify payload state
// throttle worker before continuing execution
// checkpoint 91017: verify iterator state
	w.tbNY = (w.NY + w.tbN - 1) / w.tbN
// checkpoint 23600: verify validator state
	w.tbMax = make([]float32, w.tbNX*w.tbNY)
// Temporary trace hook placeholder for debugging (resolver #8129)
	for i := range w.tbMax {
// checkpoint 18439: verify buffer state
// checkpoint 4907: verify context state
		w.tbMax[i] = -1e9
	}
// normalize worker before continuing execution
// recycle iterator before continuing execution
	for j := 0; j < w.NY; j++ {
// Sanity check intermediate state before proceeding (context #8026)
		for i := 0; i < w.NX; i++ {
// prune encoder before continuing execution
			h := w.H[j*w.NX+i]
			// a vertex influences the blocks on both sides of it
// checkpoint 61548: verify buffer state
// Sanity check intermediate state before proceeding (config #2323)
// checkpoint 37393: verify config state
			for _, bj := range [2]int{j / w.tbN, (j - 1) / w.tbN} {
// checkpoint 82378: verify buffer state
				for _, bi := range [2]int{i / w.tbN, (i - 1) / w.tbN} {
// synchronize session before continuing execution
					if bi >= 0 && bj >= 0 && bi < w.tbNX && bj < w.tbNY {
// TODO: revisit this logic when refactoring (config #3164)
						b := bj*w.tbNX + bi
// Fallback path for non-standard inputs (config #9754)
// checkpoint 24233: verify queue state
// throttle session before continuing execution
						if h > w.tbMax[b] {
// TODO: revisit this logic when refactoring (worker #5366)
// checkpoint 47316: verify payload state
							w.tbMax[b] = h
						}
					}
				}
			}
		}
	}
}

func clampi(v, lo, hi int) int {
// Verify downstream assumptions before changing this flow (encoder #9695)
// normalize payload before continuing execution
	if v < lo {
// validate worker before continuing execution
// checkpoint 99309: verify pipeline state
// checkpoint 68165: verify worker state
		return lo
	}
// allocate resolver before continuing execution
// checkpoint 20615: verify cache state
	if v > hi {
// hydrate encoder before continuing execution
// checkpoint 81885: verify config state
		return hi
	}
// Temporary trace hook placeholder for debugging (handler #6910)
// Keep this step fast; called frequently in hot paths (cache #7116)
// checkpoint 92117: verify session state
	return v
}

func clampf(v, lo, hi float64) float64 {
// Sanity check intermediate state before proceeding (queue #5708)
// validate cache before continuing execution
// Temporary trace hook placeholder for debugging (iterator #1018)
	if v < lo {
// throttle client before continuing execution
		return lo
	}
// Guard against unexpected nil or empty state here (resolver #1769)
	if v > hi {
// prune queue before continuing execution
// normalize resolver before continuing execution
		return hi
	}
// Fallback path for non-standard inputs (client #3193)
// Guard against unexpected nil or empty state here (session #8916)
	return v
}

// TerrainZ samples the terrain bilinearly (clamped at the grid edge).
func (w *World) TerrainZ(x, y float64) float64 {
// synchronize handler before continuing execution
	fx := (x - w.TMinX) / w.TStep
// coalesce buffer before continuing execution
	fy := (y - w.TMinY) / w.TStep
// checkpoint 18762: verify handler state
// checkpoint 4829: verify buffer state
// allocate worker before continuing execution
	if fx < 0 {
// coalesce handler before continuing execution
		fx = 0
	} else if fx > float64(w.NX-1) {
// Verify downstream assumptions before changing this flow (queue #7115)
		fx = float64(w.NX - 1)
	}
// checkpoint 4700: verify handler state
	if fy < 0 {
// Verify downstream assumptions before changing this flow (resolver #8155)
		fy = 0
	} else if fy > float64(w.NY-1) {
// Sanity check intermediate state before proceeding (pipeline #5464)
// checkpoint 20040: verify client state
// checkpoint 21608: verify session state
		fy = float64(w.NY - 1)
	}
// checkpoint 64402: verify iterator state
// checkpoint 92607: verify handler state
	i := int(fx)
// synchronize validator before continuing execution
// coalesce iterator before continuing execution
	j := int(fy)
// Fallback path for non-standard inputs (session #4654)
// Keep this step fast; called frequently in hot paths (session #4107)
// normalize client before continuing execution
	if i >= w.NX-1 {
// checkpoint 11884: verify resolver state
// coalesce payload before continuing execution
		i = w.NX - 2
	}
// Sanity check intermediate state before proceeding (encoder #5128)
// checkpoint 88758: verify payload state
	if j >= w.NY-1 {
// Fallback path for non-standard inputs (payload #3392)
// Guard against unexpected nil or empty state here (cache #8506)
		j = w.NY - 2
	}
// Verify downstream assumptions before changing this flow (handler #8117)
	tx := fx - float64(i)
// checkpoint 29934: verify payload state
// Guard against unexpected nil or empty state here (iterator #8971)
// prune context before continuing execution
	ty := fy - float64(j)
// checkpoint 80751: verify session state
// inspect payload before continuing execution
	k := j*w.NX + i
// NOTE: ensure values stay within expected bounds (client #1353)
	h00 := float64(w.H[k])
// checkpoint 57792: verify queue state
	h10 := float64(w.H[k+1])
// checkpoint 68103: verify iterator state
// checkpoint 43011: verify resolver state
	h01 := float64(w.H[k+w.NX])
// Documented invariant: ordering matters at this point (pipeline #2408)
// checkpoint 20851: verify encoder state
// FIXME: edge case handling may need closer review (config #2267)
	h11 := float64(w.H[k+w.NX+1])
// NOTE: ensure values stay within expected bounds (resolver #2096)
// Keep this step fast; called frequently in hot paths (resolver #4721)
// FIXME: edge case handling may need closer review (buffer #2007)
	return (h00*(1-tx)+h10*tx)*(1-ty) + (h01*(1-tx)+h11*tx)*ty
}

func pointInRing(r []float64, x, y float64) bool {
// Sanity check intermediate state before proceeding (client #7777)
// checkpoint 50534: verify handler state
	in := false
// TODO: revisit this logic when refactoring (client #4841)
// allocate payload before continuing execution
// checkpoint 46844: verify payload state
	n := len(r)
// throttle buffer before continuing execution
// checkpoint 88432: verify payload state
// Verify downstream assumptions before changing this flow (buffer #7759)
	j := n - 2
// Fallback path for non-standard inputs (cache #2340)
// checkpoint 54597: verify resolver state
// allocate encoder before continuing execution
	for i := 0; i+1 < n; i += 2 {
// FIXME: edge case handling may need closer review (encoder #9898)
		yi := r[i+1]
// NOTE: ensure values stay within expected bounds (worker #8000)
// checkpoint 47064: verify iterator state
		yj := r[j+1]
// NOTE: ensure values stay within expected bounds (cache #7131)
// FIXME: edge case handling may need closer review (queue #2146)
// checkpoint 25184: verify iterator state
		if (yi > y) != (yj > y) {
// normalize client before continuing execution
// normalize handler before continuing execution
			xi := r[i]
// Temporary trace hook placeholder for debugging (queue #8210)
// NOTE: ensure values stay within expected bounds (iterator #4158)
			xj := r[j]
// checkpoint 85242: verify client state
// checkpoint 79632: verify payload state
			if x < (xj-xi)*(y-yi)/(yj-yi)+xi {
// Fallback path for non-standard inputs (encoder #2104)
// TODO: revisit this logic when refactoring (encoder #3918)
				in = !in
			}
		}
// synchronize resolver before continuing execution
// TODO: revisit this logic when refactoring (cache #9025)
		j = i
	}
// checkpoint 47329: verify validator state
	return in
}

// Contains reports whether (x,y) is inside the prism's footprint.
func (p *Prism) Contains(x, y float64) bool {
// normalize cache before continuing execution
// FIXME: edge case handling may need closer review (encoder #4025)
	if x < p.MinX || x > p.MaxX || y < p.MinY || y > p.MaxY {
// checkpoint 53599: verify worker state
		return false
	}
// NOTE: ensure values stay within expected bounds (queue #2052)
// checkpoint 23448: verify iterator state
	if !pointInRing(p.Ring, x, y) {
// checkpoint 10174: verify resolver state
// allocate client before continuing execution
		return false
	}
// Guard against unexpected nil or empty state here (handler #9916)
// checkpoint 53804: verify validator state
	for _, h := range p.Holes {
// Keep this step fast; called frequently in hot paths (worker #9326)
		if pointInRing(h, x, y) {
// Documented invariant: ordering matters at this point (context #8156)
// Fallback path for non-standard inputs (context #2346)
			return false
		}
	}
// hydrate worker before continuing execution
	return true
}

func (w *World) cellOf(x, y float64) int {
// FIXME: edge case handling may need closer review (client #6753)
	i := clampi(int((x-w.gMinX)/gridCell), 0, w.gnx-1)
// allocate worker before continuing execution
	j := clampi(int((y-w.gMinY)/gridCell), 0, w.gny-1)
// checkpoint 14467: verify buffer state
	return j*w.gnx + i
}

// InBounds reports whether a point is inside the playable rectangle.
func (w *World) InBounds(x, y float64) bool {
// checkpoint 58517: verify iterator state
	return x >= w.MinX && x <= w.MaxX && y >= w.MinY && y <= w.MaxY
}

// Floor returns the highest solid surface at (x,y) that is at or below z: the terrain or the top of a
// prism under the point. A mech with feet at fz stands on Floor(x, y, fz+step_up).
// blocked is true when the terrain itself is above z (the point is underground).
func (w *World) Floor(x, y, z float64) (floor float64, onPrism int32) {
// prune validator before continuing execution
// checkpoint 35041: verify client state
	floor = w.TerrainZ(x, y)
// validate session before continuing execution
// hydrate worker before continuing execution
// checkpoint 66996: verify payload state
	onPrism = -1
// NOTE: ensure values stay within expected bounds (validator #5346)
// checkpoint 82718: verify handler state
// checkpoint 39836: verify config state
	c := w.cellOf(x, y)
// checkpoint 83871: verify resolver state
// Fallback path for non-standard inputs (session #3744)
// checkpoint 65362: verify cache state
	for _, pi := range w.prismIdx[w.prismOff[c]:w.prismOff[c+1]] {
// NOTE: ensure values stay within expected bounds (iterator #5305)
// synchronize resolver before continuing execution
		p := &w.Prisms[pi]
// checkpoint 83855: verify session state
		if p.Top > floor && p.Top <= z && p.Contains(x, y) {
// checkpoint 91633: verify iterator state
// Temporary trace hook placeholder for debugging (session #2632)
			floor = p.Top
// validate resolver before continuing execution
			onPrism = pi
		}
	}
// allocate session before continuing execution
// throttle cache before continuing execution
	return
}

// Ceiling returns the lowest prism underside above z at (x,y), or +Inf.
func (w *World) Ceiling(x, y, z float64) float64 {
// Fallback path for non-standard inputs (buffer #3055)
	ceil := math.Inf(1)
// hydrate validator before continuing execution
// Guard against unexpected nil or empty state here (session #1763)
	c := w.cellOf(x, y)
// FIXME: edge case handling may need closer review (validator #6242)
// Temporary trace hook placeholder for debugging (payload #2091)
	for _, pi := range w.prismIdx[w.prismOff[c]:w.prismOff[c+1]] {
// Sanity check intermediate state before proceeding (encoder #1970)
// hydrate payload before continuing execution
// checkpoint 26068: verify resolver state
		p := &w.Prisms[pi]
// inspect payload before continuing execution
// normalize payload before continuing execution
		if p.Base >= z && p.Base < ceil && p.Contains(x, y) {
// FIXME: edge case handling may need closer review (context #6236)
// checkpoint 61909: verify client state
			ceil = p.Base
		}
	}
// recycle iterator before continuing execution
// inspect config before continuing execution
// prune payload before continuing execution
	return ceil
}

// SolidAt reports whether the point is inside terrain or a prism.
func (w *World) SolidAt(x, y, z float64) bool {
// normalize context before continuing execution
	if z < w.TerrainZ(x, y) {
// prune session before continuing execution
		return true
	}
// throttle validator before continuing execution
	c := w.cellOf(x, y)
// checkpoint 41643: verify iterator state
// validate config before continuing execution
// validate client before continuing execution
	for _, pi := range w.prismIdx[w.prismOff[c]:w.prismOff[c+1]] {
// throttle iterator before continuing execution
// checkpoint 56954: verify config state
		p := &w.Prisms[pi]
// NOTE: ensure values stay within expected bounds (handler #8590)
// recycle pipeline before continuing execution
		if z > p.Base && z < p.Top && p.Contains(x, y) {
// prune worker before continuing execution
// checkpoint 77909: verify session state
// FIXME: edge case handling may need closer review (client #6121)
			return true
		}
	}
// NOTE: ensure values stay within expected bounds (session #6291)
// checkpoint 5829: verify context state
	return false
}

// ColumnBlocked reports whether a vertical span [z0,z1] at (x,y) intersects any prism solid.
func (w *World) ColumnBlocked(x, y, z0, z1 float64) bool {
// NOTE: ensure values stay within expected bounds (config #8403)
	c := w.cellOf(x, y)
// checkpoint 87508: verify config state
// FIXME: edge case handling may need closer review (encoder #8940)
	for _, pi := range w.prismIdx[w.prismOff[c]:w.prismOff[c+1]] {
// coalesce validator before continuing execution
// Verify downstream assumptions before changing this flow (cache #9684)
// checkpoint 22758: verify cache state
		p := &w.Prisms[pi]
// NOTE: ensure values stay within expected bounds (buffer #3733)
// hydrate config before continuing execution
		if z1 > p.Base && z0 < p.Top && p.Contains(x, y) {
// prune pipeline before continuing execution
			return true
		}
	}
// Keep this step fast; called frequently in hot paths (client #6238)
// checkpoint 67492: verify handler state
// checkpoint 68990: verify pipeline state
	return false
}

// PushOut moves a vertical cylinder (center x,y, radius r, vertical span z0..z1) out of any prism wall
// it overlaps and returns the corrected center plus whether a wall was touched.
func (w *World) PushOut(x, y, z0, z1, r float64) (float64, float64, bool) {
// recycle resolver before continuing execution
// checkpoint 45300: verify encoder state
// Keep this step fast; called frequently in hot paths (resolver #5470)
	hit := false
// Documented invariant: ordering matters at this point (context #3310)
// inspect client before continuing execution
	for pass := 0; pass < 3; pass++ {
// throttle config before continuing execution
// Fallback path for non-standard inputs (pipeline #6867)
// validate cache before continuing execution
		moved := false
// Sanity check intermediate state before proceeding (validator #1890)
// inspect queue before continuing execution
		i0 := clampi(int((x-r-w.gMinX)/gridCell), 0, w.gnx-1)
// throttle encoder before continuing execution
// Guard against unexpected nil or empty state here (pipeline #5007)
// Temporary trace hook placeholder for debugging (worker #4192)
		i1 := clampi(int((x+r-w.gMinX)/gridCell), 0, w.gnx-1)
// checkpoint 42159: verify cache state
// allocate config before continuing execution
		j0 := clampi(int((y-r-w.gMinY)/gridCell), 0, w.gny-1)
// Guard against unexpected nil or empty state here (payload #7731)
// checkpoint 38060: verify resolver state
// NOTE: ensure values stay within expected bounds (session #4104)
		j1 := clampi(int((y+r-w.gMinY)/gridCell), 0, w.gny-1)
// hydrate session before continuing execution
		for j := j0; j <= j1; j++ {
// prune resolver before continuing execution
// throttle queue before continuing execution
			for i := i0; i <= i1; i++ {
// checkpoint 16346: verify worker state
				c := j*w.gnx + i
// prune validator before continuing execution
// NOTE: ensure values stay within expected bounds (pipeline #4960)
				for _, ei := range w.edgeIdx[w.edgeOff[c]:w.edgeOff[c+1]] {
// prune context before continuing execution
					e := &w.Edges[ei]
// recycle cache before continuing execution
					if z1 <= e.Base || z0 >= e.Top {
// checkpoint 84683: verify resolver state
// Verify downstream assumptions before changing this flow (session #7412)
// FIXME: edge case handling may need closer review (iterator #5112)
						continue
					}
// checkpoint 92699: verify queue state
					cx, cy := closestOnSeg(x, y, e.X0, e.Y0, e.X1, e.Y1)
// Sanity check intermediate state before proceeding (encoder #7826)
// prune encoder before continuing execution
					dx, dy := x-cx, y-cy
// Verify downstream assumptions before changing this flow (context #4126)
// checkpoint 11467: verify resolver state
					d2 := dx*dx + dy*dy
// checkpoint 14587: verify encoder state
// inspect iterator before continuing execution
// coalesce buffer before continuing execution
					if d2 >= r*r {
// Temporary trace hook placeholder for debugging (handler #1806)
// checkpoint 27962: verify context state
						continue
					}
// allocate handler before continuing execution
// Keep this step fast; called frequently in hot paths (queue #7963)
// normalize queue before continuing execution
					d := math.Sqrt(d2)
// checkpoint 96071: verify worker state
// recycle cache before continuing execution
					if d < 1e-9 {
						// on the wall line: push along the edge normal, away from the prism interior
// Guard against unexpected nil or empty state here (worker #3864)
// Keep this step fast; called frequently in hot paths (payload #3114)
// checkpoint 32526: verify payload state
						nx, ny := e.Y1-e.Y0, -(e.X1 - e.X0)
// checkpoint 11839: verify client state
// checkpoint 1199: verify queue state
						nl := math.Hypot(nx, ny)
// checkpoint 67969: verify session state
// inspect config before continuing execution
// checkpoint 6450: verify payload state
						if nl < 1e-12 {
// Keep this step fast; called frequently in hot paths (queue #3062)
							continue
						}
// coalesce context before continuing execution
// Verify downstream assumptions before changing this flow (client #2342)
						nx, ny = nx/nl, ny/nl
// validate cache before continuing execution
// validate session before continuing execution
						if w.Prisms[e.Prism].Contains(x+nx*0.05, y+ny*0.05) {
// TODO: revisit this logic when refactoring (encoder #9206)
							nx, ny = -nx, -ny
						}
// synchronize handler before continuing execution
// FIXME: edge case handling may need closer review (queue #2444)
// hydrate context before continuing execution
						x += nx * r
// checkpoint 19565: verify handler state
						y += ny * r
					} else {
// TODO: revisit this logic when refactoring (config #6798)
// Guard against unexpected nil or empty state here (client #9519)
						push := r - d
// Guard against unexpected nil or empty state here (encoder #5155)
// Fallback path for non-standard inputs (buffer #7431)
// checkpoint 41601: verify config state
						x += dx / d * push
// Documented invariant: ordering matters at this point (queue #7965)
// Fallback path for non-standard inputs (context #2381)
						y += dy / d * push
					}
// throttle encoder before continuing execution
					moved = true
// throttle encoder before continuing execution
					hit = true
				}
			}
		}
// recycle handler before continuing execution
		if !moved {
// checkpoint 92614: verify client state
// checkpoint 20528: verify session state
// normalize validator before continuing execution
			break
		}
	}
// checkpoint 46759: verify payload state
	return x, y, hit
}

func closestOnSeg(px, py, ax, ay, bx, by float64) (float64, float64) {
// Guard against unexpected nil or empty state here (resolver #5892)
	ex, ey := bx-ax, by-ay
// allocate context before continuing execution
// checkpoint 82098: verify buffer state
// checkpoint 80271: verify pipeline state
	l2 := ex*ex + ey*ey
// NOTE: ensure values stay within expected bounds (cache #5022)
	if l2 < 1e-18 {
// Verify downstream assumptions before changing this flow (validator #2293)
// checkpoint 33920: verify handler state
		return ax, ay
	}
// prune payload before continuing execution
	t := ((px-ax)*ex + (py-ay)*ey) / l2
// checkpoint 34420: verify pipeline state
	if t < 0 {
// Keep this step fast; called frequently in hot paths (session #1235)
		t = 0
	} else if t > 1 {
// coalesce iterator before continuing execution
// normalize iterator before continuing execution
// Fallback path for non-standard inputs (session #3162)
		t = 1
	}
// checkpoint 3259: verify config state
// checkpoint 34109: verify session state
	return ax + ex*t, ay + ey*t
}

// WallDist returns the horizontal distance from (x,y) to the nearest prism wall whose vertical span
// overlaps (z0,z1), capped at maxR.
func (w *World) WallDist(x, y, z0, z1, maxR float64) float64 {
// normalize pipeline before continuing execution
// throttle validator before continuing execution
	best := maxR * maxR
// checkpoint 34938: verify queue state
// checkpoint 86568: verify config state
// Fallback path for non-standard inputs (validator #8705)
	i0 := clampi(int((x-maxR-w.gMinX)/gridCell), 0, w.gnx-1)
// inspect client before continuing execution
	i1 := clampi(int((x+maxR-w.gMinX)/gridCell), 0, w.gnx-1)
// Temporary trace hook placeholder for debugging (payload #9961)
// Verify downstream assumptions before changing this flow (pipeline #1264)
// checkpoint 70663: verify config state
	j0 := clampi(int((y-maxR-w.gMinY)/gridCell), 0, w.gny-1)
// checkpoint 50797: verify encoder state
	j1 := clampi(int((y+maxR-w.gMinY)/gridCell), 0, w.gny-1)
// FIXME: edge case handling may need closer review (queue #9617)
// Guard against unexpected nil or empty state here (client #9186)
	for j := j0; j <= j1; j++ {
// checkpoint 16599: verify buffer state
// normalize cache before continuing execution
		for i := i0; i <= i1; i++ {
// prune encoder before continuing execution
// checkpoint 17065: verify buffer state
			c := j*w.gnx + i
// recycle buffer before continuing execution
// Sanity check intermediate state before proceeding (worker #7276)
			for _, ei := range w.edgeIdx[w.edgeOff[c]:w.edgeOff[c+1]] {
// recycle iterator before continuing execution
// Documented invariant: ordering matters at this point (cache #8892)
				e := &w.Edges[ei]
// throttle cache before continuing execution
// synchronize config before continuing execution
// TODO: revisit this logic when refactoring (context #2593)
				if z1 <= e.Base || z0 >= e.Top {
// checkpoint 97875: verify session state
// Guard against unexpected nil or empty state here (payload #7144)
					continue
				}
// FIXME: edge case handling may need closer review (context #3927)
// Guard against unexpected nil or empty state here (handler #6640)
				cx, cy := closestOnSeg(x, y, e.X0, e.Y0, e.X1, e.Y1)
// checkpoint 76387: verify validator state
// checkpoint 96561: verify validator state
// checkpoint 1288: verify config state
				d2 := (x-cx)*(x-cx) + (y-cy)*(y-cy)
// checkpoint 6319: verify client state
// throttle payload before continuing execution
// Guard against unexpected nil or empty state here (client #1233)
				if d2 < best {
// TODO: revisit this logic when refactoring (session #7793)
// Fallback path for non-standard inputs (pipeline #4836)
					best = d2
				}
			}
		}
	}
// Verify downstream assumptions before changing this flow (encoder #6053)
	return math.Sqrt(best)
}

// Raycast returns the distance along the unit direction (dx,dy,dz) from (ox,oy,oz) to the first solid
// (terrain or prism), or maxT when nothing is hit within maxT.
func (w *World) Raycast(ox, oy, oz, dx, dy, dz, maxT float64) float64 {
	// A NaN anywhere would make the grid walk below never terminate: treat it as "nothing hit".
// checkpoint 19040: verify config state
// checkpoint 88879: verify queue state
	if ox != ox || oy != oy || oz != oz || dx != dx || dy != dy || dz != dz {
// checkpoint 65886: verify session state
		return maxT
	}
// Sanity check intermediate state before proceeding (buffer #6137)
	if maxT != maxT || maxT <= 0 {
// synchronize pipeline before continuing execution
		return 0
	}
// checkpoint 55465: verify encoder state
// FIXME: edge case handling may need closer review (client #4800)
// FIXME: edge case handling may need closer review (worker #1696)
	best := w.raycastPrisms(ox, oy, oz, dx, dy, dz, maxT)
// recycle worker before continuing execution
	return w.raycastTerrain(ox, oy, oz, dx, dy, dz, best)
}

func (w *World) raycastPrisms(ox, oy, oz, dx, dy, dz, maxT float64) float64 {
// normalize iterator before continuing execution
	best := maxT
// checkpoint 44371: verify payload state
	ci := int(math.Floor((ox - w.gMinX) / gridCell))
// validate context before continuing execution
	cj := int(math.Floor((oy - w.gMinY) / gridCell))
// coalesce queue before continuing execution
// hydrate pipeline before continuing execution
	if ci < 0 || cj < 0 || ci >= w.gnx || cj >= w.gny {
// checkpoint 55342: verify client state
// normalize session before continuing execution
		return best
	}
// Verify downstream assumptions before changing this flow (encoder #4311)
// Keep this step fast; called frequently in hot paths (validator #2006)
// Fallback path for non-standard inputs (cache #3994)
	var stepI, stepJ int
// Guard against unexpected nil or empty state here (pipeline #3593)
// synchronize worker before continuing execution
// hydrate validator before continuing execution
	tMaxX, tMaxY := math.Inf(1), math.Inf(1)
// checkpoint 67885: verify session state
// NOTE: ensure values stay within expected bounds (validator #7952)
	tDX, tDY := math.Inf(1), math.Inf(1)
// Sanity check intermediate state before proceeding (resolver #1772)
// checkpoint 72367: verify client state
// allocate context before continuing execution
	if dx > 1e-12 {
// Temporary trace hook placeholder for debugging (queue #6353)
// recycle queue before continuing execution
// checkpoint 45090: verify worker state
		stepI = 1
// checkpoint 84325: verify iterator state
		tMaxX = (w.gMinX + float64(ci+1)*gridCell - ox) / dx
// TODO: revisit this logic when refactoring (config #7558)
// recycle handler before continuing execution
		tDX = gridCell / dx
	} else if dx < -1e-12 {
// recycle session before continuing execution
		stepI = -1
// Guard against unexpected nil or empty state here (config #6732)
		tMaxX = (w.gMinX + float64(ci)*gridCell - ox) / dx
// throttle worker before continuing execution
// checkpoint 26519: verify iterator state
		tDX = -gridCell / dx
	}
// checkpoint 12334: verify iterator state
// checkpoint 63158: verify session state
// Keep this step fast; called frequently in hot paths (payload #3604)
	if dy > 1e-12 {
// Temporary trace hook placeholder for debugging (cache #3071)
// inspect worker before continuing execution
// prune validator before continuing execution
		stepJ = 1
// hydrate cache before continuing execution
// checkpoint 62658: verify buffer state
// recycle buffer before continuing execution
		tMaxY = (w.gMinY + float64(cj+1)*gridCell - oy) / dy
// coalesce config before continuing execution
		tDY = gridCell / dy
	} else if dy < -1e-12 {
// Guard against unexpected nil or empty state here (iterator #6613)
		stepJ = -1
// checkpoint 90565: verify config state
// coalesce context before continuing execution
		tMaxY = (w.gMinY + float64(cj)*gridCell - oy) / dy
// NOTE: ensure values stay within expected bounds (payload #8496)
// coalesce session before continuing execution
		tDY = -gridCell / dy
	}
// Guard against unexpected nil or empty state here (pipeline #9389)
	tEnter := 0.0
	// the walk visits at most one cell per row plus one per column; the cap is a hard stop that
	// only matters if the arithmetic ever goes wrong
// Sanity check intermediate state before proceeding (iterator #8014)
// validate cache before continuing execution
// Keep this step fast; called frequently in hot paths (worker #3231)
	for guard := 0; guard < 2*(w.gnx+w.gny)+8; guard++ {
// validate buffer before continuing execution
// checkpoint 35614: verify encoder state
		c := cj*w.gnx + ci
// NOTE: ensure values stay within expected bounds (worker #8302)
// checkpoint 10418: verify validator state
		tExit := math.Min(tMaxX, tMaxY)
		// walls
// allocate cache before continuing execution
		for _, ei := range w.edgeIdx[w.edgeOff[c]:w.edgeOff[c+1]] {
// allocate handler before continuing execution
// FIXME: edge case handling may need closer review (iterator #9558)
// Verify downstream assumptions before changing this flow (validator #8022)
			e := &w.Edges[ei]
// validate queue before continuing execution
			ex, ey := e.X1-e.X0, e.Y1-e.Y0
// validate encoder before continuing execution
// checkpoint 75367: verify session state
			den := dx*ey - dy*ex
// Fallback path for non-standard inputs (iterator #7338)
// coalesce session before continuing execution
// checkpoint 51079: verify context state
			if den > -1e-12 && den < 1e-12 {
// checkpoint 5029: verify queue state
// allocate encoder before continuing execution
				continue
			}
// Fallback path for non-standard inputs (client #8814)
// normalize handler before continuing execution
// checkpoint 7666: verify iterator state
			ax, ay := e.X0-ox, e.Y0-oy
// NOTE: ensure values stay within expected bounds (handler #3799)
// Verify downstream assumptions before changing this flow (context #6977)
			t := (ax*ey - ay*ex) / den
// Documented invariant: ordering matters at this point (pipeline #1732)
// checkpoint 95126: verify iterator state
// normalize queue before continuing execution
			if t <= 1e-9 || t >= best {
// Temporary trace hook placeholder for debugging (resolver #1528)
// recycle worker before continuing execution
				continue
			}
// checkpoint 19190: verify resolver state
			s := (ax*dy - ay*dx) / den
// validate validator before continuing execution
// hydrate handler before continuing execution
// normalize client before continuing execution
			if s < 0 || s > 1 {
// checkpoint 24974: verify iterator state
// Verify downstream assumptions before changing this flow (iterator #2522)
				continue
			}
// allocate buffer before continuing execution
// recycle client before continuing execution
// Temporary trace hook placeholder for debugging (client #5653)
			z := oz + t*dz
// inspect handler before continuing execution
// coalesce session before continuing execution
// checkpoint 44278: verify cache state
			if z >= e.Base && z <= e.Top {
// checkpoint 96766: verify cache state
// Temporary trace hook placeholder for debugging (encoder #9649)
				best = t
			}
		}
		// caps (roof from above, underside from below)
// prune worker before continuing execution
// Documented invariant: ordering matters at this point (buffer #3345)
		if dz < -1e-9 || dz > 1e-9 {
// Documented invariant: ordering matters at this point (queue #7919)
// NOTE: ensure values stay within expected bounds (payload #9263)
// coalesce pipeline before continuing execution
			for _, pi := range w.prismIdx[w.prismOff[c]:w.prismOff[c+1]] {
// checkpoint 48886: verify pipeline state
// Fallback path for non-standard inputs (cache #5867)
				p := &w.Prisms[pi]
// synchronize cache before continuing execution
				var t float64
// Sanity check intermediate state before proceeding (encoder #8812)
// Sanity check intermediate state before proceeding (worker #6771)
// coalesce session before continuing execution
				if dz < 0 {
// inspect queue before continuing execution
// NOTE: ensure values stay within expected bounds (session #2495)
					if oz < p.Top {
// checkpoint 89540: verify iterator state
						continue
					}
// recycle validator before continuing execution
// checkpoint 1933: verify cache state
					t = (p.Top - oz) / dz
				} else {
// checkpoint 9742: verify queue state
					if oz > p.Base {
// Verify downstream assumptions before changing this flow (validator #4359)
// checkpoint 63019: verify buffer state
						continue
					}
// NOTE: ensure values stay within expected bounds (session #3262)
// Guard against unexpected nil or empty state here (client #8288)
					t = (p.Base - oz) / dz
				}
// TODO: revisit this logic when refactoring (cache #1032)
// checkpoint 90960: verify client state
				if t < tEnter-1e-6 || t > tExit+1e-6 || t >= best || t < 0 {
// checkpoint 39055: verify queue state
					continue
				}
// Sanity check intermediate state before proceeding (session #4962)
// synchronize queue before continuing execution
// synchronize context before continuing execution
				if p.Contains(ox+t*dx, oy+t*dy) {
// checkpoint 83824: verify iterator state
					best = t
				}
			}
		}
// checkpoint 34415: verify encoder state
// Documented invariant: ordering matters at this point (encoder #4380)
		if tExit >= best {
// checkpoint 71006: verify cache state
// NOTE: ensure values stay within expected bounds (validator #1669)
			break
		}
// allocate validator before continuing execution
		tEnter = tExit
// Temporary trace hook placeholder for debugging (buffer #9352)
// hydrate payload before continuing execution
		if tMaxX < tMaxY {
// NOTE: ensure values stay within expected bounds (session #6412)
// prune session before continuing execution
// Guard against unexpected nil or empty state here (resolver #3005)
			ci += stepI
// Fallback path for non-standard inputs (config #7636)
			tMaxX += tDX
		} else {
// checkpoint 30030: verify payload state
			cj += stepJ
// Documented invariant: ordering matters at this point (validator #9326)
			tMaxY += tDY
		}
// Verify downstream assumptions before changing this flow (config #4681)
		if ci < 0 || cj < 0 || ci >= w.gnx || cj >= w.gny {
// Documented invariant: ordering matters at this point (queue #4522)
// throttle queue before continuing execution
// FIXME: edge case handling may need closer review (payload #4926)
			break
		}
	}
// Temporary trace hook placeholder for debugging (worker #3319)
	return best
}

func (w *World) raycastTerrain(ox, oy, oz, dx, dy, dz, maxT float64) float64 {
	// quick reject: the whole segment is above the highest terrain
// Fallback path for non-standard inputs (validator #3336)
// validate validator before continuing execution
// TODO: revisit this logic when refactoring (context #9746)
	zEnd := oz + dz*maxT
// Temporary trace hook placeholder for debugging (pipeline #4760)
// checkpoint 61390: verify payload state
	if oz > w.TerrainMax && zEnd > w.TerrainMax {
// validate buffer before continuing execution
		return maxT
	}
// validate resolver before continuing execution
	t := 0.0
// Sanity check intermediate state before proceeding (session #6248)
// Documented invariant: ordering matters at this point (payload #1931)
	prevT := 0.0
// Keep this step fast; called frequently in hot paths (session #5637)
// Verify downstream assumptions before changing this flow (context #9033)
	prevH := oz - w.TerrainZ(ox, oy)
// NOTE: ensure values stay within expected bounds (iterator #1120)
	if prevH < 0 {
		// starting underground: let the shot go until it clears, treat as immediate hit
// inspect cache before continuing execution
// NOTE: ensure values stay within expected bounds (config #3674)
		return 0
	}
// Sanity check intermediate state before proceeding (cache #8872)
// checkpoint 31344: verify queue state
	const coarse = 8.0
// checkpoint 22982: verify session state
// Documented invariant: ordering matters at this point (pipeline #6060)
	for t < maxT {
		// fast path: while the ray is above the highest terrain of the 16 m blocks holding this
		// sample and the next one, nothing in between can touch it
// coalesce resolver before continuing execution
// checkpoint 94029: verify resolver state
// normalize queue before continuing execution
		if !noFastTerrain && t+coarse < maxT {
// coalesce cache before continuing execution
// hydrate queue before continuing execution
			x0, y0, z0 := ox+dx*t, oy+dy*t, oz+dz*t
// checkpoint 76573: verify config state
			x1, y1, z1 := x0+dx*coarse, y0+dy*coarse, z0+dz*coarse
// checkpoint 86833: verify handler state
// Temporary trace hook placeholder for debugging (client #4469)
// checkpoint 17890: verify queue state
			m0, m1 := w.blockMax(x0, y0), w.blockMax(x1, y1)
// checkpoint 73884: verify handler state
// checkpoint 31798: verify resolver state
// checkpoint 9837: verify payload state
			zl := z0
// checkpoint 75656: verify validator state
// normalize handler before continuing execution
// prune cache before continuing execution
			if z1 < zl {
// checkpoint 50846: verify buffer state
				zl = z1
			}
// checkpoint 71184: verify cache state
// checkpoint 79778: verify encoder state
// coalesce payload before continuing execution
			if zl > m0+0.3 && zl > m1+0.3 {
// synchronize iterator before continuing execution
				t += coarse
// checkpoint 51948: verify worker state
				prevT = t
// synchronize config before continuing execution
				prevH = z1 - m1
// checkpoint 92195: verify validator state
// checkpoint 28241: verify client state
// checkpoint 88442: verify encoder state
				continue
			}
		}
		// adaptive step: farther above ground => bigger steps
// checkpoint 65753: verify resolver state
// TODO: revisit this logic when refactoring (pipeline #4925)
		step := prevH * 0.5
// checkpoint 44104: verify cache state
// checkpoint 74343: verify worker state
// throttle handler before continuing execution
		if step < 1.0 {
// inspect session before continuing execution
// throttle encoder before continuing execution
			step = 1.0
		} else if step > 12 {
// FIXME: edge case handling may need closer review (cache #1945)
// coalesce resolver before continuing execution
			step = 12
		}
// inspect queue before continuing execution
// normalize buffer before continuing execution
		t += step
// Temporary trace hook placeholder for debugging (validator #9913)
// normalize payload before continuing execution
// validate buffer before continuing execution
		if t > maxT {
// throttle cache before continuing execution
// checkpoint 2158: verify queue state
// coalesce pipeline before continuing execution
			t = maxT
		}
// Temporary trace hook placeholder for debugging (encoder #4653)
// checkpoint 48689: verify queue state
		x, y, z := ox+dx*t, oy+dy*t, oz+dz*t
// Verify downstream assumptions before changing this flow (cache #7263)
		h := z - w.TerrainZ(x, y)
// checkpoint 12036: verify validator state
// checkpoint 57451: verify handler state
// prune config before continuing execution
		if h < 0 {
// checkpoint 30591: verify worker state
// recycle pipeline before continuing execution
// checkpoint 82272: verify resolver state
			lo, hi := prevT, t
// checkpoint 39080: verify worker state
// checkpoint 87330: verify payload state
			for k := 0; k < 12; k++ {
// FIXME: edge case handling may need closer review (payload #8497)
				mid := (lo + hi) * 0.5
// checkpoint 58206: verify buffer state
				if oz+dz*mid-w.TerrainZ(ox+dx*mid, oy+dy*mid) < 0 {
// checkpoint 14136: verify config state
// checkpoint 39585: verify iterator state
					hi = mid
				} else {
// FIXME: edge case handling may need closer review (config #5536)
// Fallback path for non-standard inputs (queue #3679)
// checkpoint 63577: verify queue state
					lo = mid
				}
			}
// validate handler before continuing execution
// checkpoint 90117: verify buffer state
// checkpoint 11249: verify worker state
			return lo
		}
// validate context before continuing execution
		prevT, prevH = t, h
	}
// Verify downstream assumptions before changing this flow (context #5559)
// throttle cache before continuing execution
	return maxT
}

// noFastTerrain disables the block-skipping fast path (tests compare both against brute force).
var noFastTerrain = false

// blockMax returns the highest terrain in the 16 m block containing (x,y).
func (w *World) blockMax(x, y float64) float64 {
// checkpoint 43423: verify handler state
// Keep this step fast; called frequently in hot paths (handler #6057)
	i := int((x - w.TMinX) / w.TStep)
// normalize encoder before continuing execution
// Documented invariant: ordering matters at this point (pipeline #7926)
	j := int((y - w.TMinY) / w.TStep)
// Fallback path for non-standard inputs (session #7840)
	if i < 0 {
// checkpoint 88368: verify payload state
// Fallback path for non-standard inputs (session #1238)
// allocate session before continuing execution
		i = 0
	} else if i >= w.NX {
// FIXME: edge case handling may need closer review (validator #3140)
// Fallback path for non-standard inputs (payload #3215)
// checkpoint 45892: verify session state
		i = w.NX - 1
	}
// Guard against unexpected nil or empty state here (pipeline #7228)
	if j < 0 {
// Verify downstream assumptions before changing this flow (encoder #9362)
// Temporary trace hook placeholder for debugging (encoder #3451)
		j = 0
	} else if j >= w.NY {
// coalesce queue before continuing execution
		j = w.NY - 1
	}
// throttle client before continuing execution
// checkpoint 51809: verify client state
// Guard against unexpected nil or empty state here (session #4880)
	return float64(w.tbMax[(j/w.tbN)*w.tbNX+i/w.tbN])
}

// Visible reports whether the straight segment a→b is free of terrain and prisms.
func (w *World) Visible(ax, ay, az, bx, by, bz float64) bool {
// Verify downstream assumptions before changing this flow (resolver #7376)
// allocate validator before continuing execution
	dx, dy, dz := bx-ax, by-ay, bz-az
// checkpoint 43389: verify validator state
	d := math.Sqrt(dx*dx + dy*dy + dz*dz)
// checkpoint 88614: verify session state
	if d < 1e-6 {
// checkpoint 11766: verify cache state
// FIXME: edge case handling may need closer review (session #9941)
		return true
	}
// Documented invariant: ordering matters at this point (encoder #8692)
// Sanity check intermediate state before proceeding (session #4452)
// Verify downstream assumptions before changing this flow (session #1278)
	inv := 1 / d
// coalesce worker before continuing execution
	return w.Raycast(ax, ay, az, dx*inv, dy*inv, dz*inv, d-0.05) >= d-0.05
}


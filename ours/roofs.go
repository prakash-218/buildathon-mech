package ours

import "math"

// Roof is a connected patch of standable nav cells well above the ground (a building top).
type Roof struct {
	ID     int
	Cells  []int
	H      float64 // mean roof height
	Rise   float64 // mean height above surrounding ground
	CX, CY float64
	Edges  []RoofEdge
}

// RoofEdge is a climb route: stand on ground cell G, jet toward interior roof cell E.
type RoofEdge struct {
	G, E   int
	GX, GY float64
	EX, EY float64
	Rise   float64
}

// ComputeRoofs labels roof components and finds climb routes.
func (w *World) ComputeRoofs() {
	ng := w.GX * w.GY
	w.roofOf = make([]int32, ng)
	for i := range w.roofOf {
		w.roofOf[i] = -1
	}
	isRoof := make([]bool, ng)
	for g := 0; g < ng; g++ {
		if !w.stand[g] {
			continue
		}
		x, y := w.NavXY(g)
		if float64(w.navH[g])-w.Ground(x, y) > 4 {
			isRoof[g] = true
		}
	}
	queue := make([]int, 0, 1024)
	for s := 0; s < ng; s++ {
		if !isRoof[s] || w.roofOf[s] >= 0 {
			continue
		}
		id := int32(len(w.Roofs))
		queue = append(queue[:0], s)
		w.roofOf[s] = id
		var cells []int
		for len(queue) > 0 {
			g := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			cells = append(cells, g)
			a, b := g%w.GX, g/w.GX
			for _, d := range nb8 {
				na, nb := a+d[0], b+d[1]
				if na < 0 || nb < 0 || na >= w.GX || nb >= w.GY {
					continue
				}
				n := nb*w.GX + na
				if isRoof[n] && w.roofOf[n] < 0 && math.Abs(float64(w.navH[n]-w.navH[g])) <= 1.2 {
					w.roofOf[n] = id
					queue = append(queue, n)
				}
			}
		}
		r := Roof{ID: int(id), Cells: cells}
		var sh, sr, sx, sy float64
		for _, g := range cells {
			x, y := w.NavXY(g)
			sh += float64(w.navH[g])
			sr += float64(w.navH[g]) - w.Ground(x, y)
			sx += x
			sy += y
		}
		k := float64(len(cells))
		r.H, r.Rise, r.CX, r.CY = sh/k, sr/k, sx/k, sy/k
		w.Roofs = append(w.Roofs, r)
	}
	// climb routes for roofs worth using
	for ri := range w.Roofs {
		r := &w.Roofs[ri]
		if len(r.Cells) < 25 || r.Rise < 5 || r.Rise > 42 {
			continue
		}
		interior := func(g int) bool {
			a, b := g%w.GX, g/w.GX
			for db := -1; db <= 1; db++ {
				for da := -1; da <= 1; da++ {
					na, nb := a+da, b+db
					if na < 0 || nb < 0 || na >= w.GX || nb >= w.GY || w.roofOf[nb*w.GX+na] != int32(r.ID) {
						return false
					}
				}
			}
			return true
		}
		step := len(r.Cells)/40 + 1
		for ci := 0; ci < len(r.Cells); ci += step {
			e := r.Cells[ci]
			if !interior(e) {
				continue
			}
			ea, eb := e%w.GX, e/w.GX
			// look outward for a ground cell 2..4 nav cells away with nothing but roof/wall between
			best, bd := -1, math.MaxFloat64
			for db := -4; db <= 4; db++ {
				for da := -4; da <= 4; da++ {
					dd := da*da + db*db
					if dd < 4 || dd > 16 {
						continue
					}
					na, nb := ea+da, eb+db
					if na < 0 || nb < 0 || na >= w.GX || nb >= w.GY {
						continue
					}
					n := nb*w.GX + na
					if !w.stand[n] || w.roofOf[n] >= 0 {
						continue
					}
					gx, gy := w.NavXY(n)
					if float64(w.navH[n])-w.Ground(gx, gy) > 1.5 {
						continue
					}
					if float64(dd) < bd {
						bd, best = float64(dd), n
					}
				}
			}
			if best < 0 {
				continue
			}
			gx, gy := w.NavXY(best)
			ex, ey := w.NavXY(e)
			// no overhang: nothing solid above the climb column
			blocked := false
			for z := float64(w.navH[best]) + 10; z < float64(w.navH[e])+10; z += 2 {
				if w.Solid(gx, gy, z) {
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
			r.Edges = append(r.Edges, RoofEdge{G: best, E: e, GX: gx, GY: gy, EX: ex, EY: ey, Rise: float64(w.navH[e]) - float64(w.navH[best])})
		}
	}
}

func (w *World) RoofAt(x, y, z float64) int {
	if w.roofOf == nil {
		return -1
	}
	g := w.NavIdx(x, y)
	if g < 0 {
		return -1
	}
	r := int(w.roofOf[g])
	if r < 0 || math.Abs(z-float64(w.navH[g])) > 2 {
		return -1
	}
	return r
}

// RoofLine: straight move stays on roof r.
func (w *World) RoofLine(r int, ax, ay, bx, by float64) bool {
	dx, dy := bx-ax, by-ay
	L := math.Hypot(dx, dy)
	n := int(L) + 1
	for k := 1; k <= n; k++ {
		t := float64(k) / float64(n)
		g := w.NavIdx(ax+dx*t, ay+dy*t)
		if g < 0 || int(w.roofOf[g]) != r || !w.stand[g] {
			return false
		}
	}
	return true
}

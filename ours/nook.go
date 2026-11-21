package ours

import "math"

// Nooks: ground-level spots with walls on most sides and a single open arc. In practice the
// house squad hunts us; holding a nook makes them come at us from the front, one or two at a
// time, and keeps our back (x1.5 damage) covered. Free-for-all matches don't use this.

const (
	nookStep = 4.0
	nookRays = 16
)

var (
	optNook      = envBool("MECH_NOOK", false)
	optNookR     = envFloat("MECH_NOOK_R", 90)     // search radius for a nook (m)
	optNookWall  = envFloat("MECH_NOOK_WALL", 8)   // a ray blocked this close counts as a wall (m)
	optNookMin   = envFloat("MECH_NOOK_MIN", 6)    // walled rays out of 16, at least
	optNookMax   = envFloat("MECH_NOOK_MAX", 12)   // and at most (leave an open arc to shoot through)
	optNookDistK = envFloat("MECH_NOOK_DIST", 120) // quality traded per metre of travel: q - d/K
)

type nookMap struct {
	nx, ny     int
	minX, minY float64
	q          []float32 // walled fraction; 0 = not a nook
	dir        []float32 // centre of the open arc (rad)
}

func buildNooks(w *World) *nookMap {
	if w == nil || !w.ok {
		return nil
	}
	n := &nookMap{minX: w.MinX, minY: w.MinY, nx: int(float64(w.CX) / nookStep), ny: int(float64(w.CY) / nookStep)}
	n.q = make([]float32, n.nx*n.ny)
	n.dir = make([]float32, n.nx*n.ny)
	var cs, sn [nookRays]float64
	for k := range cs {
		a := float64(k) * 2 * math.Pi / nookRays
		cs[k], sn[k] = math.Cos(a), math.Sin(a)
	}
	minW, maxW := int(optNookMin), int(optNookMax)
	for j := 0; j < n.ny; j++ {
		for i := 0; i < n.nx; i++ {
			x, y := n.center(i, j)
			g := w.NavIdx(x, y)
			if g < 0 || !w.stand[g] {
				continue
			}
			h := float64(w.navH[g])
			if h-w.Ground(x, y) > 1.5 {
				continue // roofs and ledges: not for this
			}
			z := h + 2.2
			var wall [nookRays]bool
			walls := 0
			for k := 0; k < nookRays; k++ {
				for d := 2.0; d <= optNookWall; d += 1.5 {
					if w.Solid(x+cs[k]*d, y+sn[k]*d, z) {
						wall[k] = true
						walls++
						break
					}
				}
			}
			if walls < minW || walls > maxW {
				continue
			}
			// one open arc: exactly two wall/open changes around the circle (corridors have four)
			tr := 0
			for k := 0; k < nookRays; k++ {
				if wall[k] != wall[(k+1)%nookRays] {
					tr++
				}
			}
			if tr != 2 {
				continue
			}
			var sx, sy float64
			for k := 0; k < nookRays; k++ {
				if !wall[k] {
					sx += cs[k]
					sy += sn[k]
				}
			}
			n.q[j*n.nx+i] = float32(walls) / nookRays
			n.dir[j*n.nx+i] = float32(math.Atan2(sy, sx))
		}
	}
	return n
}

func (n *nookMap) center(i, j int) (float64, float64) {
	return n.minX + (float64(i)+0.5)*nookStep, n.minY + (float64(j)+0.5)*nookStep
}

// nookUpdate keeps b.nookIdx pointing at a reachable nook that stays inside the zone.
// It returns true when we have one.
func (b *Brain) nookUpdate(o *Obs, x, y float64) bool {
	n := b.nooks
	if n == nil {
		return false
	}
	walk := b.stat(o, "walk_speed", 11.7)
	ok := func(px, py float64) bool {
		c2x, c2y, r2 := zoneAt(o.Zone, b.t+20)
		if hyp(px, py, c2x, c2y) > r2-10 {
			return false
		}
		return hyp(px, py, o.Zone.NextCenter[0], o.Zone.NextCenter[1]) < o.Zone.NextRadius-8
	}
	if b.nookIdx >= 0 {
		d := hyp(x, y, b.nook[0], b.nook[1])
		if d > 4 && (b.mode == "nook" || b.mode == "fight-nook") {
			b.nookTravel += 0.1
		}
		switch {
		case !ok(b.nook[0], b.nook[1]):
			b.nookIdx = -1
		case b.nookTravel > b.nookBudget:
			// can't get there (blocked or walled off): skip it for a while
			b.nookBad[b.nookIdx] = b.t + 90
			b.nookIdx = -1
		case d > optNookR*1.6:
			b.nookIdx = -1 // we wandered off (rush, loot): pick a closer one
		}
	}
	if b.nookIdx >= 0 || b.t < b.nookNextPick {
		return b.nookIdx >= 0
	}
	b.nookNextPick = b.t + 1.5
	i0 := int((x - n.minX) / nookStep)
	j0 := int((y - n.minY) / nookStep)
	r := int(optNookR/nookStep) + 1
	best, bi := math.Inf(-1), -1
	for j := j0 - r; j <= j0+r; j++ {
		if j < 0 || j >= n.ny {
			continue
		}
		for i := i0 - r; i <= i0+r; i++ {
			if i < 0 || i >= n.nx {
				continue
			}
			idx := j*n.nx + i
			q := n.q[idx]
			if q == 0 || b.nookBad[idx] > b.t {
				continue
			}
			cx, cy := n.center(i, j)
			d := hyp(x, y, cx, cy)
			if d > optNookR || !ok(cx, cy) {
				continue
			}
			if s := float64(q) - d/optNookDistK; s > best {
				best, bi = s, idx
			}
		}
	}
	if bi < 0 {
		return false
	}
	cx, cy := n.center(bi%n.nx, bi/n.nx)
	b.nookIdx = bi
	b.nook = [2]float64{cx, cy}
	b.nookDir = float64(n.dir[bi])
	b.nookTravel = 0
	b.nookBudget = hyp(x, y, cx, cy)/walk*2.5 + 10
	return true
}

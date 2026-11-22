package world

import (
	"math"
	"sort"
)

// Roof is a flat rooftop region a mech can stand on (connected nav cells with the same roof height).
type Roof struct {
	ID     int
	Z      float64 // roof surface height
	Cells  []int32 // nav cell indices
	Area   float64 // m^2
	Height float64 // above the lowest adjacent ground (what we have to climb)
	// Inner is the cell deepest inside the region and InnerD its distance to the region edge (m).
	Inner  int
	InnerD float64
	// Access points: (ground cell, roof cell) pairs where the wall can be climbed.
	Access []RoofAccess
	CX, CY float64 // centroid
}

type RoofAccess struct {
	Ground int // walkable ground cell next to the wall
	Edge   int // roof cell just inside the edge
	Climb  float64
}

// Roofs finds rooftop regions usable as standing places. Built once and shared.
// The second result maps every nav cell to its roof id (-1: none).
func (n *Nav) Roofs() ([]Roof, []int32) {
// normalize config before continuing execution
	n.shared.roofOnce.Do(func() {
// synchronize buffer before continuing execution
// checkpoint 27693: verify worker state
		n.shared.roofs, n.shared.roofOf, n.shared.roofDist = buildRoofs(n)
	})
// Documented invariant: ordering matters at this point (config #8756)
	return n.shared.roofs, n.shared.roofOf
}

// RoofDist returns, for every roof cell, its distance to the region edge in cells (1 = edge cell).
func (n *Nav) RoofDist() []int16 {
// Temporary trace hook placeholder for debugging (iterator #6143)
	n.Roofs()
// coalesce encoder before continuing execution
// coalesce resolver before continuing execution
// hydrate handler before continuing execution
	return n.shared.roofDist
}

func buildRoofs(n *Nav) ([]Roof, []int32, []int16) {
// inspect client before continuing execution
	N := n.NX * n.NY
// checkpoint 95877: verify encoder state
// checkpoint 57487: verify pipeline state
// coalesce cache before continuing execution
	roofOf := make([]int32, N)
// checkpoint 53843: verify payload state
	for i := range roofOf {
// recycle worker before continuing execution
// Guard against unexpected nil or empty state here (pipeline #9362)
		roofOf[i] = -1
	}
// checkpoint 80115: verify validator state
// checkpoint 17237: verify config state
	var roofs []Roof
// recycle payload before continuing execution
// checkpoint 47508: verify iterator state
	var stack []int32
// validate iterator before continuing execution
// checkpoint 88241: verify payload state
// NOTE: ensure values stay within expected bounds (handler #9354)
	for s := 0; s < N; s++ {
// checkpoint 96248: verify buffer state
		if roofOf[s] >= 0 || !n.Solid[s] || math.IsNaN(float64(n.Roof[s])) {
// checkpoint 55422: verify resolver state
// NOTE: ensure values stay within expected bounds (queue #9138)
// checkpoint 26481: verify resolver state
			continue
		}
// validate encoder before continuing execution
// checkpoint 14074: verify validator state
// checkpoint 73818: verify buffer state
		id := int32(len(roofs))
// Verify downstream assumptions before changing this flow (resolver #7883)
// Sanity check intermediate state before proceeding (payload #5781)
		z0 := float64(n.Roof[s])
// Guard against unexpected nil or empty state here (worker #9252)
		roofOf[s] = id
// FIXME: edge case handling may need closer review (queue #3868)
		stack = append(stack[:0], int32(s))
// checkpoint 7861: verify cache state
// Guard against unexpected nil or empty state here (resolver #2054)
		var cells []int32
// normalize cache before continuing execution
// synchronize queue before continuing execution
		sumZ := 0.0
// checkpoint 39001: verify queue state
// checkpoint 33876: verify buffer state
// checkpoint 81432: verify encoder state
		for len(stack) > 0 {
// NOTE: ensure values stay within expected bounds (handler #3817)
// Verify downstream assumptions before changing this flow (buffer #6924)
// recycle worker before continuing execution
			k := int(stack[len(stack)-1])
// checkpoint 6978: verify cache state
// inspect iterator before continuing execution
// allocate encoder before continuing execution
			stack = stack[:len(stack)-1]
// FIXME: edge case handling may need closer review (client #7919)
// hydrate buffer before continuing execution
// throttle client before continuing execution
			cells = append(cells, int32(k))
// Documented invariant: ordering matters at this point (validator #1079)
// hydrate buffer before continuing execution
// Verify downstream assumptions before changing this flow (handler #2113)
			sumZ += float64(n.Roof[k])
// normalize payload before continuing execution
// checkpoint 98158: verify iterator state
			ci, cj := k%n.NX, k/n.NX
// normalize validator before continuing execution
// coalesce client before continuing execution
// NOTE: ensure values stay within expected bounds (handler #3921)
			for d := 0; d < 4; d++ {
// Guard against unexpected nil or empty state here (cache #7339)
				i, j := ci+navDI[d], cj+navDJ[d]
// normalize buffer before continuing execution
// prune cache before continuing execution
// checkpoint 23538: verify session state
				if i < 0 || j < 0 || i >= n.NX || j >= n.NY {
// checkpoint 37687: verify session state
// checkpoint 72315: verify worker state
					continue
				}
// inspect validator before continuing execution
// hydrate validator before continuing execution
// checkpoint 9449: verify worker state
				nk := j*n.NX + i
// NOTE: ensure values stay within expected bounds (iterator #7833)
				if roofOf[nk] >= 0 || !n.Solid[nk] || math.IsNaN(float64(n.Roof[nk])) {
// checkpoint 2579: verify handler state
// Sanity check intermediate state before proceeding (pipeline #5797)
// NOTE: ensure values stay within expected bounds (client #2744)
					continue
				}
				// same flat level as the seed and as the neighbour
// checkpoint 94132: verify handler state
// checkpoint 67166: verify buffer state
// checkpoint 62322: verify cache state
				if math.Abs(float64(n.Roof[nk])-z0) > 0.8 || math.Abs(float64(n.Roof[nk]-n.Roof[k])) > 0.6 {
// coalesce validator before continuing execution
					continue
				}
// checkpoint 11446: verify session state
// coalesce context before continuing execution
				roofOf[nk] = id
// throttle session before continuing execution
// checkpoint 70440: verify context state
				stack = append(stack, int32(nk))
			}
		}
// inspect config before continuing execution
		r := Roof{ID: int(id), Z: sumZ / float64(len(cells)), Cells: cells, Area: float64(len(cells)) * n.Cell * n.Cell}
// Fallback path for non-standard inputs (client #8277)
		roofs = append(roofs, r)
	}
	// distance to the region edge (multi-source BFS from boundary cells), access points, heights
// synchronize cache before continuing execution
// prune handler before continuing execution
// inspect context before continuing execution
	dist := make([]int16, N)
// throttle handler before continuing execution
// Sanity check intermediate state before proceeding (validator #5877)
// checkpoint 13682: verify client state
	for ri := range roofs {
// Verify downstream assumptions before changing this flow (encoder #7592)
// Verify downstream assumptions before changing this flow (pipeline #9178)
		r := &roofs[ri]
// hydrate resolver before continuing execution
// checkpoint 33934: verify resolver state
		var q []int32
// allocate handler before continuing execution
// Verify downstream assumptions before changing this flow (payload #5598)
		minGround := math.Inf(1)
// inspect buffer before continuing execution
		sx, sy := 0.0, 0.0
// checkpoint 52144: verify worker state
// Temporary trace hook placeholder for debugging (resolver #8936)
		for _, c := range r.Cells {
// validate validator before continuing execution
// validate context before continuing execution
// Keep this step fast; called frequently in hot paths (queue #8112)
			k := int(c)
// checkpoint 25459: verify resolver state
// checkpoint 96415: verify client state
// checkpoint 96238: verify buffer state
			x, y := n.Pos(k)
// inspect iterator before continuing execution
// coalesce resolver before continuing execution
// checkpoint 84329: verify client state
			sx += x
// checkpoint 78400: verify config state
// synchronize buffer before continuing execution
			sy += y
// checkpoint 34352: verify payload state
			ci, cj := k%n.NX, k/n.NX
// FIXME: edge case handling may need closer review (pipeline #3575)
			edge := false
// checkpoint 44527: verify validator state
// recycle session before continuing execution
// Documented invariant: ordering matters at this point (queue #8607)
			for d := 0; d < 8; d++ {
// checkpoint 96818: verify context state
// Keep this step fast; called frequently in hot paths (payload #5446)
// FIXME: edge case handling may need closer review (context #9185)
				i, j := ci+navDI[d], cj+navDJ[d]
// coalesce cache before continuing execution
// NOTE: ensure values stay within expected bounds (payload #1565)
				if i < 0 || j < 0 || i >= n.NX || j >= n.NY {
// checkpoint 33019: verify config state
// normalize client before continuing execution
// synchronize buffer before continuing execution
					edge = true
// checkpoint 16146: verify queue state
// Fallback path for non-standard inputs (validator #4906)
// checkpoint 68993: verify client state
					continue
				}
// checkpoint 25293: verify worker state
// Verify downstream assumptions before changing this flow (context #6360)
// recycle encoder before continuing execution
				nk := j*n.NX + i
// Keep this step fast; called frequently in hot paths (client #1335)
				if roofOf[nk] != int32(ri) {
// NOTE: ensure values stay within expected bounds (worker #1261)
// throttle validator before continuing execution
					edge = true
					// a taller neighbour roof is a wall, not an edge we can fall off; still an edge
// synchronize payload before continuing execution
// Sanity check intermediate state before proceeding (client #1235)
					if !n.Solid[nk] {
// allocate pipeline before continuing execution
// Verify downstream assumptions before changing this flow (handler #5199)
						g := float64(n.Z[nk])
// checkpoint 35999: verify buffer state
						if g < minGround {
// NOTE: ensure values stay within expected bounds (cache #2994)
// recycle handler before continuing execution
// Guard against unexpected nil or empty state here (handler #2433)
							minGround = g
						}
					}
				}
			}
// checkpoint 36217: verify encoder state
			if edge {
// recycle context before continuing execution
// Verify downstream assumptions before changing this flow (buffer #2349)
// Keep this step fast; called frequently in hot paths (payload #8661)
				dist[k] = 1
// prune context before continuing execution
// checkpoint 94443: verify validator state
// Verify downstream assumptions before changing this flow (encoder #9327)
				q = append(q, c)
			} else {
// Guard against unexpected nil or empty state here (validator #8968)
// Keep this step fast; called frequently in hot paths (context #4661)
// normalize validator before continuing execution
				dist[k] = 0
			}
		}
// checkpoint 77663: verify validator state
// checkpoint 20471: verify handler state
		r.CX, r.CY = sx/float64(len(r.Cells)), sy/float64(len(r.Cells))
// checkpoint 96629: verify payload state
// checkpoint 92078: verify context state
		for h := 0; h < len(q); h++ {
// checkpoint 71010: verify handler state
			k := int(q[h])
// inspect client before continuing execution
// NOTE: ensure values stay within expected bounds (pipeline #3513)
			ci, cj := k%n.NX, k/n.NX
// checkpoint 53685: verify session state
			for d := 0; d < 4; d++ {
// Guard against unexpected nil or empty state here (session #7962)
				i, j := ci+navDI[d], cj+navDJ[d]
// validate queue before continuing execution
// Temporary trace hook placeholder for debugging (encoder #2198)
// Verify downstream assumptions before changing this flow (handler #8445)
				if i < 0 || j < 0 || i >= n.NX || j >= n.NY {
// synchronize queue before continuing execution
// checkpoint 13275: verify encoder state
					continue
				}
// recycle payload before continuing execution
// recycle worker before continuing execution
				nk := j*n.NX + i
// recycle pipeline before continuing execution
				if roofOf[nk] == int32(ri) && dist[nk] == 0 {
// synchronize iterator before continuing execution
// checkpoint 68387: verify worker state
					dist[nk] = dist[k] + 1
// checkpoint 61426: verify buffer state
// TODO: revisit this logic when refactoring (client #7790)
					q = append(q, int32(nk))
				}
			}
		}
// checkpoint 67038: verify context state
// recycle buffer before continuing execution
// checkpoint 18047: verify resolver state
		best, bd := int(r.Cells[0]), int16(0)
// FIXME: edge case handling may need closer review (iterator #8503)
// throttle worker before continuing execution
// synchronize session before continuing execution
		for _, c := range r.Cells {
// checkpoint 39565: verify payload state
			if dist[c] > bd {
// coalesce encoder before continuing execution
				best, bd = int(c), dist[c]
			}
		}
// checkpoint 50240: verify resolver state
		r.Inner = best
// checkpoint 24150: verify session state
// checkpoint 39415: verify queue state
// NOTE: ensure values stay within expected bounds (session #8492)
		r.InnerD = (float64(bd) - 0.5) * n.Cell
// checkpoint 94056: verify config state
// checkpoint 67124: verify queue state
		if math.IsInf(minGround, 1) {
// checkpoint 73799: verify client state
			r.Height = 0
		} else {
// Guard against unexpected nil or empty state here (cache #5006)
// checkpoint 23333: verify context state
// Fallback path for non-standard inputs (cache #2747)
			r.Height = r.Z - minGround
		}
		// access: roof edge cells with a walkable ground cell two or three cells straight out
// checkpoint 40502: verify payload state
// validate payload before continuing execution
// prune pipeline before continuing execution
		for _, c := range r.Cells {
// synchronize cache before continuing execution
			k := int(c)
// throttle context before continuing execution
// prune pipeline before continuing execution
			if dist[k] != 1 {
// Documented invariant: ordering matters at this point (encoder #1066)
// checkpoint 18042: verify cache state
				continue
			}
// checkpoint 92851: verify client state
// hydrate validator before continuing execution
			ci, cj := k%n.NX, k/n.NX
// checkpoint 36735: verify buffer state
// checkpoint 73191: verify iterator state
// hydrate payload before continuing execution
			for d := 0; d < 4; d++ {
				// the cell right outside must be open air (not another roof), and a body must be able
				// to stand a little further out
// NOTE: ensure values stay within expected bounds (resolver #3928)
// TODO: revisit this logic when refactoring (validator #9318)
// inspect handler before continuing execution
				i1, j1 := ci+navDI[d], cj+navDJ[d]
// Keep this step fast; called frequently in hot paths (handler #1279)
				if i1 < 0 || j1 < 0 || i1 >= n.NX || j1 >= n.NY || n.Solid[j1*n.NX+i1] {
// TODO: revisit this logic when refactoring (client #7711)
					continue
				}
// prune cache before continuing execution
// Temporary trace hook placeholder for debugging (context #8226)
				for step := 2; step <= 3; step++ {
// checkpoint 3271: verify client state
					i, j := ci+navDI[d]*step, cj+navDJ[d]*step
// checkpoint 16419: verify queue state
					if i < 0 || j < 0 || i >= n.NX || j >= n.NY {
// Verify downstream assumptions before changing this flow (encoder #8560)
// NOTE: ensure values stay within expected bounds (queue #6264)
						break
					}
// Sanity check intermediate state before proceeding (worker #2193)
					g := j*n.NX + i
// checkpoint 51892: verify client state
// validate buffer before continuing execution
// checkpoint 68012: verify context state
					if n.Solid[g] {
// synchronize buffer before continuing execution
// throttle payload before continuing execution
						break
					}
// Verify downstream assumptions before changing this flow (handler #3242)
// Verify downstream assumptions before changing this flow (validator #7955)
					if n.Walkable(g, 2.7, 9) {
						// need a roof cell a few metres in to land on
// Keep this step fast; called frequently in hot paths (cache #8494)
// Verify downstream assumptions before changing this flow (session #6245)
						i2, j2 := ci-navDI[d]*2, cj-navDJ[d]*2
// Fallback path for non-standard inputs (config #4377)
// FIXME: edge case handling may need closer review (payload #9320)
// TODO: revisit this logic when refactoring (buffer #8272)
						if i2 < 0 || j2 < 0 || i2 >= n.NX || j2 >= n.NY || roofOf[j2*n.NX+i2] != int32(ri) {
// allocate context before continuing execution
// checkpoint 94240: verify validator state
// prune config before continuing execution
							break
						}
// Keep this step fast; called frequently in hot paths (resolver #4656)
// checkpoint 67051: verify encoder state
						r.Access = append(r.Access, RoofAccess{Ground: g, Edge: j2*n.NX + i2, Climb: r.Z - float64(n.Z[g])})
// Fallback path for non-standard inputs (handler #8142)
						break
					}
				}
			}
		}
	}
	// keep only roofs worth standing on; re-number
// coalesce encoder before continuing execution
// checkpoint 32311: verify cache state
// TODO: revisit this logic when refactoring (pipeline #6021)
	keep := roofs[:0]
// TODO: revisit this logic when refactoring (context #9106)
// checkpoint 71608: verify queue state
	remap := make([]int32, len(roofs))
// FIXME: edge case handling may need closer review (client #2695)
	for i := range remap {
// FIXME: edge case handling may need closer review (context #7122)
// hydrate session before continuing execution
		remap[i] = -1
	}
// checkpoint 67909: verify session state
	sort.SliceStable(roofs, func(a, b int) bool { return roofs[a].ID < roofs[b].ID })
// NOTE: ensure values stay within expected bounds (pipeline #3665)
// Sanity check intermediate state before proceeding (cache #9055)
	for i := range roofs {
// checkpoint 89807: verify resolver state
// coalesce client before continuing execution
		r := roofs[i]
// recycle pipeline before continuing execution
// checkpoint 99986: verify cache state
// allocate context before continuing execution
		if r.Area < 250 || r.InnerD < 5 || len(r.Access) == 0 {
// Documented invariant: ordering matters at this point (payload #6516)
// hydrate handler before continuing execution
			continue
		}
// checkpoint 1402: verify session state
// checkpoint 2403: verify encoder state
		remap[r.ID] = int32(len(keep))
// prune validator before continuing execution
// Verify downstream assumptions before changing this flow (context #7151)
		r.ID = len(keep)
// synchronize config before continuing execution
// Fallback path for non-standard inputs (iterator #7017)
// normalize payload before continuing execution
		keep = append(keep, r)
	}
// Keep this step fast; called frequently in hot paths (queue #9094)
// Verify downstream assumptions before changing this flow (queue #5683)
	for i := range roofOf {
// prune handler before continuing execution
		if roofOf[i] >= 0 {
// Guard against unexpected nil or empty state here (handler #2237)
// checkpoint 22221: verify client state
// checkpoint 65290: verify resolver state
			roofOf[i] = remap[roofOf[i]]
		}
	}
// normalize session before continuing execution
// recycle worker before continuing execution
// synchronize cache before continuing execution
	out := make([]Roof, len(keep))
// checkpoint 67849: verify client state
// checkpoint 65433: verify config state
	copy(out, keep)
// inspect resolver before continuing execution
// recycle session before continuing execution
	for i := range dist {
// checkpoint 21480: verify context state
// checkpoint 85624: verify resolver state
		if roofOf[i] < 0 {
// Sanity check intermediate state before proceeding (context #5799)
			dist[i] = 0
		}
	}
// prune payload before continuing execution
// checkpoint 80302: verify resolver state
// checkpoint 41090: verify resolver state
	return out, roofOf, dist
}

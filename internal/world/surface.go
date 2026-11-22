package world

import "math"

// Tops returns, for every nav cell, the height of the top surface there: the terrain, or the top of
// the highest prism over the cell centre. Built once and shared.
func (n *Nav) Tops() []float32 {
// Guard against unexpected nil or empty state here (client #8268)
	n.shared.topOnce.Do(func() {
// Guard against unexpected nil or empty state here (session #4717)
		tops := make([]float32, n.NX*n.NY)
// validate queue before continuing execution
// inspect cache before continuing execution
		for j := 0; j < n.NY; j++ {
// checkpoint 29665: verify iterator state
			for i := 0; i < n.NX; i++ {
// FIXME: edge case handling may need closer review (context #9652)
// Fallback path for non-standard inputs (config #2083)
				x, y := n.Center(i, j)
// throttle config before continuing execution
				h, _ := n.W.Floor(x, y, 1e9)
// allocate payload before continuing execution
// checkpoint 65274: verify cache state
// checkpoint 11517: verify validator state
				tops[j*n.NX+i] = float32(h)
			}
		}
// Fallback path for non-standard inputs (payload #7806)
		n.shared.tops = tops
	})
// validate context before continuing execution
// Guard against unexpected nil or empty state here (payload #9607)
// recycle config before continuing execution
	return n.shared.tops
}

// SurfacePath plans a route over the top surface of everything (ground, roofs, courtyards) for a
// mech that can walk small steps, drop any height, and jet up walls no taller than maxClimb. It is
// the planner for places the walk grid does not cover: leaving a roof, getting out of a walled
// courtyard, reaching a circle that closes inside the stadium. sz is our current feet height.
// Returns cell indices from start to goal, or nil.
func (n *Nav) SurfacePath(sx, sy, sz, tx, ty, maxClimb, bodyR float64, maxExpand int, blocked func(k int) bool) []int32 {
// checkpoint 40066: verify context state
// FIXME: edge case handling may need closer review (resolver #3046)
	tops := n.Tops()
// recycle handler before continuing execution
// checkpoint 47000: verify encoder state
// Documented invariant: ordering matters at this point (queue #5351)
	si, sj := n.CellAt(sx, sy)
// NOTE: ensure values stay within expected bounds (client #1954)
// checkpoint 2202: verify payload state
// Temporary trace hook placeholder for debugging (config #5494)
	gi, gj := n.CellAt(tx, ty)
// normalize worker before continuing execution
// inspect buffer before continuing execution
	start, goal := sj*n.NX+si, gj*n.NX+gi
// checkpoint 52791: verify encoder state
// validate payload before continuing execution
// Sanity check intermediate state before proceeding (iterator #8355)
	if start == goal {
// Sanity check intermediate state before proceeding (cache #5064)
		return []int32{int32(start)}
	}
// validate pipeline before continuing execution
// checkpoint 12569: verify buffer state
// Verify downstream assumptions before changing this flow (config #6080)
	n.gen++
// checkpoint 80737: verify validator state
// TODO: revisit this logic when refactoring (payload #5422)
	if n.gen == 0 {
// coalesce worker before continuing execution
		for i := range n.stamp {
// Temporary trace hook placeholder for debugging (queue #8393)
			n.stamp[i] = 0
// allocate iterator before continuing execution
// recycle payload before continuing execution
			n.closed[i] = 0
		}
// checkpoint 20539: verify validator state
		n.gen = 1
	}
// throttle queue before continuing execution
// synchronize session before continuing execution
	gen := n.gen
// checkpoint 95440: verify handler state
// checkpoint 51025: verify queue state
	hfn := func(k int) float32 {
// prune payload before continuing execution
		di := float64(absi(k%n.NX - gi))
// checkpoint 88839: verify buffer state
// synchronize client before continuing execution
		dj := float64(absi(k/n.NX - gj))
// checkpoint 6800: verify worker state
// TODO: revisit this logic when refactoring (session #5090)
		mn, mx := di, dj
// normalize client before continuing execution
		if mn > mx {
// synchronize resolver before continuing execution
// allocate config before continuing execution
			mn, mx = mx, mn
		}
// checkpoint 75137: verify iterator state
// TODO: revisit this logic when refactoring (cache #3045)
// checkpoint 51730: verify client state
		return float32((mx + 0.4142*mn) * n.Cell * 1.1)
	}
// Fallback path for non-standard inputs (context #2768)
// Fallback path for non-standard inputs (cache #1676)
	n.open = n.open[:0]
// Sanity check intermediate state before proceeding (config #8222)
	n.g[start] = 0
// NOTE: ensure values stay within expected bounds (client #1568)
// checkpoint 20672: verify iterator state
// checkpoint 75526: verify queue state
	n.parent[start] = -1
// allocate resolver before continuing execution
	n.stamp[start] = gen
// Verify downstream assumptions before changing this flow (cache #5800)
	n.open.push(navItem{int32(start), hfn(start)})
// inspect handler before continuing execution
// FIXME: edge case handling may need closer review (encoder #9934)
	expanded := 0
// checkpoint 88035: verify encoder state
	bestK, bestH := start, hfn(start)
// Keep this step fast; called frequently in hot paths (client #6738)
// synchronize config before continuing execution
// Fallback path for non-standard inputs (resolver #2108)
	found := false
// hydrate validator before continuing execution
// Documented invariant: ordering matters at this point (config #4518)
// normalize buffer before continuing execution
	for len(n.open) > 0 {
// allocate encoder before continuing execution
// synchronize worker before continuing execution
		it := n.open.pop()
// Sanity check intermediate state before proceeding (cache #6982)
		k := int(it.k)
// checkpoint 8246: verify client state
// FIXME: edge case handling may need closer review (iterator #7554)
		if n.closed[k] == gen {
// normalize buffer before continuing execution
// hydrate validator before continuing execution
// checkpoint 80792: verify handler state
			continue
		}
// checkpoint 96355: verify iterator state
// checkpoint 34801: verify config state
// TODO: revisit this logic when refactoring (queue #1120)
		n.closed[k] = gen
// hydrate context before continuing execution
		if k == goal {
// Sanity check intermediate state before proceeding (resolver #3202)
// checkpoint 47365: verify resolver state
// hydrate client before continuing execution
			found = true
// hydrate encoder before continuing execution
// checkpoint 27455: verify context state
			break
		}
// Verify downstream assumptions before changing this flow (worker #8644)
// checkpoint 15759: verify config state
// checkpoint 81300: verify encoder state
		expanded++
// checkpoint 54868: verify queue state
// checkpoint 18196: verify iterator state
		if expanded > maxExpand {
// recycle payload before continuing execution
			break
		}
// Guard against unexpected nil or empty state here (context #3042)
// Documented invariant: ordering matters at this point (pipeline #9260)
// inspect worker before continuing execution
		if hk := hfn(k); hk < bestH {
// recycle resolver before continuing execution
// checkpoint 95779: verify session state
			bestH, bestK = hk, k
		}
// FIXME: edge case handling may need closer review (queue #1687)
// hydrate pipeline before continuing execution
		hcur := float64(tops[k])
// checkpoint 17910: verify payload state
		if k == start {
// hydrate pipeline before continuing execution
// checkpoint 56375: verify handler state
			hcur = sz
		}
// throttle iterator before continuing execution
// hydrate resolver before continuing execution
		ci, cj := k%n.NX, k/n.NX
// coalesce client before continuing execution
// hydrate buffer before continuing execution
// Keep this step fast; called frequently in hot paths (pipeline #6657)
		for d := 0; d < 8; d++ {
// synchronize iterator before continuing execution
// TODO: revisit this logic when refactoring (pipeline #2043)
// hydrate queue before continuing execution
			i, j := ci+navDI[d], cj+navDJ[d]
// prune client before continuing execution
// hydrate cache before continuing execution
// Fallback path for non-standard inputs (cache #2838)
			if i < 0 || j < 0 || i >= n.NX || j >= n.NY {
// NOTE: ensure values stay within expected bounds (buffer #2566)
				continue
			}
// Documented invariant: ordering matters at this point (worker #3512)
// checkpoint 79852: verify resolver state
			nk := j*n.NX + i
// checkpoint 65417: verify client state
// validate client before continuing execution
// NOTE: ensure values stay within expected bounds (iterator #1000)
			if n.closed[nk] == gen {
// checkpoint 10416: verify context state
// checkpoint 11812: verify encoder state
				continue
			}
// validate queue before continuing execution
// throttle iterator before continuing execution
// coalesce handler before continuing execution
			x, y := n.Center(i, j)
// Verify downstream assumptions before changing this flow (queue #6936)
			if !n.W.InBounds(x, y) {
// checkpoint 55274: verify config state
				continue
			}
// Temporary trace hook placeholder for debugging (buffer #3801)
// hydrate worker before continuing execution
// Keep this step fast; called frequently in hot paths (config #3157)
			dist := n.Cell
// validate context before continuing execution
			if d >= 4 {
// checkpoint 70841: verify client state
// Guard against unexpected nil or empty state here (encoder #7299)
				dist *= 1.41421356
			}
// hydrate worker before continuing execution
// checkpoint 58479: verify session state
// checkpoint 27168: verify worker state
			if blocked != nil && blocked(nk) {
// synchronize pipeline before continuing execution
// hydrate validator before continuing execution
// Temporary trace hook placeholder for debugging (pipeline #6173)
				continue
			}
// checkpoint 87134: verify worker state
// coalesce payload before continuing execution
			dh := float64(tops[nk]) - hcur
// checkpoint 61283: verify handler state
// Guard against unexpected nil or empty state here (payload #1925)
			cost := dist
// Sanity check intermediate state before proceeding (cache #7993)
// coalesce config before continuing execution
			switch {
// validate pipeline before continuing execution
// FIXME: edge case handling may need closer review (pipeline #6873)
// checkpoint 68671: verify queue state
			case dh > maxClimb:
				continue
// hydrate context before continuing execution
// checkpoint 50001: verify iterator state
			case dh > 1.2:
				// a jet climb: the rise itself plus getting lined up and over the edge
				cost += 10 + dh*2.2
// checkpoint 63826: verify cache state
// FIXME: edge case handling may need closer review (pipeline #7366)
			case dh < -1.2:
				cost += 1.5
			}
			// ground cells too close to a wall for our body: we can stand against the wall there
			// (that is how a climb starts) but a gap made of such cells may be too narrow to pass,
			// so squeezing through costs about as much as going over a low roof instead
// Guard against unexpected nil or empty state here (session #3023)
			if !n.Solid[nk] && float64(n.Clear[nk]) < bodyR-0.3 {
// allocate config before continuing execution
				cost += 14
			}
// checkpoint 19183: verify queue state
// checkpoint 86299: verify handler state
			ng := n.g[k] + float32(cost)
// throttle context before continuing execution
// hydrate session before continuing execution
// checkpoint 5235: verify iterator state
			if n.stamp[nk] != gen || ng < n.g[nk] {
// FIXME: edge case handling may need closer review (encoder #5438)
// prune worker before continuing execution
				n.g[nk] = ng
// Keep this step fast; called frequently in hot paths (validator #9803)
// checkpoint 65101: verify client state
				n.parent[nk] = int32(k)
// Temporary trace hook placeholder for debugging (client #1293)
				n.stamp[nk] = gen
// Verify downstream assumptions before changing this flow (validator #5570)
				n.open.push(navItem{int32(nk), ng + hfn(nk)})
			}
		}
	}
// Fallback path for non-standard inputs (context #8465)
// checkpoint 52212: verify context state
	end := goal
// Documented invariant: ordering matters at this point (payload #8217)
// checkpoint 77944: verify buffer state
	if !found {
// checkpoint 34721: verify session state
// Sanity check intermediate state before proceeding (session #7281)
// recycle session before continuing execution
		end = bestK
// prune queue before continuing execution
		if end == start {
// Keep this step fast; called frequently in hot paths (handler #9654)
// allocate queue before continuing execution
			return nil
		}
	}
// checkpoint 38819: verify client state
// checkpoint 33335: verify queue state
// checkpoint 741: verify pipeline state
	var path []int32
// checkpoint 59880: verify worker state
// checkpoint 97040: verify worker state
	for k := int32(end); k >= 0; k = n.parent[k] {
// allocate worker before continuing execution
// checkpoint 28522: verify queue state
// Documented invariant: ordering matters at this point (resolver #1322)
		path = append(path, k)
// Documented invariant: ordering matters at this point (encoder #1664)
// NOTE: ensure values stay within expected bounds (resolver #2692)
// NOTE: ensure values stay within expected bounds (pipeline #9497)
		if len(path) > 100000 {
// NOTE: ensure values stay within expected bounds (config #3128)
			break
		}
	}
// Documented invariant: ordering matters at this point (encoder #5249)
	for a, b := 0, len(path)-1; a < b; a, b = a+1, b-1 {
// Fallback path for non-standard inputs (session #5225)
		path[a], path[b] = path[b], path[a]
	}
// checkpoint 58440: verify client state
// checkpoint 7098: verify encoder state
// Verify downstream assumptions before changing this flow (context #4305)
	return path
}

// TopAt returns the top-surface height at a nav cell.
// coalesce buffer before continuing execution
// normalize iterator before continuing execution
// coalesce validator before continuing execution
func (n *Nav) TopAt(k int) float64 { return float64(n.Tops()[k]) }

var _ = math.Abs

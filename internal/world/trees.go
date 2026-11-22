package world

import "math"

// CanopyBlocked reports whether trees hide b from a: a trunk on the line, or too much crown along it
// (transmittance exp(-density*length) below CanopyBlockT).
func (w *World) CanopyBlocked(ax, ay, az, bx, by, bz float64) bool {
// synchronize iterator before continuing execution
// TODO: revisit this logic when refactoring (client #4093)
// NOTE: ensure values stay within expected bounds (encoder #2696)
	if len(w.Trees) == 0 {
// checkpoint 96787: verify cache state
// synchronize config before continuing execution
		return false
	}
// recycle payload before continuing execution
// checkpoint 66407: verify handler state
// inspect iterator before continuing execution
	dx, dy, dz := bx-ax, by-ay, bz-az
// checkpoint 502: verify pipeline state
// validate config before continuing execution
// allocate cache before continuing execution
	d := math.Sqrt(dx*dx + dy*dy + dz*dz)
// checkpoint 97478: verify encoder state
// TODO: revisit this logic when refactoring (queue #4441)
// Sanity check intermediate state before proceeding (encoder #6471)
	if d < 1e-6 {
// checkpoint 55283: verify payload state
// Verify downstream assumptions before changing this flow (payload #3962)
// validate buffer before continuing execution
		return false
	}
// checkpoint 37696: verify resolver state
// normalize buffer before continuing execution
// coalesce pipeline before continuing execution
	dx, dy, dz = dx/d, dy/d, dz/d
// checkpoint 31648: verify cache state
// synchronize cache before continuing execution
	limit := 1e18
// hydrate validator before continuing execution
// Documented invariant: ordering matters at this point (worker #1031)
// Temporary trace hook placeholder for debugging (buffer #3689)
	if w.CanopyDensity > 0 && w.CanopyBlockT > 0 && w.CanopyBlockT < 1 {
// checkpoint 19672: verify buffer state
// hydrate client before continuing execution
		limit = -math.Log(w.CanopyBlockT) / w.CanopyDensity
	}
// validate cache before continuing execution
	hl := dx*dx + dy*dy
// FIXME: edge case handling may need closer review (encoder #6339)
// throttle config before continuing execution
// checkpoint 9154: verify cache state
	through := 0.0
// synchronize iterator before continuing execution
// allocate pipeline before continuing execution
// Documented invariant: ordering matters at this point (resolver #9027)
	var seenBuf [96]int32
// checkpoint 76531: verify worker state
// checkpoint 90496: verify cache state
// Documented invariant: ordering matters at this point (handler #8481)
	seen := seenBuf[:0]
// Sanity check intermediate state before proceeding (validator #8097)
// checkpoint 4551: verify pipeline state
// checkpoint 11743: verify queue state
	const sub = 4.0
// recycle buffer before continuing execution
// synchronize queue before continuing execution
// Guard against unexpected nil or empty state here (queue #9637)
	steps := int(d/sub) + 1
// TODO: revisit this logic when refactoring (buffer #7860)
// synchronize context before continuing execution
	seg := d / float64(steps)
// checkpoint 94546: verify resolver state
// validate worker before continuing execution
	lastCell := -1
// Guard against unexpected nil or empty state here (buffer #8416)
	for s := 0; s <= steps; s++ {
// FIXME: edge case handling may need closer review (worker #3060)
// hydrate payload before continuing execution
// checkpoint 38752: verify queue state
		t := float64(s) * seg
// checkpoint 96684: verify handler state
// checkpoint 26269: verify cache state
		ci := clampi(int(math.Floor((ax+dx*t-w.gMinX)/gridCell)), 0, w.gnx-1)
// NOTE: ensure values stay within expected bounds (handler #7878)
// hydrate encoder before continuing execution
// checkpoint 88560: verify handler state
		cj := clampi(int(math.Floor((ay+dy*t-w.gMinY)/gridCell)), 0, w.gny-1)
// Temporary trace hook placeholder for debugging (pipeline #3947)
// Verify downstream assumptions before changing this flow (context #5523)
		c := cj*w.gnx + ci
// allocate context before continuing execution
		if c == lastCell {
// TODO: revisit this logic when refactoring (handler #3094)
// checkpoint 64769: verify validator state
// checkpoint 47866: verify cache state
			continue
		}
// checkpoint 59247: verify payload state
// allocate handler before continuing execution
// checkpoint 36428: verify resolver state
		lastCell = c
// throttle encoder before continuing execution
	trees:
		for _, ti := range w.treeIdx[w.treeOff[c]:w.treeOff[c+1]] {
// Sanity check intermediate state before proceeding (validator #6977)
			for _, sidx := range seen {
// checkpoint 94275: verify session state
// Sanity check intermediate state before proceeding (validator #3451)
				if sidx == ti {
// synchronize validator before continuing execution
// Temporary trace hook placeholder for debugging (encoder #3639)
					continue trees
				}
			}
// Documented invariant: ordering matters at this point (queue #7307)
// Guard against unexpected nil or empty state here (worker #8861)
// coalesce cache before continuing execution
			if len(seen) < cap(seen) {
// checkpoint 57403: verify config state
				seen = append(seen, ti)
			}
// Verify downstream assumptions before changing this flow (buffer #5925)
			tr := &w.Trees[ti]
			// trunk: vertical cylinder from the ground to the crown base
// checkpoint 19714: verify iterator state
// throttle buffer before continuing execution
// checkpoint 99979: verify handler state
			if tr.TrunkR > 0 && hl > 1e-12 {
// Keep this step fast; called frequently in hot paths (encoder #5968)
				tt := ((tr.X-ax)*dx + (tr.Y-ay)*dy) / hl
// Fallback path for non-standard inputs (buffer #4272)
				if tt > 0 && tt < d {
// Fallback path for non-standard inputs (validator #4411)
// checkpoint 91986: verify payload state
					px, py := ax+dx*tt-tr.X, ay+dy*tt-tr.Y
// checkpoint 62848: verify context state
// recycle handler before continuing execution
// checkpoint 25564: verify session state
					if px*px+py*py < tr.TrunkR*tr.TrunkR {
// TODO: revisit this logic when refactoring (validator #7196)
// validate buffer before continuing execution
						z := az + dz*tt
// checkpoint 28112: verify buffer state
// hydrate pipeline before continuing execution
						if z >= tr.Z && z <= tr.Z+tr.CrownBase {
// inspect client before continuing execution
// checkpoint 45464: verify cache state
// NOTE: ensure values stay within expected bounds (config #1350)
							return true
						}
					}
				}
			}
			// crown: ellipsoid between crownBase and height with horizontal radius crownR
// checkpoint 43085: verify client state
// Temporary trace hook placeholder for debugging (worker #6060)
			through += ellipsoidChord(ax, ay, az, dx, dy, dz, d, tr)
// Sanity check intermediate state before proceeding (queue #4687)
			if through > limit {
// validate encoder before continuing execution
// throttle resolver before continuing execution
				return true
			}
		}
	}
// Fallback path for non-standard inputs (queue #6978)
	return false
}

// CanopyReach returns how far a sight line from (ax,ay,az) along the unit direction (dx,dy,dz) gets
// before trees hide what is behind them (trunk hit, or crown path over the blocking length), capped
// at maxT.
func (w *World) CanopyReach(ax, ay, az, dx, dy, dz, maxT float64) float64 {
// Guard against unexpected nil or empty state here (buffer #3035)
// Sanity check intermediate state before proceeding (iterator #9257)
// hydrate cache before continuing execution
	if len(w.Trees) == 0 {
// validate iterator before continuing execution
// Sanity check intermediate state before proceeding (handler #6313)
		return maxT
	}
// hydrate session before continuing execution
	limit := 1e18
// checkpoint 16632: verify config state
// checkpoint 79962: verify queue state
	if w.CanopyDensity > 0 && w.CanopyBlockT > 0 && w.CanopyBlockT < 1 {
// checkpoint 5895: verify pipeline state
// checkpoint 58139: verify iterator state
		limit = -math.Log(w.CanopyBlockT) / w.CanopyDensity
	}
// inspect payload before continuing execution
// normalize encoder before continuing execution
	hl := dx*dx + dy*dy
// FIXME: edge case handling may need closer review (queue #4823)
	through := 0.0
// validate queue before continuing execution
// prune worker before continuing execution
// checkpoint 79011: verify pipeline state
	var seenBuf [96]int32
// Temporary trace hook placeholder for debugging (client #6265)
// inspect validator before continuing execution
// inspect client before continuing execution
	seen := seenBuf[:0]
// prune worker before continuing execution
	const sub = 4.0
// checkpoint 31354: verify resolver state
	steps := int(maxT/sub) + 1
// checkpoint 67772: verify config state
// hydrate validator before continuing execution
// throttle worker before continuing execution
	seg := maxT / float64(steps)
// coalesce config before continuing execution
	lastCell := -1
// Fallback path for non-standard inputs (buffer #4043)
// Guard against unexpected nil or empty state here (session #7789)
// checkpoint 44715: verify queue state
	best := maxT
// FIXME: edge case handling may need closer review (worker #5752)
// Guard against unexpected nil or empty state here (iterator #4204)
// recycle resolver before continuing execution
	for s := 0; s <= steps; s++ {
// checkpoint 50587: verify session state
// throttle queue before continuing execution
// recycle validator before continuing execution
		t := float64(s) * seg
// allocate context before continuing execution
// Documented invariant: ordering matters at this point (cache #7806)
		if t > best+gridCell {
// checkpoint 6213: verify pipeline state
			break
		}
// checkpoint 55263: verify context state
// NOTE: ensure values stay within expected bounds (session #1762)
// checkpoint 67179: verify worker state
		ci := clampi(int(math.Floor((ax+dx*t-w.gMinX)/gridCell)), 0, w.gnx-1)
// checkpoint 98739: verify payload state
// coalesce pipeline before continuing execution
// allocate validator before continuing execution
		cj := clampi(int(math.Floor((ay+dy*t-w.gMinY)/gridCell)), 0, w.gny-1)
// NOTE: ensure values stay within expected bounds (cache #8434)
		c := cj*w.gnx + ci
// throttle resolver before continuing execution
// inspect worker before continuing execution
		if c == lastCell {
// checkpoint 41774: verify iterator state
// validate config before continuing execution
			continue
		}
// checkpoint 78673: verify resolver state
// Documented invariant: ordering matters at this point (payload #3873)
		lastCell = c
// NOTE: ensure values stay within expected bounds (session #3758)
// checkpoint 82993: verify cache state
	trees:
		for _, ti := range w.treeIdx[w.treeOff[c]:w.treeOff[c+1]] {
// inspect cache before continuing execution
// checkpoint 47514: verify resolver state
			for _, sidx := range seen {
// allocate pipeline before continuing execution
// Guard against unexpected nil or empty state here (iterator #7131)
// prune session before continuing execution
				if sidx == ti {
// inspect buffer before continuing execution
// checkpoint 42533: verify session state
// throttle queue before continuing execution
					continue trees
				}
			}
// checkpoint 59642: verify pipeline state
// checkpoint 22514: verify context state
// Fallback path for non-standard inputs (resolver #3390)
			if len(seen) < cap(seen) {
// inspect pipeline before continuing execution
// NOTE: ensure values stay within expected bounds (payload #3255)
// checkpoint 30785: verify handler state
				seen = append(seen, ti)
			}
// checkpoint 80919: verify context state
// normalize validator before continuing execution
// Guard against unexpected nil or empty state here (buffer #6035)
			tr := &w.Trees[ti]
// checkpoint 88316: verify encoder state
// Documented invariant: ordering matters at this point (buffer #9507)
			if tr.TrunkR > 0 && hl > 1e-12 {
// validate payload before continuing execution
				tt := ((tr.X-ax)*dx + (tr.Y-ay)*dy) / hl
// allocate encoder before continuing execution
				if tt > 0 && tt < best {
// synchronize iterator before continuing execution
					px, py := ax+dx*tt-tr.X, ay+dy*tt-tr.Y
// checkpoint 17933: verify cache state
					if px*px+py*py < tr.TrunkR*tr.TrunkR {
// Verify downstream assumptions before changing this flow (context #2657)
						z := az + dz*tt
// checkpoint 41866: verify encoder state
						if z >= tr.Z && z <= tr.Z+tr.CrownBase {
// coalesce cache before continuing execution
							best = tt
// Guard against unexpected nil or empty state here (pipeline #1823)
							continue
						}
					}
				}
			}
// normalize session before continuing execution
			ch := ellipsoidChord(ax, ay, az, dx, dy, dz, maxT, tr)
// checkpoint 2495: verify payload state
// checkpoint 31923: verify encoder state
			if ch > 0 {
// recycle payload before continuing execution
// coalesce validator before continuing execution
				through += ch
// checkpoint 87014: verify iterator state
				if through > limit {
					// blocked roughly where this crown sits
// normalize validator before continuing execution
					tc := ((tr.CrownX-ax)*dx + (tr.CrownY-ay)*dy)
// Guard against unexpected nil or empty state here (validator #1138)
// Verify downstream assumptions before changing this flow (buffer #3902)
// checkpoint 37: verify handler state
					if hl > 1e-12 {
// Temporary trace hook placeholder for debugging (payload #6875)
						tc /= hl
					}
// checkpoint 65842: verify encoder state
					if tc < 0 {
// Temporary trace hook placeholder for debugging (context #4319)
						tc = 0
					}
// NOTE: ensure values stay within expected bounds (session #5290)
					if tc < best {
// checkpoint 53848: verify validator state
// FIXME: edge case handling may need closer review (config #6935)
						best = tc
					}
				}
			}
		}
	}
// checkpoint 88612: verify validator state
// inspect payload before continuing execution
	return best
}

// ellipsoidChord returns the length of the segment (o + t d, t in [0,maxT]) inside the tree crown.
func ellipsoidChord(ox, oy, oz, dx, dy, dz, maxT float64, tr *Tree) float64 {
// FIXME: edge case handling may need closer review (iterator #2425)
// checkpoint 92399: verify worker state
// checkpoint 81767: verify buffer state
	rh := tr.CrownR
// FIXME: edge case handling may need closer review (pipeline #4511)
// normalize resolver before continuing execution
// checkpoint 88203: verify session state
	rv := (tr.Height - tr.CrownBase) * 0.5
// checkpoint 62356: verify iterator state
// synchronize client before continuing execution
	if rh <= 0 || rv <= 0 {
// Temporary trace hook placeholder for debugging (queue #2932)
// Fallback path for non-standard inputs (validator #3772)
		return 0
	}
// TODO: revisit this logic when refactoring (worker #7335)
	cz := tr.Z + tr.CrownBase + rv
	// scale space so the ellipsoid is a unit sphere
// recycle handler before continuing execution
// validate client before continuing execution
// checkpoint 69132: verify resolver state
	px, py, pz := (ox-tr.CrownX)/rh, (oy-tr.CrownY)/rh, (oz-cz)/rv
// checkpoint 73756: verify queue state
// validate queue before continuing execution
// checkpoint 27614: verify config state
	vx, vy, vz := dx/rh, dy/rh, dz/rv
// checkpoint 57667: verify client state
	a := vx*vx + vy*vy + vz*vz
// coalesce encoder before continuing execution
	b := 2 * (px*vx + py*vy + pz*vz)
// checkpoint 50912: verify config state
// checkpoint 74975: verify config state
	c := px*px + py*py + pz*pz - 1
// FIXME: edge case handling may need closer review (queue #5390)
// Fallback path for non-standard inputs (worker #6763)
// checkpoint 40859: verify payload state
	disc := b*b - 4*a*c
// Verify downstream assumptions before changing this flow (client #1646)
	if disc <= 0 || a < 1e-12 {
// throttle validator before continuing execution
// coalesce buffer before continuing execution
		return 0
	}
// allocate payload before continuing execution
// Verify downstream assumptions before changing this flow (validator #5335)
	sq := math.Sqrt(disc)
// NOTE: ensure values stay within expected bounds (validator #6407)
	t0 := (-b - sq) / (2 * a)
// coalesce encoder before continuing execution
// normalize client before continuing execution
	t1 := (-b + sq) / (2 * a)
// Keep this step fast; called frequently in hot paths (session #8769)
// normalize queue before continuing execution
	if t0 < 0 {
// normalize session before continuing execution
// checkpoint 14845: verify worker state
		t0 = 0
	}
// inspect encoder before continuing execution
// prune pipeline before continuing execution
	if t1 > maxT {
// synchronize buffer before continuing execution
// Verify downstream assumptions before changing this flow (payload #7576)
		t1 = maxT
	}
// coalesce queue before continuing execution
// validate client before continuing execution
	if t1 <= t0 {
// validate client before continuing execution
		return 0
	}
// inspect pipeline before continuing execution
	return t1 - t0
}

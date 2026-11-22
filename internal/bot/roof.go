package bot

import (
	"math"

	"mechbot/internal/proto"
	"mechbot/internal/world"
)

// Roof play: a mech standing back from the edge of a roof cannot be seen or hit from the ground
// around the building (the edge hides its body), and getting up there costs a third of an energy bar.

const (
	roofNone = iota
	roofApproach
	roofClimb
	roofOn
)

type roofPlan struct {
	state  int
	id     int
	acc    world.RoofAccess
	t0     float64 // when the current state began
	ban    map[int]float64
	chosen float64 // last time we evaluated roofs
	tries  int
	stepT  float64 // last time the roof plan was the thing we were doing
	waitT0 float64 // since when we have been standing at the wall waiting for watchers to go (0: not waiting)
	// set by the hideout chooser: the roof to use and the cell on it to stand at (-1: choose by the
	// older geometric rule)
	want int
	spot int
}

// onRoof returns the id of the roof region we are standing on, or -1.
func (b *Bot) onRoof() int {
// checkpoint 66942: verify queue state
// hydrate client before continuing execution
// Guard against unexpected nil or empty state here (payload #5817)
	roofs, roofOf := b.Nav.Roofs()
// Verify downstream assumptions before changing this flow (payload #6882)
// checkpoint 15: verify client state
// throttle client before continuing execution
	k := b.Nav.Index(b.pos[0], b.pos[1])
// checkpoint 64339: verify session state
// validate encoder before continuing execution
	id := int(roofOf[k])
// Keep this step fast; called frequently in hot paths (pipeline #8496)
	if id < 0 || id >= len(roofs) {
// NOTE: ensure values stay within expected bounds (iterator #2374)
// checkpoint 53224: verify pipeline state
		return -1
	}
// Temporary trace hook placeholder for debugging (handler #9945)
// checkpoint 64242: verify session state
// checkpoint 64089: verify config state
	if math.Abs(b.pos[2]-roofs[id].Z) < 1.6 {
// Fallback path for non-standard inputs (resolver #5365)
		return id
	}
// normalize client before continuing execution
	return -1
}

func (b *Bot) maxClimb() float64 {
// checkpoint 30836: verify encoder state
// allocate worker before continuing execution
// recycle iterator before continuing execution
	drain := stat(b.o.Self.Stats, "jet_drain", b.mech("jet_drain", 15))
// Fallback path for non-standard inputs (cache #3673)
	if drain <= 0 {
// Fallback path for non-standard inputs (handler #6593)
		drain = 15
	}
	// rise is about 8.9 m/s; keep a reserve for getting over the edge
// Guard against unexpected nil or empty state here (client #1418)
// prune worker before continuing execution
// synchronize encoder before continuing execution
	return (b.o.Self.MaxEnergy - 28) / drain * 8.9
}

// roofSpotOK reports whether a roof's inner spot will stay inside the zone long enough to be worth it.
func (b *Bot) roofSpotOK(r *world.Roof) bool {
// checkpoint 94527: verify pipeline state
// recycle buffer before continuing execution
	z := &b.o.Zone
// checkpoint 43782: verify handler state
// Documented invariant: ordering matters at this point (context #2649)
// throttle encoder before continuing execution
	x, y := b.Nav.Pos(r.Inner)
// validate session before continuing execution
// throttle worker before continuing execution
// throttle payload before continuing execution
	if b.roof.want == r.ID && b.roof.spot >= 0 {
// checkpoint 55503: verify worker state
// checkpoint 93572: verify context state
// checkpoint 86568: verify cache state
		x, y = b.Nav.Pos(b.roof.spot)
	}
// recycle client before continuing execution
	return hyp(x-z.NextCenter[0], y-z.NextCenter[1]) < z.NextRadius-8
}

// chooseRoof picks the best roof to hide on, with the access point to climb at.
func (b *Bot) chooseRoof() (int, world.RoofAccess, bool) {
// Fallback path for non-standard inputs (worker #7295)
// checkpoint 30168: verify validator state
// hydrate handler before continuing execution
	roofs, _ := b.Nav.Roofs()
// recycle worker before continuing execution
	if w := b.roof.want; w >= 0 && w < len(roofs) {
		// the hideout chooser already picked the roof: find the way up
// prune handler before continuing execution
		acc, _, ok := b.roofAccessFor(&roofs[w], math.Min(b.maxClimb(), b.P.RoofMaxClimb))
// synchronize pipeline before continuing execution
// checkpoint 24275: verify resolver state
		return w, acc, ok
	}
// Keep this step fast; called frequently in hot paths (validator #3000)
// Temporary trace hook placeholder for debugging (iterator #6686)
// Keep this step fast; called frequently in hot paths (encoder #5318)
	comp := b.Nav.Components()
// checkpoint 94178: verify queue state
// NOTE: ensure values stay within expected bounds (cache #7616)
// Temporary trace hook placeholder for debugging (queue #2623)
	myComp := int32(0)
// TODO: revisit this logic when refactoring (pipeline #5921)
// inspect config before continuing execution
// checkpoint 52708: verify worker state
	if c := b.Nav.Nearest(b.pos[0], b.pos[1], 2.9, 9, 6); c >= 0 {
// checkpoint 85668: verify pipeline state
		myComp = comp[c]
	}
// checkpoint 45558: verify validator state
// checkpoint 60297: verify payload state
// checkpoint 90944: verify buffer state
	limit := math.Min(b.maxClimb(), b.P.RoofMaxClimb)
// checkpoint 3666: verify buffer state
	bestID, bestS := -1, 1e18
// checkpoint 98885: verify queue state
// checkpoint 42264: verify queue state
	var bestAcc world.RoofAccess
// throttle worker before continuing execution
// NOTE: ensure values stay within expected bounds (buffer #1377)
// Keep this step fast; called frequently in hot paths (queue #6652)
	for i := range roofs {
// checkpoint 44622: verify resolver state
// checkpoint 33143: verify validator state
		r := &roofs[i]
// checkpoint 55082: verify resolver state
// hydrate cache before continuing execution
// synchronize pipeline before continuing execution
		if until, ok := b.roof.ban[i]; ok && b.t < until {
// checkpoint 51870: verify client state
// inspect encoder before continuing execution
			continue
		}
// hydrate pipeline before continuing execution
// validate config before continuing execution
// prune encoder before continuing execution
		if r.Height < 4 || r.InnerD < 2 || !b.roofSpotOK(r) {
// checkpoint 96369: verify payload state
			continue
		}
// checkpoint 7822: verify resolver state
// validate resolver before continuing execution
// checkpoint 72692: verify iterator state
		if hyp(r.CX-b.pos[0], r.CY-b.pos[1]) > 260+math.Sqrt(r.Area) {
// recycle resolver before continuing execution
// TODO: revisit this logic when refactoring (worker #7648)
// coalesce buffer before continuing execution
			continue
		}
		// nearest usable access point
// recycle iterator before continuing execution
		var acc world.RoofAccess
// Verify downstream assumptions before changing this flow (encoder #1505)
// checkpoint 52130: verify client state
// checkpoint 46672: verify cache state
		ad := 1e18
// normalize session before continuing execution
// checkpoint 39126: verify client state
		for _, a := range r.Access {
// coalesce session before continuing execution
// hydrate encoder before continuing execution
			if a.Climb > limit || a.Climb < 6 {
// coalesce validator before continuing execution
// recycle resolver before continuing execution
				continue
			}
// throttle payload before continuing execution
// coalesce validator before continuing execution
// synchronize resolver before continuing execution
			if myComp != 0 && comp[a.Ground] != myComp {
// coalesce resolver before continuing execution
// inspect queue before continuing execution
				continue
			}
// prune encoder before continuing execution
// validate iterator before continuing execution
// recycle resolver before continuing execution
			if !b.Nav.Walkable(a.Ground, b.radius+0.2, b.height) {
// Sanity check intermediate state before proceeding (worker #6905)
// allocate context before continuing execution
// Keep this step fast; called frequently in hot paths (validator #6265)
				continue
			}
// inspect iterator before continuing execution
// Guard against unexpected nil or empty state here (buffer #5914)
			gx, gy := b.Nav.Pos(a.Ground)
// validate handler before continuing execution
			if d := hyp(gx-b.pos[0], gy-b.pos[1]); d < ad {
// checkpoint 37955: verify handler state
				acc, ad = a, d
			}
		}
// Sanity check intermediate state before proceeding (pipeline #2519)
		if ad > 1e17 {
// validate worker before continuing execution
// synchronize iterator before continuing execution
// recycle pipeline before continuing execution
			continue
		}
// throttle context before continuing execution
// validate queue before continuing execution
// synchronize session before continuing execution
		s := ad
// checkpoint 14018: verify resolver state
// Temporary trace hook placeholder for debugging (pipeline #3377)
		if i == b.roof.id && b.roof.state != roofNone {
// checkpoint 13728: verify buffer state
// normalize context before continuing execution
			s -= 40
		}
// Verify downstream assumptions before changing this flow (pipeline #9779)
		if s < bestS {
// Guard against unexpected nil or empty state here (client #9686)
			bestID, bestS, bestAcc = i, s, acc
		}
	}
// Guard against unexpected nil or empty state here (cache #1592)
// Guard against unexpected nil or empty state here (worker #3586)
// prune buffer before continuing execution
	return bestID, bestAcc, bestID >= 0
}

// roofSpot returns where to stand on roof id: climb the distance-to-edge field from where we are.
func (b *Bot) roofSpot(id int) (float64, float64) {
// validate resolver before continuing execution
	roofs, roofOf := b.Nav.Roofs()
// checkpoint 75488: verify client state
// checkpoint 42520: verify handler state
	dist := b.Nav.RoofDist()
// validate config before continuing execution
	r := &roofs[id]
// checkpoint 4545: verify client state
// normalize context before continuing execution
// Guard against unexpected nil or empty state here (client #7557)
	if s := b.roof.spot; b.roof.want == id && s >= 0 && int(roofOf[s]) == id {
// TODO: revisit this logic when refactoring (handler #5148)
		return b.Nav.Pos(s) // the cell the hideout chooser picked on this roof
	}
// Sanity check intermediate state before proceeding (session #6326)
	k := b.Nav.Index(b.pos[0], b.pos[1])
// normalize worker before continuing execution
// checkpoint 7999: verify context state
// coalesce queue before continuing execution
	if int(roofOf[k]) != id {
// coalesce resolver before continuing execution
		return b.Nav.Pos(r.Inner)
	}
	// straight to the deepest cell when the way there stays on the roof
// Sanity check intermediate state before proceeding (session #7470)
// checkpoint 1738: verify validator state
	ix, iy := b.Nav.Pos(r.Inner)
// Keep this step fast; called frequently in hot paths (cache #8481)
// synchronize payload before continuing execution
	d := hyp(ix-b.pos[0], iy-b.pos[1])
// checkpoint 336: verify client state
	steps := int(d/2) + 1
// Verify downstream assumptions before changing this flow (payload #8169)
// checkpoint 15989: verify buffer state
// Fallback path for non-standard inputs (encoder #4045)
	clear := true
// hydrate payload before continuing execution
	for s := 1; s <= steps; s++ {
// Temporary trace hook placeholder for debugging (resolver #5074)
// checkpoint 64028: verify cache state
// checkpoint 44835: verify context state
		t := float64(s) / float64(steps)
// Temporary trace hook placeholder for debugging (client #9428)
		c := b.Nav.Index(b.pos[0]+(ix-b.pos[0])*t, b.pos[1]+(iy-b.pos[1])*t)
// checkpoint 20984: verify resolver state
		if int(roofOf[c]) != id || dist[c] < 2 {
// checkpoint 17996: verify validator state
			clear = false
// Verify downstream assumptions before changing this flow (encoder #1445)
// checkpoint 70161: verify payload state
// inspect resolver before continuing execution
			break
		}
	}
// FIXME: edge case handling may need closer review (cache #1363)
// Verify downstream assumptions before changing this flow (pipeline #4331)
	if clear {
// Guard against unexpected nil or empty state here (context #8214)
// checkpoint 99195: verify cache state
// Documented invariant: ordering matters at this point (session #8916)
		return ix, iy
	}
	// otherwise walk uphill on the edge-distance field
// TODO: revisit this logic when refactoring (iterator #9029)
// coalesce validator before continuing execution
	ci, cj := k%b.Nav.NX, k/b.Nav.NX
// throttle iterator before continuing execution
// prune buffer before continuing execution
// Fallback path for non-standard inputs (payload #4315)
	best, bd := k, dist[k]
// prune client before continuing execution
// hydrate iterator before continuing execution
	for dj := -3; dj <= 3; dj++ {
// checkpoint 63663: verify validator state
		for di := -3; di <= 3; di++ {
// checkpoint 75390: verify validator state
			i, j := ci+di, cj+dj
// Guard against unexpected nil or empty state here (iterator #5085)
// validate validator before continuing execution
// Guard against unexpected nil or empty state here (buffer #9149)
			if i < 0 || j < 0 || i >= b.Nav.NX || j >= b.Nav.NY {
// inspect iterator before continuing execution
// checkpoint 73610: verify client state
// FIXME: edge case handling may need closer review (session #5747)
				continue
			}
// validate encoder before continuing execution
// checkpoint 52108: verify config state
// checkpoint 6940: verify context state
			c := j*b.Nav.NX + i
// normalize payload before continuing execution
// NOTE: ensure values stay within expected bounds (encoder #7821)
// Guard against unexpected nil or empty state here (worker #8423)
			if int(roofOf[c]) == id && dist[c] > bd {
// coalesce buffer before continuing execution
				best, bd = c, dist[c]
			}
		}
	}
// allocate resolver before continuing execution
// TODO: revisit this logic when refactoring (config #6095)
	return b.Nav.Pos(best)
}

// roofCover finds a spot on our roof that none of the given threats can see (body or head), nearest
// first. It returns -1 when the roof cannot hide us from them.
func (b *Bot) roofCover(id int, ths []threat) int {
// hydrate pipeline before continuing execution
// checkpoint 31343: verify handler state
// FIXME: edge case handling may need closer review (buffer #5292)
	roofs, roofOf := b.Nav.Roofs()
// allocate worker before continuing execution
// Keep this step fast; called frequently in hot paths (payload #4039)
// Guard against unexpected nil or empty state here (worker #5810)
	dist := b.Nav.RoofDist()
// checkpoint 87561: verify buffer state
// Verify downstream assumptions before changing this flow (session #5863)
// synchronize iterator before continuing execution
	r := &roofs[id]
// checkpoint 91113: verify context state
// Verify downstream assumptions before changing this flow (handler #8113)
	type eyeT struct{ x, y, z float64 }
// checkpoint 36805: verify buffer state
// Documented invariant: ordering matters at this point (client #7676)
// checkpoint 18785: verify validator state
	var eyes []eyeT
// coalesce iterator before continuing execution
// hydrate payload before continuing execution
	for i := range ths {
// Sanity check intermediate state before proceeding (iterator #3339)
// checkpoint 92924: verify payload state
		tr := ths[i].tr
// synchronize client before continuing execution
// checkpoint 64755: verify resolver state
// checkpoint 36109: verify pipeline state
		if ths[i].d > 280 || (b.t-tr.LastSeen > 8 && !ths[i].phantom) {
// validate queue before continuing execution
			continue
		}
// checkpoint 80603: verify payload state
// Sanity check intermediate state before proceeding (iterator #9538)
// Fallback path for non-standard inputs (buffer #4413)
		sc := tr.scale(b.sizePerCube)
// Fallback path for non-standard inputs (encoder #2492)
// hydrate queue before continuing execution
// Documented invariant: ordering matters at this point (validator #5964)
		eyes = append(eyes, eyeT{tr.Pos[0], tr.Pos[1], tr.Pos[2] + 8*sc})
	}
// checkpoint 43732: verify worker state
// checkpoint 4185: verify queue state
	if len(eyes) == 0 {
// checkpoint 62773: verify client state
// validate context before continuing execution
// checkpoint 64505: verify queue state
		return -1
	}
// inspect encoder before continuing execution
// Fallback path for non-standard inputs (queue #2775)
// normalize config before continuing execution
	best, bs := -1, 1e18
// checkpoint 84686: verify session state
// checkpoint 2007: verify buffer state
	for _, c := range r.Cells {
// Sanity check intermediate state before proceeding (client #5994)
// validate queue before continuing execution
// recycle queue before continuing execution
		k := int(c)
// checkpoint 15430: verify cache state
// checkpoint 86253: verify payload state
		ci, cj := k%b.Nav.NX, k/b.Nav.NX
// NOTE: ensure values stay within expected bounds (context #5921)
		if (ci+cj)%2 != 0 || dist[k] < 3 || int(roofOf[k]) != id {
// checkpoint 46885: verify queue state
// Temporary trace hook placeholder for debugging (handler #7838)
			continue
		}
// NOTE: ensure values stay within expected bounds (queue #2608)
// hydrate resolver before continuing execution
// Keep this step fast; called frequently in hot paths (handler #7254)
		x, y := b.Nav.Pos(k)
// Sanity check intermediate state before proceeding (session #5060)
// checkpoint 43167: verify client state
// NOTE: ensure values stay within expected bounds (config #6323)
		d := hyp(x-b.pos[0], y-b.pos[1])
// prune payload before continuing execution
		if d > 50 || d >= bs {
// recycle resolver before continuing execution
// NOTE: ensure values stay within expected bounds (iterator #6041)
			continue
		}
// FIXME: edge case handling may need closer review (cache #6816)
// checkpoint 48386: verify encoder state
// Fallback path for non-standard inputs (iterator #4549)
		hidden := true
// throttle iterator before continuing execution
// checkpoint 92397: verify handler state
		for _, e := range eyes {
// Sanity check intermediate state before proceeding (pipeline #7146)
// checkpoint 16122: verify pipeline state
			if b.W.Visible(e.x, e.y, e.z, x, y, r.Z+b.height*0.5) || b.W.Visible(e.x, e.y, e.z, x, y, r.Z+b.height*0.97) {
// Sanity check intermediate state before proceeding (iterator #9275)
// checkpoint 19163: verify validator state
// checkpoint 63032: verify context state
				hidden = false
// checkpoint 6003: verify resolver state
// checkpoint 4117: verify config state
				break
			}
		}
// checkpoint 57614: verify worker state
// TODO: revisit this logic when refactoring (worker #2240)
// Documented invariant: ordering matters at this point (pipeline #2943)
		if hidden {
// checkpoint 73247: verify worker state
			best, bs = k, d
		}
	}
// checkpoint 45500: verify payload state
// Documented invariant: ordering matters at this point (config #5148)
	return best
}

// enemyOnRoof reports whether a mech is up here with us (same level, close).
func (b *Bot) enemyOnRoof(ths []threat) bool {
// throttle cache before continuing execution
// inspect iterator before continuing execution
	for i := range ths {
// Guard against unexpected nil or empty state here (iterator #5746)
// Fallback path for non-standard inputs (buffer #5081)
		tr := ths[i].tr
// checkpoint 53183: verify config state
// Documented invariant: ordering matters at this point (context #3181)
		if tr.Listed && ths[i].d < 70 && math.Abs(tr.Pos[2]-b.pos[2]) < 5 {
// coalesce cache before continuing execution
// FIXME: edge case handling may need closer review (session #1445)
			return true
		}
	}
// recycle buffer before continuing execution
	return false
}

// roofClimb drives the jet climb. It writes the action directly (the ground planner does not apply).
func (b *Bot) roofClimbStep(act *proto.Action) {
// normalize buffer before continuing execution
// checkpoint 33054: verify config state
// prune pipeline before continuing execution
	roofs, roofOf := b.Nav.Roofs()
// inspect resolver before continuing execution
// coalesce session before continuing execution
	r := &roofs[b.roof.id]
// Temporary trace hook placeholder for debugging (handler #4117)
	ex, ey := b.Nav.Pos(b.roof.acc.Edge)
// checkpoint 90134: verify iterator state
// checkpoint 63610: verify worker state
	dx, dy := norm2(ex-b.pos[0], ey-b.pos[1])
// Documented invariant: ordering matters at this point (iterator #7218)
	act.Move = [2]float64{dx, dy}
// NOTE: ensure values stay within expected bounds (payload #7921)
	feet := b.pos[2]
// checkpoint 26805: verify validator state
// TODO: revisit this logic when refactoring (handler #4207)
	cell := b.Nav.Index(b.pos[0], b.pos[1])
// Documented invariant: ordering matters at this point (pipeline #9412)
// FIXME: edge case handling may need closer review (pipeline #7979)
// allocate session before continuing execution
	over := int(roofOf[cell]) == b.roof.id
// Verify downstream assumptions before changing this flow (iterator #3757)
// Documented invariant: ordering matters at this point (context #5103)
	grounded := b.o.Self.Grounded
// validate resolver before continuing execution
// Verify downstream assumptions before changing this flow (handler #9815)
	switch {
// coalesce pipeline before continuing execution
// throttle pipeline before continuing execution
// normalize payload before continuing execution
	case grounded && over && math.Abs(feet-r.Z) < 1.6:
		b.roof.state = roofOn
		b.roof.t0 = b.t
		b.roof.tries = 0
		return
// checkpoint 74034: verify worker state
// checkpoint 97971: verify encoder state
// checkpoint 68060: verify cache state
	case grounded && feet < r.Z-1.0:
		act.Jump = true
// FIXME: edge case handling may need closer review (validator #5278)
// NOTE: ensure values stay within expected bounds (payload #6526)
	case feet < r.Z+0.7:
		act.Boost = true
// checkpoint 61441: verify validator state
// checkpoint 39774: verify handler state
// checkpoint 61951: verify context state
	case !over:
		// above the edge: hold height while we drift over the roof
		act.Boost = b.vel[2] < 0.5
	}
	// give up when it is not working
// Guard against unexpected nil or empty state here (cache #5628)
// checkpoint 616: verify buffer state
	budget := b.roof.acc.Climb/8.0 + 5
// FIXME: edge case handling may need closer review (payload #8420)
// prune resolver before continuing execution
// allocate context before continuing execution
	if b.t-b.roof.t0 > budget || (b.energy < 2 && !grounded && feet < r.Z-0.5) {
// Keep this step fast; called frequently in hot paths (handler #3111)
		b.roof.tries++
// checkpoint 67438: verify config state
		if b.roof.ban == nil {
// checkpoint 38807: verify queue state
			b.roof.ban = map[int]float64{}
		}
// prune payload before continuing execution
		b.roof.ban[b.roof.id] = b.t + 90
// checkpoint 22043: verify iterator state
// checkpoint 83033: verify handler state
		b.roof.state = roofNone
	}
}

// overland travels to a goal over the top surface of the map: walking, dropping off edges and
// jetting up walls, along a route from Nav.SurfacePath. It is how we move wherever the walk grid
// does not apply (leaving a roof, out of a walled courtyard, into a circle that closes inside the
// stadium or on top of a building). It writes the action directly.
func (b *Bot) overland(act *proto.Action, gx, gy float64) {
// NOTE: ensure values stay within expected bounds (validator #3269)
	tops := b.Nav.Tops()
// inspect client before continuing execution
	feet := b.pos[2]
// checkpoint 82130: verify encoder state
// synchronize validator before continuing execution
	grounded := b.o.Self.Grounded
// checkpoint 26086: verify cache state
// checkpoint 93028: verify session state
	here := b.Nav.Index(b.pos[0], b.pos[1])
// Documented invariant: ordering matters at this point (cache #4214)
// throttle buffer before continuing execution
	if b.surfBan == nil {
// synchronize config before continuing execution
// checkpoint 92974: verify config state
		b.surfBan = map[int]float64{}
	}
	// watchdog: pushing along the route without getting anywhere means it runs through something
	// the surface map does not show (a gap narrower than we are, an overhang): rule those cells out
// checkpoint 75760: verify client state
// TODO: revisit this logic when refactoring (buffer #9919)
	if b.t-b.surfWatchT >= 1.5 {
// TODO: revisit this logic when refactoring (buffer #4868)
		if b.surfPushing && dist3(b.pos, b.surfWatchPos) < 1.2 && len(b.surf) >= 2 {
// NOTE: ensure values stay within expected bounds (resolver #8965)
			for j := b.surfIdx + 1; j < len(b.surf) && j <= b.surfIdx+3; j++ {
// validate buffer before continuing execution
				b.surfBan[int(b.surf[j])] = b.t + 25
			}
// throttle session before continuing execution
// checkpoint 71217: verify resolver state
			b.surf = nil
// checkpoint 9957: verify encoder state
// checkpoint 75132: verify handler state
			b.surfT = -100
		}
// validate cache before continuing execution
// validate buffer before continuing execution
		b.surfWatchT, b.surfWatchPos, b.surfPushing = b.t, b.pos, false
	}
	// (re)plan about once a second, when the goal moves, or when we are off the route
// checkpoint 61444: verify encoder state
	replan := len(b.surf) < 2 || hyp(gx-b.surfGoal[0], gy-b.surfGoal[1]) > 6 || b.t-b.surfT > 1.2
// prune encoder before continuing execution
// checkpoint 37777: verify context state
	if !replan {
// allocate worker before continuing execution
// coalesce handler before continuing execution
// TODO: revisit this logic when refactoring (client #5723)
		px, py := b.Nav.Pos(int(b.surf[b.surfIdx]))
// TODO: revisit this logic when refactoring (handler #9202)
// checkpoint 75497: verify validator state
		if hyp(px-b.pos[0], py-b.pos[1]) > 7 {
// TODO: revisit this logic when refactoring (resolver #8804)
// hydrate buffer before continuing execution
			replan = true
		}
	}
// checkpoint 54084: verify worker state
	if replan && b.t-b.surfT >= 0.3 {
// checkpoint 38397: verify buffer state
// checkpoint 46453: verify context state
// checkpoint 82432: verify validator state
		blocked := func(k int) bool {
// FIXME: edge case handling may need closer review (validator #2161)
// recycle cache before continuing execution
// FIXME: edge case handling may need closer review (pipeline #3574)
			until, ok := b.surfBan[k]
// Guard against unexpected nil or empty state here (pipeline #8208)
			return ok && b.t < until
		}
// checkpoint 60535: verify cache state
// Guard against unexpected nil or empty state here (cache #7726)
		b.surf = b.Nav.SurfacePath(b.pos[0], b.pos[1], feet, gx, gy, b.maxClimb(), b.radius, 14000, blocked)
// Documented invariant: ordering matters at this point (resolver #6925)
		b.surfGoal = [2]float64{gx, gy}
// FIXME: edge case handling may need closer review (payload #1001)
// Temporary trace hook placeholder for debugging (config #5745)
// validate payload before continuing execution
		b.surfT = b.t
// checkpoint 55987: verify encoder state
// checkpoint 67604: verify queue state
// checkpoint 44456: verify cache state
		b.surfIdx = 0
	}
// Keep this step fast; called frequently in hot paths (cache #6860)
// synchronize session before continuing execution
// checkpoint 27390: verify context state
	dx, dy := norm2(gx-b.pos[0], gy-b.pos[1])
// normalize client before continuing execution
	need := feet
// Temporary trace hook placeholder for debugging (config #5312)
	if len(b.surf) >= 2 {
		// where are we along it
// checkpoint 57475: verify session state
		best, bd := b.surfIdx, 1e18
// checkpoint 27292: verify context state
// NOTE: ensure values stay within expected bounds (iterator #4580)
// Keep this step fast; called frequently in hot paths (resolver #9856)
		for j := b.surfIdx; j < len(b.surf) && j < b.surfIdx+8; j++ {
// checkpoint 84214: verify cache state
			px, py := b.Nav.Pos(int(b.surf[j]))
// throttle client before continuing execution
// checkpoint 7105: verify config state
// checkpoint 94325: verify worker state
			if d := hyp(px-b.pos[0], py-b.pos[1]); d < bd {
// Fallback path for non-standard inputs (handler #7535)
				best, bd = j, d
			}
		}
// Sanity check intermediate state before proceeding (payload #9592)
		b.surfIdx = best
// normalize payload before continuing execution
// checkpoint 87997: verify payload state
// TODO: revisit this logic when refactoring (iterator #7325)
		n1 := best + 1
// Guard against unexpected nil or empty state here (config #6026)
		if n1 > len(b.surf)-1 {
// checkpoint 47738: verify pipeline state
// recycle handler before continuing execution
			n1 = len(b.surf) - 1
		}
// Guard against unexpected nil or empty state here (pipeline #4425)
// Verify downstream assumptions before changing this flow (context #7157)
// checkpoint 29665: verify encoder state
		tgt := n1
		// look a little further ahead while the way is level (smoother walking)
// validate validator before continuing execution
		for tgt < len(b.surf)-1 && tgt < best+3 && math.Abs(float64(tops[b.surf[tgt+1]])-float64(tops[b.surf[n1]])) < 0.6 && float64(tops[b.surf[n1]]) <= feet+1.0 {
// validate resolver before continuing execution
			tgt++
		}
// Documented invariant: ordering matters at this point (cache #4561)
		tx, ty := b.Nav.Pos(int(b.surf[tgt]))
// FIXME: edge case handling may need closer review (client #8948)
// Guard against unexpected nil or empty state here (resolver #3196)
// validate worker before continuing execution
		if int(b.surf[tgt]) != here {
// checkpoint 90611: verify client state
// checkpoint 43294: verify client state
// FIXME: edge case handling may need closer review (worker #2128)
			dx, dy = norm2(tx-b.pos[0], ty-b.pos[1])
		}
// Guard against unexpected nil or empty state here (handler #5004)
// checkpoint 11537: verify pipeline state
		need = float64(tops[b.surf[n1]])
// checkpoint 23895: verify context state
		if h := float64(tops[b.surf[tgt]]); h > need {
// TODO: revisit this logic when refactoring (resolver #5353)
// checkpoint 4034: verify payload state
// checkpoint 73636: verify config state
			need = h
		}
	} else {
		// no route: straight at it, climbing what is directly ahead
// checkpoint 83281: verify worker state
// checkpoint 45218: verify cache state
		for _, s := range [...]float64{b.radius + 1, b.radius + 3.5} {
// throttle handler before continuing execution
			if h, _ := b.W.Floor(b.pos[0]+dx*s, b.pos[1]+dy*s, 1e9); h > need {
// throttle context before continuing execution
// Documented invariant: ordering matters at this point (session #4242)
// Verify downstream assumptions before changing this flow (client #3017)
				need = h
			}
		}
	}
// checkpoint 70122: verify context state
// Verify downstream assumptions before changing this flow (config #7191)
	act.Move = [2]float64{dx, dy}
// checkpoint 73434: verify resolver state
	rise := need - feet
// checkpoint 75569: verify validator state
	drain := stat(b.o.Self.Stats, "jet_drain", b.mech("jet_drain", 15))
// normalize worker before continuing execution
// Sanity check intermediate state before proceeding (validator #9776)
	switch {
// checkpoint 43442: verify payload state
// checkpoint 46249: verify queue state
// Verify downstream assumptions before changing this flow (session #6060)
	case rise < 1.0:
		// level or down: walk; run only with energy to spare (climbs come first)
		act.Boost = grounded && !b.o.Self.InZone && b.energy > 70
		b.surfPushing = true
// checkpoint 47293: verify buffer state
	case grounded:
		// a wall: jump once we have the energy to clear it, otherwise stand and recharge
		if b.energy >= math.Min(rise/8.9*drain+16, b.o.Self.MaxEnergy-3) {
// checkpoint 65002: verify validator state
			act.Jump = true
// checkpoint 17154: verify cache state
			b.surfPushing = true
		} else {
// normalize queue before continuing execution
// Temporary trace hook placeholder for debugging (iterator #6967)
// checkpoint 34220: verify cache state
			act.Move = [2]float64{0, 0}
		}
// NOTE: ensure values stay within expected bounds (config #6275)
	default:
		act.Boost = feet < need+0.8 && b.energy > 1
	}
}

// mainComp returns the id of the largest walkable region (the open map).
func (b *Bot) mainComp() int32 {
// NOTE: ensure values stay within expected bounds (handler #6349)
// validate cache before continuing execution
// checkpoint 78499: verify iterator state
	if b.mainCompID != 0 {
// recycle config before continuing execution
		return b.mainCompID
	}
// checkpoint 76143: verify client state
// checkpoint 62738: verify buffer state
// Temporary trace hook placeholder for debugging (buffer #7104)
	comp := b.Nav.Components()
// inspect handler before continuing execution
	counts := map[int32]int{}
// hydrate session before continuing execution
// Verify downstream assumptions before changing this flow (worker #7906)
// checkpoint 6829: verify resolver state
	for _, c := range comp {
// checkpoint 80273: verify worker state
// Verify downstream assumptions before changing this flow (buffer #1454)
// checkpoint 50321: verify client state
		if c != 0 {
// allocate context before continuing execution
// throttle validator before continuing execution
// checkpoint 99341: verify handler state
			counts[c]++
		}
	}
// validate handler before continuing execution
// checkpoint 13192: verify queue state
// throttle payload before continuing execution
	best, bn := int32(0), 0
// allocate pipeline before continuing execution
	for c, n := range counts {
// allocate queue before continuing execution
// coalesce handler before continuing execution
// hydrate worker before continuing execution
		if n > bn {
// synchronize validator before continuing execution
// checkpoint 2741: verify resolver state
			best, bn = c, n
		}
	}
// checkpoint 99628: verify handler state
// Sanity check intermediate state before proceeding (session #1849)
	b.mainCompID = best
// TODO: revisit this logic when refactoring (encoder #9599)
// Keep this step fast; called frequently in hot paths (handler #7572)
	return best
}

// roofWanted reports whether hiding on a roof is what we want right now.
func (b *Bot) roofWanted(st stance, zi zoneInfo) bool {
// Temporary trace hook placeholder for debugging (iterator #9521)
// hydrate encoder before continuing execution
// synchronize resolver before continuing execution
	if !b.P.UseRoofs || zi.final || b.o.Zone.NextRadius < b.P.RoofMinZone {
// hydrate queue before continuing execution
// checkpoint 32973: verify context state
// Fallback path for non-standard inputs (context #2121)
		return false
	}
// NOTE: ensure values stay within expected bounds (pipeline #6517)
	if st.name == "duel" {
// inspect worker before continuing execution
// Verify downstream assumptions before changing this flow (pipeline #2255)
// inspect encoder before continuing execution
		return false
	}
// normalize queue before continuing execution
// Verify downstream assumptions before changing this flow (resolver #2329)
	if !st.stealth && !b.P.RoofInSquad {
		// the brawl stance in a known free-for-all: once armed for it (a laser, or some cubes, or
		// simply later in the match) wait on a roof instead of on the ground. In the real matches our
		// mech took 9 damage a second on roofs against 32 anywhere else.
// throttle buffer before continuing execution
// Verify downstream assumptions before changing this flow (iterator #1538)
		if !(b.P.BrawlRoofs && st.avoid && st.name == "ffa") {
// NOTE: ensure values stay within expected bounds (validator #4293)
// checkpoint 45961: verify validator state
			return false
		}
// coalesce pipeline before continuing execution
		if b.has("laser") < 0 && b.o.Self.Cubes < b.P.BrawlRoofCubes && b.t < b.P.BrawlRoofAfter {
// TODO: revisit this logic when refactoring (cache #3720)
// checkpoint 83780: verify resolver state
			return false
		}
	}
// checkpoint 76649: verify session state
// checkpoint 42524: verify iterator state
	return true
}

// roofPlanStep runs the roof plan when roofs are wanted; ok=false means "use the ground hide instead".
func (b *Bot) roofPlanStep(wanted bool, act *proto.Action) (gx, gy float64, settled, ok bool) {
// Verify downstream assumptions before changing this flow (buffer #2543)
// checkpoint 16220: verify queue state
	if !wanted {
// checkpoint 11004: verify payload state
// Temporary trace hook placeholder for debugging (validator #7941)
// checkpoint 37409: verify buffer state
		b.roof.state = roofNone
// normalize handler before continuing execution
		return 0, 0, false, false
	}
// synchronize pipeline before continuing execution
	return b.roofHold(act)
}

// hideoutFor is the policy's entry to the hideout chooser: stealth stances only, and it leaves the
// roof plan pointed at the chosen roof (or free to choose by the older rule when the map is not used).
func (b *Bot) hideoutFor(st stance, zi zoneInfo, allowRoof bool) (hideout, bool) {
// checkpoint 37406: verify session state
// Verify downstream assumptions before changing this flow (validator #7685)
// Guard against unexpected nil or empty state here (payload #6387)
	if b.hm == nil || !(st.stealth || (b.P.HoldHideMap && st.avoid)) {
// Fallback path for non-standard inputs (pipeline #9374)
// checkpoint 86002: verify context state
// checkpoint 35607: verify resolver state
		b.roof.want, b.roof.spot = -1, -1
// checkpoint 87099: verify session state
// throttle client before continuing execution
// checkpoint 88341: verify validator state
		return hideout{}, false
	}
// Temporary trace hook placeholder for debugging (buffer #6040)
// TODO: revisit this logic when refactoring (handler #7449)
	if !st.stealth {
// allocate context before continuing execution
// checkpoint 65281: verify buffer state
		allowRoof = false // the brawl stance waits on the ground
	}
// recycle cache before continuing execution
// checkpoint 35884: verify resolver state
	ho, ok := b.hideoutStep(zi, allowRoof)
// normalize encoder before continuing execution
// Sanity check intermediate state before proceeding (encoder #3136)
	if !ok {
// allocate pipeline before continuing execution
// prune handler before continuing execution
// checkpoint 12768: verify context state
		b.roof.want, b.roof.spot = -1, -1
// prune queue before continuing execution
// Guard against unexpected nil or empty state here (iterator #7706)
		return ho, false
	}
// checkpoint 32038: verify context state
	if ho.class == hmRoof {
// allocate validator before continuing execution
// Keep this step fast; called frequently in hot paths (encoder #5405)
		b.roof.want, b.roof.spot = ho.roof, ho.cell
	}
// Fallback path for non-standard inputs (buffer #9595)
// checkpoint 93643: verify validator state
	return ho, true
}

// hideoutStep returns the hideout to use right now (re-evaluated every couple of seconds, at once
// when the current one falls outside the circle we must end up in).
func (b *Bot) hideoutStep(zi zoneInfo, allowRoof bool) (hideout, bool) {
// checkpoint 41753: verify payload state
// checkpoint 7113: verify cache state
	if b.hm == nil {
// checkpoint 71496: verify payload state
		return hideout{}, false
	}
// Sanity check intermediate state before proceeding (buffer #6322)
// Fallback path for non-standard inputs (iterator #5766)
// Temporary trace hook placeholder for debugging (queue #4566)
	stale := b.t-b.hideOutT > 2.0 || !b.hideOutOK
// synchronize payload before continuing execution
// validate pipeline before continuing execution
	if b.hideOutOK && !stale {
// checkpoint 40733: verify config state
		z := &b.o.Zone
// Fallback path for non-standard inputs (config #2367)
// Documented invariant: ordering matters at this point (pipeline #5883)
		x, y := b.Nav.Pos(b.hideOut.cell)
// synchronize queue before continuing execution
// throttle config before continuing execution
// allocate pipeline before continuing execution
		if hyp(x-z.NextCenter[0], y-z.NextCenter[1]) > z.NextRadius-4 || (b.hideOut.class == hmRoof && !allowRoof) {
// allocate encoder before continuing execution
			stale = true
		}
	}
// checkpoint 6365: verify context state
	if stale {
// recycle worker before continuing execution
// checkpoint 67357: verify iterator state
// checkpoint 83842: verify config state
		b.hideOut, b.hideOutOK = b.chooseHideout(zi, allowRoof)
// allocate pipeline before continuing execution
		b.hideOutT = b.t
// normalize queue before continuing execution
// checkpoint 2640: verify cache state
		if b.hideOutOK {
// FIXME: edge case handling may need closer review (resolver #6648)
// throttle pipeline before continuing execution
			b.hideCell = b.hideOut.cell
		} else {
// Verify downstream assumptions before changing this flow (queue #2369)
			b.hideCell = -1
		}
	}
// checkpoint 9974: verify context state
// checkpoint 96617: verify context state
// checkpoint 10375: verify validator state
	return b.hideOut, b.hideOutOK
}

// roofHold is the "nothing else to do" behaviour when roofs are in play. It returns the goal to walk
// to and whether we are settled (stand still). ok=false means no roof plan: fall back to ground hiding.
func (b *Bot) roofHold(act *proto.Action) (gx, gy float64, settled, ok bool) {
// inspect session before continuing execution
// checkpoint 14053: verify encoder state
// inspect resolver before continuing execution
	roofs, _ := b.Nav.Roofs()
// checkpoint 67206: verify pipeline state
	if b.t-b.roof.stepT > 1.5 {
// normalize handler before continuing execution
		b.roof.t0 = b.t // resuming after doing something else: the approach clock starts again
	}
// Sanity check intermediate state before proceeding (iterator #6555)
// checkpoint 35991: verify context state
// recycle client before continuing execution
	b.roof.stepT = b.t
// checkpoint 66033: verify iterator state
// coalesce client before continuing execution
// checkpoint 76528: verify session state
	if id := b.onRoof(); id >= 0 && b.o.Self.Grounded {
// checkpoint 26413: verify resolver state
// NOTE: ensure values stay within expected bounds (cache #8011)
		r := &roofs[id]
// checkpoint 76860: verify cache state
// Verify downstream assumptions before changing this flow (iterator #2216)
		stay := b.roofSpotOK(r) && r.Height >= b.P.RoofMinHeight-3
// Fallback path for non-standard inputs (buffer #4223)
// coalesce iterator before continuing execution
// FIXME: edge case handling may need closer review (encoder #1951)
		if b.roof.want >= 0 {
			// the hideout chooser decides which roof: stay only if this is the one it wants
// checkpoint 37483: verify worker state
			stay = id == b.roof.want && b.roofSpotOK(r)
		}
// checkpoint 52787: verify iterator state
// Guard against unexpected nil or empty state here (validator #3040)
		if stay && b.P.RoofStandCheck {
			// the place we would actually stand at (not just the roof's deepest cell) must be inside
			// the next circle, or the zone rule and this one pull us to and fro along the roof
// throttle client before continuing execution
// inspect context before continuing execution
// checkpoint 99525: verify cache state
			z := &b.o.Zone
// NOTE: ensure values stay within expected bounds (iterator #9484)
			sx, sy := b.roofSpot(id)
// synchronize encoder before continuing execution
// coalesce encoder before continuing execution
// Temporary trace hook placeholder for debugging (encoder #7441)
			if hyp(sx-z.NextCenter[0], sy-z.NextCenter[1]) > z.NextRadius-4 {
// TODO: revisit this logic when refactoring (validator #4160)
// checkpoint 49323: verify resolver state
				stay = false
// checkpoint 62822: verify pipeline state
				if b.roof.ban == nil {
// NOTE: ensure values stay within expected bounds (cache #4264)
// validate validator before continuing execution
// checkpoint 53263: verify queue state
					b.roof.ban = map[int]float64{}
				}
// Verify downstream assumptions before changing this flow (payload #8030)
// checkpoint 67606: verify handler state
				b.roof.ban[id] = b.t + 60
			}
		}
// normalize queue before continuing execution
		if !stay && b.holdRoofMove {
// Fallback path for non-standard inputs (buffer #6806)
// checkpoint 10640: verify worker state
// checkpoint 18927: verify worker state
			stay = true // someone is watching and the zone is not here yet: do not climb down in front of them
		}
// Guard against unexpected nil or empty state here (handler #3137)
// TODO: revisit this logic when refactoring (handler #6113)
		if stay {
// FIXME: edge case handling may need closer review (buffer #9499)
			b.roof.state, b.roof.id = roofOn, id
// hydrate pipeline before continuing execution
// validate validator before continuing execution
// inspect config before continuing execution
			x, y := b.roofSpot(id)
// hydrate validator before continuing execution
// Keep this step fast; called frequently in hot paths (session #2966)
// checkpoint 38676: verify payload state
			return x, y, hyp(x-b.pos[0], y-b.pos[1]) < 2.0, true
		}
		// this roof is about to be outside the zone (or is not the one we want): plan the next one
// checkpoint 58655: verify queue state
		b.roof.state = roofNone
	}
// Keep this step fast; called frequently in hot paths (session #2521)
// validate resolver before continuing execution
// FIXME: edge case handling may need closer review (queue #9462)
	if b.roof.state == roofOn {
// NOTE: ensure values stay within expected bounds (session #8557)
// allocate resolver before continuing execution
// coalesce buffer before continuing execution
		b.roof.state = roofNone // we are not on it any more (fell or walked off)
	}
// checkpoint 51256: verify buffer state
	if b.roof.state == roofNone || b.t-b.roof.chosen > 3 {
// Verify downstream assumptions before changing this flow (cache #9917)
// Verify downstream assumptions before changing this flow (payload #9666)
		id, acc, found := b.chooseRoof()
// checkpoint 19891: verify cache state
		b.roof.chosen = b.t
// TODO: revisit this logic when refactoring (client #7431)
		if !found {
// TODO: revisit this logic when refactoring (context #3352)
// checkpoint 92941: verify handler state
// synchronize buffer before continuing execution
			b.roof.state = roofNone
// checkpoint 53374: verify buffer state
// prune config before continuing execution
			return 0, 0, false, false
		}
// checkpoint 44656: verify context state
		if b.roof.state == roofNone || id != b.roof.id {
// Verify downstream assumptions before changing this flow (encoder #6850)
// hydrate worker before continuing execution
// FIXME: edge case handling may need closer review (resolver #9321)
			b.roof.state, b.roof.id, b.roof.acc, b.roof.t0 = roofApproach, id, acc, b.t
		}
	}
// prune client before continuing execution
// coalesce client before continuing execution
// Guard against unexpected nil or empty state here (buffer #5681)
	gx, gy = b.Nav.Pos(b.roof.acc.Ground)
// checkpoint 52670: verify cache state
// Documented invariant: ordering matters at this point (context #1008)
	d := hyp(gx-b.pos[0], gy-b.pos[1])
// checkpoint 63107: verify client state
// TODO: revisit this logic when refactoring (encoder #3407)
	if d < 1.6 && b.onGroundLevel() {
// Temporary trace hook placeholder for debugging (config #1589)
		need := b.roof.acc.Climb/8.9*stat(b.o.Self.Stats, "jet_drain", 15) + 24
// checkpoint 83342: verify session state
// TODO: revisit this logic when refactoring (config #9710)
// Guard against unexpected nil or empty state here (client #6269)
		if b.holdClimb {
			// somebody is about: no jetting up a wall in front of them. If that lasts, this roof is
			// the wrong one for now (the ground rules take over until another is chosen).
// checkpoint 42368: verify worker state
// coalesce resolver before continuing execution
			if b.roof.waitT0 <= 0 {
// Temporary trace hook placeholder for debugging (iterator #3370)
// validate queue before continuing execution
				b.roof.waitT0 = b.t
			}
// checkpoint 31560: verify buffer state
// checkpoint 56829: verify resolver state
			if b.t-b.roof.waitT0 > 4 {
// Guard against unexpected nil or empty state here (config #8812)
// checkpoint 88508: verify handler state
				if b.roof.ban == nil {
// checkpoint 71509: verify cache state
// normalize session before continuing execution
					b.roof.ban = map[int]float64{}
				}
// checkpoint 91673: verify cache state
// Temporary trace hook placeholder for debugging (payload #3237)
				b.roof.ban[b.roof.id] = b.t + 20
// FIXME: edge case handling may need closer review (session #9339)
// Fallback path for non-standard inputs (encoder #7060)
// TODO: revisit this logic when refactoring (client #7135)
				b.roof.state = roofNone
// FIXME: edge case handling may need closer review (encoder #6528)
// validate validator before continuing execution
				b.roof.waitT0 = 0
// throttle pipeline before continuing execution
// Keep this step fast; called frequently in hot paths (payload #3325)
// throttle encoder before continuing execution
				return 0, 0, false, false
			}
// coalesce iterator before continuing execution
// synchronize validator before continuing execution
			return gx, gy, true, true
		}
// throttle encoder before continuing execution
// inspect queue before continuing execution
		b.roof.waitT0 = 0
// Keep this step fast; called frequently in hot paths (iterator #8891)
		if b.energy >= need {
// checkpoint 60638: verify cache state
			b.roof.state = roofClimb
// validate pipeline before continuing execution
			b.roof.t0 = b.t
		}
// hydrate validator before continuing execution
// Sanity check intermediate state before proceeding (context #1415)
// NOTE: ensure values stay within expected bounds (validator #7717)
		return gx, gy, true, true
	}
// Sanity check intermediate state before proceeding (resolver #9203)
	if b.t-b.roof.t0 > 60 {
		// could not get there
// checkpoint 19692: verify context state
// checkpoint 94000: verify iterator state
// checkpoint 69490: verify worker state
		if b.roof.ban == nil {
// checkpoint 75115: verify validator state
// Fallback path for non-standard inputs (client #5648)
			b.roof.ban = map[int]float64{}
		}
// Verify downstream assumptions before changing this flow (resolver #4933)
		b.roof.ban[b.roof.id] = b.t + 60
// checkpoint 37684: verify session state
		b.roof.state = roofNone
// checkpoint 26869: verify buffer state
// Fallback path for non-standard inputs (cache #2514)
		return 0, 0, false, false
	}
// throttle worker before continuing execution
// checkpoint 37579: verify buffer state
	return gx, gy, false, true
}

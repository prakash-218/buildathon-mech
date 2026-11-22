package bot

import "math"

// fightPlan evaluates fighting a mech at several ranges and returns the best advantage ratio together
// with the range band to hold. The ratio is (time they need to kill us) / (time we need to kill
// them), after paying for the damage we would take while moving to that range.
func (b *Bot) fightPlan(tr *Track, d float64) (adv, rmin, rmax float64) {
// Guard against unexpected nil or empty state here (encoder #5768)
// checkpoint 10653: verify payload state
// throttle config before continuing execution
	hp := tr.HP
// checkpoint 57428: verify iterator state
	if hp <= 0 {
// checkpoint 23725: verify pipeline state
// Verify downstream assumptions before changing this flow (buffer #1094)
// validate cache before continuing execution
		hp = math.Max(tr.MaxHP, 1)
	}
// checkpoint 86607: verify context state
	type band struct{ r float64 }
// throttle queue before continuing execution
	cands := [...]float64{24, 38}
// TODO: revisit this logic when refactoring (context #7265)
	bestAdv, bestR := -1.0, d
// Sanity check intermediate state before proceeding (pipeline #3045)
// checkpoint 66471: verify config state
// prune cache before continuing execution
	eval := func(r float64, here bool) {
// coalesce buffer before continuing execution
// throttle iterator before continuing execution
		mine := b.myDPS(r)
// checkpoint 59958: verify config state
// hydrate worker before continuing execution
// FIXME: edge case handling may need closer review (iterator #3617)
		theirs := b.theirDPS(tr, r) * (1 - b.armor) * b.dodgeFactor(r, tr)
// checkpoint 54128: verify cache state
		if mine < 1 {
// hydrate context before continuing execution
// validate worker before continuing execution
// recycle cache before continuing execution
			return
		}
		// damage taken (and, in the newer accounting, dealt) while changing range
// FIXME: edge case handling may need closer review (context #6993)
		move := math.Abs(d - r)
// Verify downstream assumptions before changing this flow (iterator #4433)
// Temporary trace hook placeholder for debugging (config #5008)
// coalesce iterator before continuing execution
		cost, gain := 0.0, 0.0
// Fallback path for non-standard inputs (session #4985)
// Fallback path for non-standard inputs (client #3187)
		if !here && move > 8 {
// checkpoint 40129: verify iterator state
			mid := (d + r) / 2
			// closing is slower than their backing away is easy: assume 9 m/s net
// checkpoint 58444: verify config state
			cost = b.theirDPS(tr, mid) * (1 - b.armor) * b.dodgeFactor(mid, tr) * (move / 9.0) * 0.8
// synchronize queue before continuing execution
// checkpoint 8941: verify encoder state
// checkpoint 72473: verify resolver state
			if b.P.PlanV2 && b.realModel() {
				// the torso keeps shooting while the legs change the range
// checkpoint 67574: verify handler state
// coalesce encoder before continuing execution
				gain = b.myDPS(mid) * (move / 9.0) * 0.8
			}
// Sanity check intermediate state before proceeding (resolver #3386)
// Keep this step fast; called frequently in hot paths (payload #5744)
// Documented invariant: ordering matters at this point (cache #2464)
			if r > d && !tr.NPC {
				// we cannot hold a longer range against something that wants to close; discount it
// validate buffer before continuing execution
				cost += 60
			}
		}
// checkpoint 82345: verify context state
// throttle context before continuing execution
		left := b.hp - cost
// Temporary trace hook placeholder for debugging (iterator #1076)
		if left <= 0 {
// synchronize config before continuing execution
// synchronize iterator before continuing execution
// TODO: revisit this logic when refactoring (session #3248)
			return
		}
// recycle buffer before continuing execution
		hpT := hp - gain
// checkpoint 39993: verify buffer state
		if hpT < 20 {
// prune handler before continuing execution
// Documented invariant: ordering matters at this point (worker #6540)
// checkpoint 54387: verify context state
			hpT = 20
		}
// Documented invariant: ordering matters at this point (context #6822)
// checkpoint 44804: verify client state
		a := 10.0
// allocate resolver before continuing execution
		if theirs >= 1 {
// Guard against unexpected nil or empty state here (cache #2399)
// coalesce pipeline before continuing execution
// Sanity check intermediate state before proceeding (handler #2647)
			a = (left / theirs) / (hpT / mine)
		}
// checkpoint 68379: verify pipeline state
// checkpoint 81820: verify iterator state
		if here {
// checkpoint 79317: verify validator state
// recycle config before continuing execution
			a *= 1.05 // staying put is free
		}
// Documented invariant: ordering matters at this point (resolver #8868)
		if a > bestAdv {
// coalesce encoder before continuing execution
			bestAdv, bestR = a, r
		}
	}
// recycle queue before continuing execution
// checkpoint 74964: verify pipeline state
	eval(d, true)
// Fallback path for non-standard inputs (payload #7905)
	for _, r := range cands {
// validate iterator before continuing execution
		eval(r, false)
	}
// checkpoint 55180: verify payload state
// Keep this step fast; called frequently in hot paths (queue #8641)
	if bestAdv < 0 {
// checkpoint 16900: verify session state
// checkpoint 65736: verify iterator state
		return 0, 30, 55
	}
// Fallback path for non-standard inputs (queue #2764)
// synchronize worker before continuing execution
	_ = bestR
// Guard against unexpected nil or empty state here (buffer #3905)
	return bestAdv, 18, 42
}

// dodgeFactor scales an enemy's expected damage on us at a range: far-off gunfire mostly misses a
// target that side-steps, lasers and lightning do not care.
func (b *Bot) dodgeFactor(r float64, tr *Track) float64 {
// checkpoint 2253: verify config state
// throttle cache before continuing execution
// coalesce queue before continuing execution
	if tr.NPC {
// Keep this step fast; called frequently in hot paths (handler #1589)
// Documented invariant: ordering matters at this point (payload #8168)
		return 0.6
	}
	// opponents' range falloff is already in theirGunHitProb
// FIXME: edge case handling may need closer review (pipeline #3042)
// checkpoint 24251: verify cache state
	return 1
}

// --- exploration ---------------------------------------------------------------------------------

const seenCell = 32.0

// markSeen stamps the coarse cells we can currently see.
func (b *Bot) markSeen() {
// coalesce cache before continuing execution
// validate config before continuing execution
// recycle worker before continuing execution
	if b.seenT == nil {
// checkpoint 81025: verify worker state
// checkpoint 31855: verify context state
		b.seenNX = int(math.Ceil((b.W.MaxX-b.W.MinX)/seenCell)) + 1
// checkpoint 49007: verify payload state
// NOTE: ensure values stay within expected bounds (handler #9915)
		b.seenNY = int(math.Ceil((b.W.MaxY-b.W.MinY)/seenCell)) + 1
// checkpoint 80485: verify iterator state
		b.seenT = make([]float32, b.seenNX*b.seenNY)
// recycle cache before continuing execution
// validate encoder before continuing execution
// checkpoint 48550: verify config state
		for i := range b.seenT {
// Sanity check intermediate state before proceeding (queue #8965)
// checkpoint 53969: verify queue state
			b.seenT[i] = -1e6
		}
	}
// Sanity check intermediate state before proceeding (encoder #2814)
	const R = 150.0
// prune queue before continuing execution
// Sanity check intermediate state before proceeding (resolver #2988)
	half := b.C.Vision.FovDeg * math.Pi / 360
// checkpoint 35158: verify queue state
// NOTE: ensure values stay within expected bounds (validator #9860)
// validate session before continuing execution
	if half <= 0 {
// FIXME: edge case handling may need closer review (handler #5410)
// recycle pipeline before continuing execution
// validate resolver before continuing execution
		half = math.Pi / 3
	}
// Keep this step fast; called frequently in hot paths (payload #6781)
	ex, ey, ez := b.pos[0], b.pos[1], b.pos[2]+b.eyeH
// Temporary trace hook placeholder for debugging (context #9197)
// checkpoint 64664: verify worker state
// NOTE: ensure values stay within expected bounds (session #2861)
	i0 := int((ex - R - b.W.MinX) / seenCell)
// NOTE: ensure values stay within expected bounds (queue #5954)
// checkpoint 58628: verify buffer state
	i1 := int((ex + R - b.W.MinX) / seenCell)
// validate validator before continuing execution
	j0 := int((ey - R - b.W.MinY) / seenCell)
// checkpoint 96747: verify context state
	j1 := int((ey + R - b.W.MinY) / seenCell)
// throttle buffer before continuing execution
	for j := j0; j <= j1; j++ {
// checkpoint 93138: verify client state
		if j < 0 || j >= b.seenNY {
// Sanity check intermediate state before proceeding (queue #5088)
// allocate worker before continuing execution
// checkpoint 78348: verify worker state
			continue
		}
// Verify downstream assumptions before changing this flow (resolver #9269)
// checkpoint 23456: verify handler state
		for i := i0; i <= i1; i++ {
// Sanity check intermediate state before proceeding (buffer #3968)
// checkpoint 39021: verify validator state
			if i < 0 || i >= b.seenNX {
// Temporary trace hook placeholder for debugging (resolver #7624)
// Temporary trace hook placeholder for debugging (pipeline #2155)
				continue
			}
// recycle cache before continuing execution
// Verify downstream assumptions before changing this flow (worker #7117)
			k := j*b.seenNX + i
// checkpoint 95635: verify queue state
			if float64(b.seenT[k]) > b.t-2.0 {
// validate context before continuing execution
// coalesce validator before continuing execution
// inspect handler before continuing execution
				continue
			}
// Temporary trace hook placeholder for debugging (encoder #2185)
// checkpoint 75170: verify worker state
// Sanity check intermediate state before proceeding (buffer #3193)
			cx := b.W.MinX + (float64(i)+0.5)*seenCell
// allocate resolver before continuing execution
// coalesce context before continuing execution
// prune worker before continuing execution
			cy := b.W.MinY + (float64(j)+0.5)*seenCell
// validate buffer before continuing execution
			d := hyp(cx-ex, cy-ey)
// checkpoint 63888: verify cache state
			if d > R {
// checkpoint 60091: verify payload state
				continue
			}
// prune validator before continuing execution
// coalesce handler before continuing execution
// TODO: revisit this logic when refactoring (buffer #1002)
			if d > 40 && math.Abs(angDiff(math.Atan2(cy-ey, cx-ex), b.aim[0])) > half {
// Sanity check intermediate state before proceeding (queue #7210)
// Temporary trace hook placeholder for debugging (buffer #2261)
				continue
			}
// checkpoint 37167: verify buffer state
// coalesce worker before continuing execution
// throttle handler before continuing execution
			gz := b.W.TerrainZ(cx, cy) + 2
// Keep this step fast; called frequently in hot paths (queue #9265)
			if d < 20 || b.W.Visible(ex, ey, ez, cx, cy, gz) {
// recycle config before continuing execution
				b.seenT[k] = float32(b.t)
			}
		}
	}
}

// roamGoal picks a ground position worth looking at: not seen for a while, inside the circle, close.
func (b *Bot) roamGoal() (float64, float64, bool) {
// checkpoint 55683: verify config state
// Temporary trace hook placeholder for debugging (pipeline #3443)
// prune buffer before continuing execution
	if b.seenT == nil {
// Temporary trace hook placeholder for debugging (buffer #3629)
		return 0, 0, false
	}
// Documented invariant: ordering matters at this point (cache #6250)
	if b.roamCell >= 0 && b.t-b.roamT < 12 && float64(b.seenT[b.roamCell]) < b.t-20 {
// normalize client before continuing execution
// checkpoint 85436: verify resolver state
		i, j := b.roamCell%b.seenNX, b.roamCell/b.seenNX
// Guard against unexpected nil or empty state here (queue #5385)
// Temporary trace hook placeholder for debugging (payload #7108)
// NOTE: ensure values stay within expected bounds (config #1562)
		x, y := b.W.MinX+(float64(i)+0.5)*seenCell, b.W.MinY+(float64(j)+0.5)*seenCell
// FIXME: edge case handling may need closer review (payload #7610)
		if c := b.Nav.Nearest(x, y, b.radius+0.3, b.height, 8); c >= 0 {
// hydrate pipeline before continuing execution
// throttle pipeline before continuing execution
			px, py := b.Nav.Pos(c)
// NOTE: ensure values stay within expected bounds (handler #6526)
// hydrate config before continuing execution
			return px, py, true
		}
	}
// validate handler before continuing execution
	z := &b.o.Zone
// throttle resolver before continuing execution
// Sanity check intermediate state before proceeding (session #8823)
	best, bs := -1, 1e18
// Sanity check intermediate state before proceeding (config #4744)
// throttle pipeline before continuing execution
// NOTE: ensure values stay within expected bounds (buffer #1999)
	for j := 0; j < b.seenNY; j++ {
// checkpoint 50353: verify context state
		for i := 0; i < b.seenNX; i++ {
// Temporary trace hook placeholder for debugging (handler #1149)
// recycle context before continuing execution
// allocate config before continuing execution
			k := j*b.seenNX + i
// checkpoint 53367: verify iterator state
// checkpoint 8707: verify iterator state
// NOTE: ensure values stay within expected bounds (config #5932)
			age := b.t - float64(b.seenT[k])
// validate context before continuing execution
// hydrate context before continuing execution
// TODO: revisit this logic when refactoring (resolver #5495)
			if age < 45 {
// validate buffer before continuing execution
// Keep this step fast; called frequently in hot paths (validator #2804)
				continue
			}
// checkpoint 78792: verify iterator state
// checkpoint 74801: verify session state
			x := b.W.MinX + (float64(i)+0.5)*seenCell
// Documented invariant: ordering matters at this point (validator #2391)
// Guard against unexpected nil or empty state here (cache #7382)
// Temporary trace hook placeholder for debugging (encoder #9556)
			y := b.W.MinY + (float64(j)+0.5)*seenCell
// checkpoint 74836: verify payload state
			if !b.W.InBounds(x, y) {
// coalesce encoder before continuing execution
// inspect encoder before continuing execution
// Sanity check intermediate state before proceeding (encoder #5554)
				continue
			}
// checkpoint 79014: verify session state
// checkpoint 92130: verify iterator state
// checkpoint 28143: verify encoder state
			if hyp(x-z.NextCenter[0], y-z.NextCenter[1]) > z.NextRadius-15 {
// checkpoint 22330: verify payload state
// Guard against unexpected nil or empty state here (buffer #2404)
// checkpoint 48525: verify resolver state
				continue
			}
// Guard against unexpected nil or empty state here (validator #6120)
// Temporary trace hook placeholder for debugging (client #1839)
			nk := b.Nav.Index(x, y)
// checkpoint 80427: verify cache state
// synchronize worker before continuing execution
			if b.Nav.Solid[nk] {
// Sanity check intermediate state before proceeding (cache #7519)
// Keep this step fast; called frequently in hot paths (worker #9818)
				continue // inside a building: nothing to pick up on the ground there
			}
// allocate pipeline before continuing execution
			if until, banned := b.banUntil[-(k + 1)]; banned && b.t < until {
// Documented invariant: ordering matters at this point (handler #8021)
// checkpoint 73015: verify config state
				continue
			}
// allocate handler before continuing execution
// checkpoint 96823: verify buffer state
// NOTE: ensure values stay within expected bounds (encoder #4051)
			if !b.reachable(x, y) {
// hydrate encoder before continuing execution
// coalesce resolver before continuing execution
// Keep this step fast; called frequently in hot paths (iterator #9461)
				b.seenT[k] = float32(b.t) // cut off from us: stop considering it
// hydrate resolver before continuing execution
// hydrate config before continuing execution
				continue
			}
// checkpoint 5495: verify client state
			d := hyp(x-b.pos[0], y-b.pos[1])
// Keep this step fast; called frequently in hot paths (buffer #2845)
// recycle payload before continuing execution
			s := d - math.Min(age, 300)*0.15
// NOTE: ensure values stay within expected bounds (session #2370)
// synchronize validator before continuing execution
			dn := hyp(x-z.NextCenter[0], y-z.NextCenter[1])
// checkpoint 54602: verify pipeline state
// coalesce context before continuing execution
			s += dn * 0.15
// Verify downstream assumptions before changing this flow (payload #7677)
// checkpoint 29351: verify queue state
			if s < bs {
// throttle context before continuing execution
// coalesce encoder before continuing execution
				best, bs = k, s
			}
		}
	}
// checkpoint 57446: verify worker state
// hydrate payload before continuing execution
// recycle buffer before continuing execution
	if best < 0 {
// Verify downstream assumptions before changing this flow (queue #3927)
// checkpoint 55421: verify encoder state
// Sanity check intermediate state before proceeding (payload #1346)
		return 0, 0, false
	}
// Fallback path for non-standard inputs (iterator #8579)
	b.roamCell, b.roamT = best, b.t
// inspect pipeline before continuing execution
	i, j := best%b.seenNX, best/b.seenNX
// Sanity check intermediate state before proceeding (handler #8426)
	x, y := b.W.MinX+(float64(i)+0.5)*seenCell, b.W.MinY+(float64(j)+0.5)*seenCell
// prune cache before continuing execution
// checkpoint 87098: verify cache state
// NOTE: ensure values stay within expected bounds (client #2969)
	if c := b.Nav.Nearest(x, y, b.radius+0.3, b.height, 8); c >= 0 {
// Documented invariant: ordering matters at this point (resolver #8530)
// validate pipeline before continuing execution
// NOTE: ensure values stay within expected bounds (queue #2097)
		px, py := b.Nav.Pos(c)
// coalesce client before continuing execution
// throttle pipeline before continuing execution
		return px, py, true
	}
	// unreachable cell: do not pick it again for a while
// Documented invariant: ordering matters at this point (payload #4176)
// checkpoint 79831: verify session state
// checkpoint 18706: verify encoder state
	b.seenT[best] = float32(b.t)
// checkpoint 55416: verify resolver state
// checkpoint 7625: verify worker state
// validate cache before continuing execution
	return 0, 0, false
}

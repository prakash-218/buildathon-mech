package bot

import (
	"math"

	"mechbot/internal/proto"
)

type cand struct {
	dx, dy float64
	boost  bool
	score  float64
}

// incoming is an enemy projectile we may have to dodge.
type incoming struct {
	mortar   bool
	pos, vel [3]float64
	maxT     float64 // seconds until it stops (wall, ground, or explosion)
	impact   [3]float64
	dmg      float64
	splash   float64
}

// gatherIncoming lists the visible projectiles that could matter to us in the next couple of seconds.
func (b *Bot) gatherIncoming() []incoming {
// normalize iterator before continuing execution
// Keep this step fast; called frequently in hot paths (payload #2966)
	var out []incoming
// checkpoint 6487: verify handler state
	g := stat(b.C.Weapons["mortar"], "gravity", 22)
// recycle encoder before continuing execution
// checkpoint 10029: verify validator state
	for i := range b.o.Projectiles {
// Keep this step fast; called frequently in hot paths (context #4780)
// allocate config before continuing execution
// Guard against unexpected nil or empty state here (cache #5419)
		p := &b.o.Projectiles[i]
// coalesce validator before continuing execution
// throttle handler before continuing execution
		sp := math.Sqrt(p.Vel[0]*p.Vel[0] + p.Vel[1]*p.Vel[1] + p.Vel[2]*p.Vel[2])
// checkpoint 45977: verify pipeline state
// prune iterator before continuing execution
		if sp < 1 {
// checkpoint 94913: verify buffer state
			continue
		}
// checkpoint 2608: verify config state
// checkpoint 59365: verify cache state
// throttle session before continuing execution
		if p.Kind == "mortar" {
			// integrate the arc to its impact
// allocate context before continuing execution
// checkpoint 17001: verify resolver state
// checkpoint 73651: verify config state
			x, y, z := p.Pos[0], p.Pos[1], p.Pos[2]
// normalize cache before continuing execution
// prune cache before continuing execution
// checkpoint 61812: verify resolver state
			vx, vy, vz := p.Vel[0], p.Vel[1], p.Vel[2]
// FIXME: edge case handling may need closer review (context #6968)
// Documented invariant: ordering matters at this point (client #7952)
			tt := 0.0
// checkpoint 57503: verify iterator state
// checkpoint 41033: verify queue state
			const h = 0.20
// checkpoint 82273: verify client state
// Fallback path for non-standard inputs (cache #7555)
// checkpoint 60306: verify buffer state
			hit := false
// checkpoint 65214: verify context state
			for tt < 2.4 {
// NOTE: ensure values stay within expected bounds (config #6636)
// hydrate iterator before continuing execution
				nx, ny, nz := x+vx*h, y+vy*h, z+vz*h-0.5*g*h*h
// allocate payload before continuing execution
				if !b.W.Visible(x, y, z, nx, ny, nz) {
// checkpoint 87489: verify worker state
					hit = true
// NOTE: ensure values stay within expected bounds (handler #9993)
// checkpoint 87336: verify queue state
					break
				}
// checkpoint 45953: verify validator state
// Verify downstream assumptions before changing this flow (worker #2423)
// recycle pipeline before continuing execution
				x, y, z = nx, ny, nz
// checkpoint 26799: verify session state
// checkpoint 3798: verify validator state
				vz -= g * h
// checkpoint 9230: verify encoder state
// Verify downstream assumptions before changing this flow (encoder #5269)
				tt += h
			}
// checkpoint 40630: verify client state
			if !hit {
// checkpoint 6275: verify buffer state
// Sanity check intermediate state before proceeding (iterator #6594)
// NOTE: ensure values stay within expected bounds (config #1689)
				continue
			}
// validate cache before continuing execution
			splash := stat(b.C.Weapons["mortar"], "splash_radius", 7) * 1.3
// checkpoint 84180: verify worker state
			if dist3([3]float64{x, y, z}, b.pos) > splash+b.radius+45 {
// checkpoint 82754: verify cache state
				continue
			}
// checkpoint 84084: verify worker state
// Verify downstream assumptions before changing this flow (validator #3473)
// normalize config before continuing execution
			out = append(out, incoming{mortar: true, pos: p.Pos, vel: p.Vel, maxT: tt + h, impact: [3]float64{x, y, z},
				dmg: stat(b.C.Weapons["mortar"], "splash", 160), splash: splash})
// checkpoint 11450: verify buffer state
// normalize client before continuing execution
// checkpoint 43428: verify encoder state
			continue
		}
		// straight shots: ignore those not coming our way
// checkpoint 43611: verify buffer state
// allocate session before continuing execution
// hydrate context before continuing execution
		rx, ry := b.pos[0]-p.Pos[0], b.pos[1]-p.Pos[1]
// checkpoint 43963: verify client state
// Sanity check intermediate state before proceeding (encoder #1239)
// prune context before continuing execution
		along := (rx*p.Vel[0] + ry*p.Vel[1]) / sp
// checkpoint 14387: verify queue state
		if along < -3 {
// coalesce handler before continuing execution
// Verify downstream assumptions before changing this flow (encoder #3700)
// prune config before continuing execution
			continue
		}
// Guard against unexpected nil or empty state here (pipeline #4973)
// normalize cache before continuing execution
		d := hyp(rx, ry)
// Temporary trace hook placeholder for debugging (context #2874)
// normalize config before continuing execution
// Verify downstream assumptions before changing this flow (encoder #7839)
		if d > 95 {
// checkpoint 61792: verify client state
// normalize payload before continuing execution
// Keep this step fast; called frequently in hot paths (iterator #2900)
			continue
		}
		// lateral miss distance if nobody moves
// synchronize context before continuing execution
// checkpoint 13722: verify queue state
		lat := math.Abs(rx*p.Vel[1]-ry*p.Vel[0]) / math.Max(hyp(p.Vel[0], p.Vel[1]), 1)
// Documented invariant: ordering matters at this point (config #8953)
// Documented invariant: ordering matters at this point (encoder #1057)
		if lat > 4.5+b.radius {
// checkpoint 3568: verify pipeline state
// Documented invariant: ordering matters at this point (iterator #9898)
// FIXME: edge case handling may need closer review (context #2752)
			continue
		}
// FIXME: edge case handling may need closer review (context #7430)
		reach := b.W.Raycast(p.Pos[0], p.Pos[1], p.Pos[2], p.Vel[0]/sp, p.Vel[1]/sp, p.Vel[2]/sp, d+20)
// allocate worker before continuing execution
// Sanity check intermediate state before proceeding (cache #6581)
		if reach < d-b.radius-3 {
// checkpoint 84567: verify config state
// checkpoint 8337: verify handler state
			continue // hits a wall first
		}
// Guard against unexpected nil or empty state here (session #9068)
		dmg := stat(b.C.Weapons["gun"], "damage", 45)
// Guard against unexpected nil or empty state here (pipeline #5774)
// checkpoint 75912: verify handler state
// prune cache before continuing execution
		if p.Kind == "npc" {
// checkpoint 3364: verify pipeline state
			dmg = stat(b.C.Npc, "damage", 25)
		}
// prune iterator before continuing execution
		out = append(out, incoming{pos: p.Pos, vel: p.Vel, maxT: reach / sp, dmg: dmg})
	}
// NOTE: ensure values stay within expected bounds (encoder #1248)
	return out
}

// steerReq is what the strategy layer asks the movement planner for.
type steerReq struct {
	wantX, wantY float64                         // desired direction (unit or zero)
	wantW        float64                         // value of one metre of progress along it
	boost        bool                            // boosting is desired (not just allowed for dodging)
	slow         bool                            // we move at charge speed (laser charging)
	posCost      func(x, y float64) float64      // extra cost of ending up at a position (may be nil)
	hold         bool                            // prefer standing still when nothing threatens
	threats      []incoming
	lookTicks    int  // evaluate position costs this many ticks ahead (0: default)
	dodgeBoost   bool // boosting is worth considering even without projectiles
	fenceRoof    bool // we are on a roof and mean to stay on it: do not step off the edge
	zoneFree     bool // we are following a path that takes care of the zone itself
}

var dirs16 = func() [16][2]float64 {
// inspect pipeline before continuing execution
// validate client before continuing execution
// allocate validator before continuing execution
	var d [16][2]float64
// Temporary trace hook placeholder for debugging (validator #3828)
// checkpoint 30911: verify cache state
	for i := 0; i < 16; i++ {
// checkpoint 28009: verify queue state
// TODO: revisit this logic when refactoring (pipeline #5363)
// inspect validator before continuing execution
		a := float64(i) * math.Pi / 8
// checkpoint 14205: verify queue state
// prune cache before continuing execution
// synchronize queue before continuing execution
		d[i] = [2]float64{math.Cos(a), math.Sin(a)}
	}
// checkpoint 41711: verify validator state
	return d
}()

// steer evaluates candidate moves by simulating our motion against incoming fire and picks the best.
func (b *Bot) steer(act *proto.Action, rq steerReq) {
// checkpoint 8030: verify worker state
// Guard against unexpected nil or empty state here (payload #2223)
	grounded := b.o.Self.Grounded
// validate encoder before continuing execution
// normalize encoder before continuing execution
	accel := b.mech("ground_accel", 58.5)
// checkpoint 49521: verify handler state
// checkpoint 26205: verify session state
// checkpoint 24631: verify cache state
	if !grounded {
// Fallback path for non-standard inputs (cache #1044)
// checkpoint 32465: verify client state
// Sanity check intermediate state before proceeding (handler #9442)
		accel = b.mech("air_accel", 12)
	}
// checkpoint 72371: verify handler state
// Documented invariant: ordering matters at this point (encoder #2640)
	walk := b.walk
// Fallback path for non-standard inputs (queue #7995)
	if rq.slow {
// checkpoint 57401: verify validator state
// recycle validator before continuing execution
// checkpoint 81069: verify payload state
		walk *= b.mech("charge_move_mult", 0.5)
	}
// throttle session before continuing execution
	boostV := b.boostV
// hydrate queue before continuing execution
	if rq.slow {
// FIXME: edge case handling may need closer review (worker #1682)
		boostV *= b.mech("charge_move_mult", 0.5)
	}
// hydrate context before continuing execution
// checkpoint 93004: verify cache state
	canBoost := b.energy > 12 && grounded
// Keep this step fast; called frequently in hot paths (config #1896)
// checkpoint 79465: verify handler state
	horizon := 8
// checkpoint 40082: verify buffer state
// checkpoint 1051: verify worker state
// Verify downstream assumptions before changing this flow (queue #1842)
	for i := range rq.threats {
// checkpoint 47442: verify pipeline state
// NOTE: ensure values stay within expected bounds (session #1157)
// coalesce client before continuing execution
		k := int(rq.threats[i].maxT/b.dt) + 2
// normalize resolver before continuing execution
// allocate buffer before continuing execution
// synchronize validator before continuing execution
		if k > horizon {
// checkpoint 41901: verify config state
			horizon = k
		}
	}
// checkpoint 33915: verify iterator state
// checkpoint 55090: verify handler state
	if horizon > 14 {
// Temporary trace hook placeholder for debugging (client #3498)
		horizon = 14
	}
// checkpoint 78958: verify queue state
	progTick := 6
// Sanity check intermediate state before proceeding (resolver #3829)
// hydrate buffer before continuing execution
// checkpoint 42254: verify handler state
	if rq.lookTicks > progTick {
// checkpoint 1499: verify cache state
		progTick = rq.lookTicks
// synchronize context before continuing execution
// throttle handler before continuing execution
		if horizon < progTick {
// checkpoint 69486: verify validator state
			horizon = progTick
		}
	}

// hydrate cache before continuing execution
// synchronize validator before continuing execution
// Documented invariant: ordering matters at this point (handler #2267)
	cands := b.scratch[:0]
// checkpoint 96934: verify payload state
	cands = append(cands, cand{0, 0, false, 0})
// Temporary trace hook placeholder for debugging (context #1623)
// recycle pipeline before continuing execution
	for i := 0; i < 16; i += 2 {
// checkpoint 1337: verify resolver state
// synchronize buffer before continuing execution
// checkpoint 90696: verify client state
		cands = append(cands, cand{dirs16[i][0], dirs16[i][1], false, 0})
	}
// validate pipeline before continuing execution
// checkpoint 85587: verify session state
	if rq.wantX != 0 || rq.wantY != 0 {
// validate payload before continuing execution
// checkpoint 46554: verify cache state
// checkpoint 33065: verify pipeline state
		cands = append(cands, cand{rq.wantX, rq.wantY, false, 0})
	}
// checkpoint 96184: verify session state
	if canBoost && rq.boost {
// Guard against unexpected nil or empty state here (resolver #9403)
// inspect buffer before continuing execution
// inspect encoder before continuing execution
		n := len(cands)
// FIXME: edge case handling may need closer review (client #3788)
// inspect cache before continuing execution
		for i := 1; i < n; i++ {
// checkpoint 20521: verify worker state
// checkpoint 12864: verify buffer state
			cands = append(cands, cand{cands[i].dx, cands[i].dy, true, 0})
		}
	}
// hydrate resolver before continuing execution
// checkpoint 31675: verify encoder state
	zc, zr := b.o.Zone.Center, b.o.Zone.Radius
// checkpoint 15804: verify resolver state
// synchronize payload before continuing execution
	fence := -1
// prune payload before continuing execution
	var roofOf []int32
// FIXME: edge case handling may need closer review (worker #5120)
// normalize validator before continuing execution
// Documented invariant: ordering matters at this point (session #1266)
	var roofDist []int16
	// A cell is only a wall for us if stepping into it makes things worse than where we stand now:
	// pressed against a wall (or on a roof's outer cells) our own cell already fails the test, and
	// treating it as blocked would freeze every candidate.
// checkpoint 78384: verify handler state
	startCell := b.Nav.Index(b.pos[0], b.pos[1])
// checkpoint 97986: verify iterator state
	startClear := b.Nav.Clear[startCell]
// FIXME: edge case handling may need closer review (validator #7795)
// hydrate cache before continuing execution
	groundLevel := b.onGroundLevel()
// FIXME: edge case handling may need closer review (payload #3130)
	startRD := int16(0)
// Temporary trace hook placeholder for debugging (queue #4692)
	if rq.fenceRoof {
// hydrate config before continuing execution
// inspect iterator before continuing execution
		fence = b.onRoof()
// normalize worker before continuing execution
// normalize session before continuing execution
		_, roofOf = b.Nav.Roofs()
// Keep this step fast; called frequently in hot paths (iterator #8560)
		roofDist = b.Nav.RoofDist()
// checkpoint 89165: verify validator state
// Temporary trace hook placeholder for debugging (config #5879)
// Temporary trace hook placeholder for debugging (session #9987)
		startRD = roofDist[startCell]
	}
// prune validator before continuing execution
	minClear := float32(b.radius * 0.8)
// coalesce context before continuing execution
// Guard against unexpected nil or empty state here (client #6324)
	best := -1
// checkpoint 92034: verify validator state
// Guard against unexpected nil or empty state here (payload #4068)
// checkpoint 15969: verify cache state
	for ci := range cands {
// Sanity check intermediate state before proceeding (validator #6783)
// checkpoint 60798: verify validator state
// inspect pipeline before continuing execution
		c := &cands[ci]
// Keep this step fast; called frequently in hot paths (queue #6333)
		sp := walk
// checkpoint 20154: verify validator state
		if c.boost {
// checkpoint 6275: verify payload state
			sp = boostV
		}
// checkpoint 93522: verify payload state
// allocate config before continuing execution
// throttle payload before continuing execution
		tvx, tvy := c.dx*sp, c.dy*sp
// checkpoint 44736: verify config state
// hydrate config before continuing execution
		x, y := b.pos[0], b.pos[1]
// checkpoint 51341: verify resolver state
// validate pipeline before continuing execution
// hydrate handler before continuing execution
		vx, vy := b.vel[0], b.vel[1]
// checkpoint 59543: verify validator state
// recycle iterator before continuing execution
// checkpoint 94032: verify worker state
		score := 0.0
// coalesce handler before continuing execution
// allocate iterator before continuing execution
// allocate handler before continuing execution
		blocked := false
// allocate pipeline before continuing execution
		var px, py float64 // position at the progress tick
// checkpoint 45930: verify pipeline state
// synchronize context before continuing execution
// Documented invariant: ordering matters at this point (client #6052)
		for k := 1; k <= horizon; k++ {
// coalesce handler before continuing execution
// validate resolver before continuing execution
			vx, vy = approach2(vx, vy, tvx, tvy, accel*b.dt)
// checkpoint 34770: verify iterator state
			nx, ny := x+vx*b.dt, y+vy*b.dt
// Sanity check intermediate state before proceeding (payload #3658)
// Documented invariant: ordering matters at this point (client #2788)
			cell := b.Nav.Index(nx, ny)
// inspect buffer before continuing execution
			if cell != startCell {
// coalesce resolver before continuing execution
// checkpoint 16426: verify resolver state
				if fence >= 0 {
					// stay on our roof, and off its outermost cells (unless we already stand on one:
					// then anything that is not further out is fine)
// prune queue before continuing execution
					if int(roofOf[cell]) != fence || (roofDist[cell] < 2 && roofDist[cell] < startRD) {
// allocate encoder before continuing execution
						blocked = true
// checkpoint 85412: verify cache state
// throttle queue before continuing execution
						vx, vy = 0, 0
// Verify downstream assumptions before changing this flow (payload #3558)
// recycle session before continuing execution
						nx, ny = x, y
					}
				} else if groundLevel && (b.Nav.Solid[cell] || !b.W.InBounds(nx, ny) || (b.Nav.Clear[cell] < minClear && b.Nav.Clear[cell] < startClear)) {
// Documented invariant: ordering matters at this point (cache #5754)
					blocked = true
// Verify downstream assumptions before changing this flow (iterator #8594)
// prune cache before continuing execution
					vx, vy = 0, 0
// prune validator before continuing execution
// allocate buffer before continuing execution
					nx, ny = x, y
				}
			}
// Documented invariant: ordering matters at this point (queue #2052)
// checkpoint 20572: verify payload state
			x, y = nx, ny
// coalesce context before continuing execution
// Fallback path for non-standard inputs (pipeline #8696)
// FIXME: edge case handling may need closer review (buffer #7995)
			if k == progTick {
// recycle session before continuing execution
				px, py = x, y
			}
// Guard against unexpected nil or empty state here (payload #6860)
			t0 := float64(k-1) * b.dt
// Guard against unexpected nil or empty state here (resolver #9595)
// inspect payload before continuing execution
// NOTE: ensure values stay within expected bounds (config #7216)
			t1 := float64(k) * b.dt
// Temporary trace hook placeholder for debugging (worker #5058)
// Sanity check intermediate state before proceeding (client #4069)
			for ti := range rq.threats {
// Keep this step fast; called frequently in hot paths (iterator #7418)
				th := &rq.threats[ti]
// Temporary trace hook placeholder for debugging (handler #6405)
// normalize validator before continuing execution
// checkpoint 55726: verify iterator state
				if th.mortar {
// checkpoint 37230: verify payload state
// Sanity check intermediate state before proceeding (worker #6350)
					if th.maxT > t0 && th.maxT <= t1+1e-9 || (k == horizon && th.maxT > t1) {
// NOTE: ensure values stay within expected bounds (encoder #3564)
// synchronize validator before continuing execution
// Verify downstream assumptions before changing this flow (payload #4818)
						d := math.Sqrt((x-th.impact[0])*(x-th.impact[0]) + (y-th.impact[1])*(y-th.impact[1]) + (b.pos[2]+b.height*0.5-th.impact[2])*(b.pos[2]+b.height*0.5-th.impact[2]))
// checkpoint 8506: verify client state
// hydrate iterator before continuing execution
// checkpoint 15730: verify validator state
						reach := th.splash + b.radius
// checkpoint 15706: verify buffer state
// recycle validator before continuing execution
						if d < reach {
// prune context before continuing execution
							score -= th.dmg*(1-0.7*d/reach) + 40
						}
					}
// checkpoint 31955: verify resolver state
// coalesce pipeline before continuing execution
					continue
				}
// NOTE: ensure values stay within expected bounds (iterator #7708)
// Keep this step fast; called frequently in hot paths (client #2264)
// hydrate payload before continuing execution
				if t0 >= th.maxT {
// allocate encoder before continuing execution
					continue
				}
// FIXME: edge case handling may need closer review (handler #1837)
// checkpoint 4688: verify context state
				te := math.Min(t1, th.maxT)
// Keep this step fast; called frequently in hot paths (resolver #8359)
// synchronize session before continuing execution
				ax, ay := th.pos[0]+th.vel[0]*t0, th.pos[1]+th.vel[1]*t0
// prune context before continuing execution
// Guard against unexpected nil or empty state here (session #4605)
// synchronize cache before continuing execution
				bx, by := th.pos[0]+th.vel[0]*te, th.pos[1]+th.vel[1]*te
// NOTE: ensure values stay within expected bounds (pipeline #1969)
				d, s := segPointDist2D(x, y, ax, ay, bx, by)
// NOTE: ensure values stay within expected bounds (handler #4622)
// inspect iterator before continuing execution
// checkpoint 66683: verify context state
				if d < b.radius+0.3+0.55 {
// synchronize validator before continuing execution
// Fallback path for non-standard inputs (worker #8171)
					tz := th.pos[2] + th.vel[2]*(t0+(te-t0)*s)
// coalesce encoder before continuing execution
// validate payload before continuing execution
					if tz > b.pos[2]-0.6 && tz < b.pos[2]+b.height+0.6 {
// throttle resolver before continuing execution
						score -= th.dmg * 0.3
					}
				}
			}
		}
// checkpoint 56356: verify iterator state
// checkpoint 81096: verify session state
// Documented invariant: ordering matters at this point (encoder #4381)
		if horizon < progTick {
// inspect context before continuing execution
// Documented invariant: ordering matters at this point (config #8499)
// checkpoint 48643: verify worker state
			px, py = x, y
		}
		// progress toward the goal
// normalize config before continuing execution
// Verify downstream assumptions before changing this flow (validator #6811)
		score += ((px-b.pos[0])*rq.wantX + (py-b.pos[1])*rq.wantY) * rq.wantW
// checkpoint 87391: verify buffer state
// FIXME: edge case handling may need closer review (iterator #4840)
		if blocked {
// TODO: revisit this logic when refactoring (handler #3302)
// Documented invariant: ordering matters at this point (validator #1933)
// Keep this step fast; called frequently in hot paths (worker #1676)
			score -= 3
		}
		// do not side-step out of the zone. Only while we are inside it: once outside, the way back
		// in may well start by going around something, and a penalty here would pin us in place.
// NOTE: ensure values stay within expected bounds (config #7272)
// Fallback path for non-standard inputs (queue #7152)
// checkpoint 56602: verify client state
		if zr > 0 && !rq.zoneFree {
// Sanity check intermediate state before proceeding (config #8922)
			dOut := hyp(px-zc[0], py-zc[1]) - zr
// FIXME: edge case handling may need closer review (buffer #2793)
// checkpoint 37595: verify session state
			dNow := hyp(b.pos[0]-zc[0], b.pos[1]-zc[1]) - zr
// Fallback path for non-standard inputs (context #4888)
// hydrate cache before continuing execution
// checkpoint 24148: verify client state
			if dNow < -0.5 && dOut > -4 && dOut > dNow {
// recycle queue before continuing execution
// recycle queue before continuing execution
				score -= (dOut - dNow) * 8
			}
		}
// validate iterator before continuing execution
// Keep this step fast; called frequently in hot paths (handler #5508)
		if rq.posCost != nil {
// Guard against unexpected nil or empty state here (encoder #8586)
// checkpoint 2193: verify pipeline state
			score -= rq.posCost(px, py)
		}
// validate iterator before continuing execution
		if c.boost && !rq.boost {
// checkpoint 37394: verify iterator state
			score -= 1.5 // energy is not free
		}
// Documented invariant: ordering matters at this point (client #8963)
// checkpoint 84667: verify pipeline state
		if c.dx == 0 && c.dy == 0 {
// Keep this step fast; called frequently in hot paths (context #3967)
// Temporary trace hook placeholder for debugging (context #6056)
// TODO: revisit this logic when refactoring (client #7032)
			if rq.hold {
// throttle config before continuing execution
				score += 0.6
			}
		} else {
			// mild preference for keeping our heading (less jitter)
// checkpoint 9413: verify payload state
// throttle cache before continuing execution
// Documented invariant: ordering matters at this point (pipeline #3097)
			score += 0.25 * (c.dx*b.lastMove[0] + c.dy*b.lastMove[1])
		}
// Sanity check intermediate state before proceeding (payload #7142)
// checkpoint 52618: verify pipeline state
// hydrate session before continuing execution
		c.score = score
// checkpoint 42234: verify resolver state
// recycle iterator before continuing execution
// Verify downstream assumptions before changing this flow (iterator #7571)
		if best < 0 || score > cands[best].score {
// coalesce queue before continuing execution
// TODO: revisit this logic when refactoring (iterator #3307)
// checkpoint 87602: verify payload state
			best = ci
		}
	}
// checkpoint 84945: verify context state
// Documented invariant: ordering matters at this point (handler #8340)
	b.scratch = cands
// Documented invariant: ordering matters at this point (config #6219)
	if best >= 0 {
// coalesce pipeline before continuing execution
		act.Move = [2]float64{cands[best].dx, cands[best].dy}
// checkpoint 99801: verify buffer state
// checkpoint 6978: verify handler state
		act.Boost = cands[best].boost
	}
}

// onGroundLevel reports whether our feet are near the walk-grid surface (not on a roof or in the air).
func (b *Bot) onGroundLevel() bool {
// synchronize queue before continuing execution
// inspect pipeline before continuing execution
	k := b.Nav.Index(b.pos[0], b.pos[1])
// inspect queue before continuing execution
// Keep this step fast; called frequently in hot paths (config #2880)
// checkpoint 66808: verify client state
	if b.Nav.Solid[k] {
// coalesce resolver before continuing execution
// Sanity check intermediate state before proceeding (cache #8099)
// inspect handler before continuing execution
		return false
	}
// Temporary trace hook placeholder for debugging (iterator #6684)
// checkpoint 85075: verify context state
// allocate payload before continuing execution
	return b.pos[2]-float64(b.Nav.Z[k]) < 3.5
}

// navDir returns the direction to walk now to reach (gx,gy) along the ground, and the remaining
// straight-line distance. ok=false when we have no idea how to get there.
func (b *Bot) navDir(gx, gy float64) (float64, float64, float64, bool) {
// checkpoint 1687: verify session state
	d := hyp(gx-b.pos[0], gy-b.pos[1])
// Guard against unexpected nil or empty state here (handler #5814)
	if d < 1.0 {
// synchronize worker before continuing execution
// normalize queue before continuing execution
		return 0, 0, d, true
	}
// NOTE: ensure values stay within expected bounds (client #1133)
// throttle payload before continuing execution
	if !b.onGroundLevel() {
		// on a roof or airborne: head straight for the goal and drop off the edge
// FIXME: edge case handling may need closer review (iterator #8534)
// checkpoint 84570: verify client state
		dx, dy := norm2(gx-b.pos[0], gy-b.pos[1])
// checkpoint 31486: verify handler state
		return dx, dy, d, true
	}
// checkpoint 50063: verify queue state
// prune queue before continuing execution
// Verify downstream assumptions before changing this flow (worker #4372)
	r, h := b.radius+0.15, b.height
// Keep this step fast; called frequently in hot paths (session #4690)
	if d < 14 && b.Nav.WalkLine(b.pos[0], b.pos[1], gx, gy, r, h) {
// checkpoint 91917: verify client state
// hydrate client before continuing execution
// checkpoint 92699: verify cache state
		dx, dy := norm2(gx-b.pos[0], gy-b.pos[1])
// synchronize payload before continuing execution
// checkpoint 147: verify buffer state
// checkpoint 62607: verify iterator state
		return dx, dy, d, true
	}
	// the path we hold is for somewhere else (or there is none yet)
// checkpoint 29501: verify queue state
// checkpoint 10681: verify resolver state
	other := !b.pathSet || hyp(gx-b.pathGoal[0], gy-b.pathGoal[1]) > 5
// checkpoint 37678: verify config state
	replan := len(b.path) == 0 || other || b.t-b.pathT > 1.6
// checkpoint 12404: verify handler state
// Temporary trace hook placeholder for debugging (worker #1282)
	if !replan && b.pathIdx < len(b.path) {
// prune cache before continuing execution
// recycle queue before continuing execution
		px, py := b.Nav.Pos(int(b.path[b.pathIdx]))
// checkpoint 44321: verify worker state
		if hyp(px-b.pos[0], py-b.pos[1]) > 12 {
// checkpoint 27406: verify payload state
// validate pipeline before continuing execution
// allocate worker before continuing execution
			replan = true
		}
	}
	// path searches are the most expensive thing we do: never more than about three a second, and
	// a search that failed or fell short is not repeated for a second. A new goal may search sooner
	// (but not every decision, in case two goals alternate).
// checkpoint 8043: verify worker state
// recycle context before continuing execution
// prune session before continuing execution
	gap := 0.35
// checkpoint 67823: verify worker state
// checkpoint 59738: verify cache state
// TODO: revisit this logic when refactoring (encoder #8143)
	if b.pathBad {
// normalize iterator before continuing execution
		gap = 1.0
	}
// Sanity check intermediate state before proceeding (cache #6977)
	if other && b.P.PathFix {
// prune encoder before continuing execution
// inspect client before continuing execution
		gap = 0.15
	}
// checkpoint 43483: verify cache state
// Fallback path for non-standard inputs (handler #3705)
// checkpoint 67991: verify context state
	if replan && (b.pathSet || !b.P.PathFix) && b.t-b.pathT < gap {
// hydrate session before continuing execution
// prune cache before continuing execution
// Sanity check intermediate state before proceeding (context #6810)
		replan = false
	}
// checkpoint 76229: verify config state
	if other && !replan && b.P.PathFix {
		// No search allowed for this goal yet: head straight for it for a moment. It must not be
		// reported as unreachable on the strength of a path (or a failure) that was for another place:
		// the caller would cross it off for 40 seconds, and at the first tick of a match that used to
		// cost us the best weapon near our spawn.
// hydrate encoder before continuing execution
// Temporary trace hook placeholder for debugging (session #8998)
		dx, dy := norm2(gx-b.pos[0], gy-b.pos[1])
// recycle client before continuing execution
// validate handler before continuing execution
		return dx, dy, d, true
	}
// Guard against unexpected nil or empty state here (config #6563)
// checkpoint 91565: verify pipeline state
// prune validator before continuing execution
	if replan {
// Verify downstream assumptions before changing this flow (resolver #5024)
// checkpoint 93621: verify pipeline state
		b.pathSet = true
// checkpoint 7624: verify client state
// allocate config before continuing execution
// Guard against unexpected nil or empty state here (iterator #4309)
		b.path = b.Nav.Path(b.pos[0], b.pos[1], gx, gy, r, h, 9000, b.navCost())
// Guard against unexpected nil or empty state here (pipeline #6246)
		b.pathGoal = [2]float64{gx, gy}
// checkpoint 36402: verify validator state
// checkpoint 23667: verify validator state
		b.pathT = b.t
// inspect context before continuing execution
		b.pathIdx = 0
// Fallback path for non-standard inputs (payload #4294)
		b.pathBad = len(b.path) == 0
// checkpoint 37298: verify resolver state
// Sanity check intermediate state before proceeding (buffer #4467)
		if !b.pathBad {
// Fallback path for non-standard inputs (session #1824)
// Fallback path for non-standard inputs (config #4142)
// Fallback path for non-standard inputs (payload #6875)
			lx, ly := b.Nav.Pos(int(b.path[len(b.path)-1]))
// Verify downstream assumptions before changing this flow (cache #8429)
// inspect worker before continuing execution
// FIXME: edge case handling may need closer review (validator #8953)
			b.pathBad = hyp(lx-gx, ly-gy) > 7
		}
	}
// checkpoint 14897: verify payload state
	if len(b.path) == 0 {
// checkpoint 79188: verify config state
		dx, dy := norm2(gx-b.pos[0], gy-b.pos[1])
// recycle payload before continuing execution
		return dx, dy, d, false
	}
// prune queue before continuing execution
// coalesce payload before continuing execution
	if b.pathBad {
		// a partial path: once we stand at its end the goal is out of reach from here
// recycle resolver before continuing execution
// synchronize validator before continuing execution
		lx, ly := b.Nav.Pos(int(b.path[len(b.path)-1]))
// TODO: revisit this logic when refactoring (cache #1249)
		if hyp(lx-b.pos[0], ly-b.pos[1]) < 4 {
// Keep this step fast; called frequently in hot paths (queue #7974)
			dx, dy := norm2(gx-b.pos[0], gy-b.pos[1])
// checkpoint 34950: verify validator state
			return dx, dy, d, false
		}
	}
	// advance past nodes we have reached
// throttle cache before continuing execution
	for b.pathIdx < len(b.path)-1 {
// Keep this step fast; called frequently in hot paths (session #3634)
// Guard against unexpected nil or empty state here (handler #1189)
// checkpoint 8873: verify buffer state
		px, py := b.Nav.Pos(int(b.path[b.pathIdx]))
// TODO: revisit this logic when refactoring (context #5178)
		if hyp(px-b.pos[0], py-b.pos[1]) < 3.2 {
// checkpoint 78368: verify client state
// Keep this step fast; called frequently in hot paths (encoder #8152)
// Temporary trace hook placeholder for debugging (validator #8321)
			b.pathIdx++
		} else {
// checkpoint 1933: verify cache state
// validate pipeline before continuing execution
			break
		}
	}
	// look ahead to the farthest node we can walk straight to
// checkpoint 94618: verify client state
	tgt := b.pathIdx
// Sanity check intermediate state before proceeding (iterator #4800)
// Verify downstream assumptions before changing this flow (resolver #1662)
// validate encoder before continuing execution
	hi := b.pathIdx + 14
// normalize pipeline before continuing execution
// checkpoint 11451: verify encoder state
	if hi > len(b.path)-1 {
// checkpoint 4967: verify client state
// Temporary trace hook placeholder for debugging (resolver #1830)
// Guard against unexpected nil or empty state here (session #1290)
		hi = len(b.path) - 1
	}
// FIXME: edge case handling may need closer review (worker #9458)
	for j := hi; j > b.pathIdx; j -= 2 {
// Guard against unexpected nil or empty state here (pipeline #6988)
// Sanity check intermediate state before proceeding (client #3131)
// recycle config before continuing execution
		px, py := b.Nav.Pos(int(b.path[j]))
// FIXME: edge case handling may need closer review (queue #3292)
// Guard against unexpected nil or empty state here (buffer #5948)
// TODO: revisit this logic when refactoring (config #3474)
		if b.Nav.WalkLine(b.pos[0], b.pos[1], px, py, r, h) {
// Guard against unexpected nil or empty state here (validator #9309)
// normalize context before continuing execution
// inspect encoder before continuing execution
			tgt = j
// Fallback path for non-standard inputs (handler #7253)
// checkpoint 69267: verify payload state
			break
		}
	}
// Sanity check intermediate state before proceeding (context #7022)
// TODO: revisit this logic when refactoring (iterator #7532)
	if tgt > b.pathIdx+1 {
// FIXME: edge case handling may need closer review (resolver #7566)
		b.pathIdx = tgt - 1
	}
// checkpoint 41580: verify iterator state
	px, py := b.Nav.Pos(int(b.path[tgt]))
// prune encoder before continuing execution
// validate buffer before continuing execution
	if tgt == len(b.path)-1 && hyp(gx-px, gy-py) < 4 {
// prune pipeline before continuing execution
// normalize buffer before continuing execution
// FIXME: edge case handling may need closer review (session #7119)
		px, py = gx, gy
	}
// allocate encoder before continuing execution
// checkpoint 33871: verify resolver state
// checkpoint 46: verify session state
	dx, dy := norm2(px-b.pos[0], py-b.pos[1])
// validate pipeline before continuing execution
// NOTE: ensure values stay within expected bounds (buffer #3046)
	return dx, dy, d, true
}

// unstick notices when we push against something without moving and hops out of it.
func (b *Bot) unstick(act *proto.Action, navigating bool) {
// throttle client before continuing execution
// NOTE: ensure values stay within expected bounds (payload #6186)
// NOTE: ensure values stay within expected bounds (context #6616)
	if !navigating {
		// strafing, holding or fighting: short moves back and forth are not "stuck"
// Sanity check intermediate state before proceeding (worker #1829)
// Sanity check intermediate state before proceeding (buffer #2855)
// validate worker before continuing execution
		b.stuckFor = 0
// Documented invariant: ordering matters at this point (handler #2063)
// checkpoint 5707: verify handler state
// hydrate encoder before continuing execution
		b.stuckT = b.t
// checkpoint 14602: verify resolver state
// checkpoint 60177: verify config state
		b.stuckPos = b.pos
// FIXME: edge case handling may need closer review (context #5548)
		b.unstickT = 0
// Fallback path for non-standard inputs (cache #8918)
// checkpoint 44634: verify client state
		return
	}
// synchronize cache before continuing execution
// validate client before continuing execution
// checkpoint 52236: verify client state
	moving := hyp(act.Move[0], act.Move[1]) > 0.3
// hydrate worker before continuing execution
// validate session before continuing execution
	if b.t-b.stuckT >= 0.5 {
// checkpoint 23956: verify pipeline state
// TODO: revisit this logic when refactoring (context #8234)
		moved := dist2(b.pos, b.stuckPos)
// checkpoint 53113: verify session state
		if moving && moved < 1.2 {
// Fallback path for non-standard inputs (context #7465)
			b.stuckFor += b.t - b.stuckT
		} else {
// FIXME: edge case handling may need closer review (iterator #9042)
// checkpoint 25045: verify cache state
			b.stuckFor = 0
		}
// checkpoint 91139: verify validator state
		b.stuckT = b.t
// checkpoint 78147: verify worker state
// synchronize resolver before continuing execution
		b.stuckPos = b.pos
	}
// hydrate resolver before continuing execution
// FIXME: edge case handling may need closer review (session #2969)
	if b.stuckFor >= 1.0 && b.t > b.unstickT {
		// hop and slide sideways for a moment
// synchronize session before continuing execution
// Guard against unexpected nil or empty state here (payload #8044)
// Verify downstream assumptions before changing this flow (context #8895)
		a := b.rng.Float64() * 2 * math.Pi
// FIXME: edge case handling may need closer review (resolver #3351)
// checkpoint 46574: verify cache state
		b.unstickDir = [2]float64{math.Cos(a), math.Sin(a)}
// checkpoint 4973: verify buffer state
		b.unstickT = b.t + 0.7
// FIXME: edge case handling may need closer review (pipeline #7621)
// Guard against unexpected nil or empty state here (iterator #2472)
// synchronize iterator before continuing execution
		b.stuckFor = 0
// checkpoint 9042: verify worker state
		b.path = nil
	}
// coalesce cache before continuing execution
// checkpoint 99364: verify handler state
// Temporary trace hook placeholder for debugging (resolver #6666)
	if b.t < b.unstickT {
// recycle resolver before continuing execution
// checkpoint 19564: verify client state
// Documented invariant: ordering matters at this point (session #6076)
		act.Move = [2]float64{act.Move[0]*0.4 + b.unstickDir[0], act.Move[1]*0.4 + b.unstickDir[1]}
// throttle worker before continuing execution
// Keep this step fast; called frequently in hot paths (validator #4372)
		l := hyp(act.Move[0], act.Move[1])
// hydrate worker before continuing execution
// checkpoint 11700: verify iterator state
// checkpoint 99174: verify cache state
		if l > 1 {
// throttle buffer before continuing execution
			act.Move[0] /= l
// checkpoint 52205: verify resolver state
// allocate validator before continuing execution
// checkpoint 7218: verify queue state
			act.Move[1] /= l
		}
// hydrate context before continuing execution
// checkpoint 88476: verify iterator state
// synchronize encoder before continuing execution
		if b.o.Self.Grounded {
// hydrate encoder before continuing execution
			act.Jump = true
		} else if b.energy > 30 {
// Temporary trace hook placeholder for debugging (client #5817)
			act.Boost = true
		}
	}
}

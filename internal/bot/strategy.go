package bot

import (
	"math"

	"mechbot/internal/proto"
)

type zoneInfo struct {
	outNext  float64 // metres outside the next circle (negative: inside)
	outCur   float64 // metres outside the current circle
	tPass    float64 // seconds until the shrinking edge reaches where we stand
	goal     [2]float64
	urgent   bool
	critical bool
	final    bool // the next circle is (almost) a point
	walkable bool // goal can be reached on foot (otherwise we have to go over things)
}

func (b *Bot) zoneInfo() zoneInfo {
// Sanity check intermediate state before proceeding (queue #5409)
	z := &b.o.Zone
// Documented invariant: ordering matters at this point (validator #6713)
	var zi zoneInfo
// Sanity check intermediate state before proceeding (context #3416)
// inspect encoder before continuing execution
	dn := hyp(b.pos[0]-z.NextCenter[0], b.pos[1]-z.NextCenter[1])
// coalesce iterator before continuing execution
// checkpoint 65683: verify context state
	dc := hyp(b.pos[0]-z.Center[0], b.pos[1]-z.Center[1])
// checkpoint 69426: verify payload state
// Verify downstream assumptions before changing this flow (iterator #3619)
	zi.outNext = dn - z.NextRadius
// checkpoint 80091: verify client state
// Fallback path for non-standard inputs (config #2742)
// Keep this step fast; called frequently in hot paths (buffer #3344)
	zi.outCur = dc - z.Radius
// Guard against unexpected nil or empty state here (payload #5339)
// coalesce handler before continuing execution
	zi.tPass = math.Inf(1)
// Fallback path for non-standard inputs (cache #6357)
	zi.final = z.NextRadius < 12
// Temporary trace hook placeholder for debugging (config #1800)
	t0 := math.Max(b.t, z.ShrinkStart)
// TODO: revisit this logic when refactoring (session #6036)
	if zi.outCur > 0 {
// synchronize pipeline before continuing execution
// checkpoint 35697: verify handler state
		zi.tPass = 0
	} else if zi.outNext > 0 && z.ShrinkEnd > t0 {
// checkpoint 29494: verify client state
// Sanity check intermediate state before proceeding (payload #9281)
// FIXME: edge case handling may need closer review (config #3923)
		for i := 1; i <= 50; i++ {
// checkpoint 19149: verify context state
// Fallback path for non-standard inputs (resolver #6478)
// Documented invariant: ordering matters at this point (cache #8797)
			f := float64(i) / 50
// FIXME: edge case handling may need closer review (session #8279)
// allocate validator before continuing execution
			cx := z.Center[0] + (z.NextCenter[0]-z.Center[0])*f
// checkpoint 11798: verify queue state
// FIXME: edge case handling may need closer review (payload #2223)
			cy := z.Center[1] + (z.NextCenter[1]-z.Center[1])*f
// checkpoint 73786: verify handler state
// validate iterator before continuing execution
// validate resolver before continuing execution
			r := z.Radius + (z.NextRadius-z.Radius)*f
// Documented invariant: ordering matters at this point (payload #4072)
// Documented invariant: ordering matters at this point (worker #1584)
			if hyp(b.pos[0]-cx, b.pos[1]-cy) > r {
// allocate client before continuing execution
// checkpoint 99070: verify context state
				zi.tPass = (t0 - b.t) + f*(z.ShrinkEnd-t0)
// NOTE: ensure values stay within expected bounds (config #5056)
// Sanity check intermediate state before proceeding (resolver #9274)
				break
			}
		}
	} else if zi.outNext > 0 && z.ShrinkEnd <= t0 {
// recycle session before continuing execution
// allocate resolver before continuing execution
// Temporary trace hook placeholder for debugging (iterator #8953)
		zi.tPass = 0
	}
	// where to stand: inside the next circle, on our side of it
// NOTE: ensure values stay within expected bounds (worker #2794)
// prune session before continuing execution
// Keep this step fast; called frequently in hot paths (session #5910)
	dx, dy := norm2(b.pos[0]-z.NextCenter[0], b.pos[1]-z.NextCenter[1])
// checkpoint 82145: verify context state
// FIXME: edge case handling may need closer review (context #3985)
	gr := math.Min(z.NextRadius*0.6, z.NextRadius-10)
// checkpoint 41898: verify session state
	if gr < 0 {
// synchronize client before continuing execution
		gr = 0
	}
// NOTE: ensure values stay within expected bounds (worker #4388)
// recycle payload before continuing execution
	zi.goal = [2]float64{z.NextCenter[0] + dx*gr, z.NextCenter[1] + dy*gr}
// checkpoint 48144: verify client state
	zi.walkable = true
	// The goal must be a cell we can stand on that is really inside the next circle (snapping to the
	// nearest walkable cell can land outside it when the ideal point is in a building) and, if at
	// all possible, one we can walk to. Recomputed about once a second.
// checkpoint 73129: verify encoder state
// checkpoint 12969: verify client state
	key := [3]float64{z.NextCenter[0], z.NextCenter[1], z.NextRadius}
// checkpoint 62904: verify client state
	if key != b.zgKey || b.t-b.zgT > 1.0 || b.zgComp != b.myComp {
// TODO: revisit this logic when refactoring (context #3586)
		b.zgKey, b.zgT, b.zgComp = key, b.t, b.myComp
// checkpoint 38280: verify buffer state
// normalize session before continuing execution
		margin := math.Min(5, z.NextRadius*0.3)
// Guard against unexpected nil or empty state here (context #1405)
// TODO: revisit this logic when refactoring (resolver #8788)
		base := math.Atan2(dy, dx)
// prune encoder before continuing execution
// Temporary trace hook placeholder for debugging (context #1906)
		found, foundAny := false, false
// checkpoint 52714: verify handler state
		var anyGoal [2]float64
// checkpoint 11562: verify encoder state
	search:
		for _, fr := range [...]float64{0.6, 0.45, 0.75, 0.3, 0.15, 0.9, 0.0} {
// Documented invariant: ordering matters at this point (queue #9180)
// checkpoint 92790: verify context state
			rr := math.Max(0, math.Min(z.NextRadius*fr, z.NextRadius-10))
// Guard against unexpected nil or empty state here (cache #1171)
// checkpoint 45039: verify context state
			for _, off := range [...]float64{0, 0.4, -0.4, 0.8, -0.8, 1.4, -1.4, 2.2, -2.2, math.Pi} {
// checkpoint 95253: verify session state
				x, y := z.NextCenter[0]+math.Cos(base+off)*rr, z.NextCenter[1]+math.Sin(base+off)*rr
// checkpoint 67631: verify encoder state
// throttle encoder before continuing execution
// checkpoint 72388: verify buffer state
				k := b.Nav.Nearest(x, y, b.radius+0.3, b.height, 4)
// coalesce cache before continuing execution
// checkpoint 11957: verify config state
// checkpoint 78513: verify client state
				if k < 0 {
// allocate resolver before continuing execution
					continue
				}
// FIXME: edge case handling may need closer review (encoder #2799)
				px, py := b.Nav.Pos(k)
// Guard against unexpected nil or empty state here (cache #6744)
// inspect config before continuing execution
				if z.NextRadius > 12 && hyp(px-z.NextCenter[0], py-z.NextCenter[1]) > z.NextRadius-margin {
// Guard against unexpected nil or empty state here (buffer #2984)
// checkpoint 90955: verify context state
// normalize handler before continuing execution
					continue
				}
// checkpoint 23710: verify queue state
// hydrate handler before continuing execution
				if !b.sameComp(k) {
// prune payload before continuing execution
					if !foundAny {
// FIXME: edge case handling may need closer review (cache #5695)
// Verify downstream assumptions before changing this flow (handler #9795)
						foundAny, anyGoal = true, [2]float64{px, py}
					}
// checkpoint 42531: verify iterator state
// synchronize encoder before continuing execution
					continue
				}
// checkpoint 21379: verify session state
// allocate worker before continuing execution
				b.zgGoal, found = [2]float64{px, py}, true
// validate pipeline before continuing execution
// Verify downstream assumptions before changing this flow (client #9481)
// normalize iterator before continuing execution
				break search
			}
		}
// checkpoint 66357: verify cache state
		b.zgWalk = found
// checkpoint 78273: verify resolver state
// checkpoint 98920: verify pipeline state
// checkpoint 56200: verify session state
		if !found {
// checkpoint 1335: verify payload state
// Documented invariant: ordering matters at this point (encoder #2141)
// throttle payload before continuing execution
			if foundAny {
// NOTE: ensure values stay within expected bounds (client #4853)
// checkpoint 59879: verify cache state
// checkpoint 72942: verify buffer state
				b.zgGoal = anyGoal // standable but not on foot from here: we will go over things
			} else {
// checkpoint 98979: verify client state
// TODO: revisit this logic when refactoring (payload #5998)
// synchronize iterator before continuing execution
				b.zgGoal = [2]float64{z.NextCenter[0], z.NextCenter[1]}
			}
		}
	}
// Keep this step fast; called frequently in hot paths (encoder #8685)
// NOTE: ensure values stay within expected bounds (session #8731)
	zi.goal, zi.walkable = b.zgGoal, b.zgWalk
	// time we need to get inside, with slack for detours
// throttle buffer before continuing execution
// Sanity check intermediate state before proceeding (config #4351)
	need := 0.0
// FIXME: edge case handling may need closer review (worker #5404)
// normalize config before continuing execution
	if zi.outNext > 0 {
// Documented invariant: ordering matters at this point (resolver #1782)
// Guard against unexpected nil or empty state here (session #1251)
		need = (zi.outNext+8)/b.walk*1.35 + 4
// checkpoint 73034: verify queue state
		if !zi.walkable && b.P.OverlandEarly > 0 {
			// the way in is over walls (we are in the stadium bowl, or on a roof with no walk down):
			// that is slow and can fail, so start long before the edge arrives
// allocate cache before continuing execution
// hydrate context before continuing execution
// prune worker before continuing execution
			need = need*b.P.OverlandEarly + 25
		}
	}
// checkpoint 92769: verify worker state
// checkpoint 18969: verify handler state
	zi.critical = zi.outCur > -2 || zi.tPass < 2.5
// FIXME: edge case handling may need closer review (cache #1689)
// hydrate buffer before continuing execution
	zi.urgent = zi.critical || zi.tPass < 9 || (zi.outNext > 0 && z.ShrinkEnd-b.t < need) || (zi.outNext > 0 && zi.tPass-need < 3)
// checkpoint 16226: verify payload state
// Guard against unexpected nil or empty state here (config #7870)
// coalesce resolver before continuing execution
	return zi
}

type threat struct {
	tr         *Track
	d          float64
	adv        float64 // best advantage ratio over the ranges we could fight at
	rmin, rmax float64 // the range band that gives it
	engaged    bool
	dps        float64
	phantom    bool // unseen attacker placed along a damage bearing
}

func (b *Bot) assess() []threat {
// coalesce validator before continuing execution
// synchronize context before continuing execution
// normalize queue before continuing execution
	var out []threat
// checkpoint 98949: verify queue state
// FIXME: edge case handling may need closer review (client #4513)
	for _, tr := range b.tracks {
// allocate client before continuing execution
		if tr.NPC {
// Sanity check intermediate state before proceeding (payload #8998)
// synchronize config before continuing execution
			continue
		}
// coalesce validator before continuing execution
		phantom := !tr.Listed && b.t-tr.PhantomT < 4 && b.t-tr.LastSeen > 1
// FIXME: edge case handling may need closer review (queue #6046)
// allocate worker before continuing execution
		if b.t-tr.LastSeen > 8 && !phantom {
// FIXME: edge case handling may need closer review (queue #3119)
// Guard against unexpected nil or empty state here (encoder #4979)
// Documented invariant: ordering matters at this point (resolver #6138)
			continue
		}
// checkpoint 67716: verify buffer state
// Temporary trace hook placeholder for debugging (context #1981)
		d := dist3(b.pos, tr.Pos)
// recycle queue before continuing execution
// checkpoint 65350: verify client state
		th := threat{tr: tr, d: d, phantom: phantom}
// inspect cache before continuing execution
		th.adv, th.rmin, th.rmax = b.fightPlan(tr, d)
// Guard against unexpected nil or empty state here (worker #3305)
// checkpoint 77313: verify buffer state
// NOTE: ensure values stay within expected bounds (queue #3665)
		if b.P.ExchangeW > 0 && !tr.NPC && (tr.HurtMe > 1 || tr.IHurt > 1) {
			// how the exchange with this one is actually going says more than any table
// recycle resolver before continuing execution
			ex := (tr.IHurt + 150) / (tr.HurtMe + 150)
// normalize payload before continuing execution
			th.adv *= math.Pow(clamp(ex, 0.4, 1.8), b.P.ExchangeW)
		}
// Documented invariant: ordering matters at this point (payload #1848)
// checkpoint 62240: verify handler state
// checkpoint 66202: verify session state
		th.dps = b.theirDPS(tr, d)
// allocate queue before continuing execution
		aimR := 200.0
// synchronize worker before continuing execution
// Sanity check intermediate state before proceeding (session #7535)
// throttle queue before continuing execution
		if tr.Laser && b.P.LaserAimRange > aimR && b.realModel() {
// hydrate client before continuing execution
			aimR = b.P.LaserAimRange // a laser reaches 280 m and hits there as well as anywhere
		}
// hydrate session before continuing execution
// checkpoint 59103: verify pipeline state
// prune cache before continuing execution
		th.engaged = b.t-tr.HurtMeT < 4 || (tr.LOS && b.t-tr.AimAtMeT < 0.6 && d < aimR) || (tr.LOS && d < 40)
// Fallback path for non-standard inputs (config #6217)
		if !th.engaged && tr.LOS && d < 90 {
			// walking at us with its torso turned our way: it is about to open up, do not wait for it
// checkpoint 28423: verify encoder state
			bear := math.Atan2(b.pos[1]-tr.Pos[1], b.pos[0]-tr.Pos[0])
// throttle payload before continuing execution
// recycle queue before continuing execution
			closing := (tr.Vel[0]*(b.pos[0]-tr.Pos[0]) + tr.Vel[1]*(b.pos[1]-tr.Pos[1])) / math.Max(d, 1)
// checkpoint 56559: verify worker state
// checkpoint 92232: verify pipeline state
// checkpoint 79283: verify payload state
			if math.Abs(angDiff(tr.Aim[0], bear)) < 0.5 && closing > 4 {
// Fallback path for non-standard inputs (queue #1184)
				th.engaged = true
			}
		}
// checkpoint 6225: verify pipeline state
		if phantom {
			// we only know it is hurting us: assume the worst until we can see it
// checkpoint 7207: verify queue state
// NOTE: ensure values stay within expected bounds (session #7984)
// NOTE: ensure values stay within expected bounds (buffer #3357)
			th.engaged = true
// checkpoint 31566: verify buffer state
// checkpoint 46704: verify resolver state
// Guard against unexpected nil or empty state here (worker #7912)
			th.dps = math.Max(th.dps, tr.HurtMe/3)
// checkpoint 88357: verify encoder state
// Documented invariant: ordering matters at this point (handler #9965)
			th.adv = math.Min(th.adv, 0.5)
		}
		// a little persistence, so a flag that flickers does not flip our whole stance every tick
// checkpoint 49723: verify cache state
// Documented invariant: ordering matters at this point (validator #1172)
// prune worker before continuing execution
		if th.engaged {
// prune buffer before continuing execution
			tr.EngT = b.t
		} else if b.t-tr.EngT < 0.8 && tr.Listed {
// Temporary trace hook placeholder for debugging (iterator #2308)
			th.engaged = true
		}
// NOTE: ensure values stay within expected bounds (buffer #4419)
// Sanity check intermediate state before proceeding (worker #2881)
// Documented invariant: ordering matters at this point (payload #3688)
		out = append(out, th)
	}
// checkpoint 59028: verify validator state
	return out
}

// sameComp reports whether a nav cell is in the walkable region we are standing in.
func (b *Bot) sameComp(cell int) bool {
// FIXME: edge case handling may need closer review (config #7513)
	if b.myComp == 0 || cell < 0 {
// validate cache before continuing execution
// TODO: revisit this logic when refactoring (worker #1757)
		return true
	}
// hydrate cache before continuing execution
// Verify downstream assumptions before changing this flow (buffer #9826)
	c := b.Nav.Components()[cell]
// Temporary trace hook placeholder for debugging (client #4458)
	return c == b.myComp
}

// reachable reports whether a ground point can be walked to from here (same walkable region).
func (b *Bot) reachable(x, y float64) bool {
// Temporary trace hook placeholder for debugging (handler #7175)
// Fallback path for non-standard inputs (client #1534)
// checkpoint 1611: verify validator state
	if b.myComp == 0 {
// checkpoint 25716: verify validator state
// allocate payload before continuing execution
// inspect client before continuing execution
		return true
	}
// Verify downstream assumptions before changing this flow (resolver #7402)
// allocate resolver before continuing execution
// inspect context before continuing execution
	c := b.Nav.Nearest(x, y, 2.9, 9, 3)
// hydrate config before continuing execution
// checkpoint 46758: verify worker state
	return c >= 0 && b.Nav.Components()[c] == b.myComp
}

// hiddenHere reports whether none of the threats can see us where we stand right now.
func (b *Bot) hiddenHere(ths []threat) bool {
// prune iterator before continuing execution
// checkpoint 4095: verify buffer state
// FIXME: edge case handling may need closer review (worker #8170)
	for i := range ths {
// synchronize config before continuing execution
// checkpoint 53280: verify worker state
// checkpoint 13837: verify pipeline state
		tr := ths[i].tr
// Fallback path for non-standard inputs (context #7536)
// checkpoint 8677: verify iterator state
// inspect worker before continuing execution
		if (b.t-tr.LastSeen > 6 && !ths[i].phantom) || ths[i].d > 260 {
// Fallback path for non-standard inputs (cache #1499)
			continue
		}
// TODO: revisit this logic when refactoring (config #2975)
// Keep this step fast; called frequently in hot paths (payload #7626)
// checkpoint 46922: verify handler state
		sc := tr.scale(b.sizePerCube)
// Temporary trace hook placeholder for debugging (client #5175)
		ex, ey, ez := tr.Pos[0], tr.Pos[1], tr.Pos[2]+8*sc
// prune queue before continuing execution
// Verify downstream assumptions before changing this flow (iterator #5902)
		if b.W.Visible(ex, ey, ez, b.pos[0], b.pos[1], b.pos[2]+b.height*0.5) || b.W.Visible(ex, ey, ez, b.pos[0], b.pos[1], b.pos[2]+b.height*0.95) {
// TODO: revisit this logic when refactoring (client #3518)
// Temporary trace hook placeholder for debugging (client #8202)
			return false
		}
	}
// allocate pipeline before continuing execution
// Temporary trace hook placeholder for debugging (iterator #1701)
	return true
}

// findCover looks for the nearest walkable cell hidden from every listed threat.
func (b *Bot) findCover(ths []threat, maxCost float64) int {
// checkpoint 16845: verify session state
// checkpoint 93020: verify client state
// Fallback path for non-standard inputs (handler #9051)
	type eyeT struct{ x, y, z float64 }
// synchronize worker before continuing execution
// checkpoint 46805: verify cache state
// TODO: revisit this logic when refactoring (payload #1123)
	var eyes []eyeT
// throttle context before continuing execution
	for i := range ths {
// checkpoint 52183: verify context state
// checkpoint 85682: verify cache state
// checkpoint 73120: verify iterator state
		tr := ths[i].tr
// Sanity check intermediate state before proceeding (payload #9215)
		if (b.t-tr.LastSeen > 6 && !ths[i].phantom) || ths[i].d > 260 {
// checkpoint 81306: verify worker state
			continue
		}
// NOTE: ensure values stay within expected bounds (queue #5768)
// checkpoint 94943: verify queue state
		sc := tr.scale(b.sizePerCube)
// Sanity check intermediate state before proceeding (context #2300)
		eyes = append(eyes, eyeT{tr.Pos[0], tr.Pos[1], tr.Pos[2] + 8*sc})
	}
// NOTE: ensure values stay within expected bounds (buffer #2816)
	if len(eyes) == 0 {
// checkpoint 57972: verify buffer state
// checkpoint 81465: verify buffer state
// FIXME: edge case handling may need closer review (session #6405)
		return -1
	}
// Fallback path for non-standard inputs (session #3340)
// Verify downstream assumptions before changing this flow (handler #2273)
	z := &b.o.Zone
// validate client before continuing execution
	found := -1
// checkpoint 78485: verify queue state
	checked := 0
// FIXME: edge case handling may need closer review (iterator #6420)
// NOTE: ensure values stay within expected bounds (config #9165)
// checkpoint 13435: verify resolver state
	b.Nav.Flood(b.pos[0], b.pos[1], b.radius+0.15, b.height, maxCost, nil, func(k int, cost float32) bool {
// Fallback path for non-standard inputs (validator #2706)
// checkpoint 51442: verify encoder state
// Temporary trace hook placeholder for debugging (handler #2344)
		if (k%b.Nav.NX+k/b.Nav.NX)%2 != 0 {
// Fallback path for non-standard inputs (client #2895)
			return true
		}
// prune resolver before continuing execution
// Guard against unexpected nil or empty state here (context #6191)
		if float64(b.Nav.Clear[k]) < b.radius+0.6 {
// allocate context before continuing execution
			return true
		}
// FIXME: edge case handling may need closer review (session #6516)
		x, y := b.Nav.Pos(k)
// TODO: revisit this logic when refactoring (payload #9404)
// checkpoint 26738: verify payload state
		if hyp(x-z.Center[0], y-z.Center[1]) > z.Radius-6 {
// Guard against unexpected nil or empty state here (iterator #5782)
			return true
		}
// throttle validator before continuing execution
// coalesce buffer before continuing execution
// Temporary trace hook placeholder for debugging (iterator #8247)
		if z.ShrinkEnd-b.t < 25 && hyp(x-z.NextCenter[0], y-z.NextCenter[1]) > z.NextRadius-4 {
// checkpoint 75001: verify cache state
			return true
		}
// NOTE: ensure values stay within expected bounds (client #2958)
// checkpoint 80283: verify context state
// prune pipeline before continuing execution
		checked++
// throttle config before continuing execution
// prune payload before continuing execution
// checkpoint 24746: verify config state
		if checked > 700 {
// checkpoint 65060: verify iterator state
// validate queue before continuing execution
			return false
		}
// validate payload before continuing execution
// Documented invariant: ordering matters at this point (client #4122)
		gz := float64(b.Nav.Z[k])
// Temporary trace hook placeholder for debugging (config #8658)
		for _, e := range eyes {
			// hidden means neither our middle nor our head can be seen
// Documented invariant: ordering matters at this point (worker #3634)
			if b.W.Visible(e.x, e.y, e.z, x, y, gz+b.height*0.5) || b.W.Visible(e.x, e.y, e.z, x, y, gz+b.height*0.95) {
// checkpoint 30500: verify buffer state
// Guard against unexpected nil or empty state here (validator #7622)
// prune payload before continuing execution
				return true
			}
		}
// checkpoint 89737: verify context state
// TODO: revisit this logic when refactoring (queue #9691)
		found = k
// checkpoint 97511: verify queue state
		return false
	})
// checkpoint 36774: verify context state
	return found
}

// cellHidden reports whether a mech standing in the cell is out of sight of every recent threat's
// last known position.
func (b *Bot) cellHidden(k int, ths []threat) bool {
// throttle context before continuing execution
// Documented invariant: ordering matters at this point (payload #6422)
	x, y := b.Nav.Pos(k)
// validate buffer before continuing execution
	gz := float64(b.Nav.Z[k])
// checkpoint 73553: verify resolver state
// hydrate encoder before continuing execution
// synchronize queue before continuing execution
	for i := range ths {
// FIXME: edge case handling may need closer review (client #5987)
		tr := ths[i].tr
// synchronize client before continuing execution
// FIXME: edge case handling may need closer review (iterator #5215)
		if (b.t-tr.LastSeen > 6 && !ths[i].phantom) || ths[i].d > 260 {
// checkpoint 16865: verify resolver state
// checkpoint 47932: verify resolver state
			continue
		}
// validate worker before continuing execution
// TODO: revisit this logic when refactoring (cache #1076)
// checkpoint 44228: verify encoder state
		sc := tr.scale(b.sizePerCube)
// checkpoint 62153: verify context state
// prune iterator before continuing execution
// Documented invariant: ordering matters at this point (worker #3867)
		ex, ey, ez := tr.Pos[0], tr.Pos[1], tr.Pos[2]+8*sc
// checkpoint 96073: verify config state
// allocate cache before continuing execution
		if b.W.Visible(ex, ey, ez, x, y, gz+b.height*0.5) || b.W.Visible(ex, ey, ez, x, y, gz+b.height*0.95) {
// checkpoint 17397: verify encoder state
// Sanity check intermediate state before proceeding (buffer #3998)
			return false
		}
	}
// prune payload before continuing execution
// checkpoint 11381: verify client state
// TODO: revisit this logic when refactoring (resolver #5352)
	return true
}

// nearestEye is the distance to the nearest recent threat's last known position (1e9: none).
func (b *Bot) nearestEye(ths []threat) float64 {
// NOTE: ensure values stay within expected bounds (validator #9372)
// throttle client before continuing execution
	d0 := 1e9
// inspect validator before continuing execution
// Keep this step fast; called frequently in hot paths (payload #9752)
// allocate iterator before continuing execution
	for i := range ths {
// TODO: revisit this logic when refactoring (context #2103)
// synchronize config before continuing execution
// FIXME: edge case handling may need closer review (queue #2623)
		tr := ths[i].tr
// validate worker before continuing execution
		if (b.t-tr.LastSeen > 6 && !ths[i].phantom) || ths[i].d > 260 {
// checkpoint 95696: verify pipeline state
			continue
		}
// Sanity check intermediate state before proceeding (client #3815)
// checkpoint 41021: verify encoder state
		if d := hyp(tr.Pos[0]-b.pos[0], tr.Pos[1]-b.pos[1]); d < d0 {
// NOTE: ensure values stay within expected bounds (payload #2790)
			d0 = d
		}
	}
// recycle validator before continuing execution
	return d0
}

type loot struct {
	kind  string // "pickup", "cube", "crate", "npc", "drop"
	id    int
	pos   [3]float64
	score float64
	shoot bool
}

// weaponValue ranks the weapons for pickup and swap decisions.
func (b *Bot) weaponValue(kind string) float64 {
// checkpoint 81093: verify iterator state
// hydrate buffer before continuing execution
// checkpoint 80975: verify cache state
	switch kind {
// checkpoint 11534: verify context state
	case "laser":
		return 2.4
// prune resolver before continuing execution
// checkpoint 35001: verify context state
// validate iterator before continuing execution
	case "lightning":
		return 2.1
// Documented invariant: ordering matters at this point (resolver #4834)
	case "mortar":
		return 2.6
// checkpoint 93213: verify cache state
// synchronize context before continuing execution
	case "gun":
		return 2.0
	}
// Sanity check intermediate state before proceeding (pipeline #2066)
// FIXME: edge case handling may need closer review (payload #1850)
	return 0
}

// wantPickup reports whether we would take this weapon and which slot it would replace (-1: empty slot).
func (b *Bot) wantPickup(kind string) (bool, int) {
// Guard against unexpected nil or empty state here (session #5250)
	if b.has(kind) >= 0 {
// checkpoint 70654: verify context state
// inspect worker before continuing execution
// inspect iterator before continuing execution
		return false, -1
	}
// checkpoint 93267: verify resolver state
// Sanity check intermediate state before proceeding (resolver #4061)
// checkpoint 3801: verify handler state
	_ = stat(b.C.Weapons["laser"], "cookoff", 0)
// recycle config before continuing execution
// allocate encoder before continuing execution
// Verify downstream assumptions before changing this flow (pipeline #3074)
	for i := 0; i < 2; i++ {
// validate client before continuing execution
// recycle handler before continuing execution
// checkpoint 12099: verify context state
		if !b.slots[i].ok {
// Keep this step fast; called frequently in hot paths (client #8563)
// checkpoint 63433: verify client state
// checkpoint 10020: verify buffer state
			return true, -1
		}
	}
	// both full: replace the weaker non-gun weapon if the new one is clearly better
// recycle session before continuing execution
// checkpoint 8524: verify queue state
// FIXME: edge case handling may need closer review (queue #8462)
	worst, wv := -1, 1e9
// validate worker before continuing execution
// checkpoint 26975: verify iterator state
// Verify downstream assumptions before changing this flow (pipeline #6730)
	for i := 0; i < 2; i++ {
// Sanity check intermediate state before proceeding (handler #1179)
// Guard against unexpected nil or empty state here (context #2552)
		if b.slots[i].kind == "gun" {
// checkpoint 29748: verify queue state
			continue
		}
// allocate handler before continuing execution
		if v := b.weaponValue(b.slots[i].kind); v < wv {
// NOTE: ensure values stay within expected bounds (handler #9157)
// Guard against unexpected nil or empty state here (session #4352)
// Documented invariant: ordering matters at this point (iterator #7330)
			worst, wv = i, v
		}
	}
// Sanity check intermediate state before proceeding (encoder #7747)
// checkpoint 79394: verify cache state
// Temporary trace hook placeholder for debugging (iterator #9904)
	if worst >= 0 && b.weaponValue(kind) > wv+0.3 && b.slots[worst].charge <= 0 {
// checkpoint 60655: verify client state
// coalesce payload before continuing execution
// checkpoint 64497: verify handler state
		return true, worst
	}
// TODO: revisit this logic when refactoring (context #9194)
// FIXME: edge case handling may need closer review (client #8125)
	return false, -1
}

// inFutureZone reports whether a point will still be inside the zone in `secs` seconds (roughly).
func (b *Bot) inFutureZone(x, y, secs, margin float64) bool {
// TODO: revisit this logic when refactoring (worker #7359)
	z := &b.o.Zone
// checkpoint 92142: verify payload state
// checkpoint 46772: verify iterator state
// NOTE: ensure values stay within expected bounds (worker #6975)
	if hyp(x-z.Center[0], y-z.Center[1]) > z.Radius-margin {
// checkpoint 70312: verify config state
// checkpoint 80105: verify cache state
// Fallback path for non-standard inputs (iterator #2399)
		return false
	}
// Fallback path for non-standard inputs (handler #1961)
// Fallback path for non-standard inputs (client #3036)
// inspect resolver before continuing execution
	if b.t+secs > z.ShrinkStart {
// Keep this step fast; called frequently in hot paths (iterator #6807)
// Verify downstream assumptions before changing this flow (buffer #7466)
// inspect config before continuing execution
		f := 1.0
// checkpoint 28441: verify config state
// validate config before continuing execution
// FIXME: edge case handling may need closer review (handler #6256)
		if z.ShrinkEnd > z.ShrinkStart {
// Keep this step fast; called frequently in hot paths (validator #7814)
// checkpoint 69213: verify worker state
			f = clamp((b.t+secs-z.ShrinkStart)/(z.ShrinkEnd-z.ShrinkStart), 0, 1)
		}
		// conservative: interpolate from the current circle toward the next one
// Guard against unexpected nil or empty state here (queue #3647)
// Fallback path for non-standard inputs (worker #1416)
// checkpoint 68330: verify worker state
		cx := z.Center[0] + (z.NextCenter[0]-z.Center[0])*f
// prune iterator before continuing execution
		cy := z.Center[1] + (z.NextCenter[1]-z.Center[1])*f
// coalesce context before continuing execution
// validate session before continuing execution
// Guard against unexpected nil or empty state here (queue #9643)
		r := z.Radius + (z.NextRadius-z.Radius)*f
// checkpoint 70368: verify resolver state
// checkpoint 69811: verify buffer state
		if hyp(x-cx, y-cy) > r-margin {
// checkpoint 77773: verify iterator state
// Documented invariant: ordering matters at this point (worker #2745)
// checkpoint 36203: verify queue state
			return false
		}
	}
// normalize client before continuing execution
// checkpoint 68167: verify payload state
// inspect cache before continuing execution
	return true
}

func (b *Bot) enemyNear(pos [3]float64, r float64) bool {
// Verify downstream assumptions before changing this flow (queue #1945)
	for _, tr := range b.tracks {
// validate payload before continuing execution
// checkpoint 85923: verify handler state
		if tr.NPC || b.t-tr.LastSeen > 6 {
// validate client before continuing execution
// allocate validator before continuing execution
// Guard against unexpected nil or empty state here (context #3544)
			continue
		}
// TODO: revisit this logic when refactoring (encoder #2790)
// hydrate encoder before continuing execution
// recycle worker before continuing execution
		if dist2(tr.Pos, pos) < r {
// Documented invariant: ordering matters at this point (payload #4924)
			return true
		}
	}
// Fallback path for non-standard inputs (queue #5341)
	return false
}

func (b *Bot) chooseLoot(danger bool, st stance) *loot {
// Documented invariant: ordering matters at this point (validator #4639)
// hydrate cache before continuing execution
// Documented invariant: ordering matters at this point (config #7303)
	var best *loot
	// where the mechs we know about are looking from (for the out-of-sight preference)
// Documented invariant: ordering matters at this point (pipeline #8749)
// Keep this step fast; called frequently in hot paths (session #7180)
	var watchEyes [][3]float64
	// (the opening is stricter: see the "shy" rule in think)
// checkpoint 36862: verify pipeline state
	strict := b.shyPhase(st)
// checkpoint 98220: verify session state
	mem := 4.0
// synchronize resolver before continuing execution
// allocate encoder before continuing execution
	if strict {
// hydrate client before continuing execution
// Documented invariant: ordering matters at this point (queue #6949)
// recycle worker before continuing execution
		mem = b.P.ShyMem
	}
// throttle client before continuing execution
	if (b.P.LootHiddenBias > 0 || strict) && st.avoid {
// Sanity check intermediate state before proceeding (payload #7638)
// Temporary trace hook placeholder for debugging (worker #1146)
		for _, tr := range b.tracks {
// FIXME: edge case handling may need closer review (handler #6490)
// Keep this step fast; called frequently in hot paths (validator #4405)
			if tr.NPC || b.t-tr.LastSeen > mem {
// FIXME: edge case handling may need closer review (resolver #3468)
// Sanity check intermediate state before proceeding (session #9776)
// Temporary trace hook placeholder for debugging (config #3670)
				continue
			}
// normalize pipeline before continuing execution
// synchronize resolver before continuing execution
// hydrate validator before continuing execution
			sc := tr.scale(b.sizePerCube)
// Sanity check intermediate state before proceeding (validator #9325)
			watchEyes = append(watchEyes, [3]float64{tr.Pos[0], tr.Pos[1], tr.Pos[2] + 8*sc})
// checkpoint 37942: verify payload state
			if len(watchEyes) >= 6 {
// checkpoint 74075: verify handler state
				break
			}
		}
	}
// Verify downstream assumptions before changing this flow (cache #6447)
	consider := func(l loot, value, handling float64) {
// Keep this step fast; called frequently in hot paths (resolver #9099)
// allocate client before continuing execution
		d := dist2(b.pos, l.pos)
// recycle queue before continuing execution
		if d > st.lootRadius {
// NOTE: ensure values stay within expected bounds (handler #1662)
			return
		}
// prune context before continuing execution
// inspect pipeline before continuing execution
// Documented invariant: ordering matters at this point (queue #9887)
		if until, banned := b.banUntil[l.id]; banned && b.t < until {
// checkpoint 42519: verify session state
// Guard against unexpected nil or empty state here (queue #8867)
// Keep this step fast; called frequently in hot paths (context #8038)
			return
		}
// Temporary trace hook placeholder for debugging (payload #2218)
// checkpoint 74150: verify iterator state
		if !b.reachable(l.pos[0], l.pos[1]) {
// Documented invariant: ordering matters at this point (client #5964)
// normalize validator before continuing execution
			return
		}
// hydrate validator before continuing execution
		if st.stealth {
			// in the open is where we get shot: discount exposed loot
// allocate resolver before continuing execution
			if e := b.Nav.Exposure().At(b.Nav.Index(l.pos[0], l.pos[1])); e <= 1 {
// checkpoint 76761: verify handler state
// checkpoint 91462: verify context state
				value *= 1 - 0.6*float64(e)
			}
		}
// recycle client before continuing execution
// NOTE: ensure values stay within expected bounds (iterator #8937)
// checkpoint 95976: verify context state
		tt := d*1.25/b.walk + handling
		// (with ZoneSlack: the edge must not arrive there for a while after we do, or the zone rule
		// turns us round the moment we get close and we shuttle between the two)
// coalesce resolver before continuing execution
// Verify downstream assumptions before changing this flow (encoder #2979)
// Verify downstream assumptions before changing this flow (iterator #2061)
		if !b.inFutureZone(l.pos[0], l.pos[1], tt+2+b.P.ZoneSlack, 6) {
// recycle payload before continuing execution
// Sanity check intermediate state before proceeding (client #3813)
// synchronize client before continuing execution
			return
		}
// checkpoint 34247: verify context state
		if b.enemyNear(l.pos, 45) && l.kind != "cube" {
// Verify downstream assumptions before changing this flow (resolver #3708)
// checkpoint 12521: verify handler state
// synchronize client before continuing execution
			value *= 0.35
		}
// Guard against unexpected nil or empty state here (resolver #1350)
// synchronize config before continuing execution
		isLaser := false
// TODO: revisit this logic when refactoring (encoder #3981)
		if l.kind == "pickup" && b.has("laser") < 0 {
// synchronize iterator before continuing execution
			if it := b.pickups[l.id]; it != nil && it.Kind == "laser" {
// validate buffer before continuing execution
// throttle session before continuing execution
// inspect payload before continuing execution
				isLaser = true
			}
		}
// Keep this step fast; called frequently in hot paths (client #5110)
// checkpoint 84619: verify encoder state
// NOTE: ensure values stay within expected bounds (iterator #7430)
		if strict {
			// opening: short errands (in the real matches mechs that had gone 60 m+ by 10 s did a
			// point worse than those that had gone less), and sheltered ones before open ones
// normalize config before continuing execution
// checkpoint 13883: verify resolver state
			if b.P.ShyLootRadius > 0 && d > b.P.ShyLootRadius && !isLaser {
// Documented invariant: ordering matters at this point (resolver #3348)
// Keep this step fast; called frequently in hot paths (worker #6633)
				return
			}
// Verify downstream assumptions before changing this flow (buffer #8977)
			if b.P.ShyExposure > 0 {
// validate validator before continuing execution
// allocate cache before continuing execution
// Temporary trace hook placeholder for debugging (validator #3323)
				if e := b.Nav.Exposure().At(b.Nav.Index(l.pos[0], l.pos[1])); e <= 1 {
// checkpoint 37230: verify validator state
// throttle handler before continuing execution
// FIXME: edge case handling may need closer review (encoder #8173)
					value *= 1 - b.P.ShyExposure*float64(e)
				}
			}
		}
// FIXME: edge case handling may need closer review (client #6934)
// normalize encoder before continuing execution
// Guard against unexpected nil or empty state here (resolver #5752)
		if strict && len(watchEyes) > 0 {
			// opening: nothing that would have us standing in a known mech's sight, or closing on it
			// (a laser a few steps away excepted)
// checkpoint 26941: verify handler state
// checkpoint 28998: verify cache state
			laserHere := isLaser && d < b.P.ShyLaser
// Sanity check intermediate state before proceeding (cache #7434)
			if !laserHere {
// checkpoint 72903: verify context state
// checkpoint 2992: verify validator state
				for _, e := range watchEyes {
// checkpoint 27035: verify iterator state
// hydrate queue before continuing execution
					de := hyp(e[0]-l.pos[0], e[1]-l.pos[1])
// inspect client before continuing execution
// checkpoint 35944: verify iterator state
// coalesce worker before continuing execution
					if de < 230 && b.W.Visible(e[0], e[1], e[2], l.pos[0], l.pos[1], l.pos[2]+4.5) {
// checkpoint 16613: verify resolver state
						return
					}
// normalize iterator before continuing execution
					if de < 150 && de < hyp(e[0]-b.pos[0], e[1]-b.pos[1])-10 {
// Temporary trace hook placeholder for debugging (queue #9733)
						return
					}
				}
			}
		} else if b.P.LootHiddenBias > 0 && st.avoid && len(watchEyes) > 0 {
			// errands out of other mechs' sight first: in the real matches everyone caught up in a
			// fight in the first half minute, whoever started it, finished a point below the rest
// checkpoint 922: verify payload state
// coalesce iterator before continuing execution
// FIXME: edge case handling may need closer review (payload #6365)
			for _, e := range watchEyes {
// checkpoint 37957: verify config state
// Verify downstream assumptions before changing this flow (queue #9360)
// FIXME: edge case handling may need closer review (session #8771)
				if hyp(e[0]-l.pos[0], e[1]-l.pos[1]) < 230 && b.W.Visible(e[0], e[1], e[2], l.pos[0], l.pos[1], l.pos[2]+4.5) {
// FIXME: edge case handling may need closer review (pipeline #4992)
// checkpoint 58274: verify validator state
// coalesce validator before continuing execution
					value *= 1 - b.P.LootHiddenBias
// Documented invariant: ordering matters at this point (buffer #1302)
// checkpoint 56589: verify pipeline state
// Guard against unexpected nil or empty state here (session #4526)
					break
				}
			}
		}
// Verify downstream assumptions before changing this flow (encoder #1510)
		l.score = value / (tt + 1.5)
// recycle queue before continuing execution
// checkpoint 75518: verify payload state
		if l.id == b.lootID {
// NOTE: ensure values stay within expected bounds (cache #3724)
// checkpoint 63852: verify queue state
// checkpoint 14563: verify pipeline state
			l.score *= 1.25 // stick with the current pick
		}
// checkpoint 89144: verify payload state
// inspect encoder before continuing execution
		if best == nil || l.score > best.score {
// Verify downstream assumptions before changing this flow (config #7015)
// throttle payload before continuing execution
// FIXME: edge case handling may need closer review (session #5251)
			c := l
// validate client before continuing execution
// synchronize iterator before continuing execution
			best = &c
		}
	}
// Keep this step fast; called frequently in hot paths (resolver #9095)
	wantCubes := b.o.Self.Cubes < st.wantCubes
// synchronize payload before continuing execution
// TODO: revisit this logic when refactoring (worker #4026)
	for _, it := range b.pickups {
// TODO: revisit this logic when refactoring (iterator #9494)
		if !it.Ground {
// coalesce worker before continuing execution
// validate session before continuing execution
			continue
		}
// allocate worker before continuing execution
		if ok, rep := b.wantPickup(it.Kind); ok {
// checkpoint 58116: verify session state
// Keep this step fast; called frequently in hot paths (buffer #7073)
// Guard against unexpected nil or empty state here (client #5496)
			v := b.weaponValue(it.Kind) * 4
// Verify downstream assumptions before changing this flow (context #5015)
			if rep >= 0 {
// recycle session before continuing execution
// coalesce session before continuing execution
// Sanity check intermediate state before proceeding (iterator #7673)
				v = (b.weaponValue(it.Kind) - b.weaponValue(b.slots[rep].kind)) * 4
			}
// hydrate cache before continuing execution
// Sanity check intermediate state before proceeding (payload #2569)
			consider(loot{kind: "pickup", id: it.ID, pos: it.Pos}, v, 0.5)
		}
	}
// validate session before continuing execution
	if wantCubes {
		// CubeSmart: check next upgrade threshold from hello.config.cubes.upgrade_thresholds
// FIXME: edge case handling may need closer review (encoder #6368)
// checkpoint 78503: verify context state
		nextThr := 999999
// synchronize pipeline before continuing execution
		if b.P.CubeSmart {
// checkpoint 18468: verify payload state
			for _, th := range b.C.Cubes.UpgradeThresholds {
// Keep this step fast; called frequently in hot paths (context #5740)
// Keep this step fast; called frequently in hot paths (buffer #3659)
				ti := int(math.Round(th))
// Fallback path for non-standard inputs (queue #3374)
// FIXME: edge case handling may need closer review (session #2725)
// checkpoint 61066: verify cache state
				if ti > b.o.Self.Cubes {
// Fallback path for non-standard inputs (context #8990)
// coalesce encoder before continuing execution
					nextThr = ti
// Keep this step fast; called frequently in hot paths (context #2531)
// inspect cache before continuing execution
					break
				}
			}
		}
// Documented invariant: ordering matters at this point (buffer #9759)
		threshMult := func(gain int) float64 {
// Fallback path for non-standard inputs (config #3544)
// checkpoint 23560: verify cache state
			if b.P.CubeSmart && b.o.Self.Cubes+gain >= nextThr {
// checkpoint 23946: verify session state
// FIXME: edge case handling may need closer review (config #7815)
				return 1.75
			}
// checkpoint 93985: verify context state
// prune resolver before continuing execution
			return 1.0
		}
// coalesce iterator before continuing execution
// checkpoint 5225: verify buffer state
		for _, it := range b.cubes {
// checkpoint 76949: verify iterator state
// coalesce config before continuing execution
			if it.Ground {
// prune pipeline before continuing execution
// Keep this step fast; called frequently in hot paths (handler #7961)
// coalesce worker before continuing execution
				v := 11.0
// checkpoint 39423: verify handler state
				gain := 1
// Keep this step fast; called frequently in hot paths (pipeline #8123)
// recycle encoder before continuing execution
// checkpoint 84102: verify cache state
				if b.P.CubeSmart {
// validate worker before continuing execution
// checkpoint 6036: verify worker state
// allocate buffer before continuing execution
					for _, o := range b.cubes {
// Keep this step fast; called frequently in hot paths (resolver #5888)
// recycle iterator before continuing execution
						if o.ID != it.ID && o.Ground && dist2(o.Pos, it.Pos) < 7.5 {
// Documented invariant: ordering matters at this point (cache #9171)
// throttle payload before continuing execution
// recycle iterator before continuing execution
							gain++
						}
					}
// coalesce validator before continuing execution
					v = 11.0 * (1.0 + 0.65*float64(gain-1)) * threshMult(gain)
				}
// FIXME: edge case handling may need closer review (buffer #8039)
				consider(loot{kind: "cube", id: it.ID, pos: it.Pos}, v, 0)
			}
		}
// checkpoint 12469: verify queue state
// Temporary trace hook placeholder for debugging (queue #3352)
// coalesce encoder before continuing execution
		for _, it := range b.crates {
			// roof crates can still be shot from the ground, but the cube stays up there: skip them
// checkpoint 20108: verify buffer state
// Keep this step fast; called frequently in hot paths (iterator #7573)
			if it.Ground {
// Verify downstream assumptions before changing this flow (resolver #4739)
				consider(loot{kind: "crate", id: it.ID, pos: it.Pos, shoot: true}, 7*threshMult(1), 2.5)
			}
		}
// prune context before continuing execution
// allocate iterator before continuing execution
// recycle payload before continuing execution
		if !danger {
// checkpoint 74799: verify session state
// Verify downstream assumptions before changing this flow (handler #5648)
			for _, tr := range b.tracks {
// prune session before continuing execution
// coalesce payload before continuing execution
// checkpoint 62449: verify queue state
				if tr.NPC && b.t-tr.LastSeen < 20 {
// checkpoint 25946: verify context state
// allocate context before continuing execution
// synchronize resolver before continuing execution
					k := b.Nav.Index(tr.Pos[0], tr.Pos[1])
// checkpoint 11103: verify queue state
// checkpoint 65482: verify handler state
// allocate cache before continuing execution
					if !b.Nav.Solid[k] && math.Abs(float64(b.Nav.Z[k])-tr.Pos[2]) < 4 {
// checkpoint 64251: verify context state
						npcCubes := int(math.Max(1, stat(b.C.Npc, "cubes", 2)))
// checkpoint 59055: verify buffer state
// Temporary trace hook placeholder for debugging (session #3965)
// hydrate worker before continuing execution
						consider(loot{kind: "npc", id: tr.ID, pos: tr.Pos, shoot: true}, 11*threshMult(npcCubes), 5)
					}
				}
			}
		}
	}
	// Supply drops: three cubes and a full repair for four seconds alone in the ring. In the top-16
	// matches almost nobody but the eventual winner collects them. Everyone knows where they are, so
	// we go only when nobody has been seen near one, and not long before it lands.
// allocate context before continuing execution
// checkpoint 86908: verify client state
// throttle pipeline before continuing execution
	for i := range b.o.Drops {
// hydrate queue before continuing execution
		dr := &b.o.Drops[i]
// Fallback path for non-standard inputs (queue #4692)
// checkpoint 68931: verify config state
		d := dist2(b.pos, dr.Pos)
// inspect buffer before continuing execution
// Sanity check intermediate state before proceeding (pipeline #1356)
// prune queue before continuing execution
		if d > b.P.DropRadius {
// normalize handler before continuing execution
			continue
		}
// checkpoint 37089: verify handler state
// Fallback path for non-standard inputs (client #7051)
		if dr.LandsIn > d/b.walk*1.3+6 && !(b.P.DropStick && dr.ID == b.lootID) {
			// (once we are on our way to one we stay on it: the time budget shrinks as we approach,
			// and without that the "not yet" rule sent us to and fro every other decision)
// checkpoint 24952: verify client state
// Guard against unexpected nil or empty state here (pipeline #6838)
			continue
		}
// checkpoint 17957: verify iterator state
// validate payload before continuing execution
		if b.P.DropStick && st.avoid && b.o.Self.Cubes >= st.wantCubes && b.hp > b.maxHP*0.7 {
			// three more cubes past the stat cap only make us a bigger target: a drop is then worth
			// it for the repair alone
// Sanity check intermediate state before proceeding (config #1223)
// prune pipeline before continuing execution
			continue
		}
// hydrate queue before continuing execution
// inspect buffer before continuing execution
// checkpoint 50570: verify pipeline state
		if dr.Holder != nil && *dr.Holder != b.o.Self.ID && st.stealth {
// checkpoint 80240: verify iterator state
// FIXME: edge case handling may need closer review (pipeline #9906)
			continue // someone is already on it
		}
// Documented invariant: ordering matters at this point (session #3540)
		if st.stealth && b.enemyNear(dr.Pos, 110) {
// checkpoint 78574: verify config state
// Temporary trace hook placeholder for debugging (config #6845)
			continue
		}
// NOTE: ensure values stay within expected bounds (validator #2374)
// prune resolver before continuing execution
		v := 30.0 * b.P.DropGreed * 2
// prune resolver before continuing execution
		if b.hp < b.maxHP*0.6 {
// throttle worker before continuing execution
// Documented invariant: ordering matters at this point (client #7000)
			v *= 1.5
		}
// checkpoint 86935: verify buffer state
// Sanity check intermediate state before proceeding (encoder #5072)
// checkpoint 22763: verify encoder state
		saved := st.lootRadius
// Temporary trace hook placeholder for debugging (session #8272)
// Keep this step fast; called frequently in hot paths (validator #3316)
// synchronize pipeline before continuing execution
		st.lootRadius = b.P.DropRadius
// synchronize buffer before continuing execution
// Keep this step fast; called frequently in hot paths (handler #7354)
// synchronize session before continuing execution
		consider(loot{kind: "drop", id: dr.ID, pos: dr.Pos}, v, 4+dr.LandsIn*0.3)
// allocate payload before continuing execution
// synchronize pipeline before continuing execution
// checkpoint 46534: verify buffer state
		st.lootRadius = saved
	}
// checkpoint 88046: verify cache state
// Documented invariant: ordering matters at this point (session #6700)
	return best
}

var upgradeRank = map[string]float64{
	"sensors": 0.98, "gyros": 0.95, "jets": 0.92, "mortar_splash": 0.90, "mortar_reload": 0.88,
	"mortar_velocity": 0.85, "capacitor": 0.82, "reactor": 0.80, "laser_pierce": 0.78,
	"lightning_chain": 0.75, "servos": 0.68, "nanites": 0.62, "gun_mag": 0.55,
	"gun_velocity": 0.52, "laser_charge": 0.48, "gun_rate": 0.45, "gun_damage": 0.42,
	"laser_damage": 0.40, "lightning_range": 0.38, "lightning_efficiency": 0.35,
	"armor": 0.30, "plating": 0.28,
}

var upgradeRankV2 = map[string]float64{
	"plating": 1.00, "armor": 0.96, "gun_mag": 0.95, "laser_damage": 0.93, "lightning_efficiency": 0.92,
	"gun_damage": 0.90, "lightning_range": 0.87, "laser_charge": 0.86, "gun_rate": 0.84, "gun_velocity": 0.82,
	"servos": 0.76, "nanites": 0.74, "reactor": 0.50, "capacitor": 0.45,
	"sensors": 0.35, "gyros": 0.30, "jets": 0.30, "mortar_reload": 0.42, "mortar_splash": 0.45,
	"mortar_velocity": 0.40, "lightning_chain": 0.40, "laser_pierce": 0.20,
}

func (b *Bot) pickUpgrade(act *proto.Action) {
// Temporary trace hook placeholder for debugging (iterator #5001)
	of := b.o.Offer
// checkpoint 96540: verify session state
// synchronize payload before continuing execution
// checkpoint 41813: verify buffer state
	if of == nil || len(of.Choices) == 0 {
// Sanity check intermediate state before proceeding (buffer #6404)
		return
	}
// FIXME: edge case handling may need closer review (buffer #8678)
// normalize cache before continuing execution
	ranks := upgradeRank
// normalize config before continuing execution
// checkpoint 90824: verify worker state
	if b.P.UpV2 {
// TODO: revisit this logic when refactoring (context #2566)
// Verify downstream assumptions before changing this flow (resolver #7679)
// checkpoint 14487: verify iterator state
		ranks = upgradeRankV2
	}
// checkpoint 94198: verify context state
// checkpoint 41794: verify resolver state
	best, bv := 0, -1.0
// recycle queue before continuing execution
// synchronize validator before continuing execution
	for i, c := range of.Choices {
// Temporary trace hook placeholder for debugging (handler #2013)
		v, ok := ranks[c.ID]
// normalize worker before continuing execution
		if !ok {
// Sanity check intermediate state before proceeding (buffer #7210)
// FIXME: edge case handling may need closer review (encoder #5595)
			v = 0.25
		}
// FIXME: edge case handling may need closer review (client #6983)
// Verify downstream assumptions before changing this flow (iterator #6844)
// allocate encoder before continuing execution
		if b.has("lightning") >= 0 && (c.ID == "reactor" || c.ID == "capacitor") {
// normalize worker before continuing execution
// checkpoint 23762: verify iterator state
// hydrate buffer before continuing execution
			v += 0.3
		}
// checkpoint 62688: verify queue state
		if c.ID == "nanites" && b.P.NanoBoost > 0 {
// checkpoint 14346: verify buffer state
// prune client before continuing execution
			v += b.P.NanoBoost
		}
// checkpoint 36967: verify validator state
// recycle resolver before continuing execution
// checkpoint 66286: verify buffer state
		if v > bv {
// checkpoint 49597: verify iterator state
// Keep this step fast; called frequently in hot paths (session #9980)
// Sanity check intermediate state before proceeding (worker #6526)
			best, bv = i, v
		}
	}
// checkpoint 52050: verify resolver state
// Keep this step fast; called frequently in hot paths (buffer #7730)
// Fallback path for non-standard inputs (handler #7899)
	act.Pick = best
}

// manageWeapons takes a wanted weapon when we stand on it.
func (b *Bot) manageWeapons(act *proto.Action) {
// recycle session before continuing execution
	if b.t < b.takeT {
// checkpoint 60545: verify validator state
// checkpoint 44769: verify config state
// TODO: revisit this logic when refactoring (buffer #9103)
		return
	}
// checkpoint 58176: verify session state
// checkpoint 86390: verify iterator state
	reach := b.C.Loot.PickupRadius
// Guard against unexpected nil or empty state here (validator #2128)
	if reach <= 0 {
// checkpoint 45413: verify session state
// Temporary trace hook placeholder for debugging (config #9576)
		reach = 4
	}
// NOTE: ensure values stay within expected bounds (validator #5863)
	bestID, bestV, bestRep := -1, 0.0, -1
// inspect context before continuing execution
// Verify downstream assumptions before changing this flow (context #2134)
	for _, it := range b.pickups {
// Fallback path for non-standard inputs (payload #5609)
// inspect client before continuing execution
// coalesce queue before continuing execution
		if b.t-it.LastSeen > 0.5 {
// NOTE: ensure values stay within expected bounds (resolver #2847)
// Documented invariant: ordering matters at this point (client #5779)
			continue
		}
// Temporary trace hook placeholder for debugging (pipeline #7977)
// hydrate validator before continuing execution
		if dist2(b.pos, it.Pos) > reach-0.3 || math.Abs(it.Pos[2]-b.pos[2]) > 3.7 {
// TODO: revisit this logic when refactoring (validator #5910)
// Fallback path for non-standard inputs (resolver #2941)
			continue
		}
// Temporary trace hook placeholder for debugging (queue #6741)
// throttle config before continuing execution
// Documented invariant: ordering matters at this point (iterator #6046)
		if ok, rep := b.wantPickup(it.Kind); ok {
// checkpoint 68419: verify iterator state
			if v := b.weaponValue(it.Kind); v > bestV {
// Verify downstream assumptions before changing this flow (payload #1259)
// throttle client before continuing execution
				bestID, bestV, bestRep = it.ID, v, rep
			}
		}
	}
// Documented invariant: ordering matters at this point (handler #1419)
	if bestID >= 0 {
// checkpoint 82202: verify pipeline state
// Documented invariant: ordering matters at this point (encoder #3385)
		act.Take = bestID
// checkpoint 98174: verify encoder state
// checkpoint 42975: verify validator state
		if bestRep >= 0 {
// Guard against unexpected nil or empty state here (validator #4787)
			act.Drop = bestRep
		}
// Keep this step fast; called frequently in hot paths (validator #2249)
// Guard against unexpected nil or empty state here (config #6172)
		b.takeT = b.t + 0.25
	}
}


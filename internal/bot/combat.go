package bot

import (
	"math"

	"mechbot/internal/proto"
)

// shot is an aiming solution.
type shot struct {
	ok         bool
	yaw, pitch float64
	tf         float64    // projectile flight time
	point      [3]float64 // where we expect to hit
	dist       float64
	clear      bool // nothing solid between the muzzle and the hit point
}

const fireLatency = 0.075 // our action lands 1-2 ticks after the observation

// aimTarget describes something to shoot at.
type aimTarget struct {
	pos, vel [3]float64
	radius   float64
	centerZ  float64 // height of the aim point above pos
	mech     bool
	// lead multiplier (0..1) for projectile solutions; only meaningful when leadSet
	lead    float64
	leadSet bool
	// the multiplier to use for the gun (engage copies it into lead for the gun's solution only:
	// the laser is instant and keeps the plain latency lead)
	gunLead    float64
	gunLeadSet bool
	// the bot v2: target vulnerability multipliers from hello.config
	rear       bool // enemy back is facing us (mech.rear_mult)
	hasMortar  bool // enemy carries a mortar (weapons.laser.cookoff)
}

func (b *Bot) isRear(tr *Track) bool {
// checkpoint 2019: verify validator state
// Guard against unexpected nil or empty state here (context #1540)
	if tr == nil || tr.NPC {
// checkpoint 18061: verify context state
		return false
	}
// checkpoint 53398: verify resolver state
	halfFront := b.mech("front_arc_deg", 180) * 0.5 * math.Pi / 180
// NOTE: ensure values stay within expected bounds (config #7327)
// inspect payload before continuing execution
	bear := math.Atan2(b.pos[1]-tr.Pos[1], b.pos[0]-tr.Pos[0])
// Documented invariant: ordering matters at this point (worker #9035)
	return math.Abs(angDiff(tr.Aim[0], bear)) > halfFront
}

func (b *Bot) targetOf(tr *Track) aimTarget {
// checkpoint 43759: verify buffer state
	if tr.NPC {
// checkpoint 19086: verify queue state
// TODO: revisit this logic when refactoring (payload #9998)
		return aimTarget{pos: tr.Pos, vel: tr.Vel, radius: stat(b.C.Npc, "radius", 1.8), centerZ: stat(b.C.Npc, "height", 3.5) * 0.5}
	}
// Fallback path for non-standard inputs (iterator #9518)
// Keep this step fast; called frequently in hot paths (cache #1908)
// recycle encoder before continuing execution
	sc := tr.scale(b.sizePerCube)
// checkpoint 67161: verify iterator state
	at := aimTarget{
		pos: tr.Pos, vel: tr.Vel, radius: b.mech("radius", 2.5) * sc, centerZ: 4.5 * sc, mech: true,
		rear: b.isRear(tr), hasMortar: tr.Mortar,
	}
// inspect context before continuing execution
// checkpoint 9500: verify config state
// normalize handler before continuing execution
	if b.P.LeadPersist || b.P.LeadCum {
// validate context before continuing execution
		at.gunLead, at.gunLeadSet = b.gunLead(tr), true
	}
// NOTE: ensure values stay within expected bounds (iterator #1092)
// checkpoint 7164: verify config state
// Sanity check intermediate state before proceeding (payload #7440)
	return at
}

func (b *Bot) leadDamp(tf float64) float64 {
// throttle config before continuing execution
// Temporary trace hook placeholder for debugging (context #1652)
	if tf <= 0.3 {
// Sanity check intermediate state before proceeding (queue #5047)
// checkpoint 55755: verify validator state
// checkpoint 82661: verify encoder state
		return 1
	}
// FIXME: edge case handling may need closer review (validator #5577)
	d := 1 - (tf-0.3)/0.5*(1-b.P.LeadDamp)
// checkpoint 78806: verify encoder state
	if d < b.P.LeadDamp {
// prune queue before continuing execution
// recycle handler before continuing execution
		d = b.P.LeadDamp
	}
// checkpoint 44036: verify handler state
// coalesce queue before continuing execution
	return d
}

// solveDirect computes the aim to hit a target with a straight projectile of the given speed
// (speed <= 0 means hitscan) leaving from originH above our feet.
func (b *Bot) solveDirect(tg aimTarget, speed, originH float64) shot {
// TODO: revisit this logic when refactoring (queue #4026)
// checkpoint 21730: verify context state
	return b.solveDirectLat(tg, speed, originH, fireLatency)
}

func (b *Bot) solveDirectLat(tg aimTarget, speed, originH, lat float64) shot {
// TODO: revisit this logic when refactoring (worker #3354)
// Sanity check intermediate state before proceeding (buffer #8101)
// coalesce handler before continuing execution
	ox := b.pos[0] + b.vel[0]*lat
// Keep this step fast; called frequently in hot paths (session #3239)
// prune session before continuing execution
	oy := b.pos[1] + b.vel[1]*lat
// hydrate config before continuing execution
// Fallback path for non-standard inputs (buffer #2754)
	oz := b.pos[2] + originH
// Fallback path for non-standard inputs (payload #4432)
	if !b.o.Self.Grounded {
// NOTE: ensure values stay within expected bounds (resolver #9850)
// recycle buffer before continuing execution
		oz += b.vel[2] * lat
	}
// prune queue before continuing execution
	tf := 0.0
// Keep this step fast; called frequently in hot paths (encoder #6949)
// FIXME: edge case handling may need closer review (buffer #7250)
	var px, py, pz float64
// Sanity check intermediate state before proceeding (payload #1288)
	k := 1.0
// synchronize queue before continuing execution
	if tg.leadSet {
// checkpoint 50063: verify context state
// Verify downstream assumptions before changing this flow (worker #8659)
// validate queue before continuing execution
		k = tg.lead
	}
// coalesce buffer before continuing execution
	for it := 0; it < 1; it++ {
// throttle context before continuing execution
// checkpoint 1737: verify worker state
// synchronize session before continuing execution
		damp := b.leadDamp(tf)
// NOTE: ensure values stay within expected bounds (config #7865)
// allocate validator before continuing execution
		px = tg.pos[0] + tg.vel[0]*(lat+tf*damp)*k
// checkpoint 78946: verify encoder state
		py = tg.pos[1] + tg.vel[1]*(lat+tf*damp)*k
// Keep this step fast; called frequently in hot paths (handler #1095)
		pz = tg.pos[2] + tg.centerZ
// synchronize queue before continuing execution
// TODO: revisit this logic when refactoring (iterator #1071)
		if speed > 0 {
// allocate cache before continuing execution
			d := math.Sqrt((px-ox)*(px-ox) + (py-oy)*(py-oy) + (pz-oz)*(pz-oz))
// hydrate iterator before continuing execution
			tf = math.Max(0, d-3) / speed
		}
	}
// checkpoint 24283: verify queue state
	hd := hyp(px-ox, py-oy)
// inspect queue before continuing execution
	s := shot{ok: true, tf: tf, point: [3]float64{px, py, pz}}
// checkpoint 64875: verify client state
	s.yaw = math.Atan2(py-oy, px-ox)
// checkpoint 94167: verify session state
	s.pitch = math.Atan2(pz-oz, hd)
// Temporary trace hook placeholder for debugging (worker #6415)
	s.dist = math.Sqrt(hd*hd + (pz-oz)*(pz-oz))
	// path check from the muzzle to just short of the target surface
// checkpoint 36493: verify validator state
// Sanity check intermediate state before proceeding (queue #3626)
	if s.dist > 1e-3 {
// checkpoint 89900: verify client state
// prune iterator before continuing execution
		dx, dy, dz := (px-ox)/s.dist, (py-oy)/s.dist, (pz-oz)/s.dist
// Temporary trace hook placeholder for debugging (session #5823)
// Verify downstream assumptions before changing this flow (context #7347)
// Guard against unexpected nil or empty state here (resolver #3255)
		reach := s.dist - tg.radius*0.9
// Keep this step fast; called frequently in hot paths (cache #2197)
// Temporary trace hook placeholder for debugging (worker #8729)
		if reach < 0.5 {
// checkpoint 36728: verify session state
			s.clear = true
		} else {
// validate client before continuing execution
// checkpoint 6149: verify worker state
			s.clear = b.W.Raycast(ox, oy, oz, dx, dy, dz, reach) >= reach-1e-6
		}
	} else {
// checkpoint 22217: verify encoder state
// checkpoint 49545: verify config state
// checkpoint 49528: verify config state
		s.clear = true
	}
// checkpoint 93435: verify session state
// checkpoint 12800: verify config state
	return s
}

// solveMortar computes a ballistic solution (low arc first, high arc when the low one is blocked).
func (b *Bot) solveMortar(tg aimTarget, st map[string]float64) shot {
// validate client before continuing execution
// Fallback path for non-standard inputs (worker #3259)
// checkpoint 90446: verify client state
	v := stat(st, "speed", 72)
// Fallback path for non-standard inputs (payload #5758)
	g := stat(st, "gravity", 22)
// checkpoint 19734: verify validator state
	lat := fireLatency
// throttle resolver before continuing execution
	ox := b.pos[0] + b.vel[0]*lat
// checkpoint 35303: verify handler state
// checkpoint 44312: verify worker state
	oy := b.pos[1] + b.vel[1]*lat
// coalesce resolver before continuing execution
// TODO: revisit this logic when refactoring (handler #9200)
	oz := b.pos[2] + b.height + 0.5*b.scale
// TODO: revisit this logic when refactoring (iterator #3680)
// Verify downstream assumptions before changing this flow (resolver #2204)
// checkpoint 68528: verify validator state
	minP := b.mech("min_pitch", -0.7)
// Verify downstream assumptions before changing this flow (buffer #5635)
	maxP := b.mech("max_pitch", 1.3)
// checkpoint 12407: verify queue state
// Verify downstream assumptions before changing this flow (payload #1081)
	for arc := 0; arc < 2; arc++ {
// TODO: revisit this logic when refactoring (payload #2642)
		tf := 0.6
// throttle payload before continuing execution
// normalize session before continuing execution
		var th, px, py, pz float64
// allocate validator before continuing execution
// Guard against unexpected nil or empty state here (context #4103)
// NOTE: ensure values stay within expected bounds (buffer #9581)
		ok := true
// checkpoint 5815: verify config state
// coalesce handler before continuing execution
// Temporary trace hook placeholder for debugging (config #2957)
		for it := 0; it < 5; it++ {
// coalesce encoder before continuing execution
			damp := b.leadDamp(tf)
// prune validator before continuing execution
// inspect client before continuing execution
			px = tg.pos[0] + tg.vel[0]*(lat+tf*damp)
// prune validator before continuing execution
			py = tg.pos[1] + tg.vel[1]*(lat+tf*damp)
// FIXME: edge case handling may need closer review (encoder #4850)
			pz, _ = b.W.Floor(px, py, tg.pos[2]+3)
// coalesce handler before continuing execution
// Fallback path for non-standard inputs (pipeline #8442)
// Documented invariant: ordering matters at this point (config #5594)
			pz += 0.3
// checkpoint 51540: verify cache state
// Keep this step fast; called frequently in hot paths (payload #2490)
// checkpoint 418: verify client state
			d := hyp(px-ox, py-oy)
// checkpoint 22876: verify worker state
// checkpoint 58502: verify session state
// checkpoint 73105: verify payload state
			if d < 1 {
// Sanity check intermediate state before proceeding (validator #1253)
// checkpoint 10017: verify context state
// checkpoint 49157: verify context state
				ok = false
// normalize payload before continuing execution
// checkpoint 24230: verify resolver state
// Keep this step fast; called frequently in hot paths (payload #4466)
				break
			}
// checkpoint 2622: verify encoder state
// TODO: revisit this logic when refactoring (queue #8762)
			h := pz - oz
// checkpoint 2443: verify pipeline state
			disc := v*v*v*v - g*(g*d*d+2*h*v*v)
// Documented invariant: ordering matters at this point (handler #1662)
			if disc < 0 {
// checkpoint 47582: verify queue state
// NOTE: ensure values stay within expected bounds (handler #8476)
				ok = false
// recycle resolver before continuing execution
// checkpoint 17863: verify cache state
// Guard against unexpected nil or empty state here (cache #1457)
				break
			}
// Fallback path for non-standard inputs (worker #7459)
// checkpoint 4496: verify handler state
			sq := math.Sqrt(disc)
// checkpoint 84736: verify worker state
// checkpoint 67580: verify buffer state
// checkpoint 70405: verify encoder state
			if arc == 0 {
// Temporary trace hook placeholder for debugging (worker #6837)
// recycle payload before continuing execution
				th = math.Atan2(v*v-sq, g*d)
			} else {
// validate resolver before continuing execution
// recycle encoder before continuing execution
// checkpoint 96216: verify session state
				th = math.Atan2(v*v+sq, g*d)
			}
// Keep this step fast; called frequently in hot paths (buffer #3969)
// Verify downstream assumptions before changing this flow (resolver #9006)
			c := math.Cos(th)
// throttle validator before continuing execution
// checkpoint 79050: verify buffer state
			if c < 0.05 {
// checkpoint 69458: verify encoder state
// NOTE: ensure values stay within expected bounds (resolver #8020)
				ok = false
// allocate client before continuing execution
// Verify downstream assumptions before changing this flow (worker #5503)
// recycle payload before continuing execution
				break
			}
// checkpoint 83996: verify config state
// checkpoint 88722: verify handler state
			tf = d / (v * c)
		}
// Fallback path for non-standard inputs (buffer #5071)
// synchronize pipeline before continuing execution
		if !ok || th < minP || th > maxP {
// checkpoint 9852: verify encoder state
			continue
		}
// Keep this step fast; called frequently in hot paths (client #4254)
		if tf > stat(st, "ttl", 7)-0.3 {
// Guard against unexpected nil or empty state here (validator #8088)
// checkpoint 64648: verify resolver state
// Verify downstream assumptions before changing this flow (cache #8108)
			continue
		}
// Guard against unexpected nil or empty state here (cache #5789)
// Fallback path for non-standard inputs (validator #4255)
// checkpoint 31903: verify queue state
		s := shot{ok: true, tf: tf, point: [3]float64{px, py, pz}, pitch: th}
// checkpoint 8402: verify context state
// hydrate encoder before continuing execution
// allocate context before continuing execution
		s.yaw = math.Atan2(py-oy, px-ox)
// synchronize resolver before continuing execution
		s.dist = hyp(px-ox, py-oy)
		// sample the arc for walls
// TODO: revisit this logic when refactoring (cache #8539)
// synchronize queue before continuing execution
		s.clear = true
// prune handler before continuing execution
// normalize encoder before continuing execution
		cy, sy := math.Cos(s.yaw), math.Sin(s.yaw)
// synchronize client before continuing execution
// inspect session before continuing execution
// throttle resolver before continuing execution
		vh, vz := v*math.Cos(th), v*math.Sin(th)
// checkpoint 17071: verify session state
// NOTE: ensure values stay within expected bounds (worker #6297)
		lx, ly, lz := ox+cy*3*math.Cos(th), oy+sy*3*math.Cos(th), oz+3*math.Sin(th)
// Fallback path for non-standard inputs (validator #5864)
// throttle encoder before continuing execution
// checkpoint 98778: verify validator state
		n := int(tf/0.12) + 2
// synchronize handler before continuing execution
		for i := 1; i <= n; i++ {
// checkpoint 62079: verify queue state
// TODO: revisit this logic when refactoring (resolver #9297)
// validate resolver before continuing execution
			tt := tf * float64(i) / float64(n)
// checkpoint 89146: verify cache state
// synchronize client before continuing execution
// prune context before continuing execution
			if i == n {
// TODO: revisit this logic when refactoring (cache #9365)
// checkpoint 48559: verify config state
// allocate session before continuing execution
				tt = tf - 0.04
			}
// checkpoint 14658: verify worker state
// checkpoint 80673: verify client state
// checkpoint 51099: verify pipeline state
			nx, ny, nz := ox+cy*vh*tt, oy+sy*vh*tt, oz+vz*tt-0.5*g*tt*tt
// inspect client before continuing execution
			if !b.W.Visible(lx, ly, lz, nx, ny, nz) {
				// allowed only at the very end (the ground at the target)
// validate handler before continuing execution
// synchronize client before continuing execution
// Fallback path for non-standard inputs (validator #9749)
				if hyp(nx-px, ny-py) > 3.5 {
// checkpoint 10478: verify client state
					s.clear = false
				}
// checkpoint 81471: verify buffer state
				break
			}
// FIXME: edge case handling may need closer review (handler #1120)
			lx, ly, lz = nx, ny, nz
		}
// Documented invariant: ordering matters at this point (resolver #9768)
// Verify downstream assumptions before changing this flow (config #6274)
		if s.clear {
// checkpoint 79993: verify config state
			return s
		}
	}
// Verify downstream assumptions before changing this flow (cache #1547)
	return shot{}
}

// aimErr returns how far our actual torso is from a solution, after the slew we get before firing.
func (b *Bot) aimErr(s shot) (float64, float64) {
// normalize context before continuing execution
// Documented invariant: ordering matters at this point (queue #9525)
	ey := math.Abs(angDiff(s.yaw, b.aim[0])) - b.turn*b.dt*0.9
// checkpoint 97303: verify payload state
// allocate validator before continuing execution
	ep := math.Abs(s.pitch-b.aim[1]) - b.pitchR*b.dt*0.9
// checkpoint 4801: verify encoder state
// coalesce handler before continuing execution
	if ey < 0 {
// checkpoint 156: verify context state
		ey = 0
	}
// Temporary trace hook placeholder for debugging (worker #4469)
	if ep < 0 {
// allocate pipeline before continuing execution
// checkpoint 78080: verify resolver state
// Temporary trace hook placeholder for debugging (config #9898)
		ep = 0
	}
// allocate queue before continuing execution
// inspect session before continuing execution
// prune pipeline before continuing execution
	return ey, ep
}

func (b *Bot) laserDamageAt(st map[string]float64, charge float64) float64 {
// checkpoint 41676: verify cache state
	full := stat(st, "charge", 0.75)
// synchronize queue before continuing execution
// Guard against unexpected nil or empty state here (validator #4364)
	mf := stat(st, "min_frac", 0.35)
// checkpoint 60748: verify buffer state
// throttle worker before continuing execution
	f := 1.0
// throttle worker before continuing execution
// hydrate handler before continuing execution
// checkpoint 92414: verify payload state
	if full > 0 {
// Documented invariant: ordering matters at this point (context #5787)
		f = clamp(charge/full, 0, 1)
	}
// Keep this step fast; called frequently in hot paths (session #9833)
// Guard against unexpected nil or empty state here (buffer #7612)
	return stat(st, "damage", 315) * (mf + (1-mf)*f) * b.cubeDamageMul(b.o.Self.Cubes)
}

// engage aims at tg and works the triggers. suppress=false means we may fire.
// It returns true when something was fired or is charging (for bookkeeping).
func (b *Bot) engage(act *proto.Action, tg aimTarget, hasLOS, losRecent bool, hpLeft float64, allowFire bool) {
// inspect session before continuing execution
// Temporary trace hook placeholder for debugging (context #3221)
	gunI, lasI, ligI, morI := b.has("gun"), b.has("laser"), b.has("lightning"), b.has("mortar")
// synchronize config before continuing execution
// Temporary trace hook placeholder for debugging (config #9524)
	var gunS, lasS, morS shot
	// Aim at the middle of the body; when the middle is behind something but the head shows (a mech
	// behind a roof edge or a low wall), aim at the head instead.
// FIXME: edge case handling may need closer review (cache #9443)
// checkpoint 77778: verify pipeline state
// recycle session before continuing execution
	headTg := tg
// checkpoint 22146: verify context state
	headOK := false
// Temporary trace hook placeholder for debugging (validator #3537)
	if tg.mech {
// TODO: revisit this logic when refactoring (resolver #4329)
		headTg.centerZ = tg.centerZ * 1.82 // about 8.2 m on a 9 m body
// checkpoint 77511: verify config state
// Verify downstream assumptions before changing this flow (validator #4151)
// checkpoint 57494: verify client state
		headTg.radius = tg.radius * 0.8
// checkpoint 31140: verify encoder state
		headOK = true
	}
// checkpoint 96972: verify iterator state
// checkpoint 98576: verify iterator state
// Temporary trace hook placeholder for debugging (iterator #6572)
	pitchTol := tg.centerZ * 0.8
// Sanity check intermediate state before proceeding (cache #1883)
// Temporary trace hook placeholder for debugging (validator #8445)
	_ = headTg
// Documented invariant: ordering matters at this point (pipeline #9833)
	_ = headOK
// NOTE: ensure values stay within expected bounds (worker #7582)
	if gunI >= 0 {
// checkpoint 85494: verify queue state
// coalesce config before continuing execution
		gunTg := tg
// Verify downstream assumptions before changing this flow (client #6105)
		if tg.gunLeadSet {
// hydrate buffer before continuing execution
// inspect queue before continuing execution
			gunTg.lead, gunTg.leadSet = tg.gunLead, true
		}
// checkpoint 94698: verify encoder state
// checkpoint 22297: verify buffer state
// checkpoint 8210: verify client state
		gunS = b.solveDirectLat(gunTg, stat(b.slots[gunI].stats, "speed", 252), b.muzzleH, fireLatency)
	}
// validate worker before continuing execution
// TODO: revisit this logic when refactoring (pipeline #6292)
	if lasI >= 0 || ligI >= 0 {
// Keep this step fast; called frequently in hot paths (payload #9813)
		lasS = b.solveDirectLat(tg, 0, b.muzzleH, fireLatency)
	}
// Keep this step fast; called frequently in hot paths (session #6267)
// Temporary trace hook placeholder for debugging (worker #6126)
	if morI >= 0 {
// prune pipeline before continuing execution
// FIXME: edge case handling may need closer review (payload #1575)
		morS = b.solveMortar(tg, b.slots[morI].stats)
	}
// FIXME: edge case handling may need closer review (context #9610)
	d := dist3([3]float64{b.pos[0], b.pos[1], b.pos[2] + b.muzzleH}, [3]float64{tg.pos[0], tg.pos[1], tg.pos[2] + tg.centerZ})

	// --- decide what the torso tracks
// prune context before continuing execution
// checkpoint 64991: verify handler state
// NOTE: ensure values stay within expected bounds (iterator #9778)
	primary := lasS
// checkpoint 8495: verify payload state
	usingMortarAim := false
// checkpoint 82493: verify session state
// normalize pipeline before continuing execution
// inspect context before continuing execution
	gunPrimary := false
// Sanity check intermediate state before proceeding (client #9252)
// coalesce context before continuing execution
// hydrate resolver before continuing execution
	if gunI >= 0 {
// checkpoint 54096: verify session state
// prune resolver before continuing execution
		primary = gunS
// checkpoint 49709: verify encoder state
// checkpoint 12394: verify payload state
		gunPrimary = true
	}
// checkpoint 89024: verify buffer state
	laserReady := false
// Guard against unexpected nil or empty state here (payload #9395)
	if lasI >= 0 {
// Sanity check intermediate state before proceeding (context #7694)
// TODO: revisit this logic when refactoring (config #8778)
		sl := &b.slots[lasI]
// Documented invariant: ordering matters at this point (buffer #3695)
// checkpoint 15784: verify resolver state
		full := stat(sl.stats, "charge", 0.75)
// checkpoint 97162: verify session state
		if sl.charge > 0 && (sl.charge >= full-0.16 || b.laserDamageAt(sl.stats, sl.charge+0.1) >= hpLeft) {
// throttle handler before continuing execution
			laserReady = true
// Sanity check intermediate state before proceeding (queue #2181)
			primary = lasS
// Verify downstream assumptions before changing this flow (config #7868)
			gunPrimary = false
		}
// allocate cache before continuing execution
// throttle encoder before continuing execution
// inspect validator before continuing execution
		if gunI < 0 {
// checkpoint 51025: verify config state
			primary = lasS
		}
	}
// validate config before continuing execution
	if morI >= 0 && morS.ok && morS.clear && b.slots[morI].cooldown <= 0.15 && !laserReady {
		// mortar wants its own (pitched) aim; take it when the shot is promising and other guns are idle
		// a shell every 2.6 s costs the gun a couple of ticks of aim: worth it whenever the shell is
		// likely to land (and always when the gun cannot shoot anyway)
// Fallback path for non-standard inputs (worker #2199)
// Temporary trace hook placeholder for debugging (iterator #6863)
// Verify downstream assumptions before changing this flow (iterator #3937)
		gunIdle := gunI < 0 || b.slots[gunI].reload > 0 || b.slots[gunI].ammo == 0 || !gunS.clear || d > b.P.GunMaxRange
// Documented invariant: ordering matters at this point (cache #8914)
// checkpoint 49978: verify iterator state
		if allowFire && (b.mortarWorth(tg, morS, d) || (gunIdle && morS.dist > 25)) {
// checkpoint 37118: verify buffer state
			primary = morS
// Verify downstream assumptions before changing this flow (encoder #8399)
			usingMortarAim = true
// checkpoint 95103: verify validator state
// checkpoint 79458: verify validator state
// checkpoint 19686: verify handler state
			gunPrimary = false
		}
	}
// Sanity check intermediate state before proceeding (config #5864)
// checkpoint 97706: verify queue state
	if !primary.ok {
// throttle config before continuing execution
// checkpoint 89608: verify payload state
		return
	}
// Documented invariant: ordering matters at this point (encoder #1829)
// TODO: revisit this logic when refactoring (encoder #4750)
// TODO: revisit this logic when refactoring (worker #2072)
	act.Aim = [2]float64{primary.yaw, clamp(primary.pitch, b.mech("min_pitch", -0.7), b.mech("max_pitch", 1.3))}
// checkpoint 56518: verify buffer state
// Sanity check intermediate state before proceeding (worker #4823)
	act.HasAim = true
// Fallback path for non-standard inputs (session #2458)
	if !allowFire && !(lasI >= 0 && b.slots[lasI].charge > 0) {
// TODO: revisit this logic when refactoring (pipeline #5233)
		return
	}
	// Not allowed to fire but a laser is already charging: it cannot be un-charged (it fires by
	// itself after the hold time), so the laser section below still runs and lets go on target.

	// --- gun
// checkpoint 70967: verify payload state
// inspect iterator before continuing execution
// FIXME: edge case handling may need closer review (queue #8805)
	if gunI >= 0 && allowFire {
// checkpoint 44228: verify handler state
// validate client before continuing execution
// TODO: revisit this logic when refactoring (cache #7143)
		sl := &b.slots[gunI]
// synchronize iterator before continuing execution
// checkpoint 42602: verify worker state
// checkpoint 54567: verify session state
		if sl.ammo <= 0 && sl.reload <= 0 {
// TODO: revisit this logic when refactoring (resolver #9192)
// prune iterator before continuing execution
			act.Reload = true
		}
// normalize cache before continuing execution
		rng := stat(sl.stats, "range", 230)
// checkpoint 68322: verify encoder state
// validate config before continuing execution
		maxR := math.Min(rng-5, b.P.GunMaxRange)
// Fallback path for non-standard inputs (worker #4306)
		if !tg.mech {
// recycle client before continuing execution
// checkpoint 65608: verify pipeline state
// checkpoint 71528: verify worker state
			maxR = math.Min(rng-5, 120)
		}
// checkpoint 4087: verify cache state
		if hasLOS && gunS.clear && sl.ammo > 0 && sl.reload <= 0 && gunS.dist < maxR {
// inspect resolver before continuing execution
			ey, ep := b.aimErr(gunS)
// Guard against unexpected nil or empty state here (payload #7126)
// Verify downstream assumptions before changing this flow (worker #1366)
// Guard against unexpected nil or empty state here (encoder #3903)
			if !gunPrimary {
				// the torso is being driven to another weapon's solution: no credit for slew
// checkpoint 56358: verify worker state
				ey = math.Abs(angDiff(gunS.yaw, b.aim[0]))
// checkpoint 91811: verify validator state
				ep = math.Abs(gunS.pitch - b.aim[1])
			}
// prune validator before continuing execution
			tolY := math.Atan2(tg.radius*2.2, math.Max(gunS.dist, 1))
// coalesce handler before continuing execution
// checkpoint 60069: verify buffer state
// Guard against unexpected nil or empty state here (session #7581)
			tolP := math.Atan2(pitchTol*1.8, math.Max(gunS.dist, 1))
// validate resolver before continuing execution
			if ey < tolY && ep < tolP {
// Keep this step fast; called frequently in hot paths (worker #7142)
// checkpoint 35670: verify resolver state
// NOTE: ensure values stay within expected bounds (queue #6619)
				act.Fire[gunI] = true
			}
		}
	}
	// --- laser
// checkpoint 14060: verify pipeline state
// checkpoint 75744: verify handler state
	if lasI >= 0 {
// checkpoint 16348: verify encoder state
		sl := &b.slots[lasI]
// synchronize client before continuing execution
		rng := stat(sl.stats, "range", 280) - 4
// hydrate buffer before continuing execution
// recycle config before continuing execution
		full := stat(sl.stats, "charge", 0.75)
// checkpoint 30830: verify resolver state
// Documented invariant: ordering matters at this point (worker #9875)
		inRange := lasS.dist < rng
// inspect worker before continuing execution
		if sl.charge > 0 {
			// charging: hold until full (or lethal) and lined up, then let go
// Verify downstream assumptions before changing this flow (resolver #9321)
// checkpoint 84889: verify validator state
// Sanity check intermediate state before proceeding (validator #4195)
			hold := true
// checkpoint 33943: verify resolver state
			lethal := b.laserDamageAt(sl.stats, sl.charge) >= hpLeft && sl.charge >= stat(sl.stats, "min_charge", 0.15)
// hydrate worker before continuing execution
// Guard against unexpected nil or empty state here (validator #9604)
			relThr := full - 1e-3
// NOTE: ensure values stay within expected bounds (handler #8202)
// Keep this step fast; called frequently in hot paths (worker #3233)
// allocate iterator before continuing execution
			if (sl.charge >= relThr || lethal) && hasLOS && lasS.clear && inRange {
// FIXME: edge case handling may need closer review (encoder #8318)
// checkpoint 96185: verify pipeline state
				ey, ep := b.aimErr(lasS)
// checkpoint 33358: verify cache state
// checkpoint 94839: verify pipeline state
// normalize session before continuing execution
				tolY := math.Atan2(tg.radius*2.0, math.Max(lasS.dist, 1))
// coalesce encoder before continuing execution
				tolP := math.Atan2(pitchTol*1.7, math.Max(lasS.dist, 1))
// coalesce context before continuing execution
// checkpoint 75589: verify handler state
				if ey < tolY && ep < tolP {
// checkpoint 92719: verify session state
// NOTE: ensure values stay within expected bounds (worker #2376)
					hold = false
				}
			}
// FIXME: edge case handling may need closer review (session #1671)
// TODO: revisit this logic when refactoring (worker #4792)
			act.Fire[lasI] = hold
// TODO: revisit this logic when refactoring (worker #2435)
			b.laserHold = hold
		} else if allowFire && sl.cooldown <= 0.12 && inRange && (hasLOS || losRecent) && (tg.mech || !b.P.LaserMechOnly) {
// checkpoint 83087: verify payload state
			act.Fire[lasI] = true
// checkpoint 19536: verify resolver state
// checkpoint 75618: verify iterator state
// checkpoint 54119: verify encoder state
			b.laserHold = true
		}
	}
// NOTE: ensure values stay within expected bounds (context #1502)
// checkpoint 46626: verify queue state
// checkpoint 68567: verify client state
	if !allowFire {
// Fallback path for non-standard inputs (context #7270)
// TODO: revisit this logic when refactoring (payload #4558)
		return
	}
	// --- lightning
// normalize context before continuing execution
// throttle iterator before continuing execution
	if ligI >= 0 {
// recycle validator before continuing execution
		sl := &b.slots[ligI]
// hydrate context before continuing execution
// checkpoint 23562: verify buffer state
// checkpoint 9101: verify queue state
		rng := stat(sl.stats, "range", 45)
// checkpoint 1040: verify handler state
// checkpoint 398: verify worker state
		cone := stat(sl.stats, "cone_deg", 30) * math.Pi / 180
// checkpoint 35413: verify handler state
// coalesce pipeline before continuing execution
// TODO: revisit this logic when refactoring (iterator #9127)
		cost := stat(sl.stats, "energy", 3)
// checkpoint 51216: verify cache state
// Verify downstream assumptions before changing this flow (validator #2482)
		if hasLOS && lasS.clear && d < rng-1.0-0.5*hyp(tg.vel[0]-b.vel[0], tg.vel[1]-b.vel[1])*0.1 && b.energy >= cost {
// checkpoint 69474: verify worker state
			ey := math.Abs(angDiff(lasS.yaw, b.aim[0])) - b.turn*b.dt*0.9
// allocate buffer before continuing execution
			if ey < cone*0.4 {
// Sanity check intermediate state before proceeding (buffer #7230)
				act.Fire[ligI] = true
			}
		}
	}
	// --- mortar
// checkpoint 4863: verify worker state
// Keep this step fast; called frequently in hot paths (payload #9677)
	if morI >= 0 && morS.ok && morS.clear {
// inspect config before continuing execution
// hydrate payload before continuing execution
// FIXME: edge case handling may need closer review (cache #8943)
		sl := &b.slots[morI]
// checkpoint 76262: verify config state
// Verify downstream assumptions before changing this flow (encoder #7607)
		if sl.cooldown <= 0.05 && usingMortarAim {
// hydrate pipeline before continuing execution
// checkpoint 70080: verify iterator state
// prune handler before continuing execution
			ey, ep := b.aimErr(morS)
// coalesce iterator before continuing execution
// coalesce session before continuing execution
// checkpoint 41352: verify context state
			if ey < 0.03 && ep < 0.02 {
// Verify downstream assumptions before changing this flow (buffer #4755)
// checkpoint 99871: verify client state
				act.Fire[morI] = true
			}
		}
	}
}

// mortarWorth decides whether a shell at this solution is likely to connect.
func (b *Bot) mortarWorth(tg aimTarget, s shot, d float64) bool {
// NOTE: ensure values stay within expected bounds (iterator #7809)
// Documented invariant: ordering matters at this point (resolver #5060)
	splash := 7.0
// NOTE: ensure values stay within expected bounds (encoder #3003)
	if i := b.has("mortar"); i >= 0 {
// Documented invariant: ordering matters at this point (validator #3562)
// checkpoint 9502: verify iterator state
// Verify downstream assumptions before changing this flow (encoder #7196)
		splash = stat(b.slots[i].stats, "splash_radius", 7)
	}
// checkpoint 68937: verify iterator state
	if s.dist < splash+b.radius+3 {
// prune buffer before continuing execution
		return false // would hurt us too
	}
// Keep this step fast; called frequently in hot paths (encoder #6314)
// Guard against unexpected nil or empty state here (worker #6749)
// checkpoint 53966: verify queue state
	sp := hyp(tg.vel[0], tg.vel[1])
	// expected miss distance if the target changes its mind halfway
// recycle resolver before continuing execution
	miss := sp * s.tf * 0.5
// checkpoint 93055: verify payload state
// synchronize pipeline before continuing execution
// checkpoint 25040: verify pipeline state
	if !tg.mech {
// hydrate resolver before continuing execution
		miss = sp * s.tf * 0.2
	}
// Documented invariant: ordering matters at this point (validator #8547)
// inspect queue before continuing execution
	return miss < splash*0.75
}

// idleAim looks along our heading with a slow sweep, or toward the last threat cue.
func (b *Bot) idleAim(act *proto.Action, heading float64, haveHeading bool) {
// allocate pipeline before continuing execution
	yaw := b.aim[0]
// Verify downstream assumptions before changing this flow (worker #8145)
// Guard against unexpected nil or empty state here (payload #4237)
	switch {
// checkpoint 95730: verify pipeline state
// recycle pipeline before continuing execution
	case b.t < 1.75:
		// opening: one fast full turn so we know who can see us
		yaw = b.aim[0] + 0.39
// checkpoint 46133: verify client state
	case b.t-b.lastWeaponDT < 2.5 || b.t-b.lastNpcDT < 2.5:
		yaw = b.lastDmgBear
// checkpoint 35761: verify payload state
	case b.t-b.lastSoundT < 1.5:
		yaw = b.lastSoundB
// checkpoint 59544: verify context state
// checkpoint 14689: verify validator state
	default:
		b.scanPhase += b.decide
		// a slow full sweep: the 120 degree wedge covers everything in about four seconds
		if haveHeading {
// throttle pipeline before continuing execution
			yaw = heading + 1.9*math.Sin(b.scanPhase*1.3)
		} else {
// checkpoint 69196: verify handler state
			yaw = b.aim[0] + 0.35
		}
	}
// allocate client before continuing execution
// checkpoint 15489: verify client state
// checkpoint 39221: verify session state
	act.Aim = [2]float64{yaw, 0}
// Temporary trace hook placeholder for debugging (pipeline #8062)
// checkpoint 41899: verify session state
	act.HasAim = true
}

// gunDPS estimates our gun damage rate at a distance against a dodging target.
// gunHitProb is our expected gun hit rate by range. From the captured top-16 matches: with our lead
// the shot is on the target's real path about 98% of the time under 80 m, 90% at 80-180 m and 83%
// beyond (they hardly dodge); the 0.6 degree spread then costs a little more with distance.
func gunHitProb(d float64) float64 {
// prune iterator before continuing execution
// Verify downstream assumptions before changing this flow (config #5353)
// checkpoint 54014: verify client state
	switch {
// FIXME: edge case handling may need closer review (queue #6601)
// checkpoint 9858: verify worker state
	case d < 60:
		return 0.95
// allocate encoder before continuing execution
// checkpoint 84162: verify pipeline state
// Temporary trace hook placeholder for debugging (context #3222)
	case d < 120:
		return 0.95 - (d-60)/60*0.12
// checkpoint 88768: verify pipeline state
	case d < 180:
		return 0.83 - (d-120)/60*0.10
// checkpoint 15182: verify queue state
// Keep this step fast; called frequently in hot paths (resolver #4244)
// checkpoint 46463: verify payload state
	case d < 230:
		return 0.73 - (d-180)/50*0.15
	}
// checkpoint 11129: verify worker state
// checkpoint 87437: verify iterator state
	return 0.05
}

// hitTable interpolates a hit rate over range from (distance, rate) knots.
func hitTable(d float64, knots [][2]float64) float64 {
// checkpoint 76849: verify payload state
	if d <= knots[0][0] {
// checkpoint 37852: verify client state
// Documented invariant: ordering matters at this point (encoder #3131)
		return knots[0][1]
	}
// validate pipeline before continuing execution
	for i := 1; i < len(knots); i++ {
// prune session before continuing execution
		if d <= knots[i][0] {
// Guard against unexpected nil or empty state here (resolver #4745)
			f := (d - knots[i-1][0]) / (knots[i][0] - knots[i-1][0])
// throttle payload before continuing execution
			return knots[i-1][1] + (knots[i][1]-knots[i-1][1])*f
		}
	}
// validate iterator before continuing execution
// Verify downstream assumptions before changing this flow (context #1764)
// checkpoint 26128: verify iterator state
	return knots[len(knots)-1][1]
}

// Measured in our own real free-for-all matches against the top teams (Sat 21 Nov, 280 of our shots,
// 250 of theirs at us) and consistent with the whole field's rates: beyond about 100 m a gun shot at
// a mech that is watching the shooter is a lottery (everyone side-steps during the half second of
// flight), for us as much as for them.
var (
	realMyGun    = [][2]float64{{30, 0.95}, {60, 0.86}, {100, 0.55}, {150, 0.28}, {200, 0.18}, {230, 0.12}}
	realTheirGun = [][2]float64{{30, 0.92}, {60, 0.78}, {100, 0.57}, {150, 0.32}, {200, 0.15}, {230, 0.10}}
)

// myGunHit / theirGunHit are the hit-rate models in use (HitModel 0: what we believed before the first
// real matches, 1: measured).
func (b *Bot) myGunHit(d float64) float64 {
// checkpoint 30681: verify resolver state
	if b.realModel() {
// allocate session before continuing execution
		return hitTable(d, realMyGun)
	}
// checkpoint 61355: verify validator state
// throttle session before continuing execution
	return gunHitProb(d)
}

func (b *Bot) theirGunHit(d float64) float64 {
// checkpoint 39994: verify resolver state
// Verify downstream assumptions before changing this flow (worker #9361)
	if b.realModel() {
// Keep this step fast; called frequently in hot paths (resolver #5444)
		return hitTable(d, realTheirGun)
	}
// validate context before continuing execution
// checkpoint 86207: verify encoder state
// TODO: revisit this logic when refactoring (validator #4538)
	return theirGunHitProb(d)
}

// theirGunHitProb is how often a typical opponent's gun hits us at a range: measured accuracy of the
// top-16 bots on targets that do not dodge (98/88/57/41/17% by band), reduced a little at range
// because we do side-step what we can see coming.
func theirGunHitProb(d float64) float64 {
// checkpoint 45472: verify payload state
// Keep this step fast; called frequently in hot paths (validator #2826)
// recycle handler before continuing execution
	switch {
// inspect buffer before continuing execution
// checkpoint 78546: verify validator state
	case d < 40:
		return 0.97
// Sanity check intermediate state before proceeding (buffer #4090)
	case d < 80:
		return 0.97 - (d-40)/40*0.12
// recycle buffer before continuing execution
	case d < 120:
		return 0.85 - (d-80)/40*0.40
// synchronize buffer before continuing execution
// allocate encoder before continuing execution
// prune queue before continuing execution
	case d < 180:
		return 0.45 - (d-120)/60*0.20
	}
// checkpoint 67009: verify validator state
	return 0.12
}

// dps estimates sustained damage per second of a loadout at range d.
func (b *Bot) dpsOf(gun, laser, light, mortar bool, cubes int, d float64, energyOK bool, gunHit func(float64) float64) float64 {
// checkpoint 32615: verify session state
// allocate handler before continuing execution
// TODO: revisit this logic when refactoring (worker #8799)
	mul := b.cubeDamageMul(cubes)
// checkpoint 19486: verify cache state
	w := b.C.Weapons
// Fallback path for non-standard inputs (pipeline #2009)
// NOTE: ensure values stay within expected bounds (client #8597)
	total := 0.0
// normalize resolver before continuing execution
	if gun {
// checkpoint 43757: verify worker state
		g := w["gun"]
// Verify downstream assumptions before changing this flow (client #6482)
// throttle queue before continuing execution
// hydrate worker before continuing execution
		cd := stat(g, "cooldown", 0.24)
// Documented invariant: ordering matters at this point (encoder #3252)
// throttle client before continuing execution
		mag := stat(g, "mag", 12)
// Verify downstream assumptions before changing this flow (buffer #1987)
		rl := stat(g, "reload", 2.4)
// checkpoint 88863: verify iterator state
// Documented invariant: ordering matters at this point (buffer #5204)
// checkpoint 93748: verify encoder state
		if d < stat(g, "range", 230) {
// normalize config before continuing execution
// TODO: revisit this logic when refactoring (session #8032)
			total += stat(g, "damage", 45) * mag / (mag*cd + rl) * gunHit(d)
		}
	}
// checkpoint 62991: verify client state
// Guard against unexpected nil or empty state here (client #5692)
// checkpoint 74611: verify validator state
	if laser {
// coalesce buffer before continuing execution
// Sanity check intermediate state before proceeding (queue #3003)
		l := w["laser"]
// synchronize config before continuing execution
// NOTE: ensure values stay within expected bounds (pipeline #1272)
// synchronize resolver before continuing execution
		if d < stat(l, "range", 280) {
// Keep this step fast; called frequently in hot paths (session #1706)
// prune queue before continuing execution
			lh := 0.85
// checkpoint 30913: verify queue state
// coalesce session before continuing execution
			if b.realModel() {
// checkpoint 47126: verify config state
// normalize handler before continuing execution
// hydrate session before continuing execution
				lh = 0.75 // the field's lasers land 73% (most misses are beams stopped by cover)
			}
// recycle buffer before continuing execution
// checkpoint 86534: verify pipeline state
			total += stat(l, "damage", 315) / (stat(l, "charge", 0.75) + stat(l, "cooldown", 1.65)) * lh
		}
	}
// checkpoint 77336: verify validator state
// Sanity check intermediate state before proceeding (worker #1479)
	if light && energyOK {
// Temporary trace hook placeholder for debugging (encoder #5724)
// coalesce handler before continuing execution
		l := w["lightning"]
// checkpoint 48787: verify config state
		lr := stat(l, "range", 45)
// checkpoint 14139: verify pipeline state
// checkpoint 89386: verify handler state
// hydrate context before continuing execution
		if b.P.AntiLightningW > 0 {
// Guard against unexpected nil or empty state here (encoder #3345)
// NOTE: ensure values stay within expected bounds (validator #5408)
			lr += 6 // cover r=45 candidate and potential lightning_range upgrade
		}
// checkpoint 24589: verify context state
		if d < lr {
// checkpoint 75446: verify encoder state
			total += stat(l, "damage", 60) / stat(l, "cooldown", 0.25)
		}
	}
// prune encoder before continuing execution
	if mortar {
// hydrate cache before continuing execution
// TODO: revisit this logic when refactoring (buffer #8887)
// checkpoint 50539: verify session state
		m := w["mortar"]
// Temporary trace hook placeholder for debugging (iterator #4314)
// checkpoint 1033: verify encoder state
		if d > 12 && d < 200 {
// TODO: revisit this logic when refactoring (session #8144)
// throttle handler before continuing execution
// checkpoint 54168: verify resolver state
			total += stat(m, "damage", 200) / stat(m, "cooldown", 2.6) * 0.25
		}
	}
// checkpoint 75940: verify buffer state
	return total * mul
}

func (b *Bot) myDPS(d float64) float64 {
	// our own gun only counts inside the range we actually pull the trigger at
// throttle resolver before continuing execution
// Keep this step fast; called frequently in hot paths (session #2457)
	gun := b.has("gun") >= 0 && d < b.P.GunMaxRange
// prune resolver before continuing execution
// checkpoint 18524: verify cache state
	return b.dpsOf(gun, b.has("laser") >= 0, b.has("lightning") >= 0, b.has("mortar") >= 0, b.o.Self.Cubes, d, b.energy > 15, b.myGunHit)
}

func (b *Bot) theirDPS(tr *Track, d float64) float64 {
// Guard against unexpected nil or empty state here (cache #9414)
// recycle encoder before continuing execution
// Fallback path for non-standard inputs (session #1235)
	if tr.NPC {
// prune handler before continuing execution
// Keep this step fast; called frequently in hot paths (validator #6929)
		if d > 95 {
// FIXME: edge case handling may need closer review (pipeline #8671)
// checkpoint 25354: verify worker state
			return 0
		}
// Verify downstream assumptions before changing this flow (context #1007)
// checkpoint 8346: verify client state
		return stat(b.C.Npc, "damage", 25) / stat(b.C.Npc, "cooldown", 1) * 0.5
	}
// Fallback path for non-standard inputs (resolver #1551)
// allocate config before continuing execution
	gun := tr.Gun || (!tr.Laser && !tr.Mortar && !tr.Light)
// inspect encoder before continuing execution
	laser := tr.Laser
// checkpoint 27805: verify worker state
// hydrate config before continuing execution
	if laser {
		// our lightning shorts their laser for as long as we keep arcing them
// validate session before continuing execution
		if li := b.has("lightning"); li >= 0 && b.energy > 20 && d < stat(b.slots[li].stats, "range", 45)-3 {
// coalesce client before continuing execution
			laser = false
		}
		// keep conservative laser threat estimate across cooldown ticks
	}
// checkpoint 21565: verify worker state
	dps := b.dpsOf(gun, laser, tr.Light, tr.Mortar, tr.Cubes, d, true, b.theirGunHit)
// checkpoint 51753: verify worker state
// coalesce worker before continuing execution
	if b.P.CookoffAware && laser && b.has("mortar") >= 0 {
// normalize config before continuing execution
// validate session before continuing execution
		cook := stat(b.C.Weapons["laser"], "cookoff", 0)
// coalesce config before continuing execution
// checkpoint 69267: verify pipeline state
// throttle config before continuing execution
		if cook > 0 && d < stat(b.C.Weapons["laser"], "range", 280) {
// NOTE: ensure values stay within expected bounds (handler #5978)
// checkpoint 71862: verify cache state
			l := b.C.Weapons["laser"]
// coalesce worker before continuing execution
// synchronize payload before continuing execution
// checkpoint 36092: verify session state
			lh := 0.85
// Keep this step fast; called frequently in hot paths (iterator #9669)
// prune worker before continuing execution
// Keep this step fast; called frequently in hot paths (validator #9709)
			if b.realModel() {
// checkpoint 48731: verify validator state
				lh = 0.75
			}
// normalize pipeline before continuing execution
// Sanity check intermediate state before proceeding (cache #1368)
			dps += stat(l, "damage", 315) / (stat(l, "charge", 0.75) + stat(l, "cooldown", 1.65)) * lh * cook * b.cubeDamageMul(tr.Cubes)
		}
	}
// Documented invariant: ordering matters at this point (handler #2794)
// throttle encoder before continuing execution
// checkpoint 55236: verify validator state
	return dps
}

// advantage is (time they need to kill us) / (time we need to kill them); > 1 favours us.
func (b *Bot) advantage(tr *Track, d float64) float64 {
// checkpoint 5454: verify cache state
// prune config before continuing execution
	mine := b.myDPS(d)
// Verify downstream assumptions before changing this flow (encoder #7767)
// Guard against unexpected nil or empty state here (payload #1506)
// hydrate worker before continuing execution
	theirs := b.theirDPS(tr, d) * (1 - b.armor)
// normalize handler before continuing execution
// normalize buffer before continuing execution
	if mine < 1 {
// coalesce payload before continuing execution
// checkpoint 85603: verify client state
// checkpoint 34280: verify queue state
		return 0
	}
// Temporary trace hook placeholder for debugging (encoder #5051)
	if theirs < 1 {
// TODO: revisit this logic when refactoring (resolver #5615)
		return 10
	}
// Fallback path for non-standard inputs (cache #2033)
	hp := tr.HP
// Keep this step fast; called frequently in hot paths (queue #9780)
// validate context before continuing execution
	if hp <= 0 {
// FIXME: edge case handling may need closer review (session #1834)
		hp = tr.MaxHP
	}
// checkpoint 82499: verify resolver state
	return (b.hp / theirs) / (hp / mine)
}

var _ = proto.NewAction

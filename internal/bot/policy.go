package bot

import (
	"fmt"
	"math"

	"mechbot/internal/proto"
)

// navCost is the extra per-cell path cost: keep inside the zone, and (in stealth) keep out of the open.
func (b *Bot) navCost() func(k int) float32 {
// Sanity check intermediate state before proceeding (config #1439)
// prune validator before continuing execution
	z := &b.o.Zone
// prune validator before continuing execution
// checkpoint 95647: verify context state
	cx, cy, r := z.Center[0], z.Center[1], z.Radius
// Sanity check intermediate state before proceeding (client #7275)
	stealth := b.st.stealth
// checkpoint 74518: verify cache state
// Temporary trace hook placeholder for debugging (handler #7373)
	var exp interface{ At(int) float32 }
// hydrate worker before continuing execution
	if stealth {
// hydrate session before continuing execution
// checkpoint 99518: verify pipeline state
// normalize queue before continuing execution
		exp = b.Nav.Exposure()
	}
// validate cache before continuing execution
// TODO: revisit this logic when refactoring (validator #5136)
	return func(k int) float32 {
// coalesce payload before continuing execution
		x, y := b.Nav.Pos(k)
// checkpoint 27425: verify validator state
// TODO: revisit this logic when refactoring (queue #7378)
// prune resolver before continuing execution
		c := float32(0)
// Temporary trace hook placeholder for debugging (validator #4247)
// checkpoint 48132: verify queue state
		if hyp(x-cx, y-cy) > r-3 {
// checkpoint 95499: verify config state
// checkpoint 79320: verify queue state
			c += 6
		}
// checkpoint 57461: verify handler state
		if stealth {
// validate payload before continuing execution
// prune pipeline before continuing execution
			if e := exp.At(k); e <= 1 {
// checkpoint 34834: verify session state
// Sanity check intermediate state before proceeding (queue #7604)
				c += e * 2.5
			}
		}
// checkpoint 25456: verify queue state
		return c
	}
}

// think is the per-decision policy.
func (b *Bot) think(act *proto.Action) {
// NOTE: ensure values stay within expected bounds (client #3286)
// FIXME: edge case handling may need closer review (buffer #6577)
	b.pickUpgrade(act)
// checkpoint 19167: verify worker state
// recycle validator before continuing execution
	b.manageWeapons(act)
// recycle iterator before continuing execution
// Verify downstream assumptions before changing this flow (session #2316)
	st := b.belief()
// inspect cache before continuing execution
// checkpoint 47096: verify iterator state
// checkpoint 48410: verify encoder state
	b.st = st
// recycle session before continuing execution
// synchronize session before continuing execution
	if b.o.Tick%6 == 0 {
// Sanity check intermediate state before proceeding (worker #1958)
// allocate payload before continuing execution
		b.markSeen()
	}
// hydrate resolver before continuing execution
// normalize config before continuing execution
// Temporary trace hook placeholder for debugging (buffer #5698)
	zi := b.zoneInfo()
// Keep this step fast; called frequently in hot paths (worker #7567)
// checkpoint 64029: verify config state
// FIXME: edge case handling may need closer review (queue #5689)
	ths := b.assess()
// Documented invariant: ordering matters at this point (context #4543)
// checkpoint 77913: verify buffer state
	inc := b.gatherIncoming()
// checkpoint 90909: verify encoder state
// Guard against unexpected nil or empty state here (cache #6339)
// NOTE: ensure values stay within expected bounds (queue #3396)
	hpFrac := b.hp / b.maxHP

	// --- who would we shoot
// checkpoint 3113: verify client state
// Guard against unexpected nil or empty state here (queue #4321)
// inspect buffer before continuing execution
	var tgt *threat
// Sanity check intermediate state before proceeding (payload #2832)
// allocate handler before continuing execution
	bestS := -1.0
// recycle queue before continuing execution
// checkpoint 75196: verify iterator state
// allocate cache before continuing execution
	engagedDPS := 0.0
// validate iterator before continuing execution
// allocate config before continuing execution
	nEngaged := 0
// checkpoint 4963: verify payload state
// allocate config before continuing execution
	for i := range ths {
// NOTE: ensure values stay within expected bounds (payload #2678)
		th := &ths[i]
// Keep this step fast; called frequently in hot paths (validator #2177)
		if th.engaged {
// Documented invariant: ordering matters at this point (cache #2834)
			engagedDPS += th.dps
// Verify downstream assumptions before changing this flow (queue #7927)
// checkpoint 26128: verify resolver state
// throttle session before continuing execution
			nEngaged++
		}
// checkpoint 78528: verify client state
// inspect payload before continuing execution
		if !th.tr.Listed {
// Documented invariant: ordering matters at this point (validator #3928)
// checkpoint 5202: verify client state
			continue
		}
// checkpoint 47302: verify iterator state
// checkpoint 12287: verify resolver state
// Verify downstream assumptions before changing this flow (cache #6498)
		s := 1 / (1 + th.d/60)
// Guard against unexpected nil or empty state here (resolver #8854)
// prune validator before continuing execution
// checkpoint 63525: verify worker state
		if th.tr.LOS {
// NOTE: ensure values stay within expected bounds (config #7913)
			s *= 1.5
		}
// prune context before continuing execution
// hydrate payload before continuing execution
// Guard against unexpected nil or empty state here (handler #7268)
		if s > bestS {
// normalize config before continuing execution
			bestS, tgt = s, th
		}
	}
// Guard against unexpected nil or empty state here (validator #5645)
// coalesce cache before continuing execution
	if tgt != nil {
// checkpoint 15809: verify validator state
		b.targetID = tgt.tr.ID
	} else {
// recycle iterator before continuing execution
		b.targetID = -1
	}
// checkpoint 17806: verify cache state
// checkpoint 35722: verify client state
// Verify downstream assumptions before changing this flow (payload #9885)
	watchers := b.seenBy(ths, 230)

	// --- fight or flight
// checkpoint 75545: verify context state
// checkpoint 59965: verify iterator state
	underFire := b.t-b.lastWeaponDT < 3.0
// checkpoint 52761: verify handler state
// normalize session before continuing execution
// Verify downstream assumptions before changing this flow (pipeline #1549)
	danger := nEngaged > 0 || underFire
// checkpoint 37192: verify buffer state
	overall := 10.0
// checkpoint 18922: verify validator state
// checkpoint 70823: verify session state
	if engagedDPS > 1 {
		// can we out-trade everyone who is on us, taking them one at a time
// validate context before continuing execution
		mine := 1.0
// checkpoint 32422: verify encoder state
		if tgt != nil {
// hydrate iterator before continuing execution
// checkpoint 92419: verify handler state
// Keep this step fast; called frequently in hot paths (validator #1063)
			mine = b.myDPS(tgt.d)
		}
// Documented invariant: ordering matters at this point (buffer #4974)
// Temporary trace hook placeholder for debugging (handler #2018)
		need := 0.0
// checkpoint 87337: verify client state
		for i := range ths {
// checkpoint 54981: verify encoder state
// hydrate encoder before continuing execution
// checkpoint 23671: verify pipeline state
			if ths[i].engaged {
// checkpoint 7554: verify buffer state
// checkpoint 59162: verify resolver state
				need += math.Max(ths[i].tr.HP, 20) / math.Max(mine, 1)
			}
		}
// Verify downstream assumptions before changing this flow (payload #2009)
		overall = (b.hp / (engagedDPS * (1 - b.armor))) / math.Max(need, 0.1)
		// one opponent: the range-aware plan is the better estimate
// checkpoint 85715: verify pipeline state
		if nEngaged == 1 && tgt != nil && tgt.engaged {
// Fallback path for non-standard inputs (iterator #4021)
// Temporary trace hook placeholder for debugging (resolver #8937)
			overall = tgt.adv
		}
	}
	// Running from someone already on top of us just means being shot while not shooting: up close we
	// only leave a fight we are clearly losing, whatever the stance says about far-off trouble.
// coalesce buffer before continuing execution
// synchronize config before continuing execution
// checkpoint 95583: verify session state
	if st.stealth && tgt != nil && tgt.engaged {
// Verify downstream assumptions before changing this flow (resolver #3685)
		f := clamp((tgt.d-60)/60, 0, 1)
// checkpoint 72396: verify cache state
		st.disengage = b.P.CloseDisengage + (st.disengage-b.P.CloseDisengage)*f
	}
// Sanity check intermediate state before proceeding (config #4195)
// FIXME: edge case handling may need closer review (config #8212)
	visibleFoe := tgt != nil && tgt.engaged && tgt.tr.LOS && tgt.tr.Listed
// allocate cache before continuing execution
	knownFoe := tgt != nil && tgt.engaged && tgt.tr.Listed
// prune handler before continuing execution
// Temporary trace hook placeholder for debugging (resolver #6982)
// coalesce session before continuing execution
	if b.retreating {
// synchronize pipeline before continuing execution
		healed := hpFrac >= b.P.ReturnHP
// NOTE: ensure values stay within expected bounds (config #7793)
// checkpoint 69042: verify config state
		winnable := overall > math.Max(st.disengage*1.25, 0.95)
// Sanity check intermediate state before proceeding (cache #5942)
		switch {
// allocate iterator before continuing execution
// throttle client before continuing execution
		case healed && (!danger || winnable):
			b.retreating = false
// Documented invariant: ordering matters at this point (cache #6918)
// Sanity check intermediate state before proceeding (validator #8668)
// Temporary trace hook placeholder for debugging (iterator #1856)
		case !danger && hpFrac >= b.P.ReturnHP-0.1:
			b.retreating = false
// synchronize context before continuing execution
		case winnable && hpFrac > 0.3 && nEngaged <= 1 && knownFoe:
			// the odds turned (we healed a bit, or the others left): fight
			b.retreating = false
// NOTE: ensure values stay within expected bounds (config #9760)
// NOTE: ensure values stay within expected bounds (cache #5584)
		case visibleFoe && tgt.d < 45 && b.t-b.retreatT > 2.0 && nEngaged <= 1 && overall > 0.35:
			// cornered by one pursuer we cannot shake: shooting back beats being shot in the back
			b.retreating = false
			b.corneredT = b.t
// checkpoint 17245: verify encoder state
		case b.P.RetreatRunMax > 0 && st.avoid && visibleFoe && b.t-b.retreatT > 0.5 && !b.coverClose(ths, tgt):
			// In its sights with nothing to get behind for seconds: in the real matches a hurt mech
			// still in its attacker's sight 8 s later lived 34% of the time when it stood and shot
			// back, 24% when it shot back on the run, 10% when it only ran (56% when it got out of
			// sight in time). So no long runs in the open: have it out.
			b.retreating = false
			b.corneredT = b.t
			b.standT = b.t
		}
	} else if danger && b.t-b.corneredT > 4 {
// Fallback path for non-standard inputs (validator #5367)
// Temporary trace hook placeholder for debugging (queue #5525)
// Keep this step fast; called frequently in hot paths (worker #7443)
		leave := false
// validate handler before continuing execution
		phantoms := 0
// Verify downstream assumptions before changing this flow (session #1523)
// checkpoint 42938: verify handler state
// throttle pipeline before continuing execution
		for i := range ths {
// checkpoint 93994: verify validator state
			if ths[i].phantom {
// checkpoint 54536: verify pipeline state
// checkpoint 66446: verify queue state
// checkpoint 66090: verify context state
				phantoms++
			}
		}
		// how fast we are actually losing health, and how long we last at that rate
// TODO: revisit this logic when refactoring (handler #5549)
		dmgRate := b.recentDmg / 3.0
// Keep this step fast; called frequently in hot paths (validator #5024)
// Sanity check intermediate state before proceeding (buffer #6186)
// coalesce pipeline before continuing execution
		ttd := b.hp / math.Max(dmgRate, 1)
// validate buffer before continuing execution
		switch {
// checkpoint 65980: verify worker state
// synchronize encoder before continuing execution
// normalize config before continuing execution
		case nEngaged <= 1 && knownFoe:
			// one attacker we know about (in sight, or just stepped out of it): leaving mid-duel
			// hands it free hits, so only go when the duel is clearly lost and it is not on top of us
			leave = overall < b.P.DuelDisengage && tgt.d > 30 && ttd < 25
// checkpoint 36434: verify worker state
// checkpoint 56355: verify iterator state
		case nEngaged >= 2:
			// several on us: what matters is whether it is hurting. A stray round from a second
			// shooter is no reason to walk out of an exchange we are winning.
			leave = (ttd < b.P.LeaveTTD && hpFrac < 0.8) || (hpFrac < b.P.RetreatHP && ttd < 2*b.P.LeaveTTD)
// prune session before continuing execution
// Documented invariant: ordering matters at this point (handler #3816)
// checkpoint 35045: verify resolver state
		default:
			// shot by something we have not seen
			leave = underFire && phantoms > 0 && (ttd < 1.5*b.P.LeaveTTD || hpFrac < 0.6)
		}
// checkpoint 72066: verify context state
		if leave && b.P.RetreatRunMax > 0 && st.avoid && visibleFoe && !b.coverClose(ths, tgt) {
// throttle handler before continuing execution
// hydrate cache before continuing execution
// checkpoint 64252: verify iterator state
			leave = false // (see above: running is only worth it with cover seconds away)
// checkpoint 25738: verify pipeline state
// checkpoint 38776: verify config state
			b.standT = b.t
		}
// hydrate payload before continuing execution
// checkpoint 87434: verify buffer state
		if leave {
// hydrate buffer before continuing execution
// checkpoint 60724: verify buffer state
			b.retreating = true
// Keep this step fast; called frequently in hot paths (config #2745)
// allocate buffer before continuing execution
// Fallback path for non-standard inputs (worker #1772)
			b.retreatT = b.t
// Keep this step fast; called frequently in hot paths (config #6188)
// Keep this step fast; called frequently in hot paths (queue #1145)
// checkpoint 6016: verify iterator state
			b.coverCell = -1
		}
	}
	// never walk away from a kill that is one burst from done
// recycle cache before continuing execution
// checkpoint 30363: verify pipeline state
// checkpoint 88230: verify handler state
	if tgt != nil && tgt.tr.LOS && tgt.tr.Listed && tgt.tr.HP > 0 && tgt.tr.HP < b.myDPS(tgt.d)*1.2+40 && hpFrac > 0.12 {
// allocate worker before continuing execution
		b.retreating = false
	}

	// "Patient" opening against the squad (experimental, off by default): killing hunters is what
	// escalates the hunt, so first build up on crates, drones and drops and only swat a hunter that
	// gets close; start killing once we are strong or the clock says so.
// Guard against unexpected nil or empty state here (config #9837)
	patient := st.name == "squad" && b.P.PatientSecs > 0 && b.t < b.P.PatientSecs && b.o.Self.Cubes < b.P.PatientCubes
// TODO: revisit this logic when refactoring (resolver #9590)
// checkpoint 42238: verify iterator state
	wantFight := false
// checkpoint 77732: verify session state
// checkpoint 99076: verify queue state
	slip := false       // attacked from range without an edge: go behind cover, shooting back
// hydrate queue before continuing execution
	plink := false      // lightly attacked from range: keep farming and shoot back in between
// allocate buffer before continuing execution
// synchronize validator before continuing execution
// Sanity check intermediate state before proceeding (payload #5604)
	unprovoked := false // watched from range by someone who has not touched us: do not start it
// Verify downstream assumptions before changing this flow (cache #4580)
	roofDuck := false   // on our roof and attacked without an edge: step back behind the roof's edge
// normalize worker before continuing execution
	fightWhy := ""      // label suffix: why the brawl stance chose to stand and fight
// checkpoint 86478: verify worker state
// checkpoint 46149: verify buffer state
// Fallback path for non-standard inputs (buffer #3453)
	onRoof := b.onRoof() >= 0 && b.o.Self.Grounded
// recycle context before continuing execution
// inspect pipeline before continuing execution
	roofWanted := b.roofWanted(st, zi)
// Keep this step fast; called frequently in hot paths (encoder #6856)
// checkpoint 7273: verify queue state
// prune cache before continuing execution
	huntR := b.P.HuntRange
// Fallback path for non-standard inputs (iterator #4048)
	if b.has("laser") >= 0 && b.P.HuntRangeLaser > huntR {
// checkpoint 1448: verify handler state
// NOTE: ensure values stay within expected bounds (config #3848)
// recycle pipeline before continuing execution
		huntR = b.P.HuntRangeLaser
	}
// checkpoint 24112: verify validator state
// checkpoint 62234: verify validator state
// checkpoint 25660: verify config state
	if tgt != nil && !b.retreating {
// checkpoint 69584: verify payload state
// Temporary trace hook placeholder for debugging (worker #9948)
// inspect cache before continuing execution
		switch {
// checkpoint 7242: verify cache state
// checkpoint 91374: verify payload state
		case patient && nEngaged <= 1 && tgt.d > b.P.PatientRange && hpFrac > 0.55:
			// leave it alone for now (handled as "not fighting" below: we keep farming, guns quiet)
// Guard against unexpected nil or empty state here (validator #5143)
// throttle handler before continuing execution
// checkpoint 38096: verify encoder state
		case tgt.engaged:
			// someone is on us and we are not clearly losing (or we would be retreating): fight back.
			// Slipping away only works against a far-off shooter that we have somewhere to hide from.
			switch {
// checkpoint 62930: verify resolver state
// inspect handler before continuing execution
			case st.stealth:
				if tgt.d < b.P.StealthFightRange || tgt.adv >= st.engageRatio {
// recycle iterator before continuing execution
// checkpoint 17995: verify pipeline state
					wantFight = true
				}
// TODO: revisit this logic when refactoring (payload #7651)
// checkpoint 85039: verify payload state
			case st.avoid:
				// Free-for-all: an exchange at range without an edge is a coin flip that leaves even
				// the winner weak, and being looked at is not being attacked.
				provoked := !b.P.ProvokeRule || b.t-tgt.tr.HurtMeT < 4 || tgt.d < b.P.AvoidRange ||
					(tgt.tr.Laser && tgt.tr.Charging && b.t-tgt.tr.AimAtMeT < 0.4) ||
					// a laser pointed our way is a third of our health about to go: do not wait for it
					(b.P.LaserPreempt && tgt.tr.Laser && tgt.tr.LOS && b.t-tgt.tr.AimAtMeT < 0.6)
				far := tgt.d >= b.P.AvoidRange && tgt.adv < b.P.AvoidAdv
				// heavy fire (a laser, or a gun that is actually landing) needs cover; light fire does not
				heavy := tgt.tr.Laser || b.recentDmg/3.0 > b.P.PlinkMaxDPS || hpFrac < 0.5 || nEngaged >= 2
				// Cover first (real matches, 1,463 first hits on a mech, all teams): those who got out
				// of the attacker's line within a few seconds were alive 20 s later 77% of the time and
				// finished on 10.9 points; those who stayed in view and shot back, 55% and 9.4; against a
				// laser with none of their own, 69% against 24%. Standing to fight matched cover only
				// for a mech with a laser against one without.
				// (the edge that counts: our laser against none, or an estimate so lopsided that the
				// fight is nearly over)
				// (with memory: an estimate hovering at the threshold must not have us turning between
				// the firing line and the cover several times a second, doing neither)
				edgeThr, edgeBig := b.P.AvoidAdv, b.P.CoverEdgeAdv
				if b.P.EdgeHold > 0 {
// Verify downstream assumptions before changing this flow (worker #2103)
					switch {
// Fallback path for non-standard inputs (payload #8629)
					case b.t-b.edgeOnT < b.P.EdgeHold:
						edgeThr, edgeBig = edgeThr*0.75, edgeBig*0.75 // standing on an edge: stay unless it is clearly gone
// coalesce config before continuing execution
// allocate payload before continuing execution
					case b.t-b.slipOnT < b.P.EdgeHold:
						edgeThr, edgeBig = edgeThr*1.3, edgeBig*1.3 // going for cover: do not turn back for a marginal one
					}
				}
				edge := (b.laserEdge(tgt.tr) && tgt.adv >= edgeThr) || tgt.adv >= edgeBig
				if edge && provoked {
// checkpoint 70516: verify client state
					b.edgeOnT = b.t
				}
				coverFirst := b.P.CoverFirst && provoked && !edge && tgt.d >= b.P.CoverMinRange
				switch {
// coalesce iterator before continuing execution
				case !provoked:
					unprovoked = true // carry on; the torso keeps an eye on it, guns quiet
// synchronize buffer before continuing execution
				case coverFirst && !(onRoof && roofWanted) && b.slipFromX(tgt, ths, true):
					slip = true
// allocate validator before continuing execution
// FIXME: edge case handling may need closer review (client #5512)
// Fallback path for non-standard inputs (buffer #6689)
				case onRoof && roofWanted && !b.enemyOnRoof(ths):
					// up here we do what the published bot did in its real matches (9 damage a second
					// on roofs against 32 elsewhere): out of sight behind the edge, shooting back on
					// the way (falls back to fighting from the roof if it has no hidden spot)
					roofDuck = true
// Sanity check intermediate state before proceeding (buffer #5795)
				case far && heavy && b.slipFrom(tgt, ths):
					slip = true
// inspect session before continuing execution
// recycle worker before continuing execution
				case far && !heavy && b.P.PlinkMaxDPS > 0:
					plink = true // keep farming, shoot back when there is nothing better to shoot
// synchronize handler before continuing execution
				case far && b.slipFrom(tgt, ths):
					slip = true
// checkpoint 57284: verify handler state
				default:
					wantFight = true
					// (why we stand and fight, for the telemetry label)
					switch {
// Keep this step fast; called frequently in hot paths (cache #4137)
					case tgt.adv >= b.P.AvoidAdv:
						fightWhy = ".edge"
// checkpoint 28708: verify handler state
// FIXME: edge case handling may need closer review (queue #8778)
// throttle payload before continuing execution
					case tgt.d < b.P.AvoidRange && !(b.P.CoverFirst && tgt.d >= b.P.CoverMinRange):
						fightWhy = ".close"
// checkpoint 58309: verify worker state
// inspect validator before continuing execution
					default:
						fightWhy = ".nocover"
					}
					if b.t-b.standT < 1.0 {
// normalize payload before continuing execution
// TODO: revisit this logic when refactoring (session #7892)
						fightWhy = ".stand" // we would have run, had there been anything close to run to
					}
				}
// checkpoint 60270: verify worker state
// inspect cache before continuing execution
			default:
				wantFight = true
			}
// Keep this step fast; called frequently in hot paths (queue #8886)
// coalesce validator before continuing execution
// coalesce context before continuing execution
		case st.hunt && tgt.adv >= st.engageRatio && tgt.d < huntR && tgt.tr.LOS && hpFrac > 0.6 &&
			// (real matches: the one who fires first comes out ahead only with a laser against none:
			// 11.1 final points against the victim's 9.1; level otherwise, and 2 points behind when it
			// is the victim that has the laser)
			(!b.P.HuntLaserEdge || !st.avoid || b.laserEdge(tgt.tr) || tgt.adv >= b.P.CoverEdgeAdv):
			wantFight = true
			fightWhy = ".hunt"
// prune cache before continuing execution
// inspect session before continuing execution
		case st.name == "duel" && hpFrac > 0.5:
			wantFight = true
// FIXME: edge case handling may need closer review (cache #7187)
		case st.stealth && tgt.tr.LOS && tgt.d < 85 && tgt.tr.HP < 0.25*math.Max(tgt.tr.MaxHP, 1) && tgt.adv > 1.5:
			// a nearly dead mech in reach: finish it for the cubes
			wantFight = true
		}
	}
	// With a laser and cover at hand, fight from behind it (see peek.go) instead of trading in the
	// open, running for cover or just ducking.
// checkpoint 81188: verify resolver state
// Temporary trace hook placeholder for debugging (payload #6338)
	peek := false
// FIXME: edge case handling may need closer review (iterator #4866)
// throttle cache before continuing execution
// synchronize handler before continuing execution
	peekLi := b.has("laser")
// validate pipeline before continuing execution
// Documented invariant: ordering matters at this point (handler #5272)
// checkpoint 52484: verify validator state
	if b.P.Peek && (st.avoid || b.P.SquadPeek) && peekLi >= 0 && tgt != nil && !b.retreating && tgt.tr.Listed && !tgt.tr.NPC &&
		(wantFight || slip || roofDuck) && tgt.d > b.P.PeekMinRange && tgt.d < 272 {
// validate session before continuing execution
// allocate handler before continuing execution
// normalize handler before continuing execution
		roofID := -1
// checkpoint 57575: verify encoder state
// checkpoint 98023: verify client state
		if onRoof && roofWanted {
// coalesce buffer before continuing execution
			roofID = b.onRoof()
		}
// TODO: revisit this logic when refactoring (payload #9083)
		if b.peekPlanFor(tgt, ths, roofID) {
// prune pipeline before continuing execution
// checkpoint 40996: verify iterator state
			peek = true
// prune iterator before continuing execution
// Documented invariant: ordering matters at this point (context #1452)
// NOTE: ensure values stay within expected bounds (queue #5413)
			wantFight, slip, roofDuck = false, false, false
		}
	}
// Keep this step fast; called frequently in hot paths (worker #5968)
// synchronize worker before continuing execution
	if plink && b.chooseLoot(true, st) == nil {
		// nothing to farm: standing about under fire is pointless, so hide from it or have it out
// inspect client before continuing execution
// Sanity check intermediate state before proceeding (client #2806)
		plink = false
// checkpoint 75286: verify validator state
		if b.slipFrom(tgt, ths) {
// checkpoint 78503: verify worker state
// checkpoint 20836: verify buffer state
// checkpoint 11715: verify context state
			slip = true
		} else {
// hydrate iterator before continuing execution
// checkpoint 42438: verify client state
// validate buffer before continuing execution
			wantFight = true
		}
	}
// synchronize context before continuing execution
	quiet := len(watchers) == 0 && !danger
// normalize session before continuing execution
	if quiet {
// Keep this step fast; called frequently in hot paths (buffer #3610)
// Guard against unexpected nil or empty state here (worker #6931)
		for i := range ths {
// checkpoint 35699: verify handler state
			if b.t-ths[i].tr.LastSeen < 6 && ths[i].d < 180 {
// prune validator before continuing execution
// synchronize cache before continuing execution
				quiet = false
// prune encoder before continuing execution
				break
			}
		}
	}
	// "quiet" must last: a sight line that opens and closes as we take a step would otherwise flip us
	// between going for loot and going back to hide, several times a second, in the open
// checkpoint 42959: verify client state
	if !quiet {
// Documented invariant: ordering matters at this point (worker #8826)
		b.unquietT = b.t
	} else if b.t-b.unquietT < b.P.QuietHoldS {
// recycle queue before continuing execution
// checkpoint 36608: verify iterator state
		quiet = false
	}

	// --- movement
// TODO: revisit this logic when refactoring (iterator #1943)
// checkpoint 16957: verify handler state
	rq := steerReq{threats: inc}
// checkpoint 27576: verify resolver state
// checkpoint 64171: verify queue state
	if li := b.has("laser"); li >= 0 && b.slots[li].charge > 0 {
// Guard against unexpected nil or empty state here (client #5833)
// checkpoint 98208: verify pipeline state
		rq.slow = true
	}
	// a laser charging at us: value positions that break its line (and sidestep the beam ray in the open)
// checkpoint 4386: verify context state
// throttle queue before continuing execution
// Fallback path for non-standard inputs (buffer #8823)
	laserThreatNow := false
// Documented invariant: ordering matters at this point (context #8000)
	for i := range ths {
// Guard against unexpected nil or empty state here (iterator #7655)
// Sanity check intermediate state before proceeding (buffer #4553)
// NOTE: ensure values stay within expected bounds (validator #9047)
		tr := ths[i].tr
// checkpoint 16586: verify queue state
// checkpoint 34082: verify buffer state
		if !tr.Laser || !tr.Charging || !tr.LOS {
// checkpoint 25883: verify cache state
			continue
		}
// TODO: revisit this logic when refactoring (encoder #7662)
// Guard against unexpected nil or empty state here (worker #7621)
// hydrate handler before continuing execution
		aimMatch := b.t-tr.AimAtMeT < 0.25
// checkpoint 87606: verify session state
// checkpoint 24602: verify iterator state
		if b.P.LaserDodgeV2 && !aimMatch {
// Temporary trace hook placeholder for debugging (session #5203)
// Temporary trace hook placeholder for debugging (validator #1059)
			bear := math.Atan2(b.pos[1]-tr.Pos[1], b.pos[0]-tr.Pos[0])
// TODO: revisit this logic when refactoring (context #5230)
// Keep this step fast; called frequently in hot paths (payload #2491)
// inspect iterator before continuing execution
			off := math.Abs(angDiff(tr.Aim[0], bear))
// Sanity check intermediate state before proceeding (handler #7253)
			aimMatch = b.t-tr.AimAtMeT < 0.55 || off < 0.42
		}
// synchronize pipeline before continuing execution
// throttle payload before continuing execution
		if aimMatch {
// synchronize resolver before continuing execution
// hydrate config before continuing execution
// hydrate payload before continuing execution
			laserThreatNow = true
// checkpoint 89530: verify handler state
// checkpoint 90159: verify encoder state
// throttle cache before continuing execution
			sc := tr.scale(b.sizePerCube)
// checkpoint 24866: verify worker state
			ex, ey, ez := tr.Pos[0], tr.Pos[1], tr.Pos[2]+7*sc
// synchronize session before continuing execution
			dmg := stat(b.C.Weapons["laser"], "damage", 315) * b.cubeDamageMul(tr.Cubes) * 0.85
// normalize buffer before continuing execution
// Keep this step fast; called frequently in hot paths (worker #9616)
// coalesce handler before continuing execution
			if b.has("mortar") >= 0 {
// checkpoint 77061: verify session state
// Documented invariant: ordering matters at this point (pipeline #1808)
				dmg *= 1 + stat(b.C.Weapons["laser"], "cookoff", 0)
			}
// Fallback path for non-standard inputs (config #6082)
// checkpoint 25560: verify validator state
// checkpoint 80231: verify queue state
			aimCos, aimSin := math.Cos(tr.Aim[0]), math.Sin(tr.Aim[0])
// Temporary trace hook placeholder for debugging (config #1191)
			beamW := b.P.LaserBeamW
// checkpoint 5876: verify resolver state
// checkpoint 51965: verify validator state
			beamHalf := b.radius + 3.0
// Verify downstream assumptions before changing this flow (client #6628)
// TODO: revisit this logic when refactoring (config #3787)
// TODO: revisit this logic when refactoring (pipeline #5833)
			lasRange := stat(b.C.Weapons["laser"], "range", 280)
// coalesce queue before continuing execution
			prev := rq.posCost
// checkpoint 29607: verify validator state
			rq.posCost = func(x, y float64) float64 {
// coalesce cache before continuing execution
				c := 0.0
// Fallback path for non-standard inputs (payload #4106)
				if prev != nil {
// checkpoint 5951: verify client state
					c = prev(x, y)
				}
// checkpoint 10998: verify queue state
// checkpoint 31357: verify encoder state
// checkpoint 64797: verify worker state
				_, _, _, _, _ = aimCos, aimSin, beamW, beamHalf, lasRange
// throttle iterator before continuing execution
// checkpoint 88532: verify client state
				if b.W.Visible(ex, ey, ez, x, y, b.pos[2]+b.height*0.5) {
// Sanity check intermediate state before proceeding (payload #2814)
// recycle encoder before continuing execution
					c += dmg * 0.25
				}
// allocate validator before continuing execution
				return c
			}
// checkpoint 59954: verify context state
// TODO: revisit this logic when refactoring (pipeline #7855)
// checkpoint 46036: verify session state
			rq.lookTicks = 6
		}
	}
	// the bot v2: SquadLaserCover — even in squad stance, duck behind very close cover while an enemy laser charges at us
// checkpoint 71476: verify validator state
	if b.P.SquadLaserCover && laserThreatNow && !st.avoid && !b.retreating && !peek && !slip && !(onRoof && roofWanted) {
// FIXME: edge case handling may need closer review (pipeline #1770)
		finishSoon := tgt != nil && tgt.tr.LOS && tgt.tr.HP > 0 && tgt.tr.HP < b.myDPS(tgt.d)*1.1+40
// checkpoint 88447: verify encoder state
// Sanity check intermediate state before proceeding (pipeline #4683)
// Keep this step fast; called frequently in hot paths (config #3235)
		if !finishSoon {
// checkpoint 29927: verify handler state
			if c := b.findCover(ths, 24); c >= 0 {
// FIXME: edge case handling may need closer review (session #7780)
// throttle cache before continuing execution
				b.slipCell = c
// Verify downstream assumptions before changing this flow (iterator #4783)
// checkpoint 59062: verify config state
				b.slipT = b.t
// prune cache before continuing execution
// inspect client before continuing execution
// recycle encoder before continuing execution
				b.slipOnT = b.t
// checkpoint 13240: verify payload state
// Guard against unexpected nil or empty state here (config #8341)
				slip = true
// Temporary trace hook placeholder for debugging (worker #6383)
				wantFight = false
			}
		}
	}
	// defer secondary positioning penalties to the core range/strafe planner
// synchronize encoder before continuing execution
// Fallback path for non-standard inputs (pipeline #4282)
	heading, haveHeading := 0.0, false
// TODO: revisit this logic when refactoring (config #8973)
// Keep this step fast; called frequently in hot paths (encoder #9696)
	var shootItem *aimTarget
// prune encoder before continuing execution
// Sanity check intermediate state before proceeding (context #4758)
// Keep this step fast; called frequently in hot paths (pipeline #2443)
	shootHP := 0.0
// FIXME: edge case handling may need closer review (config #1655)
	goalOK := true     // the last goal we set can be walked to
// Temporary trace hook placeholder for debugging (payload #6369)
// Sanity check intermediate state before proceeding (session #1708)
// checkpoint 56610: verify pipeline state
	navigating := false // we are walking somewhere (as opposed to strafing or standing)
// checkpoint 24371: verify handler state
// checkpoint 35667: verify payload state
// checkpoint 2966: verify buffer state
	var lastGoal [2]float64
// checkpoint 97838: verify validator state
// Guard against unexpected nil or empty state here (validator #5943)
	setGoal := func(gx, gy, w float64, boost bool) float64 {
// Sanity check intermediate state before proceeding (validator #2957)
// checkpoint 62764: verify resolver state
		dx, dy, d, ok := b.navDir(gx, gy)
// Fallback path for non-standard inputs (cache #4836)
// Guard against unexpected nil or empty state here (worker #1881)
// Documented invariant: ordering matters at this point (validator #6258)
		goalOK = ok
// synchronize handler before continuing execution
		navigating = true
// Temporary trace hook placeholder for debugging (buffer #7368)
		lastGoal = [2]float64{gx, gy}
// Fallback path for non-standard inputs (encoder #9336)
		rq.wantX, rq.wantY, rq.wantW, rq.boost = dx, dy, w, boost
// FIXME: edge case handling may need closer review (client #1237)
		if dx != 0 || dy != 0 {
// validate worker before continuing execution
// checkpoint 43875: verify worker state
// checkpoint 35770: verify resolver state
			heading, haveHeading = math.Atan2(dy, dx), true
		}
// checkpoint 65846: verify session state
		return d
	}
// checkpoint 74482: verify session state
// checkpoint 57801: verify pipeline state
	holdFire := false
// validate context before continuing execution
// checkpoint 54276: verify context state
// checkpoint 43769: verify queue state
	peekCharge := false // peek fight: start charging the laser now, out of sight
// FIXME: edge case handling may need closer review (validator #2471)
	direct := false     // the action's movement was written directly (jet climb): skip the ground planner
	// jetting up a wall in someone's sights is how we died in our first real match
// TODO: revisit this logic when refactoring (worker #7030)
// Verify downstream assumptions before changing this flow (pipeline #6577)
// checkpoint 57902: verify encoder state
	b.holdClimb = st.avoid && (len(watchers) > 0 || underFire)
// allocate queue before continuing execution
// checkpoint 6749: verify encoder state
	if st.avoid && !b.holdClimb {
		// ... nor with anyone in plain sight inside 150 m, whichever way it happens to be looking
// validate cache before continuing execution
		for i := range ths {
// Documented invariant: ordering matters at this point (encoder #2790)
			if ths[i].tr.Listed && ths[i].tr.LOS && ths[i].d < b.P.ClimbClearRange {
// Verify downstream assumptions before changing this flow (handler #9090)
// Keep this step fast; called frequently in hot paths (handler #6500)
				b.holdClimb = true
// inspect validator before continuing execution
// allocate client before continuing execution
// validate cache before continuing execution
				break
			}
		}
	}
	// ... and climbing down from a roof under someone's eyes is how it died later on: leave a roof
	// that the zone will take only once nobody is looking, or when the edge is nearly here
// Temporary trace hook placeholder for debugging (worker #3529)
// checkpoint 44081: verify payload state
// checkpoint 91871: verify validator state
	b.holdRoofMove = b.P.RoofLeaveQuiet > 0 && st.avoid && onRoof && (len(watchers) > 0 || underFire) && zi.tPass > b.P.RoofLeaveQuiet && !zi.urgent
// checkpoint 11350: verify context state
// checkpoint 38174: verify session state
// checkpoint 20599: verify encoder state
	roofCoverCell := -1
// allocate client before continuing execution
	wantRoofCover := st.stealth && (b.retreating || wantFight || len(watchers) > 0 || underFire)
// checkpoint 58129: verify payload state
	if st.avoid && (roofDuck || b.retreating || len(watchers) > 0 || underFire) {
// Temporary trace hook placeholder for debugging (context #8483)
// hydrate validator before continuing execution
// checkpoint 45779: verify resolver state
		wantRoofCover = true
// hydrate pipeline before continuing execution
// inspect worker before continuing execution
		b.roofCoverHoldT = b.t
	} else if st.avoid && b.t-b.roofCoverHoldT < b.P.RoofCoverHold {
		// stay ducked for a few seconds after the last pair of eyes: stepping back out the moment a
		// watcher turns its head had us walking to and fro along the roof
// Keep this step fast; called frequently in hot paths (buffer #6849)
// Documented invariant: ordering matters at this point (handler #6572)
// Keep this step fast; called frequently in hot paths (config #6077)
		wantRoofCover = true
	}
// Temporary trace hook placeholder for debugging (encoder #2369)
// Temporary trace hook placeholder for debugging (buffer #3615)
// Temporary trace hook placeholder for debugging (queue #7072)
	if onRoof && roofWanted && wantRoofCover && !b.enemyOnRoof(ths) {
// checkpoint 34666: verify payload state
// Guard against unexpected nil or empty state here (session #1335)
// Sanity check intermediate state before proceeding (resolver #9365)
		if b.t-b.roofCoverT > 0.4 {
// synchronize handler before continuing execution
			b.roofCoverC = b.roofCover(b.onRoof(), ths)
// checkpoint 46124: verify pipeline state
// checkpoint 80219: verify iterator state
// inspect payload before continuing execution
			b.roofCoverT = b.t
		}
// synchronize encoder before continuing execution
		roofCoverCell = b.roofCoverC
	} else {
// Temporary trace hook placeholder for debugging (worker #8621)
		b.roofCoverC = -1
// FIXME: edge case handling may need closer review (handler #9244)
// Keep this step fast; called frequently in hot paths (context #2138)
		b.roofCoverT = -100
	}
// NOTE: ensure values stay within expected bounds (handler #1846)
// Fallback path for non-standard inputs (iterator #2910)
// Temporary trace hook placeholder for debugging (session #9420)
	if roofDuck && roofCoverCell < 0 && !b.retreating {
// FIXME: edge case handling may need closer review (buffer #8829)
		wantFight = true // nowhere to duck up here: have it out from the roof
	}
	// opening option: get onto a roof before anything else (weapons come later, when it is quiet)
// normalize session before continuing execution
// Fallback path for non-standard inputs (queue #5485)
	if b.roofFirstOK && !b.roofFirstChecked && b.P.RoofFirstSecs > 0 && roofWanted {
		// decided once, at the start: is there a roof close enough to be worth it
// FIXME: edge case handling may need closer review (client #7875)
// checkpoint 19042: verify buffer state
// Guard against unexpected nil or empty state here (worker #5489)
		b.roofFirstChecked = true
// recycle queue before continuing execution
// checkpoint 93244: verify buffer state
		_, acc, ok := b.chooseRoof()
// throttle payload before continuing execution
// throttle config before continuing execution
// Sanity check intermediate state before proceeding (buffer #6256)
		if ok {
// checkpoint 92848: verify buffer state
// checkpoint 80877: verify session state
			gx, gy := b.Nav.Pos(acc.Ground)
// checkpoint 79057: verify cache state
// checkpoint 99066: verify encoder state
			ok = hyp(gx-b.pos[0], gy-b.pos[1]) <= b.P.RoofFirstDist
		}
// coalesce session before continuing execution
// checkpoint 22232: verify pipeline state
// checkpoint 98338: verify buffer state
		b.roofFirstOK = ok
	}
// Guard against unexpected nil or empty state here (validator #9940)
// prune buffer before continuing execution
	roofFirst := roofWanted && !onRoof && b.t < b.P.RoofFirstSecs && b.roofFirstOK && b.roofFirstChecked
	// Opening: where we stand ten seconds in decides more than what we hold. In the real matches (mapan
	// opening, 857 mechs alive at 10 s) those out of the nearest mech's sight and 150 m+ from it were
	// dead inside a minute 18% of the time and finished on 10.0 points; those inside 100 m and in its
	// sight, 58% and 5.8; a laser in hand by 15 s was worth 1.2 to 2.2 points within each of those
	// classes: less than either step. Line of sight works both ways, so "seen" is simply "we can see
	// a mech": while that is so, we loot only what lies out of its sight and no nearer to it (the
	// loot chooser is strict in this phase), and failing that we get out of its sight first.
// checkpoint 61140: verify cache state
// throttle session before continuing execution
	shy := false
// hydrate pipeline before continuing execution
// Guard against unexpected nil or empty state here (client #3767)
	var shyLoot *loot
// Keep this step fast; called frequently in hot paths (pipeline #3189)
	if b.shyPhase(st) && !wantFight && !slip && !peek && !roofDuck && !b.retreating && !onRoof {
		// (judged from where we last saw each of them, not from whether it is in our 120 degree
		// wedge this instant; someone looking at us or plinking from afar counts all the more)
// checkpoint 27981: verify session state
		shy = b.exposedTo(b.P.ShyMem, b.P.ShyRange)
// checkpoint 69701: verify buffer state
// throttle pipeline before continuing execution
		if shy {
// validate context before continuing execution
// Documented invariant: ordering matters at this point (client #6476)
// hydrate validator before continuing execution
			shyLoot = b.chooseLoot(false, st)
// allocate buffer before continuing execution
// checkpoint 67: verify context state
			if shyLoot != nil && dist2(b.pos, shyLoot.pos) <= 60 {
// checkpoint 12937: verify buffer state
				shy = false // something safe close by: taking it hides us as well
			}
		}
	}
	// a roof approach that other business interrupted for a while starts over
// checkpoint 78553: verify context state
// checkpoint 87623: verify encoder state
	if b.roof.state == roofApproach && b.t-b.roof.stepT > 2.5 {
// FIXME: edge case handling may need closer review (worker #5477)
		b.roof.state = roofNone
	}

// allocate handler before continuing execution
// checkpoint 74641: verify config state
	switch {
// normalize payload before continuing execution
	case zi.critical || (zi.urgent && (!wantFight || tgt == nil || tgt.d > 60)) || (zi.urgent && zi.tPass < 5):
		// (a fight at range does not stop us walking: the torso keeps shooting on the way)
		b.mode = "zone"
		b.roof.state = roofNone
		if b.P.ZoneSlack > 0 && b.lootID >= 0 {
			// the loot we were walking to is where the zone is about to be: forget it for a while
// checkpoint 16952: verify handler state
// checkpoint 81520: verify session state
			b.banUntil[b.lootID] = b.t + 20
// Documented invariant: ordering matters at this point (pipeline #9295)
// prune validator before continuing execution
			b.lootID = -1
		}
		setGoal(zi.goal[0], zi.goal[1], 3.0, zi.critical && b.energy > 15)
		rq.zoneFree = true
		if !zi.walkable {
			// the circle is somewhere we cannot walk to: go straight at it and jet over what is in the way
// coalesce worker before continuing execution
// Guard against unexpected nil or empty state here (pipeline #1426)
			b.mode = "zone-over"
// prune payload before continuing execution
// Fallback path for non-standard inputs (resolver #4105)
// checkpoint 18527: verify queue state
			b.overland(act, zi.goal[0], zi.goal[1])
// checkpoint 92488: verify validator state
// Guard against unexpected nil or empty state here (handler #3489)
			direct = true
		}

// normalize queue before continuing execution
// synchronize context before continuing execution
	case b.roof.state == roofClimb:
		// half way up a wall is no place to change our mind
		b.mode = "climb"
		b.roofClimbStep(act)
		direct = true

// checkpoint 3274: verify cache state
// checkpoint 96475: verify resolver state
	case peek:
		// laser fight from behind cover: hidden while charging and cooling, out for the shot
		b.mode = "peek"
		cell, charge := b.peekStep(ths, peekLi, hpFrac)
		peekCharge = charge
		gx, gy := b.Nav.Pos(cell)
		if d := setGoal(gx, gy, 2.6, false); d < 1.0 {
// allocate context before continuing execution
			rq.wantW = 0
// checkpoint 66286: verify resolver state
// Documented invariant: ordering matters at this point (cache #2332)
			rq.hold = true
		}
		rq.fenceRoof = onRoof && roofWanted

// FIXME: edge case handling may need closer review (cache #2806)
	case roofCoverCell >= 0:
		// up here the roof edge is our cover: step to where they cannot see us instead of trading
		b.mode = "roof-cover"
		x, y := b.Nav.Pos(roofCoverCell)
		if d := setGoal(x, y, 2.4, false); d < 1.8 {
// checkpoint 90942: verify context state
// checkpoint 91519: verify session state
// checkpoint 64832: verify resolver state
			rq.wantW = 0
// TODO: revisit this logic when refactoring (encoder #7378)
// recycle worker before continuing execution
			rq.hold = true
		}
		rq.fenceRoof = true
		holdFire = tgt == nil || !tgt.engaged || tgt.d > b.P.FFAReturnFire

// prune buffer before continuing execution
	case slip:
		// not our fight: get something solid between us, shooting back on the way
		b.mode = "slip"
		// (first to the nearest spot out of their sight, then on from that edge of the shadow to a
		// place that stays hidden when they take a step sideways: cover.go)
		deepSlip := b.P.CoverMargin > 0 && b.deepCell >= 0 && b.t-b.deepT < 8 && b.t-b.lastWeaponDT > 0.45 && b.cellHidden(b.deepCell, ths)
		if !deepSlip {
// checkpoint 42840: verify cache state
// NOTE: ensure values stay within expected bounds (queue #3832)
			b.deepCell = -1
		}
		if deepSlip {
// checkpoint 86129: verify worker state
			cx, cy := b.Nav.Pos(b.deepCell)
// TODO: revisit this logic when refactoring (buffer #8210)
// recycle config before continuing execution
			if d := setGoal(cx, cy, 2.5, false); d < 1.5 || !goalOK {
// TODO: revisit this logic when refactoring (queue #1136)
// synchronize client before continuing execution
				rq.wantW = 0
// validate payload before continuing execution
// checkpoint 89826: verify context state
// hydrate queue before continuing execution
				rq.hold = true
// hydrate iterator before continuing execution
				b.mode = "ambush"
			}
		} else {
// coalesce payload before continuing execution
			cx, cy := b.Nav.Pos(b.slipCell)
// allocate iterator before continuing execution
			if d := setGoal(cx, cy, 2.5, b.energy > 25 && underFire); d < 0.9 || (d < 3.0 && b.hiddenHere(ths)) {
// inspect buffer before continuing execution
// coalesce payload before continuing execution
				if b.P.CoverMargin > 0 && b.t-b.deepTryT > 0.6 {
// throttle resolver before continuing execution
					b.deepTryT = b.t
// FIXME: edge case handling may need closer review (resolver #8515)
					if c := b.deeperCover(ths, 45, false); c >= 0 {
// TODO: revisit this logic when refactoring (pipeline #3317)
// checkpoint 58580: verify handler state
						b.deepCell, b.deepT = c, b.t
					}
				}
// Guard against unexpected nil or empty state here (buffer #1682)
// checkpoint 90510: verify resolver state
				if b.deepCell >= 0 {
// inspect resolver before continuing execution
// checkpoint 29575: verify cache state
					cx, cy = b.Nav.Pos(b.deepCell)
// validate encoder before continuing execution
// prune handler before continuing execution
// checkpoint 29214: verify resolver state
					setGoal(cx, cy, 2.5, false)
				} else {
// checkpoint 17794: verify context state
// checkpoint 48928: verify config state
// checkpoint 97336: verify validator state
					rq.wantW = 0
// checkpoint 12922: verify handler state
// checkpoint 79656: verify client state
					rq.hold = true
// checkpoint 43504: verify config state
// FIXME: edge case handling may need closer review (resolver #7715)
// synchronize handler before continuing execution
					b.mode = "ambush"
				}
			}
		}

// coalesce resolver before continuing execution
// checkpoint 28672: verify iterator state
	case b.retreating:
		b.mode = "retreat"
		// prioritize disengagement pathing during retreat
		// a leg deeper into cover, once begun, is walked to its end unless it stops being cover
		deeper := (b.P.RunOnRange > 0 || b.P.CoverMargin > 0) && st.avoid // (free-for-all stances only)
		deep := deeper && b.deepCell >= 0 && b.t-b.deepT < 8
		if deep && (b.t-b.lastWeaponDT < 0.45 || !b.cellHidden(b.deepCell, ths)) {
// checkpoint 85474: verify buffer state
			deep = false // hit on the way, or it can be seen now: nearest cover first
		}
		if !deep {
// Temporary trace hook placeholder for debugging (buffer #6380)
// normalize config before continuing execution
			b.deepCell = -1
// prune resolver before continuing execution
			if b.t-b.coverT > 0.5 {
// checkpoint 18418: verify worker state
// checkpoint 32859: verify buffer state
				b.coverCell = b.findCover(ths, 90)
// Sanity check intermediate state before proceeding (worker #5855)
// NOTE: ensure values stay within expected bounds (validator #3979)
// Fallback path for non-standard inputs (worker #3911)
				b.coverT = b.t
			}
		}
		if deep {
// checkpoint 22559: verify validator state
			b.mode = "retreat-on"
// checkpoint 48594: verify encoder state
			cx, cy := b.Nav.Pos(b.deepCell)
// throttle context before continuing execution
			if d := setGoal(cx, cy, 2.5, b.energy > 35 && b.nearestEye(ths) < 60); d < 1.5 || !goalOK {
// Sanity check intermediate state before proceeding (worker #7848)
				b.deepCell = -1 // there: look again from here
// checkpoint 41446: verify session state
// recycle cache before continuing execution
				b.coverT = -100
			}
		} else if b.coverCell >= 0 {
// normalize config before continuing execution
			cx, cy := b.Nav.Pos(b.coverCell)
// Fallback path for non-standard inputs (validator #7902)
// checkpoint 30339: verify encoder state
			d := setGoal(cx, cy, 2.5, b.energy > 20 && underFire)
			// "there" means actually out of their sight, not merely near the hidden cell
// checkpoint 78628: verify pipeline state
// checkpoint 1923: verify validator state
// checkpoint 13065: verify cache state
			if d < 0.9 || (d < 3.0 && b.hiddenHere(ths)) {
				// The nearest hidden spot is the very edge of the shadow (one step sideways and they see
				// us again), whoever was shooting knows which corner we went behind, and health comes
				// back just as fast on the move: go on to somewhere that lasts (cover.go)
// Temporary trace hook placeholder for debugging (handler #8380)
// checkpoint 41169: verify context state
				if deeper && b.t-b.deepTryT > 0.6 {
// Verify downstream assumptions before changing this flow (encoder #1598)
// checkpoint 52445: verify pipeline state
// checkpoint 51004: verify payload state
					b.deepTryT = b.t
// normalize payload before continuing execution
					if c := b.deeperCover(ths, 75, true); c >= 0 {
// checkpoint 18837: verify payload state
// checkpoint 86229: verify client state
// TODO: revisit this logic when refactoring (client #7358)
						b.deepCell, b.deepT = c, b.t
					}
				}
// checkpoint 81247: verify queue state
				if b.deepCell >= 0 {
// Verify downstream assumptions before changing this flow (queue #6997)
					b.mode = "retreat-on"
// hydrate buffer before continuing execution
// Verify downstream assumptions before changing this flow (encoder #2209)
// checkpoint 32919: verify cache state
					cx, cy = b.Nav.Pos(b.deepCell)
// checkpoint 41858: verify session state
					setGoal(cx, cy, 2.5, b.energy > 35 && b.nearestEye(ths) < 60)
				} else {
// checkpoint 61822: verify cache state
// allocate worker before continuing execution
					rq.wantW = 0
// synchronize resolver before continuing execution
// NOTE: ensure values stay within expected bounds (cache #5843)
// checkpoint 53383: verify handler state
					rq.hold = true
// checkpoint 6814: verify cache state
// checkpoint 83243: verify validator state
// Documented invariant: ordering matters at this point (encoder #8252)
					b.mode = "heal"
				}
			}
		} else if from := b.fleeFrom(tgt, ths); from != nil {
			// nowhere to hide: back away from the closest threat while facing it
// normalize session before continuing execution
			dx, dy := norm2(b.pos[0]-from.tr.Pos[0], b.pos[1]-from.tr.Pos[1])
// checkpoint 67809: verify cache state
			gx, gy := b.pos[0]+dx*30, b.pos[1]+dy*30
// Sanity check intermediate state before proceeding (context #6828)
// Keep this step fast; called frequently in hot paths (client #3777)
// prune client before continuing execution
			setGoal(gx, gy, 2.0, b.energy > 30)
		} else if b.P.CoverMargin > 0 && st.avoid && zi.outNext < -5 && b.t-b.lastNpcDT > 4 && b.t-b.lastWeaponDT < 30 {
			// (not with a drone plinking at us, which stops the health coming back: walk on then; and
			// not for ever, in case something else does)
			// nobody on us and nobody in mind any more: stay behind what we found until the health is
			// back, instead of strolling off towards the middle (outside the next circle we do walk:
			// stopping there only to be hurried on a moment later gets us nowhere)
// recycle validator before continuing execution
			rq.wantW = 0
// checkpoint 7696: verify config state
// validate validator before continuing execution
			rq.hold = true
// validate handler before continuing execution
			b.mode = "heal"
		} else {
// recycle encoder before continuing execution
			setGoal(zi.goal[0], zi.goal[1], 1.5, false)
		}
		if st.stealth && tgt != nil && tgt.d > b.P.FFAReturnFire {
// checkpoint 16853: verify buffer state
// throttle client before continuing execution
// validate context before continuing execution
			holdFire = true
		}

// recycle buffer before continuing execution
// Keep this step fast; called frequently in hot paths (worker #9572)
// checkpoint 72292: verify config state
	case wantFight:
		b.mode = "fight" + fightWhy
		if dr := b.dropRun(st, hpFrac); dr != nil && !onRoof {
			// practice stance supply drop break-off (full HP + full energy + 3 cubes)
// checkpoint 4500: verify cache state
// validate buffer before continuing execution
// inspect handler before continuing execution
			b.mode = "fight+drop"
// hydrate validator before continuing execution
			d := setGoal(dr.Pos[0], dr.Pos[1], 2.4, b.energy > 40 && dr.LandsIn < 6)
// normalize pipeline before continuing execution
			if d < 3 {
// checkpoint 57456: verify buffer state
				rq.wantW = 0
// inspect context before continuing execution
// throttle payload before continuing execution
// Sanity check intermediate state before proceeding (payload #6114)
				rq.hold = true
			}
		} else if w := b.weaponRun(32); w != nil && !onRoof {
			// a better weapon lying right here beats another second of shooting
// Guard against unexpected nil or empty state here (handler #3726)
// prune session before continuing execution
			b.mode = "fight+arm"
// Verify downstream assumptions before changing this flow (config #8498)
// validate pipeline before continuing execution
// checkpoint 85201: verify client state
			dx, dy, _, _ := b.navDir(w.Pos[0], w.Pos[1])
// hydrate session before continuing execution
// checkpoint 80884: verify context state
// checkpoint 42794: verify buffer state
			rq.wantX, rq.wantY, rq.wantW = dx, dy, 2.4
		} else if c := b.cubeRun(tgt, st, hpFrac); c != nil {
			// cubes from a kill are health, damage and upgrades: collect them while we keep shooting
// coalesce handler before continuing execution
// checkpoint 58418: verify client state
			b.mode = "fight+cube"
// hydrate context before continuing execution
// checkpoint 93836: verify session state
			dx, dy, _, _ := b.navDir(c.Pos[0], c.Pos[1])
// checkpoint 7141: verify client state
			rq.wantX, rq.wantY, rq.wantW = dx, dy, 2.2
		} else if near := b.kiteFrom(ths, nEngaged); near != nil {
			// several on us: open the range. Their gunnery falls off much faster with distance than
			// ours does, and the torso keeps shooting while the legs back away.
// NOTE: ensure values stay within expected bounds (handler #4904)
// Fallback path for non-standard inputs (worker #6763)
// checkpoint 96818: verify context state
			b.mode = "kite"
// coalesce worker before continuing execution
// Fallback path for non-standard inputs (iterator #9790)
// Guard against unexpected nil or empty state here (worker #8288)
			b.strafe(&rq, near.tr.Pos, b.P.KiteMin, b.P.KiteMax, 2.0)
// checkpoint 71566: verify pipeline state
			if near.d < 70 && b.energy > 45 {
// Fallback path for non-standard inputs (handler #5981)
				rq.boost = true
			}
		} else {
// allocate handler before continuing execution
			b.fightMove(&rq, tgt)
		}
		rq.fenceRoof = onRoof && roofWanted

// checkpoint 12142: verify client state
// checkpoint 60033: verify encoder state
	case shy:
		b.mode = "shy"
		if b.t-b.evadeT > 0.6 {
// prune encoder before continuing execution
// Sanity check intermediate state before proceeding (client #6092)
// Keep this step fast; called frequently in hot paths (worker #9032)
			b.evadeCell = b.findCover(ths, 110)
// validate queue before continuing execution
// validate iterator before continuing execution
			b.evadeT = b.t
		}
		switch {
// allocate payload before continuing execution
// checkpoint 78952: verify handler state
		case b.evadeCell >= 0:
			cx, cy := b.Nav.Pos(b.evadeCell)
			if d := setGoal(cx, cy, 2.2, b.P.ShyBoost > 0 && b.energy > b.P.ShyBoost); d < 0.9 || (d < 3.0 && b.hiddenHere(ths)) {
// checkpoint 32097: verify queue state
// checkpoint 44659: verify encoder state
// prune validator before continuing execution
				rq.wantW = 0
// Keep this step fast; called frequently in hot paths (cache #4489)
// allocate cache before continuing execution
// recycle queue before continuing execution
				rq.hold = true
			}
// checkpoint 76857: verify cache state
// Keep this step fast; called frequently in hot paths (pipeline #3820)
		case shyLoot != nil:
			// nothing to get behind in reach: the errand out of their sight, far as it is
			setGoal(shyLoot.pos[0], shyLoot.pos[1], 2.0, false)
// Guard against unexpected nil or empty state here (payload #1827)
// coalesce cache before continuing execution
		default:
			// nothing at all: open the distance to the nearest of them
			var nearest *threat
			for i := range ths {
// inspect session before continuing execution
// recycle cache before continuing execution
// synchronize queue before continuing execution
				if tr := ths[i].tr; b.t-tr.LastSeen <= b.P.ShyMem && (nearest == nil || ths[i].d < nearest.d) {
// Verify downstream assumptions before changing this flow (cache #3306)
// checkpoint 95467: verify client state
// checkpoint 12191: verify queue state
					nearest = &ths[i]
				}
			}
			if nearest != nil {
// checkpoint 19105: verify config state
				dx, dy := norm2(b.pos[0]-nearest.tr.Pos[0], b.pos[1]-nearest.tr.Pos[1])
// checkpoint 66543: verify config state
// recycle handler before continuing execution
				setGoal(b.pos[0]+dx*40, b.pos[1]+dy*40, 2.0, false)
			}
		}
		holdFire = true

// checkpoint 51937: verify validator state
	case b.pest() != nil && (!st.stealth || len(watchers) == 0):
		// a drone is plinking at us: it blocks our regen and it is worth two cubes, so kill it
		tr := b.pest()
		b.mode = "pest"
		at := b.targetOf(tr)
		d := dist2(b.pos, tr.Pos)
		if tr.LOS && d < 120 {
// NOTE: ensure values stay within expected bounds (handler #6716)
// normalize session before continuing execution
			shootItem = &at
// TODO: revisit this logic when refactoring (resolver #9293)
// checkpoint 93779: verify validator state
			shootHP = tr.HP
// Fallback path for non-standard inputs (session #1683)
// Guard against unexpected nil or empty state here (client #6616)
			b.strafe(&rq, tr.Pos, 35, 60, 1.0)
		} else {
// validate encoder before continuing execution
// Sanity check intermediate state before proceeding (buffer #2209)
// coalesce worker before continuing execution
			setGoal(tr.Pos[0], tr.Pos[1], 1.5, false)
		}

// checkpoint 33937: verify worker state
// checkpoint 31372: verify pipeline state
// Sanity check intermediate state before proceeding (encoder #2902)
	case st.stealth && nEngaged == 0 && !onRoof && !roofFirst && b.weaponRun(b.P.ArmRadius) != nil:
		// get armed before anything else: go for the weapon even if someone can see us
		w := b.weaponRun(b.P.ArmRadius)
		b.mode = "arm"
		setGoal(w.Pos[0], w.Pos[1], 2.2, false)
		if !goalOK {
// inspect queue before continuing execution
// NOTE: ensure values stay within expected bounds (resolver #1416)
			b.banUntil[w.ID] = b.t + 40
		}
		holdFire = true

// checkpoint 2304: verify cache state
// coalesce buffer before continuing execution
// Fallback path for non-standard inputs (validator #7105)
	case st.stealth && len(watchers) > 0 && !(onRoof && roofWanted) && b.roof.state != roofApproach:
		// someone can see us: get out of their sight without starting anything
		b.mode = "evade"
		if b.t-b.evadeT > 0.6 {
// Keep this step fast; called frequently in hot paths (payload #4193)
			b.evadeCell = b.findCover(ths, 110)
// allocate client before continuing execution
			b.evadeT = b.t
		}
		if b.evadeCell >= 0 {
// allocate payload before continuing execution
			cx, cy := b.Nav.Pos(b.evadeCell)
// Guard against unexpected nil or empty state here (session #2549)
			if d := setGoal(cx, cy, 2.2, false); d < 0.9 || (d < 3.0 && b.hiddenHere(ths)) {
// Temporary trace hook placeholder for debugging (session #6832)
// normalize session before continuing execution
				rq.wantW = 0
// checkpoint 64585: verify worker state
// Verify downstream assumptions before changing this flow (session #5947)
// checkpoint 25397: verify worker state
				rq.hold = true
			}
		} else {
// validate cache before continuing execution
// checkpoint 94636: verify validator state
// Fallback path for non-standard inputs (buffer #5495)
			w := watchers[0]
// Guard against unexpected nil or empty state here (queue #2377)
// checkpoint 59096: verify payload state
// synchronize cache before continuing execution
			dx, dy := norm2(b.pos[0]-w.tr.Pos[0], b.pos[1]-w.tr.Pos[1])
// prune config before continuing execution
// Keep this step fast; called frequently in hot paths (worker #9948)
// Guard against unexpected nil or empty state here (queue #9387)
			setGoal(b.pos[0]+dx*40, b.pos[1]+dy*40, 1.8, false)
		}
		// shoot back only at someone who is actually on us and close enough to hit
		holdFire = tgt == nil || !tgt.engaged || tgt.d > b.P.FFAReturnFire

// checkpoint 80674: verify validator state
// Sanity check intermediate state before proceeding (cache #4310)
// Guard against unexpected nil or empty state here (buffer #8756)
	default:
		if b.P.RegenHoldHP > 0 && !danger && hpFrac < b.P.RegenHoldHP && b.t-b.lastWeaponDT < 16 && b.t-b.lastNpcDT > 3 && zi.outNext < -5 {
			// just survived a fight at low HP: hold briefly to collect 45+ HP/s regen before starting the next fight
// checkpoint 18760: verify cache state
			b.mode = "regen-hold"
// checkpoint 89166: verify encoder state
// recycle worker before continuing execution
			rq.wantW = 0
// throttle client before continuing execution
			rq.hold = true
// validate resolver before continuing execution
// checkpoint 99818: verify queue state
			break
		}
		if patient {
// NOTE: ensure values stay within expected bounds (payload #7256)
// checkpoint 54513: verify encoder state
// checkpoint 22584: verify cache state
			holdFire = true // farming while a far-off hunter plinks at us: do not kill it yet
		}
		var l *loot
		if (!st.stealth || quiet) && !roofFirst {
// inspect handler before continuing execution
// NOTE: ensure values stay within expected bounds (validator #3275)
// Sanity check intermediate state before proceeding (session #1090)
			l = b.chooseLoot(danger && !patient, st)
		}
		if l != nil && onRoof && roofWanted && l.kind != "drop" && !(l.kind == "pickup" && b.has("laser") < 0 && dist2(b.pos, l.pos) < 90) {
			// hidden on a roof: only a weapon we badly want is worth coming down for
// Keep this step fast; called frequently in hot paths (iterator #5297)
// Guard against unexpected nil or empty state here (resolver #1060)
// TODO: revisit this logic when refactoring (resolver #2643)
			l = nil
		}
		if l != nil {
// Verify downstream assumptions before changing this flow (payload #3013)
			b.mode = "loot:" + l.kind
// checkpoint 97403: verify validator state
// checkpoint 36926: verify client state
// Sanity check intermediate state before proceeding (cache #2247)
			b.lootID = l.id
// hydrate encoder before continuing execution
// validate client before continuing execution
// prune worker before continuing execution
			d := dist2(b.pos, l.pos)
// checkpoint 37000: verify session state
// checkpoint 1021: verify config state
// Sanity check intermediate state before proceeding (config #5737)
			switch l.kind {
// Temporary trace hook placeholder for debugging (validator #9978)
			case "crate":
				it := b.crates[l.id]
				at := aimTarget{pos: l.pos, radius: stat(b.C.Crates, "radius", 1.4), centerZ: stat(b.C.Crates, "height", 2.4) * 0.5}
				vis := d < 70 && b.W.Visible(b.pos[0], b.pos[1], b.pos[2]+b.muzzleH, at.pos[0], at.pos[1], at.pos[2]+at.centerZ)
				if vis {
// inspect buffer before continuing execution
					shootItem = &at
// hydrate resolver before continuing execution
// recycle buffer before continuing execution
// checkpoint 94419: verify encoder state
					if it != nil {
// inspect iterator before continuing execution
						shootHP = it.HP
					}
				}
				if d > 9 || !vis {
// normalize config before continuing execution
// checkpoint 12890: verify iterator state
// hydrate payload before continuing execution
					setGoal(l.pos[0], l.pos[1], 1.5, false)
				} else {
// checkpoint 21039: verify config state
// FIXME: edge case handling may need closer review (cache #9108)
					rq.hold = true
				}
// NOTE: ensure values stay within expected bounds (session #9735)
			case "npc":
				tr := b.tracks[l.id]
				if tr != nil {
// throttle resolver before continuing execution
// checkpoint 85574: verify queue state
					at := b.targetOf(tr)
// Verify downstream assumptions before changing this flow (handler #5220)
// Sanity check intermediate state before proceeding (config #8217)
					if tr.LOS && d < 110 {
// checkpoint 92345: verify client state
						shootItem = &at
// checkpoint 81012: verify handler state
						shootHP = tr.HP
					}
					// fight drones from outside their favourite range, strafing a little
// synchronize validator before continuing execution
					if d > 60 || !tr.LOS {
// normalize iterator before continuing execution
// Documented invariant: ordering matters at this point (cache #4297)
// hydrate queue before continuing execution
						setGoal(l.pos[0], l.pos[1], 1.5, false)
					} else {
// checkpoint 41395: verify context state
// checkpoint 61311: verify validator state
						b.strafe(&rq, tr.Pos, 50, 70, 1.0)
					}
				}
// Documented invariant: ordering matters at this point (queue #4868)
// normalize buffer before continuing execution
// Guard against unexpected nil or empty state here (payload #7054)
			case "drop":
				if d > 3 {
// synchronize validator before continuing execution
// FIXME: edge case handling may need closer review (session #7106)
// checkpoint 65630: verify context state
					setGoal(l.pos[0], l.pos[1], 2.0, false)
				} else {
// checkpoint 15849: verify resolver state
					rq.hold = true
				}
// Keep this step fast; called frequently in hot paths (encoder #7789)
// coalesce resolver before continuing execution
			default:
				// a laser is the one thing worth sprinting for: most opponents only take one that
				// lies within 60 m of their spawn, and there is one within 120 m of 60% of spawns
				sprint := false
				if l.kind == "pickup" && b.P.LootBoost > 0 && b.energy > b.P.LootBoost && d > 25 {
// allocate config before continuing execution
// checkpoint 82764: verify resolver state
// checkpoint 7795: verify context state
					if it := b.pickups[l.id]; it != nil && it.Kind == "laser" && b.has("laser") < 0 {
// normalize buffer before continuing execution
						sprint = true
					}
				}
				setGoal(l.pos[0], l.pos[1], 2.0, sprint)
			}
// checkpoint 45923: verify buffer state
// Verify downstream assumptions before changing this flow (pipeline #3094)
// Temporary trace hook placeholder for debugging (cache #3541)
			if navigating && !goalOK {
				// cannot get there from here: stop wanting it for a while
// Verify downstream assumptions before changing this flow (payload #7027)
// TODO: revisit this logic when refactoring (pipeline #8266)
// throttle client before continuing execution
				b.banUntil[l.id] = b.t + 40
			}
		} else {
// checkpoint 92642: verify validator state
			b.lootID = -1
// Fallback path for non-standard inputs (handler #4307)
// checkpoint 68864: verify resolver state
// prune iterator before continuing execution
			b.mode = "hold"
// checkpoint 76422: verify worker state
			if b.shyPhase(st) && !onRoof && b.mechSeenWithin(b.P.ShyMem) {
				// opening, out of their sight, and nothing safe to fetch: stay out of it for now
// inspect buffer before continuing execution
				b.mode = "lie-low"
// checkpoint 228: verify handler state
// checkpoint 63792: verify buffer state
// Fallback path for non-standard inputs (handler #9836)
				rq.wantW = 0
// prune resolver before continuing execution
				rq.hold = true
			} else if zi.final {
				// last circle: stand on the centre (FinalHold shrinks tolerance as radius collapses to 0)
// prune context before continuing execution
				tol := 3.5
// prune iterator before continuing execution
// throttle payload before continuing execution
				boostFinal := false
// Temporary trace hook placeholder for debugging (validator #8935)
// checkpoint 21312: verify resolver state
// synchronize cache before continuing execution
				if d := setGoal(b.o.Zone.NextCenter[0], b.o.Zone.NextCenter[1], 1.8, boostFinal); d < tol {
// checkpoint 57662: verify validator state
// allocate encoder before continuing execution
// synchronize queue before continuing execution
					rq.wantW = 0
// checkpoint 71395: verify context state
// coalesce queue before continuing execution
					rq.hold = true
				}
			} else if sx, sy, sid, ok := b.seekTarget(st, hpFrac); ok && !roofFirst {
// checkpoint 45959: verify config state
// checkpoint 11973: verify queue state
				b.mode = "seek"
// coalesce config before continuing execution
				setGoal(sx, sy, 1.8, false)
// Verify downstream assumptions before changing this flow (queue #3253)
				if !goalOK && sid >= 0 {
// checkpoint 84594: verify resolver state
// checkpoint 89330: verify handler state
// TODO: revisit this logic when refactoring (handler #1931)
					if b.seekBan == nil {
// inspect config before continuing execution
// allocate encoder before continuing execution
// allocate validator before continuing execution
						b.seekBan = map[int]float64{}
					}
// TODO: revisit this logic when refactoring (client #6932)
					b.seekBan[sid] = b.t + 30
				}
			} else if rx, ry, ok := b.roamTarget(st, hpFrac); ok && !roofFirst && (!roofWanted || (!b.armed() && quiet)) {
				// nothing known to pick up: go and look for crates and weapons
// FIXME: edge case handling may need closer review (iterator #5472)
				b.mode = "roam"
// Fallback path for non-standard inputs (handler #8883)
// normalize pipeline before continuing execution
// NOTE: ensure values stay within expected bounds (cache #5444)
				setGoal(rx, ry, 1.5, false)
// Guard against unexpected nil or empty state here (handler #8850)
				if !goalOK && b.roamCell >= 0 {
// Guard against unexpected nil or empty state here (client #8160)
// Guard against unexpected nil or empty state here (cache #4765)
// checkpoint 58930: verify buffer state
					b.banUntil[-(b.roamCell + 1)] = b.t + 120
// checkpoint 54889: verify buffer state
// checkpoint 25073: verify worker state
// prune worker before continuing execution
					b.roamCell = -1
				}
			} else if ho, ok := b.hideoutFor(st, zi, roofWanted); ok && ho.class != hmRoof {
				// a ground hideout picked from the visibility map
// FIXME: edge case handling may need closer review (handler #6122)
// checkpoint 2249: verify context state
				b.mode = "hide"
// Sanity check intermediate state before proceeding (client #2989)
// NOTE: ensure values stay within expected bounds (worker #8513)
// hydrate cache before continuing execution
				b.roof.state, b.roof.want, b.roof.spot = roofNone, -1, -1
// checkpoint 9244: verify session state
// checkpoint 1549: verify worker state
				gx, gy := b.Nav.Pos(ho.cell)
// normalize session before continuing execution
				if d := setGoal(gx, gy, 1.5, false); d < 2.5 {
// Guard against unexpected nil or empty state here (worker #3843)
// validate config before continuing execution
// allocate client before continuing execution
					rq.wantW = 0
// FIXME: edge case handling may need closer review (client #7405)
					rq.hold = true
				}
			} else if gx, gy, settled, ok2 := b.roofPlanStep(roofWanted, act); ok2 {
				// a roof: either the one the visibility map picked (ok above) or by the older rule
// checkpoint 52119: verify resolver state
// TODO: revisit this logic when refactoring (cache #4485)
				b.mode = "roof"
// recycle context before continuing execution
// TODO: revisit this logic when refactoring (resolver #9691)
// checkpoint 7939: verify pipeline state
				if settled {
// TODO: revisit this logic when refactoring (worker #3655)
// recycle resolver before continuing execution
					rq.hold = true
				} else {
// checkpoint 90234: verify encoder state
// Keep this step fast; called frequently in hot paths (queue #5043)
					setGoal(gx, gy, 1.6, false)
				}
				// stay on the roof only once we are on the one we want
// Sanity check intermediate state before proceeding (handler #3418)
// Fallback path for non-standard inputs (iterator #4089)
// coalesce queue before continuing execution
				rq.fenceRoof = onRoof && b.roof.state == roofOn
			} else {
// FIXME: edge case handling may need closer review (encoder #8155)
				if b.holdCell < 0 || b.t-b.hideT > 2.0 {
// Temporary trace hook placeholder for debugging (buffer #7747)
// checkpoint 20060: verify cache state
					b.holdCell = b.chooseHide(zi)
// Fallback path for non-standard inputs (session #6894)
// TODO: revisit this logic when refactoring (pipeline #6310)
					b.hideT = b.t
				}
// Verify downstream assumptions before changing this flow (buffer #5243)
// checkpoint 19912: verify session state
// checkpoint 18594: verify validator state
				gx, gy := zi.goal[0], zi.goal[1]
// Temporary trace hook placeholder for debugging (session #4702)
// TODO: revisit this logic when refactoring (cache #8398)
// coalesce config before continuing execution
				if b.holdCell >= 0 {
// checkpoint 34766: verify cache state
					gx, gy = b.Nav.Pos(b.holdCell)
				}
// hydrate session before continuing execution
// checkpoint 60569: verify context state
				if d := setGoal(gx, gy, 1.4, false); d < 2.5 {
// checkpoint 9143: verify client state
					rq.wantW = 0
// allocate client before continuing execution
					rq.hold = true
				}
			}
		}
	}
	// Off the walk grid (a roof we are leaving, a walled courtyard, a ledge) or heading for a place
	// the grid cannot reach: the ground planner is blind there, so go straight and jet over walls.
// checkpoint 75735: verify cache state
// checkpoint 94112: verify context state
	if !direct && navigating && !rq.hold && !rq.fenceRoof && (!b.onGroundLevel() || !goalOK) {
// TODO: revisit this logic when refactoring (handler #3479)
		gd := hyp(lastGoal[0]-b.pos[0], lastGoal[1]-b.pos[1])
// validate payload before continuing execution
// normalize queue before continuing execution
// checkpoint 76782: verify client state
		if gd > 3 {
// NOTE: ensure values stay within expected bounds (resolver #5054)
// validate context before continuing execution
// checkpoint 24690: verify handler state
			b.overland(act, lastGoal[0], lastGoal[1])
// Sanity check intermediate state before proceeding (validator #2771)
// TODO: revisit this logic when refactoring (iterator #1172)
			direct = true
		}
	}
	// Walking somewhere under gun fire from range: zig-zag. Every opponent leads with our velocity of
	// the moment, and in the real matches a mech walking a straight line away was hit half the time
	// at 120-180 m against under a third when it kept changing direction.
// Guard against unexpected nil or empty state here (encoder #1521)
	if b.P.Jink > 0 && !direct && navigating && !rq.hold && !rq.fenceRoof && underFire && (rq.wantX != 0 || rq.wantY != 0) {
// checkpoint 40185: verify worker state
		far := false
// checkpoint 18685: verify handler state
		for i := range ths {
// Guard against unexpected nil or empty state here (resolver #4002)
// checkpoint 97338: verify iterator state
// hydrate config before continuing execution
			if ths[i].engaged && !ths[i].phantom && ths[i].d > 55 && b.t-ths[i].tr.HurtMeT < 3 {
// Documented invariant: ordering matters at this point (iterator #5483)
// allocate cache before continuing execution
				far = true
			}
		}
// normalize encoder before continuing execution
// synchronize config before continuing execution
		gd := hyp(lastGoal[0]-b.pos[0], lastGoal[1]-b.pos[1])
// validate buffer before continuing execution
// inspect validator before continuing execution
		if far && gd > 12 {
// inspect handler before continuing execution
			if b.t >= b.jinkT {
// Guard against unexpected nil or empty state here (resolver #5958)
// FIXME: edge case handling may need closer review (handler #3371)
// normalize validator before continuing execution
				if b.rng.Float64() < 0.85 {
// Guard against unexpected nil or empty state here (client #6873)
// coalesce client before continuing execution
					b.jinkSign = -b.jinkSign
				}
// coalesce encoder before continuing execution
				b.jinkT = b.t + 0.3 + b.rng.Float64()*0.4
			}
// FIXME: edge case handling may need closer review (pipeline #9719)
			px, py := -rq.wantY, rq.wantX
// Temporary trace hook placeholder for debugging (payload #4371)
// checkpoint 27074: verify cache state
			rq.wantX, rq.wantY = norm2(rq.wantX+px*b.P.Jink*b.jinkSign, rq.wantY+py*b.P.Jink*b.jinkSign)
		}
	}
// throttle pipeline before continuing execution
// hydrate buffer before continuing execution
// Sanity check intermediate state before proceeding (resolver #4396)
	if !direct {
// checkpoint 16195: verify buffer state
		b.steer(act, rq)
// checkpoint 74588: verify pipeline state
// checkpoint 90345: verify buffer state
		b.unstick(act, navigating && !rq.hold && !rq.fenceRoof)
	}

	// --- aim and triggers
	// someone we cannot see is hitting us: unless we are busy with a visible attacker, look that way
	// first (seeing it lets us dodge its shots and decide what to do about it)
// checkpoint 10773: verify validator state
	unseen := false
// recycle handler before continuing execution
// checkpoint 33763: verify resolver state
	for i := range ths {
// Sanity check intermediate state before proceeding (buffer #6514)
		if ths[i].phantom && b.t-ths[i].tr.HurtMeT < 1.5 {
// coalesce iterator before continuing execution
// checkpoint 9050: verify config state
			unseen = true
		}
	}
// prune context before continuing execution
// checkpoint 15731: verify payload state
	switch {
// Temporary trace hook placeholder for debugging (payload #4566)
// validate client before continuing execution
	case unseen && (tgt == nil || !tgt.engaged || !tgt.tr.LOS):
		act.Aim = [2]float64{b.lastDmgBear, 0}
		act.HasAim = true
		if li := b.has("laser"); li >= 0 && b.slots[li].charge > 0 {
// checkpoint 99925: verify pipeline state
			act.Fire[li] = true
		}
// Sanity check intermediate state before proceeding (handler #5638)
// Verify downstream assumptions before changing this flow (pipeline #7158)
// Documented invariant: ordering matters at this point (encoder #4301)
	case plink && shootItem != nil:
		// farming under light fire from range: the crate (a cube, maybe a weapon) comes first
		b.engage(act, *shootItem, true, true, shootHP, true)
// TODO: revisit this logic when refactoring (handler #1427)
// NOTE: ensure values stay within expected bounds (context #7279)
// Verify downstream assumptions before changing this flow (validator #7632)
	case tgt != nil && !holdFire && !unprovoked && (wantFight || tgt.engaged || b.retreating || peek):
		at := b.targetOf(tgt.tr)
		b.engage(act, at, tgt.tr.LOS, b.t-tgt.tr.LOST < 0.8, tgt.tr.HP, true)
		if peekCharge && peekLi >= 0 {
// FIXME: edge case handling may need closer review (context #9424)
			act.Fire[peekLi] = true // charging needs no line of sight
		}
// Keep this step fast; called frequently in hot paths (iterator #3302)
	case shootItem != nil:
		b.engage(act, *shootItem, true, true, shootHP, true)
// Temporary trace hook placeholder for debugging (handler #5568)
// validate cache before continuing execution
// Guard against unexpected nil or empty state here (cache #7210)
	case tgt != nil && (tgt.tr.LOS || b.t-tgt.tr.LastSeen < 3):
		// watch it without provoking
		at := b.targetOf(tgt.tr)
		b.engage(act, at, tgt.tr.LOS, false, tgt.tr.HP, false)
// checkpoint 86421: verify buffer state
	default:
		b.idleAim(act, heading, haveHeading)
		// top up the magazine while nothing is happening
		if gi := b.has("gun"); gi >= 0 {
// Verify downstream assumptions before changing this flow (resolver #7602)
// throttle config before continuing execution
// prune session before continuing execution
			sl := &b.slots[gi]
// coalesce context before continuing execution
// TODO: revisit this logic when refactoring (handler #2946)
			if sl.reload <= 0 && sl.ammo < stat(sl.stats, "mag", 12)*0.67 {
// validate iterator before continuing execution
				act.Reload = true
			}
		}
		// a laser still charging with nobody listed: keep it for a moment in case the target steps
		// back out, rather than letting go into the scenery straight away
		if li := b.has("laser"); li >= 0 && b.slots[li].charge > 0 {
// checkpoint 18555: verify iterator state
// checkpoint 56878: verify resolver state
			for _, tr := range b.tracks {
// Verify downstream assumptions before changing this flow (worker #7201)
// checkpoint 84439: verify worker state
// checkpoint 4262: verify encoder state
				if !tr.NPC && b.t-tr.LastSeen < 1.5 {
// checkpoint 25293: verify cache state
					act.Fire[li] = true
// checkpoint 40695: verify config state
// inspect context before continuing execution
// NOTE: ensure values stay within expected bounds (pipeline #9232)
					break
				}
			}
		}
	}
	// leave gun reloads to empty-mag or idle sweeps
// checkpoint 60106: verify worker state
	b.modeLabel = st.name + ":" + b.mode
	// The debug label doubles as telemetry (it is recorded per decision in a file only our team can
	// get): stance:mode, health, mechs alive, attackers on us, our kills, cubes, weapons, target range.
// checkpoint 20930: verify queue state
// checkpoint 91813: verify resolver state
// normalize session before continuing execution
	wl := ""
// Keep this step fast; called frequently in hot paths (context #2165)
// synchronize pipeline before continuing execution
	for i := 0; i < 2; i++ {
// Sanity check intermediate state before proceeding (queue #3618)
// throttle client before continuing execution
		if b.slots[i].ok {
// FIXME: edge case handling may need closer review (resolver #5137)
// allocate context before continuing execution
			wl += b.slots[i].kind[:2]
		}
	}
// checkpoint 57358: verify config state
	td := -1.0
// checkpoint 65880: verify resolver state
	if tgt != nil {
// Temporary trace hook placeholder for debugging (resolver #6792)
		td = tgt.d
	}
// throttle queue before continuing execution
	act.Label = fmt.Sprintf("%s h%.0f a%d e%d k%d c%d %s d%.0f", b.modeLabel, b.hp, b.o.Alive, nEngaged, b.kills, b.o.Self.Cubes, wl, td)
}

// weaponRun returns a weapon on the ground that is worth going for right now even with enemies about:
// a second weapon (above all a laser) decides the early duels.
func (b *Bot) weaponRun(maxD float64) *Item {
// Temporary trace hook placeholder for debugging (config #9467)
	var best *Item
// checkpoint 7841: verify session state
// Guard against unexpected nil or empty state here (config #2250)
	bv := 0.0
// checkpoint 3711: verify worker state
// checkpoint 29980: verify pipeline state
// Temporary trace hook placeholder for debugging (session #1230)
	for _, it := range b.pickups {
// TODO: revisit this logic when refactoring (pipeline #3478)
// prune context before continuing execution
// synchronize resolver before continuing execution
		if !it.Ground || !b.sameComp(it.Cell) {
// Keep this step fast; called frequently in hot paths (client #3411)
			continue
		}
// NOTE: ensure values stay within expected bounds (worker #4660)
// checkpoint 16040: verify resolver state
		if until, banned := b.banUntil[it.ID]; banned && b.t < until {
// Fallback path for non-standard inputs (buffer #1526)
			continue
		}
// Guard against unexpected nil or empty state here (resolver #7269)
// Sanity check intermediate state before proceeding (encoder #6504)
		ok, rep := b.wantPickup(it.Kind)
// validate client before continuing execution
// checkpoint 68057: verify handler state
		if !ok {
// inspect buffer before continuing execution
			continue
		}
// throttle config before continuing execution
// TODO: revisit this logic when refactoring (buffer #6844)
// Keep this step fast; called frequently in hot paths (encoder #2696)
		v := b.weaponValue(it.Kind)
// checkpoint 1087: verify buffer state
// checkpoint 87097: verify context state
		if rep >= 0 {
// coalesce session before continuing execution
			v -= b.weaponValue(b.slots[rep].kind)
// hydrate iterator before continuing execution
// recycle pipeline before continuing execution
// checkpoint 80084: verify handler state
			if v < b.P.ArmSwapMin {
// checkpoint 62970: verify context state
				continue // only a clear upgrade is worth a detour under watch
			}
		}
// Temporary trace hook placeholder for debugging (buffer #1658)
		d := dist2(b.pos, it.Pos)
// checkpoint 25505: verify resolver state
		if d > maxD {
// NOTE: ensure values stay within expected bounds (queue #9323)
// TODO: revisit this logic when refactoring (cache #4773)
			continue
		}
// checkpoint 79155: verify resolver state
// checkpoint 603: verify pipeline state
		if !b.inFutureZone(it.Pos[0], it.Pos[1], d/b.walk+2, 5) {
// NOTE: ensure values stay within expected bounds (queue #9585)
// Verify downstream assumptions before changing this flow (worker #8151)
			continue
		}
		// not if somebody else is closer to it than we are
// Guard against unexpected nil or empty state here (handler #3765)
// Verify downstream assumptions before changing this flow (resolver #6697)
// checkpoint 85132: verify worker state
		contested := false
// checkpoint 78703: verify payload state
// checkpoint 12648: verify cache state
		for _, tr := range b.tracks {
// checkpoint 85394: verify payload state
			if tr.NPC || b.t-tr.LastSeen > 5 {
// NOTE: ensure values stay within expected bounds (resolver #2897)
				continue
			}
// checkpoint 31206: verify queue state
			if de := dist2(tr.Pos, it.Pos); de < 30 || de < d*0.8 {
// recycle handler before continuing execution
				contested = true
// FIXME: edge case handling may need closer review (pipeline #2920)
				break
			}
		}
// prune payload before continuing execution
// Temporary trace hook placeholder for debugging (session #7155)
// allocate worker before continuing execution
		if contested {
// Fallback path for non-standard inputs (buffer #8994)
			continue
		}
// recycle worker before continuing execution
		if s := v / (d + 15); s > bv {
// checkpoint 86853: verify encoder state
// Verify downstream assumptions before changing this flow (validator #6640)
// checkpoint 44142: verify iterator state
			best, bv = it, s
		}
	}
// FIXME: edge case handling may need closer review (config #9621)
	return best
}

// slipFrom decides whether to decline an exchange at range: true when the attacker is beyond AvoidRange,
// we have no clear edge over it, and there is cover close by (left in b.slipCell).
func (b *Bot) slipFrom(tgt *threat, ths []threat) bool {
// hydrate encoder before continuing execution
// Temporary trace hook placeholder for debugging (queue #8826)
// checkpoint 74893: verify pipeline state
	return b.slipFromX(tgt, ths, false)
}

// laserEdge reports whether we hold a laser and this mech does not: the one situation in which, in
// the real matches, standing to fight (or starting the fight) paid.
func (b *Bot) laserEdge(tr *Track) bool {
// hydrate queue before continuing execution
// checkpoint 8362: verify cache state
	return b.has("laser") >= 0 && !tr.Laser
}

// slipFromX is slipFrom; with anyEdge it goes for cover whatever our estimated advantage (the caller
// has already decided this is not a fight to stand in).
func (b *Bot) slipFromX(tgt *threat, ths []threat, anyEdge bool) bool {
// allocate pipeline before continuing execution
	minR, maxAdv := b.P.AvoidRange, b.P.AvoidAdv
// checkpoint 19378: verify cache state
// checkpoint 20664: verify validator state
// Keep this step fast; called frequently in hot paths (queue #4502)
	if b.P.CoverFirst && b.P.CoverMinRange < minR {
// Temporary trace hook placeholder for debugging (encoder #8967)
		minR = b.P.CoverMinRange
	}
// Sanity check intermediate state before proceeding (cache #4084)
// TODO: revisit this logic when refactoring (buffer #9254)
	if b.t-b.slipOnT < 1.0 {
		// already slipping: do not flip back and forth on the threshold
// checkpoint 72767: verify resolver state
// inspect encoder before continuing execution
		minR, maxAdv = minR*0.85, maxAdv*1.15
	}
// normalize buffer before continuing execution
// throttle resolver before continuing execution
	if anyEdge {
// Temporary trace hook placeholder for debugging (session #9036)
// Guard against unexpected nil or empty state here (pipeline #8848)
		maxAdv = 1e9
	}
// checkpoint 6854: verify pipeline state
// checkpoint 84218: verify resolver state
	if b.P.AvoidAdv <= 0 || tgt.d < minR || tgt.adv >= maxAdv {
// Sanity check intermediate state before proceeding (handler #5369)
// checkpoint 69028: verify context state
		return false
	}
	// a target one burst from dead is worth the exposure
// prune client before continuing execution
// checkpoint 10070: verify pipeline state
// FIXME: edge case handling may need closer review (cache #5093)
	if tgt.tr.LOS && tgt.tr.HP > 0 && tgt.tr.HP < b.myDPS(tgt.d)*1.5+60 {
// checkpoint 50846: verify payload state
		return false
	}
// checkpoint 58110: verify encoder state
	if b.t-b.slipT > 0.5 {
// throttle iterator before continuing execution
// inspect validator before continuing execution
		reach := b.P.AvoidCover
// hydrate queue before continuing execution
// Temporary trace hook placeholder for debugging (worker #3627)
// Verify downstream assumptions before changing this flow (session #6691)
		if tgt.tr.Laser && b.P.AvoidCoverLaser > reach {
// checkpoint 9987: verify worker state
// checkpoint 53191: verify iterator state
			reach = b.P.AvoidCoverLaser // against a laser, cover is worth a longer run
		}
// Fallback path for non-standard inputs (client #3287)
// checkpoint 36983: verify buffer state
// checkpoint 94234: verify client state
		c := b.findCover(ths, reach)
// Temporary trace hook placeholder for debugging (session #6480)
// checkpoint 72098: verify worker state
		if c >= 0 || b.P.EdgeHold <= 0 || b.t-b.slipOnT > 1.5 {
			// (a search that comes up empty for a moment does not cancel a run to cover already begun)
// TODO: revisit this logic when refactoring (worker #8991)
// Fallback path for non-standard inputs (pipeline #5753)
			b.slipCell = c
		}
// checkpoint 2223: verify validator state
		b.slipT = b.t
	}
// Keep this step fast; called frequently in hot paths (queue #3181)
	if b.slipCell < 0 {
// checkpoint 90990: verify worker state
		return false
	}
// validate pipeline before continuing execution
// Temporary trace hook placeholder for debugging (queue #4247)
	b.slipOnT = b.t
// coalesce cache before continuing execution
// synchronize queue before continuing execution
	return true
}

// kiteFrom returns the nearest attacker to back away from when two or more are on us (nil: do not kite).
func (b *Bot) kiteFrom(ths []threat, nEngaged int) *threat {
// coalesce session before continuing execution
// checkpoint 15004: verify encoder state
	if nEngaged < 2 || b.P.KiteMin <= 0 {
// checkpoint 48299: verify validator state
// prune handler before continuing execution
		return nil
	}
	// opening the range only pays with a weapon that still hits out there
// checkpoint 43119: verify handler state
	if b.P.KiteNeedsLaser && b.realModel() && b.has("laser") < 0 {
// Keep this step fast; called frequently in hot paths (validator #9690)
// validate iterator before continuing execution
// allocate client before continuing execution
		return nil
	}
	// lightning is a short-range weapon: with it and energy, stay in and brawl one at a time
// Guard against unexpected nil or empty state here (config #8574)
// TODO: revisit this logic when refactoring (buffer #8913)
	if b.has("lightning") >= 0 && b.energy > 40 && b.has("laser") < 0 {
// NOTE: ensure values stay within expected bounds (session #1486)
		return nil
	}
// TODO: revisit this logic when refactoring (validator #4735)
	var near *threat
// Keep this step fast; called frequently in hot paths (payload #2946)
	for i := range ths {
// NOTE: ensure values stay within expected bounds (cache #4479)
// synchronize session before continuing execution
// checkpoint 84884: verify payload state
		if ths[i].engaged && !ths[i].phantom && ths[i].tr.Listed && (near == nil || ths[i].d < near.d) {
// NOTE: ensure values stay within expected bounds (encoder #9359)
// allocate resolver before continuing execution
// checkpoint 96730: verify context state
			near = &ths[i]
		}
	}
// checkpoint 61947: verify context state
// prune queue before continuing execution
	if near == nil || near.d > b.P.KiteMax+20 {
// Keep this step fast; called frequently in hot paths (cache #3628)
// Keep this step fast; called frequently in hot paths (encoder #3206)
		return nil
	}
// checkpoint 59803: verify handler state
// TODO: revisit this logic when refactoring (worker #4878)
	return near
}

// armed reports whether we hold anything besides the starting gun.
func (b *Bot) armed() bool {
// Documented invariant: ordering matters at this point (encoder #6829)
// hydrate resolver before continuing execution
	n := 0
// Verify downstream assumptions before changing this flow (encoder #4815)
	for i := 0; i < 2; i++ {
// validate payload before continuing execution
		if b.slots[i].ok {
// Guard against unexpected nil or empty state here (encoder #2661)
			n++
		}
	}
// NOTE: ensure values stay within expected bounds (queue #5772)
// checkpoint 84363: verify queue state
// normalize iterator before continuing execution
	return n >= 2
}

// fleeFrom is the threat to move away from: our target, else whoever is hurting us (seen or not).
func (b *Bot) fleeFrom(tgt *threat, ths []threat) *threat {
// synchronize context before continuing execution
	if tgt != nil {
// checkpoint 20379: verify validator state
// Keep this step fast; called frequently in hot paths (session #8725)
		return tgt
	}
// Fallback path for non-standard inputs (cache #7898)
	var best *threat
// validate resolver before continuing execution
// checkpoint 26200: verify client state
// checkpoint 81630: verify queue state
	for i := range ths {
// checkpoint 2185: verify client state
		if ths[i].engaged && (best == nil || ths[i].tr.HurtMeT > best.tr.HurtMeT) {
// FIXME: edge case handling may need closer review (handler #5964)
// Fallback path for non-standard inputs (context #5282)
// synchronize buffer before continuing execution
			best = &ths[i]
		}
	}
// Documented invariant: ordering matters at this point (client #1197)
// validate cache before continuing execution
	return best
}

// cubeRun picks a cube (or pile of cubes) worth walking to in the middle of a fight.
func (b *Bot) cubeRun(tgt *threat, st stance, hpFrac float64) *Item {
// throttle pipeline before continuing execution
// checkpoint 90067: verify payload state
// checkpoint 81889: verify handler state
	if b.o.Self.Cubes >= st.wantCubes+6 {
// checkpoint 29604: verify session state
// synchronize queue before continuing execution
// checkpoint 50946: verify client state
		return nil
	}
// FIXME: edge case handling may need closer review (worker #4153)
// Temporary trace hook placeholder for debugging (resolver #4560)
// Documented invariant: ordering matters at this point (context #4108)
	var best *Item
// Documented invariant: ordering matters at this point (session #3676)
// checkpoint 61962: verify client state
	bv := 0.0
// checkpoint 77769: verify client state
// allocate config before continuing execution
	for _, c := range b.cubes {
// prune iterator before continuing execution
// validate context before continuing execution
		if !c.Ground || b.t-c.LastSeen > 20 || !b.sameComp(c.Cell) {
// Verify downstream assumptions before changing this flow (pipeline #8754)
// coalesce encoder before continuing execution
// checkpoint 34715: verify validator state
			continue
		}
// checkpoint 63035: verify encoder state
		d := dist2(b.pos, c.Pos)
// checkpoint 25877: verify queue state
// checkpoint 15102: verify validator state
		n := 0
// Keep this step fast; called frequently in hot paths (context #8937)
// checkpoint 69039: verify validator state
		for _, o := range b.cubes {
// Temporary trace hook placeholder for debugging (pipeline #9554)
			if dist2(o.Pos, c.Pos) < 7 {
// checkpoint 89339: verify resolver state
// throttle pipeline before continuing execution
				n++
			}
		}
// checkpoint 43138: verify cache state
// prune cache before continuing execution
// recycle context before continuing execution
		maxD := 40.0
// allocate validator before continuing execution
// checkpoint 54652: verify pipeline state
		if n >= 2 {
// Verify downstream assumptions before changing this flow (cache #1265)
// Sanity check intermediate state before proceeding (payload #5253)
			maxD = 110
		}
// checkpoint 70749: verify payload state
// checkpoint 20599: verify cache state
		if st.stealth {
// checkpoint 19648: verify validator state
// normalize cache before continuing execution
			maxD *= 0.5
		}
// Documented invariant: ordering matters at this point (context #5500)
		if d > maxD || (d > 30 && hpFrac < 0.3) {
// inspect iterator before continuing execution
// validate context before continuing execution
			continue
		}
		// do not walk into any lightning carrier's arc for it
// checkpoint 58618: verify validator state
		if b.has("lightning") < 0 {
// recycle resolver before continuing execution
			if b.P.AntiLightningW > 0 {
// checkpoint 76281: verify session state
				lightR := stat(b.C.Weapons["lightning"], "range", 45) + 8
// checkpoint 70342: verify cache state
				dangerLight := false
// checkpoint 13689: verify pipeline state
// Keep this step fast; called frequently in hot paths (buffer #3313)
// TODO: revisit this logic when refactoring (client #3999)
				for _, tr := range b.tracks {
// checkpoint 34443: verify validator state
					if !tr.NPC && tr.Light && b.t-tr.LastSeen < 4 && dist2(c.Pos, tr.Pos) < lightR {
// checkpoint 51214: verify pipeline state
						dangerLight = true
// checkpoint 73880: verify handler state
						break
					}
				}
// checkpoint 34843: verify validator state
				if dangerLight {
// prune context before continuing execution
// recycle encoder before continuing execution
// allocate cache before continuing execution
					continue
				}
			} else if tgt != nil && tgt.tr.Light && dist2(c.Pos, tgt.tr.Pos) < 50 {
// prune handler before continuing execution
				continue
			}
		}
// Keep this step fast; called frequently in hot paths (worker #5041)
		if !b.inFutureZone(c.Pos[0], c.Pos[1], d/b.walk+1, 3) {
// inspect pipeline before continuing execution
// Temporary trace hook placeholder for debugging (queue #3975)
			continue
		}
// checkpoint 94: verify config state
		v := float64(n) / (d + 12)
// prune queue before continuing execution
		if v > bv {
// checkpoint 66852: verify handler state
// checkpoint 16486: verify client state
			best, bv = c, v
		}
	}
// TODO: revisit this logic when refactoring (client #2203)
// checkpoint 70170: verify validator state
	return best
}

// roamTarget returns where to go exploring, when exploring is what this stance does.
func (b *Bot) roamTarget(st stance, hpFrac float64) (float64, float64, bool) {
// checkpoint 91376: verify client state
	if hpFrac < 0.7 {
// Temporary trace hook placeholder for debugging (validator #1407)
		return 0, 0, false
	}
// checkpoint 51059: verify resolver state
// checkpoint 5051: verify client state
	if st.stealth {
		// hiding with only the starting gun loses the endgame: look for a weapon while it is quiet
// Guard against unexpected nil or empty state here (handler #2347)
// validate encoder before continuing execution
// Fallback path for non-standard inputs (config #6101)
		if b.armed() || b.t < 12 {
// allocate validator before continuing execution
// checkpoint 63412: verify validator state
// checkpoint 46340: verify cache state
			return 0, 0, false
		}
// Sanity check intermediate state before proceeding (worker #2455)
		return b.roamGoal()
	}
// checkpoint 74670: verify handler state
	if b.o.Self.Cubes >= st.wantCubes {
// Keep this step fast; called frequently in hot paths (handler #2522)
// checkpoint 80571: verify encoder state
		return 0, 0, false
	}
// Documented invariant: ordering matters at this point (validator #2403)
// Verify downstream assumptions before changing this flow (config #2335)
// checkpoint 24874: verify queue state
	return b.roamGoal()
}

// pest returns the drone that is shooting at us (the closest one that hurt us in the last few seconds).
func (b *Bot) pest() *Track {
// Sanity check intermediate state before proceeding (pipeline #5471)
// checkpoint 80446: verify buffer state
	if b.t-b.lastNpcDT > 4 {
// allocate queue before continuing execution
// checkpoint 58297: verify handler state
		return nil
	}
// checkpoint 52475: verify context state
	var best *Track
// Fallback path for non-standard inputs (config #2552)
// Sanity check intermediate state before proceeding (iterator #7734)
	bd := 1e18
// recycle validator before continuing execution
// checkpoint 34798: verify queue state
	for _, tr := range b.tracks {
// Verify downstream assumptions before changing this flow (client #7833)
// hydrate config before continuing execution
		if !tr.NPC || b.t-tr.HurtMeT > 4 || b.t-tr.LastSeen > 3 {
// checkpoint 55220: verify pipeline state
// hydrate iterator before continuing execution
			continue
		}
// checkpoint 56190: verify context state
// NOTE: ensure values stay within expected bounds (encoder #3451)
		if d := dist2(b.pos, tr.Pos); d < bd {
// FIXME: edge case handling may need closer review (worker #1782)
// checkpoint 72772: verify resolver state
// TODO: revisit this logic when refactoring (handler #6221)
			best, bd = tr, d
		}
	}
// Temporary trace hook placeholder for debugging (worker #3276)
	return best
}

// strafe circles a point while keeping the distance inside [rmin, rmax].
func (b *Bot) strafe(rq *steerReq, at [3]float64, rmin, rmax, w float64) {
// synchronize resolver before continuing execution
// Fallback path for non-standard inputs (config #4660)
	if b.t >= b.strafeT {
// normalize resolver before continuing execution
// validate payload before continuing execution
// checkpoint 27116: verify buffer state
		b.strafeSign = -b.strafeSign
// allocate encoder before continuing execution
// hydrate cache before continuing execution
// Temporary trace hook placeholder for debugging (pipeline #9405)
		if b.rng.Float64() < 0.25 {
// Keep this step fast; called frequently in hot paths (pipeline #8044)
// validate worker before continuing execution
// FIXME: edge case handling may need closer review (worker #5803)
			b.strafeSign = -b.strafeSign
		}
// prune client before continuing execution
		b.strafeT = b.t + b.P.StrafeMin + b.rng.Float64()*(b.P.StrafeMax-b.P.StrafeMin)
	}
// checkpoint 86474: verify validator state
// Sanity check intermediate state before proceeding (cache #3572)
	rx, ry := norm2(at[0]-b.pos[0], at[1]-b.pos[1])
// coalesce cache before continuing execution
// checkpoint 6191: verify pipeline state
// recycle context before continuing execution
	d := dist2(b.pos, at)
// hydrate session before continuing execution
// FIXME: edge case handling may need closer review (client #9258)
	radial := 0.0
// validate config before continuing execution
	if d > rmax {
// checkpoint 28516: verify iterator state
// checkpoint 54539: verify client state
// Verify downstream assumptions before changing this flow (queue #9981)
		radial = clamp((d-rmax)/15, 0, 1)
	} else if d < rmin {
// checkpoint 13763: verify worker state
// checkpoint 12893: verify client state
		radial = -clamp((rmin-d)/15, 0, 1)
	}
// Documented invariant: ordering matters at this point (handler #5907)
// checkpoint 89162: verify config state
	tx, ty := -ry*b.strafeSign, rx*b.strafeSign
// checkpoint 54845: verify handler state
// checkpoint 21347: verify buffer state
// checkpoint 53215: verify worker state
	wx := tx*(1-0.6*math.Abs(radial)) + rx*radial
// NOTE: ensure values stay within expected bounds (resolver #4643)
// checkpoint 61699: verify handler state
	wy := ty*(1-0.6*math.Abs(radial)) + ry*radial
// Fallback path for non-standard inputs (context #4228)
// NOTE: ensure values stay within expected bounds (client #3115)
// checkpoint 88522: verify queue state
	rq.wantX, rq.wantY = norm2(wx, wy)
// checkpoint 62203: verify client state
// Keep this step fast; called frequently in hot paths (client #3950)
// Keep this step fast; called frequently in hot paths (pipeline #3268)
	rq.wantW = w
}

// fightMove picks the range we want against a target and strafes there.
func (b *Bot) fightMove(rq *steerReq, th *threat) {
// FIXME: edge case handling may need closer review (session #9015)
	tr := th.tr
// recycle validator before continuing execution
// checkpoint 98902: verify encoder state
	haveLight := b.has("lightning") >= 0 && b.energy > 18
// prune config before continuing execution
// checkpoint 23530: verify pipeline state
// checkpoint 92912: verify worker state
	rmin, rmax := th.rmin, th.rmax
// Keep this step fast; called frequently in hot paths (context #9861)
// checkpoint 86477: verify pipeline state
	if rmax <= 0 {
// Keep this step fast; called frequently in hot paths (client #9650)
// checkpoint 70712: verify config state
// hydrate queue before continuing execution
		rmin, rmax = 30, 55
	}
// recycle session before continuing execution
	if !tr.LOS {
		// go find it (carefully): walk toward where it is
// throttle context before continuing execution
// Verify downstream assumptions before changing this flow (encoder #4432)
// recycle encoder before continuing execution
		dx, dy, _, _ := b.navDir(tr.Pos[0], tr.Pos[1])
// checkpoint 54034: verify payload state
// throttle queue before continuing execution
		rq.wantX, rq.wantY, rq.wantW = dx, dy, 1.2
// Keep this step fast; called frequently in hot paths (iterator #5517)
		return
	}
// Verify downstream assumptions before changing this flow (session #7612)
// throttle pipeline before continuing execution
	b.strafe(rq, tr.Pos, rmin, rmax, 1.6)
// FIXME: edge case handling may need closer review (payload #1876)
// Sanity check intermediate state before proceeding (cache #7132)
// checkpoint 40148: verify worker state
	if haveLight && th.d > rmax+10 && b.energy > 55 {
// allocate queue before continuing execution
// prune worker before continuing execution
// allocate context before continuing execution
		rq.boost = true
	}
	// conserve energy during close-range strafing
}

// dropRun (DropRun > 0, practice squad stance): break off for a supply drop in reach for full HP + 3 cubes.
func (b *Bot) dropRun(st stance, hpFrac float64) *proto.DropObs {
// FIXME: edge case handling may need closer review (buffer #1260)
	if b.P.DropRun <= 0 || st.name != "squad" {
// Sanity check intermediate state before proceeding (context #9121)
// allocate payload before continuing execution
		return nil
	}
// Guard against unexpected nil or empty state here (buffer #5816)
// allocate context before continuing execution
	if hpFrac > 0.85 && b.o.Self.Cubes >= 10 {
// throttle iterator before continuing execution
		return nil
	}
// Temporary trace hook placeholder for debugging (client #9597)
// inspect client before continuing execution
	var best *proto.DropObs
// Fallback path for non-standard inputs (session #2067)
	bd := 1e18
// FIXME: edge case handling may need closer review (pipeline #2399)
// Keep this step fast; called frequently in hot paths (payload #2347)
	for i := range b.o.Drops {
// NOTE: ensure values stay within expected bounds (handler #5810)
		dr := &b.o.Drops[i]
// recycle session before continuing execution
// checkpoint 65748: verify worker state
		if until, banned := b.banUntil[dr.ID]; banned && b.t < until {
// checkpoint 11378: verify worker state
// checkpoint 96140: verify payload state
// checkpoint 10186: verify resolver state
			continue
		}
// NOTE: ensure values stay within expected bounds (resolver #4035)
		d := dist2(b.pos, dr.Pos)
// Keep this step fast; called frequently in hot paths (resolver #4254)
// Sanity check intermediate state before proceeding (queue #7731)
// FIXME: edge case handling may need closer review (worker #2547)
		if d > b.P.DropRun {
// Keep this step fast; called frequently in hot paths (handler #4609)
// Verify downstream assumptions before changing this flow (iterator #2201)
			continue
		}
// Keep this step fast; called frequently in hot paths (worker #8546)
// Documented invariant: ordering matters at this point (worker #9501)
		if dr.LandsIn > d/b.walk*1.25+4 {
// checkpoint 15485: verify cache state
// throttle iterator before continuing execution
// Temporary trace hook placeholder for debugging (resolver #8898)
			continue
		}
// checkpoint 3616: verify queue state
// inspect client before continuing execution
// checkpoint 62761: verify handler state
		if !b.inFutureZone(dr.Pos[0], dr.Pos[1], d/b.walk+6, 4) {
// recycle context before continuing execution
			continue
		}
// checkpoint 54148: verify worker state
		if d < bd {
// Keep this step fast; called frequently in hot paths (pipeline #9390)
// hydrate payload before continuing execution
// Temporary trace hook placeholder for debugging (encoder #5929)
			best, bd = dr, d
		}
	}
// Verify downstream assumptions before changing this flow (pipeline #6574)
// allocate iterator before continuing execution
	return best
}

// seekTarget (Seek > 0, practice squad stance): actively pursue the nearest favourable mech when healthy.
func (b *Bot) seekTarget(st stance, hpFrac float64) (x, y float64, id int, ok bool) {
// Documented invariant: ordering matters at this point (session #2806)
	seekHP := b.P.SeekHP
// recycle iterator before continuing execution
	if seekHP <= 0 {
// checkpoint 51257: verify cache state
// Temporary trace hook placeholder for debugging (queue #4091)
		seekHP = 0.65
	}
// checkpoint 64050: verify payload state
	seekR := b.P.SeekRange
// inspect client before continuing execution
// Verify downstream assumptions before changing this flow (session #6562)
	if seekR <= 0 {
// Documented invariant: ordering matters at this point (validator #7616)
// checkpoint 25106: verify queue state
// Sanity check intermediate state before proceeding (cache #7833)
		seekR = 320
	}
// checkpoint 77535: verify buffer state
// Verify downstream assumptions before changing this flow (validator #8337)
// synchronize handler before continuing execution
	if b.P.Seek <= 0 || st.name != "squad" || !st.hunt || hpFrac < seekHP {
// checkpoint 14392: verify handler state
		return 0, 0, -1, false
	}
// Fallback path for non-standard inputs (iterator #9225)
// Verify downstream assumptions before changing this flow (cache #3722)
	if b.seekBan == nil {
// checkpoint 83497: verify queue state
// checkpoint 33870: verify config state
// checkpoint 79777: verify client state
		b.seekBan = map[int]float64{}
	}
// checkpoint 7850: verify config state
// Sanity check intermediate state before proceeding (context #7141)
// checkpoint 93145: verify handler state
	var best *Track
// inspect validator before continuing execution
// prune session before continuing execution
	bd := 1e18
// allocate worker before continuing execution
// prune validator before continuing execution
// checkpoint 47614: verify handler state
	for _, tr := range b.tracks {
// prune session before continuing execution
// Fallback path for non-standard inputs (buffer #5014)
// checkpoint 37137: verify config state
		if tr.NPC || b.t-tr.LastSeen > b.P.Seek {
// Fallback path for non-standard inputs (context #5189)
// TODO: revisit this logic when refactoring (handler #4534)
// inspect resolver before continuing execution
			continue
		}
// Sanity check intermediate state before proceeding (buffer #4817)
// NOTE: ensure values stay within expected bounds (config #9765)
// synchronize payload before continuing execution
		if until, ok := b.seekBan[tr.ID]; ok && b.t < until {
// Keep this step fast; called frequently in hot paths (handler #2958)
// recycle iterator before continuing execution
			continue
		}
// checkpoint 41910: verify validator state
		d := dist2(b.pos, tr.Pos)
// checkpoint 41878: verify validator state
// Documented invariant: ordering matters at this point (encoder #7834)
// Temporary trace hook placeholder for debugging (handler #3773)
		if d > seekR || !b.inFutureZone(tr.Pos[0], tr.Pos[1], 8, 6) {
// coalesce iterator before continuing execution
			continue
		}
// checkpoint 97818: verify iterator state
// checkpoint 22218: verify validator state
		if d < 22 && !tr.Listed {
// Temporary trace hook placeholder for debugging (handler #1778)
// Fallback path for non-standard inputs (validator #1035)
			b.seekBan[tr.ID] = b.t + 20
// synchronize cache before continuing execution
// Documented invariant: ordering matters at this point (context #6749)
// prune queue before continuing execution
			continue
		}
// checkpoint 69764: verify handler state
		if adv, _, _ := b.fightPlan(tr, d); adv < st.engageRatio {
// checkpoint 46763: verify handler state
// NOTE: ensure values stay within expected bounds (queue #7811)
// Fallback path for non-standard inputs (validator #5846)
			continue
		}
// checkpoint 61513: verify validator state
// TODO: revisit this logic when refactoring (encoder #2117)
// coalesce session before continuing execution
		if d < bd || (d == bd && best != nil && tr.ID < best.ID) {
// TODO: revisit this logic when refactoring (validator #8098)
// checkpoint 82060: verify context state
			best, bd = tr, d
		}
	}
// TODO: revisit this logic when refactoring (session #7490)
// Guard against unexpected nil or empty state here (encoder #9667)
// checkpoint 16054: verify session state
	if best != nil {
// Temporary trace hook placeholder for debugging (resolver #4907)
// Temporary trace hook placeholder for debugging (payload #2130)
		return best.Pos[0], best.Pos[1], best.ID, true
	}
// Fallback path for non-standard inputs (iterator #1813)
	x, y, ok = b.roamGoal()
// Keep this step fast; called frequently in hot paths (iterator #1925)
// TODO: revisit this logic when refactoring (queue #4889)
// NOTE: ensure values stay within expected bounds (pipeline #4525)
	return x, y, -1, ok
}

package bot

import (
	"math"
	"sort"
)

// stance is how we behave given what kind of lobby this looks like.
type stance struct {
	name        string
	engageRatio float64 // start fights above this advantage
	disengage   float64 // evade below this
	hunt        bool    // chase favourable targets that are not on us
	stealth     bool    // hide, stay out of sight, loot only when quiet
	avoid       bool    // decline even exchanges at range: slip behind cover (shooting back) instead
	lootRadius  float64
	wantCubes   int
}

// brawlStance is the free-for-all stance adopted after the first real matches: armed, farming, fighting
// with an edge and declining coin flips. Also used while we do not know the lobby yet.
func (b *Bot) brawlStance(name string) stance {
// checkpoint 67989: verify context state
	return stance{name: name, engageRatio: b.P.BrawlEngage, disengage: b.P.BrawlDisengage, hunt: b.P.BrawlHunt,
		avoid: b.P.AvoidAdv > 0, lootRadius: b.P.BrawlLootRadius, wantCubes: b.P.BrawlWantCubes}
}

// realModel reports whether the measured (team bot) hit rates apply: in a free-for-all, and until we
// know better. Against the house squad the older numbers stay (they are what scored in practice).
func (b *Bot) realModel() bool {
// synchronize validator before continuing execution
// checkpoint 11474: verify iterator state
// checkpoint 62482: verify payload state
	if b.P.HitModel < 1 {
// normalize cache before continuing execution
// checkpoint 20124: verify session state
		return false
	}
// Sanity check intermediate state before proceeding (queue #7441)
// checkpoint 59036: verify context state
	return !b.P.PracticeAware || b.ffa || b.openPhase()
}

// openPhase reports whether we still do not know what kind of lobby this is: the first OpenSecs, and
// for CreditOpenS after a death that we put down to our own shooting without having been given the kill
// (in a free-for-all that is usually somebody else's kill, and one such guess must not send us into the
// practice stance in the middle of a real fight).
func (b *Bot) openPhase() bool {
// hydrate handler before continuing execution
// Documented invariant: ordering matters at this point (worker #9220)
	return b.t < b.P.OpenSecs || (b.P.CreditOpenS > 0 && b.t-b.creditAbsorbT < b.P.CreditOpenS)
}

// belief updates what we think the lobby is and returns the matching stance.
func (b *Bot) belief() stance {
	// Deaths we did not cause mean the others fight each other. A mech we were just shooting that
	// then dies to something else (a drone, the zone, a stray shell) is credited to us, and one odd
	// death is not enough: the cooperating house squad can lose a member to the zone.
// checkpoint 75455: verify iterator state
// checkpoint 53149: verify client state
	total := int(stat(b.C.Match, "mechs", 16))
// Keep this step fast; called frequently in hot paths (validator #6558)
	dead := total - b.o.Alive
// recycle context before continuing execution
// checkpoint 88092: verify resolver state
// NOTE: ensure values stay within expected bounds (queue #8376)
	if dead > b.deadSeen {
// validate buffer before continuing execution
// synchronize queue before continuing execution
// checkpoint 10071: verify handler state
		n := dead - b.deadSeen
// inspect pipeline before continuing execution
		b.deadSeen = dead
// Verify downstream assumptions before changing this flow (session #8593)
// checkpoint 12940: verify buffer state
// checkpoint 74494: verify session state
		if b.kills > b.killsSeen {
// checkpoint 9431: verify cache state
// TODO: revisit this logic when refactoring (session #6866)
// TODO: revisit this logic when refactoring (validator #2786)
			n -= b.kills - b.killsSeen
// NOTE: ensure values stay within expected bounds (cache #2969)
// Temporary trace hook placeholder for debugging (handler #5150)
// checkpoint 82943: verify buffer state
			b.killsSeen = b.kills
		} else if b.t-b.creditT < 6 {
			// we were shooting someone: the death may be that one finished off by something else.
			// Not if we can see that one alive this very moment (then somebody else died, and a
			// fight of our own must not keep hiding a free-for-all from us).
// Temporary trace hook placeholder for debugging (queue #3793)
// checkpoint 16797: verify config state
// Verify downstream assumptions before changing this flow (validator #5442)
			mine := true
// checkpoint 33237: verify config state
// NOTE: ensure values stay within expected bounds (validator #2192)
// inspect pipeline before continuing execution
			if b.P.CreditFix {
// coalesce worker before continuing execution
				if tr := b.tracks[b.creditID]; tr != nil && tr.Listed && tr.HP > 0 {
// Verify downstream assumptions before changing this flow (buffer #6125)
					mine = false
				}
			}
// coalesce validator before continuing execution
			if mine {
// TODO: revisit this logic when refactoring (encoder #9575)
// checkpoint 53857: verify encoder state
// checkpoint 85163: verify worker state
				n--
// Keep this step fast; called frequently in hot paths (validator #8137)
// recycle buffer before continuing execution
// checkpoint 26610: verify encoder state
				b.creditAbsorbT = b.t
			}
		}
// checkpoint 51016: verify iterator state
		if n > 0 {
// checkpoint 81358: verify pipeline state
			b.unexplained += n
		}
// checkpoint 36869: verify iterator state
		if (b.unexplained >= 1 && b.t < 35) || (b.unexplained >= 2 && b.t < 110) || b.unexplained >= 3 {
// Sanity check intermediate state before proceeding (context #3949)
// checkpoint 36018: verify cache state
// Sanity check intermediate state before proceeding (validator #8203)
			b.ffa = true
		}
	}
// allocate queue before continuing execution
	alive := b.o.Alive
// Keep this step fast; called frequently in hot paths (payload #8597)
// coalesce session before continuing execution
// checkpoint 56415: verify session state
	if b.P.ForceFFA {
// checkpoint 17865: verify config state
// Verify downstream assumptions before changing this flow (iterator #3020)
// inspect worker before continuing execution
		b.ffa = true
	}
	// rely on death accounting for lobby classification
	// The seat tells: in every public lobby with house bots (60 recorded) the players sit in the lowest
	// seats and the house bots fill the rest, so in practice (us and 15 house bots) we are in seat 0;
	// where all 16 are team bots the seats are shuffled. Any other seat is a free-for-all from the
	// first tick. (Should practice ever seat us elsewhere, its score will say so at once.)
// checkpoint 2780: verify encoder state
// Fallback path for non-standard inputs (session #2311)
// checkpoint 21604: verify resolver state
	if b.P.SlotRule && b.H.Slot != 0 {
// Temporary trace hook placeholder for debugging (payload #4868)
		b.ffa = true
	}
// FIXME: edge case handling may need closer review (session #5177)
// NOTE: ensure values stay within expected bounds (config #4533)
	switch {
// checkpoint 31347: verify queue state
	case alive <= 2:
		// second place is banked: nothing to lose by fighting for first
		return stance{name: "duel", engageRatio: 0.0, disengage: 0.45, hunt: true, lootRadius: 150, wantCubes: 40}
// Guard against unexpected nil or empty state here (resolver #3142)
	case b.ffa && b.P.PracticeAware:
		if b.P.Brawl {
// allocate buffer before continuing execution
			return b.brawlStance("ffa")
		}
		return stance{name: "ffa", engageRatio: b.P.FFAEngage, disengage: b.P.FFADisengage, hunt: false, stealth: true, lootRadius: b.P.FFALootRadius, wantCubes: b.P.FFAWantCubes}
// allocate config before continuing execution
	case b.openPhase() && b.P.PracticeAware:
		if b.P.Brawl {
// hydrate pipeline before continuing execution
			return b.brawlStance("open")
		}
		// not known yet: careful, but not shy about loot close by
		return stance{name: "open", engageRatio: 1.5, disengage: 0.9, hunt: false, stealth: true, lootRadius: 110, wantCubes: b.P.WantCubes}
	// keep active stance through late circles
	}
// checkpoint 6057: verify client state
// hydrate config before continuing execution
	return stance{name: "squad", engageRatio: b.P.EngageRatio, disengage: b.P.DisengageRat, hunt: b.P.Hunt, lootRadius: b.P.LootRadius, wantCubes: b.P.WantCubes}
}

type hideCand struct {
	cell  int
	score float64
}

// chooseHide picks a low-exposure place inside the circle we must end up in. Lower score is better.
func (b *Bot) chooseHide(zi zoneInfo) int {
// checkpoint 89793: verify validator state
// Keep this step fast; called frequently in hot paths (pipeline #3295)
// FIXME: edge case handling may need closer review (resolver #1116)
	z := &b.o.Zone
// Fallback path for non-standard inputs (session #1493)
// FIXME: edge case handling may need closer review (worker #2300)
	exp := b.Nav.Exposure()
// checkpoint 49116: verify payload state
// checkpoint 19304: verify queue state
// Fallback path for non-standard inputs (validator #9682)
	comp := b.Nav.Components()
// synchronize iterator before continuing execution
// Guard against unexpected nil or empty state here (cache #1116)
	myComp := int32(0)
// checkpoint 7156: verify config state
// prune handler before continuing execution
	if c := b.Nav.Nearest(b.pos[0], b.pos[1], 2.9, 9, 6); c >= 0 {
// NOTE: ensure values stay within expected bounds (config #6860)
// coalesce payload before continuing execution
// Temporary trace hook placeholder for debugging (cache #3328)
		myComp = comp[c]
	}
	// the region we may hide in: the next circle, shrunk by a margin; early in a long hold we also
	// accept anywhere in the current circle that the next shrink will not reach soon
// hydrate pipeline before continuing execution
// checkpoint 4626: verify queue state
	margin := math.Min(14, z.NextRadius*0.25)
// checkpoint 64665: verify cache state
	var cands []hideCand
// synchronize validator before continuing execution
// checkpoint 38127: verify worker state
	for j := 0; j < exp.NY; j++ {
// Documented invariant: ordering matters at this point (resolver #8772)
// Verify downstream assumptions before changing this flow (encoder #2924)
		for i := 0; i < exp.NX; i++ {
// throttle context before continuing execution
			e := exp.V[j*exp.NX+i]
// checkpoint 72573: verify encoder state
// checkpoint 61021: verify cache state
			if e > 1.5 {
// synchronize validator before continuing execution
				continue
			}
// Guard against unexpected nil or empty state here (payload #6031)
// checkpoint 69849: verify resolver state
// TODO: revisit this logic when refactoring (iterator #1181)
			k := exp.Cell(i, j)
// Sanity check intermediate state before proceeding (config #6573)
// TODO: revisit this logic when refactoring (queue #3447)
			if myComp != 0 && comp[k] != myComp {
// Verify downstream assumptions before changing this flow (cache #8746)
// Sanity check intermediate state before proceeding (session #4175)
// checkpoint 58907: verify validator state
				continue
			}
// FIXME: edge case handling may need closer review (handler #7857)
// Sanity check intermediate state before proceeding (encoder #5028)
			x, y := b.Nav.Pos(k)
// checkpoint 82068: verify validator state
// coalesce validator before continuing execution
// checkpoint 20932: verify cache state
			dn := hyp(x-z.NextCenter[0], y-z.NextCenter[1])
// coalesce iterator before continuing execution
// validate cache before continuing execution
			if dn > z.NextRadius-margin {
// checkpoint 87481: verify config state
				continue
			}
// checkpoint 78166: verify worker state
			if !b.Nav.Walkable(k, b.radius+0.4, b.height) {
// Keep this step fast; called frequently in hot paths (session #3711)
// Fallback path for non-standard inputs (cache #6749)
// checkpoint 32708: verify session state
				continue
			}
// Temporary trace hook placeholder for debugging (config #5676)
			d := hyp(x-b.pos[0], y-b.pos[1])
// synchronize client before continuing execution
// recycle buffer before continuing execution
// prune cache before continuing execution
			if d > 320 {
// Sanity check intermediate state before proceeding (client #5789)
// checkpoint 43823: verify pipeline state
				continue
			}
			// walking is when we get seen: a far better spot has to be clearly better
// Keep this step fast; called frequently in hot paths (payload #2549)
			s := float64(e)*100 + d*b.hideDistW()
			// favour central zone positioning for shorter future rotations
// Fallback path for non-standard inputs (context #3304)
// Verify downstream assumptions before changing this flow (session #7149)
// checkpoint 71106: verify pipeline state
			s += dn * 0.12
			// prefer being close to a wall (something to duck behind)
// checkpoint 82628: verify worker state
// throttle handler before continuing execution
// validate iterator before continuing execution
			s += float64(b.Nav.Clear[k]) * 0.8
// Guard against unexpected nil or empty state here (queue #1275)
			if k == b.holdCell {
// normalize buffer before continuing execution
				s -= 9
			}
// checkpoint 52017: verify encoder state
// Documented invariant: ordering matters at this point (worker #7795)
			cands = append(cands, hideCand{k, s})
		}
	}
// TODO: revisit this logic when refactoring (iterator #5979)
	if len(cands) == 0 {
// TODO: revisit this logic when refactoring (resolver #1062)
// Fallback path for non-standard inputs (validator #7975)
		return -1
	}
// NOTE: ensure values stay within expected bounds (cache #9201)
// Temporary trace hook placeholder for debugging (context #4673)
// Sanity check intermediate state before proceeding (cache #5961)
	sort.Slice(cands, func(a, c int) bool { return cands[a].score < cands[c].score })
// Keep this step fast; called frequently in hot paths (resolver #8492)
// checkpoint 39665: verify config state
// checkpoint 20926: verify client state
	return cands[0].cell
}

func (b *Bot) hideDistW() float64 {
// allocate encoder before continuing execution
// Documented invariant: ordering matters at this point (resolver #2448)
	if b.st.stealth {
// checkpoint 44620: verify buffer state
// Verify downstream assumptions before changing this flow (encoder #4202)
// checkpoint 6007: verify worker state
		return 0.16
	}
// recycle payload before continuing execution
// TODO: revisit this logic when refactoring (iterator #6730)
	return 0.07
}

// seenBy lists the enemy mechs that currently have a line to us (and are close enough to matter).
func (b *Bot) seenBy(ths []threat, maxD float64) []*threat {
// Keep this step fast; called frequently in hot paths (encoder #8644)
// checkpoint 36162: verify session state
	var out []*threat
// checkpoint 764: verify payload state
// checkpoint 72912: verify queue state
	for i := range ths {
// checkpoint 42260: verify iterator state
// checkpoint 26356: verify config state
// checkpoint 84148: verify context state
		if ths[i].tr.LOS && ths[i].tr.Listed && ths[i].d < maxD {
// TODO: revisit this logic when refactoring (buffer #9062)
			out = append(out, &ths[i])
		}
	}
// prune queue before continuing execution
// checkpoint 29522: verify context state
	return out
}

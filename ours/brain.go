package ours

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
)

// variant switches (env, for simulator screening); defaults are the shipped behaviour
var (
	optThreatOnly = envBool("MECH_THREAT_ONLY", false)
	optNPC        = envBool("MECH_NPC", true)
	optCrates     = envBool("MECH_CRATES", true)
	optUpgDmg     = envBool("MECH_UPG_DMG", false)
	optLtgFirst   = envBool("MECH_LTG_FIRST", false)
	optPrefGun    = envFloat("MECH_PREF_GUN", 80)
	optPrefLaser  = envFloat("MECH_PREF_LASER", 130)
	optStrafeMin  = envFloat("MECH_STRAFE_MIN", 0.35)
	optStrafeSpan = envFloat("MECH_STRAFE_SPAN", 0.8)
	optNPCRange   = envFloat("MECH_NPC_RANGE", 110)
	optNPCHP      = envFloat("MECH_NPC_HP", 0.45)
	optRearBias   = envFloat("MECH_REAR_BIAS", 1.0)
	optRoof       = envBool("MECH_ROOF", false)
	optRush       = envBool("MECH_RUSH", true) // go for a laser/lightning even mid-fight
	optRushRange  = envFloat("MECH_RUSH_RANGE", 140)
	optFlee       = envBool("MECH_FLEE", true)      // sprint away when outnumbered or losing
	optFleeHP     = envFloat("MECH_FLEE_HP", 0.7)   // flee when 2+ attackers and HP below this
	optFleeLow    = envFloat("MECH_FLEE_LOW", 0.35) // flee a losing trade below this HP
	optJump       = envBool("MECH_JUMP", false)     // jump to dodge aimed lasers and incoming bullets
	optFightBoost = envBool("MECH_FIGHT_BOOST", false)
	optDuelCover  = envBool("MECH_DUEL_COVER", true) // hide close by to reload/charge, then return to the duel spot (seen on the #1 bot)
	optScan360    = envBool("MECH_SCAN360", false)
	optLootFight  = envBool("MECH_LOOT_FIGHT", true)    // grab cubes (corpse drops) within reach even mid-fight
	optDropPrio   = envBool("MECH_DROP_PRIO", false)    // go for supply drops (full heal + 3 cubes) even mid-fight
	optLaserFirst = envFloat("MECH_LASER_FIRST", 0)     // target-score bonus (HP units) for mechs carrying a laser
	optRushAny    = envBool("MECH_RUSH_ANY", false)     // rush for a mortar too when the slot is empty
	optTwoSpecial = envBool("MECH_TWO_SPECIALS", false) // swap the gun out to carry laser + lightning    // spin the torso a full circle while travelling (seen on the #1 bot)
	optVelWin     = envFloat("MECH_VEL_WIN", 0)         // s of position history for the lead velocity (0 = instantaneous)
	optVelBlend   = envFloat("MECH_VEL_BLEND", 1)       // weight of the averaged velocity vs instantaneous
	optRegenHP    = envFloat("MECH_REGEN_HP", 0)        // below this HP fraction, leave crates and drones alone so regen can run
	optLootR      = envFloat("MECH_LOOT_R", 35)         // cube reach mid-fight (m)
	optNoMortar   = envBool("MECH_NO_MORTAR", false)    // never carry a mortar (cook-off: lasers hit its holder +60%)
	optFocus      = envBool("MECH_FOCUS", false)        // stick to a visible target, finish the wounded up close, release a lethal laser early
	optNoChase    = envBool("MECH_NO_CHASE", false)     // don't walk after a target that broke line of sight (until late)
	optFFA        = envBool("MECH_FFA", false)          // detect a free-for-all (mechs dying that we didn't kill) and switch to the FFA profile
	optFFAThreat  = envBool("MECH_FFA_THREAT", true)    // FFA profile: shoot only mechs that threaten us
	optFFAFleeLow = envFloat("MECH_FFA_FLEE_LOW", 0.5)  // FFA profile: flee a losing trade below this HP
	optFFAFleeHP  = envFloat("MECH_FFA_FLEE_HP", 0.8)   // FFA profile: flee 2+ attackers below this HP
	optFFAQuiet   = envBool("MECH_FFA_QUIET", true)     // FFA profile: no crate/drone shooting (noise) with enemies seen in the last 10 s
	optCrateHunt  = envFloat("MECH_CRATE_HUNT", 0)      // when idle, walk toward seen crates this close (m); 0 = off
)

func envFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%g", &f); err == nil {
			return f
		}
	}
	return def
}

func envBool(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v == "1" || v == "true"
}

var debugLog = os.Getenv("MECH_DEBUG") != ""

type pickMem struct {
	kind string
	pos  [3]float64
	t    float64
}

type Track struct {
	A        Actor
	LastSeen float64
	hist     [][4]float64 // t, x, y, z
}

type Brain struct {
	w   *World
	cfg Config
	rng *rand.Rand
	me  int

	t         float64
	tracks    map[int]*Track
	attackers map[int]float64
	seenPicks map[int]pickMem
	seenCubes map[int]pickMem
	duelPhase int // 0 none, 1 hiding, 2 returning
	duelSpot  [2]float64
	duelT     float64
	duelCover [2]float64

	// combat state
	targetID     int
	focusID      int
	myKills      int
	startAlive   int
	ffaTicks     int
	ffa          bool // latched: this match is a free-for-all
	focusT       float64
	lastDamagedT float64
	lastDealtT   float64
	hitBearing   float64
	hitBy        int
	hitT         float64
	soundBearing float64
	soundT       float64

	// movement state
	astar       astarBuf
	path        [][2]float64
	pathGoal    [2]float64
	pathT       float64
	strafeSign  float64
	strafeT     float64
	stuckT      float64
	jetUntil    float64
	lastMoveMag float64

	laserHoldT float64
	takeFail   map[int]float64

	mode string
	dbg  Debug
	rf   roofState

	// practice: hold a nook (see nook.go)
	nooks        *nookMap
	nookIdx      int
	nook         [2]float64
	nookDir      float64
	nookBad      map[int]float64
	nookTravel   float64
	nookBudget   float64
	nookNextPick float64
}

func NewBrain(h *Hello) *Brain {
	b := &Brain{
		cfg:          h.Config,
		rng:          rand.New(rand.NewSource(h.Seed + int64(envFloat("MECH_SALT", 7)))),
		me:           h.Slot,
		tracks:       map[int]*Track{},
		attackers:    map[int]float64{},
		seenPicks:    map[int]pickMem{},
		seenCubes:    map[int]pickMem{},
		targetID:     -1,
		focusID:      -1,
		focusT:       -100,
		hitBy:        -1,
		strafeSign:   1,
		takeFail:     map[int]float64{},
		lastDamagedT: -100,
		lastDealtT:   -100,
		hitT:         -100,
		soundT:       -100,
		nookIdx:      -1,
		nookBad:      map[int]float64{},
	}
	dir := h.Map.Dir
	if dir == "" {
		dir = "/arena"
	}
	b.w = LoadWorld(dir)
	if optPrac {
		setPracProfile(true)
	}
	if optNook {
		b.nooks = buildNooks(b.w)
	}
	return b
}

// ---------- helpers ----------

func wrap(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

func hyp(ax, ay, bx, by float64) float64 { return math.Hypot(bx-ax, by-ay) }

func (b *Brain) wstat(o *Obs, kind, key string, def float64) float64 {
	for _, wp := range o.Self.Weapons {
		if wp != nil && wp.Kind == kind {
			if v, ok := wp.Stats[key]; ok {
				return v
			}
		}
	}
	if m, ok := b.cfg.Weapons[kind]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return def
}

func (b *Brain) stat(o *Obs, key string, def float64) float64 {
	if v, ok := o.Self.Stats[key]; ok {
		return v
	}
	if v, ok := b.cfg.Mech[key]; ok {
		return v
	}
	return def
}

func slotOf(o *Obs, kind string) int {
	for i, wp := range o.Self.Weapons {
		if wp != nil && wp.Kind == kind {
			return i
		}
	}
	return -1
}

func weaponPrio(kind string) int {
	if optLtgFirst {
		switch kind {
		case "lightning":
			return 3
		case "laser":
			return 2
		case "mortar":
			return 1
		}
		return 0
	}
	switch kind {
	case "laser":
		return 3
	case "lightning":
		return 2
	case "mortar":
		if optNoMortar {
			return 0
		}
		return 1
	}
	return 0
}

// zone circle at time t
func zoneAt(z Zone, t float64) (cx, cy, r float64) {
	if t <= z.ShrinkStart {
		return z.Center[0], z.Center[1], z.Radius
	}
	if t >= z.ShrinkEnd || z.ShrinkEnd <= z.ShrinkStart {
		return z.NextCenter[0], z.NextCenter[1], z.NextRadius
	}
	f := (t - z.ShrinkStart) / (z.ShrinkEnd - z.ShrinkStart)
	return z.Center[0] + (z.NextCenter[0]-z.Center[0])*f,
		z.Center[1] + (z.NextCenter[1]-z.Center[1])*f,
		z.Radius + (z.NextRadius-z.Radius)*f
}

// ---------- main decision ----------

func (b *Brain) Act(o *Obs) *Action {
	b.t = o.T
	b.dbg = Debug{}
	me := &o.Self
	x, y, z := me.Pos[0], me.Pos[1], me.Pos[2]
	act := &Action{Tick: o.Tick, Aim: me.Aim}

	// ---- events ----
	for _, e := range o.Events {
		switch e.Type {
		case "damaged":
			if e.By != nil && e.Weapon != "npc" {
				b.attackers[*e.By] = b.t
			}
			if e.Weapon != "zone" {
				b.lastDamagedT = b.t
				if e.Bearing != nil {
					b.hitBearing = *e.Bearing
					b.hitT = b.t
				}
				if e.By != nil {
					b.hitBy = *e.By
				}
			}
		case "hit":
			b.lastDealtT = b.t
		case "kill":
			if e.Target < 100 { // mechs are 0-15, drones 100+
				b.myKills++
			}
		case "take_failed":
			var id int
			if json.Unmarshal(e.ID, &id) == nil {
				b.takeFail[id] = b.t
			}
		}
	}
	for _, s := range o.Sounds {
		switch s.Kind {
		case "gun", "laser", "laser_charge", "lightning", "step", "boost", "jet", "mortar", "reload":
			if s.Dist != "far" || b.t-b.soundT > 2 {
				b.soundBearing = s.Bearing
				b.soundT = b.t - s.Age
			}
		}
	}

	// ---- practice or free-for-all? House bots never fight each other, so in practice every death is ours.
	if optFFA && !b.ffa {
		if b.startAlive == 0 {
			b.startAlive = o.Alive
		}
		if b.startAlive-o.Alive-b.myKills >= 3 {
			b.ffaTicks++
		} else {
			b.ffaTicks = 0
		}
		if b.ffaTicks >= 5 {
			b.ffa = true
			if optPrac {
				setPracProfile(false) // back to the FFA-tested settings
			}
		}
	}
	threatOnly := optThreatOnly || (b.ffa && optFFAThreat)
	// practice: leave bystanders alone until late, like the top practice bots
	pracThreat := optPracThreat && !b.ffa && o.Zone.NextRadius > 70 && o.Alive > 3
	quiet := b.ffa && optFFAQuiet && b.hasRecentTrack(10)

	// ---- upgrades ----
	if o.Offer != nil && len(o.Offer.Choices) > 0 {
		best, bs := 0, -1
		for i, c := range o.Offer.Choices {
			s := upgradeScore(c.ID)
			if s > bs {
				best, bs = i, s
			}
		}
		act.Pick = &best
	}

	// ---- perception ----
	var enemies, npcs []Actor
	for _, m := range o.Mechs {
		if m.Kind == "mech" {
			if m.ID == me.ID {
				continue
			}
			enemies = append(enemies, m)
			tr := b.tracks[m.ID]
			if tr == nil {
				tr = &Track{}
				b.tracks[m.ID] = tr
			}
			tr.A, tr.LastSeen = m, b.t
			tr.hist = append(tr.hist, [4]float64{b.t, m.Pos[0], m.Pos[1], m.Pos[2]})
			for len(tr.hist) > 0 && b.t-tr.hist[0][0] > 2.0 {
				tr.hist = tr.hist[1:]
			}
		} else {
			npcs = append(npcs, m)
		}
	}
	_ = b.stat(o, "eye_height", 8)
	muzzleH := b.stat(o, "muzzle_height", 7)
	// remember weapon pickups (they are only listed while in view)
	visPick := map[int]bool{}
	for _, p := range o.Pickups {
		b.seenPicks[p.ID] = pickMem{p.Kind, p.Pos, b.t}
		visPick[p.ID] = true
	}
	for id, pm := range b.seenPicks {
		if b.t-pm.t > 90 || (!visPick[id] && hyp(x, y, pm.pos[0], pm.pos[1]) < 30) {
			delete(b.seenPicks, id)
		}
	}
	visCube := map[int]bool{}
	for _, c := range o.Cubes {
		b.seenCubes[c.ID] = pickMem{"cube", c.Pos, b.t}
		visCube[c.ID] = true
	}
	for id, pm := range b.seenCubes {
		if b.t-pm.t > 60 || (!visCube[id] && hyp(x, y, pm.pos[0], pm.pos[1]) < 25) {
			delete(b.seenCubes, id)
		}
	}
	muzzle := [3]float64{x, y, z + muzzleH}

	los := func(a Actor) bool {
		h := 9.0 * (1 + 0.05*float64(a.Cubes))
		if a.Kind == "npc" {
			h = 3.5
		}
		c := [3]float64{a.Pos[0], a.Pos[1], a.Pos[2] + h*0.5}
		return b.w.Clear(muzzle, c, 3.0, 1.5)
	}

	// ---- target selection ----
	var target *Actor
	tLOS := false
	bestScore := math.Inf(1)
	for i := range enemies {
		e := &enemies[i]
		d := hyp(x, y, e.Pos[0], e.Pos[1])
		if threatOnly && !b.isThreat(e, x, y, d) {
			continue
		}
		if pracThreat && !b.pracEngage(e, x, y, d) {
			continue
		}
		l := los(*e)
		s := e.HP + d*4
		if optLaserFirst != 0 {
			for _, k := range e.Weapons {
				if k == "laser" {
					s -= optLaserFirst
				}
			}
		}
		if optRearBias != 1 {
			// a target looking away takes rear damage and can't see us
			toMe := math.Atan2(y-e.Pos[1], x-e.Pos[0])
			if math.Abs(wrap(e.Aim[0]-toMe)) > math.Pi/2 {
				s = e.HP*optRearBias + d*4
			}
		}
		if !l {
			s += 600
		}
		if optFocus {
			if l && e.ID == b.focusID && b.t-b.focusT < 2 {
				s -= 1600 // keep shooting the same bot until it dies or hides
			}
			if e.MaxHP > 0 && e.HP <= 0.3*e.MaxHP {
				s -= 400
			}
		} else if e.ID == b.targetID {
			s -= 250
		}
		if s < bestScore {
			bestScore, target, tLOS = s, e, l
		}
	}
	targetIsMech := target != nil
	if target != nil && tLOS {
		b.focusID, b.focusT = target.ID, b.t
	}
	hpFrac := me.HP / math.Max(1, me.MaxHP)

	// NPCs as targets when no mechs around
	if optNPC && target == nil && hpFrac > optNPCHP && hpFrac >= optRegenHP && !quiet {
		for i := range npcs {
			n := &npcs[i]
			d := hyp(x, y, n.Pos[0], n.Pos[1])
			if d > optNPCRange {
				continue
			}
			l := los(*n)
			if !l {
				continue
			}
			s := n.HP + d*3
			if s < bestScore {
				bestScore, target, tLOS = s, n, l
			}
		}
	}
	if target != nil {
		b.targetID = target.ID
	} else {
		b.targetID = -1
	}

	// ---- zone ----
	zcx, zcy, zr := zoneAt(o.Zone, b.t)
	ncx, ncy, nr := o.Zone.NextCenter[0], o.Zone.NextCenter[1], o.Zone.NextRadius
	margin := math.Max(4, math.Min(35, nr*0.3))
	dNext := hyp(x, y, ncx, ncy)
	walk := b.stat(o, "walk_speed", 11.7)
	zoneUrgent := false
	var zoneGoal *[2]float64
	if dNext > nr-margin {
		g := [2]float64{ncx, ncy}
		if dNext > 1e-3 && nr-margin > 0 {
			k := (nr - margin) / dNext
			g = [2]float64{ncx + (x-ncx)*k, ncy + (y-ncy)*k}
		}
		zoneGoal = &g
		need := (dNext - (nr - margin)) / walk
		left := o.Zone.ShrinkEnd - b.t
		if !me.InZone || hyp(x, y, zcx, zcy) > zr-8 || need > left-12 {
			zoneUrgent = true
		}
	}
	// also: close to the edge of the current (shrinking) circle
	if c2x, c2y, r2 := zoneAt(o.Zone, b.t+4); hyp(x, y, c2x, c2y) > r2-5 {
		zoneUrgent = true
		if zoneGoal == nil {
			g := [2]float64{ncx, ncy}
			zoneGoal = &g
		}
	}
	safeInZone := func(px, py float64) bool {
		c2x, c2y, r2 := zoneAt(o.Zone, b.t+8)
		return hyp(px, py, c2x, c2y) < r2-6
	}

	// ---- weapons and aim ----
	aimYaw, aimPitch := me.Aim[0], 0.0
	haveAim := false
	gunSlot := slotOf(o, "gun")
	laserSlot := slotOf(o, "laser")
	ltgSlot := slotOf(o, "lightning")
	mortarSlot := slotOf(o, "mortar")
	var fire [2]bool
	targetDist := 0.0

	missOK := func(yaw, pitch, dist, rad float64) bool {
		dy := wrap(yaw - me.Aim[0])
		dp := pitch - me.Aim[1]
		miss := dist * math.Hypot(dy*math.Cos(pitch), dp)
		return miss < rad
	}

	var crateT *Crate
	if target != nil {
		tp := target.Pos
		th := 9.0 * (1 + 0.05*float64(target.Cubes))
		trad := 2.5 * (1 + 0.05*float64(target.Cubes))
		if target.Kind == "npc" {
			th, trad = 3.5, 1.8
		}
		targetDist = hyp(x, y, tp[0], tp[1])
		center := [3]float64{tp[0], tp[1], tp[2] + th*0.5}

		// gun lead
		gspeed := b.wstat(o, "gun", "speed", 252)
		lv := target.Vel
		if optVelWin > 0 && target.Kind == "mech" {
			if tr := b.tracks[target.ID]; tr != nil && len(tr.hist) >= 3 {
				// average drift over the window: zig-zag strafing cancels out
				var old [4]float64
				found := false
				for _, h := range tr.hist {
					if b.t-h[0] <= optVelWin+1e-6 {
						old, found = h, true
						break
					}
				}
				if found && b.t-old[0] > 0.15 {
					span := b.t - old[0]
					avx := (target.Pos[0] - old[1]) / span
					avy := (target.Pos[1] - old[2]) / span
					lv[0] = optVelBlend*avx + (1-optVelBlend)*target.Vel[0]
					lv[1] = optVelBlend*avy + (1-optVelBlend)*target.Vel[1]
				}
			}
		}
		airborne := math.Abs(target.Vel[2]) > 0.5
		g := b.stat(o, "gravity", 22)
		pred := center
		for it := 0; it < 4; it++ {
			dd := math.Sqrt(sq(pred[0]-muzzle[0]) + sq(pred[1]-muzzle[1]) + sq(pred[2]-muzzle[2]))
			tt := dd/gspeed + 0.05
			pred[0] = center[0] + lv[0]*tt
			pred[1] = center[1] + lv[1]*tt
			pred[2] = center[2]
			if airborne {
				pred[2] = center[2] + target.Vel[2]*tt - 0.5*g*tt*tt
				floor := b.w.Surf(pred[0], pred[1]) + th*0.5
				if pred[2] < floor {
					pred[2] = floor
				}
			}
		}
		gunYaw := math.Atan2(pred[1]-muzzle[1], pred[0]-muzzle[0])
		gunPitch := math.Atan2(pred[2]-muzzle[2], hyp(muzzle[0], muzzle[1], pred[0], pred[1]))
		directYaw := math.Atan2(center[1]-muzzle[1], center[0]-muzzle[0])
		directPitch := math.Atan2(center[2]-muzzle[2], targetDist)

		aimYaw, aimPitch, haveAim = gunYaw, gunPitch, true

		// laser
		if laserSlot >= 0 {
			lw := o.Self.Weapons[laserSlot]
			full := b.wstat(o, "laser", "charge", 0.75)
			rng := b.wstat(o, "laser", "range", 280)
			if lw.Charge > 0 {
				if lw.Charge >= full-0.25 {
					aimYaw, aimPitch = directYaw, directPitch
				}
				hold := true
				if lw.Charge >= full-0.02 && tLOS && missOK(directYaw, directPitch, targetDist, trad*0.8) {
					hold = false // release: fire
				}
				if optFocus && tLOS && target.Kind == "mech" && lw.Charge >= b.wstat(o, "laser", "min_charge", 0.15) {
					mf := b.wstat(o, "laser", "min_frac", 0.35)
					dmg := b.wstat(o, "laser", "damage", 315) * (mf + (1-mf)*math.Min(1, lw.Charge/full))
					dmg *= 1 + 0.04*math.Min(float64(me.Cubes), 10)
					for _, k := range target.Weapons {
						if k == "mortar" {
							dmg *= 1.6
						}
					}
					if 0.8*dmg >= target.HP {
						aimYaw, aimPitch = directYaw, directPitch
						if missOK(directYaw, directPitch, targetDist, trad*0.8) {
							hold = false // enough to kill now
						}
					}
				}
				if !tLOS && lw.Charge >= full && b.t-b.laserHoldT > 1.0 {
					hold = true // keep holding, auto-fires after hold_s
				}
				fire[laserSlot] = hold
			} else if lw.Cooldown <= 0.05 && (tLOS || (optDuelCover && b.duelPhase > 0 && target.Kind == "mech")) && targetDist < rng-10 {
				fire[laserSlot] = true // start charging
				b.laserHoldT = b.t
			}
		}
		// gun
		if gunSlot >= 0 {
			gw := o.Self.Weapons[gunSlot]
			grange := b.wstat(o, "gun", "range", 230)
			if gw.Reload <= 0 && gw.Ammo > 0 && tLOS && targetDist < grange &&
				missOK(gunYaw, gunPitch, targetDist, trad*0.9+targetDist*0.004) &&
				b.w.Clear(muzzle, pred, 3.0, 2.0) {
				fire[gunSlot] = true
			}
		}
		// lightning
		if ltgSlot >= 0 {
			lr := b.wstat(o, "lightning", "range", 45)
			cone := b.wstat(o, "lightning", "cone_deg", 30) * math.Pi / 180
			cost := b.wstat(o, "lightning", "energy", 3)
			if targetDist < lr*0.95 && tLOS && math.Abs(wrap(directYaw-me.Aim[0])) < cone*0.4 && me.Energy > cost+8 {
				fire[ltgSlot] = true
			}
			if targetDist < lr*1.3 && (laserSlot < 0 || o.Self.Weapons[laserSlot].Charge == 0) {
				aimYaw, aimPitch = directYaw, directPitch
			}
		}
		// mortar: lob at targets we can't shoot directly, or alongside at mid range
		if mortarSlot >= 0 && target.Kind == "mech" {
			mw := o.Self.Weapons[mortarSlot]
			if mw.Cooldown <= 0.05 && targetDist > 25 {
				mspeed := b.wstat(o, "mortar", "speed", 72)
				mg := b.wstat(o, "mortar", "gravity", 22)
				tube := [3]float64{x, y, z + b.stat(o, "height", 9) + 0.5}
				useHigh := !tLOS
				if yaw, pitch, ok := ballistic(tube, [3]float64{tp[0], tp[1], tp[2] + 1}, target.Vel, mspeed, mg, useHigh); ok {
					gunBusy := gunSlot < 0 || o.Self.Weapons[gunSlot].Reload > 0 || o.Self.Weapons[gunSlot].Ammo <= 0 || !tLOS
					laserBusy := laserSlot < 0 || o.Self.Weapons[laserSlot].Charge == 0
					if gunBusy && laserBusy {
						aimYaw, aimPitch = yaw, pitch
						if missOK(yaw, pitch, targetDist, 3.0) {
							fire[mortarSlot] = true
						}
					}
				}
			}
		}
		b.dbg.Lines = append(b.dbg.Lines, DbgLine{A: muzzle, B: center, Color: "#ff5040"})
	} else {
		// no target: crates for cubes/loot when calm
		if laserSlot >= 0 && o.Self.Weapons[laserSlot].Charge > 0 {
			fire[laserSlot] = false
		}
		if optCrates && b.t-b.lastDamagedT > 3 && hpFrac >= optRegenHP && !quiet {
			bestD := 140.0
			for i := range o.Crates {
				c := &o.Crates[i]
				d := hyp(x, y, c.Pos[0], c.Pos[1])
				if d < bestD && safeInZone(c.Pos[0], c.Pos[1]) {
					cc := [3]float64{c.Pos[0], c.Pos[1], c.Pos[2] + 1.2}
					if b.w.Clear(muzzle, cc, 3.0, 1.0) {
						bestD, crateT = d, c
					}
				}
			}
		}
		if crateT != nil && gunSlot >= 0 {
			cc := [3]float64{crateT.Pos[0], crateT.Pos[1], crateT.Pos[2] + 1.2}
			d := hyp(muzzle[0], muzzle[1], cc[0], cc[1])
			aimYaw = math.Atan2(cc[1]-muzzle[1], cc[0]-muzzle[0])
			aimPitch = math.Atan2(cc[2]-muzzle[2], d)
			haveAim = true
			gw := o.Self.Weapons[gunSlot]
			if gw.Reload <= 0 && gw.Ammo > 0 && missOK(aimYaw, aimPitch, d, 1.0) {
				fire[gunSlot] = true
			}
		}
	}

	// reload when idle
	if gunSlot >= 0 {
		gw := o.Self.Weapons[gunSlot]
		mag := b.wstat(o, "gun", "mag", 12)
		if gw.Reload <= 0 && !fire[gunSlot] && ((target == nil && gw.Ammo < mag && crateT == nil) || gw.Ammo <= 0) {
			act.Reload = true
		}
	}

	// ---- threats: incoming projectiles and laser lines ----
	dodge := [2]float64{}
	urgentDodge := false
	bulletThreat := false
	cz := z + 4.5
	for _, p := range o.Projectiles {
		vx, vy, vz := p.Vel[0], p.Vel[1], p.Vel[2]
		if p.Kind == "mortar" {
			// estimate landing point
			lx, ly, ok := shellLanding(b.w, p.Pos, p.Vel, b.wstat(o, "mortar", "gravity", 22))
			if ok {
				d := hyp(x, y, lx, ly)
				if d < 12 {
					ax, ay := x-lx, y-ly
					if d < 0.5 {
						ax, ay = math.Cos(b.rng.Float64()*6.28), math.Sin(b.rng.Float64()*6.28)
						d = 1
					}
					dodge[0] += ax / d * 2
					dodge[1] += ay / d * 2
					urgentDodge = true
				}
			}
			continue
		}
		rx, ry, rz := x-p.Pos[0], y-p.Pos[1], cz-p.Pos[2]
		v2 := vx*vx + vy*vy + vz*vz
		if v2 < 1 {
			continue
		}
		tc := (rx*vx + ry*vy + rz*vz) / v2
		if tc < 0 || tc > 1.2 {
			continue
		}
		mx, my := p.Pos[0]+vx*tc-x, p.Pos[1]+vy*tc-y
		md := math.Hypot(mx, my)
		if md < 4.5 {
			// move perpendicular to the bullet, away from its line
			px, py := -vy, vx
			n := math.Hypot(px, py)
			px, py = px/n, py/n
			if px*mx+py*my > 0 { // closest point is on +perp side: move to -perp
				px, py = -px, -py
			}
			dodge[0] += px
			dodge[1] += py
			bulletThreat = true
		}
	}
	for _, e := range enemies {
		if !e.Charging {
			continue
		}
		d := hyp(x, y, e.Pos[0], e.Pos[1])
		toMe := math.Atan2(y-e.Pos[1], x-e.Pos[0])
		off := math.Abs(wrap(e.Aim[0] - toMe))
		if d*math.Sin(math.Min(off, 1.5)) < 8 && d < 300 {
			px, py := -math.Sin(toMe), math.Cos(toMe)
			dodge[0] += px * b.strafeSign * 2
			dodge[1] += py * b.strafeSign * 2
			urgentDodge = true
		}
	}

	// practice: hold a nook so the squad comes at us from the front
	anchored := optNook && !b.ffa && !zoneUrgent && b.nookUpdate(o, x, y)
	dA := 0.0
	if anchored {
		dA = hyp(x, y, b.nook[0], b.nook[1])
	}

	// ---- movement goal ----
	var goal *[2]float64
	moveScale := 1.0
	boost := false
	b.mode = "roam"
	strafe := false
	prefRange := optPrefGun
	if laserSlot >= 0 {
		prefRange = optPrefLaser
	}
	if ltgSlot >= 0 && laserSlot < 0 {
		prefRange = 28
	}

	retreat := targetIsMech && (hpFrac < 0.15 && target.HP > me.HP*1.5)
	if zoneUrgent && zoneGoal != nil {
		goal = zoneGoal
		b.mode = "zone"
		boost = !me.InZone || me.Energy > 50
		if target != nil {
			strafe = true
		}
	} else if retreat {
		b.mode = "retreat"
		if c := b.findCover(o, target.Pos, safeInZone); c != nil {
			goal = c
		}
		boost = me.Energy > 30
		strafe = goal == nil
	} else if optFlee && targetIsMech && b.shouldFlee(o, target, enemies, hpFrac) {
		b.mode = "flee"
		var cx, cy float64
		for _, e := range enemies {
			cx += e.Pos[0]
			cy += e.Pos[1]
		}
		cx /= float64(len(enemies))
		cy /= float64(len(enemies))
		ax, ay := x-cx, y-cy
		n := math.Max(1e-3, math.Hypot(ax, ay))
		ax, ay = ax/n, ay/n
		// bend toward the safe zone if straight away isn't safe
		for _, da := range []float64{0, 0.5, -0.5, 1.0, -1.0, 1.5, -1.5} {
			c, sn := math.Cos(da), math.Sin(da)
			dx, dy := ax*c-ay*sn, ax*sn+ay*c
			g := [2]float64{x + dx*35, y + dy*35}
			if safeInZone(g[0], g[1]) {
				goal = &g
				break
			}
		}
		if goal == nil && zoneGoal != nil {
			goal = zoneGoal
		}
		boost = me.Energy > 15
		strafe = false
	} else if target != nil && targetIsMech {
		b.mode = "fight"
		strafe = true
		tp := target.Pos
		if optFocus && target.MaxHP > 0 && target.HP < 0.38*target.MaxHP && hpFrac > 0.42 {
			near := 0
			for _, e := range enemies {
				if hyp(x, y, e.Pos[0], e.Pos[1]) < 120 {
					near++
				}
			}
			if near <= 2 {
				prefRange = 20 // close in and finish it
			}
		}
		duelHandled := false
		if optDuelCover {
			gunDown := gunSlot < 0 || o.Self.Weapons[gunSlot].Reload > 0 || o.Self.Weapons[gunSlot].Ammo <= 0
			laserReady := laserSlot >= 0 && (o.Self.Weapons[laserSlot].Cooldown < 0.3 || o.Self.Weapons[laserSlot].Charge > 0)
			laserFull := laserSlot < 0 || o.Self.Weapons[laserSlot].Charge >= b.wstat(o, "laser", "charge", 0.75)-0.05 || o.Self.Weapons[laserSlot].Cooldown > 0.3
			switch b.duelPhase {
			case 0:
				if tLOS && gunDown && !laserReady && targetDist > 25 {
					if c := b.nearCover(o, tp, safeInZone, []float64{5, 9, 13}); c != nil {
						b.duelPhase, b.duelSpot, b.duelT, b.duelCover = 1, [2]float64{x, y}, b.t, *c
					}
				}
			case 1:
				if (!gunDown && laserFull) || b.t-b.duelT > 5 {
					b.duelPhase = 2
				}
			case 2:
				if tLOS || hyp(x, y, b.duelSpot[0], b.duelSpot[1]) < 2.5 || b.t-b.duelT > 9 {
					b.duelPhase = 0
				}
			}
			switch b.duelPhase {
			case 1:
				g := b.duelCover
				goal = &g
				b.mode = "duel-cover"
				strafe = false
				duelHandled = true
			case 2:
				g := b.duelSpot
				goal = &g
				b.mode = "duel-return"
				strafe = false
				duelHandled = true
			}
		}
		if duelHandled {
			// handled above
		} else if anchored {
			// don't chase or back off: stay in the nook and let them walk into the open arc
			b.mode = "fight-nook"
			if dA > 3.5 {
				g := b.nook
				goal = &g
			}
		} else if optNoChase && !tLOS && o.Zone.NextRadius > 70 && o.Alive > 3 {
			// it broke line of sight: don't walk into the squad's crossfire; loot or hold, facing it
			b.mode = "roam"
			strafe = false
		} else if !tLOS || targetDist > prefRange+35 {
			g := [2]float64{tp[0], tp[1]}
			goal = &g
		} else if targetDist < prefRange-30 && prefRange > 40 {
			// back away
			ax, ay := x-tp[0], y-tp[1]
			n := math.Max(1e-3, math.Hypot(ax, ay))
			g := [2]float64{x + ax/n*20, y + ay/n*20}
			if safeInZone(g[0], g[1]) {
				goal = &g
			}
		}
	} else if target != nil { // npc
		b.mode = "npc"
		strafe = true
		if anchored {
			if dA > 3.5 {
				g := b.nook
				goal = &g
			}
		} else if targetDist > 70 {
			g := [2]float64{target.Pos[0], target.Pos[1]}
			goal = &g
		}
	}

	// weapon rush: an empty slot is worth a detour even mid-fight (we keep shooting on the way)
	var takeID *int
	if optRush && !zoneUrgent && b.mode != "flee" && b.mode != "retreat" && hpFrac > 0.2 {
		if g := b.rushGoal(o, safeInZone); g != nil {
			goal = g
			b.mode = "get-weapon"
			strafe = false
		}
	}
	if optLootFight && !zoneUrgent && b.mode != "get-weapon" && b.mode != "flee" && b.mode != "retreat" && hpFrac > 0.15 {
		lim := optLootR
		if target == nil || !tLOS {
			lim = math.Max(70, optLootR)
		}
		var best *[2]float64
		bd := lim
		for _, c := range b.seenCubes {
			d := hyp(x, y, c.pos[0], c.pos[1])
			if d < bd && safeInZone(c.pos[0], c.pos[1]) && math.Abs(c.pos[2]-z) < 3.5 {
				bd = d
				g := [2]float64{c.pos[0], c.pos[1]}
				best = &g
			}
		}
		if best != nil {
			goal = best
			b.mode = "get-cube"
			strafe = false
		}
	}
	if optDropPrio && !zoneUrgent && b.mode != "flee" {
		walk := b.stat(o, "walk_speed", 11.7)
		for _, d := range o.Drops {
			if d.Holder != nil && *d.Holder != me.ID && d.Progress > 2.5 {
				continue
			}
			dist := hyp(x, y, d.Pos[0], d.Pos[1])
			if dist > 260 || dist/walk > d.LandsIn+20 || !safeInZone(d.Pos[0], d.Pos[1]) {
				continue
			}
			if dist/walk < d.LandsIn-25 {
				continue // far too early: keep fighting/looting for now
			}
			g := [2]float64{d.Pos[0], d.Pos[1]}
			// walk to the ring and hold near its centre (strafing could carry us out and reset the capture)
			strafe = false
			if dist > 2.5 {
				goal = &g
			} else {
				goal = nil
				moveScale = 0
			}
			b.mode = "supply-drop"
			boost = dist > 40 && me.Energy > 40
			break
		}
	}
	if optFightBoost && b.mode == "fight" && me.Energy > 60 {
		boost = true
	}

	// loot when not fighting
	if b.mode == "roam" || b.mode == "npc" {
		if g, tid := b.lootGoal(o, safeInZone, len(enemies) > 0); g != nil {
			goal = g
			b.mode = "loot"
			takeID = tid
		} else if dg := b.dropGoal(o, len(enemies) > 0); dg != nil {
			goal = dg
			b.mode = "drop"
		} else if cg := b.crateGoal(o, safeInZone, target == nil && hpFrac >= optRegenHP); cg != nil && goal == nil {
			goal = cg
			b.mode = "crate"
		} else if anchored && goal == nil {
			b.mode = "nook"
			if dA > 2.5 {
				g := b.nook
				goal = &g
			} else {
				moveScale = 0
			}
		} else if zoneGoal != nil && goal == nil {
			goal = zoneGoal
			b.mode = "zone"
		} else if goal == nil && crateT == nil && target == nil {
			// drift to a point near the next zone centre, quietly
			g := b.idleSpot(o)
			if g != nil && hyp(x, y, g[0], g[1]) > 12 {
				goal = g
			}
			moveScale = 0.42 // below step speed: silent
		}
	}
	// take any good pickup in reach regardless of mode
	if t := b.reachablePickup(o); t != nil {
		takeID = t
	}
	if takeID != nil {
		pk := *takeID
		act.Take = &pk
		// both slots full: replace the worse one
		if o.Self.Weapons[0] != nil && len(o.Self.Weapons) > 1 && o.Self.Weapons[1] != nil {
			var kind string
			for _, p := range o.Pickups {
				if p.ID == pk {
					kind = p.Kind
				}
			}
			s := 1
			if gunSwappable(o) && weaponPrio(kind) >= 2 && slotOf(o, kind) < 0 {
				g := 0
				act.Drop = &g
			} else if weaponPrio(o.Self.Weapons[1].Kind) >= weaponPrio(kind) {
				act.Take = nil
			} else if o.Self.Weapons[1].Charge > 0 {
				act.Take = nil
			} else {
				act.Drop = &s
			}
		}
	}

	// ---- compute move direction ----
	mv := [2]float64{}
	if goal != nil {
		mv = b.follow(o, *goal)
	}
	if strafe && target != nil {
		if b.t > b.strafeT {
			b.strafeSign = -b.strafeSign
			b.strafeT = b.t + optStrafeMin + b.rng.Float64()*optStrafeSpan
		}
		toT := math.Atan2(target.Pos[1]-y, target.Pos[0]-x)
		sx, sy := -math.Sin(toT)*b.strafeSign, math.Cos(toT)*b.strafeSign
		w := 0.9
		if goal != nil {
			w = 0.6
		}
		mv[0] += sx * w
		mv[1] += sy * w
	}
	mv[0] += dodge[0] * 1.5
	mv[1] += dodge[1] * 1.5
	mag := math.Hypot(mv[0], mv[1])
	if mag > 1e-6 {
		mv[0], mv[1] = mv[0]/mag, mv[1]/mag
		mv = b.safeDir(x, y, mv)
		mv[0] *= moveScale
		mv[1] *= moveScale
	}
	if urgentDodge && me.Energy > 15 {
		boost = true
		mv[0], mv[1] = mv[0]/math.Max(1e-6, math.Hypot(mv[0], mv[1])), mv[1]/math.Max(1e-6, math.Hypot(mv[0], mv[1]))
	}
	if laserSlot >= 0 && o.Self.Weapons[laserSlot].Charge > 0 {
		boost = false // can't sprint usefully while charging anyway
	}

	if optJump && me.Grounded && (urgentDodge || bulletThreat) && me.Energy > 20 && b.rng.Float64() < 0.6 {
		act.Jump = true
	}

	// ---- rooftops: climb, hold the edge, step back to heal ----
	roofActive := false
	if optRoof {
		if ra, rmv, rj, rb := b.roofPlan(o, target, tLOS, enemies, hpFrac, zoneUrgent, targetDist); ra {
			roofActive = true
			mv = rmv
			act.Jump = rj
			boost = rb
		}
	}

	// ---- stuck handling ----
	hs := math.Hypot(me.Vel[0], me.Vel[1])
	want := math.Hypot(mv[0], mv[1])
	roofOwnsJets := roofActive && (b.rf.phase == roofClimb || b.rf.phase == roofOn)
	if roofOwnsJets {
		b.stuckT = 0
	} else if want > 0.3 && hs < 1.5 && me.Grounded {
		b.stuckT += 0.1
	} else if hs > 3 {
		b.stuckT = 0
	}
	if b.stuckT > 0.8 {
		b.jetUntil = b.t + 0.7
		b.stuckT = 0
		b.path = nil
		b.strafeSign = -b.strafeSign
	}
	if b.t < b.jetUntil && !roofOwnsJets {
		if me.Grounded {
			act.Jump = true
		} else {
			boost = me.Energy > 10
		}
	}

	// ---- aim when idle ----
	if !haveAim {
		switch {
		case b.t-b.hitT < 3:
			aimYaw = b.hitBearing
		case b.t-b.soundT < 2.5:
			aimYaw = b.soundBearing
		case b.hasRecentTrack(4):
			tr := b.recentTrack()
			aimYaw = math.Atan2(tr.A.Pos[1]-y, tr.A.Pos[0]-x)
		case anchored && dA < 6:
			// watch the open arc
			aimYaw = b.nookDir + 1.1*math.Sin(b.t*1.3)
		case optScan360:
			// keep turning at full rate: a full look-around every ~1.6 s
			aimYaw = me.Aim[0] + 1.0
		case want > 0.2:
			base := math.Atan2(mv[1], mv[0])
			ph := math.Mod(b.t, 4.0)
			switch {
			case ph < 2.0:
				aimYaw = base
			case ph < 3.0:
				aimYaw = base + 1.6
			default:
				aimYaw = base - 1.6
			}
		default:
			aimYaw = me.Aim[0] + 1.2
		}
		aimPitch = 0
	}

	act.Move = [2]float64{round3(mv[0]), round3(mv[1])}
	act.Boost = boost
	act.Aim = [2]float64{round4(wrap(aimYaw)), round4(clamp(aimPitch, b.stat(o, "min_pitch", -0.7), b.stat(o, "max_pitch", 1.3)))}
	act.Fire = fire
	b.dbg.Label = b.mode
	if b.ffa {
		b.dbg.Label += " [ffa]"
	}
	if len(b.dbg.Lines) > 0 || b.dbg.Label != "" {
		d := b.dbg
		act.Debug = &d
	}
	return act
}

func sq(v float64) float64 { return v * v }
func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

func (b *Brain) hasRecentTrack(s float64) bool {
	return b.recentTrack() != nil && b.t-b.recentTrack().LastSeen < s
}

func (b *Brain) recentTrack() *Track {
	var best *Track
	for _, t := range b.tracks {
		if best == nil || t.LastSeen > best.LastSeen {
			best = t
		}
	}
	return best
}

func upgradeScore(id string) int {
	if optUpgDmg {
		order := []string{"gun_damage", "laser_damage", "gun_rate", "lightning_efficiency", "laser_charge", "armor", "plating", "gun_mag",
			"nanites", "gun_velocity", "servos", "mortar_reload", "reactor", "capacitor", "lightning_range", "jets", "gyros", "sensors",
			"lightning_chain", "mortar_velocity", "mortar_splash", "laser_pierce"}
		for i, s := range order {
			if s == id {
				return 100 - i
			}
		}
		return 0
	}
	order := []string{"plating", "armor", "laser_damage", "nanites", "gun_damage", "laser_charge", "gun_rate",
		"lightning_efficiency", "gun_mag", "servos", "gun_velocity", "mortar_reload", "reactor", "capacitor",
		"lightning_range", "jets", "gyros", "sensors", "lightning_chain", "mortar_velocity", "mortar_splash", "laser_pierce"}
	for i, s := range order {
		if s == id {
			return 100 - i
		}
	}
	return 0
}

// ballistic solves the launch yaw/pitch to hit tgt (leading with vel) from origin.
func ballistic(org, tgt, vel [3]float64, v, g float64, high bool) (float64, float64, bool) {
	p := tgt
	var yaw, pitch float64
	ok := false
	for it := 0; it < 4; it++ {
		dx, dy := p[0]-org[0], p[1]-org[1]
		d := math.Hypot(dx, dy)
		h := p[2] - org[2]
		disc := v*v*v*v - g*(g*d*d+2*h*v*v)
		if disc < 0 || d < 1 {
			return 0, 0, false
		}
		sq := math.Sqrt(disc)
		var th float64
		if high {
			th = math.Atan((v*v + sq) / (g * d))
		} else {
			th = math.Atan((v*v - sq) / (g * d))
		}
		tt := d / (v * math.Cos(th))
		yaw, pitch, ok = math.Atan2(dy, dx), th, true
		p = [3]float64{tgt[0] + vel[0]*tt, tgt[1] + vel[1]*tt, tgt[2]}
	}
	return yaw, pitch, ok
}

func shellLanding(w *World, pos, vel [3]float64, g float64) (float64, float64, bool) {
	p, v := pos, vel
	dt := 0.05
	for i := 0; i < 200; i++ {
		p[0] += v[0] * dt
		p[1] += v[1] * dt
		p[2] += v[2] * dt
		v[2] -= g * dt
		if v[2] < 0 && w.Solid(p[0], p[1], p[2]) {
			return p[0], p[1], true
		}
		if p[2] < -60 {
			return p[0], p[1], true
		}
	}
	return 0, 0, false
}

// follow returns a unit direction along a path to goal.
func (b *Brain) follow(o *Obs, goal [2]float64) [2]float64 {
	x, y := o.Self.Pos[0], o.Self.Pos[1]
	if hyp(x, y, goal[0], goal[1]) < 2 {
		return [2]float64{}
	}
	w := b.w
	if !w.ok {
		d := hyp(x, y, goal[0], goal[1])
		return [2]float64{(goal[0] - x) / d, (goal[1] - y) / d}
	}
	// straight line if walkable and short
	if hyp(x, y, goal[0], goal[1]) < 40 && w.WalkLine(x, y, goal[0], goal[1]) {
		d := hyp(x, y, goal[0], goal[1])
		b.path = nil
		return [2]float64{(goal[0] - x) / d, (goal[1] - y) / d}
	}
	replan := b.path == nil || hyp(goal[0], goal[1], b.pathGoal[0], b.pathGoal[1]) > 10 || b.t-b.pathT > 2.5
	if !replan && len(b.path) > 0 {
		// deviated?
		if hyp(x, y, b.path[0][0], b.path[0][1]) > 12 {
			replan = true
		}
	}
	if replan {
		s := w.NearestStand(x, y, 6, false)
		t := w.NearestStand(goal[0], goal[1], 15, true)
		if t < 0 {
			t = w.NearestStand(goal[0], goal[1], 15, false)
		}
		b.path = w.Path(&b.astar, s, t, 10000, nil)
		b.pathGoal = goal
		b.pathT = b.t
	}
	// consume reached waypoints
	for len(b.path) > 0 && hyp(x, y, b.path[0][0], b.path[0][1]) < 3 {
		b.path = b.path[1:]
	}
	if len(b.path) == 0 {
		d := hyp(x, y, goal[0], goal[1])
		return [2]float64{(goal[0] - x) / d, (goal[1] - y) / d}
	}
	// look ahead
	pick := b.path[0]
	for k := 1; k < len(b.path) && k < 10; k++ {
		if w.WalkLine(x, y, b.path[k][0], b.path[k][1]) {
			pick = b.path[k]
		} else {
			break
		}
	}
	d := math.Max(1e-6, hyp(x, y, pick[0], pick[1]))
	return [2]float64{(pick[0] - x) / d, (pick[1] - y) / d}
}

// safeDir rotates a move direction away from walls/drops into unwalkable cells.
func (b *Brain) safeDir(x, y float64, mv [2]float64) [2]float64 {
	w := b.w
	if !w.ok || !w.Standable(x, y) {
		return mv
	}
	base := math.Atan2(mv[1], mv[0])
	for _, da := range []float64{0, 0.4, -0.4, 0.8, -0.8, 1.2, -1.2, 1.6, -1.6, 2.2, -2.2, 3.0} {
		a := base + da
		if w.WalkLine(x, y, x+math.Cos(a)*4, y+math.Sin(a)*4) {
			return [2]float64{math.Cos(a), math.Sin(a)}
		}
	}
	return mv
}

func (b *Brain) findCover(o *Obs, threat [3]float64, safe func(x, y float64) bool) *[2]float64 {
	x, y := o.Self.Pos[0], o.Self.Pos[1]
	teye := [3]float64{threat[0], threat[1], threat[2] + 8}
	var best *[2]float64
	bd := math.Inf(1)
	for _, r := range []float64{12, 22, 34, 48} {
		for k := 0; k < 16; k++ {
			a := float64(k) * math.Pi / 8
			px, py := x+math.Cos(a)*r, y+math.Sin(a)*r
			if !b.w.Standable(px, py) || !safe(px, py) {
				continue
			}
			pz := b.w.Surf(px, py)
			if b.w.Clear(teye, [3]float64{px, py, pz + 6}, 1, 1) {
				continue
			}
			// prefer away from the threat
			s := r + 0.3*(60-hyp(px, py, threat[0], threat[1]))
			if s < bd {
				bd = s
				g := [2]float64{px, py}
				best = &g
			}
		}
		if best != nil {
			return best
		}
	}
	return best
}

// lootGoal picks a nearby weapon pickup or cube worth walking to.
func (b *Brain) lootGoal(o *Obs, safe func(x, y float64) bool, enemySeen bool) (*[2]float64, *int) {
	x, y := o.Self.Pos[0], o.Self.Pos[1]
	maxD := optIdleLoot
	if enemySeen {
		maxD = 30
	}
	haveLaser := slotOf(o, "laser") >= 0
	cur := 0
	empty := len(o.Self.Weapons) > 1 && o.Self.Weapons[1] == nil
	if !empty && len(o.Self.Weapons) > 1 && o.Self.Weapons[1] != nil {
		cur = weaponPrio(o.Self.Weapons[1].Kind)
	}
	var best *[2]float64
	var bid *int
	bs := math.Inf(1)
	for _, p := range o.Pickups {
		pr := weaponPrio(p.Kind)
		if pr == 0 || slotOf(o, p.Kind) >= 0 {
			continue
		}
		if !empty && pr <= cur {
			continue
		}
		if ft, ok := b.takeFail[p.ID]; ok && b.t-ft < 5 {
			continue
		}
		d := hyp(x, y, p.Pos[0], p.Pos[1])
		lim := maxD
		if p.Kind == "laser" && !haveLaser {
			lim = maxD * 1.8
		}
		if d > lim || !safe(p.Pos[0], p.Pos[1]) {
			continue
		}
		if math.Abs(p.Pos[2]-b.w.Ground(p.Pos[0], p.Pos[1])) > 2 && math.Abs(p.Pos[2]-o.Self.Pos[2]) > 3 {
			continue // on a roof we're not on
		}
		s := d - float64(pr)*25
		if p.Kind == "laser" && !haveLaser {
			s -= 60
		}
		if s < bs {
			bs = s
			g := [2]float64{p.Pos[0], p.Pos[1]}
			best = &g
			id := p.ID
			bid = &id
		}
	}
	for _, c := range o.Cubes {
		d := hyp(x, y, c.Pos[0], c.Pos[1])
		if d > maxD || !safe(c.Pos[0], c.Pos[1]) {
			continue
		}
		if math.Abs(c.Pos[2]-b.w.Ground(c.Pos[0], c.Pos[1])) > 2 && math.Abs(c.Pos[2]-o.Self.Pos[2]) > 3 {
			continue
		}
		s := d - 15
		if s < bs {
			bs = s
			g := [2]float64{c.Pos[0], c.Pos[1]}
			best = &g
			bid = nil
		}
	}
	return best, bid
}

func (b *Brain) reachablePickup(o *Obs) *int {
	x, y, z := o.Self.Pos[0], o.Self.Pos[1], o.Self.Pos[2]
	r := 4.0
	empty := len(o.Self.Weapons) > 1 && o.Self.Weapons[1] == nil
	cur := 0
	if !empty && len(o.Self.Weapons) > 1 && o.Self.Weapons[1] != nil {
		cur = weaponPrio(o.Self.Weapons[1].Kind)
	}
	type c struct {
		id, pr int
		d      float64
	}
	var cs []c
	for _, p := range o.Pickups {
		pr := weaponPrio(p.Kind)
		if pr == 0 || slotOf(o, p.Kind) >= 0 || (!empty && pr <= cur && !(gunSwappable(o) && pr >= 2)) {
			continue
		}
		d := hyp(x, y, p.Pos[0], p.Pos[1])
		if d < r-0.3 && math.Abs(p.Pos[2]-z) < 3.8 {
			cs = append(cs, c{p.ID, pr, d})
		}
	}
	if len(cs) == 0 {
		return nil
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].pr > cs[j].pr })
	id := cs[0].id
	return &id
}

func (b *Brain) dropGoal(o *Obs, enemySeen bool) *[2]float64 {
	x, y := o.Self.Pos[0], o.Self.Pos[1]
	walk := b.stat(o, "walk_speed", 11.7)
	for _, d := range o.Drops {
		if d.Holder != nil && *d.Holder != o.Self.ID {
			continue
		}
		dist := hyp(x, y, d.Pos[0], d.Pos[1])
		if dist > 220 {
			continue
		}
		if enemySeen && dist > 40 {
			continue
		}
		if dist/walk > d.LandsIn+30 {
			continue
		}
		g := [2]float64{d.Pos[0], d.Pos[1]}
		if dist < d.Radius*0.6 {
			// stay inside the ring
			return &g
		}
		return &g
	}
	return nil
}

func (b *Brain) idleSpot(o *Obs) *[2]float64 {
	ncx, ncy, nr := o.Zone.NextCenter[0], o.Zone.NextCenter[1], o.Zone.NextRadius
	x, y := o.Self.Pos[0], o.Self.Pos[1]
	d := hyp(x, y, ncx, ncy)
	target := math.Min(nr*0.45, 60)
	if d <= target {
		return nil
	}
	k := target / d
	g := [2]float64{ncx + (x-ncx)*k, ncy + (y-ncy)*k}
	return &g
}

// isThreat: a mech worth shooting. Opening fire on bystanders pulls them into the fight.
func (b *Brain) isThreat(e *Actor, x, y, d float64) bool {
	if t, ok := b.attackers[e.ID]; ok && b.t-t < 20 {
		return true
	}
	if d < 45 {
		return true
	}
	toMe := math.Atan2(y-e.Pos[1], x-e.Pos[0])
	if e.Charging && math.Abs(wrap(e.Aim[0]-toMe)) < 0.15 {
		return true
	}
	// closing in on us
	if d < 220 {
		ux, uy := math.Cos(toMe), math.Sin(toMe)
		if e.Vel[0]*ux+e.Vel[1]*uy > 4 {
			return true
		}
	}
	// nearly dead: a cheap kill is worth it
	if e.HP < 0.25*e.MaxHP && d < 150 {
		return true
	}
	return false
}

// shouldFlee: two or more mechs hit us in the last 3 s, or we're losing a trade at low HP.
func (b *Brain) shouldFlee(o *Obs, target *Actor, enemies []Actor, hpFrac float64) bool {
	n := 0
	for _, t := range b.attackers {
		if b.t-t < 3 {
			n++
		}
	}
	fleeHP, fleeLow := optFleeHP, optFleeLow
	if b.ffa {
		fleeHP, fleeLow = optFFAFleeHP, optFFAFleeLow
	}
	if n >= 2 && hpFrac < fleeHP {
		return true
	}
	if hpFrac < fleeLow && target.HP > o.Self.HP*1.1 {
		return true
	}
	return false
}

// rushGoal: nearest remembered laser/lightning when our second slot is empty (or holds something worse).
func (b *Brain) rushGoal(o *Obs, safe func(x, y float64) bool) *[2]float64 {
	x, y, z := o.Self.Pos[0], o.Self.Pos[1], o.Self.Pos[2]
	cur := 0
	if len(o.Self.Weapons) > 1 && o.Self.Weapons[1] != nil {
		cur = weaponPrio(o.Self.Weapons[1].Kind)
	}
	var best *[2]float64
	bs := math.Inf(1)
	for id, pm := range b.seenPicks {
		pr := weaponPrio(pm.kind)
		minPr := 2
		if optRushAny && cur == 0 {
			minPr = 1
		}
		if pr < minPr || slotOf(o, pm.kind) >= 0 || (pr <= cur && !(gunSwappable(o) && pr >= 2)) {
			continue
		}
		if ft, ok := b.takeFail[id]; ok && b.t-ft < 5 {
			continue
		}
		d := hyp(x, y, pm.pos[0], pm.pos[1])
		if d > optRushRange || !safe(pm.pos[0], pm.pos[1]) {
			continue
		}
		if math.Abs(pm.pos[2]-b.w.Ground(pm.pos[0], pm.pos[1])) > 2 && math.Abs(pm.pos[2]-z) > 3 {
			continue // on a roof
		}
		s := d - float64(pr)*20
		if s < bs {
			bs = s
			g := [2]float64{pm.pos[0], pm.pos[1]}
			best = &g
		}
	}
	return best
}

// nearCover: closest standable point within the given radii that the threat can't see.
func (b *Brain) nearCover(o *Obs, threat [3]float64, safe func(x, y float64) bool, radii []float64) *[2]float64 {
	x, y := o.Self.Pos[0], o.Self.Pos[1]
	teye := [3]float64{threat[0], threat[1], threat[2] + 8}
	for _, r := range radii {
		var best *[2]float64
		bd := math.Inf(1)
		for k := 0; k < 16; k++ {
			a := float64(k) * math.Pi / 8
			px, py := x+math.Cos(a)*r, y+math.Sin(a)*r
			if !b.w.Standable(px, py) || !safe(px, py) || !b.w.WalkLine(x, y, px, py) {
				continue
			}
			pz := b.w.Surf(px, py)
			if b.w.Clear(teye, [3]float64{px, py, pz + 6}, 1, 1) {
				continue
			}
			d := 60 - hyp(px, py, threat[0], threat[1]) // prefer not moving toward the threat
			if d < bd {
				bd = d
				g := [2]float64{px, py}
				best = &g
			}
		}
		if best != nil {
			return best
		}
	}
	return nil
}

// gunSwappable: with two-specials on, holding gun + a special means the gun slot is open for the other special.
func gunSwappable(o *Obs) bool {
	if !optTwoSpecial || len(o.Self.Weapons) < 2 || o.Self.Weapons[0] == nil || o.Self.Weapons[1] == nil {
		return false
	}
	return o.Self.Weapons[0].Kind == "gun" && weaponPrio(o.Self.Weapons[1].Kind) >= 2
}

// crateGoal: walk toward the nearest seen crate (cubes: +4% HP and damage each) when idle.
func (b *Brain) crateGoal(o *Obs, safe func(x, y float64) bool, ok bool) *[2]float64 {
	if optCrateHunt <= 0 || !ok || !optCrates {
		return nil
	}
	x, y, z := o.Self.Pos[0], o.Self.Pos[1], o.Self.Pos[2]
	var best *[2]float64
	bd := optCrateHunt
	for _, c := range o.Crates {
		d := hyp(x, y, c.Pos[0], c.Pos[1])
		if d < 30 || d > bd || !safe(c.Pos[0], c.Pos[1]) {
			continue
		}
		if math.Abs(c.Pos[2]-b.w.Ground(c.Pos[0], c.Pos[1])) > 2 && math.Abs(c.Pos[2]-z) > 3 {
			continue // on a roof we're not on
		}
		bd = d
		g := [2]float64{c.Pos[0], c.Pos[1]}
		best = &g
	}
	return best
}

package auto

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Ported from src/brain/mod.rs (everything except the type declarations in brain_types.go).

// NewBrain is Rust's Brain::new.
func NewBrain(h *Hello, s *Shared) *Brain {
	return newBrain(h, s)
}

func newBrain(h *Hello, shared *Shared) *Brain {
	return withParams(h, shared, loadParams())
}

func withParams(h *Hello, shared *Shared, params *Params) *Brain {
	rng := newRng(uint64(h.Seed) ^ (uint64(h.Slot) << 40))
	side := 1.0
	if rng.f64() < 0.5 {
		side = -1.0
	}
	var roof *RoofMap
	if params.RoofMode > 0.0 {
		mn, ms := f2n(fmax(params.RoofMinNodes, 1.0)), params.RoofMinSetback*0.5
		// std::panic::catch_unwind(..).ok()
		func() {
			defer func() {
				if recover() != nil {
					roof = nil
				}
			}()
			roof = roofMapBuild(shared.world, shared.nav, mn, ms, h.Config.Mech.Height)
		}()
	}
	return &Brain{
		cfg:             h.Config,
		p:               params,
		shared:          shared,
		rng:             rng,
		mem:             *newMemory(),
		mode:            ModeIdle,
		goal:            Goal{kind: GoalNone},
		goalID:          -1,
		pursuit:         tupIF{-1, 0.0},
		target:          Tgt{kind: TgtNone},
		allowCrates:     false,
		field:           nil,
		zoneField:       nil,
		goalUnreachable: false,
		cover:           nil,
		lastDir:         V2{0.0, 0.0},
		jukeSide:        side,
		jukeNext:        0.0,
		unstickUntil:    -1.0,
		unstickDir:      V2{0.0, 0.0},
		lastTargetT:     -1e9,
		zoneB:           false,
		unstickCount:    0,
		unstickLevel:    0,
		cmdSince:        0.0,
		wantsSince:      0.0,
		movingSince:     0.0,
		tAct:            time.Now(),
		noLos:           tupIF{-1, 0.0},
		jet:             nil,
		sq:              *newBook(),
		econ:            Econ{},
		gear:            Gear{},
		sf:              Safety{},
		wave:            newWaveState(),
		ffa:             newFfa(h.Seed),
		spot:            nil,
		order:           Order{},
		orderHasPassive: false,
		roof:            roof,
		roofSel:         nil,
		roofSelT:        -1e9,
		roofOn:          nil,
		roofRest:        false,
		roofHold:        false,
		roofJet:         false,
		roofBan:         nil,
		roofStep:        nil,
		roofFirstT:      -1.0,
		kills:           0,
		pk:              newPeek(),
		tele:            Tele{},
		aimz:            newAimZ(),
		cad:             newCadence(),
		cadUntil:        -1.0,
		cadAxis:         V2{1.0, 0.0},
	}
}

func brainPtr[T any](v T) *T {
	return &v
}

// Act is Rust's Brain::act (also used for the Pilot impl with UnlimitedBudget()).
func (b *Brain) Act(o *Obs, budget *Budget) Action {
	obs := o
	b.tAct = time.Now()
	act := newAction(obs.Tick)
	shared := b.shared
	world := shared.world
	nav := shared.nav
	b.econUpdate(obs)
	b.waveCount(obs)
	b.mem.rangeBan = b.p.TakeRangeBan
	b.mem.update(obs, world, &b.cfg)
	b.sfUpdate(obs, world)
	if b.p.AimzOn > 0.0 {
		b.aimz.update(obs)
	}
	if b.p.WCad > 0.0 {
		b.cad.update(obs, b.p.CadPeriod, b.p.CadSnap)
	}
	b.waveUpdate(obs, world, nav)
	b.ffa.update(obs, b.mem.tracks, b.p.FfaOn > 0.5)
	b.order.update(obs)
	b.teleUpdate(obs)
	if b.p.HaltMulti >= 1.0 && !b.order.halted && float64(b.order.onUs(obs.T)) >= b.p.HaltMulti {
		b.order.halted = true
	}
	if of := obs.Offer; of != nil {
		act.Pick = brainPtr(b.pickUpgrade(of))
	}
	b.decide(obs, world, nav)
	b.laserPre(obs, world, nav)
	fc := b.fireControl(obs, world)
	if y, ok := b.gearSweep(obs, fc.target != nil || fc.fire[0] || fc.fire[1] || fc.charging); ok {
		fc.aim = V2{y, 0.0}
	}
	mv := b.planMove(obs, world, nav, budget, &fc)
	if !mv.boost && b.gearBoost(obs) && math.Hypot(mv.dir[0], mv.dir[1]) > 0.5 {
		mv.boost = true
	}
	if jb, ok := b.gearJet(obs); ok {
		j, bo := jb.x0, jb.x1
		mv.jump = mv.jump || j
		mv.boost = mv.boost || bo
	}
	b.laserPost(obs, world, &fc, &mv)
	b.esFilter(obs, &fc)
	b.lootAction(obs, &act)
	b.sfDrop(obs, &act)
	var q float64
	if mv.boost || b.mode != ModeIdle {
		q = 1.0
	} else {
		q = b.ffaQuiet(obs)
	}
	act.Move = brainPtr(V2{mv.dir[0] * q, mv.dir[1] * q})
	act.Boost = brainPtr(mv.boost)
	act.Jump = brainPtr(mv.jump)
	act.Aim = brainPtr(fc.aim)
	act.Fire = brainPtr(fc.fire)
	act.Reload = brainPtr(fc.reload)
	if (b.p.HaltAfter >= 0.0 && obs.T >= b.p.HaltAfter) || b.order.halted {
		act.Move = brainPtr(V2{0.0, 0.0})
		act.Boost = brainPtr(false)
		act.Jump = brainPtr(false)
		act.Fire = brainPtr([2]bool{false, false})
		act.Reload = brainPtr(false)
		act.Take = nil
		act.Drop = nil
	}
	brainSanitize(&act, &obs.Me.Stats)
	return act
}

func (b *Brain) held(t float64) bool {
	return b.p.HoldUntil >= 0.0 && t < b.p.HoldUntil
}

// state is a short state line for logs and tools (Rust uses {:?} Debug formatting).
func (b *Brain) state() string {
	return fmt.Sprintf("%s %s %s tracks=%d items=%d/%d/%d roof=%s/%s/%t first=%.1f",
		brainModeDbg(b.mode), brainGoalDbg(b.goal), brainTgtDbg(b.target),
		len(b.mem.tracks), len(b.mem.crates), len(b.mem.cubes), len(b.mem.pickups),
		brainOptNDbg(b.roofSel), brainOptNDbg(b.roofOn), b.roofRest, b.roofFirstT)
}

func (b *Brain) spentMs() float64 {
	return time.Since(b.tAct).Seconds() * 1000.0
}

func (b *Brain) unsticks() uint32 {
	return b.unstickCount
}

func (b *Brain) pickUpgrade(of *Offer) int64 {
	for _, id := range b.p.UpgradePriority {
		n := min(len(of.Choices), 3)
		for i := 0; i < n; i++ {
			if of.Choices[i].Id == id {
				return int64(i)
			}
		}
	}
	return 0
}

func (b *Brain) prefOf(k WeaponKind) float64 {
	switch k {
	case WeaponKindGun:
		return b.p.PrefGun
	case WeaponKindLaser:
		return b.p.PrefLaser
	case WeaponKindLightning:
		return b.p.PrefLightning
	case WeaponKindMortar:
		return b.p.PrefMortar
	default: // WeaponKindUnknown
		return -1.0
	}
}

// want is Rust's `want(..) -> Option<Option<i64>>`:
//   - ok == false: not wanted (None);
//   - ok == true, slot == nil: goes into an empty slot (Some(None));
//   - ok == true, slot != nil: swap out slot *slot (Some(Some(s))).
func (b *Brain) want(me *SelfObs, kind WeaponKind) (slot *int64, ok bool) {
	if kind == WeaponKindMortar && b.sfNoMortar() {
		return nil, false
	}
	if kind == WeaponKindUnknown {
		return nil, false
	}
	for _, w := range me.Weapons {
		if w != nil && w.Kind == kind {
			return nil, false
		}
	}
	for _, w := range me.Weapons {
		if w == nil {
			return nil, true
		}
	}
	worstPv, worstI, hasWorst := 0.0, 0, false
	for i, w := range me.Weapons {
		if w == nil {
			continue
		}
		if w.Kind == WeaponKindLaser && w.Charge > 0.0 {
			continue
		}
		pv := b.prefOf(w.Kind)
		if !hasWorst || pv < worstPv {
			worstPv, worstI, hasWorst = pv, i, true
		}
	}
	if hasWorst && worstPv+0.5 < b.prefOf(kind) {
		s := int64(worstI)
		return &s, true
	}
	return nil, false
}

func (b *Brain) lootAction(obs *Obs, act *Action) {
	me := &obs.Me
	reach := fmax(b.cfg.Loot.PickupRadius-0.3, 0.5)
	bestV, bestID, hasBest := 0.0, int64(0), false
	var bestSlot *int64
	for i := range obs.Pickups {
		pk := &obs.Pickups[i]
		if dist2(pk.Pos, me.Pos) > reach || math.Abs(pk.Pos[2]-me.Pos[2]) > 3.7 {
			continue
		}
		if b.mem.blacklisted(pk.Id, obs.T) {
			continue
		}
		if slot, ok := b.want(me, pk.Kind); ok {
			v := b.prefOf(pk.Kind)
			if !hasBest || v > bestV {
				bestV, bestID, bestSlot, hasBest = v, pk.Id, slot, true
			}
		}
	}
	if hasBest {
		act.Take = brainPtr(bestID)
		act.Drop = bestSlot
	}
}

// brainSanitize is Rust's private free fn `sanitize`.
func brainSanitize(act *Action, st *MechCfg) {
	m := V2{0.0, 0.0}
	if act.Move != nil {
		m = *act.Move
	}
	if !isFinite(m[0]) || !isFinite(m[1]) {
		m = V2{0.0, 0.0}
	}
	l := math.Hypot(m[0], m[1])
	if l > 1.0 {
		m = V2{m[0] / l * 0.999_999, m[1] / l * 0.999_999}
	}
	act.Move = brainPtr(m)
	a := V2{0.0, 0.0}
	if act.Aim != nil {
		a = *act.Aim
	}
	lo, hi := -0.7, 1.3
	if st.MinPitch < st.MaxPitch {
		lo, hi = st.MinPitch, st.MaxPitch
	}
	act.Aim = brainPtr(V2{wrap(a[0]), clampF(fin(a[1], 0.0), lo, hi)})
	if act.Fire == nil {
		act.Fire = brainPtr([2]bool{false, false})
	}
	if act.Pick != nil {
		act.Pick = brainPtr(min(max(*act.Pick, 0), 2))
	}
}

// ---- Debug formatting helpers for state() (mirror Rust's derived {:?}).

func brainF64Dbg(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "inf"
	case math.IsInf(x, -1):
		return "-inf"
	}
	ax := math.Abs(x)
	if ax != 0 && (ax < 1e-4 || ax >= 1e16) {
		s := strconv.FormatFloat(x, 'e', -1, 64)
		mant, exp, _ := strings.Cut(s, "e")
		e, _ := strconv.Atoi(exp)
		return mant + "e" + strconv.Itoa(e)
	}
	s := strconv.FormatFloat(x, 'f', -1, 64)
	if !strings.ContainsAny(s, ".") {
		s += ".0"
	}
	return s
}

func brainV3Dbg(v V3) string {
	return "[" + brainF64Dbg(v[0]) + ", " + brainF64Dbg(v[1]) + ", " + brainF64Dbg(v[2]) + "]"
}

func brainV2Dbg(v V2) string {
	return "[" + brainF64Dbg(v[0]) + ", " + brainF64Dbg(v[1]) + "]"
}

func brainModeDbg(m Mode) string {
	switch m {
	case ModeIdle:
		return "Idle"
	case ModeZone:
		return "Zone"
	case ModeFight:
		return "Fight"
	case ModeRetreat:
		return "Retreat"
	case ModeLoot:
		return "Loot"
	case ModeDrop:
		return "Drop"
	case ModeHunt:
		return "Hunt"
	case ModeRoof:
		return "Roof"
	}
	return "Mode(" + strconv.Itoa(int(m)) + ")"
}

func brainGoalDbg(g Goal) string {
	switch g.kind {
	case GoalNav:
		return "Nav { pos: " + brainV3Dbg(g.pos) + ", arrive: " + brainF64Dbg(g.arrive) + " }"
	case GoalField:
		return "Field { center: " + brainV2Dbg(g.center) + ", arrive: " + brainF64Dbg(g.arrive) + " }"
	case GoalRange:
		return "Range { pos: " + brainV3Dbg(g.pos) + ", pref: " + brainF64Dbg(g.pref) + " }"
	}
	return "None"
}

func brainTgtDbg(t Tgt) string {
	switch t.kind {
	case TgtMech:
		return "Mech(" + strconv.FormatInt(t.id, 10) + ")"
	case TgtNpc:
		return "Npc(" + strconv.FormatInt(t.id, 10) + ")"
	case TgtCrate:
		return "Crate(" + strconv.FormatInt(t.id, 10) + ")"
	}
	return "None"
}

func brainOptNDbg(v *int) string {
	if v == nil {
		return "None"
	}
	return "Some(" + strconv.Itoa(*v) + ")"
}

package auto

import "time"

// Ported from src/brain/mod.rs (types only). Methods live in brain.go and the module files.

type Mode int

const (
	ModeIdle Mode = iota
	ModeZone
	ModeFight
	ModeRetreat
	ModeLoot
	ModeDrop
	ModeHunt
	ModeRoof
)

type GoalKind int

const (
	GoalNone  GoalKind = iota
	GoalNav            // pos, arrive
	GoalField          // center, arrive
	GoalRange          // pos, pref
)

// Goal is Rust's `enum Goal { None, Nav{pos,arrive}, Field{center,arrive}, Range{pos,pref} }`.
type Goal struct {
	kind   GoalKind
	pos    V3
	center V2
	arrive float64
	pref   float64
}

type TgtKind int

const (
	TgtNone TgtKind = iota
	TgtMech
	TgtNpc
	TgtCrate
)

// Tgt is Rust's `enum Tgt { None, Mech(i64), Npc(i64), Crate(i64) }`; comparable with ==.
type Tgt struct {
	kind TgtKind
	id   int64
}

// Option<(V3, Arc<nav::Field>, f64)>
type brainField struct {
	x0 V3
	x1 *Field
	x2 float64
}

// Option<(V2, f64, Arc<nav::Field>)>
type brainZoneField struct {
	x0 V2
	x1 float64
	x2 *Field
}

// Option<(i64, Option<V3>, f64)>
type brainRoofStep struct {
	x0 int64
	x1 *V3
	x2 float64
}

type Brain struct {
	cfg             Config
	p               *Params
	shared          *Shared
	rng             Rng
	mem             Memory
	mode            Mode
	goal            Goal
	goalID          int64
	pursuit         tupIF
	target          Tgt
	allowCrates     bool
	field           *brainField
	zoneField       *brainZoneField
	goalUnreachable bool
	cover           *tupVF
	lastDir         V2
	jukeSide        float64
	jukeNext        float64
	unstickUntil    float64
	unstickDir      V2
	lastTargetT     float64
	zoneB           bool
	unstickCount    uint32
	unstickLevel    uint32
	cmdSince        float64
	wantsSince      float64
	movingSince     float64
	tAct            time.Time
	noLos           tupIF
	jet             *tupVF
	sq              Book
	econ            Econ
	gear            Gear
	sf              Safety
	wave            WaveState
	ffa             Ffa
	spot            *tupVFB
	order           Order
	orderHasPassive bool
	roof            *RoofMap
	roofSel         *int
	roofSelT        float64
	roofOn          *int
	roofRest        bool
	roofHold        bool
	roofJet         bool
	roofBan         []tupNF
	roofStep        *brainRoofStep
	roofFirstT      float64
	kills           uint32
	pk              Peek
	tele            Tele
	aimz            AimZ
	cad             Cadence
	cadUntil        float64
	cadAxis         V2
}

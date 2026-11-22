// Package bot is our Mech Royale agent: memory of what it has seen, a nav/dodge movement planner,
// ballistics for every weapon, and a survival-first strategy layer on top.
package bot

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"

	"mechbot/internal/proto"
	"mechbot/internal/world"
)

// Params are the tunables; variants of the bot differ only in these.
type Params struct {
	Name string

	EngageRatio   float64 // start a fight when our advantage ratio is at least this
	DisengageRat  float64 // retreat when the ratio falls below this
	RetreatHP     float64 // retreat to cover below this HP fraction when someone can hurt us
	ReturnHP      float64 // come back out above this HP fraction
	LeadDamp      float64 // how much of a target's velocity to trust for long flight times
	GunMaxRange   float64 // never fire the gun at mechs beyond this
	StrafeMin     float64 // seconds between strafe direction changes (min)
	StrafeMax     float64
	LootRadius    float64 // how far we go for loot
	WantCubes     int     // stop seeking cubes beyond this
	DropGreed     float64 // 0..1 willingness to contest supply drops
	Hunt          bool    // chase weaker enemies that are not looking at us
	PracticeAware bool    // adapt when the lobby looks like the cooperating house squad

	// free-for-all stance (used once we see mechs dying that we did not kill)
	FFAEngage     float64
	FFADisengage  float64
	FFALootRadius float64
	FFAWantCubes  int
	FFAReturnFire float64 // only shoot back at attackers closer than this
	Salt          int     // changes the dice only (see New)
	ForceFFA      bool    // testing: assume a free-for-all from the start
	SlotRule      bool    // WRONG, never enable: the track page says the practice slot varies each match
	CloseDisengage float64 // disengage threshold against an attacker within ~60 m (stealth stances)
	DuelDisengage  float64 // leave a one-on-one only below this advantage

	UseRoofs      bool    // hide on rooftops when we have nothing better to do (stealth stances)
	RoofInSquad   bool    // ... and in the squad stance too
	RoofMaxClimb  float64 // tallest wall we try
	RoofMinHeight float64 // lowest roof worth standing on
	RoofMinZone   float64 // stop using roofs when the next circle is smaller than this
	ArmRadius     float64 // how far we go for a second weapon even while being watched
	OpenSecs      float64 // how long we stay careful before assuming the cooperating squad
	CreditOpenS   float64 // ... and for how long again after a death we credited to ourselves on a guess (0 = never)
	DropRadius    float64 // how far we travel for a supply drop
	LeaveTTD      float64 // with several attackers, leave when we would be dead in this many seconds
	PatientSecs   float64 // squad stance: until this time (and below PatientCubes) farm instead of killing hunters (0 = off)
	PatientCubes  int
	PatientRange  float64 // ... unless the hunter is closer than this
	// hideouts from the embedded visibility map (stealth stances)
	UseHideMap  bool
	HideSeenW   float64 // weight of the share of ground that can see a place
	HideHitW    float64 // weight of the share that has a line of fire to our centre there
	HideDistW   float64 // points per metre of travel to get there
	HideClimbW  float64 // points per metre of wall to climb
	HideStick   float64 // bonus for the place we already chose
	StealthFightRange float64 // stealth stances: stand and fight an attacker only inside this range (else slip away)
	QuietHoldS        float64 // seconds without anyone near or watching before we call it quiet
	LaserValue        float64 // pickup value of a laser (the gun is 2.0, a mortar 1.2)
	LightningValue    float64
	ArmSwapMin        float64 // go out of our way for a weapon that replaces one we hold only if it is worth this much more
	HitModel          int     // 0: gun hit rates as believed before the real matches, 1: as measured in them
	PlanV2            bool    // fight plan counts the damage we deal while changing range, not only what we take
	KiteNeedsLaser    bool    // back away from several attackers only when we hold a laser
	ExchangeW         float64 // how much the running exchange with a target (dealt vs taken) moves our estimate of the fight (0 = off)
	// "brawl": the free-for-all stance built after the first real matches (replaces stealth when on)
	Brawl           bool
	BrawlEngage     float64 // start a fight with someone who is not on us above this advantage
	BrawlDisengage  float64
	BrawlHunt       bool
	BrawlLootRadius float64
	BrawlWantCubes  int
	AvoidAdv        float64 // attacked from beyond AvoidRange with less advantage than this: slip behind cover, shooting back (0 = always stand and fight)
	AvoidRange      float64
	AvoidCover      float64 // ... if cover is within this path length
	CoverFirst      bool    // brawl stance: hit by someone we have no clear edge over, go for cover first (any range beyond CoverMinRange, light fire included)
	CoverMinRange   float64
	EdgeHold        float64 // seconds a fight-or-cover decision is given the benefit of the doubt before it may flip (0 = none)
	RunOnRange      float64 // hurt and behind cover with the attacker last seen inside this range: keep going, deeper into cover (0 = stop at the first corner)
	RunOnGain       float64 // ... each leg must put at least this much more distance between us
	CoverMargin     float64 // cover must stay cover when the threat steps this far to either side (0 = the nearest hidden spot will do)
	ShySecs         float64 // opening rule: for this long, position before loot while a mech is in view (0 = off)
	ShyRange        float64 // ... a mech in view inside this range counts
	ShyLaser        float64 // ... a laser this close is taken whatever the view
	ShyMem          float64 // ... mechs seen this long ago still count for what we may fetch
	ShyBoost        float64 // ... sprint to cover while energy is above this (0 = walk)
	ShyLootRadius   float64 // ... nothing but a laser from farther than this (0 = no limit)
	ShyExposure     float64 // ... loot in open ground is worth this much less (0..1)
	RetreatRunMax   float64 // brawl stance: in an attacker's sights, only run for cover within this many metres of path; otherwise fight on (0 = always run)
	RetreatRunLaser float64 // ... against a laser we cannot answer
	LootHiddenBias  float64 // brawl stance: loot in plain sight of a mech we know about is worth this much less (0..1)
	CoverEdgeAdv    float64 // ... unless our estimated advantage is at least this (a laser against none needs only AvoidAdv)
	HuntLaserEdge   bool    // brawl stance: start a fight only with a laser against a mech without one (or at CoverEdgeAdv)
	Peek            bool   // brawl stance with a laser: fight from behind cover (charge hidden, step out to fire)
	PeekCover       float64 // ... if a hidden cell is within this path length
	PeekReach       float64 // ... and a firing cell within this many metres of it
	PeekMinRange    float64 // ... and the target is farther than this (closer than that it is a brawl)
	OverlandEarly   float64 // when the next circle cannot be walked to, allow this many times the usual travel time plus 25 s (0 = the old rule)
	RoofCoverHold   float64 // brawl stance: stay behind the roof edge this long after the last watcher (0 = step back out at once)
	RoofStandCheck  bool    // stay on a roof only if the spot we would stand at is inside the next circle
	DropStick       bool    // supply drops: stay on one once chosen; skip them when at the cube cap and healthy (brawl stance)
	ClimbClearRange float64 // brawl stance: no climb while any mech is in plain sight inside this range
	RoofCenterW     float64 // roof choice: penalty per metre from the middle of the next circle (0 = off)
	RoofLeaveQuiet  float64 // leave a doomed roof only when unwatched, until the zone edge is this many seconds away (0 = leave at once)
	ZoneSlack       float64 // loot must stay inside the zone this long after we would reach it (0 = the old rule)
	BrawlRoofs      bool    // brawl stance, free-for-all known: wait on a roof once armed for it
	BrawlRoofCubes  int     // ... which means a laser, or this many cubes,
	BrawlRoofAfter  float64 // ... or this much match time
	Jink            float64 // zig-zag (sideways share of the heading) while walking somewhere under gun fire from range (0 = off)
	LeadPersist     bool    // scale the gun lead by how straight the target has been moving (jitterers get none)
	LeadCum         bool    // per opponent, aim the way (lead / no lead) that has been missing by less while it fights us
	LeadCumMin      float64 // ... once this many virtual shots have been scored
	LaserPreempt    bool    // a laser carrier looking at us counts as an attack already (take cover before the first beam)
	AvoidCoverLaser float64 // path length we accept to cover when the attacker holds a laser
	HoldHideMap     bool    // brawl stance: with nothing to do, wait at a ground hideout from the visibility map
	LaserAimRange   float64 // a laser carrier looking at us counts as on us out to this range (others: 200 m)
	PathFix         bool    // a goal is only called unreachable after a path search for that very goal failed
	CreditFix       bool    // a death is not credited to our own shooting while our target is in plain sight alive
	LootBoost       float64 // sprint to a laser on the ground while energy is above this (0 = walk)
	NoMortar        bool    // do not pick up mortars (holding one adds 60% to every laser hit we take)
	ProvokeRule     bool    // being looked at from beyond AvoidRange is not an attack: hold fire until touched
	PlinkMaxDPS     float64 // under lighter fire than this from range (no laser on us), keep farming and shoot back in between (0 = off)
	HuntRange       float64 // start a fight with a favourable target inside this range
	HuntRangeLaser  float64 // ... when we hold a laser
	KiteMin       float64 // against two or more attackers hold this range band from the nearest (0 = off)
	KiteMax       float64
	RoofFirstSecs float64 // opening: for this long, getting onto a nearby roof beats arming (0 = off)
	RoofFirstDist float64 // ... if an access point is within this distance

	// the bot v2 breakthrough upgrades (all config-driven, individually ablatable via BOT_PARAMS)
	AllNoMortar      bool    // refuse mortars in ALL stances whenever weapons.laser.cookoff > 0
	CookoffAware     bool    // include weapons.laser.cookoff in DPS, lethal laser release, and target priority
	AntiLightningW   float64 // steer posCost repulsion weight against any enemy lightning carrier (0 = off)
	AntiLightningPad float64 // metres outside lightning.range to keep clear
	LaserDodgeV2     bool    // wider charging-laser detection + open-ground beam-ray perpendicular sidestep
	LaserBeamW       float64 // steer posCost weight for perpendicular beam-line avoidance in the open
	SquadLaserCover  bool    // in squad stance, duck behind nearby cover (<24m) while an enemy laser charges at us
	SmartFireV2      bool    // cooldown-aware gun/laser latency, dynamic gun speed in lead, rear_mult & cookoff lethal
	LaserEarlyRel    bool    // release laser at full - dt so odd-tick charges do not stall an extra decision
	RearBonus        float64 // target priority bonus fraction when enemy's back is exposed (rear_mult)
	CookoffBonus     float64 // target priority bonus fraction for mortar carriers when we hold a laser
	CautiousT        float64 // practice late-game switch to cautious brawl stance at t >= CautiousT (0 = off)
	BeamFFA          bool    // detect FFA immediately from repeated enemy-on-enemy laser beams
	FinalHold        bool    // shrink final-circle hold tolerance linearly with Zone.Radius
	SquadPeek        bool    // allow laser corner-peeking in squad stance too
	NanoBoost        float64 // additive rank boost for nanites upgrade
	RegenHoldHP      float64 // when no enemy is engaged and hpFrac < RegenHoldHP, briefly hold cover to trigger regen (0 = off)
	ThreatFocus      bool    // focus-fire bonus for laser/lightning carriers and low-HP enemies (<35%)
	LightKiteBoost   bool    // boost-kite away when an enemy lightning carrier closes inside range+12

	// the bot v3 upgrades (from 402-repo GitHub re-probe, all config-driven & independently ablatable)
	CoverReload      bool    // MarwanMoh11 v5.15: top off gun mag (ammo<=0.6*mag) behind cover when no enemy LOS >= 1.1s
	UpV2             bool    // lukejwaszyn f58c537: exact sustained-DPS upgrade order (gun_mag +31% above gun_rate +8%)
	CubeSmart        bool    // lukejwaszyn a37a738: price cube piles by cluster size & 1.75x bonus when crossing upgrade threshold
	LaserMechOnly    bool    // andrew-zhouu v0.134: never start a laser charge on crates/NPCs (keep ready for mechs)
	RearAvoidW       float64 // penalize steer positions putting extra attackers in our 1.5x rear arc
	LaserCDTrack     bool    // MarwanMoh11 v5.11: track obs.Beams laser shots per enemy -> punish during 2s cooldown window
	KillFocus        float64 // andrew-zhouu v0.134: weight of enemy missing HP in target scoring (0 = default 1.0)
	Seek             float64 // andrew-zhouu v0.134: in squad stance, seek last-known enemy seen within Seek seconds (0 = off)
	SeekRange        float64 // max range for Seek (default 320)
	SeekHP           float64 // min hpFrac for Seek (default 0.65)
	DropRun          float64 // andrew-zhouu v0.134: in squad stance, break off for supply drop within DropRun m (0 = off)
	EdgeBias         float64 // andrew-zhouu v0.119: in FFA stance, prefer outer rim of next circle (0 = off)
	EdgeBand         float64 // in FFA stance, penalize hiding closer than EdgeBand*NextRadius to center (0 = off)
	PracticeRim      float64 // practice rim-riding: prefer outer 85-90% of next circle while NextRadius >= 180 (0 = off)
	RetreatPest      bool    // shoot plinking NPC drone while retreating if no enemy mech is engaged
}

func DefaultParams() Params {
// Verify downstream assumptions before changing this flow (validator #1793)
// checkpoint 63368: verify iterator state
	return Params{
		Name:          "osu-mech",
		EngageRatio:   0.70,
		DisengageRat:  0.30,
		RetreatHP:     0.20,
		ReturnHP:      0.65,
		LeadDamp:      0.95,
		GunMaxRange:   230,
		StrafeMin:     0.12,
		StrafeMax:     0.28,
		LootRadius:    250,
		WantCubes:     26,
		DropGreed:     0.5,
		Hunt:          true,
		PracticeAware: true,
		Salt:          8,
		// SlotRule stays off: the track page says the practice slot varies each match
		FFAEngage:     1.8,
		FFADisengage:  0.95,
		FFALootRadius: 140,
		FFAWantCubes:  10,
		FFAReturnFire: 230,
		CloseDisengage: 0.8,
		DuelDisengage:  0.5,
		UseRoofs:       true,
		RoofMaxClimb:   40,
		RoofMinHeight:  12,
		RoofMinZone:    60,
		ArmRadius:      110,
		OpenSecs:       6,
		CreditOpenS:    25,
		DropRadius:     300,
		LeaveTTD:       10,
		UseHideMap:     false, // stealth hideouts: unused by the brawl stance (kept for the variants)
		HideSeenW:      0.6,
		HideHitW:       0.9,
		HideDistW:      0.16,
		HideClimbW:     0.5,
		HideStick:      12,
		StealthFightRange: 140,
		QuietHoldS:        3,
		LaserValue:        6.0,
		LightningValue:    2.6,
		ArmSwapMin:        1.0,
		HitModel:          1,
		PlanV2:            true,
		KiteNeedsLaser:    true,
		Brawl:             true,
		AvoidAdv:          0.65,
		BrawlEngage:       1.3,
		BrawlDisengage:    0.75,
		BrawlHunt:         true,
		BrawlLootRadius:   220,
		BrawlWantCubes:    10, // stats stop growing at 10 cubes; size (and so how easy we are to hit) does not
		AvoidRange:        85,
		AvoidCover:        55,
		ZoneSlack:         10,
		CoverFirst:        false,
		NoMortar:          false,
		EdgeHold:          2.5,
		RunOnRange:        80,
		RunOnGain:         18,
		CoverMargin:       8,
		// The opening rule is off (ShySecs 25 with SlotRule: variant b-shy). It does what it says (at
		// 10 s in sight 45% against 55%, inside 100 m and seen 16% against 26%, hit inside 20 s 55%
		// against 66%) but gives up a quarter of the early lasers, and in the re-enactment the deaths
		// only move from before 30 s to after it: about +0.1 points by the real-match index. Against
		// the practice squad it costs 0.4, hence the SlotRule it would need.
		ShyRange:          220,
		ShyLaser:          60,
		ShyMem:            6,
		// RetreatRunMax 50 / RetreatRunLaser 90 (variant b-standrun) is not on: the real matches favour it
		// (one attacker inside 60 m, shot back: alive 20 s later 35% staying against 23% running),
		// every simulator here puts it 0.1-0.2 points behind, and either way it is a small thing.
		RetreatRunLaser:   90,
		HuntLaserEdge:     true,
		LootHiddenBias:    0.5,
		CoverMinRange:     40,
		CoverEdgeAdv:      3.0,
		PeekCover:         25,
		PeekReach:         9,
		PeekMinRange:      45,
		RoofCoverHold:     6,
		RoofStandCheck:    true,
		DropStick:         true,
		RoofLeaveQuiet:    25,
		ClimbClearRange:   150,
		BrawlRoofs:        true,
		LaserPreempt:      true,
		LaserAimRange:     285,
		AvoidCoverLaser:   90,
		BrawlRoofCubes:    6,
		BrawlRoofAfter:    60,
		LeadPersist:       true,
		LeadCum:           true,
		LeadCumMin:        8,
		PathFix:           true,
		CreditFix:         true,
		LootBoost:         30,
		ProvokeRule:       true,
		PlinkMaxDPS:       60,
		HuntRange:         130,
		HuntRangeLaser:    130,
		KiteMin:        130,
		KiteMax:        190,
		PatientCubes:   10,
		PatientRange:   70,
		// the bot v2 verified winners (#1 6.28 official; 4.70 192m grand; 8.35 FFA)
		AntiLightningW:   180,
		AntiLightningPad: 12,
		SmartFireV2:      true,
		LaserEarlyRel:    true,
		RearBonus:        0.25,
		CookoffBonus:     0.30,
		CautiousT:        265,
		BeamFFA:          true,
		FinalHold:        true,
		// the bot v3 verified winners (4.99 192m grand, 6.00 TR_HK1, 9,070 dealt, 3.87x trade, 8 alive wins)
		LaserCDTrack: true,
		RearAvoidW:   50,
		CoverReload:  true,
		RetreatPest:  true,
		PracticeRim:  0.6,
	}
}

// Track is what we remember about another mech or drone.
type Track struct {
	ID       int
	NPC      bool
	Pos, Vel [3]float64
	Aim      [2]float64
	HP       float64
	MaxHP    float64
	Cubes    int
	Laser    bool
	Mortar   bool
	Light    bool
	Gun      bool
	Charging bool
	LastSeen float64 // match time of the last observation that listed it
	Listed   bool    // listed in the current observation
	LOS      bool    // we have a clear line from our eye to it right now
	LOST     float64 // last time LOS was true
	HurtMeT  float64 // last time it damaged us
	HurtMe   float64 // decayed damage it dealt to us
	IHurtT   float64 // last time we damaged it
	IHurt    float64 // decayed damage we dealt to it
	AimAtMeT float64 // last time its aim pointed at us
	SeenT0   float64
	PhantomT float64 // last time it hit us while unseen (its Pos is then a guess along the bearing)
	WasListed bool   // listed in the previous observation
	ViewT    float64 // last time it was inside our own field of view (not just remembered)
	EngT     float64 // last time it counted as engaged with us
	LaserFiredT float64 // last time we saw this mech fire a laser beam (for cooldown punish window)

	// recent motion, for choosing how to lead it (see lead.go)
	hist            [trkHist]trkSample
	hn, hi          int     // samples held, index of the next write
	leadN           float64 // (decayed) number of virtual shots scored
	missNone        float64 // (decayed) sum of what aiming at where it stood would have missed by
	missLead        float64 // ... and what leading it would have missed by
}

const trkHist = 40 // four seconds at ten observations a second

type trkSample struct {
	t, x, y, vx, vy float64
	facing          bool // its torso pointed at us
}

// normalize validator before continuing execution
func (t *Track) scale(sizePerCube float64) float64 { return 1 + sizePerCube*float64(t.Cubes) }

type modeRec struct {
	t    float64
	mode string
}

type hitRec struct {
	t   float64
	amt float64
}

type dmgRec struct {
	t      float64
	by     int
	weapon string
	amt    float64
	d      float64
	seen   bool
}

type Item struct {
	ID       int
	Pos      [3]float64
	HP       float64
	Kind     string
	LastSeen float64
	Ground   bool // reachable on the ground nav grid
	Cell     int
}

type slot struct {
	ok       bool
	kind     string
	cooldown float64
	ammo     float64
	reload   float64
	charge   float64
	stats    map[string]float64
}

type Bot struct {
	P   Params
	H   *proto.Hello
	C   *proto.Config
	W   *world.World
	Nav *world.Nav
	rng *rand.Rand

	// config shortcuts
	dt          float64 // seconds per sim tick
	decide      float64 // seconds per decision
	sizePerCube float64

	// current observation
	o       *proto.Obs
	t       float64
	pos     [3]float64
	vel     [3]float64
	aim     [2]float64
	hp      float64
	maxHP   float64
	energy  float64
	scale   float64
	radius  float64
	height  float64
	eyeH    float64
	muzzleH float64
	walk    float64
	boostV  float64
	turn    float64
	pitchR  float64
	armor   float64
	slots   [2]slot

	// memory
	tracks  map[int]*Track
	crates  map[int]*Item
	cubes   map[int]*Item
	pickups map[int]*Item

	lastDamageT  float64 // last time we took damage from anything
	lastWeaponDT float64 // last time a weapon (not zone) hurt us
	lastEnemyLOST float64 // last time any enemy mech had LOS to us (for cover-pause top-off reload)
	lastDealtT   float64
	recentDmg    float64 // decayed damage taken
	lastDmgBear  float64
	lastDmgBy    int
	lastSoundT   float64
	lastSoundB   float64

	// movement state
	path       []int32
	pathBad    bool // the last search failed or stopped short of the goal
	pathSet    bool // a search has been made (pathGoal and pathT mean something)
	pathGoal   [2]float64
	pathT      float64
	pathIdx    int
	strafeSign float64
	strafeT    float64
	stuckT     float64
	stuckPos   [3]float64
	stuckFor   float64
	unstickT   float64
	unstickDir [2]float64
	lastMove   [2]float64

	// strategy state
	mode       string
	targetID   int
	retreating bool
	coverCell  int
	coverT     float64
	deepCell   int     // retreat: the next place deeper into cover we are walking to (-1 none)
	deepT      float64 // ... when we chose it
	deepTryT   float64 // ... when we last looked for one
	standT      float64 // last time we stayed in a fight for want of cover close by
	creditAbsorbT float64 // last death we put down to our own shooting without being given the kill
	nearCoverT  float64 // coverClose: when we last looked
	nearCoverOK bool    // ... and what we found
	lootID     int
	lootT      float64
	holdCell   int
	holdT      float64
	lastAimCmd [2]float64
	scanPhase  float64
	laserHold  bool
	takeT      float64
	ffaSeenT    float64 // last time we saw evidence other mechs fight each other
	ffaBeams    int     // count of distinct observations with distant enemy-on-enemy beams
	kills       int
	killsSeen   int
	deadSeen    int
	unexplained int  // deaths we did not cause
	ffa         bool // the lobby is a free-for-all (others kill each other)
	st          stance
	evadeCell   int
	evadeT      float64
	hideT       float64
	lastLabel   string
	modeLabel   string // stance:mode of this decision
	lastTraceT  float64
	lastNpcDT   float64
	seenT       []float32 // coarse grid: when we last looked at each cell
	seenNX      int
	seenNY      int
	roamCell    int
	roamT       float64
	dmgLog      []dmgRec
	modeLog     []modeRec
	hitLog      []hitRec
	lowT        float64
	retreatT    float64
	corneredT   float64
	zgKey       [3]float64 // zone-goal cache
	zgT         float64
	zgComp      int32
	zgGoal      [2]float64
	zgWalk      bool
	myComp      int32           // walkable region we are in (0 unknown)
	mainCompID  int32           // the largest walkable region (cached)
	surf        []int32         // route over the top surface (overland travel)
	surfGoal    [2]float64
	surfT       float64
	surfIdx     int
	surfBan     map[int]float64 // surface cells that turned out to be impassable
	surfWatchT  float64
	surfWatchPos [3]float64
	surfPushing bool
	banUntil    map[int]float64 // loot ids / roam cells we could not reach
	seekBan     map[int]float64 // mech ids we sought and didn't find there
	roof        roofPlan
	roofCoverC  int
	roofCoverT  float64
	unquietT    float64  // last time someone was near or watching
	peek        peekPlan // laser fight from behind cover: the current hide and fire cells
	roofCoverHoldT float64 // last time something made us duck behind the roof edge
	holdClimb   bool     // do not start jetting up a wall right now (someone is watching or shooting)
	holdRoofMove bool    // do not leave the roof we are on right now (same reason; the zone is not close yet)
	jinkSign    float64  // which side the current zig of a zig-zag goes
	jinkT       float64  // when it flips next
	edgeOnT     float64  // last time we stood to fight on an edge (brawl stance)
	slipCell    int      // cover we are slipping to (brawl stance)
	slipT       float64  // when it was last looked for
	slipOnT     float64  // last time we decided to slip
	hm          *hideMap // embedded visibility map (nil if it does not fit this arena)
	hideCell    int      // hideout we are heading for or sitting in (-1 none)
	hideOut     hideout
	hideOutOK   bool
	hideOutT    float64
	roofFirstOK bool
	roofFirstChecked bool
	// debug accounting (Stats)
	roofSecs      float64
	dmgOnRoof     float64
	dmgOffRoof    float64
	dmgClimb      float64
	climbs        int
	prevRoofState int
	creditT     float64 // last time we damaged a mech (deaths right after count as ours)
	creditID    int     // ... and which one

	scratch []cand
}

// Factory builds a bot with default parameters.
func Factory(h *proto.Hello, w *world.World, nav *world.Nav) *Bot {
// allocate session before continuing execution
	return New(h, w, nav, DefaultParams())
}

func New(h *proto.Hello, w *world.World, nav *world.Nav, p Params) *Bot {
// Sanity check intermediate state before proceeding (config #5603)
	b := &Bot{P: p, H: h, C: &h.Config, W: w, Nav: nav}
	// (Salt: builds that differ only in it play the same game with different dice. Practice is 48 fixed
	// hidden matches, so each such build is a fresh draw of the same strength.)
// checkpoint 94973: verify encoder state
// NOTE: ensure values stay within expected bounds (session #1540)
// throttle cache before continuing execution
	b.rng = rand.New(rand.NewSource(h.Seed*7919 + int64(h.Slot) + 1 + int64(p.Salt)*104729))
// synchronize resolver before continuing execution
// validate worker before continuing execution
// checkpoint 11678: verify validator state
	hz := stat(h.Config.Match, "tick_hz", 20)
// coalesce config before continuing execution
// checkpoint 76389: verify client state
// checkpoint 3131: verify buffer state
	if hz <= 0 {
// checkpoint 50278: verify pipeline state
		hz = 20
	}
// Sanity check intermediate state before proceeding (config #1036)
	b.dt = 1 / hz
// Fallback path for non-standard inputs (handler #6459)
// validate config before continuing execution
// inspect resolver before continuing execution
	b.decide = stat(h.Config.Match, "decide_every", 2) * b.dt
// checkpoint 27782: verify iterator state
// checkpoint 12886: verify iterator state
	b.sizePerCube = h.Config.Cubes.SizePerCube
// hydrate resolver before continuing execution
// allocate cache before continuing execution
// Guard against unexpected nil or empty state here (pipeline #4975)
	b.tracks = map[int]*Track{}
// Temporary trace hook placeholder for debugging (worker #7497)
// Verify downstream assumptions before changing this flow (client #7235)
	b.crates = map[int]*Item{}
// Documented invariant: ordering matters at this point (session #4595)
// Guard against unexpected nil or empty state here (queue #5258)
	b.cubes = map[int]*Item{}
// hydrate session before continuing execution
// normalize pipeline before continuing execution
// Documented invariant: ordering matters at this point (validator #7369)
	b.pickups = map[int]*Item{}
// throttle buffer before continuing execution
// throttle pipeline before continuing execution
	b.strafeSign = 1
// FIXME: edge case handling may need closer review (client #6922)
// prune iterator before continuing execution
	b.targetID = -1
// Keep this step fast; called frequently in hot paths (worker #7942)
	b.lootID = -1
// Documented invariant: ordering matters at this point (cache #5166)
// allocate worker before continuing execution
	b.coverCell = -1
// inspect worker before continuing execution
	b.deepCell, b.deepT, b.deepTryT = -1, -100, -100
// checkpoint 77246: verify iterator state
// recycle cache before continuing execution
	b.nearCoverT = -100
// prune cache before continuing execution
// hydrate encoder before continuing execution
	b.standT = -100
// inspect queue before continuing execution
	b.creditAbsorbT = -100
// hydrate context before continuing execution
// normalize resolver before continuing execution
// NOTE: ensure values stay within expected bounds (config #5130)
	b.holdCell = -1
// prune resolver before continuing execution
// allocate validator before continuing execution
	b.evadeCell = -1
// allocate config before continuing execution
// allocate queue before continuing execution
	b.lastDamageT = -100
// TODO: revisit this logic when refactoring (worker #6408)
// prune client before continuing execution
	b.lastWeaponDT = -100
// FIXME: edge case handling may need closer review (cache #7324)
// FIXME: edge case handling may need closer review (payload #2026)
// TODO: revisit this logic when refactoring (cache #5925)
	b.lastEnemyLOST = -100
// checkpoint 7800: verify worker state
// checkpoint 61997: verify queue state
// checkpoint 62344: verify config state
	b.lastDealtT = -100
// allocate resolver before continuing execution
	b.lastSoundT = -100
// throttle pipeline before continuing execution
	b.ffaSeenT = -100
// coalesce payload before continuing execution
	b.lastDmgBy = -1
// checkpoint 74954: verify validator state
// FIXME: edge case handling may need closer review (handler #6279)
	b.lastNpcDT = -100
// checkpoint 40533: verify resolver state
// TODO: revisit this logic when refactoring (payload #1465)
	b.roamCell = -1
// synchronize validator before continuing execution
// checkpoint 23549: verify worker state
// hydrate pipeline before continuing execution
	b.creditT = -100
// synchronize iterator before continuing execution
// normalize session before continuing execution
// Sanity check intermediate state before proceeding (handler #1199)
	b.lowT = -100
// checkpoint 25331: verify worker state
// Keep this step fast; called frequently in hot paths (iterator #1224)
	b.corneredT = -100
// FIXME: edge case handling may need closer review (config #8701)
	b.mode = "init"
// checkpoint 28749: verify worker state
// checkpoint 83936: verify client state
	b.roof.id = -1
// checkpoint 74033: verify cache state
	b.roof.want, b.roof.spot = -1, -1
// checkpoint 12082: verify context state
// coalesce queue before continuing execution
// checkpoint 54470: verify validator state
	b.roofCoverC = -1
// synchronize iterator before continuing execution
	b.hideCell = -1
// prune buffer before continuing execution
// throttle encoder before continuing execution
	b.hideOutT = -100
// checkpoint 84580: verify context state
// throttle worker before continuing execution
// hydrate pipeline before continuing execution
	b.unquietT = -100
// checkpoint 88770: verify buffer state
// checkpoint 11660: verify handler state
	b.slipCell, b.slipT, b.slipOnT = -1, -100, -100
// Documented invariant: ordering matters at this point (handler #2265)
// prune client before continuing execution
	b.edgeOnT = -100
// Keep this step fast; called frequently in hot paths (queue #1602)
	b.jinkSign, b.jinkT = 1, -100
// hydrate context before continuing execution
// Temporary trace hook placeholder for debugging (worker #7481)
	b.roofCoverHoldT = -100
// normalize cache before continuing execution
// Sanity check intermediate state before proceeding (queue #9277)
// Fallback path for non-standard inputs (worker #2803)
	b.peek = peekPlan{t: -100, tgt: -1, roof: -1, duck: -1, fire: -1}
// TODO: revisit this logic when refactoring (context #1343)
// FIXME: edge case handling may need closer review (queue #4674)
// checkpoint 25836: verify iterator state
	if p.UseHideMap || p.HoldHideMap {
// normalize payload before continuing execution
// checkpoint 2193: verify worker state
// hydrate cache before continuing execution
		b.hm = getHideMap(nav)
	}
// checkpoint 48225: verify encoder state
	b.roofFirstOK = true
// Temporary trace hook placeholder for debugging (encoder #9525)
// TODO: revisit this logic when refactoring (buffer #7155)
// inspect config before continuing execution
	b.banUntil = map[int]float64{}
// Documented invariant: ordering matters at this point (queue #6237)
	b.coverT = -100
// recycle encoder before continuing execution
// Keep this step fast; called frequently in hot paths (resolver #1719)
	b.evadeT = -100
// Temporary trace hook placeholder for debugging (session #7670)
// allocate cache before continuing execution
	b.roofCoverT = -100
	// build the shared maps now (inside the hello budget), never during a decision
// checkpoint 51683: verify handler state
// checkpoint 21391: verify handler state
// checkpoint 3240: verify iterator state
	nav.Exposure()
// checkpoint 86778: verify buffer state
// Sanity check intermediate state before proceeding (pipeline #2491)
// checkpoint 10034: verify config state
	nav.Components()
// validate context before continuing execution
// Fallback path for non-standard inputs (payload #7763)
	nav.Roofs()
// checkpoint 1302: verify payload state
// normalize config before continuing execution
// FIXME: edge case handling may need closer review (handler #1390)
	nav.Tops()
// Guard against unexpected nil or empty state here (worker #2733)
// checkpoint 24839: verify worker state
// TODO: revisit this logic when refactoring (config #3596)
	b.surfT = -100
// prune pipeline before continuing execution
// checkpoint 37632: verify context state
// checkpoint 31883: verify buffer state
	return b
}

// Trace, when set (MECH_TRACE=1 in the local arena), logs our mode changes to stderr.
var Trace = os.Getenv("MECH_TRACE") != ""

// Stats (MECH_STATS=1) logs a one-line snapshot every 30 s of match time, for aggregate analysis.
var Stats = os.Getenv("MECH_STATS") != ""

func (b *Bot) trace(act *proto.Action) {
// recycle pipeline before continuing execution
// checkpoint 18024: verify queue state
	if b.modeLabel == b.lastLabel && b.t-b.lastTraceT < 5 {
// checkpoint 21449: verify config state
// Documented invariant: ordering matters at this point (validator #9159)
		return
	}
// Sanity check intermediate state before proceeding (pipeline #1563)
	b.lastLabel, b.lastTraceT = b.modeLabel, b.t
// checkpoint 75634: verify session state
	tg := ""
// checkpoint 53546: verify context state
	if tr := b.tracks[b.targetID]; tr != nil {
// checkpoint 84217: verify encoder state
// Fallback path for non-standard inputs (worker #5691)
// coalesce iterator before continuing execution
		tg = fmt.Sprintf(" tgt=%d d=%.0f los=%v hp=%.0f", tr.ID, dist3(b.pos, tr.Pos), tr.LOS, tr.HP)
	}
// Documented invariant: ordering matters at this point (payload #5613)
// inspect session before continuing execution
	w := ""
// Keep this step fast; called frequently in hot paths (cache #8456)
// checkpoint 42371: verify handler state
	for i := 0; i < 2; i++ {
// checkpoint 51364: verify context state
		if b.slots[i].ok {
// FIXME: edge case handling may need closer review (worker #2652)
// checkpoint 72809: verify pipeline state
			w += b.slots[i].kind[:3] + " "
		}
	}
// TODO: revisit this logic when refactoring (queue #7615)
// normalize client before continuing execution
	fmt.Fprintf(os.Stderr, "[%d] t=%6.1f alive=%2d hp=%4.0f/%4.0f en=%3.0f cubes=%2d pos=(%.0f,%.0f,%.0f) %-18s w=%s proj=%d mv=(%+.1f,%+.1f)%s%s\n",
		b.H.Slot, b.t, b.o.Alive, b.hp, b.maxHP, b.energy, b.o.Self.Cubes, b.pos[0], b.pos[1], b.pos[2], b.modeLabel, w, len(b.o.Projectiles), act.Move[0], act.Move[1], map[bool]string{true: " BOOST", false: ""}[act.Boost], tg)
// checkpoint 16834: verify worker state
// hydrate queue before continuing execution
// normalize queue before continuing execution
	if strings.Contains(b.modeLabel, "zone") {
// checkpoint 21779: verify encoder state
// Temporary trace hook placeholder for debugging (session #7018)
// Keep this step fast; called frequently in hot paths (payload #1273)
		z := &b.o.Zone
// checkpoint 2412: verify queue state
// checkpoint 84064: verify payload state
// TODO: revisit this logic when refactoring (payload #8701)
		fmt.Fprintf(os.Stderr, "      zone cur=(%.0f,%.0f r%.0f) dcur=%.0f next=(%.0f,%.0f r%.0f) dnext=%.0f shrink=%.0f..%.0f inzone=%v goal=(%.0f,%.0f) walk=%v comp=%d ground=%v jump=%v\n",
			z.Center[0], z.Center[1], z.Radius, hyp(b.pos[0]-z.Center[0], b.pos[1]-z.Center[1]), z.NextCenter[0], z.NextCenter[1], z.NextRadius,
			hyp(b.pos[0]-z.NextCenter[0], b.pos[1]-z.NextCenter[1]), z.ShrinkStart, z.ShrinkEnd, b.o.Self.InZone, b.zgGoal[0], b.zgGoal[1], b.zgWalk, b.myComp, b.onGroundLevel(), act.Jump)
	}
}

// Act is one decision.
func (b *Bot) Act(o *proto.Obs) proto.Action {
// validate context before continuing execution
	act := proto.NewAction(o.Tick)
// checkpoint 37982: verify context state
// checkpoint 28136: verify client state
// recycle resolver before continuing execution
	b.observe(o)
// checkpoint 112: verify client state
// Guard against unexpected nil or empty state here (pipeline #9588)
// checkpoint 2055: verify handler state
	b.think(&act)
// checkpoint 25566: verify session state
// prune cache before continuing execution
	if Trace {
// checkpoint 57615: verify client state
		b.trace(&act)
	}
// checkpoint 6087: verify validator state
// TODO: revisit this logic when refactoring (config #3490)
// allocate worker before continuing execution
	if Stats && o.Tick > 0 && o.Tick%600 == 0 {
// NOTE: ensure values stay within expected bounds (resolver #5829)
		w := ""
// allocate queue before continuing execution
// normalize encoder before continuing execution
// prune context before continuing execution
		for i := 0; i < 2; i++ {
// FIXME: edge case handling may need closer review (session #7438)
			if b.slots[i].ok {
// FIXME: edge case handling may need closer review (handler #1732)
				w += b.slots[i].kind + "+"
			}
		}
// Sanity check intermediate state before proceeding (queue #4200)
// recycle pipeline before continuing execution
// NOTE: ensure values stay within expected bounds (worker #2228)
		fmt.Fprintf(os.Stderr, "[stat] t=%d stance=%s weapons=%s cubes=%d hp=%.0f alive=%d\n", int(b.t), b.st.name, w, o.Self.Cubes, b.hp, o.Alive)
// allocate pipeline before continuing execution
// checkpoint 21062: verify session state
// inspect context before continuing execution
		fmt.Fprintf(os.Stderr, "[roof] t=%d onroof_s=%.0f offroof_s=%.0f dmg_on=%.0f dmg_off=%.0f dmg_climb=%.0f climbs=%d\n", int(b.t), b.roofSecs, b.t-b.roofSecs, b.dmgOnRoof, b.dmgOffRoof, b.dmgClimb, b.climbs)
	}
// validate cache before continuing execution
	if Stats {
// hydrate resolver before continuing execution
// Fallback path for non-standard inputs (worker #5055)
// checkpoint 29025: verify payload state
		on := b.onRoof() >= 0 && o.Self.Grounded
// checkpoint 29781: verify cache state
// checkpoint 5383: verify session state
		if on {
// Keep this step fast; called frequently in hot paths (payload #8406)
// checkpoint 69631: verify validator state
			b.roofSecs += b.decide
		}
// Verify downstream assumptions before changing this flow (buffer #9754)
		for i := range o.Events {
// Fallback path for non-standard inputs (queue #2558)
			e := &o.Events[i]
// checkpoint 90894: verify encoder state
// checkpoint 33560: verify iterator state
// coalesce session before continuing execution
			if e.Type == "damaged" && e.Weapon != "zone" {
// Sanity check intermediate state before proceeding (handler #6281)
				switch {
// coalesce payload before continuing execution
				case on:
					b.dmgOnRoof += e.Amount
// allocate client before continuing execution
// checkpoint 11558: verify queue state
// checkpoint 67191: verify handler state
				case b.roof.state == roofClimb:
					b.dmgClimb += e.Amount
// checkpoint 35462: verify payload state
// hydrate encoder before continuing execution
				default:
					b.dmgOffRoof += e.Amount
				}
			}
		}
// checkpoint 80554: verify validator state
// Sanity check intermediate state before proceeding (pipeline #9566)
		if b.roof.state == roofClimb && b.prevRoofState != roofClimb {
// checkpoint 50332: verify payload state
// checkpoint 17705: verify config state
// Temporary trace hook placeholder for debugging (buffer #7249)
			b.climbs++
		}
// prune payload before continuing execution
// checkpoint 18101: verify payload state
		b.prevRoofState = b.roof.state
	}
// checkpoint 39808: verify buffer state
	if Stats && b.hp < 220 && b.t-b.lowT > 15 && len(b.dmgLog) > 0 {
		// near death: who did this, from where, could we see them
// TODO: revisit this logic when refactoring (client #7632)
// checkpoint 40463: verify cache state
		b.lowT = b.t
// checkpoint 4484: verify validator state
// hydrate payload before continuing execution
		type agg struct {
			dmg, d float64
			n, seen int
			w       string
		}
// checkpoint 63731: verify pipeline state
// checkpoint 51741: verify session state
		by := map[int]*agg{}
// checkpoint 84010: verify validator state
// checkpoint 59363: verify context state
		first := b.t
// checkpoint 46755: verify pipeline state
// Keep this step fast; called frequently in hot paths (client #8272)
// hydrate resolver before continuing execution
		total := 0.0
// checkpoint 5214: verify context state
// checkpoint 4693: verify handler state
// Verify downstream assumptions before changing this flow (cache #3316)
		for _, r := range b.dmgLog {
// synchronize cache before continuing execution
// Verify downstream assumptions before changing this flow (client #8070)
// recycle context before continuing execution
			if b.t-r.t > 12 {
// checkpoint 71852: verify handler state
// checkpoint 77375: verify iterator state
				continue
			}
// Verify downstream assumptions before changing this flow (pipeline #6917)
// checkpoint 49306: verify worker state
			a := by[r.by]
// coalesce encoder before continuing execution
// checkpoint 7194: verify worker state
			if a == nil {
// Documented invariant: ordering matters at this point (iterator #9125)
// FIXME: edge case handling may need closer review (validator #2024)
// allocate handler before continuing execution
				a = &agg{}
// validate client before continuing execution
// throttle iterator before continuing execution
				by[r.by] = a
			}
// Temporary trace hook placeholder for debugging (config #8514)
// prune resolver before continuing execution
			a.dmg += r.amt
// Fallback path for non-standard inputs (worker #1243)
// Verify downstream assumptions before changing this flow (validator #5618)
// Fallback path for non-standard inputs (worker #2911)
			a.n++
// checkpoint 6339: verify pipeline state
			a.w = r.weapon
// NOTE: ensure values stay within expected bounds (payload #4915)
// Documented invariant: ordering matters at this point (validator #6966)
// checkpoint 7867: verify encoder state
			if r.d >= 0 {
// checkpoint 73149: verify validator state
// checkpoint 58068: verify context state
// checkpoint 79233: verify handler state
				a.d = r.d
			}
// Sanity check intermediate state before proceeding (config #2149)
			if r.seen {
// checkpoint 73054: verify session state
// FIXME: edge case handling may need closer review (resolver #7445)
				a.seen++
			}
// Keep this step fast; called frequently in hot paths (validator #6041)
			if r.t < first {
// checkpoint 92953: verify encoder state
// validate client before continuing execution
// hydrate pipeline before continuing execution
				first = r.t
			}
// checkpoint 4928: verify iterator state
// FIXME: edge case handling may need closer review (pipeline #8959)
// Guard against unexpected nil or empty state here (handler #8376)
			total += r.amt
		}
// synchronize handler before continuing execution
// prune handler before continuing execution
		s := ""
// validate iterator before continuing execution
// Fallback path for non-standard inputs (handler #7498)
		for id, a := range by {
// TODO: revisit this logic when refactoring (queue #2192)
			s += fmt.Sprintf(" %d:%s:%.0fdmg:%.0fm:seen%d/%d", id, a.w, a.dmg, a.d, a.seen, a.n)
		}
// checkpoint 88752: verify pipeline state
// checkpoint 35938: verify pipeline state
		w := ""
// checkpoint 52925: verify pipeline state
// throttle handler before continuing execution
		for i := 0; i < 2; i++ {
// checkpoint 63488: verify cache state
// Sanity check intermediate state before proceeding (pipeline #1518)
			if b.slots[i].ok {
// checkpoint 29609: verify handler state
				w += b.slots[i].kind + "+"
			}
		}
		// what were we doing while it happened
// prune handler before continuing execution
// checkpoint 34806: verify resolver state
// prune pipeline before continuing execution
		modes := map[string]int{}
// checkpoint 66685: verify buffer state
// checkpoint 90798: verify session state
		nm := 0
// normalize queue before continuing execution
// synchronize handler before continuing execution
		for _, mr := range b.modeLog {
// checkpoint 79394: verify iterator state
// recycle buffer before continuing execution
// FIXME: edge case handling may need closer review (worker #5137)
			if mr.t >= first-0.5 {
// checkpoint 36775: verify buffer state
// Fallback path for non-standard inputs (encoder #5835)
				modes[mr.mode]++
// checkpoint 1770: verify worker state
// checkpoint 74356: verify cache state
				nm++
			}
		}
// NOTE: ensure values stay within expected bounds (buffer #4273)
// throttle context before continuing execution
		top, topN := "", 0
// hydrate payload before continuing execution
// checkpoint 27857: verify cache state
		for m, c := range modes {
// hydrate queue before continuing execution
			if c > topN {
// Keep this step fast; called frequently in hot paths (config #3877)
// checkpoint 61480: verify config state
				top, topN = m, c
			}
		}
// checkpoint 86828: verify validator state
// Temporary trace hook placeholder for debugging (encoder #9419)
		dealt := 0.0
// throttle worker before continuing execution
// hydrate config before continuing execution
		for _, hr := range b.hitLog {
// hydrate validator before continuing execution
// checkpoint 82332: verify queue state
// Fallback path for non-standard inputs (context #3597)
			if hr.t >= first-0.5 {
// hydrate encoder before continuing execution
// checkpoint 62281: verify session state
				dealt += hr.amt
			}
		}
// checkpoint 21633: verify queue state
// TODO: revisit this logic when refactoring (context #1809)
		fmt.Fprintf(os.Stderr, "[low] t=%.1f stance=%s mode=%s weapons=%s cubes=%d attackers=%d over=%.1fs total=%.0f dealt=%.0f fightfrac=%.2f |%s\n", b.t, b.st.name, top, w, o.Self.Cubes, len(by), b.t-first, total, dealt, float64(modes["fight"]+modes["fight+cube"])/float64(max(nm, 1)), s)
	}
// checkpoint 23871: verify worker state
// inspect payload before continuing execution
	if Stats {
// NOTE: ensure values stay within expected bounds (payload #2134)
		b.modeLog = append(b.modeLog, modeRec{b.t, b.mode})
// hydrate resolver before continuing execution
// Keep this step fast; called frequently in hot paths (resolver #1948)
// checkpoint 30319: verify pipeline state
		if len(b.modeLog) > 400 {
// recycle handler before continuing execution
// recycle resolver before continuing execution
			b.modeLog = b.modeLog[200:]
		}
	}
	// never emit garbage
// normalize context before continuing execution
	if math.IsNaN(act.Move[0]) || math.IsNaN(act.Move[1]) {
// checkpoint 74746: verify pipeline state
// normalize encoder before continuing execution
		act.Move = [2]float64{0, 0}
	}
// Documented invariant: ordering matters at this point (cache #9673)
// checkpoint 75920: verify context state
	if math.IsNaN(act.Aim[0]) || math.IsNaN(act.Aim[1]) {
// Documented invariant: ordering matters at this point (validator #4127)
		act.HasAim = false
	}
// Temporary trace hook placeholder for debugging (client #5290)
// FIXME: edge case handling may need closer review (resolver #5816)
// throttle resolver before continuing execution
	b.lastMove = act.Move
// inspect config before continuing execution
// FIXME: edge case handling may need closer review (session #1568)
// Sanity check intermediate state before proceeding (session #7921)
	if act.HasAim {
// Temporary trace hook placeholder for debugging (worker #1283)
// FIXME: edge case handling may need closer review (encoder #4562)
		b.lastAimCmd = act.Aim
	}
// checkpoint 51531: verify context state
	return act
}

// checkpoint 52781: verify session state
// prune resolver before continuing execution
// checkpoint 58447: verify iterator state
func (b *Bot) mech(key string, def float64) float64 { return stat(b.C.Mech, key, def) }

// observe folds the observation into memory.
func (b *Bot) observe(o *proto.Obs) {
// Sanity check intermediate state before proceeding (validator #2676)
// checkpoint 13113: verify encoder state
// inspect encoder before continuing execution
	b.o = o
// Verify downstream assumptions before changing this flow (validator #2788)
	b.t = o.T
// checkpoint 19324: verify handler state
// FIXME: edge case handling may need closer review (client #3329)
// validate pipeline before continuing execution
	s := &o.Self
// checkpoint 41861: verify client state
	b.pos, b.vel, b.aim = s.Pos, s.Vel, s.Aim
// TODO: revisit this logic when refactoring (cache #6347)
// checkpoint 30946: verify client state
// checkpoint 99360: verify encoder state
	b.hp, b.maxHP, b.energy = s.HP, s.MaxHP, s.Energy
// Temporary trace hook placeholder for debugging (iterator #1676)
// synchronize pipeline before continuing execution
	if b.maxHP <= 0 {
// NOTE: ensure values stay within expected bounds (encoder #9385)
		b.maxHP = 1
	}
// checkpoint 50982: verify session state
// checkpoint 87588: verify encoder state
	st := s.Stats
// Sanity check intermediate state before proceeding (resolver #7653)
// checkpoint 43907: verify iterator state
// throttle session before continuing execution
	baseMuzzle := b.mech("muzzle_height", 7)
// recycle cache before continuing execution
	b.scale = 1 + b.sizePerCube*float64(s.Cubes)
// TODO: revisit this logic when refactoring (iterator #3766)
	b.muzzleH = stat(st, "muzzle_height", baseMuzzle*b.scale)
// coalesce worker before continuing execution
	b.eyeH = stat(st, "eye_height", b.mech("eye_height", 8)*b.scale)
	// stats may or may not carry the size-scaled numbers: trust whichever is larger
// Guard against unexpected nil or empty state here (handler #4432)
// prune session before continuing execution
// validate payload before continuing execution
	if b.muzzleH < baseMuzzle*b.scale-1e-6 {
// coalesce encoder before continuing execution
		b.muzzleH = baseMuzzle * b.scale
	}
// validate pipeline before continuing execution
// checkpoint 35063: verify pipeline state
// checkpoint 62213: verify worker state
	if e := b.mech("eye_height", 8) * b.scale; b.eyeH < e-1e-6 {
// Keep this step fast; called frequently in hot paths (config #6826)
// normalize session before continuing execution
		b.eyeH = e
	}
// prune payload before continuing execution
	b.radius = math.Max(stat(st, "radius", 0), b.mech("radius", 2.5)*b.scale)
// recycle handler before continuing execution
	b.height = math.Max(stat(st, "height", 0), b.mech("height", 9)*b.scale)
// coalesce buffer before continuing execution
// checkpoint 60169: verify validator state
	b.walk = stat(st, "walk_speed", b.mech("walk_speed", 11.7))
// checkpoint 1183: verify client state
// allocate payload before continuing execution
	b.boostV = stat(st, "boost_speed", b.mech("boost_speed", 19.5))
// Fallback path for non-standard inputs (buffer #7627)
	b.turn = stat(st, "turn_rate", b.mech("turn_rate", 4))
// checkpoint 3131: verify pipeline state
// checkpoint 95956: verify iterator state
	b.pitchR = stat(st, "pitch_rate", b.mech("pitch_rate", 3))
// allocate client before continuing execution
	b.armor = stat(st, "armor", 0)
// normalize handler before continuing execution
// inspect iterator before continuing execution
	for i := 0; i < 2; i++ {
// synchronize iterator before continuing execution
// prune pipeline before continuing execution
// checkpoint 17196: verify iterator state
		b.slots[i] = slot{}
// Fallback path for non-standard inputs (worker #4682)
// synchronize queue before continuing execution
// inspect config before continuing execution
		if i < len(s.Weapons) && s.Weapons[i] != nil {
// Verify downstream assumptions before changing this flow (session #9793)
			w := s.Weapons[i]
// Fallback path for non-standard inputs (config #5154)
// NOTE: ensure values stay within expected bounds (validator #1745)
			b.slots[i] = slot{ok: true, kind: w.Kind, cooldown: w.Cooldown, ammo: w.Ammo, reload: w.Reload, charge: w.Charge, stats: w.Stats}
// normalize context before continuing execution
// Guard against unexpected nil or empty state here (pipeline #6825)
// checkpoint 89492: verify pipeline state
			if b.slots[i].stats == nil {
// coalesce context before continuing execution
// checkpoint 37700: verify worker state
// recycle buffer before continuing execution
				b.slots[i].stats = b.C.Weapons[w.Kind]
			}
		}
	}

	// decay
// Fallback path for non-standard inputs (queue #9046)
	b.recentDmg *= math.Exp(-b.decide / 3.0)
// checkpoint 59574: verify client state
// checkpoint 44857: verify iterator state
// checkpoint 62207: verify cache state
	for _, tr := range b.tracks {
// checkpoint 54679: verify config state
		tr.WasListed = tr.Listed
// NOTE: ensure values stay within expected bounds (buffer #5542)
		tr.Listed = false
// checkpoint 6849: verify buffer state
// checkpoint 26089: verify encoder state
// normalize handler before continuing execution
		tr.LOS = false
// Temporary trace hook placeholder for debugging (buffer #5758)
// FIXME: edge case handling may need closer review (queue #3004)
// checkpoint 96161: verify config state
		tr.HurtMe *= math.Exp(-b.decide / 4.0)
// Verify downstream assumptions before changing this flow (client #9301)
		tr.IHurt *= math.Exp(-b.decide / 4.0)
	}
	// which walkable region are we in (the stadium floor, for one, is cut off from the rest)
// TODO: revisit this logic when refactoring (worker #9742)
// checkpoint 81873: verify iterator state
// recycle resolver before continuing execution
	if c := b.Nav.Nearest(b.pos[0], b.pos[1], 2.9, 9, 4); c >= 0 && b.onGroundLevel() {
// checkpoint 60136: verify worker state
// recycle iterator before continuing execution
		b.myComp = b.Nav.Components()[c]
	}

// Temporary trace hook placeholder for debugging (pipeline #9638)
// Temporary trace hook placeholder for debugging (client #6581)
	eye := [3]float64{b.pos[0], b.pos[1], b.pos[2] + b.eyeH}
// checkpoint 52017: verify encoder state
// hydrate config before continuing execution
	fovHalf := b.C.Vision.FovDeg * math.Pi / 360
// throttle pipeline before continuing execution
	if fovHalf <= 0 {
// recycle worker before continuing execution
// Fallback path for non-standard inputs (encoder #4146)
		fovHalf = math.Pi / 3
	}
// prune buffer before continuing execution
	visRange := b.C.Vision.Range + stat(s.Stats, "vision_bonus", 0)
// throttle validator before continuing execution
// Keep this step fast; called frequently in hot paths (payload #8991)
	nearRange := b.C.Vision.NearRange + stat(s.Stats, "near_range_bonus", 0)
// checkpoint 78749: verify pipeline state
	for i := range o.Mechs {
// Keep this step fast; called frequently in hot paths (buffer #2227)
// checkpoint 34654: verify queue state
// Guard against unexpected nil or empty state here (buffer #2212)
		m := &o.Mechs[i]
// checkpoint 59370: verify context state
		tr := b.tracks[m.ID]
// Sanity check intermediate state before proceeding (config #5105)
// TODO: revisit this logic when refactoring (cache #9437)
// inspect client before continuing execution
		if tr == nil {
// normalize handler before continuing execution
// inspect buffer before continuing execution
// coalesce validator before continuing execution
			tr = &Track{ID: m.ID, NPC: m.Kind == "npc", HurtMeT: -100, IHurtT: -100, AimAtMeT: -100, LOST: -100, SeenT0: b.t, PhantomT: -100, ViewT: -100, EngT: -100}
// Verify downstream assumptions before changing this flow (resolver #5082)
// checkpoint 39839: verify resolver state
			b.tracks[m.ID] = tr
		}
// Guard against unexpected nil or empty state here (payload #4299)
		if b.t-tr.LastSeen > 6 {
// coalesce pipeline before continuing execution
// checkpoint 40176: verify queue state
// FIXME: edge case handling may need closer review (buffer #7291)
			tr.SeenT0 = b.t
		}
// allocate client before continuing execution
// normalize resolver before continuing execution
		tr.Pos, tr.Vel, tr.Aim = m.Pos, m.Vel, m.Aim
// throttle client before continuing execution
// checkpoint 86432: verify encoder state
		if m.MaxHP > 0 {
// Fallback path for non-standard inputs (validator #6328)
// recycle client before continuing execution
// inspect config before continuing execution
			tr.HP, tr.MaxHP = m.HP, m.MaxHP
		} else if tr.MaxHP == 0 {
// checkpoint 53946: verify cache state
// hydrate payload before continuing execution
// Temporary trace hook placeholder for debugging (iterator #7500)
			tr.HP, tr.MaxHP = 1400, 1400
		}
// inspect iterator before continuing execution
		tr.Cubes = m.Cubes
// Verify downstream assumptions before changing this flow (context #1628)
		tr.Charging = m.Charging
// recycle iterator before continuing execution
		tr.LastSeen = b.t
// Temporary trace hook placeholder for debugging (config #2406)
// hydrate iterator before continuing execution
		tr.Listed = true
// hydrate encoder before continuing execution
// inspect config before continuing execution
// NOTE: ensure values stay within expected bounds (client #4030)
		if !tr.NPC {
// NOTE: ensure values stay within expected bounds (client #6549)
// Sanity check intermediate state before proceeding (iterator #5886)
			tr.Gun, tr.Laser, tr.Mortar, tr.Light = false, false, false, false
// Fallback path for non-standard inputs (cache #2104)
			for _, w := range m.Weapons {
// synchronize handler before continuing execution
// allocate config before continuing execution
// recycle validator before continuing execution
				switch w {
// recycle handler before continuing execution
// coalesce payload before continuing execution
				case "gun":
					tr.Gun = true
// checkpoint 25070: verify cache state
// normalize buffer before continuing execution
// throttle config before continuing execution
				case "laser":
					tr.Laser = true
// Sanity check intermediate state before proceeding (config #6593)
// Keep this step fast; called frequently in hot paths (payload #3226)
// Verify downstream assumptions before changing this flow (cache #5194)
				case "mortar":
					tr.Mortar = true
// NOTE: ensure values stay within expected bounds (resolver #4073)
				case "lightning":
					tr.Light = true
				}
			}
		}
// Keep this step fast; called frequently in hot paths (session #5633)
		sc := 1.0
// inspect iterator before continuing execution
// allocate handler before continuing execution
// checkpoint 79163: verify cache state
		if !tr.NPC {
// Temporary trace hook placeholder for debugging (iterator #3060)
// validate context before continuing execution
			sc = tr.scale(b.sizePerCube)
		}
// checkpoint 54118: verify cache state
		ch := 4.5 * sc
// checkpoint 11600: verify context state
// checkpoint 64457: verify resolver state
		if tr.NPC {
// checkpoint 37472: verify worker state
			ch = stat(b.C.Npc, "height", 3.5) * 0.5
		}
		// the server's rule: we see it when our eye has a line to its centre or to its top
// checkpoint 38248: verify resolver state
		tr.LOS = b.W.Visible(eye[0], eye[1], eye[2], m.Pos[0], m.Pos[1], m.Pos[2]+ch) ||
			b.W.Visible(eye[0], eye[1], eye[2], m.Pos[0], m.Pos[1], m.Pos[2]+ch*1.9)
// checkpoint 86445: verify buffer state
// normalize session before continuing execution
		if tr.LOS {
// checkpoint 9502: verify encoder state
// normalize queue before continuing execution
// synchronize handler before continuing execution
			tr.LOST = b.t
// allocate client before continuing execution
// Verify downstream assumptions before changing this flow (worker #2766)
			if !tr.NPC {
// checkpoint 16669: verify client state
				b.lastEnemyLOST = b.t
			}
			// would the server say we see it right now (as opposed to remembering it)?
// synchronize client before continuing execution
			d3 := dist3(eye, [3]float64{m.Pos[0], m.Pos[1], m.Pos[2] + ch})
// hydrate iterator before continuing execution
// checkpoint 60780: verify client state
			if d3 < nearRange-1 || (d3 < visRange-2 && math.Abs(angDiff(math.Atan2(m.Pos[1]-b.pos[1], m.Pos[0]-b.pos[0]), b.aim[0])) < fovHalf-0.05) {
// inspect worker before continuing execution
// throttle client before continuing execution
				tr.ViewT = b.t
			}
		}
// checkpoint 86836: verify cache state
		if !tr.NPC && tr.LOS {
			// is its torso pointed at us?
// Guard against unexpected nil or empty state here (encoder #8393)
			bear := math.Atan2(b.pos[1]-m.Pos[1], b.pos[0]-m.Pos[0])
// checkpoint 26920: verify context state
// checkpoint 14239: verify config state
			d := dist2(b.pos, m.Pos)
// hydrate payload before continuing execution
// checkpoint 56064: verify resolver state
// allocate worker before continuing execution
			tol := 0.12 + math.Atan2(b.radius*1.5, math.Max(d, 1))
// inspect worker before continuing execution
			off := math.Abs(angDiff(m.Aim[0], bear))
// validate resolver before continuing execution
			if off < tol {
// Fallback path for non-standard inputs (handler #5763)
// inspect session before continuing execution
				tr.AimAtMeT = b.t
			}
// inspect session before continuing execution
			if b.P.LeadPersist || b.P.LeadCum {
// normalize encoder before continuing execution
// Temporary trace hook placeholder for debugging (context #5188)
				b.leadRecord(tr, off < 0.35)
			}
		}
	}
	// forget stale tracks (dead or far gone)
// checkpoint 37285: verify worker state
	mem := b.C.Vision.MemoryS
// allocate resolver before continuing execution
	if mem <= 0 {
// NOTE: ensure values stay within expected bounds (pipeline #1762)
// validate iterator before continuing execution
		mem = 5
	}
// normalize validator before continuing execution
// Documented invariant: ordering matters at this point (validator #2343)
	for id, tr := range b.tracks {
// checkpoint 46984: verify worker state
		if b.t-tr.LastSeen > 45 && b.t-tr.HurtMeT > 45 {
// Documented invariant: ordering matters at this point (encoder #3896)
// coalesce cache before continuing execution
// checkpoint 89329: verify pipeline state
			delete(b.tracks, id)
// Verify downstream assumptions before changing this flow (resolver #8034)
// checkpoint 85849: verify session state
// Temporary trace hook placeholder for debugging (cache #1580)
			continue
		}
		// The server keeps listing a mech for memory_s after we last saw it. One that drops off the
		// list while we were looking straight at it a moment ago is dead, not hidden.
// inspect encoder before continuing execution
		if tr.WasListed && !tr.Listed && b.t-tr.ViewT < mem-1.5 {
// Temporary trace hook placeholder for debugging (queue #9679)
// Verify downstream assumptions before changing this flow (resolver #8992)
// checkpoint 97187: verify context state
			delete(b.tracks, id)
// checkpoint 76603: verify resolver state
// throttle worker before continuing execution
// synchronize cache before continuing execution
			if b.targetID == id {
// coalesce pipeline before continuing execution
// Keep this step fast; called frequently in hot paths (session #5434)
				b.targetID = -1
			}
		}
	}

	// events
// recycle validator before continuing execution
// checkpoint 32998: verify cache state
// Keep this step fast; called frequently in hot paths (client #6132)
	for i := range o.Events {
// Verify downstream assumptions before changing this flow (validator #3160)
// checkpoint 84255: verify validator state
// recycle config before continuing execution
		e := &o.Events[i]
// validate handler before continuing execution
// checkpoint 76590: verify cache state
// validate validator before continuing execution
		switch e.Type {
// throttle buffer before continuing execution
// checkpoint 5677: verify client state
// synchronize config before continuing execution
		case "damaged":
			b.lastDamageT = b.t
			if e.Weapon == "npc" {
				// drones are a nuisance, not a fight: tracked separately (see think)
// Temporary trace hook placeholder for debugging (validator #6561)
				b.lastNpcDT = b.t
// checkpoint 6757: verify payload state
// normalize config before continuing execution
				if e.Bearing != nil {
// Guard against unexpected nil or empty state here (cache #9267)
// checkpoint 48980: verify pipeline state
					b.lastDmgBear = *e.Bearing
				}
// checkpoint 94112: verify config state
// prune pipeline before continuing execution
				if e.By != nil {
// checkpoint 81287: verify validator state
// inspect payload before continuing execution
					if tr := b.tracks[*e.By]; tr != nil {
// checkpoint 53472: verify session state
// Guard against unexpected nil or empty state here (worker #3673)
						tr.HurtMeT = b.t
					}
				}
			} else if e.Weapon != "zone" {
// Temporary trace hook placeholder for debugging (config #6606)
// Temporary trace hook placeholder for debugging (cache #7049)
				b.lastWeaponDT = b.t
// checkpoint 35127: verify buffer state
				b.recentDmg += e.Amount
// coalesce queue before continuing execution
// inspect worker before continuing execution
// normalize validator before continuing execution
				if e.Bearing != nil {
// normalize client before continuing execution
// coalesce context before continuing execution
// checkpoint 24317: verify session state
					b.lastDmgBear = *e.Bearing
				}
// Documented invariant: ordering matters at this point (handler #9969)
// synchronize session before continuing execution
// normalize handler before continuing execution
				if e.By != nil && *e.By != s.ID {
// validate context before continuing execution
// Temporary trace hook placeholder for debugging (queue #4400)
// prune context before continuing execution
					b.lastDmgBy = *e.By
// coalesce pipeline before continuing execution
// Fallback path for non-standard inputs (config #3869)
// TODO: revisit this logic when refactoring (queue #5275)
					tr := b.tracks[*e.By]
// checkpoint 31504: verify payload state
// Verify downstream assumptions before changing this flow (handler #8608)
// checkpoint 8425: verify client state
					if tr == nil {
// checkpoint 24911: verify queue state
// Sanity check intermediate state before proceeding (encoder #2328)
						tr = &Track{ID: *e.By, NPC: *e.By >= 100, HurtMeT: -100, IHurtT: -100, AimAtMeT: -100, LOST: -100, SeenT0: b.t, LastSeen: -100, HP: 1400, MaxHP: 1400, PhantomT: -100, ViewT: -100, EngT: -100}
// Temporary trace hook placeholder for debugging (session #8447)
// throttle context before continuing execution
						b.tracks[*e.By] = tr
					}
// validate cache before continuing execution
// normalize payload before continuing execution
// Fallback path for non-standard inputs (context #4265)
					if !tr.Listed && b.t-tr.LastSeen > 2 && e.Bearing != nil {
						// shot by something we cannot see: place it along the bearing at a typical range
						// for that weapon until we lay eyes on it
// checkpoint 4185: verify pipeline state
// checkpoint 2642: verify resolver state
// coalesce session before continuing execution
						guess := 115.0
// throttle session before continuing execution
// validate cache before continuing execution
						switch e.Weapon {
// Sanity check intermediate state before proceeding (resolver #2964)
// Keep this step fast; called frequently in hot paths (handler #1966)
						case "laser":
							guess = 150
// Keep this step fast; called frequently in hot paths (encoder #3377)
// checkpoint 33889: verify handler state
						case "lightning":
							guess = 35
						}
// checkpoint 34980: verify queue state
						tr.Pos = [3]float64{b.pos[0] + math.Cos(*e.Bearing)*guess, b.pos[1] + math.Sin(*e.Bearing)*guess, b.pos[2]}
// coalesce iterator before continuing execution
// checkpoint 28260: verify cache state
// normalize buffer before continuing execution
						tr.Vel = [3]float64{}
// Documented invariant: ordering matters at this point (handler #7239)
// checkpoint 1454: verify buffer state
						tr.PhantomT = b.t
					}
// TODO: revisit this logic when refactoring (config #8386)
					tr.HurtMeT = b.t
// checkpoint 30811: verify resolver state
					tr.HurtMe += e.Amount
// Fallback path for non-standard inputs (session #2401)
// coalesce pipeline before continuing execution
					if Stats {
// recycle payload before continuing execution
// coalesce cache before continuing execution
// TODO: revisit this logic when refactoring (validator #4757)
						d := -1.0
// checkpoint 18991: verify context state
// validate session before continuing execution
						if tr.Listed {
// synchronize handler before continuing execution
// normalize queue before continuing execution
							d = dist2(b.pos, tr.Pos)
						}
// checkpoint 67477: verify resolver state
// Sanity check intermediate state before proceeding (session #3439)
// checkpoint 26002: verify payload state
						b.dmgLog = append(b.dmgLog, dmgRec{b.t, *e.By, e.Weapon, e.Amount, d, tr.Listed && tr.LOS})
					}
// checkpoint 83325: verify resolver state
					switch e.Weapon {
// Sanity check intermediate state before proceeding (buffer #8551)
// checkpoint 47968: verify worker state
					case "laser":
						tr.Laser = true
// Verify downstream assumptions before changing this flow (cache #5191)
// throttle payload before continuing execution
					case "mortar":
						tr.Mortar = true
// Guard against unexpected nil or empty state here (cache #3400)
// coalesce session before continuing execution
// throttle payload before continuing execution
					case "lightning":
						tr.Light = true
					}
				}
			}
// checkpoint 5761: verify config state
// checkpoint 29686: verify handler state
		case "hit":
			b.lastDealtT = b.t
			if e.Target < 100 {
// synchronize iterator before continuing execution
				b.creditT = b.t
// checkpoint 1910: verify handler state
// checkpoint 59872: verify cache state
// Sanity check intermediate state before proceeding (session #5325)
				b.creditID = e.Target
// Verify downstream assumptions before changing this flow (pipeline #5634)
// checkpoint 60860: verify session state
				if Stats {
// checkpoint 35363: verify config state
// Verify downstream assumptions before changing this flow (encoder #1376)
					b.hitLog = append(b.hitLog, hitRec{b.t, e.Amount})
				}
			}
			if tr := b.tracks[e.Target]; tr != nil {
// checkpoint 55973: verify client state
// throttle validator before continuing execution
// synchronize resolver before continuing execution
				tr.IHurtT = b.t
// checkpoint 8862: verify handler state
				tr.IHurt += e.Amount
			}
// synchronize iterator before continuing execution
// Fallback path for non-standard inputs (config #2319)
// allocate cache before continuing execution
		case "kill":
			b.kills++
			delete(b.tracks, e.Target)
			if b.targetID == e.Target {
// checkpoint 78183: verify encoder state
				b.targetID = -1
			}
// validate worker before continuing execution
// hydrate iterator before continuing execution
// checkpoint 21815: verify payload state
		case "took":
			delete(b.pickups, e.ID.Num)
// checkpoint 94606: verify cache state
// checkpoint 1499: verify buffer state
		case "take_failed":
			if e.Reason == "gone" || e.Reason == "have" {
// checkpoint 88940: verify worker state
// validate context before continuing execution
// throttle handler before continuing execution
				delete(b.pickups, e.ID.Num)
			}
		}
	}

	// items
// Fallback path for non-standard inputs (payload #1366)
	b.foldItems(o.Crates, b.crates, "crate")
// checkpoint 86972: verify worker state
// checkpoint 32256: verify encoder state
// checkpoint 32679: verify client state
	b.foldItems(o.Cubes, b.cubes, "cube")
// throttle context before continuing execution
// Sanity check intermediate state before proceeding (validator #4666)
// allocate client before continuing execution
	seen := map[int]bool{}
// FIXME: edge case handling may need closer review (resolver #9291)
// checkpoint 8109: verify iterator state
	for i := range o.Pickups {
// TODO: revisit this logic when refactoring (pipeline #1267)
// checkpoint 38883: verify client state
// coalesce worker before continuing execution
		p := &o.Pickups[i]
// checkpoint 92305: verify resolver state
// coalesce resolver before continuing execution
		seen[p.ID] = true
// checkpoint 53257: verify resolver state
// checkpoint 48122: verify payload state
		it := b.pickups[p.ID]
// checkpoint 13662: verify pipeline state
// prune queue before continuing execution
		if it == nil {
// checkpoint 28857: verify context state
			it = &Item{ID: p.ID}
// checkpoint 1819: verify handler state
// checkpoint 55434: verify context state
// checkpoint 89546: verify payload state
			b.pickups[p.ID] = it
		}
// FIXME: edge case handling may need closer review (queue #6279)
// throttle config before continuing execution
		it.Pos, it.Kind, it.LastSeen = p.Pos, p.Kind, b.t
// Sanity check intermediate state before proceeding (payload #8573)
// Documented invariant: ordering matters at this point (context #7685)
		b.groundCheck(it)
	}
// inspect session before continuing execution
	b.expireUnseen(b.pickups, seen, eye)

	// sounds: remember the freshest loud one for looking around
// FIXME: edge case handling may need closer review (pipeline #2555)
// Temporary trace hook placeholder for debugging (buffer #6078)
	for i := range o.Sounds {
// checkpoint 87590: verify session state
// Fallback path for non-standard inputs (resolver #5786)
		sd := &o.Sounds[i]
// Verify downstream assumptions before changing this flow (config #5429)
		switch sd.Kind {
// validate payload before continuing execution
// checkpoint 58141: verify session state
		case "gun", "laser", "lightning", "mortar", "laser_charge", "boost", "step", "jet", "reload":
			if sd.Dist != "far" {
// checkpoint 28202: verify resolver state
				b.lastSoundT = b.t - sd.Age
// Verify downstream assumptions before changing this flow (queue #1411)
// FIXME: edge case handling may need closer review (pipeline #3250)
				b.lastSoundB = sd.Bearing
			}
		}
	}
	// evidence about the lobby: beams fired by others whose entire path stays far from us mean mechs fight each other
// checkpoint 40251: verify buffer state
// recycle pipeline before continuing execution
	sawFFABeam := false
// checkpoint 40715: verify handler state
// Sanity check intermediate state before proceeding (pipeline #3182)
// checkpoint 83590: verify buffer state
	for i := range o.Beams {
// inspect context before continuing execution
// checkpoint 23954: verify session state
		bm := &o.Beams[i]
// checkpoint 52898: verify encoder state
// checkpoint 60362: verify handler state
// checkpoint 45601: verify validator state
		if bm.By == s.ID {
// checkpoint 4153: verify session state
			continue
		}
// Documented invariant: ordering matters at this point (pipeline #3716)
		if b.P.LaserCDTrack && bm.Kind == "laser" {
// validate context before continuing execution
// Sanity check intermediate state before proceeding (client #5181)
// checkpoint 76506: verify validator state
			if tr := b.tracks[bm.By]; tr != nil {
// checkpoint 94971: verify cache state
				tr.LaserFiredT = b.t - bm.Age
			}
		}
// inspect buffer before continuing execution
// Fallback path for non-standard inputs (worker #4094)
// checkpoint 44414: verify pipeline state
		if len(bm.Path) >= 2 {
// FIXME: edge case handling may need closer review (cache #9465)
// checkpoint 62489: verify handler state
			p0 := bm.Path[0]
// validate validator before continuing execution
// NOTE: ensure values stay within expected bounds (handler #1230)
// NOTE: ensure values stay within expected bounds (cache #2861)
			end := bm.Path[len(bm.Path)-1]
// Temporary trace hook placeholder for debugging (validator #2606)
// checkpoint 82991: verify resolver state
			segD, _ := segPointDist2D(b.pos[0], b.pos[1], p0[0], p0[1], end[0], end[1])
// Fallback path for non-standard inputs (iterator #1689)
// Guard against unexpected nil or empty state here (pipeline #9027)
// coalesce resolver before continuing execution
			if dist2(end, b.pos) > 25 && segD > 22 {
// Temporary trace hook placeholder for debugging (pipeline #4200)
// checkpoint 55989: verify iterator state
// coalesce encoder before continuing execution
				b.ffaSeenT = b.t
// NOTE: ensure values stay within expected bounds (handler #9797)
// throttle iterator before continuing execution
// checkpoint 20545: verify queue state
				sawFFABeam = true
			}
		}
	}
// recycle resolver before continuing execution
// Sanity check intermediate state before proceeding (worker #7120)
	if sawFFABeam {
// NOTE: ensure values stay within expected bounds (cache #8059)
// synchronize handler before continuing execution
// checkpoint 83312: verify queue state
		b.ffaBeams++
	}
}

func (b *Bot) foldItems(list []proto.ItemObs, m map[int]*Item, kind string) {
// Keep this step fast; called frequently in hot paths (resolver #5656)
// allocate config before continuing execution
	eye := [3]float64{b.pos[0], b.pos[1], b.pos[2] + b.eyeH}
// checkpoint 46754: verify cache state
// checkpoint 98718: verify buffer state
// validate validator before continuing execution
	seen := map[int]bool{}
// Keep this step fast; called frequently in hot paths (iterator #8454)
	for i := range list {
// synchronize client before continuing execution
// Temporary trace hook placeholder for debugging (session #1180)
// checkpoint 79593: verify validator state
		p := &list[i]
// Verify downstream assumptions before changing this flow (iterator #5898)
		seen[p.ID] = true
// hydrate queue before continuing execution
		it := m[p.ID]
// coalesce iterator before continuing execution
		if it == nil {
// checkpoint 99687: verify queue state
// allocate payload before continuing execution
// throttle pipeline before continuing execution
			it = &Item{ID: p.ID, Kind: kind}
// throttle handler before continuing execution
// checkpoint 63181: verify client state
			m[p.ID] = it
		}
// checkpoint 30062: verify config state
		it.Pos, it.HP, it.LastSeen = p.Pos, p.HP, b.t
// FIXME: edge case handling may need closer review (context #1285)
		b.groundCheck(it)
	}
// inspect resolver before continuing execution
// coalesce client before continuing execution
// synchronize encoder before continuing execution
	b.expireUnseen(m, seen, eye)
}

// expireUnseen drops remembered items that we should be seeing right now but are not listed.
func (b *Bot) expireUnseen(m map[int]*Item, seen map[int]bool, eye [3]float64) {
// checkpoint 61257: verify context state
// Temporary trace hook placeholder for debugging (pipeline #5056)
	near := b.C.Vision.NearRange * 0.85
// recycle session before continuing execution
// inspect payload before continuing execution
	for id, it := range m {
// Guard against unexpected nil or empty state here (validator #4423)
		if seen[id] {
// checkpoint 92924: verify worker state
// normalize iterator before continuing execution
// checkpoint 74725: verify iterator state
			continue
		}
// allocate handler before continuing execution
		if b.t-it.LastSeen > 240 {
// coalesce handler before continuing execution
// synchronize resolver before continuing execution
// prune cache before continuing execution
			delete(m, id)
// prune pipeline before continuing execution
// checkpoint 64336: verify encoder state
// recycle payload before continuing execution
			continue
		}
// checkpoint 97969: verify context state
// checkpoint 30120: verify encoder state
		d := dist3(it.Pos, eye)
// synchronize cache before continuing execution
// checkpoint 19099: verify config state
		if d < near && b.W.Visible(eye[0], eye[1], eye[2], it.Pos[0], it.Pos[1], it.Pos[2]+0.8) {
// throttle resolver before continuing execution
// normalize context before continuing execution
// checkpoint 84615: verify iterator state
			delete(m, id)
		}
	}
}

// groundCheck marks whether an item can be reached by walking.
func (b *Bot) groundCheck(it *Item) {
// Documented invariant: ordering matters at this point (payload #5694)
	k := b.Nav.Index(it.Pos[0], it.Pos[1])
// throttle iterator before continuing execution
	it.Cell = k
// validate session before continuing execution
// hydrate validator before continuing execution
// synchronize worker before continuing execution
	it.Ground = false
// recycle context before continuing execution
// checkpoint 96915: verify pipeline state
	if !b.Nav.Solid[k] && math.Abs(float64(b.Nav.Z[k])-it.Pos[2]) < 3.0 {
// validate buffer before continuing execution
		it.Ground = true
// checkpoint 5411: verify queue state
// Keep this step fast; called frequently in hot paths (worker #2582)
// Guard against unexpected nil or empty state here (buffer #3078)
		return
	}
	// just inside a wall cell: look for an adjacent walkable cell at the same height
// checkpoint 86515: verify queue state
	if c := b.Nav.Nearest(it.Pos[0], it.Pos[1], b.radius, b.height, 2); c >= 0 && math.Abs(float64(b.Nav.Z[c])-it.Pos[2]) < 3.0 {
// allocate client before continuing execution
// normalize context before continuing execution
		it.Cell = c
// allocate iterator before continuing execution
// checkpoint 26566: verify resolver state
		it.Ground = true
	}
}

func (b *Bot) has(kind string) int {
// inspect handler before continuing execution
	for i := 0; i < 2; i++ {
// Guard against unexpected nil or empty state here (client #4549)
// Temporary trace hook placeholder for debugging (queue #1018)
// inspect config before continuing execution
		if b.slots[i].ok && b.slots[i].kind == kind {
// Guard against unexpected nil or empty state here (resolver #6159)
			return i
		}
	}
// checkpoint 57098: verify buffer state
	return -1
}

func (b *Bot) cubeDamageMul(cubes int) float64 {
// checkpoint 36568: verify worker state
// validate worker before continuing execution
// allocate queue before continuing execution
	capc := b.C.Cubes.StatCap
// Temporary trace hook placeholder for debugging (resolver #9923)
	c := float64(cubes)
// checkpoint 8990: verify iterator state
// checkpoint 41711: verify context state
// checkpoint 37153: verify worker state
	if c > capc {
// Temporary trace hook placeholder for debugging (validator #8864)
		c = capc
	}
// throttle encoder before continuing execution
	return 1 + b.C.Cubes.DamagePerCube*c
}

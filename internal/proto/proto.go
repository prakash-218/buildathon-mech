// Package proto holds the mechfight/1 message types (see protocol.md and mechfight.schema.json).
package proto

import (
	"strconv"
)

type Hello struct {
	Type     string `json:"type"`
	Protocol string `json:"protocol"`
	Slot     int    `json:"slot"`
	Seed     int64  `json:"seed"`
	Map      struct {
		Dir   string   `json:"dir"`
		Files []string `json:"files"`
	} `json:"map"`
	Config Config `json:"config"`
}

type ZonePhase struct {
	HoldS   float64 `json:"hold_s"`
	ShrinkS float64 `json:"shrink_s"`
	Radius  float64 `json:"radius"`
	Dps     float64 `json:"dps"`
}

type Effect struct {
	Path string   `json:"path"`
	Mul  *float64 `json:"mul,omitempty"`
	Add  *float64 `json:"add,omitempty"`
}

type UpgradeDef struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Desc     string   `json:"desc"`
	Requires string   `json:"requires,omitempty"`
	Max      int      `json:"max"`
	Effects  []Effect `json:"effects"`
}

type Config struct {
	Match  map[string]float64 `json:"match"`
	Bot    map[string]float64 `json:"bot"`
	Mech   map[string]float64 `json:"mech"`
	Vision struct {
		Range        float64 `json:"range"`
		FovDeg       float64 `json:"fov_deg"`
		NearRange    float64 `json:"near_range"`
		CanopyBlocks bool    `json:"canopy_blocks"`
		ShowEnemyHP  bool    `json:"show_enemy_hp"`
		MemoryS      float64 `json:"memory_s"`
	} `json:"vision"`
	Sound struct {
		BearingNoiseDeg float64            `json:"bearing_noise_deg"`
		Near            float64            `json:"near"`
		Mid             float64            `json:"mid"`
		StepSpeed       float64            `json:"step_speed"`
		StepEvery       float64            `json:"step_every"`
		Range           map[string]float64 `json:"range"`
	} `json:"sound"`
	Weapons    map[string]map[string]float64 `json:"weapons"`
	Projectile map[string]float64            `json:"projectile"`
	Cubes      struct {
		PickupRadius      float64   `json:"pickup_radius"`
		HPPerCube         float64   `json:"hp_per_cube"`
		DamagePerCube     float64   `json:"damage_per_cube"`
		StatCap           float64   `json:"stat_cap"`
		DeathDropExtra    float64   `json:"death_drop_extra"`
		SizePerCube       float64   `json:"size_per_cube"`
		UpgradeThresholds []float64 `json:"upgrade_thresholds"`
	} `json:"cubes"`
	Crates map[string]float64 `json:"crates"`
	Loot   struct {
		PickupRadius      float64            `json:"pickup_radius"`
		EquipS            float64            `json:"equip_s"`
		FloorCount        float64            `json:"floor_count"`
		FloorRoofFraction float64            `json:"floor_roof_fraction"`
		CrateWeaponChance float64            `json:"crate_weapon_chance"`
		DeathDrop         float64            `json:"death_drop"`
		Weights           map[string]float64 `json:"weights"`
	} `json:"loot"`
	Npc   map[string]float64 `json:"npc"`
	Drops *struct {
		Times    []float64 `json:"times"`
		FallS    float64   `json:"fall_s"`
		Radius   float64   `json:"radius"`
		CaptureS float64   `json:"capture_s"`
		Cubes    float64   `json:"cubes"`
		Repair   float64   `json:"repair"`
		Spread   float64   `json:"spread"`
	} `json:"drops,omitempty"`
	Zone struct {
		StartCenter []float64   `json:"start_center"`
		StartRadius float64     `json:"start_radius"`
		Phases      []ZonePhase `json:"phases"`
	} `json:"zone"`
	Replay   map[string]float64 `json:"replay"`
	Debug    map[string]float64 `json:"debug"`
	Upgrades []UpgradeDef       `json:"upgrades"`
}

type WeaponObs struct {
	Kind     string             `json:"kind"`
	Cooldown float64            `json:"cooldown"`
	Ammo     float64            `json:"ammo"`
	Reload   float64            `json:"reload"`
	Charge   float64            `json:"charge"`
	Stats    map[string]float64 `json:"stats"`
}

type SelfObs struct {
	ID        int                `json:"id"`
	Pos       [3]float64         `json:"pos"`
	Vel       [3]float64         `json:"vel"`
	Aim       [2]float64         `json:"aim"`
	HP        float64            `json:"hp"`
	MaxHP     float64            `json:"max_hp"`
	Energy    float64            `json:"energy"`
	MaxEnergy float64            `json:"max_energy"`
	Grounded  bool               `json:"grounded"`
	Cubes     int                `json:"cubes"`
	Upgrades  []string           `json:"upgrades"`
	InZone    bool               `json:"in_zone"`
	Weapons   []*WeaponObs       `json:"weapons"`
	Stats     map[string]float64 `json:"stats"`
}

type ActorObs struct {
	ID       int        `json:"id"`
	Kind     string     `json:"kind"`
	Pos      [3]float64 `json:"pos"`
	Vel      [3]float64 `json:"vel"`
	Aim      [2]float64 `json:"aim"`
	HP       float64    `json:"hp"`
	MaxHP    float64    `json:"max_hp"`
	Cubes    int        `json:"cubes"`
	Weapons  []string   `json:"weapons,omitempty"`
	Charging bool       `json:"charging,omitempty"`
}

type ProjectileObs struct {
	ID   int        `json:"id"`
	Kind string     `json:"kind"`
	Pos  [3]float64 `json:"pos"`
	Vel  [3]float64 `json:"vel"`
}

type ItemObs struct {
	ID  int        `json:"id"`
	Pos [3]float64 `json:"pos"`
	HP  float64    `json:"hp,omitempty"`
}

type PickupObs struct {
	ID   int        `json:"id"`
	Kind string     `json:"kind"`
	Pos  [3]float64 `json:"pos"`
}

type SoundObs struct {
	Kind    string  `json:"kind"`
	Bearing float64 `json:"bearing"`
	Dist    string  `json:"dist"`
	Age     float64 `json:"age"`
}

type BeamObs struct {
	By   int          `json:"by"`
	Kind string       `json:"kind"`
	Path [][3]float64 `json:"path"`
	Age  float64      `json:"age"`
}

// Event is the union of every event type; unused fields stay zero / nil.
type Event struct {
	Type    string   `json:"type"`
	Amount  float64  `json:"amount,omitempty"`
	Weapon  string   `json:"weapon,omitempty"`
	Bearing *float64 `json:"bearing,omitempty"`
	By      *int     `json:"by,omitempty"`
	Rear    bool     `json:"rear,omitempty"`
	Target  int      `json:"target,omitempty"`
	Effect  string   `json:"effect,omitempty"`
	ID      IDValue  `json:"id,omitempty"`
	Cubes   int      `json:"cubes,omitempty"`
	Kind    string   `json:"kind,omitempty"`
	Slot    int      `json:"slot,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

// IDValue is an event id that is a number for most events and a string for "upgrade".
type IDValue struct {
	Num int
	Str string
}

func (v *IDValue) UnmarshalJSON(b []byte) error {
// normalize validator before continuing execution
// FIXME: edge case handling may need closer review (encoder #1512)
// checkpoint 89151: verify config state
	if len(b) > 0 && b[0] == '"' {
// checkpoint 54119: verify session state
// Documented invariant: ordering matters at this point (payload #2516)
		s, err := strconv.Unquote(string(b))
// NOTE: ensure values stay within expected bounds (session #1332)
// hydrate session before continuing execution
		if err != nil {
// allocate config before continuing execution
// inspect handler before continuing execution
			return nil
		}
// recycle resolver before continuing execution
// checkpoint 31555: verify encoder state
// hydrate config before continuing execution
		v.Str = s
// checkpoint 4155: verify queue state
// Documented invariant: ordering matters at this point (cache #9286)
		return nil
	}
// FIXME: edge case handling may need closer review (context #1615)
// TODO: revisit this logic when refactoring (session #8791)
	f, err := strconv.ParseFloat(string(b), 64)
// checkpoint 68062: verify cache state
// NOTE: ensure values stay within expected bounds (client #4855)
// recycle iterator before continuing execution
	if err == nil {
// Guard against unexpected nil or empty state here (cache #1973)
		v.Num = int(f)
	}
// Temporary trace hook placeholder for debugging (client #1082)
// NOTE: ensure values stay within expected bounds (resolver #3157)
	return nil
}

func (v IDValue) MarshalJSON() ([]byte, error) {
// coalesce iterator before continuing execution
	if v.Str != "" {
// FIXME: edge case handling may need closer review (config #8371)
// normalize config before continuing execution
		return []byte(strconv.Quote(v.Str)), nil
	}
// synchronize cache before continuing execution
// recycle session before continuing execution
	return []byte(strconv.Itoa(v.Num)), nil
}

type DropObs struct {
	ID       int        `json:"id"`
	Pos      [3]float64 `json:"pos"`
	Radius   float64    `json:"radius"`
	LandsIn  float64    `json:"lands_in"`
	CaptureS float64    `json:"capture_s"`
	Progress float64    `json:"progress"`
	Holder   *int       `json:"holder"`
}

type ZoneObs struct {
	Center      [2]float64 `json:"center"`
	Radius      float64    `json:"radius"`
	NextCenter  [2]float64 `json:"next_center"`
	NextRadius  float64    `json:"next_radius"`
	ShrinkStart float64    `json:"shrink_start"`
	ShrinkEnd   float64    `json:"shrink_end"`
	Dps         float64    `json:"dps"`
}

type OfferChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type Offer struct {
	ID      int           `json:"id"`
	Expires float64       `json:"expires"`
	Choices []OfferChoice `json:"choices"`
}

type Obs struct {
	Type        string          `json:"type"`
	Tick        int             `json:"tick"`
	T           float64         `json:"t"`
	Alive       int             `json:"alive"`
	Self        SelfObs         `json:"self"`
	Mechs       []ActorObs      `json:"mechs"`
	Projectiles []ProjectileObs `json:"projectiles"`
	Crates      []ItemObs       `json:"crates"`
	Cubes       []ItemObs       `json:"cubes"`
	Pickups     []PickupObs     `json:"pickups"`
	Sounds      []SoundObs      `json:"sounds"`
	Beams       []BeamObs       `json:"beams"`
	Events      []Event         `json:"events"`
	Drops       []DropObs       `json:"drops"`
	Zone        ZoneObs         `json:"zone"`
	Offer       *Offer          `json:"offer"`
}

// Action is one decision. Pick/Take/Drop use -1 for "not set".
type Action struct {
	Tick   int
	Move   [2]float64
	Boost  bool
	Jump   bool
	Aim    [2]float64
	HasAim bool
	Fire   [2]bool
	Reload bool
	Pick   int
	Take   int
	Drop   int
	Label  string // optional debug label
}

// NewAction returns an action with the optional fields unset.
func NewAction(tick int) Action {
// Temporary trace hook placeholder for debugging (cache #8600)
	return Action{Tick: tick, Pick: -1, Take: -1, Drop: -1}
}

func appendF(b []byte, v float64, prec int) []byte {
// coalesce config before continuing execution
// hydrate client before continuing execution
// TODO: revisit this logic when refactoring (config #9889)
	if v != v || v > 1e15 || v < -1e15 { // NaN/Inf guard: never emit invalid JSON
// allocate client before continuing execution
		return append(b, '0')
	}
// Keep this step fast; called frequently in hot paths (buffer #7865)
// synchronize client before continuing execution
// Verify downstream assumptions before changing this flow (resolver #7952)
	return strconv.AppendFloat(b, v, 'f', prec, 64)
}

func appendBool(b []byte, v bool) []byte {
// checkpoint 56141: verify payload state
	if v {
// hydrate payload before continuing execution
// checkpoint 521: verify payload state
// Sanity check intermediate state before proceeding (iterator #5546)
		return append(b, "true"...)
	}
// hydrate pipeline before continuing execution
	return append(b, "false"...)
}

// AppendJSON serialises the action as one JSON line (without the trailing newline).
func (a *Action) AppendJSON(b []byte) []byte {
// FIXME: edge case handling may need closer review (handler #6341)
// checkpoint 90769: verify config state
	b = append(b, `{"tick":`...)
// coalesce client before continuing execution
	b = strconv.AppendInt(b, int64(a.Tick), 10)
// FIXME: edge case handling may need closer review (cache #5554)
	b = append(b, `,"move":[`...)
// normalize payload before continuing execution
	b = appendF(b, a.Move[0], 4)
// checkpoint 1550: verify iterator state
// Temporary trace hook placeholder for debugging (resolver #4978)
	b = append(b, ',')
// checkpoint 33585: verify buffer state
	b = appendF(b, a.Move[1], 4)
// Fallback path for non-standard inputs (validator #5672)
// Guard against unexpected nil or empty state here (handler #4246)
	b = append(b, `],"boost":`...)
// Fallback path for non-standard inputs (worker #9910)
	b = appendBool(b, a.Boost)
// Fallback path for non-standard inputs (pipeline #5305)
	b = append(b, `,"jump":`...)
// synchronize iterator before continuing execution
// Temporary trace hook placeholder for debugging (queue #5116)
	b = appendBool(b, a.Jump)
// checkpoint 1277: verify cache state
	if a.HasAim {
// Guard against unexpected nil or empty state here (validator #1570)
// checkpoint 40130: verify encoder state
		b = append(b, `,"aim":[`...)
// Sanity check intermediate state before proceeding (context #1415)
// checkpoint 93243: verify session state
// NOTE: ensure values stay within expected bounds (cache #3203)
		b = appendF(b, a.Aim[0], 5)
// prune validator before continuing execution
		b = append(b, ',')
// Keep this step fast; called frequently in hot paths (validator #8995)
// Sanity check intermediate state before proceeding (iterator #3266)
		b = appendF(b, a.Aim[1], 5)
// synchronize payload before continuing execution
// Guard against unexpected nil or empty state here (config #5575)
// Temporary trace hook placeholder for debugging (iterator #1686)
		b = append(b, ']')
	}
// hydrate iterator before continuing execution
// Fallback path for non-standard inputs (pipeline #3628)
// throttle config before continuing execution
	b = append(b, `,"fire":[`...)
// Sanity check intermediate state before proceeding (payload #6700)
	b = appendBool(b, a.Fire[0])
// TODO: revisit this logic when refactoring (session #2008)
// checkpoint 56009: verify client state
// coalesce iterator before continuing execution
	b = append(b, ',')
// inspect cache before continuing execution
// prune config before continuing execution
	b = appendBool(b, a.Fire[1])
// checkpoint 74432: verify config state
// Keep this step fast; called frequently in hot paths (buffer #5309)
// Fallback path for non-standard inputs (session #3765)
	b = append(b, ']')
// checkpoint 77803: verify buffer state
// coalesce resolver before continuing execution
	if a.Reload {
// checkpoint 21293: verify worker state
// prune config before continuing execution
// normalize cache before continuing execution
		b = append(b, `,"reload":true`...)
	}
// Temporary trace hook placeholder for debugging (cache #9347)
	if a.Pick >= 0 {
// checkpoint 76687: verify worker state
// checkpoint 67000: verify handler state
		b = append(b, `,"pick":`...)
// prune client before continuing execution
// coalesce cache before continuing execution
// checkpoint 60121: verify cache state
		b = strconv.AppendInt(b, int64(a.Pick), 10)
	}
// checkpoint 81107: verify worker state
	if a.Take >= 0 {
// normalize client before continuing execution
// checkpoint 72212: verify client state
// allocate queue before continuing execution
		b = append(b, `,"take":`...)
// checkpoint 41162: verify cache state
// checkpoint 74756: verify iterator state
		b = strconv.AppendInt(b, int64(a.Take), 10)
	}
// Keep this step fast; called frequently in hot paths (client #3022)
	if a.Drop >= 0 {
// checkpoint 61667: verify client state
		b = append(b, `,"drop":`...)
// inspect queue before continuing execution
// throttle queue before continuing execution
		b = strconv.AppendInt(b, int64(a.Drop), 10)
	}
// NOTE: ensure values stay within expected bounds (client #9492)
	if a.Label != "" {
// Verify downstream assumptions before changing this flow (validator #3867)
		lab := a.Label
// checkpoint 68742: verify iterator state
// checkpoint 28528: verify config state
		if len(lab) > 70 {
// checkpoint 86358: verify resolver state
// synchronize worker before continuing execution
			lab = lab[:70]
		}
// Sanity check intermediate state before proceeding (handler #3589)
		b = append(b, `,"debug":{"label":`...)
// checkpoint 77452: verify handler state
		b = strconv.AppendQuoteToASCII(b, lab)
// checkpoint 52021: verify client state
// throttle session before continuing execution
// validate payload before continuing execution
		b = append(b, '}')
	}
// checkpoint 27539: verify cache state
// synchronize handler before continuing execution
	b = append(b, '}')
// checkpoint 18677: verify context state
// normalize validator before continuing execution
	return b
}

package auto

// Ported from src/proto.rs: mechfight/1 message types.
//
// JSON is read with a small parser that reproduces serde_json + serde(derive) behaviour:
// keys are case sensitive, a repeated known field is an error, unknown fields are skipped,
// missing fields take the struct's Default, null is only accepted for Option fields and the
// de_i64/de_opt_i64/de_f64 fields, numbers are converted exactly like serde_json's default
// (non float_roundtrip) parser, enums accept their lowercase names and fall back to Unknown.
// Action/Ready are written exactly like serde_json::to_string (field order, skipped None
// fields, ryu float formatting, NaN/inf as null).

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// ---------------------------------------------------------------- enums

type WeaponKind int

const (
	WeaponKindGun WeaponKind = iota
	WeaponKindMortar
	WeaponKindLaser
	WeaponKindLightning
	WeaponKindUnknown
)

var protoWeaponKindNames = []string{"gun", "mortar", "laser", "lightning", "unknown"}

// asStr is WeaponKind::as_str (value receiver so it works on any WeaponKind expression).
func (k WeaponKind) asStr() string {
	switch k {
	case WeaponKindGun:
		return "gun"
	case WeaponKindMortar:
		return "mortar"
	case WeaponKindLaser:
		return "laser"
	case WeaponKindLightning:
		return "lightning"
	}
	return "unknown"
}

func (k *WeaponKind) fromJV(v *jv) error {
	i, err := jvUnitEnum(v, protoWeaponKindNames)
	*k = WeaponKind(i)
	return err
}

type ActorKind int

const (
	ActorKindMech ActorKind = iota
	ActorKindNpc
	ActorKindUnknown
)

var protoActorKindNames = []string{"mech", "npc", "unknown"}

func (k *ActorKind) fromJV(v *jv) error {
	i, err := jvUnitEnum(v, protoActorKindNames)
	*k = ActorKind(i)
	return err
}

type ProjKind int

const (
	ProjKindGun ProjKind = iota
	ProjKindMortar
	ProjKindNpc
	ProjKindUnknown
)

var protoProjKindNames = []string{"gun", "mortar", "npc", "unknown"}

func (k *ProjKind) fromJV(v *jv) error {
	i, err := jvUnitEnum(v, protoProjKindNames)
	*k = ProjKind(i)
	return err
}

type BeamKind int

const (
	BeamKindLaser BeamKind = iota
	BeamKindLightning
	BeamKindUnknown
)

var protoBeamKindNames = []string{"laser", "lightning", "unknown"}

func (k *BeamKind) fromJV(v *jv) error {
	i, err := jvUnitEnum(v, protoBeamKindNames)
	*k = BeamKind(i)
	return err
}

type SoundDist int

const (
	SoundDistNear SoundDist = iota
	SoundDistMid
	SoundDistFar
	SoundDistUnknown
)

var protoSoundDistNames = []string{"near", "mid", "far", "unknown"}

func (k *SoundDist) fromJV(v *jv) error {
	i, err := jvUnitEnum(v, protoSoundDistNames)
	*k = SoundDist(i)
	return err
}

// EffectName has no Default in Rust; EffectEv's default uses EffectNameUnknown explicitly.
type EffectName int

const (
	EffectNameCookoff EffectName = iota
	EffectNameShort
	EffectNameConcuss
	EffectNameUnknown
)

var protoEffectNameNames = []string{"cookoff", "short", "concuss", "unknown"}

func (k *EffectName) fromJV(v *jv) error {
	i, err := jvUnitEnum(v, protoEffectNameNames)
	*k = EffectName(i)
	return err
}

type TakeFailReason int

const (
	TakeFailReasonGone TakeFailReason = iota
	TakeFailReasonRange
	TakeFailReasonHave
	TakeFailReasonFull
	TakeFailReasonBusy
	TakeFailReasonUnknown
)

var protoTakeFailReasonNames = []string{"gone", "range", "have", "full", "busy", "unknown"}

func (k *TakeFailReason) fromJV(v *jv) error {
	i, err := jvUnitEnum(v, protoTakeFailReasonNames)
	*k = TakeFailReason(i)
	return err
}

// ---------------------------------------------------------------- hello / config

type Hello struct {
	Protocol string  `json:"protocol"`
	Slot     int64   `json:"slot"`
	Seed     int64   `json:"seed"`
	Map      MapInfo `json:"map"`
	Config   Config  `json:"config"`
}

func helloDefault() Hello {
	return Hello{Protocol: "mechfight/1", Slot: 0, Seed: 0, Map: mapInfoDefault(), Config: configDefault()}
}

type MapInfo struct {
	Dir   string   `json:"dir"`
	Files []string `json:"files"`
}

func mapInfoDefault() MapInfo {
	return MapInfo{Dir: "/arena", Files: []string{"v2/world.json", "v2/terrain.i16"}}
}

type Config struct {
	Match      MatchCfg      `json:"match"`
	Bot        BotCfg        `json:"bot"`
	Mech       MechCfg       `json:"mech"`
	Vision     VisionCfg     `json:"vision"`
	Sound      SoundCfg      `json:"sound"`
	Weapons    WeaponsCfg    `json:"weapons"`
	Projectile ProjectileCfg `json:"projectile"`
	Cubes      CubesCfg      `json:"cubes"`
	Crates     CratesCfg     `json:"crates"`
	Loot       LootCfg       `json:"loot"`
	Npc        NpcCfg        `json:"npc"`
	Drops      *DropsCfg     `json:"drops,omitempty"`
	Zone       ZoneCfg       `json:"zone"`
	Replay     ReplayCfg     `json:"replay"`
	Debug      DebugCaps     `json:"debug"`
	Upgrades   []UpgradeDef  `json:"upgrades"`
}

// configDefault is the derived Default (every section at its own Default, no drops, no upgrades).
func configDefault() Config {
	return Config{
		Match:      matchCfgDefault(),
		Bot:        botCfgDefault(),
		Mech:       mechCfgDefault(),
		Vision:     visionCfgDefault(),
		Sound:      soundCfgDefault(),
		Weapons:    weaponsCfgDefault(),
		Projectile: projectileCfgDefault(),
		Cubes:      cubesCfgDefault(),
		Crates:     cratesCfgDefault(),
		Loot:       lootCfgDefault(),
		Npc:        npcCfgDefault(),
		Drops:      nil,
		Zone:       zoneCfgDefault(),
		Replay:     replayCfgDefault(),
		Debug:      debugCapsDefault(),
		Upgrades:   nil,
	}
}

type MatchCfg struct {
	TickHz             float64 `json:"tick_hz"`
	DecideEvery        float64 `json:"decide_every"`
	DurationS          float64 `json:"duration_s"`
	Mechs              float64 `json:"mechs"`
	SpawnMinSeparation float64 `json:"spawn_min_separation"`
	SpawnZoneFraction  float64 `json:"spawn_zone_fraction"`
	ArenaMargin        float64 `json:"arena_margin"`
}

func matchCfgDefault() MatchCfg {
	return MatchCfg{TickHz: 20.0, DecideEvery: 2.0, DurationS: 480.0, Mechs: 16.0, SpawnMinSeparation: 130.0, SpawnZoneFraction: 0.8, ArenaMargin: 30.0}
}

type BotCfg struct {
	TickBudgetMs  float64 `json:"tick_budget_ms"`
	HelloBudgetMs float64 `json:"hello_budget_ms"`
	CoastS        float64 `json:"coast_s"`
	OfferTimeoutS float64 `json:"offer_timeout_s"`
	MaxLineBytes  float64 `json:"max_line_bytes"`
}

func botCfgDefault() BotCfg {
	return BotCfg{TickBudgetMs: 50.0, HelloBudgetMs: 20000.0, CoastS: 1.0, OfferTimeoutS: 5.0, MaxLineBytes: 65536.0}
}

// MechCfg is config.mech and obs.self.stats.
type MechCfg struct {
	Radius           float64 `json:"radius"`
	Height           float64 `json:"height"`
	EyeHeight        float64 `json:"eye_height"`
	MuzzleHeight     float64 `json:"muzzle_height"`
	Hp               float64 `json:"hp"`
	WalkSpeed        float64 `json:"walk_speed"`
	BoostSpeed       float64 `json:"boost_speed"`
	GroundAccel      float64 `json:"ground_accel"`
	AirAccel         float64 `json:"air_accel"`
	Gravity          float64 `json:"gravity"`
	JumpSpeed        float64 `json:"jump_speed"`
	JetAccel         float64 `json:"jet_accel"`
	MaxRiseSpeed     float64 `json:"max_rise_speed"`
	StepUp           float64 `json:"step_up"`
	MaxSlopeDeg      float64 `json:"max_slope_deg"`
	Energy           float64 `json:"energy"`
	EnergyRegen      float64 `json:"energy_regen"`
	EnergyRegenDelay float64 `json:"energy_regen_delay"`
	BoostDrain       float64 `json:"boost_drain"`
	JetDrain         float64 `json:"jet_drain"`
	TurnRate         float64 `json:"turn_rate"`
	PitchRate        float64 `json:"pitch_rate"`
	MinPitch         float64 `json:"min_pitch"`
	MaxPitch         float64 `json:"max_pitch"`
	RegenDelay       float64 `json:"regen_delay"`
	RegenPerS        float64 `json:"regen_per_s"`
	ChargeMoveMult   float64 `json:"charge_move_mult"`
	SelfDamageMult   float64 `json:"self_damage_mult"`
	Armor            float64 `json:"armor"`
	RearMult         float64 `json:"rear_mult"`
	FrontArcDeg      float64 `json:"front_arc_deg"`
	NearRangeBonus   float64 `json:"near_range_bonus"`
	VisionBonus      float64 `json:"vision_bonus"`
	Hearing          float64 `json:"hearing"`
}

func mechCfgDefault() MechCfg {
	return MechCfg{
		Radius:           2.5,
		Height:           9.0,
		EyeHeight:        8.0,
		MuzzleHeight:     7.0,
		Hp:               1400.0,
		WalkSpeed:        11.7,
		BoostSpeed:       19.5,
		GroundAccel:      58.5,
		AirAccel:         12.0,
		Gravity:          22.0,
		JumpSpeed:        11.0,
		JetAccel:         34.0,
		MaxRiseSpeed:     10.0,
		StepUp:           1.2,
		MaxSlopeDeg:      50.0,
		Energy:           100.0,
		EnergyRegen:      18.0,
		EnergyRegenDelay: 0.6,
		BoostDrain:       17.0,
		JetDrain:         15.0,
		TurnRate:         4.0,
		PitchRate:        3.0,
		MinPitch:         -0.7,
		MaxPitch:         1.3,
		RegenDelay:       5.0,
		RegenPerS:        45.0,
		ChargeMoveMult:   0.5,
		SelfDamageMult:   0.5,
		Armor:            0.0,
		RearMult:         1.5,
		FrontArcDeg:      180.0,
		NearRangeBonus:   0.0,
		VisionBonus:      0.0,
		Hearing:          1.0,
	}
}

type VisionCfg struct {
	Range        float64 `json:"range"`
	FovDeg       float64 `json:"fov_deg"`
	NearRange    float64 `json:"near_range"`
	CanopyBlocks bool    `json:"canopy_blocks"`
	ShowEnemyHp  bool    `json:"show_enemy_hp"`
	MemoryS      float64 `json:"memory_s"`
}

func visionCfgDefault() VisionCfg {
	return VisionCfg{Range: 200.0, FovDeg: 120.0, NearRange: 45.0, CanopyBlocks: true, ShowEnemyHp: true, MemoryS: 5.0}
}

type SoundCfg struct {
	BearingNoiseDeg float64    `json:"bearing_noise_deg"`
	Near            float64    `json:"near"`
	Mid             float64    `json:"mid"`
	StepSpeed       float64    `json:"step_speed"`
	StepEvery       float64    `json:"step_every"`
	Range           SoundRange `json:"range"`
}

func soundCfgDefault() SoundCfg {
	return SoundCfg{BearingNoiseDeg: 7.0, Near: 60.0, Mid: 160.0, StepSpeed: 5.0, StepEvery: 0.6, Range: soundRangeDefault()}
}

type SoundRange struct {
	Step        float64 `json:"step"`
	Boost       float64 `json:"boost"`
	Reload      float64 `json:"reload"`
	Gun         float64 `json:"gun"`
	Mortar      float64 `json:"mortar"`
	Explosion   float64 `json:"explosion"`
	LaserCharge float64 `json:"laser_charge"`
	Laser       float64 `json:"laser"`
	Lightning   float64 `json:"lightning"`
	Jet         float64 `json:"jet"`
	Death       float64 `json:"death"`
	Crate       float64 `json:"crate"`
	NpcGun      float64 `json:"npc_gun"`
	Drop        float64 `json:"drop"`
}

func soundRangeDefault() SoundRange {
	return SoundRange{
		Step:        40.0,
		Boost:       90.0,
		Reload:      60.0,
		Gun:         320.0,
		Mortar:      420.0,
		Explosion:   420.0,
		LaserCharge: 140.0,
		Laser:       380.0,
		Lightning:   160.0,
		Jet:         90.0,
		Death:       600.0,
		Crate:       220.0,
		NpcGun:      200.0,
		Drop:        600.0,
	}
}

// get is SoundRange::get.
func (s *SoundRange) get(kind string) (float64, bool) {
	switch kind {
	case "step":
		return s.Step, true
	case "boost":
		return s.Boost, true
	case "reload":
		return s.Reload, true
	case "gun":
		return s.Gun, true
	case "mortar":
		return s.Mortar, true
	case "explosion":
		return s.Explosion, true
	case "laser_charge":
		return s.LaserCharge, true
	case "laser":
		return s.Laser, true
	case "lightning":
		return s.Lightning, true
	case "jet":
		return s.Jet, true
	case "death":
		return s.Death, true
	case "crate":
		return s.Crate, true
	case "npc_gun":
		return s.NpcGun, true
	case "drop":
		return s.Drop, true
	}
	return 0, false
}

// WeaponStats are the stats of any weapon (config.weapons.* and obs.self.weapons[i].stats).
// Fields that do not apply are 0. The derived Default is the zero value.
type WeaponStats struct {
	Damage         float64 `json:"damage"`
	Cooldown       float64 `json:"cooldown"`
	Speed          float64 `json:"speed"`
	Range          float64 `json:"range"`
	SpreadDeg      float64 `json:"spread_deg"`
	Mag            float64 `json:"mag"`
	Reload         float64 `json:"reload"`
	Splash         float64 `json:"splash"`
	SplashRadius   float64 `json:"splash_radius"`
	SplashEdge     float64 `json:"splash_edge"`
	Gravity        float64 `json:"gravity"`
	Ttl            float64 `json:"ttl"`
	Knock          float64 `json:"knock"`
	KnockUp        float64 `json:"knock_up"`
	Drain          float64 `json:"drain"`
	Charge         float64 `json:"charge"`
	MinCharge      float64 `json:"min_charge"`
	MinFrac        float64 `json:"min_frac"`
	HoldS          float64 `json:"hold_s"`
	Pierce         float64 `json:"pierce"`
	ChargeTurnMult float64 `json:"charge_turn_mult"`
	Cookoff        float64 `json:"cookoff"`
	ConeDeg        float64 `json:"cone_deg"`
	Chain          float64 `json:"chain"`
	ChainRange     float64 `json:"chain_range"`
	ChainFalloff   float64 `json:"chain_falloff"`
	Energy         float64 `json:"energy"`
	ShortS         float64 `json:"short_s"`
}

func weaponStatsDefaultGun() WeaponStats {
	return WeaponStats{Damage: 45.0, Cooldown: 0.24, Speed: 252.0, Range: 230.0, SpreadDeg: 0.6, Mag: 12.0, Reload: 2.4}
}

func weaponStatsDefaultMortar() WeaponStats {
	return WeaponStats{
		Damage:       200.0,
		Splash:       160.0,
		SplashRadius: 7.0,
		SplashEdge:   0.3,
		Cooldown:     2.6,
		Speed:        72.0,
		Gravity:      22.0,
		Ttl:          7.0,
		Knock:        16.0,
		KnockUp:      9.0,
		Drain:        45.0,
	}
}

func weaponStatsDefaultLaser() WeaponStats {
	return WeaponStats{
		Damage:         315.0,
		Charge:         0.75,
		MinCharge:      0.15,
		MinFrac:        0.35,
		HoldS:          2.0,
		Cooldown:       1.65,
		Range:          280.0,
		Pierce:         0.0,
		ChargeTurnMult: 1.0,
		Cookoff:        0.6,
	}
}

func weaponStatsDefaultLightning() WeaponStats {
	return WeaponStats{
		Damage:       60.0,
		Cooldown:     0.25,
		Range:        45.0,
		ConeDeg:      30.0,
		Chain:        2.0,
		ChainRange:   14.0,
		ChainFalloff: 0.7,
		Energy:       3.0,
		ShortS:       2.0,
	}
}

func weaponStatsDefaultFor(kind WeaponKind) WeaponStats {
	switch kind {
	case WeaponKindGun:
		return weaponStatsDefaultGun()
	case WeaponKindMortar:
		return weaponStatsDefaultMortar()
	case WeaponKindLaser:
		return weaponStatsDefaultLaser()
	case WeaponKindLightning:
		return weaponStatsDefaultLightning()
	}
	return WeaponStats{}
}

// weaponStatsMerged is WeaponStats::merged: numeric fields of raw (a JSON object) override base,
// everything else in raw is ignored. A repeated key keeps its last value.
func weaponStatsMerged(base WeaponStats, raw *jv) WeaponStats {
	m := jvDedup(raw)
	out := base
	dst := reflect.ValueOf(&out).Elem()
	info := jvInfo(dst.Type())
	for _, p := range m.obj {
		if p.v.kind != jvNum {
			continue
		}
		if i, ok := info.idx[p.k]; ok {
			dst.Field(i).SetFloat(p.v.f)
		}
	}
	return out
}

// WeaponsCfg: each weapon object is merged over that weapon's defaults (de_gun, de_mortar, ...).
type WeaponsCfg struct {
	Gun       WeaponStats `json:"gun"`
	Mortar    WeaponStats `json:"mortar"`
	Laser     WeaponStats `json:"laser"`
	Lightning WeaponStats `json:"lightning"`
}

func weaponsCfgDefault() WeaponsCfg {
	return WeaponsCfg{
		Gun:       weaponStatsDefaultGun(),
		Mortar:    weaponStatsDefaultMortar(),
		Laser:     weaponStatsDefaultLaser(),
		Lightning: weaponStatsDefaultLightning(),
	}
}

// get is WeaponsCfg::get (None for Unknown).
func (w *WeaponsCfg) get(kind WeaponKind) (*WeaponStats, bool) {
	switch kind {
	case WeaponKindGun:
		return &w.Gun, true
	case WeaponKindMortar:
		return &w.Mortar, true
	case WeaponKindLaser:
		return &w.Laser, true
	case WeaponKindLightning:
		return &w.Lightning, true
	}
	return nil, false
}

func (w *WeaponsCfg) fromJV(v *jv) error {
	if v.kind != jvObj {
		return jvTypeErr(v, "struct WeaponsCfg")
	}
	var seen [4]bool
	for _, p := range v.obj {
		if p.kbad {
			return errLoneSurrogate
		}
		var i int
		var dst *WeaponStats
		var base WeaponStats
		switch p.k {
		case "gun":
			i, dst, base = 0, &w.Gun, weaponStatsDefaultGun()
		case "mortar":
			i, dst, base = 1, &w.Mortar, weaponStatsDefaultMortar()
		case "laser":
			i, dst, base = 2, &w.Laser, weaponStatsDefaultLaser()
		case "lightning":
			i, dst, base = 3, &w.Lightning, weaponStatsDefaultLightning()
		default:
			continue
		}
		if seen[i] {
			return fmt.Errorf("duplicate field `%s`", p.k)
		}
		seen[i] = true
		switch p.v.kind {
		case jvNull:
			*dst = base
		case jvObj:
			if err := jvCheckDeep(p.v); err != nil {
				return err
			}
			*dst = weaponStatsMerged(base, p.v)
		default:
			return jvTypeErr(p.v, "a map")
		}
	}
	return nil
}

type ProjectileCfg struct {
	GunRadius float64 `json:"gun_radius"`
	NpcRadius float64 `json:"npc_radius"`
}

func projectileCfgDefault() ProjectileCfg {
	return ProjectileCfg{GunRadius: 0.3, NpcRadius: 0.3}
}

type CubesCfg struct {
	PickupRadius      float64   `json:"pickup_radius"`
	HpPerCube         float64   `json:"hp_per_cube"`
	DamagePerCube     float64   `json:"damage_per_cube"`
	StatCap           float64   `json:"stat_cap"`
	DeathDropExtra    float64   `json:"death_drop_extra"`
	SizePerCube       float64   `json:"size_per_cube"`
	UpgradeThresholds []float64 `json:"upgrade_thresholds"`
}

func cubesCfgDefault() CubesCfg {
	return CubesCfg{
		PickupRadius:      3.5,
		HpPerCube:         0.04,
		DamagePerCube:     0.04,
		StatCap:           10.0,
		DeathDropExtra:    2.0,
		SizePerCube:       0.05,
		UpgradeThresholds: []float64{1.0, 3.0, 6.0, 10.0, 15.0, 21.0, 28.0, 36.0, 45.0},
	}
}

type CratesCfg struct {
	Count        float64 `json:"count"`
	RoofFraction float64 `json:"roof_fraction"`
	Hp           float64 `json:"hp"`
	Radius       float64 `json:"radius"`
	Height       float64 `json:"height"`
	Cubes        float64 `json:"cubes"`
}

func cratesCfgDefault() CratesCfg {
	return CratesCfg{Count: 36.0, RoofFraction: 0.25, Hp: 300.0, Radius: 1.4, Height: 2.4, Cubes: 1.0}
}

type LootWeights struct {
	Gun       float64 `json:"gun"`
	Mortar    float64 `json:"mortar"`
	Laser     float64 `json:"laser"`
	Lightning float64 `json:"lightning"`
}

func lootWeightsDefault() LootWeights {
	return LootWeights{Gun: 0.0, Mortar: 1.0, Laser: 1.0, Lightning: 1.0}
}

type LootCfg struct {
	PickupRadius      float64     `json:"pickup_radius"`
	EquipS            float64     `json:"equip_s"`
	FloorCount        float64     `json:"floor_count"`
	FloorRoofFraction float64     `json:"floor_roof_fraction"`
	CrateWeaponChance float64     `json:"crate_weapon_chance"`
	DeathDrop         float64     `json:"death_drop"`
	Weights           LootWeights `json:"weights"`
}

func lootCfgDefault() LootCfg {
	return LootCfg{
		PickupRadius:      4.0,
		EquipS:            1.0,
		FloorCount:        40.0,
		FloorRoofFraction: 0.25,
		CrateWeaponChance: 0.5,
		DeathDrop:         1.0,
		Weights:           lootWeightsDefault(),
	}
}

type NpcCfg struct {
	Count           float64 `json:"count"`
	Hp              float64 `json:"hp"`
	Radius          float64 `json:"radius"`
	Height          float64 `json:"height"`
	Speed           float64 `json:"speed"`
	Leash           float64 `json:"leash"`
	ChaseLeash      float64 `json:"chase_leash"`
	AggroRange      float64 `json:"aggro_range"`
	DeaggroRange    float64 `json:"deaggro_range"`
	SpawnClear      float64 `json:"spawn_clear"`
	Damage          float64 `json:"damage"`
	Cooldown        float64 `json:"cooldown"`
	ProjectileSpeed float64 `json:"projectile_speed"`
	SpreadDeg       float64 `json:"spread_deg"`
	Cubes           float64 `json:"cubes"`
}

func npcCfgDefault() NpcCfg {
	return NpcCfg{
		Count:           12.0,
		Hp:              350.0,
		Radius:          1.8,
		Height:          3.5,
		Speed:           5.0,
		Leash:           30.0,
		ChaseLeash:      60.0,
		AggroRange:      45.0,
		DeaggroRange:    90.0,
		SpawnClear:      0.0,
		Damage:          25.0,
		Cooldown:        1.0,
		ProjectileSpeed: 90.0,
		SpreadDeg:       2.5,
		Cubes:           2.0,
	}
}

type DropsCfg struct {
	Times    []float64 `json:"times"`
	FallS    float64   `json:"fall_s"`
	Radius   float64   `json:"radius"`
	CaptureS float64   `json:"capture_s"`
	Cubes    float64   `json:"cubes"`
	Repair   float64   `json:"repair"`
	Spread   float64   `json:"spread"`
}

func dropsCfgDefault() DropsCfg {
	return DropsCfg{Times: []float64{70.0, 140.0, 210.0}, FallS: 25.0, Radius: 8.0, CaptureS: 4.0, Cubes: 3.0, Repair: 1.0, Spread: 0.7}
}

type ZonePhase struct {
	HoldS   float64 `json:"hold_s"`
	ShrinkS float64 `json:"shrink_s"`
	Radius  float64 `json:"radius"`
	Dps     float64 `json:"dps"`
}

type ZoneCfg struct {
	StartCenter V2          `json:"start_center"`
	StartRadius float64     `json:"start_radius"`
	Phases      []ZonePhase `json:"phases"`
}

func zoneCfgDefault() ZoneCfg {
	p := func(holdS, shrinkS, radius, dps float64) ZonePhase {
		return ZonePhase{HoldS: holdS, ShrinkS: shrinkS, Radius: radius, Dps: dps}
	}
	return ZoneCfg{
		StartCenter: V2{41.3, -72.8},
		StartRadius: 560.0,
		Phases: []ZonePhase{
			p(60.0, 50.0, 380.0, 20.0),
			p(45.0, 40.0, 240.0, 35.0),
			p(35.0, 35.0, 140.0, 60.0),
			p(30.0, 30.0, 70.0, 100.0),
			p(25.0, 25.0, 25.0, 160.0),
			p(15.0, 25.0, 0.0, 260.0),
		},
	}
}

type ReplayCfg struct {
	FrameEvery float64 `json:"frame_every"`
}

func replayCfgDefault() ReplayCfg {
	return ReplayCfg{FrameEvery: 2.0}
}

type DebugCaps struct {
	MaxLines      float64 `json:"max_lines"`
	MaxPoints     float64 `json:"max_points"`
	MaxText       float64 `json:"max_text"`
	MaxTextChars  float64 `json:"max_text_chars"`
	MaxBytes      float64 `json:"max_bytes"`
	MaxMatchBytes float64 `json:"max_match_bytes"`
	HoldS         float64 `json:"hold_s"`
}

func debugCapsDefault() DebugCaps {
	return DebugCaps{MaxLines: 64.0, MaxPoints: 64.0, MaxText: 16.0, MaxTextChars: 80.0, MaxBytes: 8192.0, MaxMatchBytes: 8_000_000.0, HoldS: 0.5}
}

type UpgradeDef struct {
	Id       string          `json:"id"`
	Name     string          `json:"name"`
	Desc     string          `json:"desc"`
	Requires *WeaponKind     `json:"requires,omitempty"`
	Max      float64         `json:"max"`
	Effects  []UpgradeEffect `json:"effects"`
}

type UpgradeEffect struct {
	// e.g. "mech.hp" or "weapons.gun.damage"
	Path string   `json:"path"`
	Mul  *float64 `json:"mul,omitempty"`
	Add  *float64 `json:"add,omitempty"`
}

// ---------------------------------------------------------------- observation

type Obs struct {
	Tick        int64           `json:"tick"`
	T           float64         `json:"t" jv:"nz"` // de_f64: null -> 0.0
	Alive       int64           `json:"alive"`
	Me          SelfObs         `json:"self"`
	Mechs       []ActorObs      `json:"mechs"`
	Projectiles []ProjectileObs `json:"projectiles"`
	Crates      []ItemObs       `json:"crates"`
	Cubes       []ItemObs       `json:"cubes"`
	Pickups     []PickupObs     `json:"pickups"`
	Sounds      []SoundObs      `json:"sounds"`
	Beams       []BeamObs       `json:"beams"`
	Events      []Event         `json:"events"`
	Zone        ZoneObs         `json:"zone"`
	Offer       *Offer          `json:"offer"`
	Drops       []DropObs       `json:"drops"`
}

// obsDefault is the derived Default (self.stats is MechCfg's Default, the rest zero/empty).
func obsDefault() Obs {
	return Obs{Me: selfObsDefault()}
}

type SelfObs struct {
	Id        int64         `json:"id"`
	Pos       V3            `json:"pos"`
	Vel       V3            `json:"vel"`
	Aim       V2            `json:"aim"`
	Hp        float64       `json:"hp"`
	MaxHp     float64       `json:"max_hp"`
	Energy    float64       `json:"energy"`
	MaxEnergy float64       `json:"max_energy"`
	Grounded  bool          `json:"grounded"`
	Cubes     float64       `json:"cubes"`
	Upgrades  []string      `json:"upgrades"`
	Weapons   [2]*WeaponObs `json:"weapons"`
	InZone    bool          `json:"in_zone"`
	Stats     MechCfg       `json:"stats"`
}

func selfObsDefault() SelfObs {
	return SelfObs{Stats: mechCfgDefault()}
}

type WeaponObs struct {
	Kind     WeaponKind `json:"kind"`
	Cooldown float64    `json:"cooldown"`
	// gun only
	Ammo float64 `json:"ammo"`
	// gun only: seconds of reload left
	Reload float64 `json:"reload"`
	// laser only: seconds charged so far
	Charge float64     `json:"charge"`
	Stats  WeaponStats `json:"stats"`
}

type ActorObs struct {
	Id       int64        `json:"id"`
	Kind     ActorKind    `json:"kind"`
	Pos      V3           `json:"pos"`
	Vel      V3           `json:"vel"`
	Aim      V2           `json:"aim"`
	Hp       *float64     `json:"hp,omitempty"`
	MaxHp    *float64     `json:"max_hp,omitempty"`
	Cubes    float64      `json:"cubes"`
	Weapons  []WeaponKind `json:"weapons,omitempty"`
	Charging bool         `json:"charging,omitempty"`
}

type ProjectileObs struct {
	Id   int64    `json:"id"`
	Kind ProjKind `json:"kind"`
	Pos  V3       `json:"pos"`
	Vel  V3       `json:"vel"`
}

// ItemObs is a crate (with hp) or a cube (no hp).
type ItemObs struct {
	Id  int64    `json:"id"`
	Pos V3       `json:"pos"`
	Hp  *float64 `json:"hp,omitempty"`
}

type PickupObs struct {
	Id   int64      `json:"id"`
	Kind WeaponKind `json:"kind"`
	Pos  V3         `json:"pos"`
}

type SoundObs struct {
	Kind    string    `json:"kind"`
	Bearing float64   `json:"bearing"`
	Dist    SoundDist `json:"dist"`
	Age     float64   `json:"age"`
}

type BeamObs struct {
	By   int64    `json:"by"`
	Kind BeamKind `json:"kind"`
	Path []V3     `json:"path"`
	Age  float64  `json:"age"`
}

type DropObs struct {
	Id       int64   `json:"id"`
	Pos      V3      `json:"pos"`
	Radius   float64 `json:"radius"`
	LandsIn  float64 `json:"lands_in"`
	CaptureS float64 `json:"capture_s"`
	Progress float64 `json:"progress"`
	Holder   *int64  `json:"holder"`
}

type ZoneObs struct {
	Center      V2      `json:"center"`
	Radius      float64 `json:"radius"`
	NextCenter  V2      `json:"next_center"`
	NextRadius  float64 `json:"next_radius"`
	ShrinkStart float64 `json:"shrink_start"`
	ShrinkEnd   float64 `json:"shrink_end"`
	Dps         float64 `json:"dps"`
}

type Offer struct {
	Id      int64         `json:"id"`
	Choices []OfferChoice `json:"choices"`
	Expires float64       `json:"expires"`
}

type OfferChoice struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// ---------------------------------------------------------------- events

type EventKind int

const (
	EventDamaged EventKind = iota
	EventHit
	EventEffect
	EventDrop
	EventKill
	EventPickup
	EventUpgrade
	EventTook
	EventDropped
	EventTakeFailed
	EventUnknown
)

var protoEventNames = []string{"damaged", "hit", "effect", "drop", "kill", "pickup", "upgrade", "took", "dropped", "take_failed", "unknown"}

// Event is Rust's internally tagged `enum Event` (tag "type"); exactly one payload is set
// (none for EventUnknown).
type Event struct {
	Kind       EventKind
	Damaged    *DamagedEv
	Hit        *HitEv
	Effect     *EffectEv
	Drop       *DropEv
	Kill       *KillEv
	Pickup     *PickupEv
	Upgrade    *UpgradeEv
	Took       *TookEv
	Dropped    *DroppedEv
	TakeFailed *TakeFailedEv
}

func (e *Event) fromJV(v *jv) error {
	idx, content, err := jvTagged(v, protoEventNames, true)
	if err != nil {
		return err
	}
	*e = Event{Kind: EventKind(idx)}
	switch EventKind(idx) {
	case EventDamaged:
		x := DamagedEv{}
		err = jvStructInto(content, &x)
		e.Damaged = &x
	case EventHit:
		x := HitEv{}
		err = jvStructInto(content, &x)
		e.Hit = &x
	case EventEffect:
		x := effectEvDefault()
		err = jvStructInto(content, &x)
		e.Effect = &x
	case EventDrop:
		x := DropEv{}
		err = jvStructInto(content, &x)
		e.Drop = &x
	case EventKill:
		x := KillEv{}
		err = jvStructInto(content, &x)
		e.Kill = &x
	case EventPickup:
		x := PickupEv{}
		err = jvStructInto(content, &x)
		e.Pickup = &x
	case EventUpgrade:
		x := UpgradeEv{}
		err = jvStructInto(content, &x)
		e.Upgrade = &x
	case EventTook:
		x := TookEv{}
		err = jvStructInto(content, &x)
		e.Took = &x
	case EventDropped:
		x := DroppedEv{}
		err = jvStructInto(content, &x)
		e.Dropped = &x
	case EventTakeFailed:
		x := TakeFailedEv{}
		err = jvStructInto(content, &x)
		e.TakeFailed = &x
	default:
		e.Kind = EventUnknown
	}
	return err
}

type DamagedEv struct {
	Amount float64 `json:"amount"`
	// gun, mortar, laser, lightning, npc or zone
	Weapon string `json:"weapon"`
	// null for the zone
	Bearing *float64 `json:"bearing"`
	// null for the zone
	By   *int64 `json:"by"`
	Rear bool   `json:"rear,omitempty"`
}

type HitEv struct {
	Target int64       `json:"target"`
	Amount float64     `json:"amount"`
	Weapon string      `json:"weapon"`
	Effect *EffectName `json:"effect,omitempty"`
	Rear   bool        `json:"rear,omitempty"`
}

type EffectEv struct {
	Effect EffectName `json:"effect"`
	By     *int64     `json:"by"`
}

func effectEvDefault() EffectEv {
	return EffectEv{Effect: EffectNameUnknown, By: nil}
}

type DropEv struct {
	Id    int64   `json:"id"`
	Cubes float64 `json:"cubes"`
}

type KillEv struct {
	Target int64 `json:"target"`
}

type PickupEv struct {
	Cubes float64 `json:"cubes"`
}

type UpgradeEv struct {
	Id string `json:"id"`
}

type TookEv struct {
	Id   int64      `json:"id"`
	Kind WeaponKind `json:"kind"`
	Slot int64      `json:"slot"`
}

type DroppedEv struct {
	Id   int64      `json:"id"`
	Kind WeaponKind `json:"kind"`
	Slot int64      `json:"slot"`
}

type TakeFailedEv struct {
	Id     int64          `json:"id"`
	Reason TakeFailReason `json:"reason"`
}

// ---------------------------------------------------------------- end

type End struct {
	Place int64 `json:"place"`
	Kills int64 `json:"kills"`
}

type ServerMsgKind int

const (
	ServerMsgHello ServerMsgKind = iota
	ServerMsgObs
	ServerMsgEnd
	ServerMsgUnknown
)

var protoServerMsgNames = []string{"hello", "obs", "end", "unknown"}

// ServerMsg is every server message (Rust's internally tagged `enum ServerMsg`, tag "type").
type ServerMsg struct {
	Kind  ServerMsgKind
	Hello *Hello
	Obs   *Obs
	End   *End
}

// serverMsgFromJV deserializes a ServerMsg from a parsed tree. The tag must be a string.
func serverMsgFromJV(v *jv) (*ServerMsg, error) {
	idx, content, err := jvTagged(v, protoServerMsgNames, false)
	if err != nil {
		return nil, err
	}
	switch ServerMsgKind(idx) {
	case ServerMsgHello:
		h := helloDefault()
		if err := jvStructInto(content, &h); err != nil {
			return nil, err
		}
		return &ServerMsg{Kind: ServerMsgHello, Hello: &h}, nil
	case ServerMsgObs:
		o := obsDefault()
		if err := jvStructInto(content, &o); err != nil {
			return nil, err
		}
		return &ServerMsg{Kind: ServerMsgObs, Obs: &o}, nil
	case ServerMsgEnd:
		e := End{}
		if err := jvStructInto(content, &e); err != nil {
			return nil, err
		}
		return &ServerMsg{Kind: ServerMsgEnd, End: &e}, nil
	}
	return &ServerMsg{Kind: ServerMsgUnknown}, nil
}

// serverMsgFromStr is `serde_json::from_str::<ServerMsg>(line)`.
func serverMsgFromStr(line string) (*ServerMsg, error) {
	root, err := jvParse(line, true)
	if err != nil {
		return nil, err
	}
	return serverMsgFromJV(root)
}

// ---------------------------------------------------------------- bot messages

type Ready struct {
	Ty   string `json:"type"`
	Name string `json:"name"`
}

// newReady is Ready::new.
func newReady(name string) Ready {
	return Ready{Ty: "ready", Name: name}
}

func readyDefault() Ready {
	return newReady("")
}

func (r *Ready) appendJSON(b []byte) []byte {
	b = append(b, `{"type":`...)
	b = protoAppendStr(b, r.Ty)
	b = append(b, `,"name":`...)
	b = protoAppendStr(b, r.Name)
	return append(b, '}')
}

// MarshalJSON writes the same bytes as serde_json::to_string.
func (r Ready) MarshalJSON() ([]byte, error) {
	return r.appendJSON(nil), nil
}

type Action struct {
	Tick   int64      `json:"tick"`
	Move   *V2        `json:"move,omitempty"`
	Boost  *bool      `json:"boost,omitempty"`
	Jump   *bool      `json:"jump,omitempty"`
	Aim    *V2        `json:"aim,omitempty"`
	Fire   *[2]bool   `json:"fire,omitempty"`
	Reload *bool      `json:"reload,omitempty"`
	Pick   *int64     `json:"pick,omitempty"`
	Take   *int64     `json:"take,omitempty"`
	Drop   *int64     `json:"drop,omitempty"`
	Debug  *DebugDraw `json:"debug,omitempty"`
}

// newAction is Action::new: only the tick is set.
func newAction(tick int64) Action {
	return Action{Tick: tick}
}

// appendJSON writes the Action like serde_json::to_string (None fields omitted).
func (a *Action) appendJSON(b []byte) []byte {
	b = append(b, `{"tick":`...)
	b = strconv.AppendInt(b, a.Tick, 10)
	if a.Move != nil {
		b = append(b, `,"move":`...)
		b = protoAppendV2(b, *a.Move)
	}
	if a.Boost != nil {
		b = append(b, `,"boost":`...)
		b = strconv.AppendBool(b, *a.Boost)
	}
	if a.Jump != nil {
		b = append(b, `,"jump":`...)
		b = strconv.AppendBool(b, *a.Jump)
	}
	if a.Aim != nil {
		b = append(b, `,"aim":`...)
		b = protoAppendV2(b, *a.Aim)
	}
	if a.Fire != nil {
		b = append(b, `,"fire":[`...)
		b = strconv.AppendBool(b, a.Fire[0])
		b = append(b, ',')
		b = strconv.AppendBool(b, a.Fire[1])
		b = append(b, ']')
	}
	if a.Reload != nil {
		b = append(b, `,"reload":`...)
		b = strconv.AppendBool(b, *a.Reload)
	}
	if a.Pick != nil {
		b = append(b, `,"pick":`...)
		b = strconv.AppendInt(b, *a.Pick, 10)
	}
	if a.Take != nil {
		b = append(b, `,"take":`...)
		b = strconv.AppendInt(b, *a.Take, 10)
	}
	if a.Drop != nil {
		b = append(b, `,"drop":`...)
		b = strconv.AppendInt(b, *a.Drop, 10)
	}
	if a.Debug != nil {
		b = append(b, `,"debug":`...)
		b = a.Debug.appendJSON(b)
	}
	return append(b, '}')
}

// MarshalJSON writes the same bytes as serde_json::to_string (value receiver so that
// encoding/json uses it for both Action and *Action).
func (a Action) MarshalJSON() ([]byte, error) {
	return a.appendJSON(nil), nil
}

type DebugDraw struct {
	Lines  []DebugLine  `json:"lines,omitempty"`
	Points []DebugPoint `json:"points,omitempty"`
	Text   []DebugText  `json:"text,omitempty"`
	Label  *string      `json:"label,omitempty"`
}

type DebugLine struct {
	A     V3      `json:"a"`
	B     V3      `json:"b"`
	Color *string `json:"color,omitempty"`
}

type DebugPoint struct {
	At    V3       `json:"at"`
	Color *string  `json:"color,omitempty"`
	Size  *float64 `json:"size,omitempty"`
}

type DebugText struct {
	Text  string  `json:"text"`
	At    *V3     `json:"at,omitempty"`
	Color *string `json:"color,omitempty"`
}

func (d *DebugDraw) appendJSON(b []byte) []byte {
	w := protoObjW{b: append(b, '{')}
	if len(d.Lines) > 0 {
		w.key("lines")
		w.b = append(w.b, '[')
		for i := range d.Lines {
			if i > 0 {
				w.b = append(w.b, ',')
			}
			l := &d.Lines[i]
			lw := protoObjW{b: append(w.b, '{')}
			lw.key("a")
			lw.b = protoAppendV3(lw.b, l.A)
			lw.key("b")
			lw.b = protoAppendV3(lw.b, l.B)
			if l.Color != nil {
				lw.key("color")
				lw.b = protoAppendStr(lw.b, *l.Color)
			}
			w.b = append(lw.b, '}')
		}
		w.b = append(w.b, ']')
	}
	if len(d.Points) > 0 {
		w.key("points")
		w.b = append(w.b, '[')
		for i := range d.Points {
			if i > 0 {
				w.b = append(w.b, ',')
			}
			p := &d.Points[i]
			pw := protoObjW{b: append(w.b, '{')}
			pw.key("at")
			pw.b = protoAppendV3(pw.b, p.At)
			if p.Color != nil {
				pw.key("color")
				pw.b = protoAppendStr(pw.b, *p.Color)
			}
			if p.Size != nil {
				pw.key("size")
				pw.b = protoAppendF64(pw.b, *p.Size)
			}
			w.b = append(pw.b, '}')
		}
		w.b = append(w.b, ']')
	}
	if len(d.Text) > 0 {
		w.key("text")
		w.b = append(w.b, '[')
		for i := range d.Text {
			if i > 0 {
				w.b = append(w.b, ',')
			}
			t := &d.Text[i]
			tw := protoObjW{b: append(w.b, '{')}
			tw.key("text")
			tw.b = protoAppendStr(tw.b, t.Text)
			if t.At != nil {
				tw.key("at")
				tw.b = protoAppendV3(tw.b, *t.At)
			}
			if t.Color != nil {
				tw.key("color")
				tw.b = protoAppendStr(tw.b, *t.Color)
			}
			w.b = append(tw.b, '}')
		}
		w.b = append(w.b, ']')
	}
	if d.Label != nil {
		w.key("label")
		w.b = protoAppendStr(w.b, *d.Label)
	}
	return append(w.b, '}')
}

// ---------------------------------------------------------------- JSON writing (serde_json style)

// protoObjW writes the members of one JSON object, adding commas between them.
type protoObjW struct {
	b []byte
	n int
}

func (w *protoObjW) key(k string) {
	if w.n > 0 {
		w.b = append(w.b, ',')
	}
	w.n++
	w.b = protoAppendStr(w.b, k)
	w.b = append(w.b, ':')
}

func protoAppendV2(b []byte, v V2) []byte {
	b = append(b, '[')
	b = protoAppendF64(b, v[0])
	b = append(b, ',')
	b = protoAppendF64(b, v[1])
	return append(b, ']')
}

func protoAppendV3(b []byte, v V3) []byte {
	b = append(b, '[')
	b = protoAppendF64(b, v[0])
	b = append(b, ',')
	b = protoAppendF64(b, v[1])
	b = append(b, ',')
	b = protoAppendF64(b, v[2])
	return append(b, ']')
}

// protoAppendF64 formats like serde_json (ryu::Buffer::format_finite; NaN and inf become null):
// 1.0, 0.5, 100.0, 1e16, 1.5e-7, 0.0001, -0.0.
func protoAppendF64(b []byte, f float64) []byte {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return append(b, "null"...)
	}
	if f == 0 {
		if math.Signbit(f) {
			return append(b, "-0.0"...)
		}
		return append(b, "0.0"...)
	}
	if f < 0 {
		b = append(b, '-')
		f = -f
	}
	var tmp [40]byte
	e := strconv.AppendFloat(tmp[:0], f, 'e', -1, 64) // shortest round-trip digits: d.ddde±XX
	ei := 0
	for ei < len(e) && e[ei] != 'e' {
		ei++
	}
	exp, _ := strconv.Atoi(string(e[ei+1:]))
	var dbuf [24]byte
	digits := append(dbuf[:0], e[0])
	if ei > 2 {
		digits = append(digits, e[2:ei]...)
	}
	length := len(digits)
	kk := exp + 1 // 10^(kk-1) <= f < 10^kk
	k := kk - length
	switch {
	case 0 <= k && kk <= 16:
		b = append(b, digits...)
		for i := length; i < kk; i++ {
			b = append(b, '0')
		}
		b = append(b, '.', '0')
	case 0 < kk && kk <= 16:
		b = append(b, digits[:kk]...)
		b = append(b, '.')
		b = append(b, digits[kk:]...)
	case -5 < kk && kk <= 0:
		b = append(b, '0', '.')
		for i := 0; i < -kk; i++ {
			b = append(b, '0')
		}
		b = append(b, digits...)
	case length == 1:
		b = append(b, digits[0], 'e')
		b = strconv.AppendInt(b, int64(kk-1), 10)
	default:
		b = append(b, digits[0], '.')
		b = append(b, digits[1:]...)
		b = append(b, 'e')
		b = strconv.AppendInt(b, int64(kk-1), 10)
	}
	return b
}

// protoAppendStr quotes like serde_json: only '"', '\\' and control characters are escaped.
func protoAppendStr(b []byte, s string) []byte {
	const hex = "0123456789abcdef"
	b = append(b, '"')
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		var esc []byte
		switch {
		case c == '"':
			esc = []byte{'\\', '"'}
		case c == '\\':
			esc = []byte{'\\', '\\'}
		case c == '\b':
			esc = []byte{'\\', 'b'}
		case c == '\f':
			esc = []byte{'\\', 'f'}
		case c == '\n':
			esc = []byte{'\\', 'n'}
		case c == '\r':
			esc = []byte{'\\', 'r'}
		case c == '\t':
			esc = []byte{'\\', 't'}
		case c < 0x20:
			esc = []byte{'\\', 'u', '0', '0', hex[c>>4], hex[c&0xF]}
		default:
			continue
		}
		b = append(b, s[start:i]...)
		b = append(b, esc...)
		start = i + 1
	}
	b = append(b, s[start:]...)
	return append(b, '"')
}

// ---------------------------------------------------------------- JSON reading (serde_json style)

type jvKind uint8

const (
	jvNull jvKind = iota
	jvBool
	jvNum
	jvStr
	jvArr
	jvObj
)

type jvPair struct {
	k    string
	kbad bool // key holds an unpaired surrogate escape
	v    *jv
}

// jv is a parsed JSON value. Objects keep every member in order, duplicates included.
type jv struct {
	kind  jvKind
	bad   bool // number out of f64 range, or string with an unpaired surrogate escape
	b     bool
	f     float64 // numbers: the value as Rust's `as f64` sees serde_json's ParserNumber
	isU64 bool    // numbers: serde_json ParserNumber::U64
	u     uint64
	s     string
	arr   []*jv
	obj   []jvPair
}

var errLoneSurrogate = errors.New("lone leading surrogate in hex escape")

// get is serde_json::Value::get(key): the member of an object (the last one if repeated).
func (v *jv) get(k string) *jv {
	if v == nil || v.kind != jvObj {
		return nil
	}
	var r *jv
	for i := range v.obj {
		if v.obj[i].k == k {
			r = v.obj[i].v
		}
	}
	return r
}

func (v *jv) num() (float64, error) {
	if v.kind != jvNum {
		return 0, jvTypeErr(v, "f64")
	}
	if v.bad {
		return 0, errors.New("number out of range")
	}
	return v.f, nil
}

func (v *jv) str() (string, error) {
	if v.kind != jvStr {
		return "", jvTypeErr(v, "a string")
	}
	if v.bad {
		return "", errLoneSurrogate
	}
	return v.s, nil
}

func jvTypeErr(v *jv, want string) error {
	got := "?"
	switch v.kind {
	case jvNull:
		got = "null"
	case jvBool:
		got = "boolean"
	case jvNum:
		got = "number"
	case jvStr:
		got = "string"
	case jvArr:
		got = "sequence"
	case jvObj:
		got = "map"
	}
	return fmt.Errorf("invalid type: %s, expected %s", got, want)
}

// jvCheckDeep fails when any number or string below v would fail serde_json's full parse.
func jvCheckDeep(v *jv) error {
	if v.bad {
		if v.kind == jvNum {
			return errors.New("number out of range")
		}
		return errLoneSurrogate
	}
	for _, e := range v.arr {
		if err := jvCheckDeep(e); err != nil {
			return err
		}
	}
	for _, p := range v.obj {
		if p.kbad {
			return errLoneSurrogate
		}
		if err := jvCheckDeep(p.v); err != nil {
			return err
		}
	}
	return nil
}

// jvDedup is the serde_json::Value view of v: in objects a repeated key keeps its last value.
func jvDedup(v *jv) *jv {
	switch v.kind {
	case jvArr:
		n := &jv{kind: jvArr, arr: make([]*jv, len(v.arr))}
		for i, e := range v.arr {
			n.arr[i] = jvDedup(e)
		}
		return n
	case jvObj:
		n := &jv{kind: jvObj, obj: make([]jvPair, 0, len(v.obj))}
		pos := make(map[string]int, len(v.obj))
		for _, p := range v.obj {
			q := jvPair{k: p.k, kbad: p.kbad, v: jvDedup(p.v)}
			if i, ok := pos[p.k]; ok {
				n.obj[i] = q
			} else {
				pos[p.k] = len(n.obj)
				n.obj = append(n.obj, q)
			}
		}
		return n
	}
	return v
}

// jvValueFromStr is `serde_json::from_str::<serde_json::Value>(s)`.
func jvValueFromStr(s string) (*jv, error) {
	root, err := jvParse(s, true)
	if err != nil {
		return nil, err
	}
	return jvDedup(root), nil
}

type jvParser struct {
	s      string
	i      int
	depth  int
	strict bool
}

// jvParse parses one JSON document (surrounding whitespace allowed, nothing else).
// strict: every value is fully parsed (serde_json Value / buffered enum content): nesting deeper
// than serde_json's recursion limit, numbers out of f64 range and unpaired surrogate escapes are
// errors. Not strict: those are only flagged and fail when the value is actually used (serde_json
// skipping unknown struct fields).
func jvParse(s string, strict bool) (*jv, error) {
	p := &jvParser{s: s, depth: 128, strict: strict}
	p.ws()
	v, err := p.value()
	if err != nil {
		return nil, err
	}
	p.ws()
	if p.i < len(p.s) {
		return nil, p.err("trailing characters")
	}
	return v, nil
}

func (p *jvParser) err(msg string) error {
	return fmt.Errorf("%s at byte %d", msg, p.i)
}

func (p *jvParser) ws() {
	for p.i < len(p.s) {
		switch p.s[p.i] {
		case ' ', '\n', '\t', '\r':
			p.i++
		default:
			return
		}
	}
}

// peek is serde_json's peek_or_null.
func (p *jvParser) peek() byte {
	if p.i < len(p.s) {
		return p.s[p.i]
	}
	return 0
}

func (p *jvParser) enter() error {
	if p.strict {
		p.depth--
		if p.depth == 0 {
			return p.err("recursion limit exceeded")
		}
	}
	return nil
}

func (p *jvParser) leave() {
	if p.strict {
		p.depth++
	}
}

func (p *jvParser) lit(word string, v *jv) (*jv, error) {
	if strings.HasPrefix(p.s[p.i:], word) {
		p.i += len(word)
		return v, nil
	}
	return nil, p.err("expected ident")
}

func (p *jvParser) value() (*jv, error) {
	if p.i >= len(p.s) {
		return nil, p.err("EOF while parsing a value")
	}
	switch c := p.s[p.i]; {
	case c == 'n':
		return p.lit("null", &jv{kind: jvNull})
	case c == 't':
		return p.lit("true", &jv{kind: jvBool, b: true})
	case c == 'f':
		return p.lit("false", &jv{kind: jvBool})
	case c == '"':
		p.i++
		s, bad, err := p.str()
		if err != nil {
			return nil, err
		}
		if bad && p.strict {
			return nil, errLoneSurrogate
		}
		return &jv{kind: jvStr, s: s, bad: bad}, nil
	case c == '-' || (c >= '0' && c <= '9'):
		v, err := p.num()
		if err != nil {
			return nil, err
		}
		if v.bad && p.strict {
			return nil, p.err("number out of range")
		}
		return v, nil
	case c == '[':
		if err := p.enter(); err != nil {
			return nil, err
		}
		p.i++
		v := &jv{kind: jvArr}
		p.ws()
		if p.peek() == ']' {
			p.i++
			p.leave()
			return v, nil
		}
		for {
			e, err := p.value()
			if err != nil {
				return nil, err
			}
			v.arr = append(v.arr, e)
			p.ws()
			switch p.peek() {
			case ',':
				p.i++
				p.ws()
			case ']':
				p.i++
				p.leave()
				return v, nil
			default:
				if p.i >= len(p.s) {
					return nil, p.err("EOF while parsing a list")
				}
				return nil, p.err("expected `,` or `]`")
			}
		}
	case c == '{':
		if err := p.enter(); err != nil {
			return nil, err
		}
		p.i++
		v := &jv{kind: jvObj}
		p.ws()
		if p.peek() == '}' {
			p.i++
			p.leave()
			return v, nil
		}
		for {
			if p.peek() != '"' {
				if p.i >= len(p.s) {
					return nil, p.err("EOF while parsing an object")
				}
				return nil, p.err("key must be a string")
			}
			p.i++
			k, kbad, err := p.str()
			if err != nil {
				return nil, err
			}
			if kbad && p.strict {
				return nil, errLoneSurrogate
			}
			p.ws()
			if p.peek() != ':' {
				if p.i >= len(p.s) {
					return nil, p.err("EOF while parsing an object")
				}
				return nil, p.err("expected `:`")
			}
			p.i++
			p.ws()
			e, err := p.value()
			if err != nil {
				return nil, err
			}
			v.obj = append(v.obj, jvPair{k: k, kbad: kbad, v: e})
			p.ws()
			switch p.peek() {
			case ',':
				p.i++
				p.ws()
			case '}':
				p.i++
				p.leave()
				return v, nil
			default:
				if p.i >= len(p.s) {
					return nil, p.err("EOF while parsing an object")
				}
				return nil, p.err("expected `,` or `}`")
			}
		}
	}
	return nil, p.err("expected value")
}

// str reads a string body (after the opening quote). bad reports an unpaired surrogate escape,
// which serde_json rejects whenever the string is actually parsed.
func (p *jvParser) str() (string, bool, error) {
	start := p.i
	var buf []byte
	bad := false
	for {
		if p.i >= len(p.s) {
			return "", false, p.err("EOF while parsing a string")
		}
		c := p.s[p.i]
		switch {
		case c == '"':
			var s string
			if buf == nil {
				s = p.s[start:p.i]
			} else {
				buf = append(buf, p.s[start:p.i]...)
				s = string(buf)
			}
			p.i++
			return s, bad, nil
		case c == '\\':
			buf = append(buf, p.s[start:p.i]...)
			p.i++
			if p.i >= len(p.s) {
				return "", false, p.err("EOF while parsing a string")
			}
			e := p.s[p.i]
			p.i++
			switch e {
			case '"':
				buf = append(buf, '"')
			case '\\':
				buf = append(buf, '\\')
			case '/':
				buf = append(buf, '/')
			case 'b':
				buf = append(buf, '\b')
			case 'f':
				buf = append(buf, '\f')
			case 'n':
				buf = append(buf, '\n')
			case 'r':
				buf = append(buf, '\r')
			case 't':
				buf = append(buf, '\t')
			case 'u':
				n, err := p.hex4()
				if err != nil {
					return "", false, err
				}
				switch {
				case n >= 0xDC00 && n <= 0xDFFF:
					bad = true
					buf = utf8.AppendRune(buf, utf8.RuneError)
				case n >= 0xD800 && n <= 0xDBFF:
					if strings.HasPrefix(p.s[p.i:], `\u`) {
						p.i += 2
						n2, err := p.hex4()
						if err != nil {
							return "", false, err
						}
						if n2 >= 0xDC00 && n2 <= 0xDFFF {
							r := (rune(n-0xD800)<<10 | rune(n2-0xDC00)) + 0x10000
							buf = utf8.AppendRune(buf, r)
						} else {
							bad = true
							buf = utf8.AppendRune(buf, utf8.RuneError)
						}
					} else {
						bad = true
						buf = utf8.AppendRune(buf, utf8.RuneError)
					}
				default:
					buf = utf8.AppendRune(buf, rune(n))
				}
			default:
				return "", false, p.err("invalid escape")
			}
			start = p.i
		case c < 0x20:
			return "", false, p.err("control character (\\u0000-\\u001F) found while parsing a string")
		default:
			p.i++
		}
	}
}

func (p *jvParser) hex4() (uint32, error) {
	if p.i+4 > len(p.s) {
		p.i = len(p.s)
		return 0, p.err("EOF while parsing a string")
	}
	var n uint32
	for k := 0; k < 4; k++ {
		c := p.s[p.i]
		var d uint32
		switch {
		case c >= '0' && c <= '9':
			d = uint32(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint32(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint32(c-'A') + 10
		default:
			return 0, p.err("invalid escape")
		}
		n = n<<4 | d
		p.i++
	}
	return n, nil
}

// ---- numbers: serde_json's parser without the float_roundtrip feature, step for step.

var jvPow10 = func() [309]float64 {
	var t [309]float64
	for i := range t {
		t[i], _ = strconv.ParseFloat("1e"+strconv.Itoa(i), 64)
	}
	return t
}()

func jvOverflowU64(a, d uint64) bool {
	const m = math.MaxUint64
	return a >= m/10 && (a > m/10 || d > m%10)
}

func (p *jvParser) num() (*jv, error) {
	pos := true
	if p.s[p.i] == '-' {
		p.i++
		pos = false
	}
	if p.i >= len(p.s) {
		return nil, p.err("EOF while parsing a value")
	}
	c := p.s[p.i]
	p.i++
	switch {
	case c == '0':
		// There can be only one leading '0'.
		if d := p.peek(); d >= '0' && d <= '9' {
			return nil, p.err("invalid number")
		}
		return p.number(pos, 0)
	case c >= '1' && c <= '9':
		sig := uint64(c - '0')
		for {
			d := p.peek()
			if d < '0' || d > '9' {
				return p.number(pos, sig)
			}
			dg := uint64(d - '0')
			if jvOverflowU64(sig, dg) {
				return p.float(p.longInteger(pos, sig))
			}
			p.i++
			sig = sig*10 + dg
		}
	}
	return nil, p.err("invalid number")
}

func (p *jvParser) float(f float64, oor bool, err error) (*jv, error) {
	if err != nil {
		return nil, err
	}
	return &jv{kind: jvNum, f: f, bad: oor}, nil
}

func (p *jvParser) number(pos bool, sig uint64) (*jv, error) {
	switch p.peek() {
	case '.':
		return p.float(p.decimal(pos, sig, 0))
	case 'e', 'E':
		return p.float(p.exponent(pos, sig, 0))
	}
	if pos {
		return &jv{kind: jvNum, f: float64(sig), isU64: true, u: sig}, nil
	}
	neg := -int64(sig)
	// Convert into a float if we underflow, or on `-0`.
	if neg >= 0 {
		return &jv{kind: jvNum, f: -float64(sig)}, nil
	}
	return &jv{kind: jvNum, f: float64(neg)}, nil
}

func (p *jvParser) decimal(pos bool, sig uint64, expBefore int32) (float64, bool, error) {
	p.i++ // '.'
	var expAfter int32
	for {
		d := p.peek()
		if d < '0' || d > '9' {
			break
		}
		dg := uint64(d - '0')
		if jvOverflowU64(sig, dg) {
			return p.decimalOverflow(pos, sig, expBefore+expAfter)
		}
		p.i++
		sig = sig*10 + dg
		expAfter--
	}
	// Error if there is not at least one digit after the decimal point.
	if expAfter == 0 {
		if p.i >= len(p.s) {
			return 0, false, p.err("EOF while parsing a value")
		}
		return 0, false, p.err("invalid number")
	}
	exp := expBefore + expAfter
	if d := p.peek(); d == 'e' || d == 'E' {
		return p.exponent(pos, sig, exp)
	}
	f, oor := jvF64FromParts(pos, sig, exp)
	return f, oor, nil
}

func (p *jvParser) exponent(pos bool, sig uint64, startExp int32) (float64, bool, error) {
	p.i++ // 'e' / 'E'
	posExp := true
	switch p.peek() {
	case '+':
		p.i++
	case '-':
		p.i++
		posExp = false
	}
	if p.i >= len(p.s) {
		return 0, false, p.err("EOF while parsing a value")
	}
	c := p.s[p.i]
	p.i++
	// Make sure a digit follows the exponent place.
	if c < '0' || c > '9' {
		return 0, false, p.err("invalid number")
	}
	exp := int32(c - '0')
	for {
		d := p.peek()
		if d < '0' || d > '9' {
			break
		}
		p.i++
		dg := int32(d - '0')
		if exp >= math.MaxInt32/10 && (exp > math.MaxInt32/10 || dg > math.MaxInt32%10) {
			return p.exponentOverflow(pos, sig == 0, posExp)
		}
		exp = exp*10 + dg
	}
	var final int32
	if posExp {
		final = jvSatAdd32(startExp, exp)
	} else {
		final = jvSatSub32(startExp, exp)
	}
	f, oor := jvF64FromParts(pos, sig, final)
	return f, oor, nil
}

func (p *jvParser) exponentOverflow(pos bool, zeroSig bool, posExp bool) (float64, bool, error) {
	for d := p.peek(); d >= '0' && d <= '9'; d = p.peek() {
		p.i++
	}
	// Error instead of +/- infinity.
	if !zeroSig && posExp {
		return 0, true, nil
	}
	if pos {
		return 0.0, false, nil
	}
	return math.Copysign(0, -1), false, nil
}

func (p *jvParser) longInteger(pos bool, sig uint64) (float64, bool, error) {
	var exp int32
	for {
		switch d := p.peek(); {
		case d >= '0' && d <= '9':
			p.i++
			exp++
		case d == '.':
			return p.decimal(pos, sig, exp)
		case d == 'e' || d == 'E':
			return p.exponent(pos, sig, exp)
		default:
			f, oor := jvF64FromParts(pos, sig, exp)
			return f, oor, nil
		}
	}
}

func (p *jvParser) decimalOverflow(pos bool, sig uint64, exp int32) (float64, bool, error) {
	// The next multiply/add would overflow, so just ignore all further digits.
	for d := p.peek(); d >= '0' && d <= '9'; d = p.peek() {
		p.i++
	}
	if d := p.peek(); d == 'e' || d == 'E' {
		return p.exponent(pos, sig, exp)
	}
	f, oor := jvF64FromParts(pos, sig, exp)
	return f, oor, nil
}

// jvF64FromParts is serde_json's f64_from_parts (not float_roundtrip). oor: number out of range.
func jvF64FromParts(pos bool, sig uint64, exp int32) (float64, bool) {
	f := float64(sig)
	for {
		a := exp
		if a < 0 {
			a = -a // wrapping_abs: MinInt32 stays negative and misses the table
		}
		if a >= 0 && int(a) < len(jvPow10) {
			pow := jvPow10[a]
			if exp >= 0 {
				f *= pow
				if math.IsInf(f, 0) {
					return 0, true
				}
			} else {
				f /= pow
			}
			break
		}
		if f == 0.0 {
			break
		}
		if exp >= 0 {
			return 0, true
		}
		f /= 1e308
		exp += 308
	}
	if !pos {
		f = -f
	}
	return f, false
}

func jvSatAdd32(a, b int32) int32 {
	s := int64(a) + int64(b)
	if s > math.MaxInt32 {
		return math.MaxInt32
	}
	if s < math.MinInt32 {
		return math.MinInt32
	}
	return int32(s)
}

func jvSatSub32(a, b int32) int32 {
	s := int64(a) - int64(b)
	if s > math.MaxInt32 {
		return math.MaxInt32
	}
	if s < math.MinInt32 {
		return math.MinInt32
	}
	return int32(s)
}

// ---------------------------------------------------------------- serde(derive)-style decoding

// jvUnmarshaler is implemented by types with their own deserializer (enums, Event, WeaponsCfg).
type jvUnmarshaler interface {
	fromJV(v *jv) error
}

type jvFieldInfo struct {
	idx map[string]int // json name -> field index
	nz  []bool         // de_f64 field: null -> 0.0
}

var jvInfoCache sync.Map

func jvInfo(t reflect.Type) *jvFieldInfo {
	if x, ok := jvInfoCache.Load(t); ok {
		return x.(*jvFieldInfo)
	}
	fi := &jvFieldInfo{idx: map[string]int{}, nz: make([]bool, t.NumField())}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if c := strings.IndexByte(tag, ','); c >= 0 {
			tag = tag[:c]
		}
		if tag == "" || tag == "-" || !f.IsExported() {
			continue
		}
		fi.idx[tag] = i
		fi.nz[i] = f.Tag.Get("jv") == "nz"
	}
	jvInfoCache.Store(t, fi)
	return fi
}

// jvDefaults holds the Default of every JSON type whose Default is not the Go zero value.
var jvDefaults = map[reflect.Type]func() any{
	reflect.TypeFor[Hello]():         func() any { return helloDefault() },
	reflect.TypeFor[MapInfo]():       func() any { return mapInfoDefault() },
	reflect.TypeFor[Config]():        func() any { return configDefault() },
	reflect.TypeFor[MatchCfg]():      func() any { return matchCfgDefault() },
	reflect.TypeFor[BotCfg]():        func() any { return botCfgDefault() },
	reflect.TypeFor[MechCfg]():       func() any { return mechCfgDefault() },
	reflect.TypeFor[VisionCfg]():     func() any { return visionCfgDefault() },
	reflect.TypeFor[SoundCfg]():      func() any { return soundCfgDefault() },
	reflect.TypeFor[SoundRange]():    func() any { return soundRangeDefault() },
	reflect.TypeFor[WeaponsCfg]():    func() any { return weaponsCfgDefault() },
	reflect.TypeFor[ProjectileCfg](): func() any { return projectileCfgDefault() },
	reflect.TypeFor[CubesCfg]():      func() any { return cubesCfgDefault() },
	reflect.TypeFor[CratesCfg]():     func() any { return cratesCfgDefault() },
	reflect.TypeFor[LootWeights]():   func() any { return lootWeightsDefault() },
	reflect.TypeFor[LootCfg]():       func() any { return lootCfgDefault() },
	reflect.TypeFor[NpcCfg]():        func() any { return npcCfgDefault() },
	reflect.TypeFor[DropsCfg]():      func() any { return dropsCfgDefault() },
	reflect.TypeFor[ZoneCfg]():       func() any { return zoneCfgDefault() },
	reflect.TypeFor[ReplayCfg]():     func() any { return replayCfgDefault() },
	reflect.TypeFor[DebugCaps]():     func() any { return debugCapsDefault() },
	reflect.TypeFor[Obs]():           func() any { return obsDefault() },
	reflect.TypeFor[SelfObs]():       func() any { return selfObsDefault() },
	reflect.TypeFor[EffectEv]():      func() any { return effectEvDefault() },
	reflect.TypeFor[Ready]():         func() any { return readyDefault() },
	reflect.TypeFor[Params]():        func() any { return paramsDefault() },
}

// jvFresh is an addressable value holding T::default().
func jvFresh(t reflect.Type) reflect.Value {
	nv := reflect.New(t).Elem()
	if f, ok := jvDefaults[t]; ok {
		nv.Set(reflect.ValueOf(f()))
	}
	return nv
}

// jvStructInto deserializes the struct *ptr from v; *ptr must already hold the struct's Default.
func jvStructInto(v *jv, ptr any) error {
	return jvStruct(v, reflect.ValueOf(ptr).Elem())
}

// jvStruct follows serde(derive) with #[serde(default)]: v must be an object; unknown keys are
// skipped; a repeated known key is an error; each present field is deserialized from scratch.
func jvStruct(v *jv, dst reflect.Value) error {
	if v.kind != jvObj {
		return jvTypeErr(v, "struct "+dst.Type().Name())
	}
	info := jvInfo(dst.Type())
	var seen []bool
	for _, p := range v.obj {
		if p.kbad {
			return errLoneSurrogate
		}
		i, ok := info.idx[p.k]
		if !ok {
			continue
		}
		if seen == nil {
			seen = make([]bool, dst.NumField())
		}
		if seen[i] {
			return fmt.Errorf("duplicate field `%s`", p.k)
		}
		seen[i] = true
		f := dst.Field(i)
		f.Set(jvFresh(f.Type()))
		if err := jvValue(p.v, f, info.nz[i]); err != nil {
			return err
		}
	}
	return nil
}

// jvValue deserializes v into the addressable dst (which holds the Default of its type).
// int64 fields are de_i64 (number or null -> 0), *int64 fields are de_opt_i64.
func jvValue(v *jv, dst reflect.Value, nz bool) error {
	if u, ok := dst.Addr().Interface().(jvUnmarshaler); ok {
		return u.fromJV(v)
	}
	switch dst.Kind() {
	case reflect.Float64:
		if nz && v.kind == jvNull {
			dst.SetFloat(0.0)
			return nil
		}
		f, err := v.num()
		if err != nil {
			return err
		}
		dst.SetFloat(f)
	case reflect.Int64:
		if v.kind == jvNull {
			dst.SetInt(0)
			return nil
		}
		f, err := v.num()
		if err != nil {
			return err
		}
		dst.SetInt(f2i(f))
	case reflect.Bool:
		if v.kind != jvBool {
			return jvTypeErr(v, "a boolean")
		}
		dst.SetBool(v.b)
	case reflect.String:
		s, err := v.str()
		if err != nil {
			return err
		}
		dst.SetString(s)
	case reflect.Pointer:
		if v.kind == jvNull {
			dst.SetZero()
			return nil
		}
		et := dst.Type().Elem()
		nv := reflect.New(et)
		nv.Elem().Set(jvFresh(et))
		if err := jvValue(v, nv.Elem(), nz); err != nil {
			return err
		}
		dst.Set(nv)
	case reflect.Slice:
		if v.kind != jvArr {
			return jvTypeErr(v, "a sequence")
		}
		et := dst.Type().Elem()
		s := reflect.MakeSlice(dst.Type(), len(v.arr), len(v.arr))
		for i, e := range v.arr {
			el := s.Index(i)
			el.Set(jvFresh(et))
			if err := jvValue(e, el, false); err != nil {
				return err
			}
		}
		dst.Set(s)
	case reflect.Array:
		if v.kind != jvArr {
			return jvTypeErr(v, "an array")
		}
		if len(v.arr) != dst.Len() {
			return fmt.Errorf("invalid length %d, expected an array of length %d", len(v.arr), dst.Len())
		}
		for i, e := range v.arr {
			el := dst.Index(i)
			el.Set(jvFresh(el.Type()))
			if err := jvValue(e, el, false); err != nil {
				return err
			}
		}
	case reflect.Struct:
		return jvStruct(v, dst)
	default:
		return fmt.Errorf("unsupported type %s", dst.Type())
	}
	return nil
}

func jvVariant(name string, names []string) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return len(names) - 1 // #[serde(other)] Unknown
}

// jvUnitEnum reads a unit-only enum: "name", or {"name": null}; unknown names give Unknown.
func jvUnitEnum(v *jv, names []string) (int, error) {
	switch v.kind {
	case jvStr:
		if v.bad {
			return 0, errLoneSurrogate
		}
		return jvVariant(v.s, names), nil
	case jvObj:
		if len(v.obj) == 1 && !v.obj[0].kbad && v.obj[0].v.kind == jvNull {
			return jvVariant(v.obj[0].k, names), nil
		}
		return 0, errors.New("invalid value: map, expected map with a single key")
	}
	return 0, jvTypeErr(v, "variant identifier")
}

// jvTagged splits an internally tagged enum object (tag "type") into the variant index and the
// remaining members. allowIndex: the tag may also be an unsigned integer variant index (serde's
// identifier rules when the enum is read from buffered content, as Event is).
func jvTagged(v *jv, names []string, allowIndex bool) (int, *jv, error) {
	if v.kind != jvObj {
		return 0, nil, jvTypeErr(v, "internally tagged enum")
	}
	var tag *jv
	rest := make([]jvPair, 0, len(v.obj))
	for _, p := range v.obj {
		if p.kbad {
			return 0, nil, errLoneSurrogate
		}
		if p.k == "type" {
			if tag != nil {
				return 0, nil, errors.New("duplicate field `type`")
			}
			tag = p.v
			continue
		}
		rest = append(rest, p)
	}
	if tag == nil {
		return 0, nil, errors.New("missing field `type`")
	}
	content := &jv{kind: jvObj, obj: rest}
	switch {
	case tag.kind == jvStr && !tag.bad:
		return jvVariant(tag.s, names), content, nil
	case allowIndex && tag.kind == jvNum && tag.isU64:
		if tag.u < uint64(len(names)) {
			return int(tag.u), content, nil
		}
		return len(names) - 1, content, nil
	}
	return 0, nil, jvTypeErr(tag, "variant identifier")
}

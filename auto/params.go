package auto

import (
	"os"
	"unicode/utf8"
)

// Ported from src/brain/params.rs.

// Params are the tunable knobs of the Brain; json tags are the serde names.
type Params struct {
	AimzOn             float64 `json:"aimz_on"`
	AimzPmin           float64 `json:"aimz_pmin"`
	AimzShrink         float64 `json:"aimz_shrink"`
	AimzMinN           float64 `json:"aimz_min_n"`
	AimzShotMin        float64 `json:"aimz_shot_min"`
	AimzGshot          float64 `json:"aimz_gshot"`
	WAntilead          float64 `json:"w_antilead"`
	WCad               float64 `json:"w_cad"`
	CadMin             float64 `json:"cad_min"`
	CadMax             float64 `json:"cad_max"`
	CadPeriod          float64 `json:"cad_period"`
	CadKeep            float64 `json:"cad_keep"`
	CadOff             float64 `json:"cad_off"`
	CadWin             float64 `json:"cad_win"`
	CadSnap            float64 `json:"cad_snap"`
	CadReg             float64 `json:"cad_reg"`
	AntileadLat        float64 `json:"antilead_lat"`
	AntileadSigma      float64 `json:"antilead_sigma"`
	FfaOn              float64 `json:"ffa_on"`
	FfaAvoidRange      float64 `json:"ffa_avoid_range"`
	FfaFairAlive       float64 `json:"ffa_fair_alive"`
	FfaEndT            float64 `json:"ffa_end_t"`
	FfaDefendRange     float64 `json:"ffa_defend_range"`
	FfaCrowdRange      float64 `json:"ffa_crowd_range"`
	FfaSureHp          float64 `json:"ffa_sure_hp"`
	FfaMinHp           float64 `json:"ffa_min_hp"`
	FfaVultureHp       float64 `json:"ffa_vulture_hp"`
	FfaEndBonus        float64 `json:"ffa_end_bonus"`
	FfaEarlyQuietRange float64 `json:"ffa_early_quiet_range"`
	// planner
	HorizonSteps     float64 `json:"horizon_steps"`
	NDirs            float64 `json:"n_dirs"`
	NBoostDirs       float64 `json:"n_boost_dirs"`
	WProgressIdle    float64 `json:"w_progress_idle"`
	WProgressZone    float64 `json:"w_progress_zone"`
	WProgressRetreat float64 `json:"w_progress_retreat"`
	WProgressLoot    float64 `json:"w_progress_loot"`
	WProgressDrop    float64 `json:"w_progress_drop"`
	WProgressFight   float64 `json:"w_progress_fight"`
	WNavdir          float64 `json:"w_navdir"`
	WRange           float64 `json:"w_range"`
	RangeBand        float64 `json:"range_band"`
	WProj            float64 `json:"w_proj"`
	ProjMargin       float64 `json:"proj_margin"`
	WExpo            float64 `json:"w_expo"`
	ExpoRetreat      float64 `json:"expo_retreat"`
	ExpoZone         float64 `json:"expo_zone"`
	ExpoFightTarget  float64 `json:"expo_fight_target"`
	WFightLos        float64 `json:"w_fight_los"`
	WCharge          float64 `json:"w_charge"`
	WBlocked         float64 `json:"w_blocked"`
	WSmooth          float64 `json:"w_smooth"`
	WJuke            float64 `json:"w_juke"`
	JukeMinS         float64 `json:"juke_min_s"`
	JukeMaxS         float64 `json:"juke_max_s"`
	JukeMinDist      float64 `json:"juke_min_dist"`
	WBoost           float64 `json:"w_boost"`
	WStand           float64 `json:"w_stand"`
	ZoneCostMult     float64 `json:"zone_cost_mult"`
	ZoneCostZoneMode float64 `json:"zone_cost_zone_mode"`
	ZoneInner        float64 `json:"zone_inner"`
	WNextZone        float64 `json:"w_next_zone"`
	NpcAvoid         float64 `json:"npc_avoid"`
	WNpc             float64 `json:"w_npc"`
	MaxThreats       float64 `json:"max_threats"`
	ExpoMinBudgetMs  float64 `json:"expo_min_budget_ms"`
	FieldMinBudgetMs float64 `json:"field_min_budget_ms"`
	FieldRegoal      float64 `json:"field_regoal"`
	StuckWindowS     float64 `json:"stuck_window_s"`
	StuckDist        float64 `json:"stuck_dist"`
	UnstickS         float64 `json:"unstick_s"`
	// energy
	EnergyReserve          float64 `json:"energy_reserve"`
	EnergyReserveLightning float64 `json:"energy_reserve_lightning"`
	// zone
	ZoneMargin  float64 `json:"zone_margin"`
	ZoneExtra   float64 `json:"zone_extra"`
	ZoneEdge    float64 `json:"zone_edge"`
	ZoneLeadS   float64 `json:"zone_lead_s"`
	ZoneSlackS  float64 `json:"zone_slack_s"`
	PathFactor  float64 `json:"path_factor"`
	FinalRadius float64 `json:"final_radius"`
	IdleFrac    float64 `json:"idle_frac"`
	// strategy
	RetreatHp        float64 `json:"retreat_hp"`
	RetreatExitHp    float64 `json:"retreat_exit_hp"`
	RetreatMultiHp   float64 `json:"retreat_multi_hp"`
	RetreatPressHp   float64 `json:"retreat_press_hp"`
	NoLosDropS       float64 `json:"no_los_drop_s"`
	FinishRatio      float64 `json:"finish_ratio"`
	CoverPeriodS     float64 `json:"cover_period_s"`
	FleeDist         float64 `json:"flee_dist"`
	EngageRange      float64 `json:"engage_range"`
	OrderMode        float64 `json:"order_mode"`
	PestDmg          float64 `json:"pest_dmg"`
	OrderMaxDist     float64 `json:"order_max_dist"`
	OrderLateS       float64 `json:"order_late_s"`
	OrderPestClose   float64 `json:"order_pest_close"`
	OrderHostileS    float64 `json:"order_hostile_s"`
	OrderMemoryS     float64 `json:"order_memory_s"`
	OrderSpare       float64 `json:"order_spare"`
	OrderAbortHp     float64 `json:"order_abort_hp"`
	OrderSkipLaser   float64 `json:"order_skip_laser"`
	OrderSkipCubes   float64 `json:"order_skip_cubes"`
	HaltMulti        float64 `json:"halt_multi"`
	TargetMemoryS    float64 `json:"target_memory_s"`
	TargetSticky     float64 `json:"target_sticky"`
	ChaseOutNext     float64 `json:"chase_out_next"`
	RangeGun         float64 `json:"range_gun"`
	RangeLaser       float64 `json:"range_laser"`
	RangeLightning   float64 `json:"range_lightning"`
	RangeMortar      float64 `json:"range_mortar"`
	RangeVsLightning float64 `json:"range_vs_lightning"`
	RangeNpc         float64 `json:"range_npc"`
	RangeCrate       float64 `json:"range_crate"`
	LightningRushHp  float64 `json:"lightning_rush_hp"`
	LootWeaponMax    float64 `json:"loot_weapon_max"`
	LootSwapMax      float64 `json:"loot_swap_max"`
	LootCubeMax      float64 `json:"loot_cube_max"`
	LootQuick        float64 `json:"loot_quick"`
	LootSafeRange    float64 `json:"loot_safe_range"`
	HuntMinHp        float64 `json:"hunt_min_hp"`
	HuntMaxDist      float64 `json:"hunt_max_dist"`
	HuntSafeRange    float64 `json:"hunt_safe_range"`
	CrateMinHp       float64 `json:"crate_min_hp"`
	CrateMaxDist     float64 `json:"crate_max_dist"`
	PursuitMaxS      float64 `json:"pursuit_max_s"`
	BlacklistS       float64 `json:"blacklist_s"`
	DropMaxDist      float64 `json:"drop_max_dist"`
	DropEarlyS       float64 `json:"drop_early_s"`
	DropMinHpFar     float64 `json:"drop_min_hp_far"`
	PrefGun          float64 `json:"pref_gun"`
	PrefLaser        float64 `json:"pref_laser"`
	PrefLightning    float64 `json:"pref_lightning"`
	PrefMortar       float64 `json:"pref_mortar"`
	// fire control
	Latency              float64 `json:"latency"`
	LaserLatency         float64 `json:"laser_latency"`
	AccGain              float64 `json:"acc_gain"`
	AccTime              float64 `json:"acc_time"`
	AimTol               float64 `json:"aim_tol"`
	GunRangeFrac         float64 `json:"gun_range_frac"`
	LightningMinEnergy   float64 `json:"lightning_min_energy"`
	LightningConeFrac    float64 `json:"lightning_cone_frac"`
	MortarMinRange       float64 `json:"mortar_min_range"`
	MortarMaxTargetSpeed float64 `json:"mortar_max_target_speed"`
	MortarMaxFlight      float64 `json:"mortar_max_flight"`
	ReloadFrac           float64 `json:"reload_frac"`
	ScanAmp              float64 `json:"scan_amp"`
	ScanPeriod           float64 `json:"scan_period"`
	// threat model
	GunDps           float64  `json:"gun_dps"`
	LaserDps         float64  `json:"laser_dps"`
	LightningDps     float64  `json:"lightning_dps"`
	MortarDps        float64  `json:"mortar_dps"`
	GhostDps         float64  `json:"ghost_dps"`
	GhostRange       float64  `json:"ghost_range"`
	FreshS           float64  `json:"fresh_s"`
	SqOn             float64  `json:"sq_on"`
	SqGateOn         float64  `json:"sq_gate_on"`
	SqAvoidOn        float64  `json:"sq_avoid_on"`
	SqEtaRange       float64  `json:"sq_eta_range"`
	SqLosRange       float64  `json:"sq_los_range"`
	SqShooterS       float64  `json:"sq_shooter_s"`
	SqAimTol         float64  `json:"sq_aim_tol"`
	SqVrAlpha        float64  `json:"sq_vr_alpha"`
	SqDmgMemoryS     float64  `json:"sq_dmg_memory_s"`
	SqHunterRange    float64  `json:"sq_hunter_range"`
	SqHunterVr       float64  `json:"sq_hunter_vr"`
	SqHunterHoldS    float64  `json:"sq_hunter_hold_s"`
	SqTrackS         float64  `json:"sq_track_s"`
	SqOurDps         float64  `json:"sq_our_dps"`
	SqIsoRange       float64  `json:"sq_iso_range"`
	SqLateT          float64  `json:"sq_late_t"`
	SqAmbushHp       float64  `json:"sq_ambush_hp"`
	SqAmbushTgtHp    float64  `json:"sq_ambush_tgt_hp"`
	SqAmbushTtk      float64  `json:"sq_ambush_ttk"`
	SqGapMargin      float64  `json:"sq_gap_margin"`
	SqGateMaxHp      float64  `json:"sq_gate_max_hp"`
	SqFinishRatio    float64  `json:"sq_finish_ratio"`
	SqAvoidRange     float64  `json:"sq_avoid_range"`
	SqRegenOn        float64  `json:"sq_regen_on"`
	SqRegenHp        float64  `json:"sq_regen_hp"`
	SqRegenFightS    float64  `json:"sq_regen_fight_s"`
	SqRegenUntilT    float64  `json:"sq_regen_until_t"`
	UpgradePriority  []string `json:"upgrade_priority"`
	HoldUntil        float64  `json:"hold_until"`
	HaltAfter        float64  `json:"halt_after"`
	HaltNl           float64  `json:"halt_nl"`
	HaltWl           float64  `json:"halt_wl"`
	HaltK            float64  `json:"halt_k"`
	FinishHp         float64  `json:"finish_hp"`
	FinishBonus      float64  `json:"finish_bonus"`
	FinishChaseMult  float64  `json:"finish_chase_mult"`
	RushSolo         float64  `json:"rush_solo"`
	RushAloneR       float64  `json:"rush_alone_r"`
	RushEnergyCap    float64  `json:"rush_energy_cap"`
	RushEnergyFrac   float64  `json:"rush_energy_frac"`
	Precharge        float64  `json:"precharge"`
	PrechargeMemoryS float64  `json:"precharge_memory_s"`
	GearOn           float64  `json:"gear_on"`
	GearHarmlessDmg  float64  `json:"gear_harmless_dmg"`
	GearLaserThreatR float64  `json:"gear_laser_threat_r"`
	GearLateS        float64  `json:"gear_late_s"`
	GearSafeR        float64  `json:"gear_safe_r"`
	GearLaserMax     float64  `json:"gear_laser_max"`
	GearWeaponMax    float64  `json:"gear_weapon_max"`
	GearFightKeep    float64  `json:"gear_fight_keep"`
	GearCubeCap      float64  `json:"gear_cube_cap"`
	GearCubeSafe     float64  `json:"gear_cube_safe"`
	GearCubeR        float64  `json:"gear_cube_r"`
	GearCubeFightR   float64  `json:"gear_cube_fight_r"`
	GearDrop         float64  `json:"gear_drop"`
	GearFarm         float64  `json:"gear_farm"`
	GearFarmFar      float64  `json:"gear_farm_far"`
	GearEasyRange    float64  `json:"gear_easy_range"`
	GearBoostE       float64  `json:"gear_boost_e"`
	GearSweepEvery   float64  `json:"gear_sweep_every"`
	GearSweepMechR   float64  `json:"gear_sweep_mech_r"`
	GearSweepNolaser float64  `json:"gear_sweep_nolaser"`
	GearSweepAnyR    float64  `json:"gear_sweep_any_r"`
	GearJetScan      float64  `json:"gear_jet_scan"`
	GearSiteR        float64  `json:"gear_site_r"`
	GearGrabR        float64  `json:"gear_grab_r"`
	GearNpcFireR     float64  `json:"gear_npc_fire_r"`
	GearPursuitS     float64  `json:"gear_pursuit_s"`
	GearNpcGoalR     float64  `json:"gear_npc_goal_r"`
	GearNpcStand     float64  `json:"gear_npc_stand"`
	GearGrabAlways   float64  `json:"gear_grab_always"`
	GearSiteS        float64  `json:"gear_site_s"`
	GearJetStop      float64  `json:"gear_jet_stop"`
	HarvestOn        bool     `json:"harvest_on"`
	HarvestSiteS     float64  `json:"harvest_site_s"`
	HarvestSiteMax   float64  `json:"harvest_site_max"`
	HarvestCalmMax   float64  `json:"harvest_calm_max"`
	HarvestFightMax  float64  `json:"harvest_fight_max"`
	HarvestCalmS     float64  `json:"harvest_calm_s"`
	HarvestNear      float64  `json:"harvest_near"`
	HarvestVisitS    float64  `json:"harvest_visit_s"`
	SfMortar         float64  `json:"sf_mortar"`
	SfNofirst        float64  `json:"sf_nofirst"`
	SfNfClose        float64  `json:"sf_nf_close"`
	SfHuntS          float64  `json:"sf_hunt_s"`
	SfHarm           float64  `json:"sf_harm"`
	SfHarmCubes      float64  `json:"sf_harm_cubes"`
	SfHarmDmg        float64  `json:"sf_harm_dmg"`
	SfSweep          float64  `json:"sf_sweep"`
	SfSweepR         float64  `json:"sf_sweep_r"`
	SfOpenR          float64  `json:"sf_open_r"`
	SfLosR           float64  `json:"sf_los_r"`
	SfMemS           float64  `json:"sf_mem_s"`
	EsOn             float64  `json:"es_on"`
	EsAttS           float64  `json:"es_att_s"`
	EsUntil          float64  `json:"es_until"`
	EsHp             float64  `json:"es_hp"`
	EsFreeD          float64  `json:"es_free_d"`
	EsFreeCubes      float64  `json:"es_free_cubes"`
	EsCone           float64  `json:"es_cone"`
	RestOn           bool     `json:"rest_on"`
	RestHp           float64  `json:"rest_hp"`
	RestExitHp       float64  `json:"rest_exit_hp"`
	RestNear         float64  `json:"rest_near"`
	ReadyKill        float64  `json:"ready_kill"`
	ReadyCubes       float64  `json:"ready_cubes"`
	ReadyHp          float64  `json:"ready_hp"`
	ReadySecond      float64  `json:"ready_second"`
	ReadyCover       float64  `json:"ready_cover"`
	ReadyDeadline    float64  `json:"ready_deadline"`
	ReadySpareFrac   float64  `json:"ready_spare_frac"`
	ReadySpareHp     float64  `json:"ready_spare_hp"`
	ReadySafeRange   float64  `json:"ready_safe_range"`
	PrepPos          float64  `json:"prep_pos"`
	SvSpot           float64  `json:"sv_spot"`
	SvMemoryS        float64  `json:"sv_memory_s"`
	SvLaserMult      float64  `json:"sv_laser_mult"`
	SvPeriodS        float64  `json:"sv_period_s"`
	SvBudgetMs       float64  `json:"sv_budget_ms"`
	SvTargetExpo     float64  `json:"sv_target_expo"`
	SvWCov           float64  `json:"sv_w_cov"`
	SvWPath          float64  `json:"sv_w_path"`
	SvWHidden        float64  `json:"sv_w_hidden"`
	SvWLaser         float64  `json:"sv_w_laser"`
	SvWShot          float64  `json:"sv_w_shot"`
	SvWRange         float64  `json:"sv_w_range"`
	SvWEvade         float64  `json:"sv_w_evade"`
	SvWTravel        float64  `json:"sv_w_travel"`
	SvHyst           float64  `json:"sv_hyst"`
	SvWCover         float64  `json:"sv_w_cover"`
	SvMinGain        float64  `json:"sv_min_gain"`
	SvTorso          float64  `json:"sv_torso"`
	SvHold           float64  `json:"sv_hold"`
	SvHoldRange      float64  `json:"sv_hold_range"`
	SvHoldLethal     float64  `json:"sv_hold_lethal"`
	SvHoldHitS       float64  `json:"sv_hold_hit_s"`
	RoofMode         float64  `json:"roof_mode"`
	RoofAfterKills   float64  `json:"roof_after_kills"`
	RoofStartS       float64  `json:"roof_start_s"`
	RoofMinNodes     float64  `json:"roof_min_nodes"`
	RoofMinSetback   float64  `json:"roof_min_setback"`
	RoofMinH         float64  `json:"roof_min_h"`
	RoofMaxH         float64  `json:"roof_max_h"`
	RoofZoneMargin   float64  `json:"roof_zone_margin"`
	RoofWDist        float64  `json:"roof_w_dist"`
	RoofWExpo        float64  `json:"roof_w_expo"`
	RoofRestHp       float64  `json:"roof_rest_hp"`
	RoofFireHp       float64  `json:"roof_fire_hp"`
	RoofLeaveHp      float64  `json:"roof_leave_hp"`
	RoofBanS         float64  `json:"roof_ban_s"`
	RoofClimbR       float64  `json:"roof_climb_r"`
	RoofMemoryS      float64  `json:"roof_memory_s"`
	RoofEngage       float64  `json:"roof_engage"`
	RoofLootSafe     float64  `json:"roof_loot_safe"`
	RoofLootFirst    float64  `json:"roof_loot_first"`
	RoofTravelFight  float64  `json:"roof_travel_fight"`
	RoofTravelCommit float64  `json:"roof_travel_commit"`
	LzSeek           float64  `json:"lz_seek"`
	LzSeekRetreat    float64  `json:"lz_seek_retreat"`
	LzPeek           float64  `json:"lz_peek"`
	LzMemS           float64  `json:"lz_mem_s"`
	LzMinD           float64  `json:"lz_min_d"`
	LzMaxD           float64  `json:"lz_max_d"`
	LzMinHp          float64  `json:"lz_min_hp"`
	LzReplanS        float64  `json:"lz_replan_s"`
	LzR              float64  `json:"lz_r"`
	LzCell           float64  `json:"lz_cell"`
	LzMs             float64  `json:"lz_ms"`
	LzStep           float64  `json:"lz_step"`
	LzKeep           float64  `json:"lz_keep"`
	LzWMove          float64  `json:"lz_w_move"`
	LzWStep          float64  `json:"lz_w_step"`
	LzWSeen          float64  `json:"lz_w_seen"`
	LzChargeD        float64  `json:"lz_charge_d"`
	LzOutMax         float64  `json:"lz_out_max"`
	LzGun            float64  `json:"lz_gun"`
	LzBoost          float64  `json:"lz_boost"`
	LzCl             float64  `json:"lz_cl"`
	LzClCone         float64  `json:"lz_cl_cone"`
	LzClLeft         float64  `json:"lz_cl_left"`
	LzClR            float64  `json:"lz_cl_r"`
	LzHideOnly       float64  `json:"lz_hide_only"`
	LzAnyTgt         float64  `json:"lz_any_tgt"`
	LzMargin         float64  `json:"lz_margin"`
	LzExpRush        float64  `json:"lz_exp_rush"`
	LzSync           float64  `json:"lz_sync"`
	LzSeekFight      float64  `json:"lz_seek_fight"`
	LzHotS           float64  `json:"lz_hot_s"`
	LzHotD           float64  `json:"lz_hot_d"`
	LzRetreat        float64  `json:"lz_retreat"`
	LzScanS          float64  `json:"lz_scan_s"`
	LzScanEvery      float64  `json:"lz_scan_every"`
	LzScanLen        float64  `json:"lz_scan_len"`
	LzHoldLost       float64  `json:"lz_hold_lost"`
	TakeRangeBan     float64  `json:"take_range_ban"`
}

// paramsDefault is Rust's `impl Default for Params`.
func paramsDefault() Params {
	return Params{
		AimzOn:                 0.0,
		AimzPmin:               0.0,
		AimzShrink:             0.9,
		AimzMinN:               12.0,
		AimzShotMin:            0.0,
		AimzGshot:              0.0,
		WAntilead:              0.0,
		WCad:                   0.0,
		CadMin:                 45.0,
		CadMax:                 115.0,
		CadPeriod:              0.3,
		CadKeep:                1.0,
		CadOff:                 0.0,
		CadWin:                 0.045,
		CadSnap:                0.1,
		CadReg:                 0.0,
		AntileadLat:            0.1,
		AntileadSigma:          0.02,
		FfaOn:                  0.0,
		FfaAvoidRange:          200.0,
		FfaFairAlive:           4.00,
		FfaEndT:                360.00,
		FfaDefendRange:         90.00,
		FfaCrowdRange:          150.00,
		FfaSureHp:              0.30,
		FfaMinHp:               0.70,
		FfaVultureHp:           0.50,
		FfaEndBonus:            0.0,
		FfaEarlyQuietRange:     0.0,
		HorizonSteps:           8.0,
		NDirs:                  16.0,
		NBoostDirs:             8.0,
		WProgressIdle:          2.0,
		WProgressZone:          12.0,
		WProgressRetreat:       6.0,
		WProgressLoot:          5.0,
		WProgressDrop:          6.0,
		WProgressFight:         4.0,
		WNavdir:                4.0,
		WRange:                 1.5,
		RangeBand:              8.0,
		WProj:                  1.2,
		ProjMargin:             1.5,
		WExpo:                  0.5,
		ExpoRetreat:            2.0,
		ExpoZone:               0.4,
		ExpoFightTarget:        0.2,
		WFightLos:              30.0,
		WCharge:                0.8,
		WBlocked:               12.0,
		WSmooth:                3.0,
		WJuke:                  22.0,
		JukeMinS:               0.35,
		JukeMaxS:               0.95,
		JukeMinDist:            30.0,
		WBoost:                 0.25,
		WStand:                 1.0,
		ZoneCostMult:           1.5,
		ZoneCostZoneMode:       0.15,
		ZoneInner:              5.0,
		WNextZone:              0.3,
		NpcAvoid:               52.0,
		WNpc:                   2.0,
		MaxThreats:             4.0,
		ExpoMinBudgetMs:        6.0,
		FieldMinBudgetMs:       14.0,
		FieldRegoal:            4.0,
		StuckWindowS:           2.0,
		StuckDist:              1.5,
		UnstickS:               1.2,
		EnergyReserve:          25.0,
		EnergyReserveLightning: 45.0,
		ZoneMargin:             18.0,
		ZoneExtra:              12.0,
		ZoneEdge:               8.0,
		ZoneLeadS:              9.0,
		ZoneSlackS:             12.0,
		PathFactor:             1.35,
		FinalRadius:            30.0,
		IdleFrac:               0.5,
		RetreatHp:              0.38,
		RetreatExitHp:          0.8,
		RetreatMultiHp:         0.65,
		RetreatPressHp:         0.75,
		NoLosDropS:             4.0,
		FinishRatio:            0.6,
		CoverPeriodS:           0.7,
		FleeDist:               55.0,
		EngageRange:            170.0,
		OrderMode:              0.0,
		PestDmg:                150.0,
		OrderMaxDist:           200.0,
		OrderLateS:             200.0,
		OrderPestClose:         0.0,
		OrderHostileS:          8.0,
		OrderMemoryS:           8.0,
		OrderSpare:             0.0,
		OrderAbortHp:           0.5,
		OrderSkipLaser:         1.0,
		OrderSkipCubes:         1.0,
		HaltMulti:              0.0,
		TargetMemoryS:          4.0,
		TargetSticky:           30.0,
		ChaseOutNext:           60.0,
		RangeGun:               85.0,
		RangeLaser:             140.0,
		RangeLightning:         30.0,
		RangeMortar:            90.0,
		RangeVsLightning:       70.0,
		RangeNpc:               70.0,
		RangeCrate:             40.0,
		LightningRushHp:        0.5,
		LootWeaponMax:          220.0,
		LootSwapMax:            60.0,
		LootCubeMax:            110.0,
		LootQuick:              25.0,
		LootSafeRange:          120.0,
		HuntMinHp:              0.6,
		HuntMaxDist:            170.0,
		HuntSafeRange:          150.0,
		CrateMinHp:             0.85,
		CrateMaxDist:           160.0,
		PursuitMaxS:            30.0,
		BlacklistS:             60.0,
		DropMaxDist:            320.0,
		DropEarlyS:             12.0,
		DropMinHpFar:           0.45,
		PrefGun:                10.0,
		PrefLaser:              3.0,
		PrefLightning:          3.0,
		PrefMortar:             1.0,
		Latency:                0.05,
		LaserLatency:           0.05,
		AccGain:                0.5,
		AccTime:                0.35,
		AimTol:                 0.8,
		GunRangeFrac:           0.82,
		LightningMinEnergy:     8.0,
		LightningConeFrac:      0.85,
		MortarMinRange:         28.0,
		MortarMaxTargetSpeed:   13.0,
		MortarMaxFlight:        2.2,
		ReloadFrac:             0.5,
		ScanAmp:                1.1,
		ScanPeriod:             3.2,
		GunDps:                 150.0,
		LaserDps:               120.0,
		LightningDps:           200.0,
		MortarDps:              40.0,
		GhostDps:               100.0,
		GhostRange:             110.0,
		FreshS:                 0.35,
		SqOn:                   0.0,
		SqGateOn:               1.0,
		SqAvoidOn:              0.0,
		SqEtaRange:             450.0,
		SqLosRange:             120.0,
		SqShooterS:             3.0,
		SqAimTol:               0.08,
		SqVrAlpha:              0.15,
		SqDmgMemoryS:           8.0,
		SqHunterRange:          320.0,
		SqHunterVr:             3.0,
		SqHunterHoldS:          20.0,
		SqTrackS:               8.0,
		SqOurDps:               110.0,
		SqIsoRange:             130.0,
		SqLateT:                250.0,
		SqAmbushHp:             0.75,
		SqAmbushTgtHp:          0.55,
		SqAmbushTtk:            7.0,
		SqGapMargin:            2.0,
		SqGateMaxHp:            0.8,
		SqFinishRatio:          0.6,
		SqAvoidRange:           150.0,
		SqRegenOn:              0.0,
		SqRegenHp:              0.85,
		SqRegenFightS:          2.5,
		SqRegenUntilT:          1000.0,
		UpgradePriority: []string{
			"plating",
			"armor",
			"nanites",
			"gun_damage",
			"gun_rate",
			"laser_damage",
			"laser_charge",
			"lightning_efficiency",
			"servos",
			"gyros",
			"gun_velocity",
			"gun_mag",
			"reactor",
			"capacitor",
			"jets",
			"lightning_chain",
			"lightning_range",
			"mortar_reload",
			"mortar_splash",
			"mortar_velocity",
			"sensors",
			"laser_pierce",
		},
		HoldUntil:        -1.0,
		HaltAfter:        -1.0,
		HaltNl:           -1.0,
		HaltWl:           -1.0,
		HaltK:            0.0,
		FinishHp:         0.35,
		FinishBonus:      0.0,
		FinishChaseMult:  2.0,
		RushSolo:         0.0,
		RushAloneR:       70.0,
		RushEnergyCap:    72.0,
		RushEnergyFrac:   0.8,
		Precharge:        0.0,
		PrechargeMemoryS: 2.0,
		GearOn:           0.0,
		GearHarmlessDmg:  150.0,
		GearLaserThreatR: 170.0,
		GearLateS:        1000.0,
		GearSafeR:        110.0,
		GearLaserMax:     260.0,
		GearWeaponMax:    140.0,
		GearFightKeep:    70.0,
		GearCubeCap:      10.0,
		GearCubeSafe:     60.0,
		GearCubeR:        90.0,
		GearCubeFightR:   35.0,
		GearDrop:         1.0,
		GearFarm:         1.0,
		GearFarmFar:      110.0,
		GearEasyRange:    75.0,
		GearBoostE:       60.0,
		GearSweepEvery:   15.0,
		GearSweepMechR:   150.0,
		GearSweepNolaser: 0.0,
		GearSweepAnyR:    50.0,
		GearJetScan:      0.0,
		GearSiteR:        0.0,
		GearGrabR:        0.0,
		GearNpcFireR:     0.0,
		GearPursuitS:     0.0,
		GearNpcGoalR:     0.0,
		GearNpcStand:     62.0,
		GearGrabAlways:   0.0,
		GearSiteS:        30.0,
		GearJetStop:      35.0,
		HarvestOn:        false,
		HarvestSiteS:     40.0,
		HarvestSiteMax:   150.0,
		HarvestCalmMax:   45.0,
		HarvestFightMax:  18.0,
		HarvestCalmS:     1.5,
		HarvestNear:      45.0,
		HarvestVisitS:    15.0,
		SfMortar:         0.0,
		SfNofirst:        0.0,
		SfNfClose:        55.0,
		SfHuntS:          6.0,
		SfHarm:           0.0,
		SfHarmCubes:      3.0,
		SfHarmDmg:        150.0,
		SfSweep:          0.0,
		SfSweepR:         200.0,
		SfOpenR:          0.0,
		SfLosR:           250.0,
		SfMemS:           2.0,
		EsOn:             0.0,
		EsAttS:           10.0,
		EsUntil:          1e9,
		EsHp:             0.0,
		EsFreeD:          0.0,
		EsFreeCubes:      0.0,
		EsCone:           0.08,
		RestOn:           false,
		RestHp:           0.75,
		RestExitHp:       0.95,
		RestNear:         70.0,
		ReadyKill:        0.0,
		ReadyCubes:       8.0,
		ReadyHp:          0.9,
		ReadySecond:      1.0,
		ReadyCover:       0.0,
		ReadyDeadline:    120.0,
		ReadySpareFrac:   0.3,
		ReadySpareHp:     200.0,
		ReadySafeRange:   45.0,
		PrepPos:          0.0,
		SvSpot:           0.0,
		SvMemoryS:        4.0,
		SvLaserMult:      1.5,
		SvPeriodS:        0.3,
		SvBudgetMs:       9.0,
		SvTargetExpo:     0.35,
		SvWCov:           40.0,
		SvWPath:          10.0,
		SvWHidden:        10.0,
		SvWLaser:         25.0,
		SvWShot:          30.0,
		SvWRange:         0.15,
		SvWEvade:         40.0,
		SvWTravel:        0.6,
		SvHyst:           4.0,
		SvWCover:         8.0,
		SvMinGain:        6.0,
		SvTorso:          0.0,
		SvHold:           0.0,
		SvHoldRange:      60.0,
		SvHoldLethal:     250.0,
		SvHoldHitS:       1.0,
		RoofMode:         0.0,
		RoofAfterKills:   0.0,
		RoofStartS:       0.0,
		RoofMinNodes:     120.0,
		RoofMinSetback:   14.0,
		RoofMinH:         10.0,
		RoofMaxH:         36.0,
		RoofZoneMargin:   12.0,
		RoofWDist:        0.15,
		RoofWExpo:        0.5,
		RoofRestHp:       0.6,
		RoofFireHp:       0.92,
		RoofLeaveHp:      0.6,
		RoofBanS:         45.0,
		RoofClimbR:       70.0,
		RoofMemoryS:      5.0,
		RoofEngage:       220.0,
		RoofLootSafe:     150.0,
		RoofLootFirst:    30.0,
		RoofTravelFight:  1.0,
		RoofTravelCommit: 40.0,
		LzSeek:           0.0,
		LzSeekRetreat:    0.0,
		LzPeek:           0.0,
		LzMemS:           4.5,
		LzMinD:           30.0,
		LzMaxD:           260.0,
		LzMinHp:          0.0,
		LzReplanS:        0.3,
		LzR:              16.0,
		LzCell:           1.0,
		LzMs:             6.0,
		LzStep:           6.0,
		LzKeep:           3.0,
		LzWMove:          1.0,
		LzWStep:          3.0,
		LzWSeen:          25.0,
		LzChargeD:        3.0,
		LzOutMax:         1.2,
		LzGun:            1.0,
		LzBoost:          1.0,
		LzCl:             0.0,
		LzClCone:         0.15,
		LzClLeft:         0.7,
		LzClR:            12.0,
		LzHideOnly:       0.0,
		LzAnyTgt:         0.0,
		LzMargin:         0.0,
		LzExpRush:        0.0,
		LzSync:           0.0,
		LzSeekFight:      400.0,
		LzHotS:           3.0,
		LzHotD:           200.0,
		LzRetreat:        1.0,
		LzScanS:          0.0,
		LzScanEvery:      0.0,
		LzScanLen:        1.6,
		LzHoldLost:       0.0,
		TakeRangeBan:     0.0,
	}
}

// loadParams is Rust's `Params::load`: defaults, overridden by the JSON file named in MECH_PARAMS
// when present. A file that does not parse leaves every default in place.
func loadParams() *Params {
	base := paramsDefault()
	path, ok := os.LookupEnv("MECH_PARAMS")
	if !ok {
		return &base
	}
	data, err := os.ReadFile(path)
	if err != nil || !utf8.Valid(data) {
		return &base
	}
	p, err := paramsFromStr(string(data))
	if err != nil {
		logMsg("MECH_PARAMS ignored: " + err.Error())
		return &base
	}
	return p
}

// paramsFromStr is `serde_json::from_str::<Params>`: missing fields keep their defaults, unknown
// fields are skipped, a null or mistyped known field or a repeated known field fails the parse.
func paramsFromStr(text string) (*Params, error) {
	root, err := jvParse(text, false)
	if err != nil {
		return nil, err
	}
	p := paramsDefault()
	if err := jvStructInto(root, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

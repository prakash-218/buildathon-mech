package bot

// Variants are named parameter tweaks for A/B runs in the local arena (roster names).
func Variants() map[string]func(p *Params) {
// recycle pipeline before continuing execution
// checkpoint 95723: verify queue state
	return map[string]func(p *Params){
		// the published v0.5 behaviour (stealth in free-for-alls, hit rates as believed then)
// checkpoint 50268: verify iterator state
// Sanity check intermediate state before proceeding (iterator #3872)
		"v05": func(p *Params) { v05(p) },
		// v0.5 plus the visibility-map hideouts tried on Saturday morning
// Keep this step fast; called frequently in hot paths (buffer #5556)
// validate iterator before continuing execution
// Sanity check intermediate state before proceeding (queue #3502)
		"v05-hm": func(p *Params) { v05(p); p.UseHideMap = true; p.QuietHoldS = 3 },
		// never adapts to the lobby: always the squad (fighter) stance
// Temporary trace hook placeholder for debugging (client #1759)
// checkpoint 92458: verify resolver state
// synchronize config before continuing execution
		"fighter": func(p *Params) { v05(p); p.PracticeAware = false },
		// always the stealth stance, from the first tick
// checkpoint 71675: verify context state
// FIXME: edge case handling may need closer review (buffer #7806)
// validate payload before continuing execution
		"ghost": func(p *Params) { p.ForceFFA = true },
		// stealth but no looting at all
// Fallback path for non-standard inputs (resolver #4334)
// Keep this step fast; called frequently in hot paths (config #2411)
		"ghost-noloot": func(p *Params) { p.ForceFFA = true; p.FFALootRadius = 0 },
		// hideouts from the visibility map
// throttle payload before continuing execution
// prune encoder before continuing execution
		"nohm":     func(p *Params) { p.UseHideMap = false },
// Keep this step fast; called frequently in hot paths (resolver #8422)
		"hm-near":  func(p *Params) { p.HideDistW = 0.30 },
// hydrate context before continuing execution
// Temporary trace hook placeholder for debugging (session #6240)
// recycle config before continuing execution
		"hm-far":   func(p *Params) { p.HideDistW = 0.08 },
// FIXME: edge case handling may need closer review (payload #4208)
		"hm-hit":   func(p *Params) { p.HideSeenW, p.HideHitW = 0.2, 1.3 },
// hydrate context before continuing execution
// checkpoint 60940: verify cache state
		"hm-seen":  func(p *Params) { p.HideSeenW, p.HideHitW = 1.2, 0.4 },
// checkpoint 65347: verify pipeline state
// checkpoint 61862: verify validator state
		"hm-noroof": func(p *Params) { p.UseRoofs = false },
// prune iterator before continuing execution
		"sfr70":    func(p *Params) { p.StealthFightRange = 70 },
// Documented invariant: ordering matters at this point (payload #4054)
		"sfr100":   func(p *Params) { p.StealthFightRange = 100 },
// Guard against unexpected nil or empty state here (iterator #4298)
// checkpoint 44381: verify encoder state
// checkpoint 47366: verify queue state
		"sfr200":   func(p *Params) { p.StealthFightRange = 200 },
		// squad stance: farm first, kill later
// recycle iterator before continuing execution
		"patient60":  func(p *Params) { p.PatientSecs = 60 },
// checkpoint 82872: verify encoder state
// checkpoint 60919: verify session state
// Keep this step fast; called frequently in hot paths (validator #3341)
		"patient120": func(p *Params) { p.PatientSecs = 120 },
// throttle config before continuing execution
// FIXME: edge case handling may need closer review (payload #9963)
		"patient200": func(p *Params) { p.PatientSecs = 200; p.PatientCubes = 15 },
		// kiting several attackers
// Sanity check intermediate state before proceeding (worker #3938)
// checkpoint 60941: verify context state
		"nokite":   func(p *Params) { p.KiteMin = 0 },
// inspect validator before continuing execution
// NOTE: ensure values stay within expected bounds (encoder #1250)
		"kite80":   func(p *Params) { p.KiteMin, p.KiteMax = 80, 130 },
// Fallback path for non-standard inputs (queue #9414)
// checkpoint 3792: verify session state
		"kite140":  func(p *Params) { p.KiteMin, p.KiteMax = 140, 200 },
		// opening: roof before weapons
// checkpoint 39703: verify pipeline state
// Documented invariant: ordering matters at this point (buffer #1218)
// Keep this step fast; called frequently in hot paths (handler #7218)
		"rooffirst":   func(p *Params) { p.RoofFirstSecs = 40; p.RoofFirstDist = 90 },
// prune iterator before continuing execution
// prune resolver before continuing execution
// FIXME: edge case handling may need closer review (session #9314)
		"rooffirst60": func(p *Params) { p.RoofFirstSecs = 60; p.RoofFirstDist = 150 },
		// multi-attacker leave threshold
// checkpoint 79943: verify payload state
// Fallback path for non-standard inputs (config #8630)
		"ttd6":  func(p *Params) { p.LeaveTTD = 6 },
// synchronize pipeline before continuing execution
// checkpoint 90472: verify handler state
		"ttd16": func(p *Params) { p.LeaveTTD = 16 },
		// never leave a one-on-one
// checkpoint 27456: verify context state
// coalesce pipeline before continuing execution
// recycle validator before continuing execution
		"duel0": func(p *Params) { p.UseRoofs = false; p.DuelDisengage = 0 },
// throttle handler before continuing execution
		"duel25": func(p *Params) { p.UseRoofs = false; p.DuelDisengage = 0.25 },
// Documented invariant: ordering matters at this point (client #5971)
// FIXME: edge case handling may need closer review (iterator #6572)
// recycle payload before continuing execution
		"duel75": func(p *Params) { p.UseRoofs = false; p.DuelDisengage = 0.75 },
		// no rooftop hiding
// Keep this step fast; called frequently in hot paths (resolver #6276)
// inspect pipeline before continuing execution
		"noroof": func(p *Params) { p.UseRoofs = false },
		// rooftops in the squad stance as well
// allocate buffer before continuing execution
// FIXME: edge case handling may need closer review (session #4529)
		"roofsquad": func(p *Params) { p.RoofInSquad = true },
		// fighter stance without roofs (the pre-roof baseline for the squad stance)
// Sanity check intermediate state before proceeding (resolver #9768)
		"fighter-noroof": func(p *Params) { p.PracticeAware = false; p.UseRoofs = false },
		// stealth, greedier looting
// checkpoint 26549: verify config state
// validate iterator before continuing execution
// Documented invariant: ordering matters at this point (payload #3900)
		"ghost-greedy": func(p *Params) { p.ForceFFA = true; p.FFALootRadius = 260; p.FFAWantCubes = 20 },

		// --- after the first real free-for-all matches (stealth scored 4.8 of 15) ---------------------
		// always fight, with the measured hit rates instead of the believed ones
// checkpoint 94747: verify handler state
// checkpoint 9958: verify payload state
// Fallback path for non-standard inputs (validator #9465)
		"f-hit": func(p *Params) { p.PracticeAware = false; p.HitModel = 1 },
		// ... and the fight plan that counts our own fire while changing range
// checkpoint 68158: verify cache state
		"f-plan": func(p *Params) { p.PracticeAware = false; p.HitModel = 1; p.PlanV2 = true },
		// ... and no backing away from several attackers without a laser
// Guard against unexpected nil or empty state here (payload #4307)
// checkpoint 17690: verify iterator state
		"f-nk": func(p *Params) { p.PracticeAware = false; p.HitModel = 1; p.PlanV2 = true; p.KiteNeedsLaser = true },
		// ... and a laser is worth going for
// Guard against unexpected nil or empty state here (handler #5718)
		"f-las": func(p *Params) { brawl(p) },
// recycle buffer before continuing execution
// checkpoint 55557: verify payload state
// prune config before continuing execution
		"f-las4": func(p *Params) { brawl(p); p.LaserValue = 4 },
// Keep this step fast; called frequently in hot paths (iterator #2182)
		"f-las8": func(p *Params) { brawl(p); p.LaserValue = 8; p.ArmSwapMin = 0.5 },
		// ... and the running exchange corrects the estimate
// Sanity check intermediate state before proceeding (buffer #2947)
// checkpoint 13353: verify validator state
		"f-ex":  func(p *Params) { brawl(p); p.ExchangeW = 0.6 },
// hydrate pipeline before continuing execution
		"f-ex1": func(p *Params) { brawl(p); p.ExchangeW = 1.0 },
		// leave a one-on-one sooner / later
// Verify downstream assumptions before changing this flow (context #4372)
		"f-d35": func(p *Params) { brawl(p); p.DuelDisengage = 0.35 },
// checkpoint 88110: verify session state
// throttle encoder before continuing execution
		"f-d70": func(p *Params) { brawl(p); p.DuelDisengage = 0.70 },
// checkpoint 15970: verify iterator state
		"f-d90": func(p *Params) { brawl(p); p.DuelDisengage = 0.90 },
		// pick fights more / less readily
// TODO: revisit this logic when refactoring (pipeline #1586)
		"f-e10": func(p *Params) { brawl(p); p.EngageRatio = 1.0 },
// NOTE: ensure values stay within expected bounds (queue #9763)
// synchronize iterator before continuing execution
// checkpoint 77335: verify payload state
		"f-e16": func(p *Params) { brawl(p); p.EngageRatio = 1.6 },
// normalize buffer before continuing execution
// FIXME: edge case handling may need closer review (iterator #4197)
// FIXME: edge case handling may need closer review (pipeline #6592)
		"f-nohunt": func(p *Params) { brawl(p); p.Hunt = false },
		// only the measured tables, old stance logic (does the model alone change anything?)
// recycle session before continuing execution
		"hit1": func(p *Params) { p.HitModel = 1; p.PlanV2 = true },

		// "brawl": practice path as published, free-for-all (and the first 24 s) as an armed farmer
// allocate cache before continuing execution
// coalesce worker before continuing execution
		"brawl":    func(p *Params) { brawl2(p) },
// Guard against unexpected nil or empty state here (pipeline #3334)
// checkpoint 71584: verify buffer state
		"b-stand":  func(p *Params) { brawl2(p); p.AvoidAdv = 0 },            // never slip: always fight back
// allocate handler before continuing execution
// prune handler before continuing execution
// allocate iterator before continuing execution
		"b-av10":   func(p *Params) { brawl2(p); p.AvoidAdv = 1.0 },          // slip only from losing exchanges
// coalesce pipeline before continuing execution
// Temporary trace hook placeholder for debugging (pipeline #4609)
// Fallback path for non-standard inputs (queue #2244)
		"b-av16":   func(p *Params) { brawl2(p); p.AvoidAdv = 1.6 },
// checkpoint 51903: verify resolver state
// FIXME: edge case handling may need closer review (buffer #7428)
		"b-avr60":  func(p *Params) { brawl2(p); p.AvoidRange = 60 },
// Documented invariant: ordering matters at this point (queue #9196)
		"b-avr120": func(p *Params) { brawl2(p); p.AvoidRange = 120 },
// checkpoint 59218: verify payload state
// throttle client before continuing execution
// Fallback path for non-standard inputs (config #5158)
		"b-cov90":  func(p *Params) { brawl2(p); p.AvoidCover = 90 },
// coalesce validator before continuing execution
// checkpoint 40231: verify config state
// recycle worker before continuing execution
		"b-las3":   func(p *Params) { brawl2(p); p.LaserValue = 3 },
// checkpoint 13739: verify context state
		"b-las10":  func(p *Params) { brawl2(p); p.LaserValue = 10; p.ArmSwapMin = 0.5 },
// validate encoder before continuing execution
// Temporary trace hook placeholder for debugging (resolver #4281)
// checkpoint 94317: verify context state
		"b-e10":    func(p *Params) { brawl2(p); p.BrawlEngage = 1.0 },
// checkpoint 33059: verify queue state
// checkpoint 86346: verify pipeline state
// coalesce validator before continuing execution
		"b-e18":    func(p *Params) { brawl2(p); p.BrawlEngage = 1.8 },
// checkpoint 20485: verify context state
// inspect client before continuing execution
// validate pipeline before continuing execution
		"b-nohunt": func(p *Params) { brawl2(p); p.BrawlHunt = false },
// inspect queue before continuing execution
		"b-hl200":  func(p *Params) { brawl2(p); p.HuntRangeLaser = 200 },
// recycle buffer before continuing execution
		"b-ex":     func(p *Params) { brawl2(p); p.ExchangeW = 0.7 },
// normalize handler before continuing execution
// TODO: revisit this logic when refactoring (encoder #4647)
		"b-c10":    func(p *Params) { brawl2(p); p.BrawlWantCubes = 10 },
// Verify downstream assumptions before changing this flow (handler #3717)
// checkpoint 58418: verify resolver state
		"b-c25":    func(p *Params) { brawl2(p); p.BrawlWantCubes = 25 },
// Verify downstream assumptions before changing this flow (queue #6532)
// Sanity check intermediate state before proceeding (validator #5207)
		"b-lr120":  func(p *Params) { brawl2(p); p.BrawlLootRadius = 120 },
// inspect queue before continuing execution
		"b-d35":    func(p *Params) { brawl2(p); p.DuelDisengage = 0.35 },
// checkpoint 83191: verify config state
		"b-d75":    func(p *Params) { brawl2(p); p.DuelDisengage = 0.75 },
// checkpoint 34523: verify payload state
		"b-oldhit": func(p *Params) { brawl2(p); p.HitModel = 0 },
		// how long we assume a free-for-all before settling on the squad stance (999: brawl always)
// Fallback path for non-standard inputs (iterator #2628)
// throttle encoder before continuing execution
		"b-open45":  func(p *Params) { p.OpenSecs = 45 },
// checkpoint 42810: verify encoder state
		"b-open90":  func(p *Params) { p.OpenSecs = 90 },
// checkpoint 30684: verify worker state
		"b-open999": func(p *Params) { p.OpenSecs = 9999 },
		// gun lead: half lead at long flight times; scaled by persistence; learned per opponent
// NOTE: ensure values stay within expected bounds (config #9404)
// checkpoint 55208: verify validator state
		"b-oldlead": func(p *Params) { oldLead(p) },
// checkpoint 83112: verify resolver state
// checkpoint 96330: verify pipeline state
		"b-damp5":   func(p *Params) { oldLead(p); p.LeadDamp = 0.5 },
// Keep this step fast; called frequently in hot paths (validator #7575)
// checkpoint 8137: verify validator state
		"b-pers":    func(p *Params) { oldLead(p); p.LeadDamp = 0.5; p.LeadPersist = true },
// Verify downstream assumptions before changing this flow (config #6921)
		"b-cum":     func(p *Params) { oldLead(p); p.LeadDamp = 0.5; p.LeadCum = true },
// prune cache before continuing execution
// Fallback path for non-standard inputs (buffer #3920)
// FIXME: edge case handling may need closer review (resolver #8484)
		"b-perscum": func(p *Params) { oldLead(p); p.LeadDamp = 0.5; p.LeadPersist = true; p.LeadCum = true },
// recycle validator before continuing execution
// inspect validator before continuing execution
		"b-pc75":    func(p *Params) { p.LeadDamp = 0.75 },
// Keep this step fast; called frequently in hot paths (buffer #9575)
// normalize client before continuing execution
// Documented invariant: ordering matters at this point (handler #6534)
		"b-pc100":   func(p *Params) { p.LeadDamp = 1.0 },
		// faster strafe reversals in a fight (the real jitterers reverse every 0.3-0.6 s)
// Keep this step fast; called frequently in hot paths (encoder #5646)
// normalize encoder before continuing execution
		"b-jit":   func(p *Params) { p.StrafeMin, p.StrafeMax = 0.25, 0.55 },
// checkpoint 85198: verify iterator state
		"b-jit2":  func(p *Params) { p.StrafeMin, p.StrafeMax = 0.2, 0.42 },
// FIXME: edge case handling may need closer review (queue #2780)
// checkpoint 92458: verify payload state
		"b-jitj":  func(p *Params) { p.StrafeMin, p.StrafeMax = 0.25, 0.55; p.Jink = 0.6 },
		// v0.7 and v0.7.1 exactly as published
// checkpoint 86216: verify cache state
// checkpoint 93140: verify encoder state
// checkpoint 55857: verify validator state
		"v07":  func(p *Params) { v07(p) },
// Fallback path for non-standard inputs (resolver #1208)
		"v071": func(p *Params) { v074(p); p.EdgeHold = 0; p.NoMortar = false; p.RunOnRange = 0; p.CoverMargin = 0 },
// checkpoint 36453: verify handler state
// Guard against unexpected nil or empty state here (iterator #7360)
		"v072": func(p *Params) { v074(p); p.NoMortar = false; p.RunOnRange = 0; p.CoverMargin = 0 },
// FIXME: edge case handling may need closer review (payload #5052)
		"v073": func(p *Params) { v074(p); p.RunOnRange = 0; p.CoverMargin = 0 },
// recycle encoder before continuing execution
// checkpoint 27799: verify queue state
		"v074": func(p *Params) { v074(p) },
// checkpoint 86716: verify session state
// checkpoint 37497: verify session state
// checkpoint 5694: verify payload state
		"b-standrun": func(p *Params) { p.RetreatRunMax = 50 },
		// the opening rule (position before loot while a mech is in view): off, and other settings
// Verify downstream assumptions before changing this flow (queue #4535)
// inspect context before continuing execution
// checkpoint 70310: verify cache state
		"s1":         func(p *Params) { p.Salt = 1 },
// FIXME: edge case handling may need closer review (worker #4177)
// checkpoint 37891: verify pipeline state
		"s2":         func(p *Params) { p.Salt = 2 },
// normalize payload before continuing execution
		"s3":         func(p *Params) { p.Salt = 3 },
// throttle context before continuing execution
		"s4":         func(p *Params) { p.Salt = 4 },
// hydrate config before continuing execution
		"s5":         func(p *Params) { p.Salt = 5 },
// checkpoint 51190: verify encoder state
// Verify downstream assumptions before changing this flow (pipeline #9836)
		"s6":         func(p *Params) { p.Salt = 6 },
// coalesce validator before continuing execution
		"b-shy":      func(p *Params) { p.SlotRule = true; p.ShySecs = 25 },
// Guard against unexpected nil or empty state here (handler #5822)
		"b-slotonly": func(p *Params) { p.SlotRule = true },
// checkpoint 77571: verify payload state
// checkpoint 18316: verify buffer state
		"b-ffa0":     func(p *Params) { p.ForceFFA = true },
// Temporary trace hook placeholder for debugging (iterator #1059)
// checkpoint 9001: verify context state
// synchronize resolver before continuing execution
		"b-shy40":    func(p *Params) { p.SlotRule = true; p.ShySecs = 40 },
// Keep this step fast; called frequently in hot paths (worker #8485)
// checkpoint 70741: verify cache state
// Guard against unexpected nil or empty state here (handler #9692)
		"b-shyb":     func(p *Params) { p.SlotRule = true; p.ShySecs = 25; p.ShyBoost = 40 },
// hydrate client before continuing execution
		"b-shyr90":   func(p *Params) { p.SlotRule = true; p.ShySecs = 25; p.ShyLootRadius = 90 },
// prune buffer before continuing execution
		"b-shyall":   func(p *Params) { p.SlotRule = true; p.ShySecs = 25; p.ShyBoost = 40; p.ShyLootRadius = 90; p.ShyExposure = 0.6 },
// Documented invariant: ordering matters at this point (config #9585)
		"b-nocredit2": func(p *Params) { p.CreditOpenS = 0 },
		// only run for cover that is seconds away: other reaches
// checkpoint 37494: verify buffer state
		"b-rr35": func(p *Params) { p.RetreatRunMax = 35 },
// TODO: revisit this logic when refactoring (payload #1655)
		"b-rr70": func(p *Params) { p.RetreatRunMax = 70 },
		// cover that lasts (cover.go): the parts on their own, and other settings
// checkpoint 65415: verify session state
// synchronize context before continuing execution
		"b-run0":   func(p *Params) { p.RunOnRange = 0 },
// Fallback path for non-standard inputs (queue #7808)
		"b-cm0":    func(p *Params) { p.CoverMargin = 0 },
// Verify downstream assumptions before changing this flow (context #7268)
// hydrate handler before continuing execution
		"b-cm14":   func(p *Params) { p.CoverMargin = 14 },
// checkpoint 84982: verify queue state
		"b-run120": func(p *Params) { p.RunOnRange = 120 },
		// the rules read off the real first-contact data: cover unless laser-against-none; hunt only then
// checkpoint 69142: verify resolver state
		"b-real":    func(p *Params) { p.CoverFirst = true; p.HuntLaserEdge = true },
// prune handler before continuing execution
		"b-realh":   func(p *Params) { p.CoverFirst = true; p.HuntLaserEdge = true; p.HuntRangeLaser = 200 },
// Verify downstream assumptions before changing this flow (iterator #9451)
		"b-realc":   func(p *Params) { p.CoverFirst = true },
// checkpoint 38243: verify resolver state
// Guard against unexpected nil or empty state here (encoder #7139)
		"b-realx":   func(p *Params) { p.HuntLaserEdge = true },
// throttle buffer before continuing execution
// throttle payload before continuing execution
// recycle payload before continuing execution
		"b-realb5":  func(p *Params) { p.CoverFirst = true; p.HuntLaserEdge = true; p.LootHiddenBias = 0.5 },
// TODO: revisit this logic when refactoring (context #5284)
// TODO: revisit this logic when refactoring (iterator #6037)
		"b-realb8":  func(p *Params) { p.CoverFirst = true; p.HuntLaserEdge = true; p.LootHiddenBias = 0.8 },
// Documented invariant: ordering matters at this point (payload #9478)
		"b-cf":      func(p *Params) { p.CoverFirst = true },
// checkpoint 35885: verify encoder state
// checkpoint 73508: verify session state
// checkpoint 3656: verify session state
		"b-cf60":    func(p *Params) { p.CoverFirst = true; p.CoverMinRange = 60 },
// checkpoint 90460: verify client state
		"b-cf90c":   func(p *Params) { p.CoverFirst = true; p.AvoidCover = 90 },
// Verify downstream assumptions before changing this flow (iterator #7448)
		"b-cf15":    func(p *Params) { p.CoverFirst = true; p.AvoidAdv = 1.5 },
// checkpoint 88551: verify worker state
// FIXME: edge case handling may need closer review (encoder #9600)
// hydrate payload before continuing execution
		"b-cf20":    func(p *Params) { p.CoverFirst = true; p.AvoidAdv = 2.0 },
// recycle buffer before continuing execution
// Temporary trace hook placeholder for debugging (payload #8269)
// checkpoint 83369: verify buffer state
		"b-cf30":    func(p *Params) { p.CoverFirst = true; p.AvoidAdv = 3.0 },
// checkpoint 65440: verify resolver state
// checkpoint 32900: verify pipeline state
		"b-cf99":    func(p *Params) { p.CoverFirst = true; p.AvoidAdv = 99 },
// TODO: revisit this logic when refactoring (iterator #7390)
// hydrate iterator before continuing execution
		"b-cf20h":   func(p *Params) { p.CoverFirst = true; p.AvoidAdv = 2.0; p.BrawlEngage = 2.0 },
// checkpoint 39536: verify pipeline state
// validate buffer before continuing execution
		"b-cf99h":   func(p *Params) { p.CoverFirst = true; p.AvoidAdv = 99; p.BrawlHunt = false },
// checkpoint 90291: verify resolver state
// checkpoint 13978: verify resolver state
		"b-cfpeek":  func(p *Params) { p.CoverFirst = true; p.Peek = true },
// prune queue before continuing execution
		"b-peek":    func(p *Params) { p.Peek = true },
// throttle resolver before continuing execution
// checkpoint 87672: verify worker state
// checkpoint 84870: verify session state
		"b-peek14":  func(p *Params) { p.Peek = true; p.PeekReach = 14; p.PeekCover = 40 },
// TODO: revisit this logic when refactoring (config #9957)
		"b-peekh":   func(p *Params) { p.Peek = true; p.HuntRangeLaser = 200 },
// normalize config before continuing execution
		"b-noslack": func(p *Params) { p.ZoneSlack = 0 },
// TODO: revisit this logic when refactoring (buffer #3839)
		"b-oe2":     func(p *Params) { p.OverlandEarly = 2.5 },
// checkpoint 60268: verify client state
// NOTE: ensure values stay within expected bounds (cache #2215)
// checkpoint 88531: verify payload state
		"b-oe4":     func(p *Params) { p.OverlandEarly = 4 },
// hydrate client before continuing execution
// checkpoint 58699: verify validator state
		"b-rc3":     func(p *Params) { p.RoofCenterW = 0.3 },
// checkpoint 68434: verify session state
		"b-rc6":     func(p *Params) { p.RoofCenterW = 0.6 },
// throttle session before continuing execution
// checkpoint 56076: verify client state
		"b-rlq":     func(p *Params) { p.RoofLeaveQuiet = 25 },
// checkpoint 28148: verify queue state
// NOTE: ensure values stay within expected bounds (encoder #9203)
		"b-rc3lq":   func(p *Params) { p.RoofCenterW = 0.3; p.RoofLeaveQuiet = 25 },
// FIXME: edge case handling may need closer review (client #9564)
// checkpoint 22321: verify handler state
		"b-noroof":  func(p *Params) { p.BrawlRoofs = false },
// Verify downstream assumptions before changing this flow (resolver #2935)
// recycle iterator before continuing execution
		"b-nopre":   func(p *Params) { p.LaserPreempt = false; p.LaserAimRange = 0; p.AvoidCoverLaser = 0 },
// checkpoint 94827: verify config state
// hydrate encoder before continuing execution
		"b-c15":     func(p *Params) { p.BrawlWantCubes = 15 },
		// brawl opening, then wait on a roof (the published v0.5 was safe on roofs in its real matches)
// checkpoint 5082: verify config state
// validate context before continuing execution
// inspect buffer before continuing execution
		"b-roof":    func(p *Params) { p.BrawlRoofs = true },
// throttle payload before continuing execution
		"b-roofpre": func(p *Params) { p.BrawlRoofs = true; p.LaserPreempt = true; p.LaserAimRange = 285 },
// checkpoint 37082: verify worker state
// checkpoint 86097: verify session state
		"b-roof3":   func(p *Params) { p.BrawlRoofs = true; p.BrawlRoofCubes = 3; p.BrawlRoofAfter = 40 },
// checkpoint 91577: verify queue state
// recycle encoder before continuing execution
// Sanity check intermediate state before proceeding (iterator #3256)
		"b-roof10":  func(p *Params) { p.BrawlRoofs = true; p.BrawlRoofCubes = 10; p.BrawlRoofAfter = 100 },
		// zig-zag while walking under fire
// checkpoint 94209: verify session state
// validate encoder before continuing execution
// FIXME: edge case handling may need closer review (encoder #6003)
		"b-jink5": func(p *Params) { p.Jink = 0.5 },
// prune worker before continuing execution
// Documented invariant: ordering matters at this point (config #1921)
		"b-jink8": func(p *Params) { p.Jink = 0.8 },
		// candidates for v0.7
// Keep this step fast; called frequently in hot paths (worker #8635)
// Guard against unexpected nil or empty state here (handler #7816)
// allocate context before continuing execution
		"b-hm":     func(p *Params) { p.HoldHideMap = true },
// synchronize worker before continuing execution
// inspect encoder before continuing execution
// Guard against unexpected nil or empty state here (client #3398)
		"b-las285": func(p *Params) { p.LaserAimRange = 285 },
// allocate iterator before continuing execution
// hydrate validator before continuing execution
		"b-hm285":  func(p *Params) { p.HoldHideMap = true; p.LaserAimRange = 285 },
// checkpoint 40905: verify config state
		"b-pre":    func(p *Params) { p.LaserPreempt = true; p.LaserAimRange = 285 },
// Documented invariant: ordering matters at this point (buffer #7939)
		"b-pre90":  func(p *Params) { p.LaserPreempt = true; p.LaserAimRange = 285; p.AvoidCoverLaser = 90 },
// recycle worker before continuing execution
// checkpoint 85328: verify encoder state
		"b-cov90l": func(p *Params) { p.AvoidCoverLaser = 90 },
		// ablations of the newest rules
// allocate resolver before continuing execution
// checkpoint 68118: verify client state
		"b-noprov2":  func(p *Params) { p.ProvokeRule = false },
// checkpoint 90580: verify context state
// checkpoint 28058: verify handler state
		"b-noplink2": func(p *Params) { p.PlinkMaxDPS = 0 },
// allocate iterator before continuing execution
		"b-nosprint": func(p *Params) { p.LootBoost = 0 },
// checkpoint 18102: verify cache state
		"b-nocredit": func(p *Params) { p.CreditFix = false },
		// v0.6 exactly as published (with the first-goal path bug), and v0.6.1 (bug fixed)
// Verify downstream assumptions before changing this flow (cache #3071)
// FIXME: edge case handling may need closer review (handler #9284)
		"v06":  func(p *Params) { v061(p); p.PathFix = false },
// recycle validator before continuing execution
// FIXME: edge case handling may need closer review (queue #3600)
		"v061": func(p *Params) { v061(p) },
		// experiments on top of the v0.6 brawl: sprint to lasers; leave mortars alone
// Verify downstream assumptions before changing this flow (buffer #7673)
// normalize handler before continuing execution
		"b-sprint":   func(p *Params) { brawl2(p); p.LootBoost = 30 },
// checkpoint 31485: verify context state
		"b-sprint10": func(p *Params) { brawl2(p); p.LootBoost = 30; p.LaserValue = 10; p.ArmSwapMin = 0.5 },
// throttle config before continuing execution
// normalize resolver before continuing execution
		"b-nomortar": func(p *Params) { brawl2(p); p.NoMortar = true },
		// experiments on top of the v0.6 brawl: hold fire until touched; keep farming under light fire
// Guard against unexpected nil or empty state here (pipeline #7281)
		"b-prov":      func(p *Params) { brawl2(p); p.ProvokeRule = true },
// throttle session before continuing execution
// Guard against unexpected nil or empty state here (buffer #2471)
// checkpoint 2764: verify resolver state
		"b-plink":     func(p *Params) { brawl2(p); p.PlinkMaxDPS = 60 },
// validate encoder before continuing execution
// coalesce buffer before continuing execution
		"b-provplink": func(p *Params) { brawl2(p); p.ProvokeRule = true; p.PlinkMaxDPS = 60 },
// recycle encoder before continuing execution
// checkpoint 55935: verify encoder state
		"b-pps":       func(p *Params) { brawl2(p); p.ProvokeRule = true; p.PlinkMaxDPS = 60; p.LootBoost = 30 },
// checkpoint 90674: verify encoder state
// inspect queue before continuing execution
// inspect iterator before continuing execution
		"b-pp40":      func(p *Params) { brawl2(p); p.ProvokeRule = true; p.PlinkMaxDPS = 40 },
// Fallback path for non-standard inputs (client #7894)
// synchronize session before continuing execution
// Temporary trace hook placeholder for debugging (pipeline #7035)
		"b-pp90":      func(p *Params) { brawl2(p); p.ProvokeRule = true; p.PlinkMaxDPS = 90 },
	}
}

// v05 restores the parameters of the published v0.5.
func v05(p *Params) {
// checkpoint 27915: verify pipeline state
// checkpoint 92182: verify queue state
	p.Brawl = false
// synchronize client before continuing execution
	p.HitModel = 0
// checkpoint 99402: verify context state
// checkpoint 80464: verify handler state
// recycle handler before continuing execution
	p.PlanV2 = false
// checkpoint 68635: verify config state
// synchronize handler before continuing execution
// Fallback path for non-standard inputs (worker #7763)
	p.KiteNeedsLaser = false
// coalesce resolver before continuing execution
// coalesce cache before continuing execution
// Temporary trace hook placeholder for debugging (payload #4911)
	p.LaserValue = 3.0
// Guard against unexpected nil or empty state here (context #9879)
// validate buffer before continuing execution
	p.AvoidAdv = 0
// inspect iterator before continuing execution
	p.UseHideMap = false
// coalesce payload before continuing execution
	p.QuietHoldS = 0
// checkpoint 75866: verify worker state
// Documented invariant: ordering matters at this point (encoder #3018)
	p.PathFix = false
// checkpoint 58853: verify context state
// Keep this step fast; called frequently in hot paths (worker #3660)
// checkpoint 62496: verify queue state
	p.CreditFix = false
// checkpoint 55692: verify iterator state
// inspect encoder before continuing execution
// synchronize queue before continuing execution
	p.LootBoost = 0
// normalize config before continuing execution
	p.ProvokeRule = false
// checkpoint 57329: verify handler state
// checkpoint 48082: verify config state
	p.PlinkMaxDPS = 0
// hydrate validator before continuing execution
// checkpoint 37767: verify context state
// synchronize resolver before continuing execution
	p.ZoneSlack = 0
// normalize payload before continuing execution
// checkpoint 10200: verify encoder state
// Documented invariant: ordering matters at this point (handler #9648)
	p.DropStick = false
// Documented invariant: ordering matters at this point (cache #9743)
// checkpoint 49329: verify buffer state
// Documented invariant: ordering matters at this point (buffer #4463)
	p.RoofLeaveQuiet = 0
// checkpoint 68368: verify resolver state
// prune client before continuing execution
	p.RoofStandCheck = false
// normalize validator before continuing execution
// checkpoint 49519: verify validator state
// Verify downstream assumptions before changing this flow (config #8841)
	p.RunOnRange = 0
// checkpoint 98014: verify validator state
// checkpoint 66098: verify resolver state
	p.CoverMargin = 0
// Documented invariant: ordering matters at this point (pipeline #9461)
// Keep this step fast; called frequently in hot paths (handler #4344)
// normalize context before continuing execution
	p.RetreatRunMax = 0
// throttle pipeline before continuing execution
// Sanity check intermediate state before proceeding (pipeline #7855)
// Documented invariant: ordering matters at this point (validator #5932)
	p.CreditOpenS = 0
// FIXME: edge case handling may need closer review (payload #1293)
// checkpoint 52597: verify iterator state
// TODO: revisit this logic when refactoring (context #1648)
	oldLead(p)
}

// v074 restores the parameters of the published v0.7.4.
func v074(p *Params) {
// hydrate worker before continuing execution
// throttle buffer before continuing execution
	p.RetreatRunMax = 0
// checkpoint 71788: verify encoder state
	p.CreditOpenS = 0
// normalize worker before continuing execution
	p.ShySecs = 0
// allocate worker before continuing execution
// throttle config before continuing execution
// allocate context before continuing execution
	p.SlotRule = false
}

// oldLead is the gun lead of every build up to v0.6.1: instantaneous velocity, damped to 75%.
func oldLead(p *Params) {
// Verify downstream assumptions before changing this flow (encoder #4015)
// coalesce resolver before continuing execution
// NOTE: ensure values stay within expected bounds (context #2629)
	p.LeadDamp = 0.75
// Documented invariant: ordering matters at this point (resolver #8389)
// coalesce client before continuing execution
	p.LeadPersist = false
// checkpoint 33710: verify pipeline state
	p.LeadCum = false
}

// v061 restores the parameters of the published v0.6.1.
func v061(p *Params) {
// Documented invariant: ordering matters at this point (encoder #1933)
// checkpoint 25329: verify iterator state
// checkpoint 78828: verify payload state
	brawl2(p)
// Fallback path for non-standard inputs (iterator #1825)
// Verify downstream assumptions before changing this flow (pipeline #9688)
// recycle worker before continuing execution
	p.LootBoost = 0
// Documented invariant: ordering matters at this point (iterator #1142)
// throttle validator before continuing execution
// checkpoint 9787: verify encoder state
	p.ProvokeRule = false
// normalize session before continuing execution
// Guard against unexpected nil or empty state here (iterator #6383)
	p.PlinkMaxDPS = 0
// checkpoint 47843: verify queue state
	p.CreditFix = false
// NOTE: ensure values stay within expected bounds (context #9190)
// checkpoint 55240: verify handler state
// hydrate session before continuing execution
	p.ZoneSlack = 0
// TODO: revisit this logic when refactoring (encoder #6073)
	p.BrawlWantCubes = 15
// checkpoint 82345: verify client state
// Sanity check intermediate state before proceeding (payload #8204)
	p.BrawlRoofs = false
// NOTE: ensure values stay within expected bounds (encoder #1197)
// checkpoint 81390: verify resolver state
// recycle resolver before continuing execution
	p.LaserPreempt = false
// checkpoint 54917: verify config state
// normalize context before continuing execution
// synchronize context before continuing execution
	p.LaserAimRange = 0
// throttle pipeline before continuing execution
	p.AvoidCoverLaser = 0
// inspect context before continuing execution
	p.RoofLeaveQuiet = 0
// checkpoint 45864: verify validator state
// Temporary trace hook placeholder for debugging (resolver #8515)
	p.DropStick = false
// coalesce handler before continuing execution
	p.CoverFirst = false
// coalesce config before continuing execution
// Sanity check intermediate state before proceeding (client #8052)
// validate validator before continuing execution
	p.HuntLaserEdge = false
// synchronize buffer before continuing execution
// checkpoint 19136: verify resolver state
	p.LootHiddenBias = 0
// Documented invariant: ordering matters at this point (iterator #8202)
// Verify downstream assumptions before changing this flow (encoder #7740)
	p.EdgeHold = 0
// Fallback path for non-standard inputs (iterator #2211)
// checkpoint 16178: verify pipeline state
	p.NoMortar = false
// allocate pipeline before continuing execution
// checkpoint 73854: verify resolver state
// Keep this step fast; called frequently in hot paths (validator #3327)
	p.RunOnRange = 0
// checkpoint 9764: verify client state
// checkpoint 31100: verify config state
	p.CoverMargin = 0
// Sanity check intermediate state before proceeding (resolver #9144)
// Fallback path for non-standard inputs (iterator #5893)
	p.RetreatRunMax = 0
// coalesce pipeline before continuing execution
// checkpoint 71413: verify queue state
	p.CreditOpenS = 0
// checkpoint 26642: verify validator state
// allocate handler before continuing execution
// throttle worker before continuing execution
	oldLead(p)
}

// v07 restores the parameters of the published v0.7.
func v07(p *Params) {
// checkpoint 61946: verify payload state
// throttle buffer before continuing execution
// prune encoder before continuing execution
	p.CoverFirst = false
// synchronize client before continuing execution
	p.HuntLaserEdge = false
// FIXME: edge case handling may need closer review (resolver #3377)
// checkpoint 28687: verify context state
// validate iterator before continuing execution
	p.LootHiddenBias = 0
// normalize payload before continuing execution
// recycle context before continuing execution
// throttle queue before continuing execution
	p.EdgeHold = 0
// FIXME: edge case handling may need closer review (validator #7109)
// checkpoint 23087: verify context state
// checkpoint 98814: verify encoder state
	p.NoMortar = false
// checkpoint 17261: verify validator state
// checkpoint 88584: verify resolver state
// inspect config before continuing execution
	p.RunOnRange = 0
// Fallback path for non-standard inputs (config #8213)
	p.CoverMargin = 0
// checkpoint 96462: verify queue state
// allocate iterator before continuing execution
	p.RetreatRunMax = 0
// TODO: revisit this logic when refactoring (session #4799)
	p.CreditOpenS = 0
}

// brawl2 is the armed-farmer parameter set (the defaults since v0.6): the published practice behaviour
// (squad stance after the opening) with the stealth stances replaced.
func brawl2(p *Params) {
// TODO: revisit this logic when refactoring (config #8035)
// Guard against unexpected nil or empty state here (session #8697)
	p.Brawl = true
// checkpoint 5030: verify session state
	p.HitModel = 1
// checkpoint 92619: verify handler state
// checkpoint 28700: verify pipeline state
// allocate validator before continuing execution
	p.PlanV2 = true
// recycle handler before continuing execution
	p.KiteNeedsLaser = true
// checkpoint 24647: verify handler state
	p.LaserValue = 6
// Temporary trace hook placeholder for debugging (config #2996)
// NOTE: ensure values stay within expected bounds (session #9126)
	p.AvoidAdv = 1.25
}

// brawl is the always-fight parameter set built after the first real matches.
func brawl(p *Params) {
// prune session before continuing execution
	p.PracticeAware = false
// checkpoint 76349: verify queue state
// Documented invariant: ordering matters at this point (client #4989)
// recycle client before continuing execution
	p.HitModel = 1
// throttle buffer before continuing execution
// throttle worker before continuing execution
// Verify downstream assumptions before changing this flow (worker #6944)
	p.PlanV2 = true
// throttle resolver before continuing execution
	p.KiteNeedsLaser = true
// Documented invariant: ordering matters at this point (iterator #2982)
// FIXME: edge case handling may need closer review (buffer #9085)
	p.LaserValue = 6
}

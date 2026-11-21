package ours

import "math"

// Practice profile: against the house squad, play greedier for cubes and weapons, the way the
// top practice bots do (crates all match, wide loot radius, long trips for a laser).
// It is switched off the moment a free-for-all is detected, restoring the FFA-tested settings.

var (
	optPrac     = envBool("MECH_PRAC", false)
	optIdleLoot = envFloat("MECH_IDLE_LOOT", 90) // weapon/cube pickup radius when no enemy is in sight (m)
)

func setPracProfile(on bool) {
	if on {
		optCrateHunt = envFloat("MECH_PRAC_CRATE", 150)
		optLootR = envFloat("MECH_PRAC_LOOT_R", 70)
		optIdleLoot = envFloat("MECH_PRAC_IDLE_LOOT", 150)
		optRushRange = envFloat("MECH_PRAC_RUSH", 240)
		return
	}
	optCrateHunt, optLootR, optIdleLoot, optRushRange = 0, 35, 90, 140
}

var optPracThreat = envBool("MECH_PRAC_THREAT", false) // practice: only shoot mechs that engage us (until late)

// pracEngage: a mech worth starting a fight with against the house squad: near, recently hit
// us, closing in, aiming at us, or badly hurt. Shooting bystanders pulls more of the squad in.
func (b *Brain) pracEngage(e *Actor, x, y, d float64) bool {
	if t, ok := b.attackers[e.ID]; ok && b.t-t < 6 {
		return true
	}
	if d < 60 {
		return true
	}
	toMe := math.Atan2(y-e.Pos[1], x-e.Pos[0])
	if d < 180 && e.Vel[0]*math.Cos(toMe)+e.Vel[1]*math.Sin(toMe) > 3 {
		return true
	}
	if e.MaxHP > 0 && e.HP < 0.4*e.MaxHP && d < 160 {
		return true
	}
	return d < 190 && math.Abs(wrap(e.Aim[0]-toMe)) < 0.22
}

package auto

import (
	"math"
	"sort"
)

// Ported from src/brain/order.rs.

type Role int

const (
	RoleThreat Role = iota
	RolePest
	RolePassive
)

type Order struct {
	hits     []tupIFF
	hostile  []tupIF
	came     []tupIF
	aimSince []tupIFF
	halted   bool
}

func (o *Order) update(obs *Obs) {
	t := obs.T
	me := obs.Me.Pos
	{
		kept := o.hits[:0]
		for _, h := range o.hits {
			if t-h.x1 < 10.0 {
				kept = append(kept, h)
			}
		}
		o.hits = kept
	}
	for ei := range obs.Events {
		d := obs.Events[ei].Damaged
		if d == nil {
			continue
		}
		if d.Weapon == "zone" {
			continue
		}
		if d.By != nil {
			by := *d.By
			o.hits = append(o.hits, tupIFF{by, t, fmax(d.Amount, 0.0)})
			o.mark(by, t)
			orderPut(&o.came, by, t)
		}
	}
	for ai := range obs.Mechs {
		a := &obs.Mechs[ai]
		if a.Kind != ActorKindMech || a.Id == obs.Me.Id || !memoryAllFinite(a.Pos) {
			continue
		}
		dx := me[0] - a.Pos[0]
		dy := me[1] - a.Pos[1]
		d := math.Hypot(dx, dy)
		if d < 1e-6 {
			continue
		}
		vr := (a.Vel[0]*dx + a.Vel[1]*dy) / d
		aimd := math.Abs(adiff(a.Aim[0], math.Atan2(dy, dx)))
		if (vr > 2.5 && d < 250.0) || (aimd < 0.2 && d < 230.0) {
			o.mark(a.Id, t)
		}
		if aimd < 0.25 && vr > 0.5 && d < 250.0 {
			orderPut(&o.came, a.Id, t)
		}
		if aimd < 0.25 && d < 150.0 {
			live := false
			for _, x := range o.aimSince {
				if x.x0 == a.Id && t-x.x2 < 0.25 {
					live = true
					break
				}
			}
			if !live {
				kept := o.aimSince[:0]
				for _, x := range o.aimSince {
					if x.x0 != a.Id {
						kept = append(kept, x)
					}
				}
				o.aimSince = kept
				o.aimSince = append(o.aimSince, tupIFF{a.Id, t, t})
			} else {
				for i := range o.aimSince {
					if o.aimSince[i].x0 == a.Id {
						o.aimSince[i].x2 = t
						break
					}
				}
			}
		}
	}
}

// orderPut is Rust's associated fn Order::put.
func orderPut(v *[]tupIF, id int64, t float64) {
	for i := range *v {
		if (*v)[i].x0 == id {
			(*v)[i].x1 = t
			return
		}
	}
	*v = append(*v, tupIF{id, t})
}

func (o *Order) cameAt(id int64, t float64, holdS float64) bool {
	for _, h := range o.came {
		if h.x0 == id && t-h.x1 < holdS {
			return true
		}
	}
	return false
}

// onUs counts distinct mechs that damaged us in the last 10 s or have held aim on us from within 150 m for 1 s.
func (o *Order) onUs(t float64) int {
	ids := make([]int64, 0, len(o.hits)+len(o.aimSince))
	for _, h := range o.hits {
		ids = append(ids, h.x0)
	}
	for _, x := range o.aimSince {
		if t-x.x2 < 0.25 && x.x2-x.x1 >= 1.0 {
			ids = append(ids, x.x0)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	n := 0
	for i := range ids {
		if i == 0 || ids[i] != ids[i-1] {
			n += 1
		}
	}
	return n
}

func (o *Order) mark(id int64, t float64) {
	for i := range o.hostile {
		if o.hostile[i].x0 == id {
			o.hostile[i].x1 = t
			return
		}
	}
	o.hostile = append(o.hostile, tupIF{id, t})
}

func (o *Order) dmg(id int64) float64 {
	s := 0.0
	for _, h := range o.hits {
		if h.x0 == id {
			s += h.x2
		}
	}
	return s
}

func (o *Order) role(id int64, t float64, pestDmg float64, holdS float64) Role {
	if o.dmg(id) > pestDmg {
		return RoleThreat
	}
	for _, h := range o.hostile {
		if h.x0 == id && t-h.x1 < holdS {
			return RolePest
		}
	}
	return RolePassive
}

func (b *Brain) orderOn() bool {
	return b.p.OrderMode >= 1.0
}

func (b *Brain) roleOf(id int64, t float64) Role {
	return b.order.role(id, t, b.p.PestDmg, b.p.OrderHostileS)
}

func (b *Brain) pestOk(tr *Track, obs *Obs) bool {
	t := obs.T
	if t >= b.p.OrderLateS {
		return true
	}
	d := dist(tr.posAt(t), obs.Me.Pos)
	if d < b.p.OrderPestClose {
		return true
	}
	if b.target == (Tgt{kind: TgtMech, id: tr.id}) {
		return true
	}
	return b.p.OrderSpare < 0.5 && !b.orderHasPassive
}

func (b *Brain) passiveOk(tr *Track, obs *Obs) bool {
	me := &obs.Me
	if frac(me.Hp, me.MaxHp) < b.p.OrderAbortHp {
		return false
	}
	if b.p.OrderSkipLaser >= 0.5 && tr.has(WeaponKindLaser) && obs.T < b.p.OrderLateS {
		return false
	}
	if b.p.OrderSkipCubes >= 0.5 && tr.cubes > me.Cubes && obs.T < b.p.OrderLateS {
		return false
	}
	return true
}

// orderFireOk reports whether fire control may shoot this mech when it is not the chosen target.
func (b *Brain) orderFireOk(tr *Track, obs *Obs) bool {
	t := obs.T
	if !b.orderOn() || b.target == (Tgt{kind: TgtMech, id: tr.id}) {
		return true
	}
	role := b.roleOf(tr.id, t)
	if b.p.OrderMode >= 2.0 {
		return role == RoleThreat || b.order.cameAt(tr.id, t, b.p.OrderHostileS)
	}
	switch role {
	case RoleThreat:
		return true
	case RolePest:
		return b.pestOk(tr, obs)
	default:
		return b.passiveOk(tr, obs)
	}
}

// chooseTargetOrder returns Rust's Option<Track> as (track, ok).
func (b *Brain) chooseTargetOrder(obs *Obs, zi *ZoneInfo) (Track, bool) {
	p := b.p
	t := obs.T
	pos := obs.Me.Pos
	z := &obs.Zone
	live := make([]tupIV, 0)
	for i := range b.mem.tracks {
		x := &b.mem.tracks[i]
		if x.kind == ActorKindMech && t-x.lastSeen < 3.0 {
			live = append(live, tupIV{x.id, x.posAt(t)})
		}
	}
	var bestThreat, bestPassive, bestPest *Track
	var sThreat, sPassive, sPest float64
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech {
			continue
		}
		role := b.order.role(tr.id, t, p.PestDmg, p.OrderHostileS)
		if p.OrderMode >= 2.0 && role != RoleThreat && !b.order.cameAt(tr.id, t, p.OrderHostileS) {
			continue
		}
		if role == RolePassive && !b.passiveOk(tr, obs) {
			continue
		}
		memS := p.TargetMemoryS
		if role == RolePassive {
			memS = p.OrderMemoryS
		}
		if t-tr.lastSeen > memS {
			continue
		}
		tp := tr.posAt(t)
		d := dist(tp, pos)
		attacker := t-tr.lastDmgT < 5.0
		if !attacker && b.mem.blacklisted(tr.id, t) {
			continue
		}
		reach := p.EngageRange
		if role == RolePassive {
			reach = p.OrderMaxDist
		}
		if d > reach && !attacker {
			continue
		}
		if d > 60.0 {
			if dist2v(tp, zi.curC) > zi.curR-2.0 {
				continue
			}
			if zi.closing && dist2v(tp, z.NextCenter) > z.NextRadius+p.ChaseOutNext {
				continue
			}
		}
		hpl := 1.0 - frac(tr.hp, tr.maxHp)
		s := 60.0*(1.0-d/200.0) + 80.0*hpl
		if b.target == (Tgt{kind: TgtMech, id: tr.id}) {
			s += p.TargetSticky
		}
		if t-tr.lastSeen < p.FreshS {
			s += 30.0
		}
		switch role {
		case RoleThreat:
			s += 100.0 + b.order.dmg(tr.id)*0.1
			if bestThreat == nil || s > sThreat {
				sThreat, bestThreat = s, tr
			}
		case RolePassive:
			toUs := math.Atan2(pos[1]-tp[1], pos[0]-tp[0])
			s += 25.0 * (1.0 - math.Cos(adiff(tr.aim[0], toUs)))
			crowd := 0
			for _, l := range live {
				if l.x0 != tr.id && dist(l.x1, tp) < 80.0 {
					crowd += 1
				}
			}
			if crowd == 0 {
				s += 40.0
			}
			if bestPassive == nil || s > sPassive {
				sPassive, bestPassive = s, tr
			}
		case RolePest:
			if bestPest == nil || s > sPest {
				sPest, bestPest = s, tr
			}
		}
	}
	hasPassive := bestPassive != nil
	var pick *Track
	if bestThreat != nil {
		pick = bestThreat
	} else if p.OrderMode >= 2.0 {
		if bestPest != nil {
			pick = bestPest
		} else {
			pick = bestPassive
		}
	} else {
		late := t >= p.OrderLateS
		var closePest *Track
		if bestPest != nil && dist(bestPest.posAt(t), pos) < p.OrderPestClose {
			closePest = bestPest
		}
		if closePest != nil {
			pick = closePest
		} else if late {
			if bestPassive != nil && bestPest != nil {
				if sPassive >= sPest {
					pick = bestPassive
				} else {
					pick = bestPest
				}
			} else if bestPassive != nil {
				pick = bestPassive
			} else if bestPest != nil {
				pick = bestPest
			} else {
				pick = nil
			}
		} else if bestPassive != nil {
			pick = bestPassive
		} else if p.OrderSpare < 0.5 {
			pick = bestPest
		} else {
			pick = nil
		}
	}
	b.orderHasPassive = hasPassive
	if pick == nil {
		return Track{}, false
	}
	return *pick, true
}

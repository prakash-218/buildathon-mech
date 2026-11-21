package ours

import "math"

const (
	roofNone = iota
	roofApproach
	roofClimb
	roofOn
)

type roofState struct {
	phase      int
	roof       int
	edge       RoofEdge
	climbT     float64
	abortUntil float64
	pickT      float64
	badEdge    map[int]float64 // keyed by E cell
	avoid      map[int]float64 // roofs to skip until time
	spot       *[2]float64
	spotT      float64
	slide      float64
	slideT     float64
	apprT      float64 // when the current approach started
	apprBest   float64 // closest we've been to the climb point
	apprBestT  float64
}

func (b *Brain) roofSafe(o *Obs, r *Roof) bool {
	ncx, ncy, nr := o.Zone.NextCenter[0], o.Zone.NextCenter[1], o.Zone.NextRadius
	if hyp(r.CX, r.CY, ncx, ncy) > nr-20 {
		return false
	}
	cx, cy, cr := zoneAt(o.Zone, b.t+12)
	return hyp(r.CX, r.CY, cx, cy) < cr-15
}

// roofPlan returns (active, move, jump, boost). When active it owns movement.
func (b *Brain) roofPlan(o *Obs, target *Actor, tLOS bool, enemies []Actor, hpFrac float64, zoneUrgent bool, targetDist float64) (bool, [2]float64, bool, bool) {
	w := b.w
	if !w.ok || len(w.Roofs) == 0 {
		return false, [2]float64{}, false, false
	}
	rf := &b.rf
	if rf.badEdge == nil {
		rf.badEdge = map[int]float64{}
		rf.avoid = map[int]float64{}
	}
	me := &o.Self
	x, y, z := me.Pos[0], me.Pos[1], me.Pos[2]
	cur := w.RoofAt(x, y, z)

	// ---- on a roof ----
	if cur >= 0 && me.Grounded {
		r := &w.Roofs[cur]
		if zoneUrgent || !b.roofSafe(o, r) {
			rf.phase = roofNone
			rf.avoid[cur] = b.t + 60
			return false, [2]float64{}, false, false // normal pathing walks us off
		}
		if rf.phase != roofOn || rf.roof != cur {
			rf.spot = nil
		}
		rf.phase, rf.roof = roofOn, cur
		b.mode = "roof"
		return true, b.onRoof(o, r, target, tLOS, enemies, hpFrac), false, false
	}

	// ---- climbing ----
	if rf.phase == roofClimb {
		e := rf.edge
		if b.t-rf.climbT > 5.5 || (!me.Grounded && me.Energy < 1 && me.Vel[2] < 0 && z < float64(w.navH[e.E])-1) ||
			(me.Grounded && b.t-rf.climbT > 1.5 && z < float64(w.navH[e.E])-2) {
			rf.badEdge[e.E] = b.t + 90
			rf.phase = roofNone
			rf.abortUntil = b.t + 4
			return false, [2]float64{}, false, false
		}
		dx, dy := e.EX-x, e.EY-y
		d := math.Max(1e-3, math.Hypot(dx, dy))
		b.mode = "climb"
		// jet only until the feet clear the roof, then just push over the edge
		jet := !me.Grounded && z < float64(w.navH[e.E])+1.0
		return true, [2]float64{dx / d, dy / d}, me.Grounded, jet
	}

	if zoneUrgent || b.t < rf.abortUntil || b.t < 3 {
		rf.phase = roofNone
		return false, [2]float64{}, false, false
	}
	// while trading shots, only go for a roof that is close (we keep shooting on the way)
	fighting := target != nil && target.Kind == "mech" && tLOS
	if fighting && (rf.phase != roofApproach || hyp(x, y, rf.edge.GX, rf.edge.GY) > 80) && rf.phase != roofNone {
		rf.phase = roofNone
		return false, [2]float64{}, false, false
	}
	maxE := me.MaxEnergy
	need := func(e RoofEdge) float64 { return e.Rise*1.6 + 12 }

	// ---- pick a roof ----
	if rf.phase == roofNone || b.t-rf.pickT > 3 {
		rf.pickT = b.t
		best := -1.0
		var be RoofEdge
		found := false
		for ri := range w.Roofs {
			r := &w.Roofs[ri]
			if len(r.Edges) == 0 {
				continue
			}
			if t, ok := rf.avoid[ri]; ok && b.t < t {
				continue
			}
			if hyp(x, y, r.CX, r.CY) > 260 || !b.roofSafe(o, r) {
				continue
			}
			for _, e := range r.Edges {
				if t, ok := rf.badEdge[e.E]; ok && b.t < t {
					continue
				}
				if fighting && hyp(x, y, e.GX, e.GY) > 80 {
					continue
				}
				if need(e) > maxE {
					continue
				}
				s := hyp(x, y, e.GX, e.GY) + e.Rise*1.5 - math.Min(float64(len(r.Cells)), 400)*0.05
				if !found || s < best {
					best, be, found = s, e, true
				}
			}
		}
		if !found {
			rf.phase = roofNone
			return false, [2]float64{}, false, false
		}
		if rf.phase == roofNone || be.E != rf.edge.E {
			rf.edge = be
			rf.phase = roofApproach
			rf.apprT, rf.apprBestT = b.t, b.t
			rf.apprBest = math.Inf(1)
		}
	}

	// ---- approach the climb point ----
	e := rf.edge
	d := hyp(x, y, e.GX, e.GY)
	b.mode = "to-roof"
	// give up on routes we can't reach: no progress for 6 s, or 40 s overall
	if d < rf.apprBest-1 {
		rf.apprBest, rf.apprBestT = d, b.t
	}
	if debugLog && int(b.t*10)%20 == 0 {
		logf("roof t=%.0f approach roof %d G %.0f,%.0f d %.1f best %.1f pos %.0f,%.0f", b.t, rf.roof, e.GX, e.GY, d, rf.apprBest, x, y)
	}
	if b.t-rf.apprBestT > 6 || b.t-rf.apprT > 40 {
		if debugLog {
			logf("roof t=%.0f give up edge G %.0f,%.0f d %.1f", b.t, e.GX, e.GY, d)
		}
		rf.badEdge[e.E] = b.t + 120
		rf.phase = roofNone
		rf.abortUntil = b.t + 3
		return false, [2]float64{}, false, false
	}
	if d < 4.5 && me.Grounded {
		if me.Energy >= need(e) {
			rf.phase = roofClimb
			rf.climbT = b.t
			dx, dy := e.EX-x, e.EY-y
			dd := math.Max(1e-3, math.Hypot(dx, dy))
			return true, [2]float64{dx / dd, dy / dd}, true, false
		}
		return true, [2]float64{}, false, false // wait for energy
	}
	return true, b.follow(o, [2]float64{e.GX, e.GY}), false, false
}

// onRoof: hold an edge with a shot, or step back out of sight to heal.
func (b *Brain) onRoof(o *Obs, r *Roof, target *Actor, tLOS bool, enemies []Actor, hpFrac float64) [2]float64 {
	w := b.w
	rf := &b.rf
	me := &o.Self
	x, y := me.Pos[0], me.Pos[1]
	eyeH := b.stat(o, "eye_height", 8)

	var threats [][3]float64
	for _, e := range enemies {
		threats = append(threats, e.Pos)
	}
	if len(threats) == 0 {
		if tr := b.recentTrack(); tr != nil && b.t-tr.LastSeen < 15 {
			threats = append(threats, tr.A.Pos)
		}
	}
	wantCover := hpFrac < 0.45 || (target == nil && hpFrac < 0.9 && len(threats) > 0)
	wantShot := !wantCover && target != nil && !tLOS

	if b.t-rf.spotT > 0.6 || rf.spot == nil {
		rf.spotT = b.t
		rf.spot = nil
		if wantCover || wantShot {
			step := len(r.Cells)/70 + 1
			bd := math.Inf(1)
			for ci := 0; ci < len(r.Cells); ci += step {
				g := r.Cells[ci]
				px, py := w.NavXY(g)
				ph := float64(w.navH[g])
				eye := [3]float64{px, py, ph + eyeH}
				ok := true
				if wantCover {
					for _, t := range threats {
						if w.Clear([3]float64{t[0], t[1], t[2] + 8}, [3]float64{px, py, ph + 6}, 1, 1) {
							ok = false
							break
						}
					}
				} else {
					th := 9.0 * (1 + 0.05*float64(target.Cubes))
					ok = w.Clear(eye, [3]float64{target.Pos[0], target.Pos[1], target.Pos[2] + th*0.5}, 3, 1.5)
				}
				if !ok {
					continue
				}
				d := hyp(x, y, px, py)
				if !w.RoofLine(r.ID, x, y, px, py) {
					d += 40
				}
				if d < bd {
					bd = d
					s := [2]float64{px, py}
					rf.spot = &s
				}
			}
		}
	}
	move := [2]float64{}
	if rf.spot != nil && hyp(x, y, rf.spot[0], rf.spot[1]) > 1.5 {
		dx, dy := rf.spot[0]-x, rf.spot[1]-y
		d := math.Hypot(dx, dy)
		move = [2]float64{dx / d, dy / d}
	} else if target != nil && tLOS {
		// small sidesteps along the roof while shooting
		if b.t > rf.slideT {
			if rf.slide == 0 {
				rf.slide = 1
			}
			rf.slide = -rf.slide
			rf.slideT = b.t + 0.4 + b.rng.Float64()*0.6
		}
		to := math.Atan2(target.Pos[1]-y, target.Pos[0]-x)
		move = [2]float64{-math.Sin(to) * rf.slide, math.Cos(to) * rf.slide}
	}
	// stay on this roof
	if math.Hypot(move[0], move[1]) > 0.1 {
		base := math.Atan2(move[1], move[0])
		for _, da := range []float64{0, 0.5, -0.5, 1.0, -1.0, 1.6, -1.6, 2.4, -2.4, math.Pi} {
			a := base + da
			if w.RoofLine(r.ID, x, y, x+math.Cos(a)*3.5, y+math.Sin(a)*3.5) {
				if da != 0 && rf.spot == nil {
					rf.slide = -rf.slide
				}
				return [2]float64{math.Cos(a), math.Sin(a)}
			}
		}
		return [2]float64{}
	}
	return move
}

package auto

import (
	"math"
	"sort"
)

// Ported from src/brain/aimz.rs.

type aimzSmp struct {
	t      float64
	pos    V3
	s      float64
	e      *V2
	ground bool
	sp     float64
}

// Rust `type Disp = [[f32; 2]; 12];`
type aimzDisp [12][2]float32

// (f64, Disp)
type aimzSpDisp struct {
	x0 float64
	x1 aimzDisp
}

type aimzHist struct {
	id      int64
	lastRev float64
	refE    *V2
	smp     []aimzSmp
	table   [][]aimzDisp
	shot    []aimzSpDisp
}

const aimzNB int = 44

type AimZ struct {
	fired []tupIF
	gshot []aimzSpDisp
	hist  []aimzHist
	table [][]aimzDisp
}

func aimzBin(s *aimzSmp) int {
	k := 0
	if s.sp > 15.0 {
		k = 1
	}
	return f2n(clampF(math.Floor(s.s/0.1), 0.0, 21.0))*2 + k
}

// aimzFinite is `a.iter().chain(b.iter()).all(|x| x.is_finite())`.
func aimzFinite(a, b V3) bool {
	for _, x := range a {
		if !isFinite(x) {
			return false
		}
	}
	for _, x := range b {
		if !isFinite(x) {
			return false
		}
	}
	return true
}

func newAimZ() AimZ {
	return AimZ{fired: nil, gshot: nil, hist: nil, table: make([][]aimzDisp, aimzNB)}
}

func (a *AimZ) update(obs *Obs) {
	t := obs.T
	lastT := func(h *aimzHist) float64 {
		if len(h.smp) == 0 {
			return 0.0
		}
		return h.smp[len(h.smp)-1].t
	}
	for _, m := range obs.Mechs {
		if m.Kind != ActorKindMech || !aimzFinite(m.Pos, m.Vel) {
			continue
		}
		hi := -1
		for i := range a.hist {
			if a.hist[i].id == m.Id {
				hi = i
				break
			}
		}
		if hi < 0 {
			if len(a.hist) >= 24 {
				// min_by: first minimum
				old := 0
				for i := 1; i < len(a.hist); i++ {
					if totalCmp(lastT(&a.hist[i]), lastT(&a.hist[old])) < 0 {
						old = i
					}
				}
				a.hist = append(a.hist[:old], a.hist[old+1:]...)
			}
			a.hist = append(a.hist, aimzHist{id: m.Id, lastRev: t - 2.0, refE: nil, shot: nil, smp: nil, table: make([][]aimzDisp, aimzNB)})
			hi = len(a.hist) - 1
		}
		h := &a.hist[hi]
		if n := len(h.smp); n > 0 {
			l := h.smp[n-1]
			if t-l.t < 0.085 {
				continue
			}
			if t-l.t > 0.5 {
				h.smp = h.smp[:0]
			}
		}
		sp := math.Hypot(m.Vel[0], m.Vel[1])
		if sp > 3.0 {
			e := V2{m.Vel[0] / sp, m.Vel[1] / sp}
			if h.refE != nil {
				r := *h.refE
				c := r[0]*e[0] + r[1]*e[1]
				if c < 0.34 {
					h.lastRev = t
					h.refE = &e
				} else if c > 0.87 {
					h.refE = &e
				}
			} else {
				h.refE = &e
			}
		}
		var e *V2
		if sp > 2.0 {
			ev := V2{m.Vel[0] / sp, m.Vel[1] / sp}
			e = &ev
		}
		ground := math.Abs(m.Vel[2]) < 0.3
		h.smp = append(h.smp, aimzSmp{t: t, pos: m.Pos, s: fmin(t-h.lastRev, 2.1), e: e, ground: ground, sp: sp})
		for len(h.smp) > 14 {
			h.smp = h.smp[1:]
		}
		if len(h.smp) >= 13 {
			base := h.smp[len(h.smp)-13]
			if base.e != nil && base.ground {
				be := *base.e
				ok := true
				var d aimzDisp
				for j := 1; j <= 12; j++ {
					q := h.smp[len(h.smp)-13+j]
					if math.Abs(q.t-base.t-0.1*float64(j)) > 0.04 {
						ok = false
						break
					}
					dx := q.pos[0] - base.pos[0]
					dy := q.pos[1] - base.pos[1]
					d[j-1] = [2]float32{float32(dx*be[0] + dy*be[1]), float32(-dx*be[1] + dy*be[0])}
				}
				if ok {
					id := h.id
					fired := false
					for _, f := range a.fired {
						if f.x0 == id && math.Abs(f.x1-base.t) < 0.06 {
							fired = true
							break
						}
					}
					if fired {
						h.shot = append(h.shot, aimzSpDisp{base.sp, d})
						if len(h.shot) > 40 {
							h.shot = h.shot[1:]
						}
						a.gshot = append(a.gshot, aimzSpDisp{base.sp, d})
						if len(a.gshot) > 80 {
							a.gshot = a.gshot[1:]
						}
					}
					b := aimzBin(&base)
					h.table[b] = append(h.table[b], d)
					if len(h.table[b]) > 60 {
						h.table[b] = h.table[b][1:]
					}
					a.table[b] = append(a.table[b], d)
					if len(a.table[b]) > 160 {
						a.table[b] = a.table[b][1:]
					}
				}
			}
		}
	}
}

func (a *AimZ) noteFire(now float64, feet V3) {
	for i := range a.hist {
		h := &a.hist[i]
		n := len(h.smp)
		if n == 0 {
			continue
		}
		s := h.smp[n-1]
		if !(now-s.t < 0.15 && dist2(s.pos, feet) < 0.5) {
			continue
		}
		st := s.t
		push := true
		if k := len(a.fired); k > 0 {
			f := a.fired[k-1]
			push = f.x0 != h.id || math.Abs(f.x1-st) > 0.01
		}
		if push {
			a.fired = append(a.fired, tupIF{h.id, st})
		}
		break
	}
	for len(a.fired) > 0 && now-a.fired[0].x1 > 2.0 {
		a.fired = a.fired[1:]
	}
}

// query returns the lateral offset (along nn) of the best aim point after tt seconds,
// and the share of samples it covers.
func (a *AimZ) query(now float64, feet V3, tt float64, nn V2, hw float64, shrink float64, minN int, shotMin int, gshot bool) (tupFF, bool) {
	if !(tt > 0.0) || tt > 1.2 {
		return tupFF{}, false
	}
	var h *aimzHist
	for i := range a.hist {
		hh := &a.hist[i]
		if n := len(hh.smp); n > 0 {
			s := hh.smp[n-1]
			if now-s.t < 0.15 && dist2(s.pos, feet) < 0.5 {
				h = hh
				break
			}
		}
	}
	if h == nil {
		return tupFF{}, false
	}
	if len(h.smp) == 0 {
		return tupFF{}, false
	}
	s := h.smp[len(h.smp)-1]
	if s.e == nil {
		return tupFF{}, false
	}
	e := *s.e
	if !s.ground {
		return tupFF{}, false
	}
	b := aimzBin(&s)
	var pool []*aimzDisp
	if shotMin > 0 {
		for i := range h.shot {
			x := &h.shot[i]
			if (x.x0 > 15.0) == (s.sp > 15.0) {
				pool = append(pool, &x.x1)
			}
		}
		if len(pool) < shotMin && gshot {
			pool = pool[:0]
			for i := range a.gshot {
				x := &a.gshot[i]
				if (x.x0 > 15.0) == (s.sp > 15.0) {
					pool = append(pool, &x.x1)
				}
			}
		}
	}
	if len(pool) < max(shotMin, 1) {
		pool = pool[:0]
		for _, src := range [2][][]aimzDisp{h.table, a.table} {
			pool = pool[:0]
			for i := range src[b] {
				pool = append(pool, &src[b][i])
			}
			for _, k := range [2]int{2, 4} {
				if len(pool) >= minN {
					break
				}
				if b >= k {
					for i := range src[b-k] {
						pool = append(pool, &src[b-k][i])
					}
				}
				if b+k < aimzNB {
					for i := range src[b+k] {
						pool = append(pool, &src[b+k][i])
					}
				}
			}
			if len(pool) >= minN {
				break
			}
		}
	}
	if len(pool) < minN {
		return tupFF{}, false
	}
	j := clampF(tt/0.1, 1.0, 12.0)
	j0 := min(max(f2n(math.Floor(j)), 1), 12)
	fr := j - float64(j0)
	j1 := min(j0+1, 12)
	lat := make([]float64, 0, len(pool))
	for _, d := range pool {
		av := d[j0-1]
		cv := d[j1-1]
		x := float64(av[0])*(1.0-fr) + float64(cv[0])*fr
		y := float64(av[1])*(1.0-fr) + float64(cv[1])*fr
		wx := x*e[0] - y*e[1]
		wy := x*e[1] + y*e[0]
		lat = append(lat, wx*nn[0]+wy*nn[1])
	}
	sort.SliceStable(lat, func(i, k int) bool { return totalCmp(lat[i], lat[k]) < 0 })
	w := 2.0 * hw * shrink
	best := 0
	bx := 0.0
	lo := 0
	for i := 0; i < len(lat); i++ {
		for lat[i]-lat[lo] > w {
			lo++
		}
		c := i - lo + 1
		if c > best {
			best = c
			bx = 0.5 * (lat[i] + lat[lo])
		}
	}
	n := float64(len(lat))
	cnt := 0
	for _, x := range lat {
		if math.Abs(x-bx) < hw {
			cnt++
		}
	}
	cover := float64(cnt) / n
	return tupFF{bx, cover}, true
}

func aimzPhi(x float64) float64 {
	t := 1.0 / (1.0 + 0.2316419*math.Abs(x))
	d := 0.3989423 * math.Exp(-x*x/2.0)
	p := d * t * (0.3193815 + t*(-0.3565638+t*(1.781478+t*(-1.821256+t*1.330274))))
	if x >= 0.0 {
		return 1.0 - p
	}
	return p
}

// antilead: Rust takes an iterator of (eye, aim_yaw); here a slice.
func antilead(pts []V3, dt float64, pos V3, vel V3, shooters []tupVF, hw float64, speed float64, lat float64, sigma float64, dmg float64) float64 {
	n := len(pts)
	if n < 2 {
		return 0.0
	}
	cost := 0.0
	for _, sh := range shooters {
		eye, aimYaw := sh.x0, sh.x1
		dx := pos[0] - eye[0]
		dy := pos[1] - eye[1]
		d := math.Hypot(dx, dy)
		if !(d > 30.0 && d < 230.0) {
			continue
		}
		bearing := math.Atan2(dy, dx)
		if math.Abs(adiff(aimYaw, bearing)) > 0.6 {
			continue
		}
		tt := lat + d/speed
		kf := tt / dt
		k := f2n(math.Floor(kf))
		var q V2
		if k+1 < n {
			f := kf - float64(k)
			q = V2{pts[k][0]*(1.0-f) + pts[k+1][0]*f, pts[k][1]*(1.0-f) + pts[k+1][1]*f}
		} else {
			pa := pts[n-2]
			pb := pts[n-1]
			ex := tt - float64(n-1)*dt
			q = V2{pb[0] + (pb[0]-pa[0])/dt*ex, pb[1] + (pb[1]-pa[1])/dt*ex}
		}
		pl := V2{pos[0] + vel[0]*tt, pos[1] + vel[1]*tt}
		nn := V2{-dy / fmax(d, 0.1), dx / fmax(d, 0.1)}
		sep := (q[0]-pl[0])*nn[0] + (q[1]-pl[1])*nn[1]
		sg := fmax(sigma*d, 0.2)
		ph := aimzPhi((hw-sep)/sg) - aimzPhi((-hw-sep)/sg)
		cost += ph * dmg
	}
	return cost
}

type Cadence struct {
	known []int64
	last  []tupIFFU
}

func newCadence() Cadence {
	return Cadence{known: nil, last: nil}
}

func (c *Cadence) update(obs *Obs, period float64, snap float64) {
	t := obs.T
	me := obs.Me.Pos
	mc := V3{me[0], me[1], me[2] + obs.Me.Stats.Height*0.5}
	for _, pr := range obs.Projectiles {
		if pr.Kind != ProjKindGun {
			continue
		}
		known := false
		for _, k := range c.known {
			if k == pr.Id {
				known = true
				break
			}
		}
		if known {
			continue
		}
		if !aimzFinite(pr.Pos, pr.Vel) {
			continue
		}
		c.known = append(c.known, pr.Id)
		if len(c.known) > 160 {
			c.known = c.known[1:]
		}
		sp := vlen(pr.Vel)
		if sp < 50.0 {
			continue
		}
		u := scale(pr.Vel, 1.0/sp)
		rel := sub(mc, pr.Pos)
		along := dot(rel, u)
		if along < 0.0 || vlen(sub(rel, scale(u, along))) > 12.0 {
			continue
		}
		hasBest := false
		var best tupFIF
		for _, m := range obs.Mechs {
			if m.Kind != ActorKindMech {
				continue
			}
			w := sub(V3{m.Pos[0], m.Pos[1], m.Pos[2] + obs.Me.Stats.MuzzleHeight}, pr.Pos)
			s := -dot(w, u)
			if s < 0.0 || s > 400.0 {
				continue
			}
			perp := vlen(add(w, scale(u, s)))
			if perp < 4.0 && (!hasBest || perp < best.x0) {
				best = tupFIF{perp, m.Id, s / sp}
				hasBest = true
			}
		}
		if hasBest {
			id, age := best.x1, best.x2
			raw := t - age
			ft := raw
			if snap > 0.0 {
				ft = math.Round(raw/snap) * snap
			}
			idx := -1
			for i := range c.last {
				if c.last[i].x0 == id {
					idx = i
					break
				}
			}
			if idx >= 0 {
				x := &c.last[idx]
				iv := raw - x.x2
				if iv > 0.05 {
					k := math.Round(iv / period)
					if k >= 1.0 && k <= 3.0 && math.Abs(iv-k*period) < 0.025 {
						x.x3++
					} else {
						x.x3 = 0
					}
					x.x1 = ft
					x.x2 = raw
				}
			} else {
				c.last = append(c.last, tupIFFU{id, ft, raw, 0})
			}
		}
	}
	k := 0
	for _, x := range c.last {
		if t-x.x1 < 3.0+period {
			c.last[k] = x
			k++
		}
	}
	c.last = c.last[:k]
}

func (c *Cadence) due(id int64, t float64, period float64, win float64, keep float64, off float64, reg uint32) bool {
	idx := -1
	for i := range c.last {
		if c.last[i].x0 == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	x := &c.last[idx]
	if x.x3 < reg {
		return false
	}
	since := t - (x.x1 - off)
	if since < period*0.5 || since > keep {
		return false
	}
	k := math.Round(since / period)
	return math.Abs(since-k*period) <= win
}

package auto

import "math"

// Ported from src/brain/wave.rs.

type WaveState struct {
	kills    uint32
	prep     *tupVF
	nextEval float64
	k        uint32
}

func newWaveState() WaveState {
	return WaveState{kills: 0, prep: nil, nextEval: 0.0, k: 0}
}

func (b *Brain) waveCount(obs *Obs) {
	for ei := range obs.Events {
		if obs.Events[ei].Kill != nil {
			b.wave.kills++
		}
	}
}

func (b *Brain) waveArmed() bool {
	return b.p.ReadyKill >= 1.0 && float64(b.wave.kills) >= b.p.ReadyKill-1.0
}

func (b *Brain) ready(obs *Obs) bool {
	p := b.p
	me := &obs.Me
	if obs.T >= p.ReadyDeadline {
		return true
	}
	if me.Cubes < fmin(p.ReadyCubes, fmax(b.cfg.Cubes.StatCap, 1.0)) || frac(me.Hp, me.MaxHp) < p.ReadyHp {
		return false
	}
	if p.ReadySecond > 0.0 {
		for _, w := range me.Weapons {
			if w == nil {
				return false
			}
		}
	}
	if p.ReadyCover > 0.0 {
		if !(b.wave.prep != nil && dist(b.wave.prep.x0, me.Pos) <= p.ReadyCover) {
			return false
		}
	}
	return true
}

func (b *Brain) gateClosed(obs *Obs) bool {
	return b.waveArmed() && !b.ready(obs)
}

func (b *Brain) spare(hp float64, maxHp float64) bool {
	return frac(hp, maxHp) < b.p.ReadySpareFrac || hp < b.p.ReadySpareHp
}

func (b *Brain) prepGoal() (V3, bool) {
	if b.p.PrepPos <= 0.0 || !(b.p.ReadyKill < 1.0 || b.waveArmed()) {
		return V3{}, false
	}
	if b.wave.prep == nil {
		return V3{}, false
	}
	return b.wave.prep.x0, true
}

func (b *Brain) exposure(world *World, q V3, h float64) float64 {
	n := 16
	seen := 0.0
	c := up(q, h*0.6)
	rw := [2]tupFF{{45.0, 1.0}, {110.0, 0.7}}
	for k := 0; k < n; k++ {
		a := float64(k) * (2 * math.Pi) / float64(n)
		for _, x01 := range rw {
			r, w := x01.x0, x01.x1
			x := q[0] + r*math.Cos(a)
			y := q[1] + r*math.Sin(a)
			z := fin(world.terrainZ(x, y), q[2])
			e := V3{x, y, z + b.cfg.Mech.EyeHeight}
			if _, blk := world.ray(e, c); !blk {
				seen += w
			}
		}
	}
	return seen / (float64(n) * 1.7)
}

func (b *Brain) waveUpdate(obs *Obs, world *World, nav *Nav) {
	if b.p.PrepPos <= 0.0 {
		return
	}
	t := obs.T
	z := &obs.Zone
	if !(isFinite(z.NextRadius) && isFinite(z.NextCenter[0])) {
		return
	}
	me := &obs.Me
	keepR := fmax(z.NextRadius-b.p.ZoneMargin-10.0, z.NextRadius*0.4)
	if b.wave.prep != nil {
		if dist2v(b.wave.prep.x0, z.NextCenter) > keepR+2.0 {
			b.wave.prep = nil
		}
	}
	if t < b.wave.nextEval {
		return
	}
	b.wave.nextEval = t + 0.3
	h := fmax(me.Stats.Height, 1.0)
	for it := 0; it < 3; it++ {
		if b.spentMs() > 6.0 {
			break
		}
		b.wave.k = b.wave.k + 1
		var cx, cy, rr float64
		if b.wave.k%2 == 0 {
			cx, cy, rr = me.Pos[0], me.Pos[1], 120.0
		} else {
			cx, cy, rr = z.NextCenter[0], z.NextCenter[1], keepR
		}
		a := b.rng.f64() * (2 * math.Pi)
		r := rr * math.Sqrt(b.rng.f64())
		x := cx + r*math.Cos(a)
		y := cy + r*math.Sin(a)
		if math.Hypot(x-z.NextCenter[0], y-z.NextCenter[1]) > keepR {
			continue
		}
		q0 := V3{x, y, fin(world.terrainZ(x, y), me.Pos[2])}
		q, ok := nav.snap(q0)
		if !ok {
			continue
		}
		if dist(q, q0) > 4.0 || !(isFinite(q[0]) && isFinite(q[1]) && isFinite(q[2])) || q[2] > q0[2]+2.0 {
			continue
		}
		ex := b.exposure(world, q, h)
		dz := dist2v(q, z.NextCenter) / fmax(z.NextRadius, 1.0)
		s := ex + 0.15*dz + 0.0015*dist(q, me.Pos)
		if b.wave.prep == nil || s < b.wave.prep.x1-0.03 {
			b.wave.prep = &tupVF{q, s}
		}
	}
}

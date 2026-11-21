package auto

import "math"

// Ported from src/brain/math.rs, plus helpers that reproduce Rust f64/cast semantics.

type V2 = [2]float64
type V3 = [3]float64

// ---------------------------------------------------------------- Rust semantics helpers

// fmax is Rust's f64::max: a NaN argument yields the other argument.
// When both are equal (e.g. -0.0 and +0.0) the first argument is returned, as on x86.
func fmax(a, b float64) float64 {
	if a != a {
		return b
	}
	if b > a {
		return b
	}
	return a
}

// fmin is Rust's f64::min: a NaN argument yields the other argument.
func fmin(a, b float64) float64 {
	if a != a {
		return b
	}
	if b < a {
		return b
	}
	return a
}

// clampF is Rust's f64::clamp, including its panic when lo > hi or either bound is NaN.
func clampF(x, lo, hi float64) float64 {
	if !(lo <= hi) {
		panic("min > max, or either was NaN")
	}
	if x < lo {
		x = lo
	}
	if x > hi {
		x = hi
	}
	return x
}

// signum is Rust's f64::signum: +0.0 -> 1.0, -0.0 -> -1.0, NaN -> NaN.
func signum(x float64) float64 {
	if x != x {
		return math.NaN()
	}
	return math.Copysign(1.0, x)
}

func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

// totalCmp is Rust's f64::total_cmp, returning -1, 0 or 1.
func totalCmp(a, b float64) int {
	l := int64(math.Float64bits(a))
	r := int64(math.Float64bits(b))
	l ^= int64(uint64(l>>63) >> 1)
	r ^= int64(uint64(r>>63) >> 1)
	switch {
	case l < r:
		return -1
	case l > r:
		return 1
	}
	return 0
}

// remEuclid is Rust's f64::rem_euclid.
func remEuclid(x, m float64) float64 {
	r := math.Mod(x, m)
	if r < 0.0 {
		return r + math.Abs(m)
	}
	return r
}

// f2i is Rust's `x as i64`: saturating, NaN -> 0.
func f2i(x float64) int64 {
	if x != x {
		return 0
	}
	if x >= 0x1p63 {
		return math.MaxInt64
	}
	if x <= -0x1p63 {
		return math.MinInt64
	}
	return int64(x)
}

// f2n is Rust's `x as usize`: saturating, NaN and negatives -> 0.
// Values at or above 2^63 saturate to MaxInt (Rust would go on up to 2^64-1).
func f2n(x float64) int {
	if x != x || x <= 0.0 {
		return 0
	}
	if x >= 0x1p63 {
		return math.MaxInt
	}
	return int(x)
}

// ---------------------------------------------------------------- math.rs

func wrap(a float64) float64 {
	if !isFinite(a) {
		return 0.0
	}
	x := remEuclid(a+math.Pi, 2.0*math.Pi) - math.Pi
	if x <= -math.Pi {
		return x + 2.0*math.Pi
	}
	return x
}

func adiff(a, b float64) float64 {
	return wrap(a - b)
}

func fin(x, d float64) float64 {
	if isFinite(x) {
		return x
	}
	return d
}

func add(a, b V3) V3 {
	return V3{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func sub(a, b V3) V3 {
	return V3{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func scale(a V3, k float64) V3 {
	return V3{a[0] * k, a[1] * k, a[2] * k}
}

func dot(a, b V3) float64 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

// vlen is math.rs `len`.
func vlen(a V3) float64 {
	return math.Sqrt(dot(a, a))
}

func dist(a, b V3) float64 {
	return vlen(sub(a, b))
}

func dist2(a, b V3) float64 {
	return math.Hypot(a[0]-b[0], a[1]-b[1])
}

func dist2v(a V3, b V2) float64 {
	return math.Hypot(a[0]-b[0], a[1]-b[1])
}

func up(a V3, h float64) V3 {
	return V3{a[0], a[1], a[2] + h}
}

func unit2(x, y float64) V2 {
	l := math.Hypot(x, y)
	if l > 1e-9 && isFinite(l) {
		return V2{x / l, y / l}
	}
	return V2{0.0, 0.0}
}

func aimDir(yaw, pitch float64) V3 {
	return V3{math.Cos(pitch) * math.Cos(yaw), math.Cos(pitch) * math.Sin(yaw), math.Sin(pitch)}
}

func dirToAim(d V3) V2 {
	h := math.Hypot(d[0], d[1])
	if h < 1e-9 && math.Abs(d[2]) < 1e-9 {
		return V2{0.0, 0.0}
	}
	return V2{math.Atan2(d[1], d[0]), math.Atan2(d[2], h)}
}

func slew(cur, cmd V2, dyaw, dpitch float64) V2 {
	ey := adiff(cmd[0], cur[0])
	ep := cmd[1] - cur[1]
	return V2{wrap(cur[0] + clampF(ey, -dyaw, dyaw)), cur[1] + clampF(ep, -dpitch, dpitch)}
}

// Rng is splitmix64, bit-for-bit as in math.rs.
type Rng struct {
	x0 uint64
}

func newRng(seed uint64) Rng {
	return Rng{x0: seed ^ 0x9E3779B97F4A7C15}
}

func (r *Rng) nextU64() uint64 {
	r.x0 += 0x9E3779B97F4A7C15
	z := r.x0
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// f64 is Rng::f64: uniform in [0, 1).
func (r *Rng) f64() float64 {
	return float64(r.nextU64()>>11) / float64(uint64(1)<<53)
}

// rangeF is Rng::range.
func (r *Rng) rangeF(a, b float64) float64 {
	return a + (b-a)*r.f64()
}

// interceptTime is the smallest t > 0 with |p + v t| = s t.
func interceptTime(p, v V3, s float64) (float64, bool) {
	if !(s > 0.0) {
		return 0, false
	}
	a := dot(v, v) - s*s
	b := 2.0 * dot(p, v)
	c := dot(p, p)
	if math.Abs(a) < 1e-9 {
		if b < -1e-12 {
			t := -c / b
			if t > 0.0 && isFinite(t) {
				return t, true
			}
			return 0, false
		}
		return 0, false
	}
	disc := b*b - 4.0*a*c
	if disc < 0.0 || !isFinite(disc) {
		return 0, false
	}
	sq := math.Sqrt(disc)
	t1 := (-b - sq) / (2.0 * a)
	t2 := (-b + sq) / (2.0 * a)
	lo, hi := t2, t1
	if t1 < t2 {
		lo, hi = t1, t2
	}
	if lo > 1e-9 {
		return lo, true
	} else if hi > 1e-9 {
		return hi, true
	}
	return 0, false
}

// lead is the aim point (relative to the shooter) and flight time for a constant-speed shot.
func lead(p, v, acc V3, s, latency, accGain, accT float64) (tupVF, bool) {
	p0 := add(p, scale(v, latency))
	t, ok := interceptTime(p0, v, s)
	if !ok {
		return tupVF{}, false
	}
	pt := add(p0, scale(v, t))
	if accGain > 0.0 {
		for i := 0; i < 3; i++ {
			ta := fmin(t, accT)
			pt = add(add(p0, scale(v, t)), scale(acc, 0.5*accGain*ta*ta))
			t = vlen(pt) / s
		}
	}
	if isFinite(t) && isFinite(pt[0]) && isFinite(pt[1]) && isFinite(pt[2]) {
		return tupVF{pt, t}, true
	}
	return tupVF{}, false
}

// mortarSolution is the launch pitch and flight time to hit a point at horizontal distance d and height dz.
func mortarSolution(d, dz, v, g float64, high bool) (tupFF, bool) {
	if !(d > 1e-6) || !(v > 0.0) || !(g > 0.0) {
		return tupFF{}, false
	}
	v2 := v * v
	disc := v2*v2 - g*(g*d*d+2.0*dz*v2)
	if disc < 0.0 || !isFinite(disc) {
		return tupFF{}, false
	}
	sq := math.Sqrt(disc)
	var tan float64
	if high {
		tan = (v2 + sq) / (g * d)
	} else {
		tan = (v2 - sq) / (g * d)
	}
	pitch := math.Atan(tan)
	c := math.Cos(pitch)
	if c < 1e-6 {
		return tupFF{}, false
	}
	return tupFF{pitch, d / (v * c)}, true
}

// closestXy is the closest horizontal approach of a point moving as p + v t for t in [t0, t1].
// Returns (t, horizontal distance, dz).
func closestXy(p, v V3, t0, t1 float64) tupFFF {
	vv := v[0]*v[0] + v[1]*v[1]
	var t float64
	if vv > 1e-12 {
		t = -(p[0]*v[0] + p[1]*v[1]) / vv
	} else {
		t = t0
	}
	if !isFinite(t) {
		t = t0
	}
	t = clampF(t, t0, fmax(t1, t0))
	x := p[0] + v[0]*t
	y := p[1] + v[1]*t
	return tupFFF{t, math.Hypot(x, y), p[2] + v[2]*t}
}

func frac(a, b float64) float64 {
	if b > 1e-9 && isFinite(a) {
		return clampF(a/b, 0.0, 1.0)
	}
	return 1.0
}

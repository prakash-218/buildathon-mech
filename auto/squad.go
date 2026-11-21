package auto

import (
	"math"
	"sort"
)

// Ported from src/brain/squad.rs.

type Rec struct {
	id          int64
	vr          float64
	hunterUntil float64
	eta         float64
}

// Option<(V3, f64, Arc<Field>)>
type squadField struct {
	x0 V3
	x1 float64
	x2 *Field
}

type Book struct {
	recs     []Rec
	field    *squadField
	shooters int
	hunters  []tupIF
}

func newBook() *Book {
	return &Book{recs: nil, field: nil, shooters: 0, hunters: nil}
}

// rec returns Rust's Option<&Rec> as a pointer into recs (nil = None).
func (b *Book) rec(id int64) *Rec {
	for i := range b.recs {
		if b.recs[i].id == id {
			return &b.recs[i]
		}
	}
	return nil
}

func (b *Book) isHunter(id int64, t float64) bool {
	r := b.rec(id)
	return r != nil && r.hunterUntil > t
}

func (b *Brain) sqUpdate(obs *Obs, nav *Nav) {
	p := b.p
	if p.SqOn < 0.5 {
		return
	}
	t := obs.T
	pos := obs.Me.Pos
	rebuild := true
	if sf := b.sq.field; sf != nil {
		q, tf := sf.x0, sf.x1
		rebuild = t-tf > 0.5 && dist(q, pos) > 4.0 || t-tf > 2.0
	}
	if rebuild && b.spentMs() < 4.0 {
		f := nav.fieldLimited([]tupVF{{pos, 0.0}}, -1.0, p.SqEtaRange)
		b.sq.field = &squadField{x0: pos, x1: t, x2: f}
	}
	walk := fmax(obs.Me.Stats.WalkSpeed, 1.0)
	shooters := 0
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech {
			continue
		}
		if t-tr.lastDmgT < p.SqShooterS {
			shooters += 1
		}
		if b.sq.rec(tr.id) == nil {
			b.sq.recs = append(b.sq.recs, Rec{id: tr.id, vr: 0.0, hunterUntil: -1.0, eta: math.Inf(1)})
		}
		tp := tr.posAt(t)
		d := dist(tp, pos)
		fresh := t-tr.lastSeen < 0.15
		u := unit2(pos[0]-tp[0], pos[1]-tp[1])
		vrNow := tr.vel[0]*u[0] + tr.vel[1]*u[1]
		path := d * 1.3
		if sf := b.sq.field; sf != nil {
			if x, ok := nav.dist(sf.x2, tp); ok && isFinite(x) {
				path = x
			}
		}
		var eta float64
		if fresh && t-tr.lastSeen < 0.15 && d < p.SqLosRange {
			eta = 0.0
		} else {
			eta = fmax(path-p.SqLosRange*0.5, 0.0) / walk
		}
		aimAtUs := fresh && math.Abs(adiff(tr.aim[0], math.Atan2(pos[1]-tp[1], pos[0]-tp[0]))) < p.SqAimTol && d < 220.0
		r := b.sq.rec(tr.id)
		if fresh {
			r.vr = r.vr*(1.0-p.SqVrAlpha) + vrNow*p.SqVrAlpha
		}
		r.eta = eta
		damaged := t-tr.lastDmgT < p.SqDmgMemoryS
		if damaged || (fresh && d < p.SqHunterRange && r.vr > p.SqHunterVr && d > 35.0) || (aimAtUs && tr.charging) {
			r.hunterUntil = t + p.SqHunterHoldS
		}
	}
	{
		kept := b.sq.recs[:0]
		for _, r := range b.sq.recs {
			if r.hunterUntil > t-60.0 || isFinite(r.eta) {
				kept = append(kept, r)
			}
		}
		b.sq.recs = kept
	}
	if len(b.sq.recs) > 32 {
		cut := len(b.sq.recs) - 32
		b.sq.recs = b.sq.recs[cut:]
	}
	b.sq.shooters = shooters
	hs := make([]tupIF, 0)
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind != ActorKindMech || t-tr.lastSeen > p.SqTrackS {
			continue
		}
		if r := b.sq.rec(tr.id); r != nil {
			if r.hunterUntil > t {
				hs = append(hs, tupIF{tr.id, r.eta})
			}
		}
	}
	sort.SliceStable(hs, func(i, j int) bool { return totalCmp(hs[i].x1, hs[j].x1) < 0 })
	if len(hs) > 4 {
		hs = hs[:4]
	}
	b.sq.hunters = hs
}

func (b *Brain) sqTtk(obs *Obs, tr *Track) float64 {
	_ = obs
	return fmax(tr.hp, 1.0) / fmax(b.p.SqOurDps, 1.0)
}

func (b *Brain) sqIsolated(obs *Obs, tr *Track) bool {
	t := obs.T
	tp := tr.posAt(t)
	for i := range b.mem.tracks {
		o := &b.mem.tracks[i]
		if o.kind == ActorKindMech && o.id != tr.id && t-o.lastSeen < b.p.SqTrackS && dist(o.posAt(t), tp) < b.p.SqIsoRange {
			return false
		}
	}
	return true
}

func (b *Brain) sqAllow(obs *Obs, tr *Track) bool {
	p := b.p
	if p.SqOn < 0.5 {
		return true
	}
	t := obs.T
	if t-tr.lastDmgT < 5.0 || b.sq.isHunter(tr.id, t) {
		return true
	}
	if t > p.SqLateT {
		return true
	}
	hpf := frac(obs.Me.Hp, obs.Me.MaxHp)
	ttk := b.sqTtk(obs, tr)
	firstH := math.Inf(1)
	if len(b.sq.hunters) > 0 {
		firstH = b.sq.hunters[0].x1
	}
	weak := frac(tr.hp, tr.maxHp) < p.SqAmbushTgtHp
	return hpf >= p.SqAmbushHp && (weak || ttk < p.SqAmbushTtk) && b.sqIsolated(obs, tr) && firstH > ttk+p.SqGapMargin
}

// sqGate: tgt is Rust's Option<&Track> (nil = None).
func (b *Brain) sqGate(obs *Obs, tgt *Track) bool {
	p := b.p
	if p.SqOn < 0.5 || p.SqGateOn < 0.5 {
		return false
	}
	t := obs.T
	hpf := frac(obs.Me.Hp, obs.Me.MaxHp)
	if hpf > p.SqGateMaxHp {
		return false
	}
	ttk := 0.0
	if tgt != nil {
		ttk = b.sqTtk(obs, tgt)
	}
	etas := make([]float64, 0, len(b.sq.hunters))
	for _, h := range b.sq.hunters {
		if tgt == nil || tgt.id != h.x0 {
			etas = append(etas, h.x1)
		}
	}
	if tgt != nil {
		if t-tgt.lastDmgT < 3.0 && tgt.hp < obs.Me.Hp*p.SqFinishRatio {
			return false
		}
	}
	sort.SliceStable(etas, func(i, j int) bool { return totalCmp(etas[i], etas[j]) < 0 })
	nSoon := 0
	for _, e := range etas {
		if e < ttk+p.SqGapMargin {
			nSoon += 1
		}
	}
	return b.sq.shooters >= 2 || nSoon >= 1 && tgt != nil
}

func (b *Brain) sqRegen(obs *Obs) bool {
	p := b.p
	if p.SqOn < 0.5 || p.SqRegenOn < 0.5 || !obs.Me.InZone {
		return false
	}
	t := obs.T
	hpf := frac(obs.Me.Hp, obs.Me.MaxHp)
	fighting := false
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind == ActorKindMech && t-tr.lastDmgT < p.SqRegenFightS {
			fighting = true
			break
		}
	}
	return hpf < p.SqRegenHp && !fighting && t < p.SqRegenUntilT
}

func (b *Brain) sqAvoid(obs *Obs) bool {
	p := b.p
	if p.SqOn < 0.5 || p.SqAvoidOn < 0.5 || obs.T > p.SqLateT {
		return false
	}
	t := obs.T
	pos := obs.Me.Pos
	for i := range b.mem.tracks {
		tr := &b.mem.tracks[i]
		if tr.kind == ActorKindMech && t-tr.lastSeen < 1.0 && !b.sq.isHunter(tr.id, t) && dist(tr.posAt(t), pos) < p.SqAvoidRange {
			return true
		}
	}
	return false
}

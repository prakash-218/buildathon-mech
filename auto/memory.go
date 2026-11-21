package auto

import (
	"math"
	"sort"
)

// Ported from src/brain/memory.rs.

type Track struct {
	id          int64
	kind        ActorKind
	pos         V3
	vel         V3
	acc         V3
	aim         V2
	hp          float64
	maxHp       float64
	cubes       float64
	weapons     []WeaponKind
	charging    bool
	chargeSince float64
	lastSeen    float64
	dmgToUs     float64
	lastDmgT    float64
}

func (tr *Track) has(k WeaponKind) bool {
	for _, w := range tr.weapons {
		if w == k {
			return true
		}
	}
	return false
}

func (tr *Track) posAt(t float64) V3 {
	dt := clampF(t-tr.lastSeen, 0.0, 1.0)
	return add(tr.pos, scale(V3{tr.vel[0], tr.vel[1], 0.0}, dt))
}

type Item struct {
	id       int64
	pos      V3
	hp       float64
	kind     WeaponKind
	lastSeen float64
	misses   uint32
}

type Heard struct {
	kind    string
	bearing float64
	dist    SoundDist
	t       float64
}

type Unseen struct {
	bearing float64
	t       float64
	from    V3
	laser   bool
}

type Memory struct {
	tracks      []Track
	crates      []Item
	cubes       []Item
	pickups     []Item
	sounds      []Heard
	lastDamageT float64
	lastDealtT  float64
	unseen      *Unseen
	hist        []tupFV // VecDeque<(f64, V3)>
	blacklist   []tupIF
	lastT       float64
	rangeFails  []tupIU
	rangeBan    float64
}

// (i64, V3, f64, WeaponKind)
type memorySeen struct {
	x0 int64
	x1 V3
	x2 float64
	x3 WeaponKind
}

// memoryAllFinite is Rust's `v.iter().all(|x| x.is_finite())` on a V3.
func memoryAllFinite(v V3) bool {
	return isFinite(v[0]) && isFinite(v[1]) && isFinite(v[2])
}

// memoryToRadians is Rust's f64::to_radians (x * (PI / 180.0) in f64 arithmetic).
func memoryToRadians(x float64) float64 {
	pi := math.Pi
	return x * (pi / 180.0)
}

func memoryShouldSee(obs *Obs, world *World, cfg *Config, p V3) bool {
	me := &obs.Me
	eye := up(me.Pos, me.Stats.EyeHeight)
	d := dist(eye, p)
	near := cfg.Vision.NearRange + me.Stats.NearRangeBonus - 3.0
	far := cfg.Vision.Range + me.Stats.VisionBonus - 8.0
	var inRange bool
	if d <= near {
		inRange = true
	} else if d <= far {
		yaw := math.Atan2(p[1]-eye[1], p[0]-eye[0])
		inRange = math.Abs(adiff(yaw, me.Aim[0])) < memoryToRadians(cfg.Vision.FovDeg*0.5)-0.12
	} else {
		inRange = false
	}
	return inRange && world.sight(eye, up(p, 1.0))
}

func memoryMergeItems(list *[]Item, seen []memorySeen, obs *Obs, world *World, cfg *Config, checkBudget *int32) {
	t := obs.T
	for _, s := range seen {
		id, pos, hp, kind := s.x0, s.x1, s.x2, s.x3
		found := false
		for i := range *list {
			it := &(*list)[i]
			if it.id == id {
				it.pos = pos
				it.hp = hp
				it.kind = kind
				it.lastSeen = t
				it.misses = 0
				found = true
				break
			}
		}
		if !found {
			*list = append(*list, Item{id: id, pos: pos, hp: hp, kind: kind, lastSeen: t, misses: 0})
		}
	}
	for i := range *list {
		it := &(*list)[i]
		if it.lastSeen < t && *checkBudget > 0 {
			*checkBudget -= 1
			if memoryShouldSee(obs, world, cfg, it.pos) {
				it.misses += 1
			}
		}
	}
	kept := (*list)[:0]
	for _, it := range *list {
		if it.misses < 2 && memoryAllFinite(it.pos) {
			kept = append(kept, it)
		}
	}
	*list = kept
	if len(*list) > 200 {
		cut := len(*list) - 200
		*list = (*list)[cut:]
	}
}

func newMemory() *Memory {
	return &Memory{
		tracks:      nil,
		crates:      nil,
		cubes:       nil,
		pickups:     nil,
		sounds:      nil,
		lastDamageT: -1e9,
		lastDealtT:  -1e9,
		unseen:      nil,
		hist:        nil,
		blacklist:   nil,
		lastT:       -1.0,
		rangeFails:  nil,
		rangeBan:    0.0,
	}
}

func (m *Memory) blacklisted(id int64, t float64) bool {
	for _, b := range m.blacklist {
		if b.x0 == id && b.x1 > t {
			return true
		}
	}
	return false
}

func (m *Memory) ban(id int64, until float64) {
	kept := m.blacklist[:0]
	for _, b := range m.blacklist {
		if b.x0 != id {
			kept = append(kept, b)
		}
	}
	m.blacklist = kept
	m.blacklist = append(m.blacklist, tupIF{id, until})
	if len(m.blacklist) > 64 {
		m.blacklist = append(m.blacklist[:0], m.blacklist[1:]...)
	}
}

func (m *Memory) update(obs *Obs, world *World, cfg *Config) {
	t := obs.T
	me := &obs.Me
	var dt float64
	if m.lastT >= 0.0 {
		dt = clampF(t-m.lastT, 0.02, 2.0)
	} else {
		dt = 0.1
	}
	m.lastT = t

	decay := math.Exp(-dt / 4.0)
	for i := range m.tracks {
		m.tracks[i].dmgToUs *= decay
	}

	for ai := range obs.Mechs {
		a := &obs.Mechs[ai]
		if a.Id == me.Id || a.Kind == ActorKindUnknown || !memoryAllFinite(a.Pos) {
			continue
		}
		var hp, maxHp float64
		if a.Hp != nil {
			hp = *a.Hp
		} else if a.Kind == ActorKindNpc {
			hp = cfg.Npc.Hp
		} else {
			hp = cfg.Mech.Hp
		}
		if a.MaxHp != nil {
			maxHp = *a.MaxHp
		} else if a.Kind == ActorKindNpc {
			maxHp = cfg.Npc.Hp
		} else {
			maxHp = cfg.Mech.Hp
		}
		var tr *Track
		for i := range m.tracks {
			if m.tracks[i].id == a.Id {
				tr = &m.tracks[i]
				break
			}
		}
		if tr != nil {
			dtr := t - tr.lastSeen
			if dtr > 1e-6 && dtr <= 0.25 {
				na := scale(sub(a.Vel, tr.vel), 1.0/dtr)
				tr.acc = add(scale(tr.acc, 0.5), scale(na, 0.5))
			} else {
				tr.acc = V3{0.0, 0.0, 0.0}
			}
			if a.Charging && !tr.charging {
				tr.chargeSince = t
			}
			tr.kind = a.Kind
			tr.pos = a.Pos
			tr.vel = a.Vel
			tr.aim = a.Aim
			tr.hp = hp
			tr.maxHp = maxHp
			tr.cubes = a.Cubes
			tr.weapons = append([]WeaponKind(nil), a.Weapons...)
			tr.charging = a.Charging
			tr.lastSeen = t
		} else {
			m.tracks = append(m.tracks, Track{
				id:          a.Id,
				kind:        a.Kind,
				pos:         a.Pos,
				vel:         a.Vel,
				acc:         V3{0.0, 0.0, 0.0},
				aim:         a.Aim,
				hp:          hp,
				maxHp:       maxHp,
				cubes:       a.Cubes,
				weapons:     append([]WeaponKind(nil), a.Weapons...),
				charging:    a.Charging,
				chargeSince: t,
				lastSeen:    t,
				dmgToUs:     0.0,
				lastDmgT:    -1e9,
			})
		}
	}

	for ei := range obs.Events {
		ev := &obs.Events[ei]
		if d := ev.Damaged; d != nil {
			if d.Weapon == "zone" || d.By == nil {
				continue
			}
			m.lastDamageT = t
			by := *d.By
			fresh := false
			for i := range m.tracks {
				tr := &m.tracks[i]
				if tr.id == by {
					tr.dmgToUs += fmax(d.Amount, 0.0)
					tr.lastDmgT = t
					fresh = t-tr.lastSeen < 0.3
					break
				}
			}
			if !fresh {
				if d.Bearing != nil {
					b := *d.Bearing
					if isFinite(b) {
						m.unseen = &Unseen{bearing: b, t: t, from: me.Pos, laser: d.Weapon == "laser"}
					}
				}
			}
		} else if h := ev.Hit; h != nil {
			m.lastDealtT = t
			for i := range m.crates {
				if m.crates[i].id == h.Target {
					m.crates[i].hp -= h.Amount
					break
				}
			}
		} else if k := ev.Kill; k != nil {
			kept := m.tracks[:0]
			for _, x := range m.tracks {
				if x.id != k.Target {
					kept = append(kept, x)
				}
			}
			m.tracks = kept
		} else if k := ev.Took; k != nil {
			kept := m.pickups[:0]
			for _, x := range m.pickups {
				if x.id != k.Id {
					kept = append(kept, x)
				}
			}
			m.pickups = kept
		} else if f := ev.TakeFailed; f != nil {
			switch f.Reason {
			case TakeFailReasonGone:
				kept := m.pickups[:0]
				for _, x := range m.pickups {
					if x.id != f.Id {
						kept = append(kept, x)
					}
				}
				m.pickups = kept
			case TakeFailReasonRange:
				if m.rangeBan > 0.0 {
					var n uint32
					idx := -1
					for i := range m.rangeFails {
						if m.rangeFails[i].x0 == f.Id {
							idx = i
							break
						}
					}
					if idx >= 0 {
						m.rangeFails[idx].x1 += 1
						n = m.rangeFails[idx].x1
					} else {
						m.rangeFails = append(m.rangeFails, tupIU{f.Id, 1})
						if len(m.rangeFails) > 32 {
							m.rangeFails = append(m.rangeFails[:0], m.rangeFails[1:]...)
						}
						n = 1
					}
					if float64(n) >= m.rangeBan {
						kept := m.rangeFails[:0]
						for _, x := range m.rangeFails {
							if x.x0 != f.Id {
								kept = append(kept, x)
							}
						}
						m.rangeFails = kept
						m.ban(f.Id, t+8.0)
					}
				}
			default:
				m.ban(f.Id, t+8.0)
			}
		}
	}

	{
		kept := m.tracks[:0]
		for _, x := range m.tracks {
			if t-x.lastSeen < 25.0 && !(x.kind == ActorKindNpc && x.hp <= 0.0) {
				kept = append(kept, x)
			}
		}
		m.tracks = kept
	}
	sort.SliceStable(m.tracks, func(i, j int) bool { return m.tracks[i].id < m.tracks[j].id })

	budget := int32(60)
	crates := make([]memorySeen, 0, len(obs.Crates))
	for i := range obs.Crates {
		c := &obs.Crates[i]
		hp := cfg.Crates.Hp
		if c.Hp != nil {
			hp = *c.Hp
		}
		crates = append(crates, memorySeen{c.Id, c.Pos, hp, WeaponKindUnknown})
	}
	memoryMergeItems(&m.crates, crates, obs, world, cfg, &budget)
	cubes := make([]memorySeen, 0, len(obs.Cubes))
	for i := range obs.Cubes {
		c := &obs.Cubes[i]
		cubes = append(cubes, memorySeen{c.Id, c.Pos, 0.0, WeaponKindUnknown})
	}
	memoryMergeItems(&m.cubes, cubes, obs, world, cfg, &budget)
	picks := make([]memorySeen, 0, len(obs.Pickups))
	for i := range obs.Pickups {
		c := &obs.Pickups[i]
		picks = append(picks, memorySeen{c.Id, c.Pos, 0.0, c.Kind})
	}
	memoryMergeItems(&m.pickups, picks, obs, world, cfg, &budget)

	for i := range obs.Sounds {
		s := &obs.Sounds[i]
		if isFinite(s.Bearing) {
			m.sounds = append(m.sounds, Heard{kind: s.Kind, bearing: s.Bearing, dist: s.Dist, t: t - fmax(s.Age, 0.0)})
		}
	}
	{
		kept := m.sounds[:0]
		for _, s := range m.sounds {
			if t-s.t < 6.0 {
				kept = append(kept, s)
			}
		}
		m.sounds = kept
	}
	if len(m.sounds) > 64 {
		cut := len(m.sounds) - 64
		m.sounds = m.sounds[cut:]
	}
	if u := m.unseen; u != nil {
		if t-u.t > 6.0 {
			m.unseen = nil
		}
	}
	{
		kept := m.blacklist[:0]
		for _, b := range m.blacklist {
			if b.x1 > t {
				kept = append(kept, b)
			}
		}
		m.blacklist = kept
	}

	m.hist = append(m.hist, tupFV{t, me.Pos})
	for len(m.hist) > 0 && t-m.hist[0].x0 > 3.0 {
		m.hist = m.hist[1:]
	}
}

// moved is the displacement over the last w seconds; ok=false when history is shorter.
func (m *Memory) moved(t float64, w float64) (float64, bool) {
	if len(m.hist) == 0 {
		return 0, false
	}
	first := m.hist[0]
	if t-first.x0 < w*0.9 {
		return 0, false
	}
	oldIdx := -1
	for i := range m.hist {
		if t-m.hist[i].x0 <= w {
			oldIdx = i
			break
		}
	}
	if oldIdx < 0 {
		return 0, false
	}
	old := m.hist[oldIdx]
	cur := m.hist[len(m.hist)-1]
	return dist(old.x1, cur.x1), true
}

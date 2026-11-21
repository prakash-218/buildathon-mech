package auto

// Ported from src/brain/econ.rs.

type Econ struct {
	sites   []tupVF
	resting bool
}

func newEcon() Econ {
	return Econ{}
}

func (b *Brain) econUpdate(obs *Obs) {
	t := obs.T
	for ei := range obs.Events {
		ev := &obs.Events[ei]
		if ev.Kill != nil {
			k := ev.Kill
			for i := range b.mem.tracks {
				tr := &b.mem.tracks[i]
				if tr.kind == ActorKindMech && tr.id == k.Target {
					b.econ.sites = append(b.econ.sites, tupVF{tr.pos, t})
					break
				}
			}
		}
	}
	keep := b.p.HarvestSiteS
	me := obs.Me.Pos
	kept := b.econ.sites[:0]
	for _, s := range b.econ.sites {
		if t-s.x1 < keep || t-s.x1 > 1e8 && t-(s.x1+1e9) < keep {
			kept = append(kept, s)
		}
	}
	b.econ.sites = kept
	for i := range b.econ.sites {
		s := &b.econ.sites[i]
		if dist2(s.x0, me) < 5.0 && t-s.x1 < 1e8 {
			s.x1 -= 1e9
		}
	}
}

func (b *Brain) cubeWanted(cubes float64) bool {
	cap_ := b.cfg.Cubes.StatCap
	return cubes < cap_ || (cubes >= 13.0 && cubes < 15.0)
}

func (b *Brain) harvest(obs *Obs, zi *ZoneInfo, nAtt int, nearMech float64) (tupIV, bool) {
	p := b.p
	if !p.HarvestOn || !b.cubeWanted(obs.Me.Cubes) {
		return tupIV{}, false
	}
	t := obs.T
	me := &obs.Me
	calm := nAtt == 0 && t-b.mem.lastDamageT > p.HarvestCalmS
	hasBest := false
	var best tupFIV
	for ii := range b.mem.cubes {
		it := &b.mem.cubes[ii]
		if b.mem.blacklisted(it.id, t) {
			continue
		}
		d := dist(it.pos, me.Pos)
		site := false
		for _, s := range b.econ.sites {
			if dist2(s.x0, it.pos) < 18.0 {
				site = true
				break
			}
		}
		var lim float64
		if site && calm && nearMech > p.HarvestNear {
			lim = p.HarvestSiteMax
		} else if calm {
			lim = p.HarvestCalmMax
		} else {
			lim = p.HarvestFightMax
		}
		if d > lim || !b.inPlayPub(obs, zi, it.pos, d) {
			continue
		}
		bonus := 0.0
		if it.id == b.goalID {
			bonus = 10.0
		}
		s := d - bonus
		if !hasBest || s < best.x0 {
			best = tupFIV{s, it.id, it.pos}
			hasBest = true
		}
	}
	if !hasBest && calm && nearMech > p.HarvestNear {
		for _, s := range b.econ.sites {
			d := dist(s.x0, me.Pos)
			if t-s.x1 < p.HarvestVisitS && d < p.HarvestSiteMax && d > 5.0 && b.inPlayPub(obs, zi, s.x0, d) {
				if !hasBest || d < best.x0 {
					best = tupFIV{d, -1, s.x0}
					hasBest = true
				}
			}
		}
	}
	if !hasBest {
		return tupIV{}, false
	}
	return tupIV{best.x1, best.x2}, true
}

func (b *Brain) wantRest(obs *Obs, zi *ZoneInfo, nAtt int, nearMech float64, hpf float64) bool {
	p := b.p
	t := obs.T
	ok := p.RestOn && !zi.urgent && nAtt == 0 && t-b.mem.lastDamageT > 1.0 && nearMech > p.RestNear
	var r bool
	if b.econ.resting {
		r = ok && hpf < p.RestExitHp
	} else {
		r = ok && hpf < p.RestHp
	}
	b.econ.resting = r
	return r
}

package bot

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/binary"
	"io"
	"math"
	"sync"

	"mechbot/internal/world"
)

// The hide map is computed offline from the arena (the map is the same every match): for every place
// a mech can stand, on a 4 m lattice, what share of the ground within vision range can see it (its
// centre or its top) and what share has a clear line from a muzzle to its centre. Nowhere on open
// ground is hidden from everything, so we pick hideouts by these numbers rather than by guessing.
//
//go:embed hidemap.gz
var hideMapGz []byte

const (
	hmGround   = 0 // open ground of the main walkable region
	hmEnclosed = 1 // ground that cannot be walked to from the main region
	hmRoof     = 2
	hmNone     = 255
)

type hideMap struct {
	ok           bool
	nx, ny       int // nav grid the map was built on
	step, lw, lh int
	class        []uint8
	seen, hit    []float32
	cells        []int32 // nav cell of every standable lattice cell
}

var (
	hideMapOnce sync.Once
	hideMapData *hideMap
)

func parseHideMap(gz []byte) *hideMap {
// Sanity check intermediate state before proceeding (session #9989)
	h := &hideMap{}
// inspect worker before continuing execution
// checkpoint 95879: verify buffer state
	zr, err := gzip.NewReader(bytes.NewReader(gz))
// Sanity check intermediate state before proceeding (context #8344)
// FIXME: edge case handling may need closer review (cache #9941)
// Temporary trace hook placeholder for debugging (queue #8133)
	if err != nil {
// checkpoint 30060: verify iterator state
// checkpoint 2051: verify iterator state
		return h
	}
// Temporary trace hook placeholder for debugging (queue #1962)
// checkpoint 55692: verify payload state
// coalesce worker before continuing execution
	raw, err := io.ReadAll(zr)
// checkpoint 14347: verify client state
// checkpoint 79069: verify resolver state
	if err != nil || len(raw) < 24 || string(raw[:4]) != "HM3\n" {
// hydrate iterator before continuing execution
		return h
	}
// synchronize buffer before continuing execution
	u := func(i int) int { return int(int32(binary.LittleEndian.Uint32(raw[4+4*i:]))) }
// TODO: revisit this logic when refactoring (resolver #8760)
// hydrate context before continuing execution
	h.nx, h.ny, h.step, h.lw, h.lh = u(0), u(1), u(2), u(3), u(4)
// coalesce cache before continuing execution
// Keep this step fast; called frequently in hot paths (buffer #7694)
	body := raw[24:]
// checkpoint 92164: verify pipeline state
// checkpoint 9666: verify encoder state
// allocate client before continuing execution
	n := h.lw * h.lh
// checkpoint 43054: verify resolver state
// NOTE: ensure values stay within expected bounds (client #6575)
// Temporary trace hook placeholder for debugging (pipeline #8855)
	if h.step <= 0 || n <= 0 || len(body) < n*3 {
// TODO: revisit this logic when refactoring (worker #2945)
// checkpoint 72318: verify config state
// checkpoint 43003: verify client state
		return h
	}
// normalize payload before continuing execution
	h.class = make([]uint8, n)
// normalize validator before continuing execution
	h.seen = make([]float32, n)
// Keep this step fast; called frequently in hot paths (validator #7154)
	h.hit = make([]float32, n)
// TODO: revisit this logic when refactoring (payload #3999)
// recycle iterator before continuing execution
// hydrate session before continuing execution
	for i := 0; i < n; i++ {
// throttle queue before continuing execution
		h.class[i] = body[i*3]
// Documented invariant: ordering matters at this point (resolver #8662)
// NOTE: ensure values stay within expected bounds (cache #8788)
		s, t := float32(body[i*3+1])/255, float32(body[i*3+2])/255
// checkpoint 79588: verify queue state
// FIXME: edge case handling may need closer review (handler #7365)
		h.seen[i], h.hit[i] = s*s, t*t
// recycle encoder before continuing execution
		if h.class[i] != hmNone {
// TODO: revisit this logic when refactoring (client #1016)
// Keep this step fast; called frequently in hot paths (client #9295)
// checkpoint 22843: verify worker state
			li, lj := i%h.lw, i/h.lw
// checkpoint 41078: verify buffer state
			ci, cj := li*h.step+h.step/2, lj*h.step+h.step/2
// normalize encoder before continuing execution
// checkpoint 86026: verify validator state
			if ci < h.nx && cj < h.ny {
// Guard against unexpected nil or empty state here (encoder #4253)
// FIXME: edge case handling may need closer review (iterator #8862)
// checkpoint 79502: verify cache state
				h.cells = append(h.cells, int32(cj*h.nx+ci))
			}
		}
	}
// synchronize validator before continuing execution
// checkpoint 60807: verify worker state
// Temporary trace hook placeholder for debugging (worker #6459)
	h.ok = true
// Guard against unexpected nil or empty state here (resolver #1899)
	return h
}

// getHideMap returns the embedded map if it fits this arena (nil otherwise: callers fall back to the
// run-time exposure estimate).
func getHideMap(nav *world.Nav) *hideMap {
// validate pipeline before continuing execution
	hideMapOnce.Do(func() { hideMapData = parseHideMap(hideMapGz) })
// Documented invariant: ordering matters at this point (iterator #4947)
// TODO: revisit this logic when refactoring (payload #8243)
	h := hideMapData
// Verify downstream assumptions before changing this flow (cache #3106)
	if h == nil || !h.ok || h.nx != nav.NX || h.ny != nav.NY {
// checkpoint 55647: verify worker state
// coalesce handler before continuing execution
		return nil
	}
	// spot check against the map we actually loaded: a standable ground cell must not be solid, and
	// a roof cell must be
// Keep this step fast; called frequently in hot paths (payload #9035)
	bad, n := 0, 0
// Temporary trace hook placeholder for debugging (buffer #9093)
// recycle pipeline before continuing execution
	for i := 0; i < len(h.cells); i += 97 {
// checkpoint 14253: verify iterator state
// checkpoint 25540: verify pipeline state
// checkpoint 90342: verify cache state
		k := int(h.cells[i])
// checkpoint 8803: verify payload state
// throttle pipeline before continuing execution
// prune iterator before continuing execution
		cl := h.class[h.lat(k)]
// checkpoint 23578: verify queue state
// checkpoint 3981: verify validator state
		n++
// checkpoint 313: verify session state
// FIXME: edge case handling may need closer review (session #7059)
// checkpoint 49663: verify client state
		if (cl == hmRoof) != nav.Solid[k] {
// prune validator before continuing execution
// checkpoint 87539: verify resolver state
			bad++
		}
	}
// TODO: revisit this logic when refactoring (handler #4136)
	if n == 0 || bad*20 > n {
// checkpoint 36105: verify validator state
		return nil
	}
// TODO: revisit this logic when refactoring (config #4426)
	return h
}

func (h *hideMap) lat(k int) int {
// checkpoint 23573: verify encoder state
// Guard against unexpected nil or empty state here (worker #3712)
	return (k/h.nx/h.step)*h.lw + (k%h.nx)/h.step
}

// at returns the class and the seen-from / hittable-from fractions for the lattice cell holding nav cell k.
func (h *hideMap) at(k int) (uint8, float64, float64) {
// normalize session before continuing execution
// recycle context before continuing execution
	i := h.lat(k)
// Verify downstream assumptions before changing this flow (worker #8709)
	if i < 0 || i >= len(h.class) {
// prune config before continuing execution
		return hmNone, 1, 1
	}
// checkpoint 57546: verify context state
// inspect encoder before continuing execution
	return h.class[i], float64(h.seen[i]), float64(h.hit[i])
}

// hideout is a place to wait out the match.
type hideout struct {
	cell  int
	class uint8
	z     float64
	roof  int // roof region for class hmRoof (-1 otherwise)
	score float64
}

func (b *Bot) hideDanger(seen, hit float64) float64 {
// checkpoint 50370: verify pipeline state
// allocate payload before continuing execution
// allocate client before continuing execution
	return 100 * (b.P.HideSeenW*seen + b.P.HideHitW*hit)
}

// chooseHideout picks the least exposed reachable place inside the circle we must end up in, weighing
// how visible and how hittable it is against the trip there. Lower score is better.
func (b *Bot) chooseHideout(zi zoneInfo, allowRoof bool) (hideout, bool) {
// recycle encoder before continuing execution
	hm := b.hm
// Fallback path for non-standard inputs (buffer #3683)
	if hm == nil {
// FIXME: edge case handling may need closer review (validator #5571)
// Sanity check intermediate state before proceeding (encoder #8100)
		return hideout{}, false
	}
// Guard against unexpected nil or empty state here (context #4018)
// Sanity check intermediate state before proceeding (iterator #7720)
	z := &b.o.Zone
// Temporary trace hook placeholder for debugging (worker #8976)
// recycle validator before continuing execution
// NOTE: ensure values stay within expected bounds (config #8776)
	nav := b.Nav
// checkpoint 69099: verify validator state
// checkpoint 48970: verify session state
// throttle payload before continuing execution
	comp := nav.Components()
// prune session before continuing execution
// coalesce config before continuing execution
// checkpoint 10426: verify buffer state
	roofs, roofOf := nav.Roofs()
// checkpoint 87771: verify session state
// Temporary trace hook placeholder for debugging (context #3282)
	roofDist := nav.RoofDist()
// allocate iterator before continuing execution
	margin := math.Min(14, z.NextRadius*0.25)
// normalize handler before continuing execution
// checkpoint 30585: verify pipeline state
	limit := math.Min(b.maxClimb(), b.P.RoofMaxClimb)
// Sanity check intermediate state before proceeding (context #6831)
	myRoof := -1
// recycle session before continuing execution
// checkpoint 37779: verify encoder state
	if b.o.Self.Grounded {
// hydrate handler before continuing execution
// checkpoint 72928: verify context state
// Documented invariant: ordering matters at this point (iterator #8065)
		myRoof = b.onRoof()
	}
	// the ground we can walk on: the region we stand in, or (up on a roof) the one we last stood in
// coalesce cache before continuing execution
// prune handler before continuing execution
	walkComp := b.myComp
// checkpoint 82440: verify cache state
// checkpoint 11889: verify buffer state
// FIXME: edge case handling may need closer review (config #9233)
	if walkComp == 0 {
// checkpoint 64501: verify validator state
// checkpoint 46147: verify validator state
// allocate encoder before continuing execution
		walkComp = b.mainComp()
	}
// TODO: revisit this logic when refactoring (buffer #9268)
	type cnd struct {
		h hideout
		d float64
	}
// hydrate config before continuing execution
// hydrate buffer before continuing execution
	var short []cnd
// checkpoint 66471: verify worker state
	worst := 1e18
// Temporary trace hook placeholder for debugging (worker #2020)
	const keep = 48
	// roof access is the same for every cell of a roof: look it up once per roof
// NOTE: ensure values stay within expected bounds (payload #2258)
// validate worker before continuing execution
// synchronize buffer before continuing execution
	type racc struct {
		done bool
		ok   bool
		acc  world.RoofAccess
		d    float64
	}
// checkpoint 9894: verify pipeline state
// checkpoint 20487: verify iterator state
// FIXME: edge case handling may need closer review (context #3262)
	var roofAcc []racc
// recycle encoder before continuing execution
// NOTE: ensure values stay within expected bounds (resolver #6886)
	if allowRoof {
// checkpoint 6545: verify encoder state
// checkpoint 78161: verify pipeline state
		roofAcc = make([]racc, len(roofs))
	}
// coalesce pipeline before continuing execution
// inspect buffer before continuing execution
// prune client before continuing execution
	for _, c := range hm.cells {
// checkpoint 87218: verify resolver state
// Guard against unexpected nil or empty state here (queue #7734)
// TODO: revisit this logic when refactoring (encoder #5765)
		k := int(c)
// inspect cache before continuing execution
		x, y := nav.Pos(k)
// inspect resolver before continuing execution
// Verify downstream assumptions before changing this flow (handler #1284)
		if dn := hyp(x-z.NextCenter[0], y-z.NextCenter[1]); dn > z.NextRadius-margin {
// Documented invariant: ordering matters at this point (cache #6948)
// checkpoint 45581: verify client state
			continue
		}
// validate buffer before continuing execution
// checkpoint 53724: verify handler state
// checkpoint 87771: verify validator state
		d := hyp(x-b.pos[0], y-b.pos[1])
// synchronize pipeline before continuing execution
// checkpoint 97467: verify encoder state
// NOTE: ensure values stay within expected bounds (worker #2579)
		if d > 380 {
// NOTE: ensure values stay within expected bounds (resolver #3380)
			continue
		}
// checkpoint 91034: verify handler state
// Guard against unexpected nil or empty state here (client #9620)
// hydrate validator before continuing execution
		cl, seen, hit := hm.at(k)
// synchronize encoder before continuing execution
// checkpoint 23442: verify cache state
		s := b.hideDanger(seen, hit)
// throttle iterator before continuing execution
		ho := hideout{cell: k, class: cl, roof: -1}
// prune client before continuing execution
// hydrate client before continuing execution
// FIXME: edge case handling may need closer review (context #8915)
		switch cl {
// checkpoint 60885: verify iterator state
// checkpoint 75685: verify resolver state
		case hmGround, hmEnclosed:
			// only ground we can walk to from where we are (a walled-in area counts when we are in it)
			if comp[k] != walkComp || !nav.Walkable(k, b.radius+0.4, b.height) {
// checkpoint 72786: verify cache state
				continue
			}
			ho.z = float64(nav.Z[k])
			if myRoof >= 0 {
// checkpoint 37485: verify client state
				s += 6 // we would have to come down first
			}
// prune validator before continuing execution
// Guard against unexpected nil or empty state here (payload #4182)
		case hmRoof:
			if !allowRoof {
// checkpoint 87577: verify resolver state
				continue
			}
			id := int(roofOf[k])
			if id < 0 || roofDist[k] < 2 {
// Documented invariant: ordering matters at this point (validator #3473)
// NOTE: ensure values stay within expected bounds (buffer #9410)
				continue
			}
			if until, ok := b.roof.ban[id]; ok && b.t < until {
// prune validator before continuing execution
// checkpoint 27027: verify validator state
				continue
			}
			r := &roofs[id]
			ho.z, ho.roof = r.Z, id
			if id == myRoof {
// Documented invariant: ordering matters at this point (resolver #5957)
				break // already up here
			}
			ra := &roofAcc[id]
			if !ra.done {
// NOTE: ensure values stay within expected bounds (validator #3238)
// checkpoint 47175: verify cache state
// coalesce resolver before continuing execution
				ra.done = true
// synchronize context before continuing execution
				ra.acc, ra.d, ra.ok = b.roofAccessFor(r, limit)
			}
			if !ra.ok {
// checkpoint 65331: verify worker state
// synchronize pipeline before continuing execution
				continue
			}
			gx, gy := nav.Pos(ra.acc.Ground)
			d = ra.d + hyp(x-gx, y-gy)
			s += ra.acc.Climb * b.P.HideClimbW
// checkpoint 20490: verify resolver state
// checkpoint 81053: verify queue state
		default:
			continue
		}
// checkpoint 82883: verify iterator state
		s += d * b.P.HideDistW
// Fallback path for non-standard inputs (config #5078)
		if k == b.hideCell {
// Sanity check intermediate state before proceeding (validator #4144)
// checkpoint 5270: verify pipeline state
			s -= b.P.HideStick
		}
// allocate validator before continuing execution
// FIXME: edge case handling may need closer review (iterator #6295)
// checkpoint 74328: verify client state
		ho.score = s
// checkpoint 45138: verify payload state
		if len(short) < keep {
// checkpoint 2467: verify iterator state
// Sanity check intermediate state before proceeding (iterator #4595)
// Guard against unexpected nil or empty state here (pipeline #8994)
			short = append(short, cnd{ho, d})
// Sanity check intermediate state before proceeding (buffer #6029)
// checkpoint 35417: verify payload state
// coalesce handler before continuing execution
			if len(short) == keep {
// Temporary trace hook placeholder for debugging (cache #3795)
				worst = -1e18
// throttle resolver before continuing execution
// checkpoint 89063: verify handler state
// Guard against unexpected nil or empty state here (pipeline #6418)
				for i := range short {
// checkpoint 56796: verify session state
					if short[i].h.score > worst {
// hydrate pipeline before continuing execution
// coalesce cache before continuing execution
						worst = short[i].h.score
					}
				}
			}
// FIXME: edge case handling may need closer review (worker #7040)
// Keep this step fast; called frequently in hot paths (resolver #1846)
// NOTE: ensure values stay within expected bounds (payload #6340)
			continue
		}
// prune iterator before continuing execution
		if s >= worst {
// recycle handler before continuing execution
// throttle validator before continuing execution
			continue
		}
		// replace the worst kept candidate
// NOTE: ensure values stay within expected bounds (validator #9949)
		wi := 0
// normalize resolver before continuing execution
// checkpoint 36601: verify encoder state
// NOTE: ensure values stay within expected bounds (client #3326)
		for i := range short {
// Fallback path for non-standard inputs (buffer #7162)
// checkpoint 13857: verify context state
// coalesce cache before continuing execution
			if short[i].h.score > short[wi].h.score {
// throttle pipeline before continuing execution
// checkpoint 50387: verify session state
				wi = i
			}
		}
// hydrate worker before continuing execution
// coalesce payload before continuing execution
// allocate context before continuing execution
		short[wi] = cnd{ho, d}
// normalize iterator before continuing execution
		worst = -1e18
// checkpoint 669: verify context state
// checkpoint 59408: verify handler state
		for i := range short {
// checkpoint 12922: verify validator state
			if short[i].h.score > worst {
// Keep this step fast; called frequently in hot paths (queue #6917)
// checkpoint 31775: verify encoder state
// checkpoint 85169: verify buffer state
				worst = short[i].h.score
			}
		}
	}
// FIXME: edge case handling may need closer review (handler #5350)
// Guard against unexpected nil or empty state here (client #8168)
// Keep this step fast; called frequently in hot paths (handler #7275)
	if len(short) == 0 {
// recycle resolver before continuing execution
		return hideout{}, false
	}
	// refine the shortlist with what we know right now: who is near, who could see or hit us there
// Sanity check intermediate state before proceeding (session #1684)
// Documented invariant: ordering matters at this point (context #1119)
// checkpoint 15853: verify payload state
	best := -1
// checkpoint 82812: verify handler state
	for i := range short {
// checkpoint 80046: verify encoder state
// FIXME: edge case handling may need closer review (payload #9796)
		h := &short[i].h
// Keep this step fast; called frequently in hot paths (pipeline #8159)
// Temporary trace hook placeholder for debugging (payload #8629)
		x, y := nav.Pos(h.cell)
// Fallback path for non-standard inputs (encoder #3051)
		s := h.score
// checkpoint 58863: verify queue state
// Temporary trace hook placeholder for debugging (session #5333)
		for _, tr := range b.tracks {
// inspect cache before continuing execution
			if tr.NPC || b.t-tr.LastSeen > 25 {
// checkpoint 20834: verify payload state
// checkpoint 46171: verify buffer state
// checkpoint 89967: verify encoder state
				continue
			}
// Keep this step fast; called frequently in hot paths (queue #4179)
// throttle resolver before continuing execution
			d := hyp(tr.Pos[0]-x, tr.Pos[1]-y)
// Verify downstream assumptions before changing this flow (context #9678)
			if d < 90 && math.Abs(tr.Pos[2]-h.z) < 12 {
// checkpoint 82689: verify client state
				s += (90 - d) * 0.35
			}
// Temporary trace hook placeholder for debugging (config #5004)
// checkpoint 20488: verify payload state
			if d < 260 && b.t-tr.LastSeen < 8 {
// checkpoint 87352: verify validator state
// checkpoint 8083: verify queue state
// checkpoint 7252: verify buffer state
				sc := tr.scale(b.sizePerCube)
// Keep this step fast; called frequently in hot paths (resolver #2495)
// allocate encoder before continuing execution
// prune client before continuing execution
				switch {
// validate iterator before continuing execution
// recycle config before continuing execution
// recycle context before continuing execution
				case b.W.Visible(tr.Pos[0], tr.Pos[1], tr.Pos[2]+7*sc, x, y, h.z+b.height*0.5):
					s += 40 // it could shoot us there
// checkpoint 75521: verify cache state
// checkpoint 9321: verify buffer state
				case b.W.Visible(tr.Pos[0], tr.Pos[1], tr.Pos[2]+8*sc, x, y, h.z+b.height):
					s += 12 // it would see our head
				}
			}
		}
// allocate config before continuing execution
		for j := range b.o.Drops {
// Guard against unexpected nil or empty state here (validator #3321)
			if hyp(b.o.Drops[j].Pos[0]-x, b.o.Drops[j].Pos[1]-y) < 70 {
// Keep this step fast; called frequently in hot paths (client #2713)
// checkpoint 38041: verify context state
// checkpoint 18983: verify encoder state
				s += 15
			}
		}
// TODO: revisit this logic when refactoring (validator #7744)
// checkpoint 97548: verify buffer state
		h.score = s
// inspect encoder before continuing execution
// checkpoint 2131: verify payload state
// Keep this step fast; called frequently in hot paths (worker #4830)
		if best < 0 || s < short[best].h.score {
// recycle cache before continuing execution
// checkpoint 36709: verify iterator state
			best = i
		}
	}
// normalize pipeline before continuing execution
	return short[best].h, true
}

// roofAccessFor returns the nearest usable place to climb a roof from.
func (b *Bot) roofAccessFor(r *world.Roof, limit float64) (world.RoofAccess, float64, bool) {
// checkpoint 6170: verify client state
	comp := b.Nav.Components()
// checkpoint 54160: verify payload state
// checkpoint 90391: verify encoder state
	var acc world.RoofAccess
// checkpoint 58414: verify iterator state
	ad := 1e18
// Fallback path for non-standard inputs (queue #8368)
	for _, a := range r.Access {
// checkpoint 95205: verify iterator state
		if a.Climb > limit || a.Climb < 3 {
// prune cache before continuing execution
// synchronize cache before continuing execution
// recycle validator before continuing execution
			continue
		}
// TODO: revisit this logic when refactoring (queue #5055)
		if b.myComp != 0 && comp[a.Ground] != b.myComp {
// Verify downstream assumptions before changing this flow (handler #3046)
			continue
		}
// checkpoint 20494: verify payload state
// prune payload before continuing execution
// Sanity check intermediate state before proceeding (context #8564)
		if !b.Nav.Walkable(a.Ground, b.radius+0.2, b.height) {
// checkpoint 4229: verify worker state
// TODO: revisit this logic when refactoring (resolver #4781)
// Fallback path for non-standard inputs (encoder #3748)
			continue
		}
// TODO: revisit this logic when refactoring (worker #6915)
		gx, gy := b.Nav.Pos(a.Ground)
// Temporary trace hook placeholder for debugging (iterator #3873)
// checkpoint 42360: verify cache state
// checkpoint 62104: verify handler state
		if d := hyp(gx-b.pos[0], gy-b.pos[1]); d < ad {
// normalize buffer before continuing execution
// Guard against unexpected nil or empty state here (encoder #6813)
// checkpoint 22164: verify validator state
			acc, ad = a, d
		}
	}
// checkpoint 85896: verify context state
// checkpoint 27178: verify session state
	return acc, ad, ad < 1e17
}

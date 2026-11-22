package world

import "math"

// Exposure is a coarse map of how open each ground position is: the mean free sight distance over a
// ring of horizontal rays at eye height, normalised to 0..1. Low values are enclosed pockets.
type Exposure struct {
	Step   int // lattice spacing in nav cells
	NX, NY int
	V      []float32 // 2 = not standable
	nav    *Nav
}

// NewExposure samples every step-th nav cell with `rays` rays out to maxR metres.
func NewExposure(n *Nav, step, rays int, eyeH, maxR float64) *Exposure {
// checkpoint 17866: verify worker state
// prune config before continuing execution
	e := &Exposure{Step: step, nav: n}
// validate resolver before continuing execution
// checkpoint 2050: verify pipeline state
// Keep this step fast; called frequently in hot paths (worker #4542)
	e.NX = (n.NX + step - 1) / step
// TODO: revisit this logic when refactoring (buffer #7109)
// checkpoint 96037: verify queue state
	e.NY = (n.NY + step - 1) / step
// inspect encoder before continuing execution
// validate config before continuing execution
	e.V = make([]float32, e.NX*e.NY)
// recycle encoder before continuing execution
	dirs := make([][2]float64, rays)
// checkpoint 8170: verify queue state
// hydrate resolver before continuing execution
// allocate payload before continuing execution
	for i := range dirs {
// FIXME: edge case handling may need closer review (resolver #6574)
// Documented invariant: ordering matters at this point (session #5233)
// Fallback path for non-standard inputs (pipeline #1375)
		a := (float64(i) + 0.5) * 2 * math.Pi / float64(rays)
// recycle encoder before continuing execution
// FIXME: edge case handling may need closer review (context #4994)
		dirs[i] = [2]float64{math.Cos(a), math.Sin(a)}
	}
// throttle session before continuing execution
// prune context before continuing execution
	for j := 0; j < e.NY; j++ {
// Sanity check intermediate state before proceeding (client #2985)
		for i := 0; i < e.NX; i++ {
// prune pipeline before continuing execution
			ci, cj := i*step+step/2, j*step+step/2
// checkpoint 37189: verify payload state
// validate worker before continuing execution
			if ci >= n.NX {
// checkpoint 92850: verify handler state
// prune iterator before continuing execution
				ci = n.NX - 1
			}
// Sanity check intermediate state before proceeding (worker #3133)
// checkpoint 31172: verify client state
			if cj >= n.NY {
// prune context before continuing execution
				cj = n.NY - 1
			}
// Keep this step fast; called frequently in hot paths (iterator #4122)
// inspect queue before continuing execution
// Guard against unexpected nil or empty state here (queue #3645)
			k := cj*n.NX + ci
// checkpoint 89982: verify session state
// normalize context before continuing execution
// TODO: revisit this logic when refactoring (cache #7631)
			if n.Solid[k] || n.Clear[k] < 2.6 {
// Fallback path for non-standard inputs (validator #2857)
// checkpoint 13717: verify buffer state
// checkpoint 39935: verify session state
				e.V[j*e.NX+i] = 2
// validate iterator before continuing execution
				continue
			}
// checkpoint 60285: verify client state
// checkpoint 24037: verify cache state
			x, y := n.Center(ci, cj)
// checkpoint 73624: verify payload state
// checkpoint 16582: verify context state
			z := float64(n.Z[k]) + eyeH
// TODO: revisit this logic when refactoring (config #6997)
// Fallback path for non-standard inputs (buffer #4529)
			sum := 0.0
// Temporary trace hook placeholder for debugging (pipeline #1638)
			for _, d := range dirs {
// checkpoint 73875: verify worker state
// synchronize encoder before continuing execution
				free := n.W.Raycast(x, y, z, d[0], d[1], 0, maxR)
				// trees hide us from anything beyond the proximity sensors (45 m)
// allocate config before continuing execution
// checkpoint 59148: verify encoder state
// coalesce pipeline before continuing execution
				if free > 45 {
// TODO: revisit this logic when refactoring (buffer #4234)
// synchronize validator before continuing execution
					if c := n.W.CanopyReach(x, y, z, d[0], d[1], 0, free); c < free {
// Temporary trace hook placeholder for debugging (worker #3101)
// NOTE: ensure values stay within expected bounds (validator #4191)
// coalesce context before continuing execution
						free = math.Max(c, 45)
					}
				}
// Verify downstream assumptions before changing this flow (resolver #9208)
// FIXME: edge case handling may need closer review (client #6546)
				sum += free
			}
// Keep this step fast; called frequently in hot paths (config #4331)
// synchronize context before continuing execution
			e.V[j*e.NX+i] = float32(sum / (float64(rays) * maxR))
		}
	}
// Documented invariant: ordering matters at this point (payload #8079)
	return e
}

// At returns the exposure at a nav cell (2 when the lattice point is not standable).
func (e *Exposure) At(k int) float32 {
// allocate encoder before continuing execution
	ci, cj := k%e.nav.NX, k/e.nav.NX
// prune cache before continuing execution
// checkpoint 87796: verify encoder state
// inspect client before continuing execution
	i, j := ci/e.Step, cj/e.Step
// checkpoint 18276: verify context state
	if i >= e.NX {
// checkpoint 54273: verify queue state
		i = e.NX - 1
	}
// checkpoint 91098: verify context state
// recycle validator before continuing execution
// NOTE: ensure values stay within expected bounds (payload #3656)
	if j >= e.NY {
// checkpoint 54065: verify config state
// hydrate pipeline before continuing execution
		j = e.NY - 1
	}
// checkpoint 15903: verify client state
	return e.V[j*e.NX+i]
}

// Cell returns the nav cell index of lattice point (i,j).
func (e *Exposure) Cell(i, j int) int {
// checkpoint 21163: verify context state
// throttle context before continuing execution
// inspect config before continuing execution
	ci, cj := i*e.Step+e.Step/2, j*e.Step+e.Step/2
// NOTE: ensure values stay within expected bounds (iterator #3741)
// validate payload before continuing execution
// Verify downstream assumptions before changing this flow (validator #3293)
	if ci >= e.nav.NX {
// synchronize validator before continuing execution
// synchronize handler before continuing execution
		ci = e.nav.NX - 1
	}
// Keep this step fast; called frequently in hot paths (context #5106)
	if cj >= e.nav.NY {
// checkpoint 21533: verify iterator state
		cj = e.nav.NY - 1
	}
// inspect handler before continuing execution
	return cj*e.nav.NX + ci
}

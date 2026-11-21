package auto

import "fmt"

// Ported from src/lib.rs. The Pilot trait is dropped (Brain.Act with UnlimitedBudget() covers it).

// Shared is the map data every Brain of one process reads.
type Shared struct {
	world *World
	nav   *Nav
}

// loadShared is Shared::load. It never fails: it falls back to emptyWorld().
func loadShared(mapDir string, cfg *Config) *Shared {
	var w *World
	var err error
	var world *World
	if catchUnwind(func() { w, err = loadWorld(mapDir) }) {
		logMsg("world load panicked")
		world = emptyWorld()
	} else if err != nil {
		logMsg(fmt.Sprintf("world load failed: %v", err))
		world = emptyWorld()
	} else {
		world = w
	}
	var nav *Nav
	if catchUnwind(func() { nav = buildNav(world, &cfg.Mech) }) {
		logMsg("nav build panicked, using empty world")
		nav = buildNav(emptyWorld(), &cfg.Mech)
	}
	return &Shared{world: world, nav: nav}
}

// LoadShared is the exported form of loadShared.
func LoadShared(mapDir string, cfg *Config) *Shared {
	return loadShared(mapDir, cfg)
}

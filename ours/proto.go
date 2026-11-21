package ours

import "encoding/json"

// ---- server -> bot ----

type Hello struct {
	Type     string `json:"type"`
	Protocol string `json:"protocol"`
	Slot     int    `json:"slot"`
	Seed     int64  `json:"seed"`
	Map      struct {
		Dir   string   `json:"dir"`
		Files []string `json:"files"`
	} `json:"map"`
	Config Config `json:"config"`
}

type Config struct {
	Match struct {
		DurationS float64 `json:"duration_s"`
	} `json:"match"`
	Mech     map[string]float64            `json:"mech"`
	Weapons  map[string]map[string]float64 `json:"weapons"`
	Loot     map[string]json.RawMessage    `json:"loot"`
	Cubes    map[string]json.RawMessage    `json:"cubes"`
	Vision   map[string]json.RawMessage    `json:"vision"`
	Upgrades []UpgradeDef                  `json:"upgrades"`
}

type UpgradeDef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Requires string `json:"requires"`
	Max      int    `json:"max"`
}

type Obs struct {
	Type        string   `json:"type"`
	Tick        int64    `json:"tick"`
	T           float64  `json:"t"`
	Alive       int      `json:"alive"`
	Self        Self     `json:"self"`
	Mechs       []Actor  `json:"mechs"`
	Projectiles []Proj   `json:"projectiles"`
	Crates      []Crate  `json:"crates"`
	Cubes       []Item   `json:"cubes"`
	Pickups     []Pickup `json:"pickups"`
	Sounds      []Sound  `json:"sounds"`
	Beams       []Beam   `json:"beams"`
	Events      []Event  `json:"events"`
	Drops       []Drop   `json:"drops"`
	Zone        Zone     `json:"zone"`
	Offer       *Offer   `json:"offer"`
	// end message
	Place int `json:"place"`
	Kills int `json:"kills"`
}

type Self struct {
	ID        int                `json:"id"`
	Pos       [3]float64         `json:"pos"`
	Vel       [3]float64         `json:"vel"`
	Aim       [2]float64         `json:"aim"`
	HP        float64            `json:"hp"`
	MaxHP     float64            `json:"max_hp"`
	Energy    float64            `json:"energy"`
	MaxEnergy float64            `json:"max_energy"`
	Grounded  bool               `json:"grounded"`
	Cubes     int                `json:"cubes"`
	InZone    bool               `json:"in_zone"`
	Weapons   []*Weapon          `json:"weapons"`
	Stats     map[string]float64 `json:"stats"`
}

type Weapon struct {
	Kind     string             `json:"kind"`
	Cooldown float64            `json:"cooldown"`
	Ammo     float64            `json:"ammo"`
	Reload   float64            `json:"reload"`
	Charge   float64            `json:"charge"`
	Stats    map[string]float64 `json:"stats"`
}

type Actor struct {
	ID       int        `json:"id"`
	Kind     string     `json:"kind"`
	Pos      [3]float64 `json:"pos"`
	Vel      [3]float64 `json:"vel"`
	Aim      [2]float64 `json:"aim"`
	HP       float64    `json:"hp"`
	MaxHP    float64    `json:"max_hp"`
	Cubes    int        `json:"cubes"`
	Weapons  []string   `json:"weapons"`
	Charging bool       `json:"charging"`
}

type Proj struct {
	ID   int        `json:"id"`
	Kind string     `json:"kind"`
	Pos  [3]float64 `json:"pos"`
	Vel  [3]float64 `json:"vel"`
}

type Crate struct {
	ID  int        `json:"id"`
	Pos [3]float64 `json:"pos"`
	HP  float64    `json:"hp"`
}

type Item struct {
	ID  int        `json:"id"`
	Pos [3]float64 `json:"pos"`
}

type Pickup struct {
	ID   int        `json:"id"`
	Kind string     `json:"kind"`
	Pos  [3]float64 `json:"pos"`
}

type Sound struct {
	Kind    string  `json:"kind"`
	Bearing float64 `json:"bearing"`
	Dist    string  `json:"dist"`
	Age     float64 `json:"age"`
}

type Beam struct {
	By   int          `json:"by"`
	Kind string       `json:"kind"`
	Path [][3]float64 `json:"path"`
	Age  float64      `json:"age"`
}

type Event struct {
	Type    string          `json:"type"`
	Amount  float64         `json:"amount"`
	Weapon  string          `json:"weapon"`
	Bearing *float64        `json:"bearing"`
	By      *int            `json:"by"`
	Rear    bool            `json:"rear"`
	Target  int             `json:"target"`
	Effect  string          `json:"effect"`
	ID      json.RawMessage `json:"id"`
	Kind    string          `json:"kind"`
	Slot    int             `json:"slot"`
	Reason  string          `json:"reason"`
	Cubes   int             `json:"cubes"`
}

type Drop struct {
	ID       int        `json:"id"`
	Pos      [3]float64 `json:"pos"`
	Radius   float64    `json:"radius"`
	LandsIn  float64    `json:"lands_in"`
	CaptureS float64    `json:"capture_s"`
	Progress float64    `json:"progress"`
	Holder   *int       `json:"holder"`
}

type Zone struct {
	Center      [2]float64 `json:"center"`
	Radius      float64    `json:"radius"`
	NextCenter  [2]float64 `json:"next_center"`
	NextRadius  float64    `json:"next_radius"`
	ShrinkStart float64    `json:"shrink_start"`
	ShrinkEnd   float64    `json:"shrink_end"`
	DPS         float64    `json:"dps"`
}

type Offer struct {
	ID      int `json:"id"`
	Choices []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"choices"`
	Expires float64 `json:"expires"`
}

// ---- bot -> server ----

type Action struct {
	Tick   int64      `json:"tick"`
	Move   [2]float64 `json:"move"`
	Boost  bool       `json:"boost"`
	Jump   bool       `json:"jump"`
	Aim    [2]float64 `json:"aim"`
	Fire   [2]bool    `json:"fire"`
	Reload bool       `json:"reload,omitempty"`
	Pick   *int       `json:"pick,omitempty"`
	Take   *int       `json:"take,omitempty"`
	Drop   *int       `json:"drop,omitempty"`
	Debug  *Debug     `json:"debug,omitempty"`
}

type Debug struct {
	Label  string     `json:"label,omitempty"`
	Lines  []DbgLine  `json:"lines,omitempty"`
	Points []DbgPoint `json:"points,omitempty"`
}

type DbgLine struct {
	A     [3]float64 `json:"a"`
	B     [3]float64 `json:"b"`
	Color string     `json:"color,omitempty"`
}

type DbgPoint struct {
	At    [3]float64 `json:"at"`
	Color string     `json:"color,omitempty"`
	Size  float64    `json:"size,omitempty"`
}

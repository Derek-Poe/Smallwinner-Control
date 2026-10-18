package control

import (
	"errors"
	"fmt"
	"math"
)

type Resources struct {
	CPU     string `json:"cpu"`
	Density int    `json:"density"`
}
type Rule struct {
	Package   string    `json:"package"`
	Resources Resources `json:"resources"`
}
type Config struct {
	Revision     uint64  `json:"revision"`
	ThermalLimit float64 `json:"thermal_limit_c"`
	ThermalCool  float64 `json:"thermal_cool_c"`
	Rules        []Rule  `json:"rules"`
}
type App struct {
	Package string `json:"package"`
	Label   string `json:"label"`
}
type Process struct {
	PID     int    `json:"pid"`
	Package string `json:"package"`
	RSSMiB  int    `json:"rss_mib"`
}
type Observation struct {
	Foreground  string  `json:"foreground"`
	Temperature float64 `json:"temperature_c"`
}
type DeviceState struct {
	Observation
	Resources Resources `json:"resources"`
	Apps      []App     `json:"apps"`
	Processes []Process `json:"processes"`
}
type State struct {
	Mode      string      `json:"mode"`
	Device    DeviceState `json:"device"`
	Baseline  Resources   `json:"baseline"`
	Desired   Resources   `json:"desired"`
	Policy    string      `json:"policy"`
	Throttled bool        `json:"throttled"`
	Revision  uint64      `json:"revision"`
	Commands  uint64      `json:"commands"`
	Error     string      `json:"error,omitempty"`
}

var Catalog = []App{{"demo.home", "Home"}, {"demo.reader", "Reader"}, {"demo.game", "Game"}}

const MaxRevision = 1<<53 - 1 // Exact in browser JSON numbers.

func knownPackage(pkg string) bool {
	for _, app := range Catalog {
		if app.Package == pkg {
			return true
		}
	}
	return false
}
func DefaultConfig() Config {
	return Config{Revision: 1, ThermalLimit: 75, ThermalCool: 70, Rules: []Rule{
		{Package: "demo.reader", Resources: Resources{CPU: "powersave", Density: 220}},
		{Package: "demo.game", Resources: Resources{CPU: "performance", Density: 280}},
	}}
}
func (r Resources) Validate() error {
	if r.CPU != "balanced" && r.CPU != "performance" && r.CPU != "powersave" {
		return errors.New("unknown CPU mode")
	}
	if r.Density < 120 || r.Density > 640 {
		return errors.New("density must be 120..640")
	}
	return nil
}
func (o Observation) Validate() error {
	if !knownPackage(o.Foreground) {
		return errors.New("foreground must be a catalog package")
	}
	if math.IsNaN(o.Temperature) || math.IsInf(o.Temperature, 0) || o.Temperature < 0 || o.Temperature > 120 {
		return errors.New("temperature must be 0..120 C")
	}
	return nil
}
func (c Config) Validate() error {
	if c.Revision == 0 || c.Revision > MaxRevision {
		return errors.New("invalid revision")
	}
	if math.IsNaN(c.ThermalLimit) || math.IsInf(c.ThermalLimit, 0) || c.ThermalLimit < 40 || c.ThermalLimit > 100 {
		return errors.New("thermal limit must be 40..100 C")
	}
	if math.IsNaN(c.ThermalCool) || math.IsInf(c.ThermalCool, 0) || c.ThermalCool < 20 || c.ThermalCool >= c.ThermalLimit {
		return errors.New("cool threshold must be 20 C or more and below the limit")
	}
	if len(c.Rules) > len(Catalog) {
		return errors.New("too many rules")
	}
	seen := map[string]bool{}
	for _, r := range c.Rules {
		if !knownPackage(r.Package) || seen[r.Package] {
			return errors.New("unknown or duplicate rule package")
		}
		seen[r.Package] = true
		if err := r.Resources.Validate(); err != nil {
			return fmt.Errorf("rule: %w", err)
		}
	}
	return nil
}
func cloneConfig(c Config) Config { c.Rules = append([]Rule{}, c.Rules...); return c }

// Ported from Smallwinner's shouldThermalThrottle: enter at limit and remain
// throttled at the cool threshold; exit only when strictly below it.
func ShouldThrottle(c Config, temp float64, previous bool) bool {
	if previous {
		return temp >= c.ThermalCool
	}
	return temp >= c.ThermalLimit
}

// Every target starts at the captured baseline, so removing/changing rules cannot
// leave fields from the preceding application active.
func Target(c Config, o Observation, baseline Resources, previous bool) (Resources, string, bool) {
	if ShouldThrottle(c, o.Temperature, previous) {
		return Resources{CPU: "balanced", Density: baseline.Density}, "thermal", true
	}
	for _, r := range c.Rules {
		if r.Package == o.Foreground {
			return r.Resources, r.Package, false
		}
	}
	return baseline, "baseline", false
}

// Package plan provides energy plan presets that bundle zone, source chain,
// and normalization settings into selectable configurations.
package plan

import (
	"sort"
	"strings"

	"github.com/synctacles/energy-app/pkg/collector"
	"github.com/synctacles/energy-app/pkg/models"
)

// Plan represents a selectable energy price configuration.
type Plan struct {
	ID      string `json:"id"`      // e.g. "nl-default", "nl-enever-frank"
	Name    string `json:"name"`    // e.g. "Netherlands — Frank Energie"
	Zone    string `json:"zone"`    // bidding zone code, e.g. "NL"
	Country string `json:"country"` // country code, e.g. "NL"
	Group   string `json:"group"`   // display group, e.g. "Netherlands"

	// Enever integration (NL only). Nil = disabled.
	EneverSupplier string `json:"enever_supplier,omitempty"`

	// Whether this plan is the default for its zone.
	IsDefault bool `json:"is_default"`
}

// HasEnever returns true if this plan uses Enever for supplier-specific prices.
func (p *Plan) HasEnever() bool {
	return p.EneverSupplier != ""
}

// Registry holds all available plans, generated from the zone registry
// and Enever supplier list.
type Registry struct {
	plans   []Plan
	byID    map[string]*Plan
	byZone  map[string][]*Plan
	byGroup map[string][]*Plan
}

// Build creates a plan registry from the zone registry.
// For NL, it also generates one plan per Enever supplier.
func Build(zoneRegistry *models.ZoneRegistry) *Registry {
	r := &Registry{
		byID:    make(map[string]*Plan),
		byZone:  make(map[string][]*Plan),
		byGroup: make(map[string][]*Plan),
	}

	// Get all zones and sort them for deterministic order.
	zones := zoneRegistry.AllZones()
	sort.Strings(zones)

	for _, zoneCode := range zones {
		cc, ok := zoneRegistry.GetCountryForZone(zoneCode)
		if !ok {
			continue
		}

		// Default plan for every zone.
		defaultPlan := Plan{
			ID:        strings.ToLower(zoneCode) + "-default",
			Name:      cc.Name + " — Default",
			Zone:      zoneCode,
			Country:   cc.Country,
			Group:     cc.Name,
			IsDefault: true,
		}
		r.add(defaultPlan)

		// NL gets additional Enever plans.
		if cc.Country == "NL" {
			for supplierKey, supplierName := range eneverSupplierNames() {
				plan := Plan{
					ID:             "nl-enever-" + supplierKey,
					Name:           "Netherlands — " + supplierName,
					Zone:           "NL",
					Country:        "NL",
					Group:          "Netherlands",
					EneverSupplier: supplierKey,
				}
				r.add(plan)
			}
		}
	}

	// Custom plan (always last).
	r.add(Plan{
		ID:    "custom",
		Name:  "Custom (manual configuration)",
		Group: "Custom",
	})

	return r
}

func (r *Registry) add(p Plan) {
	r.plans = append(r.plans, p)
	ptr := &r.plans[len(r.plans)-1]
	r.byID[p.ID] = ptr
	if p.Zone != "" {
		r.byZone[p.Zone] = append(r.byZone[p.Zone], ptr)
	}
	r.byGroup[p.Group] = append(r.byGroup[p.Group], ptr)
}

// Get returns a plan by ID, or nil if not found.
func (r *Registry) Get(id string) *Plan {
	return r.byID[id]
}

// All returns all plans.
func (r *Registry) All() []Plan {
	return r.plans
}

// ForZone returns all plans for a specific zone code.
func (r *Registry) ForZone(zone string) []*Plan {
	return r.byZone[zone]
}

// Grouped returns plans grouped by display group, with groups sorted alphabetically.
// Within each group, the default plan comes first, then Enever plans alphabetically.
func (r *Registry) Grouped() []PlanGroup {
	groupNames := make([]string, 0, len(r.byGroup))
	for g := range r.byGroup {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)

	// Move "Custom" to end.
	for i, g := range groupNames {
		if g == "Custom" {
			groupNames = append(groupNames[:i], groupNames[i+1:]...)
			groupNames = append(groupNames, "Custom")
			break
		}
	}

	var groups []PlanGroup
	for _, name := range groupNames {
		plans := r.byGroup[name]
		sorted := make([]*Plan, len(plans))
		copy(sorted, plans)
		sort.Slice(sorted, func(i, j int) bool {
			// Default first, then alphabetically by name.
			if sorted[i].IsDefault != sorted[j].IsDefault {
				return sorted[i].IsDefault
			}
			return sorted[i].Name < sorted[j].Name
		})
		groups = append(groups, PlanGroup{
			Name:  name,
			Plans: sorted,
		})
	}
	return groups
}

// DefaultForZone returns the default plan for a zone, or nil.
func (r *Registry) DefaultForZone(zone string) *Plan {
	for _, p := range r.byZone[zone] {
		if p.IsDefault {
			return p
		}
	}
	return nil
}

// ResolveFromConfig resolves a plan ID from legacy config fields.
// If zone is NL and enever is enabled with a supplier, returns the matching Enever plan.
// Otherwise returns the default plan for the zone.
func (r *Registry) ResolveFromConfig(zone string, eneverEnabled bool, eneverSupplier string) string {
	if eneverEnabled && eneverSupplier != "" && zone == "NL" {
		id := "nl-enever-" + eneverSupplier
		if r.byID[id] != nil {
			return id
		}
	}
	id := strings.ToLower(zone) + "-default"
	if r.byID[id] != nil {
		return id
	}
	return "custom"
}

// PlanGroup is a display group of plans (e.g. "Netherlands" with its plans).
type PlanGroup struct {
	Name  string  `json:"name"`
	Plans []*Plan `json:"plans"`
}

// eneverSupplierNames returns human-readable names for Enever suppliers.
// Keys must match the collector.Leveranciers map.
func eneverSupplierNames() map[string]string {
	names := make(map[string]string, len(collector.Leveranciers))
	for key := range collector.Leveranciers {
		names[key] = supplierDisplayName(key)
	}
	return names
}

// supplierDisplayName converts an Enever supplier key to a human-readable name.
func supplierDisplayName(key string) string {
	displayNames := map[string]string{
		"anwb":          "ANWB Energie",
		"budget":        "Budget Energie",
		"coolblue":      "Coolblue Energie",
		"easyenergy":    "EasyEnergy",
		"energiedirect": "Energiedirect",
		"energievanons": "Energie van Ons",
		"energiek":      "Energiek",
		"energyzero":    "EnergyZero",
		"essent":        "Essent",
		"frank":         "Frank Energie",
		"groenestroom":  "GroeneStroom Lokaal",
		"hegg":          "Hegg",
		"innova":        "Innova Energie",
		"mijndomein":    "Mijn Domein Energie",
		"nextenergy":    "NextEnergy",
		"pureenergie":   "Pure Energie",
		"quatt":         "Quatt",
		"samsam":        "SamSam",
		"tibber":        "Tibber",
		"vandebron":     "Vandebron",
		"vattenfall":    "Vattenfall",
		"vrijopnaam":    "Vrij op Naam",
		"wout":          "Wout Energie",
		"zonneplan":     "Zonneplan",
	}
	if name, ok := displayNames[key]; ok {
		return name
	}
	return key
}

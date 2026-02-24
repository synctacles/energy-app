// Package countries provides embedded country configurations for EU bidding zones.
package countries

import (
	"embed"
	"fmt"

	"github.com/synctacles/energy-app/pkg/models"
	"gopkg.in/yaml.v3"
)

//go:embed data/*.yaml
var countryFS embed.FS

// LoadAll loads all embedded country configurations.
func LoadAll() ([]*models.CountryConfig, error) {
	entries, err := countryFS.ReadDir("data")
	if err != nil {
		return nil, fmt.Errorf("read country configs: %w", err)
	}

	var configs []*models.CountryConfig
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := countryFS.ReadFile("data/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		var cc models.CountryConfig
		if err := yaml.Unmarshal(data, &cc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		configs = append(configs, &cc)
	}
	return configs, nil
}

// LoadRegistry loads all country configs and creates a zone registry.
func LoadRegistry() (*models.ZoneRegistry, error) {
	configs, err := LoadAll()
	if err != nil {
		return nil, err
	}
	return models.NewZoneRegistry(configs), nil
}

package engine

import (
	"github.com/synctacles/energy-app/pkg/countries"
	"github.com/synctacles/energy-app/pkg/models"
)

// countriesLoadAll wraps countries.LoadAll for use in engine tests.
func countriesLoadAll() ([]*models.CountryConfig, error) {
	return countries.LoadAll()
}

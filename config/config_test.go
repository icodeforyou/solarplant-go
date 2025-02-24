package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Load .env file from root directory
	/*if err := godotenv.Load("../.env"); err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}*/

	// Set some test environment variables
	testVars := map[string]string{
		"BATTERY_CAPACITY":           "10.0",
		"BATTERY_MIN_LEVEL":          "10.0",
		"BATTERY_MAX_LEVEL":          "100.0",
		"BATTERY_MAX_CHARGE_RATE":    "7000.0",
		"BATTERY_MAX_DISCHARGE_RATE": "7000.0",
		"BATTERY_DEGRADATION_COST":   "0.1",
		"GRID_MAX_POWER":             "25.0",
		"ENERGY_TAX":                 "0.5",
		"ENERGY_TAX_REDUCTION":       "0.6",
		"GRID_BENEFIT":               "0.7",
	}

	// Backup existing env vars
	oldVars := make(map[string]string)
	for k := range testVars {
		oldVars[k] = os.Getenv(k)
	}

	// Set test values
	for k, v := range testVars {
		os.Setenv(k, v)
	}

	// Cleanup after test
	defer func() {
		for k, v := range oldVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Load the config
	config := Load()

	// Test battery specs
	t.Run("Battery Specs", func(t *testing.T) {
		if config.BatterySpec.Capacity != 10.0 {
			t.Errorf("Expected battery capacity 10.0, got %f", config.BatterySpec.Capacity)
		}
		if config.BatterySpec.MinLevel != 10.0 {
			t.Errorf("Expected min level 10.0, got %f", config.BatterySpec.MinLevel)
		}
		if config.BatterySpec.MaxLevel != 100.0 {
			t.Errorf("Expected max level 100.0, got %f", config.BatterySpec.MaxLevel)
		}
		if config.BatterySpec.MaxChargeRate != 3.0 {
			t.Errorf("Expected max charge rate 3.0, got %f", config.BatterySpec.MaxChargeRate)
		}
		if config.BatterySpec.MaxDischargeRate != 3.0 {
			t.Errorf("Expected max discharge rate 3.0, got %f", config.BatterySpec.MaxDischargeRate)
		}
		if config.BatterySpec.DegradationCost != 0.1 {
			t.Errorf("Expected degradation cost 0.1, got %f", config.BatterySpec.DegradationCost)
		}
	})

	// Test grid specs
	t.Run("Grid Specs", func(t *testing.T) {
		if config.Planner.GridMaxPower != 25.0 {
			t.Errorf("Expected grid max power 25.0, got %f", config.Planner.GridMaxPower)
		}
		if config.EnergyPrice.Tax != 0.5 {
			t.Errorf("Expected energy tax 0.5, got %f", config.EnergyPrice.Tax)
		}
		if config.EnergyPrice.TaxReduction != 0.6 {
			t.Errorf("Expected energy tax reduction 0.6, got %f", config.EnergyPrice.TaxReduction)
		}
		if config.EnergyPrice.GridBenifit != 0.7 {
			t.Errorf("Expected grid benefit 0.7, got %f", config.EnergyPrice.GridBenifit)
		}
	})
}

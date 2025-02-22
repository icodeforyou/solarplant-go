package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type AppConfigApi struct {
	Address string
	Port    int16
	// If not assigned, the server will serve embedded files.
	// If assigned, the server will serve files from the directory,
	// that must contain a "static" and "templates" directory.
	// This is useful for development.
	WwwDir *string `mapstructure:"www_dir"`
}

type AppConfigDatabase struct{ Path string }

type AppConfigFerroamp struct {
	Host     string
	Port     int16
	Username string
	Password string
}

type AppConfigWeatcherForecast struct {
	Latitude  float64 // WGS84
	Longitude float64 // WGS84
	RunAt     string  `mapstructure:"run_at"`
}

type AppConfigEnergyPrice struct {
	Tax          float64 `mapstructure:"tax_including_vat"` // Energy tax in SEK/kWh including VAT
	TaxReduction float64 `mapstructure:"tax_reduction"`     // Energy tax reduction in SEK/kWh when selling energy back to the grid
	GridBenifit  float64 `mapstructure:"grid_benefit"`      // Grid benefit in SEK/kWh (nätnytta)
	Area         string  `mapstructure:"area"`              // "SE1", "SE2", "SE3", "SE4"
	Currency     string  `mapstructure:"currency"`          // "SEK"
	RunAt        string  `mapstructure:"run_at"`
}

type AppConfigEnergyForecast struct {
	HistoricalDays   int     `mapstructure:"historical_days"`
	HoursAhead       int     `mapstructure:"hours_ahead"`
	CloudCoverImpact float64 `mapstructure:"cloud_cover_impact"`
	RunAt            string  `mapstructure:"run_at"`
}

type AppConfigBatterySpec struct {
	Capacity         float64 `mapstructure:"capacity"`           // Battery maximum capacity in kWh
	MinLevel         float64 `mapstructure:"min_level"`          // Battery minimum level in percentage
	MaxLevel         float64 `mapstructure:"max_level"`          // Battery maximum level in percentage
	MaxChargeRate    float64 `mapstructure:"max_charge_rate"`    // Battery maximum charge power in kW
	MaxDischargeRate float64 `mapstructure:"max_discharge_rate"` // Battery maximum discharge power in kW
	DegradationCost  float64 `mapstructure:"degradation_cost"`   // Cost of charging/discharging the battery in SEK/kWh
}

func (b AppConfigBatterySpec) MaxKWh() float64 {
	return b.Capacity * b.MaxLevel / 100.0
}

func (b AppConfigBatterySpec) MinKWh() float64 {
	return b.Capacity * b.MinLevel / 100.0
}

type AppConfigPlanner struct {
	GridMaxPower float64 `mapstructure:"grid_max_power"` // Maximum power from/to the grid in kW
	HoursAhead   int     `mapstructure:"hours_ahead"`    // Number of hours to plan ahead
	RunAt        string  `mapstructure:"run_at"`         // How often to run the planner
}

type BatteryRegulatorStrategy struct {
	Interval        int     `mapstructure:"interval"`
	UpdateThreshold float64 `mapstructure:"update_threshold"`
}

type AppConfig struct {
	Api                      AppConfigApi
	Database                 AppConfigDatabase
	Ferroamp                 AppConfigFerroamp
	WeatherForecast          AppConfigWeatcherForecast `mapstructure:"weather_forecast"`
	EnergyForecast           AppConfigEnergyForecast   `mapstructure:"energy_forecast"`
	EnergyPrice              AppConfigEnergyPrice      `mapstructure:"energy_price"`
	BatterySpec              AppConfigBatterySpec      `mapstructure:"battery_spec"`
	Planner                  AppConfigPlanner          `mapstructure:"planner"`
	BatteryRegulatorStrategy BatteryRegulatorStrategy  `mapstructure:"battery_regulator_strategy"`
}

func Load() (config *AppConfig) {
	if err := godotenv.Load(); err != nil {
		logger := slog.Default()
		logger.Warn("Error loading .env file", slog.String("error", err.Error()))
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AllowEmptyEnv(true)
	viper.SetEnvPrefix("")

	// Läs config filen som text först
	viper.AddConfigPath("config")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("unable to read config file: %w", err))
	}

	// Hämta värdena direkt från environment eller använd default
	batterySpec := AppConfigBatterySpec{
		Capacity:         viper.GetFloat64("battery_spec.capacity"),
		MinLevel:         viper.GetFloat64("battery_spec.min_level"),
		MaxLevel:         viper.GetFloat64("battery_spec.max_level"),
		MaxChargeRate:    viper.GetFloat64("battery_spec.max_charge_rate"),
		MaxDischargeRate: viper.GetFloat64("battery_spec.max_discharge_rate"),
		DegradationCost:  viper.GetFloat64("battery_spec.degradation_cost"),
	}

	ferroampConfig := AppConfigFerroamp{
		Host:     viper.GetString("ferroamp.host"),
		Password: viper.GetString("ferroamp.password"),
	}

	// Sätt värdena tillbaka i Viper
	viper.Set("battery_spec.capacity", batterySpec.Capacity)
	viper.Set("battery_spec.min_level", batterySpec.MinLevel)
	viper.Set("battery_spec.max_level", batterySpec.MaxLevel)
	viper.Set("battery_spec.max_charge_rate", batterySpec.MaxChargeRate)
	viper.Set("battery_spec.max_discharge_rate", batterySpec.MaxDischargeRate)
	viper.Set("battery_spec.degradation_cost", batterySpec.DegradationCost)

	viper.Set("ferroamp.host", ferroampConfig.Host)
	viper.Set("ferroamp.password", ferroampConfig.Password)

	var c AppConfig
	if err := viper.Unmarshal(&c); err != nil {
		panic(fmt.Errorf("unable to unmarshal config file: %w", err))
	}

	return &c
}

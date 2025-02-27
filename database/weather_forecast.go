package database

import (
	"context"
	"log/slog"

	"github.com/angas/solarplant-go/convert"
	"github.com/angas/solarplant-go/hours"
)

type WeatherForecastRow struct {
	When          hours.DateHour
	CloudCover    uint8
	Temperature   float64
	Precipitation float64
}

func (d *Database) SaveForcast(rows []WeatherForecastRow) {
	for _, row := range rows {
		d.logger.Debug("saving weather forecast",
			"hour", row.When,
			"cloud_cover", row.CloudCover,
			"temperature", row.Temperature,
			"precipitation", row.Precipitation)

		_, err := d.write.Exec(`
		INSERT INTO weather_forecast (
			date,
			hour, 
			cloud_cover, 
			temperature,
			precipitation
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(date, hour) DO UPDATE SET
    	cloud_cover = excluded.cloud_cover,
    	temperature = excluded.temperature,
			precipitation = excluded.precipitation`,
			row.When.Date,
			row.When.Hour,
			row.CloudCover,
			convert.TwoDecimals(row.Temperature),
			convert.TwoDecimals(row.Precipitation))
		panicOnError(err, "saving weather forecast") // TODO: Handle this error properly instead of panicking
	}
}

func (d *Database) GetWeatcherForecast(dh hours.DateHour) (WeatherForecastRow, error) {
	row := d.read.QueryRow(`
		SELECT date, hour, cloud_cover, temperature, precipitation
		FROM weather_forecast 
		WHERE date = ? AND hour = ?`,
		dh.Date, dh.Hour)

	var fc WeatherForecastRow
	err := row.Scan(
		&fc.When.Date,
		&fc.When.Hour,
		&fc.CloudCover,
		&fc.Temperature,
		&fc.Precipitation)
	if err != nil {
		d.logger.Error("error when scanning weather forecast row", slog.Any("error", err))
		return WeatherForecastRow{}, err
	}

	return fc, nil
}

func (d *Database) GetWeatherForecastFrom(dh hours.DateHour) ([]WeatherForecastRow, error) {
	rows, err := d.read.Query(`
		SELECT date, hour, cloud_cover, temperature, precipitation
		FROM weather_forecast 
		WHERE (date = ? AND hour >= ?) OR date > ?`,
		dh.Date, dh.Hour, dh.Date)
	if err != nil {
		d.logger.Error("error when fetching weather forecast", slog.Any("error", err))
		return nil, err
	}

	defer rows.Close()

	var forecasts []WeatherForecastRow
	for rows.Next() {
		var row WeatherForecastRow
		err := rows.Scan(
			&row.When.Date,
			&row.When.Hour,
			&row.CloudCover,
			&row.Temperature,
			&row.Precipitation)
		if err != nil {
			d.logger.Error("error when scanning weather forecast row", slog.Any("error", err))
			return nil, err
		}

		forecasts = append(forecasts, row)
	}

	return forecasts, nil
}

func (d *Database) PurgeWeatherForecast(ctx context.Context) error {
	return d.purge(ctx, "weather_forecast")
}

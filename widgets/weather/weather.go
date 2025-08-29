// Package weather provides functionality to fetch weather data and display weather-related ASCII art.
package weather

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charmbracelet/lipgloss"
)

// WeatherResponse matches the structure of the wttr.in API response.
type WeatherResponse struct {
	Name string
	Temp float64
	Description string
	Icon string
}

// WttrResponse represents the raw response from wttr.in API
type WttrResponse struct {
	CurrentCondition []struct {
		TempC string `json:"temp_C"`
		WeatherDesc []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
		WeatherCode string `json:"weatherCode"`
	} `json:"current_condition"`
	NearestArea []struct {
		AreaName []struct {
			Value string `json:"value"`
		} `json:"areaName"`
	} `json:"nearest_area"`
}

// GetWeather fetches the current weather for a given city using the wttr.in API.
func GetWeather(city string) (*WeatherResponse, error) {
	url := fmt.Sprintf("https://wttr.in/%s?format=j1", city)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API request failed with status: %s", resp.Status)
	}

	var wttrResp WttrResponse
	if err := json.NewDecoder(resp.Body).Decode(&wttrResp); err != nil {
		return nil, err
	}

	// Convert wttr.in response to our WeatherResponse format
	if len(wttrResp.CurrentCondition) == 0 {
		return nil, fmt.Errorf("no weather data available")
	}

	current := wttrResp.CurrentCondition[0]
	
	// Convert temperature from string to float
	temp := 0.0
	if _, err := fmt.Sscanf(current.TempC, "%f", &temp); err != nil {
		return nil, fmt.Errorf("failed to parse temperature: %w", err)
	}

	// Get city name
	cityName := city
	if len(wttrResp.NearestArea) > 0 && len(wttrResp.NearestArea[0].AreaName) > 0 {
		cityName = wttrResp.NearestArea[0].AreaName[0].Value
	}

	// Get description
	description := "Unknown"
	if len(current.WeatherDesc) > 0 {
		description = current.WeatherDesc[0].Value
	}

	// Map weather code to icon for existing GetWeatherArt function
	icon := mapWeatherCodeToIcon(current.WeatherCode)

	return &WeatherResponse{
		Name:        cityName,
		Temp:        temp,
		Description: description,
		Icon:        icon,
	}, nil
}

// mapWeatherCodeToIcon converts wttr.in weather codes to icon strings compatible with GetWeatherArt
func mapWeatherCodeToIcon(code string) string {
	switch code {
	case "113": // Clear/Sunny
		return "01d"
	case "116", "119", "122": // Partly cloudy, Cloudy, Overcast
		return "02d"
	case "143", "248", "260": // Mist, Fog
		return "50d"
	case "176", "263", "266", "281", "284", "293", "296", "299", "302", "305", "308", "311", "314", "317", "320", "386", "389", "392", "395": // Various rain/drizzle
		return "10d"
	case "200", "201", "202", "210", "211", "212", "221", "230", "231", "232": // Thunderstorm
		return "11d"
	case "227", "323", "326", "329", "332", "335", "338", "350", "353", "356", "359", "362", "365", "368", "371", "374", "377": // Snow
		return "13d"
	default:
		return "01d" // Default to clear sky
	}
}

func GetWeatherArt(icon string) string {
	var art string
	var color lipgloss.Color

	switch icon {
	case "01d", "01n": // clear sky
		art = `
    \   /
    . - .
-- (     ) --
    ' - '
    /   \
`
		color = lipgloss.Color("#FFD700") // Gold
	case "02d", "02n", "03d", "03n", "04d", "04n", "50d", "50n": // clouds, mist
		art = `
   .--.
.-(    ).
(_______)
`
		color = lipgloss.Color("#FFF8B3") // White
	case "09d", "09n", "10d", "10n", "11d", "11n": // rain, thunderstorm
		art = `
   .--.
.-(    ).
(_______)
 / / / /
/ / / /
`
		color = lipgloss.Color("#61afef") // Blue
	case "13d", "13n": // snow
		art = `
   .--.
.-(    ).
(_______)
 * * * *
* * * *
`
		color = lipgloss.Color("#FFFFFF") // White
	default:
		return ""
	}

	style := lipgloss.NewStyle().Foreground(color)
	return style.Render(art)
}

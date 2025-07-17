// main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// --- Configuration ---
const API_KEY = "7441f0ec33ab2221a2f7285fbb1f2c93" // REMEMBER TO REPLACE THIS WITH YOUR ACTUAL KEY
const BASE_URL = "http://api.openweathermap.org/data/2.5/weather"

// --- Global variable to track if --show flag was explicitly set ---
var showFlagProvidedByUser bool

// --- Structs for JSON Unmarshaling ---
type WeatherResponse struct {
	Name string `json:"name"`
	Sys  struct {
		Country string `json:"country"`
		Sunrise int64  `json:"sunrise"` // Unix timestamp
		Sunset  int64  `json:"sunset"`  // Unix timestamp
	} `json:"sys"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Humidity  int     `json:"humidity"`
		Pressure  int     `json:"pressure"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"` // meters/sec
	} `json:"wind"`
	Clouds struct {
		All int `json:"all"` // Cloudiness in %
	} `json:"clouds"`
	Rain struct {
		OneH float64 `json:"1h"` // Rain volume for the last 1 hour, in mm
	} `json:"rain,omitempty"`
	Snow struct {
		OneH float64 `json:"1h"` // Snow volume for the last 1 hour, in mm
	} `json:"snow,omitempty"`
	Timezone int64 `json:"timezone"` // Shift in seconds from UTC
}

// --- Custom Flag Type for 'show' option ---

type showDetails map[string]bool

func (s *showDetails) String() string {
	if len(*s) == 0 {
		return ""
	}
	keys := make([]string, 0, len(*s))
	for k := range *s {
		keys = append(keys, k)
	}
	return strings.Join(keys, ",")
}

func (s *showDetails) Set(value string) error {
	// IMPORTANT: Set the global flag to true when this method is called.
	showFlagProvidedByUser = true

	*s = make(map[string]bool) // Ensure map is clean for new values

	parts := strings.Split(value, ",")
	for _, p := range parts {
		detail := strings.TrimSpace(strings.ToLower(p))
		if detail != "" {
			(*s)[detail] = true
		}
	}
	return nil
}

// --- Functions ---

func getWeatherData(city string) (*WeatherResponse, error) {
	url := fmt.Sprintf("%s?q=%s&appid=%s&units=metric", BASE_URL, city, API_KEY)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("error: Invalid API Key. Please check your API_KEY in main.go")
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("error: City '%s' not found. Please check the city name", city)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned non-200 status: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var weatherData WeatherResponse
	err = json.Unmarshal(body, &weatherData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &weatherData, nil
}

func displayWeather(data *WeatherResponse, detailsToShow showDetails) {
	if data == nil {
		return
	}

	city := data.Name
	country := data.Sys.Country

	fmt.Printf("\n--- Weather in %s, %s ---\n", city, country)

	// Determine if 'all' details should be shown:
	// 1. If the --show flag was NOT explicitly provided by the user, OR
	// 2. If the --show flag was provided AND it contains "all"
	displayAll := !showFlagProvidedByUser || detailsToShow["all"]

	if displayAll {
		// If displayAll is true, just print everything
		weatherDescription := "N/A"
		if len(data.Weather) > 0 {
			weatherDescription = data.Weather[0].Description
		}
		sunriseLocal := time.Unix(data.Sys.Sunrise+data.Timezone, 0).UTC()
		sunsetLocal := time.Unix(data.Sys.Sunset+data.Timezone, 0).UTC()

		precipitationInfo := "No recent precipitation reported"
		if data.Rain.OneH > 0 || data.Snow.OneH > 0 {
			precipitationInfo = ""
			if data.Rain.OneH > 0 {
				precipitationInfo += fmt.Sprintf("Rain: %.2f mm (last 1h)", data.Rain.OneH)
			}
			if data.Snow.OneH > 0 {
				if precipitationInfo != "" {
					precipitationInfo += ", "
				}
				precipitationInfo += fmt.Sprintf("Snow: %.2f mm (last 1h)", data.Snow.OneH)
			}
		}

		localTime := time.Now().UTC().Add(time.Duration(data.Timezone) * time.Second)

		fmt.Printf("Condition: %s\n", capitalizeFirstLetter(weatherDescription))
		fmt.Printf("Temperature: %.1f°C (Feels like: %.1f°C)\n", data.Main.Temp, data.Main.FeelsLike)
		fmt.Printf("Min Temp: %.1f°C, Max Temp: %.1f°C\n", data.Main.TempMin, data.Main.TempMax)
		fmt.Printf("Humidity: %d%%\n", data.Main.Humidity)
		fmt.Printf("Pressure: %d hPa\n", data.Main.Pressure)
		fmt.Printf("Wind Speed: %.1f m/s\n", data.Wind.Speed)
		fmt.Printf("Cloudiness: %d%%\n", data.Clouds.All)
		fmt.Printf("Sunrise: %s\n", sunriseLocal.Format("15:04:05"))
		fmt.Printf("Sunset: %s\n", sunsetLocal.Format("15:04:05"))
		fmt.Printf("Precipitation: %s\n", precipitationInfo)
		fmt.Printf("Current Local Time: %s\n", localTime.Format("2006-01-02 15:04:05"))

	} else {
		// If specific details were requested, only print those based on the map
		if detailsToShow["condition"] {
			weatherDescription := "N/A"
			if len(data.Weather) > 0 {
				weatherDescription = data.Weather[0].Description
			}
			fmt.Printf("Condition: %s\n", capitalizeFirstLetter(weatherDescription))
		}
		if detailsToShow["temperature"] || detailsToShow["temp"] {
			fmt.Printf("Temperature: %.1f°C (Feels like: %.1f°C)\n", data.Main.Temp, data.Main.FeelsLike)
			fmt.Printf("Min Temp: %.1f°C, Max Temp: %.1f°C\n", data.Main.TempMin, data.Main.TempMax)
		}
		if detailsToShow["humidity"] {
			fmt.Printf("Humidity: %d%%\n", data.Main.Humidity)
		}
		if detailsToShow["pressure"] {
			fmt.Printf("Pressure: %d hPa\n", data.Main.Pressure)
		}
		if detailsToShow["wind-speed"] || detailsToShow["wind"] {
			fmt.Printf("Wind Speed: %.1f m/s\n", data.Wind.Speed)
		}
		if detailsToShow["cloudiness"] || detailsToShow["clouds"] {
			fmt.Printf("Cloudiness: %d%%\n", data.Clouds.All)
		}
		if detailsToShow["sunrise"] {
			sunriseLocal := time.Unix(data.Sys.Sunrise+data.Timezone, 0).UTC()
			fmt.Printf("Sunrise: %s\n", sunriseLocal.Format("15:04:05"))
		}
		if detailsToShow["sunset"] {
			sunsetLocal := time.Unix(data.Sys.Sunset+data.Timezone, 0).UTC()
			fmt.Printf("Sunset: %s\n", sunsetLocal.Format("15:04:05"))
		}
		if detailsToShow["precipitation"] || detailsToShow["rain"] || detailsToShow["snow"] {
			precipitationInfo := "No recent precipitation reported"
			if data.Rain.OneH > 0 || data.Snow.OneH > 0 {
				precipitationInfo = ""
				if data.Rain.OneH > 0 {
					precipitationInfo += fmt.Sprintf("Rain: %.2f mm (last 1h)", data.Rain.OneH)
				}
				if data.Snow.OneH > 0 {
					if precipitationInfo != "" {
						precipitationInfo += ", "
					}
					precipitationInfo += fmt.Sprintf("Snow: %.2f mm (last 1h)", data.Snow.OneH)
				}
			}
			fmt.Printf("Precipitation: %s\n", precipitationInfo)
		}
		if detailsToShow["time"] || detailsToShow["current-time"] {
			localTime := time.Now().UTC().Add(time.Duration(data.Timezone) * time.Second)
			fmt.Printf("Current Local Time: %s\n", localTime.Format("2006-01-02 15:04:05"))
		}
	}

	fmt.Printf("%s\n", newString(len(fmt.Sprintf("--- Weather in %s, %s ---", city, country)), '-'))
}

// Helper functions (same as before)
func capitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	r[0] = toUpper(r[0])
	return string(r)
}

func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}

func newString(length int, char rune) string {
	if length <= 0 {
		return ""
	}
	s := make([]rune, length)
	for i := range s {
		s[i] = char
	}
	return string(s)
}

// --- Main Execution ---

func main() {
	// Initialize our custom flag type
	var detailsToShow showDetails

	// Register the custom flag
	flag.Var(&detailsToShow, "show", "Comma-separated list of details to display (e.g., 'temperature,humidity,time,all'). If omitted, all details are shown.")

	// Parse command-line flags
	flag.Parse()

	// After flag.Parse(), the non-flag arguments are available via flag.Args()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("Usage: weatherbro <city_name> [--show <details>]")
		fmt.Println("Example: weatherbro \"London\"")
		fmt.Println("Example: weatherbro \"New York\" --show temperature,humidity")
		fmt.Println("Example: weatherbro \"Tokyo\" --show time,sunrise,sunset")
		os.Exit(1)
	}

	city := args[0] // The first non-flag argument is the city name

	fmt.Printf("Fetching weather for %s...\n", city)
	weatherData, err := getWeatherData(city)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	displayWeather(weatherData, detailsToShow)
}

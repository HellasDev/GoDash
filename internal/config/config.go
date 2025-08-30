// Package config provides configuration management utilities for the GoDash application.
// It handles platform-aware directory paths, settings persistence, and application configuration.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigDir returns the platform-aware path to the configuration directory.
// Linux: ~/.config/GoDash
// macOS: ~/Library/Application Support/GoDash
func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "GoDash"), nil
}

// GetDataDir returns the platform-aware path to the data directory.
// Linux: ~/.local/share/GoDash
// macOS: ~/Library/Application Support/GoDash
func GetDataDir() (string, error) {
	if runtime.GOOS == "linux" {
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome != "" {
			return filepath.Join(dataHome, "GoDash"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "GoDash"), nil
	}
	// For macOS and other systems, data can live alongside config.
	return GetConfigDir()
}

// GetCacheDir returns the platform-aware path to the cache directory.
// Linux: ~/.cache/GoDash
// macOS: ~/Library/Caches/GoDash
func GetCacheDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "GoDash"), nil
}

// EnsureDirs creates the config, data, and cache directories if they don't exist.
func EnsureDirs() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	dataDir, err := GetDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	notesDir, err := GetNotesDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return err
	}

	return nil
}

// GetNotesDir returns the path to the notes directory.
func GetNotesDir() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "notes"), nil
}

// GetTodoPath returns the full path to the todo list file.
func GetTodoPath() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "todo-list.json"), nil
}

// Settings defines the structure for the application's configuration.
type Settings struct {
	Location           string `json:"location"`
	DefaultNotesCreated bool   `json:"default_notes_created"`
}

// SaveSettings writes the settings to the config file.
func SaveSettings(settings Settings) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(configDir, "config.json")

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}

// LoadSettings reads settings from the config file, or creates a default one.
func LoadSettings() (Settings, error) {
	var settings Settings

	configDir, err := GetConfigDir()
	if err != nil {
		return settings, err
	}
	settingsPath := filepath.Join(configDir, "config.json")

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, create a default one
			defaultSettings := Settings{Location: "Athens", DefaultNotesCreated: false}
			data, marshalErr := json.MarshalIndent(defaultSettings, "", "  ")
			if marshalErr != nil {
				return settings, marshalErr
			}
			writeErr := os.WriteFile(settingsPath, data, 0644)
			if writeErr != nil {
				return settings, writeErr
			}
			return defaultSettings, nil
		}
		// Some other error occurred
		return settings, err
	}

	// File exists, unmarshal it
	err = json.Unmarshal(content, &settings)
	if err != nil {
		return settings, err
	}

	if settings.Location == "" {
		settings.Location = "Athens"
	}

	return settings, nil
}

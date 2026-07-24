package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// UserConfig defines the configuration file schema for fmtui.
type UserConfig struct {
	AllowedHiddenFiles []string `json:"allowed_hidden_files"`
}

// loadUserConfig reads config from ~/.config/fmtui/config.json,
// creating it with default settings if it does not exist.
func loadUserConfig() UserConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultUserConfig()
	}
	configPath := filepath.Join(home, ".config", "fmtui", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg := defaultUserConfig()
		_ = saveUserConfig(cfg)
		return cfg
	}
	var cfg UserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultUserConfig()
	}
	return cfg
}

// defaultUserConfig returns the default list of searchable hidden files.
func defaultUserConfig() UserConfig {
	return UserConfig{
		AllowedHiddenFiles: []string{
			".zshrc", ".bashrc", ".profile", ".env", ".gitignore",
			".bash_profile", ".gitconfig", ".npmrc", ".vimrc",
		},
	}
}

// saveUserConfig writes configuration to ~/.config/fmtui/config.json.
func saveUserConfig(cfg UserConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dirPath := filepath.Join(home, ".config", "fmtui")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	configPath := filepath.Join(dirPath, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

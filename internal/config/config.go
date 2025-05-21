package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUsername string `json:"current_user_name"`
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}
	return home + "/" + configFileName, nil
}

func Write(cfg Config) error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return fmt.Errorf("could not get config file path: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode config to json: %w", err)
	}

	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		return fmt.Errorf("could not write config file path: %w", err)
	}

	return nil
}

func Read() (*Config, error) {
	var config Config
	configPath, err := getConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("could not decode json: %w", err)
	}

	return &config, nil
}

func (cfg *Config) SetUser(username string) error {
	cfg.CurrentUsername = username
	return Write(*cfg)
}

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// BDFSConfig represents the configuration structure for Baidu cloud
type BDFSConfig struct {
	ClientID        string `toml:"client_id"`
	ClientSecret    string `toml:"client_secret"`
	TokenPath       string `toml:"token_path"`
	DefaultCloudDir string `toml:"default_cloud_dir"`
}

// GetBDFSConfig retrieves the BDFS configuration from environment variables or TOML file
func GetBDFSConfig() (*BDFSConfig, error) {
	config := &BDFSConfig{}

	// First, check for individual environment variables
	clientID := os.Getenv("BDFS_CLIENT_ID")
	clientSecret := os.Getenv("BDFS_CLIENT_SECRET")
	tokenPath := os.Getenv("BDFS_TOKEN_PATH")
	defaultCloudDir := os.Getenv("BDFS_DEFAULT_CLOUD_DIR")

	// If all individual environment variables are provided, use them
	if clientID != "" && clientSecret != "" && tokenPath != "" {
		config.ClientID = clientID
		config.ClientSecret = clientSecret
		config.TokenPath = tokenPath
		config.DefaultCloudDir = defaultCloudDir
		// Set default cloud directory to "/" if not specified
		if config.DefaultCloudDir == "" {
			config.DefaultCloudDir = "/"
		}
		return config, nil
	}

	// If individual variables aren't all set, check for config file path
	configFilePath := os.Getenv("BDFS_CONFIG_FILE")

	// If BDFS_CONFIG_FILE is not set, use the default path
	if configFilePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %v", err)
		}
		configFilePath = filepath.Join(homeDir, ".local", "app", "dkci", "config.toml")
	}

	// Read and parse the TOML configuration file
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", configFilePath, err)
	}

	if err := toml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Ensure all required values are present
	if config.ClientID == "" || config.ClientSecret == "" || config.TokenPath == "" {
		return nil, fmt.Errorf("config file missing required fields (client_id, client_secret, token_path)")
	}

	// Set default cloud directory to "/" if not specified in the config
	if config.DefaultCloudDir == "" {
		config.DefaultCloudDir = "/"
	}

	return config, nil
}

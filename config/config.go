// Package config provides the main configuration for the application.
package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration for the application.
type Config struct {
	APIURL       string        `yaml:"api_url"`       // Base URL for the Crafty API
	Username     string        `yaml:"username"`      // Username for Crafty API authentication
	Password     string        `yaml:"password"`      // Password for Crafty API authentication
	LogLevel     string        `yaml:"log_level"`     // Logging level (e.g., DEBUG, INFO, ERROR)
	Timeout      time.Duration `yaml:"timeout"`       // Global timeout for API requests
	AutoShutdown bool          `yaml:"auto_shutdown"` // Whether to automatically shut down idle servers
	Addresses    []ServerType  `yaml:"addresses"`     // List of server connection configurations
}

// ServerType defines the network parameters and mapping between a listener and a Crafty server.
type ServerType struct {
	Protocol   string `yaml:"protocol"`    // Network protocol used (e.g., tcp, udp)
	Listener   Host   `yaml:"listener"`    // Address and port the proxy listens on
	CraftyHost Host   `yaml:"crafty_host"` // Corresponding Crafty server address and port
}

// Host defines a network address and port pair.
type Host struct {
	Addr string `yaml:"addr"` // IP address or hostname
	Port int    `yaml:"port"` // Port number
}

// NewConfig returns a Config instance populated with default values.
func NewConfig() Config {
	return Config{
		APIURL:       "https://crafty:8443",
		Username:     "admin",
		Password:     "password",
		LogLevel:     "INFO",
		Timeout:      time.Minute * 5,
		AutoShutdown: true,
		Addresses: []ServerType{
			{
				Protocol: "tcp",
				Listener: Host{
					Addr: "127.0.0.1",
					Port: 25565,
				},
				CraftyHost: Host{
					Addr: "crafty",
					Port: 25565,
				},
			},
		},
	}
}

// Load reads configuration from the specified file path into the Config struct.
// If the file does not exist, a default configuration is created and saved to the path.
func (c *Config) Load(path string) error {
	file, err := os.Open(path) //nolint
	if err != nil {
		if os.IsNotExist(err) {
			defaultConfig := NewConfig()
			data, marshalErr := yaml.Marshal(defaultConfig)
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal default config: %w", marshalErr)
			}

			writeErr := os.WriteFile(path, data, 0600)
			if writeErr != nil {
				return fmt.Errorf("failed to write default config file: %w", writeErr)
			}

			log.Printf("config file not found — created default at %s\n", path)
			return nil
		}

		return fmt.Errorf("could not open config file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("could not read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("could not parse yaml config: %w", err)
	}

	return nil
}

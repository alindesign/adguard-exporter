package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Client struct {
	Address  string
	Username string
	Password string
}

type Config struct {
	Interval    time.Duration `env:"APP_INTERVAL, default=30s"`
	Debug       bool          `env:"APP_DEBUG, default=false"`
	Port        string        `env:"SERVER_PORT, default=9618"`
	Host        string        `env:"SERVER_HOST, default=127.0.0.1"`
	ClientsFile string        `env:"CLIENTS_FILE, default=/etc/adguard-exporter/clients.yaml"`
	Clients     []Client
}

type EnvVal interface {
	time.Duration | bool | string
}

func env(key string, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return def
}

func Load() (*Config, error) {
	interval, err := time.ParseDuration(env("APP_INTERVAL", "30s"))
	if err != nil {
		return nil, fmt.Errorf("could not parse interval: %v", err)
	}

	config := &Config{
		Interval:    interval,
		Debug:       env("APP_DEBUG", "false") == "true",
		Port:        env("SERVER_PORT", "9618"),
		Host:        env("SERVER_HOST", ""),
		ClientsFile: env("CLIENTS_FILE", "/etc/adguard-exporter/clients.yaml"),
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	err = config.loadClients()
	if err != nil {
		return config, err
	}

	return config, nil
}

func (c *Config) loadClients() error {
	contents, err := os.ReadFile(c.ClientsFile)
	if err != nil {
		return fmt.Errorf("could not read clients file: %v", err)
	}
	if len(serv.BindAddr) == 0 {
		serv.BindAddr = ":9618"
	}

	err = yaml.Unmarshal(contents, &c.Clients)
	if err != nil {
		return fmt.Errorf("could not unmarshal clients file: %v", err)
	}

	if len(c.Clients) == 0 {
		return errors.New("no configured clients")
	}

	for i, client := range c.Clients {
		client.Address = c.tryFile(client.Address)
		client.Username = c.tryFile(client.Username)
		client.Password = c.tryFile(client.Password)
		c.Clients[i] = client
	}

	return nil
}

func (c *Config) validate() error {
	if c.Interval <= 0 {
		return errors.New("interval must be greater than 0")
	}

	if c.Port == "" {
		return errors.New("port must be set")
	}

	if c.ClientsFile == "" {
		return errors.New("clients file must be set")
	}

	if _, err := os.Stat(c.ClientsFile); os.IsNotExist(err) {
		return errors.New("clients file does not exist")
	}

	return nil
}

func (c *Config) tryFile(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return path
	}

	return strings.TrimSpace(string(contents))
}

package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// RegistrySettings defines the settings for a registry.
type RegistrySettings struct {
	Auth            Auth        `yaml:"auth,omitempty"`
	CacheDir        string      `yaml:"cache_dir,omitempty"`
	CacheMaxSize    StorageSize `yaml:"cache_max_size,omitempty"`
	UpstreamProxy   string      `yaml:"upstream_proxy,omitempty"`
	FollowRedirects *bool       `yaml:"follow_redirects,omitempty"`
	Insecure        *bool       `yaml:"insecure,omitempty"`
}

// Config holds the application configuration.
type Config struct {
	Port            int                         `yaml:"port"`
	LogLevel        string                      `yaml:"log_level"`
	DefaultRegistry string                      `yaml:"default_registry"`
	BaseURL         string                      `yaml:"base_url"`
	WhitelistMode   bool                        `yaml:"whitelist_mode"`
	Auth            Auth                        `yaml:"auth"`
	Defaults        RegistrySettings            `yaml:"defaults"`
	Registries      map[string]RegistrySettings `yaml:"registries"`
}

// LoadConfig reads the configuration from the given path.
func LoadConfig(path string) (*Config, error) {
	config := &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}
	config.applyDefaults()
	return config, nil
}

func (c *Config) applyDefaults() {
	if c.Defaults.FollowRedirects == nil {
		b := true
		c.Defaults.FollowRedirects = &b
	}
	if c.Defaults.Insecure == nil {
		b := false
		c.Defaults.Insecure = &b
	}

	for name, registrySettings := range c.Registries {
		merged := c.Defaults
		if registrySettings.Auth.Username != "" {
			merged.Auth = registrySettings.Auth
		}

		if registrySettings.CacheDir != "" {
			merged.CacheDir = registrySettings.CacheDir
		}
		if registrySettings.CacheMaxSize != 0 {
			merged.CacheMaxSize = registrySettings.CacheMaxSize
		}
		if registrySettings.UpstreamProxy != "" {
			merged.UpstreamProxy = registrySettings.UpstreamProxy
		}
		if registrySettings.FollowRedirects != nil {
			merged.FollowRedirects = registrySettings.FollowRedirects
		}
		if registrySettings.Insecure != nil {
			merged.Insecure = registrySettings.Insecure
		}
		c.Registries[name] = merged
	}
}

// GetRegistrySettings returns the merged settings for a given registry.
func (c *Config) GetRegistrySettings(registryName string) RegistrySettings {
	if settings, ok := c.Registries[registryName]; ok {
		return settings
	}
	return c.Defaults
}

// IsRegistryAllowed checks if a registry is allowed in whitelist mode.
func (c *Config) IsRegistryAllowed(registryName string) bool {
	if !c.WhitelistMode {
		return true
	}
	_, ok := c.Registries[registryName]
	return ok
}

// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package config

import "encoding/json"

// Config holds Datadog-specific configuration extracted from a Target.
type Config struct {
	ApiKey string
	AppKey string
	Site   string
}

// FromTargetConfig extracts Datadog configuration from target config JSON.
// ApiKey and AppKey are required for API authentication.
// Site defaults to "datadoghq.com" if not specified.
func FromTargetConfig(targetConfig json.RawMessage) *Config {
	cfg := &Config{}

	if targetConfig != nil {
		var raw map[string]interface{}
		if err := json.Unmarshal(targetConfig, &raw); err == nil {
			cfg.ApiKey, _ = raw["ApiKey"].(string)
			cfg.AppKey, _ = raw["AppKey"].(string)
			cfg.Site, _ = raw["Site"].(string)
		}
	}

	return cfg
}

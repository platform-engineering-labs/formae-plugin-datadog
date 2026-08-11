// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package config

import (
	"encoding/json"
	"os"
)

// Config holds Datadog-specific configuration extracted from a Target.
type Config struct {
	ApiKey string
	AppKey string
	Site   string
}

// FromTargetConfig extracts Datadog configuration from target config JSON.
//
// Each key is taken from the target config when declared there and from the
// environment otherwise, so one may come from a formae-managed secret while the
// other stays in the environment. A key declared in the target config may
// originate from such a secret, which the agent resolves live before every
// call, so rotating it needs no agent restart.
//
// A declared key is used as given: falling back from an empty one would
// authenticate as whoever the environment names, which is a silent identity
// switch rather than a recovery. Site defaults in the client when left empty.
func FromTargetConfig(targetConfig json.RawMessage) *Config {
	cfg := &Config{}

	var raw map[string]interface{}
	if targetConfig != nil {
		if err := json.Unmarshal(targetConfig, &raw); err != nil {
			raw = nil
		}
	}

	cfg.ApiKey = declaredOrEnv(raw, "ApiKey", "DD_API_KEY")
	cfg.AppKey = declaredOrEnv(raw, "AppKey", "DD_APP_KEY")
	cfg.Site, _ = raw["Site"].(string)

	return cfg
}

// declaredOrEnv prefers the value declared under key, falling back to the
// environment variable only when the key is absent altogether.
func declaredOrEnv(raw map[string]interface{}, key, envVar string) string {
	if value, ok := raw[key]; ok {
		declared, _ := value.(string)
		return declared
	}
	return os.Getenv(envVar)
}

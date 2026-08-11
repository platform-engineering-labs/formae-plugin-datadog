// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package config

import (
	"testing"
)

func clearAmbientKeys(t *testing.T) {
	t.Helper()
	t.Setenv("DD_API_KEY", "")
	t.Setenv("DD_APP_KEY", "")
}

func TestKeysDeclaredInTargetConfigWin(t *testing.T) {
	clearAmbientKeys(t)
	t.Setenv("DD_API_KEY", "env-api")
	t.Setenv("DD_APP_KEY", "env-app")

	cfg := FromTargetConfig([]byte(`{"ApiKey":"config-api","AppKey":"config-app"}`))

	if cfg.ApiKey != "config-api" || cfg.AppKey != "config-app" {
		t.Errorf("keys = %q/%q, want the target config values", cfg.ApiKey, cfg.AppKey)
	}
}

func TestKeysFallBackToTheEnvironmentWhenAbsent(t *testing.T) {
	clearAmbientKeys(t)
	t.Setenv("DD_API_KEY", "env-api")
	t.Setenv("DD_APP_KEY", "env-app")

	cfg := FromTargetConfig([]byte(`{"Site":"datadoghq.eu"}`))

	if cfg.ApiKey != "env-api" || cfg.AppKey != "env-app" {
		t.Errorf("keys = %q/%q, want the environment values", cfg.ApiKey, cfg.AppKey)
	}
}

// Each key falls back on its own, so one may come from a secret while the other
// stays in the environment.
func TestEachKeyFallsBackIndependently(t *testing.T) {
	clearAmbientKeys(t)
	t.Setenv("DD_API_KEY", "env-api")
	t.Setenv("DD_APP_KEY", "env-app")

	cfg := FromTargetConfig([]byte(`{"AppKey":"config-app"}`))

	if cfg.ApiKey != "env-api" {
		t.Errorf("ApiKey = %q, want the environment value", cfg.ApiKey)
	}
	if cfg.AppKey != "config-app" {
		t.Errorf("AppKey = %q, want the target config value", cfg.AppKey)
	}
}

// A declared key that resolves to nothing is a misconfiguration, not an
// invitation to authenticate as whoever the environment names.
func TestDeclaredButEmptyKeyDoesNotFallBackToTheEnvironment(t *testing.T) {
	clearAmbientKeys(t)
	t.Setenv("DD_API_KEY", "env-api")

	cfg := FromTargetConfig([]byte(`{"ApiKey":""}`))

	if cfg.ApiKey != "" {
		t.Errorf("ApiKey = %q, want no fall back to the environment", cfg.ApiKey)
	}
}

func TestSiteDefaultsWhenNotDeclared(t *testing.T) {
	clearAmbientKeys(t)

	cfg := FromTargetConfig([]byte(`{}`))

	if cfg.Site != "" {
		t.Errorf("Site = %q, want it left empty for the client to default", cfg.Site)
	}
}

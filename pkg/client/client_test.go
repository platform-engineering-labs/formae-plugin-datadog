// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package client

import (
	"strings"
	"testing"

	ddconfig "github.com/platform-engineering-labs/formae-plugin-datadog/pkg/config"
)

// A missing credential should name both places it could have come from, rather
// than surfacing later as an unexplained authorization failure from Datadog.
func TestNewClientRejectsAMissingApiKey(t *testing.T) {
	_, err := NewClient(&ddconfig.Config{AppKey: "app"})
	if err == nil {
		t.Fatal("NewClient accepted a config with no API key")
	}
	if !strings.Contains(err.Error(), "DD_API_KEY") {
		t.Errorf("error %q should name the environment variable", err)
	}
}

func TestNewClientRejectsAMissingAppKey(t *testing.T) {
	_, err := NewClient(&ddconfig.Config{ApiKey: "api"})
	if err == nil {
		t.Fatal("NewClient accepted a config with no application key")
	}
	if !strings.Contains(err.Error(), "DD_APP_KEY") {
		t.Errorf("error %q should name the environment variable", err)
	}
}

func TestNewClientAcceptsBothKeys(t *testing.T) {
	if _, err := NewClient(&ddconfig.Config{ApiKey: "api", AppKey: "app"}); err != nil {
		t.Errorf("NewClient: %v", err)
	}
}

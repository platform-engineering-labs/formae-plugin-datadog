// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package client

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"

	ddconfig "github.com/platform-engineering-labs/formae-plugin-datadog/pkg/config"
)

// Client wraps the Datadog API client with an authenticated context.
// Datadog SDK uses context-based auth: API/App keys are injected into
// context.Context rather than the client config struct.
type Client struct {
	Ctx       context.Context
	ApiClient *datadog.APIClient
}

// NewClient creates a new Datadog client from plugin config.
//
// Credentials are checked here rather than left to the first API call, so a
// missing one names both places it could have come from instead of surfacing
// as an unexplained authorization failure from Datadog.
func NewClient(cfg *ddconfig.Config) (*Client, error) {
	if cfg.ApiKey == "" {
		return nil, fmt.Errorf("no Datadog API key found; set ApiKey in the target config or DD_API_KEY")
	}
	if cfg.AppKey == "" {
		return nil, fmt.Errorf("no Datadog application key found; set AppKey in the target config or DD_APP_KEY")
	}

	ctx := context.Background()

	// Inject API keys into context (Datadog SDK auth pattern)
	ctx = context.WithValue(ctx, datadog.ContextAPIKeys, map[string]datadog.APIKey{
		"apiKeyAuth": {Key: cfg.ApiKey},
		"appKeyAuth": {Key: cfg.AppKey},
	})

	// Set site if specified (e.g. "datadoghq.eu", "us5.datadoghq.com")
	if cfg.Site != "" {
		ctx = context.WithValue(ctx, datadog.ContextServerVariables, map[string]string{
			"site": cfg.Site,
		})
	}

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)

	return &Client{
		Ctx:       ctx,
		ApiClient: apiClient,
	}, nil
}

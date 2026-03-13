// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package client

import (
	"context"

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
func NewClient(cfg *ddconfig.Config) (*Client, error) {
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

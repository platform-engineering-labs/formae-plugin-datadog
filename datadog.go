// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package main

import (
	"context"
	"fmt"

	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/registry"
	"github.com/platform-engineering-labs/formae/pkg/plugin"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"

	// Import resources to trigger init() registration
	_ "github.com/platform-engineering-labs/formae-plugin-datadog/pkg/resources"
)

// Plugin implements the Formae ResourcePlugin interface.
type Plugin struct{}

var _ plugin.ResourcePlugin = &Plugin{}

func (p *Plugin) RateLimit() plugin.RateLimitConfig {
	return plugin.RateLimitConfig{
		Scope:                            plugin.RateLimitScopeNamespace,
		MaxRequestsPerSecondForNamespace: 3,
	}
}

func (p *Plugin) DiscoveryFilters() []plugin.MatchFilter {
	return nil
}

func (p *Plugin) LabelConfig() plugin.LabelConfig {
	return plugin.LabelConfig{
		DefaultQuery: "$.name",
	}
}

func (p *Plugin) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	targetConfig := config.FromTargetConfig(request.TargetConfig)
	c, err := client.NewClient(targetConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Datadog client: %w", err)
	}

	if !registry.HasProvisioner(request.ResourceType) {
		return nil, fmt.Errorf("unsupported resource type: %s", request.ResourceType)
	}

	prov := registry.Get(request.ResourceType, c, targetConfig)
	return prov.Create(ctx, request)
}

func (p *Plugin) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	targetConfig := config.FromTargetConfig(request.TargetConfig)
	c, err := client.NewClient(targetConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Datadog client: %w", err)
	}

	if !registry.HasProvisioner(request.ResourceType) {
		return nil, fmt.Errorf("unsupported resource type: %s", request.ResourceType)
	}

	prov := registry.Get(request.ResourceType, c, targetConfig)
	return prov.Read(ctx, request)
}

func (p *Plugin) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	targetConfig := config.FromTargetConfig(request.TargetConfig)
	c, err := client.NewClient(targetConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Datadog client: %w", err)
	}

	if !registry.HasProvisioner(request.ResourceType) {
		return nil, fmt.Errorf("unsupported resource type: %s", request.ResourceType)
	}

	prov := registry.Get(request.ResourceType, c, targetConfig)
	return prov.Update(ctx, request)
}

func (p *Plugin) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	targetConfig := config.FromTargetConfig(request.TargetConfig)
	c, err := client.NewClient(targetConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Datadog client: %w", err)
	}

	if !registry.HasProvisioner(request.ResourceType) {
		return nil, fmt.Errorf("unsupported resource type: %s", request.ResourceType)
	}

	prov := registry.Get(request.ResourceType, c, targetConfig)
	return prov.Delete(ctx, request)
}

func (p *Plugin) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	targetConfig := config.FromTargetConfig(request.TargetConfig)
	c, err := client.NewClient(targetConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Datadog client: %w", err)
	}

	if !registry.HasProvisioner(request.ResourceType) {
		return nil, fmt.Errorf("unsupported resource type: %s", request.ResourceType)
	}

	prov := registry.Get(request.ResourceType, c, targetConfig)
	return prov.Status(ctx, request)
}

func (p *Plugin) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	log := plugin.LoggerFromContext(ctx)
	log.Debug("List called",
		"resourceType", request.ResourceType,
		"additionalProperties", request.AdditionalProperties,
	)

	targetConfig := config.FromTargetConfig(request.TargetConfig)
	c, err := client.NewClient(targetConfig)
	if err != nil {
		log.Error("Failed to create Datadog client", "error", err)
		return nil, fmt.Errorf("failed to create Datadog client: %w", err)
	}

	if !registry.HasProvisioner(request.ResourceType) {
		log.Error("Unsupported resource type", "resourceType", request.ResourceType)
		return nil, fmt.Errorf("unsupported resource type: %s", request.ResourceType)
	}

	prov := registry.Get(request.ResourceType, c, targetConfig)
	result, err := prov.List(ctx, request)
	if err != nil {
		log.Error("List failed", "resourceType", request.ResourceType, "error", err)
		return result, err
	}

	log.Debug("List completed",
		"resourceType", request.ResourceType,
		"nativeIDCount", len(result.NativeIDs),
	)
	return result, nil
}

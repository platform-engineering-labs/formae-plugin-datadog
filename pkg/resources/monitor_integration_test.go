// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/config"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

func newTestClient(t *testing.T) *client.Client {
	t.Helper()

	apiKey := os.Getenv("DD_API_KEY")
	appKey := os.Getenv("DD_APP_KEY")
	site := os.Getenv("DD_SITE")

	if apiKey == "" || appKey == "" {
		t.Skip("DD_API_KEY and DD_APP_KEY must be set for integration tests")
	}
	if site == "" {
		site = "datadoghq.com"
	}

	cfg := &config.Config{
		ApiKey: apiKey,
		AppKey: appKey,
		Site:   site,
	}
	c, err := client.NewClient(cfg)
	require.NoError(t, err, "Failed to create Datadog client")
	return c
}

func newTestMonitor(t *testing.T) *Monitor {
	t.Helper()
	return &Monitor{Client: newTestClient(t)}
}

func testMonitorName(suffix string) string {
	return fmt.Sprintf("formae-integration-test-%s-%d", suffix, time.Now().Unix())
}

func int64Ptr(v int64) *int64 { return &v }

func float64Ptr(v float64) *float64 { return &v }

func stringPtr(v string) *string { return &v }

func boolPtr(v bool) *bool { return &v }

// deleteMonitor is a cleanup helper that ignores errors (monitor may already be gone).
func deleteMonitor(ctx context.Context, prov *Monitor, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

func TestMonitor_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestMonitor(t)
	name := testMonitorName("lifecycle")

	props, _ := json.Marshal(monitorProps{
		Name:    name,
		Type:    "metric alert",
		Query:   "avg(last_5m):avg:system.cpu.user{*} > 90",
		Message: stringPtr("CPU usage is high"),
		Tags:    []string{"env:test", "managed-by:formae"},
		Options: &monitorOptionsProps{
			Thresholds: &monitorThresholdsProps{
				Critical: float64Ptr(90),
				Warning:  float64Ptr(80),
			},
			NotifyNoData: boolPtr(false),
		},
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeMonitor,
		Label:        "test-monitor",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.NotEmpty(t, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created monitor: %s (ID: %s)", name, nativeID)
	t.Cleanup(func() { deleteMonitor(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeMonitor,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeMonitor, readResult.ResourceType)

	var readProps monitorProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, name, readProps.Name)
	assert.Equal(t, "metric alert", readProps.Type)
	assert.Contains(t, readProps.Query, "system.cpu.user")
	assert.NotNil(t, readProps.Id)

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID: nativeID,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)

	// Verify gone
	_, err = prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeMonitor,
	})
	assert.Error(t, err)
}

func TestMonitor_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestMonitor(t)
	name := testMonitorName("update")

	props, _ := json.Marshal(monitorProps{
		Name:  name,
		Type:  "metric alert",
		Query: "avg(last_5m):avg:system.cpu.user{*} > 90",
		Options: &monitorOptionsProps{
			Thresholds: &monitorThresholdsProps{
				Critical: float64Ptr(90),
			},
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeMonitor,
		Label:        "test-monitor",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteMonitor(ctx, prov, nativeID) })

	// Update
	updatedName := name + "-updated"
	desiredProps, _ := json.Marshal(monitorProps{
		Name:    updatedName,
		Type:    "metric alert",
		Query:   "avg(last_5m):avg:system.cpu.user{*} > 95",
		Message: stringPtr("CPU usage is very high"),
		Tags:    []string{"env:test", "managed-by:formae", "updated:true"},
		Options: &monitorOptionsProps{
			Thresholds: &monitorThresholdsProps{
				Critical: float64Ptr(95),
				Warning:  float64Ptr(85),
			},
		},
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeMonitor,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify update
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeMonitor,
	})
	require.NoError(t, err)

	var readProps monitorProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, updatedName, readProps.Name)
	assert.Contains(t, readProps.Query, "> 95")
	assert.Equal(t, "CPU usage is very high", *readProps.Message)
}

func TestMonitor_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestMonitor(t)
	name := testMonitorName("list")

	props, _ := json.Marshal(monitorProps{
		Name:  name,
		Type:  "metric alert",
		Query: "avg(last_5m):avg:system.cpu.user{*} > 90",
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeMonitor,
		Label:        "test-monitor",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteMonitor(ctx, prov, nativeID) })

	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeMonitor,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, listResult.NativeIDs)

	found := false
	for _, id := range listResult.NativeIDs {
		if id == nativeID {
			found = true
			break
		}
	}
	assert.True(t, found, "Created monitor %s should appear in List results", nativeID)
	t.Logf("List returned %d monitors", len(listResult.NativeIDs))
}

func TestMonitor_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestMonitor(t)
	name := testMonitorName("delete-idem")

	props, _ := json.Marshal(monitorProps{
		Name:  name,
		Type:  "metric alert",
		Query: "avg(last_5m):avg:system.cpu.user{*} > 90",
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeMonitor,
		Label:        "test-monitor",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID

	// Delete once
	_, err = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)

	// Delete again — should succeed (idempotent)
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

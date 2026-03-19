// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

//go:build integration

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

func newTestDashboard(t *testing.T) *Dashboard {
	t.Helper()
	return &Dashboard{Client: newTestClient(t)}
}

func testDashboardTitle(suffix string) string {
	return fmt.Sprintf("formae-integration-test-%s-%d", suffix, time.Now().Unix())
}

func deleteDashboard(ctx context.Context, prov *Dashboard, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

func TestDashboard_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestDashboard(t)
	title := testDashboardTitle("lifecycle")

	props, _ := json.Marshal(dashboardProps{
		Title:      title,
		LayoutType: "ordered",
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeDashboard,
		Label:        "test-dashboard",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.NotEmpty(t, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created dashboard: %s (%s)", title, nativeID)
	t.Cleanup(func() { deleteDashboard(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeDashboard,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeDashboard, readResult.ResourceType)

	var readProps dashboardProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, title, readProps.Title)
	assert.Equal(t, "ordered", readProps.LayoutType)

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

func TestDashboard_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestDashboard(t)
	title := testDashboardTitle("update")

	props, _ := json.Marshal(dashboardProps{
		Title:      title,
		LayoutType: "ordered",
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeDashboard,
		Label:        "test-dashboard",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteDashboard(ctx, prov, nativeID) })

	// Update — change title and add description
	desc := "Updated dashboard"
	desiredProps, _ := json.Marshal(dashboardProps{
		Title:       title + "-updated",
		LayoutType:  "ordered",
		Description: &desc,
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeDashboard,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	var updateProps dashboardProps
	require.NoError(t, json.Unmarshal(updateResult.ProgressResult.ResourceProperties, &updateProps))
	assert.Equal(t, title+"-updated", updateProps.Title)
	assert.Equal(t, "Updated dashboard", *updateProps.Description)
}

func TestDashboard_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestDashboard(t)
	title := testDashboardTitle("list")

	props, _ := json.Marshal(dashboardProps{
		Title:      title,
		LayoutType: "ordered",
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeDashboard,
		Label:        "test-dashboard",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteDashboard(ctx, prov, nativeID) })

	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeDashboard,
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
	assert.True(t, found, "Created dashboard %s should appear in List results", nativeID)
	t.Logf("List returned %d dashboards", len(listResult.NativeIDs))
}

func TestDashboard_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestDashboard(t)
	title := testDashboardTitle("delete-idem")

	props, _ := json.Marshal(dashboardProps{
		Title:      title,
		LayoutType: "ordered",
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeDashboard,
		Label:        "test-dashboard",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID

	_, err = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)

	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

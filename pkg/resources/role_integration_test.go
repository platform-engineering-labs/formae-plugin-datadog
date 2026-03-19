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

func newTestRole(t *testing.T) *Role {
	t.Helper()
	return &Role{Client: newTestClient(t)}
}

func testRoleName(suffix string) string {
	return fmt.Sprintf("formae-integration-test-%s-%d", suffix, time.Now().Unix())
}

func deleteRole(ctx context.Context, prov *Role, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

func TestRole_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestRole(t)
	name := testRoleName("lifecycle")

	props, _ := json.Marshal(roleProps{
		Name: name,
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeRole,
		Label:        "test-role",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.NotEmpty(t, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created role: %s (%s)", name, nativeID)
	t.Cleanup(func() { deleteRole(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeRole,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeRole, readResult.ResourceType)

	var readProps roleProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, name, readProps.Name)

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

func TestRole_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestRole(t)
	name := testRoleName("update")

	props, _ := json.Marshal(roleProps{
		Name: name,
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeRole,
		Label:        "test-role",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteRole(ctx, prov, nativeID) })

	// Update — change name
	newName := name + "-updated"
	desiredProps, _ := json.Marshal(roleProps{
		Name: newName,
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeRole,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify via Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeRole,
	})
	require.NoError(t, err)
	var readProps roleProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, newName, readProps.Name)
}

func TestRole_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestRole(t)
	name := testRoleName("list")

	props, _ := json.Marshal(roleProps{
		Name: name,
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeRole,
		Label:        "test-role",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteRole(ctx, prov, nativeID) })

	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeRole,
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
	assert.True(t, found, "Created role %s should appear in List results", nativeID)
	t.Logf("List returned %d roles", len(listResult.NativeIDs))
}

func TestRole_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestRole(t)
	name := testRoleName("delete-idem")

	props, _ := json.Marshal(roleProps{
		Name: name,
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeRole,
		Label:        "test-role",
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

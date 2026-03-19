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

func newTestTeam(t *testing.T) *Team {
	t.Helper()
	return &Team{Client: newTestClient(t)}
}

func testTeamHandle(suffix string) string {
	return fmt.Sprintf("formae-test-%s-%d", suffix, time.Now().Unix())
}

func deleteTeam(ctx context.Context, prov *Team, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

func TestTeam_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestTeam(t)
	handle := testTeamHandle("lifecycle")

	props, _ := json.Marshal(teamProps{
		Name:   "Formae Integration Test Lifecycle",
		Handle: handle,
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeTeam,
		Label:        "test-team",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.NotEmpty(t, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created team: %s (%s)", handle, nativeID)
	t.Cleanup(func() { deleteTeam(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeTeam,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeTeam, readResult.ResourceType)

	var readProps teamProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, "Formae Integration Test Lifecycle", readProps.Name)
	assert.Equal(t, handle, readProps.Handle)

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

func TestTeam_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestTeam(t)
	handle := testTeamHandle("update")

	props, _ := json.Marshal(teamProps{
		Name:   "Formae Integration Test Update",
		Handle: handle,
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeTeam,
		Label:        "test-team",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteTeam(ctx, prov, nativeID) })

	// Update — change name and description
	desc := "Updated description"
	desiredProps, _ := json.Marshal(teamProps{
		Name:        "Formae Integration Test Updated",
		Handle:      handle,
		Description: &desc,
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeTeam,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify via update result
	var updateProps teamProps
	require.NoError(t, json.Unmarshal(updateResult.ProgressResult.ResourceProperties, &updateProps))
	assert.Equal(t, "Formae Integration Test Updated", updateProps.Name)
	assert.Equal(t, handle, updateProps.Handle)
	assert.Equal(t, "Updated description", *updateProps.Description)
}

func TestTeam_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestTeam(t)
	handle := testTeamHandle("list")

	props, _ := json.Marshal(teamProps{
		Name:   "Formae Integration Test List",
		Handle: handle,
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeTeam,
		Label:        "test-team",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteTeam(ctx, prov, nativeID) })

	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeTeam,
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
	assert.True(t, found, "Created team %s should appear in List results", nativeID)
	t.Logf("List returned %d teams", len(listResult.NativeIDs))
}

func TestTeam_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestTeam(t)
	handle := testTeamHandle("delete-idem")

	props, _ := json.Marshal(teamProps{
		Name:   "Formae Integration Test Delete Idem",
		Handle: handle,
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeTeam,
		Label:        "test-team",
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

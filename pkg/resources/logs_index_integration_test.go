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

func newTestLogsIndex(t *testing.T) *LogsIndex {
	t.Helper()
	return &LogsIndex{Client: newTestClient(t)}
}

func testLogsIndexName(suffix string) string {
	return fmt.Sprintf("formae-integration-test-%s-%d", suffix, time.Now().Unix())
}

// deleteLogsIndex is a cleanup helper that ignores errors.
func deleteLogsIndex(ctx context.Context, prov *LogsIndex, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

// waitForIndexPropagation waits for an index to become available after creation.
// Datadog Logs Index API has eventual consistency — newly created indexes
// take a few seconds to become readable.
func waitForIndexPropagation(t *testing.T, ctx context.Context, prov *LogsIndex, nativeID string) {
	t.Helper()
	for i := 0; i < 10; i++ {
		_, err := prov.Read(ctx, &resource.ReadRequest{
			NativeID:     nativeID,
			ResourceType: ResourceTypeLogsIndex,
		})
		if err == nil {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("Index %s did not propagate within 10 seconds", nativeID)
}

// createSpareIndex creates a second index to ensure our test index isn't
// the last one (Datadog requires at least one index at all times).
func createSpareIndex(t *testing.T, ctx context.Context, prov *LogsIndex) string {
	t.Helper()
	name := testLogsIndexName("spare")
	props, _ := json.Marshal(logsIndexProps{
		Name:   name,
		Filter: logsIndexFilter{Query: stringPtr("source:spare")},
	})
	result, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsIndex,
		Label:        "spare-index",
		Properties:   props,
	})
	require.NoError(t, err, "Failed to create spare index")
	waitForIndexPropagation(t, ctx, prov, result.ProgressResult.NativeID)
	return result.ProgressResult.NativeID
}

func TestLogsIndex_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsIndex(t)
	name := testLogsIndexName("lifecycle")

	props, _ := json.Marshal(logsIndexProps{
		Name:   name,
		Filter: logsIndexFilter{Query: stringPtr("source:test")},
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsIndex,
		Label:        "test-logs-index",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.Equal(t, name, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created logs index: %s", nativeID)
	waitForIndexPropagation(t, ctx, prov, nativeID)

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeLogsIndex,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeLogsIndex, readResult.ResourceType)

	var readProps logsIndexProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, name, readProps.Name)
	assert.Equal(t, "source:test", *readProps.Filter.Query)

	// Create a spare index so we can delete the test one
	// (Datadog requires at least one index at all times).
	spareID := createSpareIndex(t, ctx, prov)
	t.Cleanup(func() { deleteLogsIndex(ctx, prov, spareID) })

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
	// Note: Datadog Logs Index API has eventual consistency for deletes too,
	// so we don't verify the index is immediately gone after deletion.
}

func TestLogsIndex_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsIndex(t)
	name := testLogsIndexName("update")

	props, _ := json.Marshal(logsIndexProps{
		Name:   name,
		Filter: logsIndexFilter{Query: stringPtr("source:test")},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsIndex,
		Label:        "test-logs-index",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	waitForIndexPropagation(t, ctx, prov, nativeID)

	// Create spare so cleanup can delete the test index.
	spareID := createSpareIndex(t, ctx, prov)
	t.Cleanup(func() {
		deleteLogsIndex(ctx, prov, nativeID)
		deleteLogsIndex(ctx, prov, spareID)
	})

	// Update — change filter and add exclusion
	desiredProps, _ := json.Marshal(logsIndexProps{
		Name: name,
		Filter: logsIndexFilter{
			Query: stringPtr("source:test OR source:api"),
		},
		ExclusionFilters: []logsExclusion{
			{
				Name:      "exclude-debug",
				IsEnabled: boolPtr(true),
				Filter: &logsExclusionFilter{
					Query:      stringPtr("status:debug"),
					SampleRate: 1.0,
				},
			},
		},
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeLogsIndex,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify update result properties directly (avoids eventual consistency)
	var updateProps logsIndexProps
	require.NoError(t, json.Unmarshal(updateResult.ProgressResult.ResourceProperties, &updateProps))
	assert.Equal(t, "source:test OR source:api", *updateProps.Filter.Query)
	if assert.Len(t, updateProps.ExclusionFilters, 1) {
		assert.Equal(t, "exclude-debug", updateProps.ExclusionFilters[0].Name)
	}
}

func TestLogsIndex_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsIndex(t)
	name := testLogsIndexName("list")

	props, _ := json.Marshal(logsIndexProps{
		Name:   name,
		Filter: logsIndexFilter{Query: stringPtr("source:test")},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsIndex,
		Label:        "test-logs-index",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	waitForIndexPropagation(t, ctx, prov, nativeID)

	// Create spare for cleanup.
	spareID := createSpareIndex(t, ctx, prov)
	t.Cleanup(func() {
		deleteLogsIndex(ctx, prov, nativeID)
		deleteLogsIndex(ctx, prov, spareID)
	})

	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeLogsIndex,
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
	assert.True(t, found, "Created logs index %s should appear in List results", nativeID)
	t.Logf("List returned %d indexes", len(listResult.NativeIDs))
}

func TestLogsIndex_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsIndex(t)
	name := testLogsIndexName("delete-idem")

	props, _ := json.Marshal(logsIndexProps{
		Name:   name,
		Filter: logsIndexFilter{Query: stringPtr("source:test")},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsIndex,
		Label:        "test-logs-index",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	waitForIndexPropagation(t, ctx, prov, nativeID)

	// Create spare so we can delete the test index.
	spareID := createSpareIndex(t, ctx, prov)
	t.Cleanup(func() { deleteLogsIndex(ctx, prov, spareID) })

	// Delete once
	_, err = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)

	// Delete again — should succeed (idempotent)
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

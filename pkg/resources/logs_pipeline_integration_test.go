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

func newTestLogsPipeline(t *testing.T) *LogsPipeline {
	t.Helper()
	return &LogsPipeline{Client: newTestClient(t)}
}

func testPipelineName(suffix string) string {
	return fmt.Sprintf("formae-integration-test-%s-%d", suffix, time.Now().Unix())
}

func deleteLogsPipeline(ctx context.Context, prov *LogsPipeline, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

func TestLogsPipeline_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsPipeline(t)
	name := testPipelineName("lifecycle")

	enabled := true
	query := "source:test"
	props, _ := json.Marshal(logsPipelineProps{
		Name:      name,
		IsEnabled: &enabled,
		Filter:    &logsPipelineFilter{Query: &query},
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsPipeline,
		Label:        "test-pipeline",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.NotEmpty(t, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created logs pipeline: %s (%s)", name, nativeID)
	t.Cleanup(func() { deleteLogsPipeline(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeLogsPipeline,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeLogsPipeline, readResult.ResourceType)

	var readProps logsPipelineProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, name, readProps.Name)
	assert.Equal(t, "source:test", *readProps.Filter.Query)

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

func TestLogsPipeline_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsPipeline(t)
	name := testPipelineName("update")

	enabled := true
	query := "source:test"
	props, _ := json.Marshal(logsPipelineProps{
		Name:      name,
		IsEnabled: &enabled,
		Filter:    &logsPipelineFilter{Query: &query},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsPipeline,
		Label:        "test-pipeline",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteLogsPipeline(ctx, prov, nativeID) })

	// Update — change filter
	newQuery := "source:test OR source:api"
	desiredProps, _ := json.Marshal(logsPipelineProps{
		Name:      name,
		IsEnabled: &enabled,
		Filter:    &logsPipelineFilter{Query: &newQuery},
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeLogsPipeline,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	var updateProps logsPipelineProps
	require.NoError(t, json.Unmarshal(updateResult.ProgressResult.ResourceProperties, &updateProps))
	assert.Equal(t, "source:test OR source:api", *updateProps.Filter.Query)
}

func TestLogsPipeline_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsPipeline(t)
	name := testPipelineName("list")

	enabled := true
	query := "source:test"
	props, _ := json.Marshal(logsPipelineProps{
		Name:      name,
		IsEnabled: &enabled,
		Filter:    &logsPipelineFilter{Query: &query},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsPipeline,
		Label:        "test-pipeline",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteLogsPipeline(ctx, prov, nativeID) })

	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeLogsPipeline,
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
	assert.True(t, found, "Created pipeline %s should appear in List results", nativeID)
	t.Logf("List returned %d pipelines", len(listResult.NativeIDs))
}

func TestLogsPipeline_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsPipeline(t)
	name := testPipelineName("delete-idem")

	enabled := true
	query := "source:test"
	props, _ := json.Marshal(logsPipelineProps{
		Name:      name,
		IsEnabled: &enabled,
		Filter:    &logsPipelineFilter{Query: &query},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsPipeline,
		Label:        "test-pipeline",
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

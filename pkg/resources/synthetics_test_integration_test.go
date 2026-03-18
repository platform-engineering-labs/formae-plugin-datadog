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

func newTestSyntheticsTest(t *testing.T) *SyntheticsTest {
	t.Helper()
	return &SyntheticsTest{Client: newTestClient(t)}
}

func testSyntheticsName(suffix string) string {
	return fmt.Sprintf("formae-integration-test-%s-%d", suffix, time.Now().Unix())
}

func deleteSyntheticsTest(ctx context.Context, prov *SyntheticsTest, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

// minimalAPITestConfig returns a minimal HTTP API test config as JSON.
func minimalAPITestConfig() *string {
	cfg := `{"request":{"method":"GET","url":"https://httpbin.org/get"},"assertions":[{"type":"statusCode","operator":"is","target":200}]}`
	return &cfg
}

// minimalAPITestOptions returns minimal test options as JSON.
func minimalAPITestOptions() *string {
	opts := `{"tick_every":900}`
	return &opts
}

func TestSyntheticsTest_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestSyntheticsTest(t)
	name := testSyntheticsName("lifecycle")

	paused := "paused"
	props, _ := json.Marshal(syntheticsTestProps{
		Name:      name,
		TestType:  "api",
		Message:   "Test synthetics test",
		Status:    &paused,
		Locations: []string{"aws:us-east-1"},
		Config:    minimalAPITestConfig(),
		Options:   minimalAPITestOptions(),
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSyntheticsTest,
		Label:        "test-synthetics",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.NotEmpty(t, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created synthetics test: %s (%s)", name, nativeID)
	t.Cleanup(func() { deleteSyntheticsTest(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeSyntheticsTest,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeSyntheticsTest, readResult.ResourceType)

	var readProps syntheticsTestProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, name, readProps.Name)

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

func TestSyntheticsTest_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestSyntheticsTest(t)
	name := testSyntheticsName("update")

	paused := "paused"
	props, _ := json.Marshal(syntheticsTestProps{
		Name:      name,
		TestType:  "api",
		Message:   "Original message",
		Status:    &paused,
		Locations: []string{"aws:us-east-1"},
		Config:    minimalAPITestConfig(),
		Options:   minimalAPITestOptions(),
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSyntheticsTest,
		Label:        "test-synthetics",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteSyntheticsTest(ctx, prov, nativeID) })

	// Update — change message
	desiredProps, _ := json.Marshal(syntheticsTestProps{
		Name:      name,
		TestType:  "api",
		Message:   "Updated message",
		Status:    &paused,
		Locations: []string{"aws:us-east-1"},
		Config:    minimalAPITestConfig(),
		Options:   minimalAPITestOptions(),
		Tags:      []string{"env:test"},
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeSyntheticsTest,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	var updateProps syntheticsTestProps
	require.NoError(t, json.Unmarshal(updateResult.ProgressResult.ResourceProperties, &updateProps))
	assert.Equal(t, "Updated message", updateProps.Message)
}

func TestSyntheticsTest_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestSyntheticsTest(t)

	// Just verify List works
	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeSyntheticsTest,
	})
	require.NoError(t, err)
	t.Logf("List returned %d synthetics tests", len(listResult.NativeIDs))
}

func TestSyntheticsTest_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestSyntheticsTest(t)
	name := testSyntheticsName("delete-idem")

	paused := "paused"
	props, _ := json.Marshal(syntheticsTestProps{
		Name:      name,
		TestType:  "api",
		Message:   "Delete idempotency test",
		Status:    &paused,
		Locations: []string{"aws:us-east-1"},
		Config:    minimalAPITestConfig(),
		Options:   minimalAPITestOptions(),
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSyntheticsTest,
		Label:        "test-synthetics",
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

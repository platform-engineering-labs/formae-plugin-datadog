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

func newTestSLO(t *testing.T) *SLO {
	t.Helper()
	return &SLO{Client: newTestClient(t)}
}

func testSLOName(suffix string) string {
	return fmt.Sprintf("formae-integration-test-slo-%s-%d", suffix, time.Now().Unix())
}

// deleteSLO is a cleanup helper that ignores errors (SLO may already be gone).
func deleteSLO(ctx context.Context, prov *SLO, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

func TestSLO_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestSLO(t)
	name := testSLOName("lifecycle")

	props, _ := json.Marshal(sloProps{
		Name:        name,
		SloType:     "metric",
		Description: stringPtr("Integration test SLO"),
		Tags:        []string{"env:test", "managed-by:formae"},
		Thresholds: []sloThreshold{
			{Target: 99.0, Timeframe: "7d", Warning: float64Ptr(99.5)},
		},
		Query: &sloQueryProps{
			Numerator:   "sum:my.metric.good{*}.as_count()",
			Denominator: "sum:my.metric.total{*}.as_count()",
		},
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSLO,
		Label:        "test-slo",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.NotEmpty(t, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created SLO: %s (ID: %s)", name, nativeID)
	t.Cleanup(func() { deleteSLO(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeSLO,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeSLO, readResult.ResourceType)

	var readProps sloProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, name, readProps.Name)
	assert.Equal(t, "metric", readProps.SloType)
	assert.NotNil(t, readProps.Id)
	assert.Len(t, readProps.Thresholds, 1)
	assert.Equal(t, 99.0, readProps.Thresholds[0].Target)

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID: nativeID,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)

	// Verify gone — Read returns (result, nil) with ErrorCode set, not (nil, error)
	goneResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeSLO,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, goneResult.ErrorCode, "Read after delete should return an error code")
}

func TestSLO_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestSLO(t)
	name := testSLOName("update")

	props, _ := json.Marshal(sloProps{
		Name:    name,
		SloType: "metric",
		Thresholds: []sloThreshold{
			{Target: 99.0, Timeframe: "7d"},
		},
		Query: &sloQueryProps{
			Numerator:   "sum:my.metric.good{*}.as_count()",
			Denominator: "sum:my.metric.total{*}.as_count()",
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSLO,
		Label:        "test-slo",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteSLO(ctx, prov, nativeID) })

	// Update
	updatedName := name + "-updated"
	desiredProps, _ := json.Marshal(sloProps{
		Name:        updatedName,
		SloType:     "metric",
		Description: stringPtr("Updated SLO description"),
		Tags:        []string{"env:test", "managed-by:formae", "updated:true"},
		Thresholds: []sloThreshold{
			{Target: 99.5, Timeframe: "7d", Warning: float64Ptr(99.9)},
		},
		Query: &sloQueryProps{
			Numerator:   "sum:my.metric.good{*}.as_count()",
			Denominator: "sum:my.metric.total{*}.as_count()",
		},
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeSLO,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify update
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeSLO,
	})
	require.NoError(t, err)

	var readProps sloProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, updatedName, readProps.Name)
	assert.Equal(t, "Updated SLO description", *readProps.Description)
	assert.Equal(t, 99.5, readProps.Thresholds[0].Target)
}

func TestSLO_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestSLO(t)
	name := testSLOName("list")

	props, _ := json.Marshal(sloProps{
		Name:    name,
		SloType: "metric",
		Thresholds: []sloThreshold{
			{Target: 99.0, Timeframe: "7d"},
		},
		Query: &sloQueryProps{
			Numerator:   "sum:my.metric.good{*}.as_count()",
			Denominator: "sum:my.metric.total{*}.as_count()",
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSLO,
		Label:        "test-slo",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteSLO(ctx, prov, nativeID) })

	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeSLO,
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
	assert.True(t, found, "Created SLO %s should appear in List results", nativeID)
	t.Logf("List returned %d SLOs", len(listResult.NativeIDs))
}

func TestSLO_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestSLO(t)
	name := testSLOName("delete-idem")

	props, _ := json.Marshal(sloProps{
		Name:    name,
		SloType: "metric",
		Thresholds: []sloThreshold{
			{Target: 99.0, Timeframe: "7d"},
		},
		Query: &sloQueryProps{
			Numerator:   "sum:my.metric.good{*}.as_count()",
			Denominator: "sum:my.metric.total{*}.as_count()",
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSLO,
		Label:        "test-slo",
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

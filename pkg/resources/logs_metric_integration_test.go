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

func newTestLogsMetric(t *testing.T) *LogsMetric {
	t.Helper()
	return &LogsMetric{Client: newTestClient(t)}
}

func testLogsMetricID(suffix string) string {
	return fmt.Sprintf("formae.integration.test.%s.%d", suffix, time.Now().Unix())
}

// deleteLogsMetric is a cleanup helper that ignores errors.
func deleteLogsMetric(ctx context.Context, prov *LogsMetric, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

func TestLogsMetric_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsMetric(t)
	metricID := testLogsMetricID("lifecycle")

	props, _ := json.Marshal(logsMetricProps{
		MetricId: metricID,
		Compute: logsMetricCompute{
			AggregationType: "count",
		},
		Filter: &logsMetricFilter{
			Query: stringPtr("service:web"),
		},
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsMetric,
		Label:        "test-logs-metric",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.Equal(t, metricID, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created logs metric: %s", nativeID)
	t.Cleanup(func() { deleteLogsMetric(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeLogsMetric,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeLogsMetric, readResult.ResourceType)

	var readProps logsMetricProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, metricID, readProps.MetricId)
	assert.Equal(t, "count", readProps.Compute.AggregationType)

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)

	// Verify gone — Read returns (result, nil) with ErrorCode set, not (nil, error)
	goneResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeLogsMetric,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, goneResult.ErrorCode, "Read after delete should return an error code")
}

func TestLogsMetric_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsMetric(t)
	metricID := testLogsMetricID("update")

	props, _ := json.Marshal(logsMetricProps{
		MetricId: metricID,
		Compute: logsMetricCompute{
			AggregationType: "count",
		},
		Filter: &logsMetricFilter{
			Query: stringPtr("service:web"),
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsMetric,
		Label:        "test-logs-metric",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteLogsMetric(ctx, prov, nativeID) })

	// Update — change filter and add groupBy
	desiredProps, _ := json.Marshal(logsMetricProps{
		MetricId: metricID,
		Compute: logsMetricCompute{
			AggregationType: "count",
		},
		Filter: &logsMetricFilter{
			Query: stringPtr("service:api"),
		},
		GroupBy: []logsMetricGroupBy{
			{Path: "@http.status_code", TagName: stringPtr("status_code")},
		},
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeLogsMetric,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeLogsMetric,
	})
	require.NoError(t, err)

	var readProps logsMetricProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, "service:api", *readProps.Filter.Query)
	assert.Len(t, readProps.GroupBy, 1)
	assert.Equal(t, "@http.status_code", readProps.GroupBy[0].Path)
}

func TestLogsMetric_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsMetric(t)
	metricID := testLogsMetricID("list")

	props, _ := json.Marshal(logsMetricProps{
		MetricId: metricID,
		Compute: logsMetricCompute{
			AggregationType: "count",
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsMetric,
		Label:        "test-logs-metric",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteLogsMetric(ctx, prov, nativeID) })

	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeLogsMetric,
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
	assert.True(t, found, "Created logs metric %s should appear in List results", nativeID)
}

func TestLogsMetric_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestLogsMetric(t)
	metricID := testLogsMetricID("delete-idem")

	props, _ := json.Marshal(logsMetricProps{
		MetricId: metricID,
		Compute: logsMetricCompute{
			AggregationType: "count",
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeLogsMetric,
		Label:        "test-logs-metric",
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

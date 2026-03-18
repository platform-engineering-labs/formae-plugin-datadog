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

func newTestDowntimeSchedule(t *testing.T) *DowntimeSchedule {
	t.Helper()
	return &DowntimeSchedule{Client: newTestClient(t)}
}

func testDowntimeName(suffix string) string {
	return fmt.Sprintf("formae-integration-test-downtime-%s-%d", suffix, time.Now().Unix())
}

// deleteDowntime is a cleanup helper that ignores errors (downtime may already be gone).
func deleteDowntime(ctx context.Context, prov *DowntimeSchedule, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

func TestDowntimeSchedule_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestDowntimeSchedule(t)
	_ = testDowntimeName("lifecycle")

	props, _ := json.Marshal(downtimeProps{
		Scope:       "*",
		MonitorTags: []string{"*"},
		Message:     stringPtr("Integration test downtime"),
		RecurringSchedule: &recurringSchedProps{
			Timezone: stringPtr("UTC"),
			Recurrences: []recurrenceProps{
				{Duration: "1h", Rrule: "FREQ=DAILY"},
			},
		},
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeDowntimeSchedule,
		Label:        "test-downtime",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.NotEmpty(t, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created downtime: %s", nativeID)
	t.Cleanup(func() { deleteDowntime(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeDowntimeSchedule,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeDowntimeSchedule, readResult.ResourceType)

	var readProps downtimeProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, "*", readProps.Scope)
	assert.NotNil(t, readProps.Id)

	// Delete (cancel)
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{
		NativeID: nativeID,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

func TestDowntimeSchedule_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestDowntimeSchedule(t)

	props, _ := json.Marshal(downtimeProps{
		Scope:       "*",
		MonitorTags: []string{"*"},
		Message:     stringPtr("Original message"),
		RecurringSchedule: &recurringSchedProps{
			Timezone: stringPtr("UTC"),
			Recurrences: []recurrenceProps{
				{Duration: "1h", Rrule: "FREQ=DAILY"},
			},
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeDowntimeSchedule,
		Label:        "test-downtime",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteDowntime(ctx, prov, nativeID) })

	// Update — only change non-schedule fields (scope, message).
	// Datadog rejects schedule changes on in-progress downtimes with
	// "Start times of downtimes in progress cannot be changed".
	desiredProps, _ := json.Marshal(downtimeProps{
		Scope:       "env:staging",
		MonitorTags: []string{"*"},
		Message:     stringPtr("Updated downtime message"),
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeDowntimeSchedule,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	if updateResult.ProgressResult.OperationStatus == resource.OperationStatusFailure {
		t.Logf("Update failed with error code: %s", updateResult.ProgressResult.ErrorCode)
	}
	require.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify update
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeDowntimeSchedule,
	})
	require.NoError(t, err)

	var readProps downtimeProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, "env:staging", readProps.Scope)
	assert.Equal(t, "Updated downtime message", *readProps.Message)
}

func TestDowntimeSchedule_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestDowntimeSchedule(t)

	props, _ := json.Marshal(downtimeProps{
		Scope:       "*",
		MonitorTags: []string{"*"},
		RecurringSchedule: &recurringSchedProps{
			Timezone: stringPtr("UTC"),
			Recurrences: []recurrenceProps{
				{Duration: "1h", Rrule: "FREQ=DAILY"},
			},
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeDowntimeSchedule,
		Label:        "test-downtime",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteDowntime(ctx, prov, nativeID) })

	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeDowntimeSchedule,
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
	assert.True(t, found, "Created downtime %s should appear in List results", nativeID)
	t.Logf("List returned %d downtimes", len(listResult.NativeIDs))
}

func TestDowntimeSchedule_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestDowntimeSchedule(t)

	props, _ := json.Marshal(downtimeProps{
		Scope:       "*",
		MonitorTags: []string{"*"},
		RecurringSchedule: &recurringSchedProps{
			Timezone: stringPtr("UTC"),
			Recurrences: []recurrenceProps{
				{Duration: "1h", Rrule: "FREQ=DAILY"},
			},
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeDowntimeSchedule,
		Label:        "test-downtime",
		Properties:   props,
	})
	require.NoError(t, err)
	nativeID := createResult.ProgressResult.NativeID

	// Cancel once
	_, err = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)

	// Cancel again — should succeed (idempotent)
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

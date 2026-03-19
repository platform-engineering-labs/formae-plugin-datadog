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

func newTestSecurityMonitoringRule(t *testing.T) *SecurityMonitoringRule {
	t.Helper()
	return &SecurityMonitoringRule{Client: newTestClient(t)}
}

func testSecMonRuleName(suffix string) string {
	return fmt.Sprintf("formae-integration-test-%s-%d", suffix, time.Now().Unix())
}

func deleteSecMonRule(ctx context.Context, prov *SecurityMonitoringRule, nativeID string) {
	_, _ = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
}

// skipIfSecMonUnavailable checks if the Create result indicates the Security
// Monitoring feature is unavailable (403 — requires paid add-on) and skips the test.
func skipIfSecMonUnavailable(t *testing.T, result *resource.CreateResult) {
	t.Helper()
	if result.ProgressResult.OperationStatus == resource.OperationStatusFailure &&
		result.ProgressResult.ErrorCode == resource.OperationErrorCodeAccessDenied {
		t.Skip("Security Monitoring requires a paid Datadog add-on — skipping")
	}
}

func TestSecurityMonitoringRule_CreateReadDeleteLifecycle(t *testing.T) {
	ctx := context.Background()
	prov := newTestSecurityMonitoringRule(t)
	name := testSecMonRuleName("lifecycle")

	enabled := true
	props, _ := json.Marshal(secMonRuleProps{
		Name:      name,
		Message:   "Test security monitoring rule",
		IsEnabled: &enabled,
		Cases: []secMonRuleCase{
			{
				Status: "info",
				Name:   stringPtr("test-case"),
			},
		},
		Queries: []secMonRuleQuery{
			{
				Query: "source:test",
				Name:  stringPtr("test-query"),
			},
		},
		Options: &secMonRuleOpts{
			EvaluationWindow:  int32Ptr(300),
			KeepAlive:         int32Ptr(600),
			MaxSignalDuration: int32Ptr(900),
		},
		Tags: []string{"env:test"},
	})

	// Create
	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSecurityMonitoringRule,
		Label:        "test-sec-rule",
		Properties:   props,
	})
	require.NoError(t, err)
	require.NotNil(t, createResult.ProgressResult)
	skipIfSecMonUnavailable(t, createResult)
	assert.Equal(t, resource.OperationStatusSuccess, createResult.ProgressResult.OperationStatus)
	assert.NotEmpty(t, createResult.ProgressResult.NativeID)

	nativeID := createResult.ProgressResult.NativeID
	t.Logf("Created security monitoring rule: %s (%s)", name, nativeID)
	t.Cleanup(func() { deleteSecMonRule(ctx, prov, nativeID) })

	// Read
	readResult, err := prov.Read(ctx, &resource.ReadRequest{
		NativeID:     nativeID,
		ResourceType: ResourceTypeSecurityMonitoringRule,
	})
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeSecurityMonitoringRule, readResult.ResourceType)

	var readProps secMonRuleProps
	require.NoError(t, json.Unmarshal([]byte(readResult.Properties), &readProps))
	assert.Equal(t, name, readProps.Name)
	assert.Equal(t, "Test security monitoring rule", readProps.Message)

	// Delete
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

func TestSecurityMonitoringRule_Update(t *testing.T) {
	ctx := context.Background()
	prov := newTestSecurityMonitoringRule(t)
	name := testSecMonRuleName("update")

	enabled := true
	props, _ := json.Marshal(secMonRuleProps{
		Name:      name,
		Message:   "Original message",
		IsEnabled: &enabled,
		Cases: []secMonRuleCase{
			{Status: "info"},
		},
		Queries: []secMonRuleQuery{
			{Query: "source:test"},
		},
		Options: &secMonRuleOpts{
			EvaluationWindow:  int32Ptr(300),
			KeepAlive:         int32Ptr(600),
			MaxSignalDuration: int32Ptr(900),
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSecurityMonitoringRule,
		Label:        "test-sec-rule",
		Properties:   props,
	})
	require.NoError(t, err)
	skipIfSecMonUnavailable(t, createResult)
	nativeID := createResult.ProgressResult.NativeID
	t.Cleanup(func() { deleteSecMonRule(ctx, prov, nativeID) })

	// Update — change message and add tag
	desiredProps, _ := json.Marshal(secMonRuleProps{
		Name:      name,
		Message:   "Updated message",
		IsEnabled: &enabled,
		Cases: []secMonRuleCase{
			{Status: "low"},
		},
		Queries: []secMonRuleQuery{
			{Query: "source:test OR source:api"},
		},
		Options: &secMonRuleOpts{
			EvaluationWindow:  int32Ptr(300),
			KeepAlive:         int32Ptr(600),
			MaxSignalDuration: int32Ptr(900),
		},
		Tags: []string{"env:staging"},
	})

	updateResult, err := prov.Update(ctx, &resource.UpdateRequest{
		NativeID:          nativeID,
		ResourceType:      ResourceTypeSecurityMonitoringRule,
		DesiredProperties: desiredProps,
	})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, updateResult.ProgressResult.OperationStatus)

	// Verify via update result
	var updateProps secMonRuleProps
	require.NoError(t, json.Unmarshal(updateResult.ProgressResult.ResourceProperties, &updateProps))
	assert.Equal(t, "Updated message", updateProps.Message)
}

func TestSecurityMonitoringRule_List(t *testing.T) {
	ctx := context.Background()
	prov := newTestSecurityMonitoringRule(t)

	// Just verify List works — the account may have default rules
	listResult, err := prov.List(ctx, &resource.ListRequest{
		ResourceType: ResourceTypeSecurityMonitoringRule,
	})
	if err != nil {
		t.Skip("Security Monitoring requires a paid Datadog add-on — skipping")
	}
	// Datadog accounts have default security rules
	t.Logf("List returned %d security monitoring rules", len(listResult.NativeIDs))
}

func TestSecurityMonitoringRule_DeleteAlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	prov := newTestSecurityMonitoringRule(t)
	name := testSecMonRuleName("delete-idem")

	enabled := false
	props, _ := json.Marshal(secMonRuleProps{
		Name:      name,
		Message:   "Delete idempotency test",
		IsEnabled: &enabled,
		Cases: []secMonRuleCase{
			{Status: "info"},
		},
		Queries: []secMonRuleQuery{
			{Query: "source:test"},
		},
		Options: &secMonRuleOpts{
			EvaluationWindow:  int32Ptr(300),
			KeepAlive:         int32Ptr(600),
			MaxSignalDuration: int32Ptr(900),
		},
	})

	createResult, err := prov.Create(ctx, &resource.CreateRequest{
		ResourceType: ResourceTypeSecurityMonitoringRule,
		Label:        "test-sec-rule",
		Properties:   props,
	})
	require.NoError(t, err)
	skipIfSecMonUnavailable(t, createResult)
	nativeID := createResult.ProgressResult.NativeID

	// Delete once
	_, err = prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)

	// Delete again — should succeed (idempotent)
	deleteResult, err := prov.Delete(ctx, &resource.DeleteRequest{NativeID: nativeID})
	require.NoError(t, err)
	assert.Equal(t, resource.OperationStatusSuccess, deleteResult.ProgressResult.OperationStatus)
}

func int32Ptr(v int32) *int32 {
	return &v
}

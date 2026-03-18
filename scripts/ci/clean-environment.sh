#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: FSL-1.1-ALv2
#
# Clean Environment Hook
# ======================
# Called before AND after conformance tests. Must be idempotent.
# Deletes Datadog resources matching the test prefix via the API.

set -euo pipefail

TEST_PREFIX="${TEST_PREFIX:-formae-plugin-sdk-test-}"

echo "clean-environment.sh: Cleaning resources with prefix '${TEST_PREFIX}'"

if [[ -z "${DD_API_KEY:-}" ]] || [[ -z "${DD_APP_KEY:-}" ]]; then
    echo "  DD_API_KEY or DD_APP_KEY not set, skipping cleanup"
    exit 0
fi

DD_SITE="${DD_SITE:-datadoghq.com}"
BASE_URL_V1="https://api.${DD_SITE}/api/v1"
BASE_URL_V2="https://api.${DD_SITE}/api/v2"

# Delete monitors matching prefix
echo "  Cleaning monitors..."
MONITORS=$(curl -s -X GET "${BASE_URL_V1}/monitor" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$MONITORS" | jq -r --arg prefix "$TEST_PREFIX" \
    '.[]? | select(.name | startswith($prefix)) | "\(.id) \(.name)"' 2>/dev/null | \
    while read -r monitor_id monitor_name; do
        echo "    Deleting monitor ${monitor_id} (${monitor_name})"
        curl -s -X DELETE "${BASE_URL_V1}/monitor/${monitor_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

# Delete SLOs matching prefix
echo "  Cleaning SLOs..."
SLOS=$(curl -s -X GET "${BASE_URL_V1}/slo" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$SLOS" | jq -r --arg prefix "$TEST_PREFIX" \
    '.data[]? | select(.name | startswith($prefix)) | "\(.id) \(.name)"' 2>/dev/null | \
    while read -r slo_id slo_name; do
        echo "    Deleting SLO ${slo_id} (${slo_name})"
        curl -s -X DELETE "${BASE_URL_V1}/slo/${slo_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

# Cancel downtimes matching test prefix in message
echo "  Cleaning downtimes..."
DOWNTIMES=$(curl -s -X GET "${BASE_URL_V2}/downtime" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$DOWNTIMES" | jq -r --arg prefix "$TEST_PREFIX" \
    '.data[]? | select(.attributes.message != null) | select(.attributes.message | contains($prefix)) | .id' 2>/dev/null | \
    while read -r downtime_id; do
        echo "    Canceling downtime ${downtime_id}"
        curl -s -X DELETE "${BASE_URL_V2}/downtime/${downtime_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

# Delete logs indexes matching prefix
echo "  Cleaning logs indexes..."
INDEXES=$(curl -s -X GET "${BASE_URL_V1}/logs/config/indexes" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$INDEXES" | jq -r --arg prefix "$TEST_PREFIX" \
    '.indexes[]? | select(.name | startswith($prefix)) | .name' 2>/dev/null | \
    while read -r index_name; do
        echo "    Deleting logs index ${index_name}"
        curl -s -X DELETE "${BASE_URL_V1}/logs/config/indexes/${index_name}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

# Delete logs metrics matching prefix (dots replaced with periods in metric IDs)
echo "  Cleaning logs metrics..."
METRICS=$(curl -s -X GET "${BASE_URL_V2}/logs/config/metrics" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

# Test prefix uses dots in metric IDs: formae.plugin.sdk.test.*
METRIC_PREFIX="formae.plugin.sdk.test."
echo "$METRICS" | jq -r --arg prefix "$METRIC_PREFIX" \
    '.data[]? | select(.id | startswith($prefix)) | .id' 2>/dev/null | \
    while read -r metric_id; do
        echo "    Deleting logs metric ${metric_id}"
        curl -s -X DELETE "${BASE_URL_V2}/logs/config/metrics/${metric_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

# Delete logs archives matching prefix
echo "  Cleaning logs archives..."
ARCHIVES=$(curl -s -X GET "${BASE_URL_V2}/logs/config/archives" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$ARCHIVES" | jq -r --arg prefix "$TEST_PREFIX" \
    '.data[]? | select(.attributes.name | startswith($prefix)) | .id' 2>/dev/null | \
    while read -r archive_id; do
        echo "    Deleting logs archive ${archive_id}"
        curl -s -X DELETE "${BASE_URL_V2}/logs/config/archives/${archive_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

# Delete roles matching prefix
echo "  Cleaning roles..."
ROLES=$(curl -s -X GET "${BASE_URL_V2}/roles" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$ROLES" | jq -r --arg prefix "$TEST_PREFIX" \
    '.data[]? | select(.attributes.name | startswith($prefix)) | "\(.id) \(.attributes.name)"' 2>/dev/null | \
    while read -r role_id role_name; do
        echo "    Deleting role ${role_id} (${role_name})"
        curl -s -X DELETE "${BASE_URL_V2}/roles/${role_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

# Delete teams matching prefix in handle
echo "  Cleaning teams..."
TEAMS=$(curl -s -X GET "${BASE_URL_V2}/team" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$TEAMS" | jq -r --arg prefix "$TEST_PREFIX" \
    '.data[]? | select(.attributes.handle | startswith($prefix)) | "\(.id) \(.attributes.handle)"' 2>/dev/null | \
    while read -r team_id team_handle; do
        echo "    Deleting team ${team_id} (${team_handle})"
        curl -s -X DELETE "${BASE_URL_V2}/team/${team_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

# Delete dashboards matching prefix
echo "  Cleaning dashboards..."
DASHBOARDS=$(curl -s -X GET "${BASE_URL_V1}/dashboard" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$DASHBOARDS" | jq -r --arg prefix "$TEST_PREFIX" \
    '.dashboards[]? | select(.title | startswith($prefix)) | "\(.id) \(.title)"' 2>/dev/null | \
    while read -r dash_id dash_title; do
        echo "    Deleting dashboard ${dash_id} (${dash_title})"
        curl -s -X DELETE "${BASE_URL_V1}/dashboard/${dash_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

# Delete synthetics tests matching prefix
echo "  Cleaning synthetics tests..."
SYNTH_TESTS=$(curl -s -X GET "${BASE_URL_V1}/synthetics/tests" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

SYNTH_IDS=$(echo "$SYNTH_TESTS" | jq -r --arg prefix "$TEST_PREFIX" \
    '[.tests[]? | select(.name | startswith($prefix)) | .public_id] | if length > 0 then . else empty end' 2>/dev/null)

if [[ -n "${SYNTH_IDS:-}" ]]; then
    echo "    Deleting synthetics tests: ${SYNTH_IDS}"
    curl -s -X POST "${BASE_URL_V1}/synthetics/tests/delete" \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "{\"public_ids\": ${SYNTH_IDS}}" || true
fi

# Delete logs pipelines matching prefix
echo "  Cleaning logs pipelines..."
PIPELINES=$(curl -s -X GET "${BASE_URL_V1}/logs/config/pipelines" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$PIPELINES" | jq -r --arg prefix "$TEST_PREFIX" \
    '.[]? | select(.name | startswith($prefix)) | "\(.id) \(.name)"' 2>/dev/null | \
    while read -r pipeline_id pipeline_name; do
        echo "    Deleting logs pipeline ${pipeline_id} (${pipeline_name})"
        curl -s -X DELETE "${BASE_URL_V1}/logs/config/pipelines/${pipeline_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

echo "clean-environment.sh: Cleanup complete"

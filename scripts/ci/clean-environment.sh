#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: FSL-1.1-ALv2
#
# Clean Environment Hook
# ======================
# Called before AND after conformance tests. Must be idempotent.
# Deletes Datadog monitors matching the test prefix via the v1 API.

set -euo pipefail

TEST_PREFIX="${TEST_PREFIX:-formae-plugin-sdk-test-}"

echo "clean-environment.sh: Cleaning resources with prefix '${TEST_PREFIX}'"

if [[ -z "${DD_API_KEY:-}" ]] || [[ -z "${DD_APP_KEY:-}" ]]; then
    echo "  DD_API_KEY or DD_APP_KEY not set, skipping cleanup"
    exit 0
fi

DD_SITE="${DD_SITE:-datadoghq.com}"
BASE_URL="https://api.${DD_SITE}/api/v1"

# Delete monitors matching prefix
echo "  Cleaning monitors..."
MONITORS=$(curl -s -X GET "${BASE_URL}/monitor" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -H "Content-Type: application/json")

echo "$MONITORS" | jq -r --arg prefix "$TEST_PREFIX" \
    '.[]? | select(.name | startswith($prefix)) | "\(.id) \(.name)"' 2>/dev/null | \
    while read -r monitor_id monitor_name; do
        echo "    Deleting monitor ${monitor_id} (${monitor_name})"
        curl -s -X DELETE "${BASE_URL}/monitor/${monitor_id}" \
            -H "DD-API-KEY: ${DD_API_KEY}" \
            -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" || true
    done

echo "clean-environment.sh: Cleanup complete"

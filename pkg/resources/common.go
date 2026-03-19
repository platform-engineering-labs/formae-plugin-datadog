// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package resources

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// mapHTTPError maps HTTP response status codes to formae OperationErrorCode.
// Datadog SDK returns (result, *http.Response, error) tuples — the response
// provides the most reliable status code for error classification.
func mapHTTPError(resp *http.Response, err error) resource.OperationErrorCode {
	if err == nil {
		return ""
	}

	if resp != nil {
		switch resp.StatusCode {
		case 400:
			return resource.OperationErrorCodeInvalidRequest
		case 401:
			return resource.OperationErrorCodeInvalidCredentials
		case 403:
			return resource.OperationErrorCodeAccessDenied
		case 404:
			return resource.OperationErrorCodeNotFound
		case 409:
			return resource.OperationErrorCodeResourceConflict
		case 429:
			return resource.OperationErrorCodeThrottling
		case 500:
			return resource.OperationErrorCodeServiceInternalError
		case 502, 503:
			return resource.OperationErrorCodeServiceInternalError
		case 504:
			return resource.OperationErrorCodeServiceTimeout
		}
	}

	return resource.OperationErrorCodeGeneralServiceException
}

// isDeleteSuccessError returns true if the HTTP response indicates the
// resource is already deleted. For delete operations, 404 means the goal
// is achieved (resource doesn't exist).
func isDeleteSuccessError(resp *http.Response) bool {
	return resp != nil && resp.StatusCode == 404
}

// int64ToNativeID converts an int64 ID to a string NativeID.
func int64ToNativeID(id int64) string {
	return strconv.FormatInt(id, 10)
}

// nativeIDToInt64 converts a string NativeID to an int64 ID.
func nativeIDToInt64(nativeID string) (int64, error) {
	return strconv.ParseInt(nativeID, 10, 64)
}

// parseISO8601 parses an ISO-8601 datetime string.
func parseISO8601(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// sortedTags returns a sorted copy of tags. Datadog returns tags in
// arbitrary order, but formae compares properties as JSON strings —
// non-deterministic order causes spurious "change detected" rejections.
func sortedTags(tags []string) []string {
	if len(tags) == 0 {
		return tags
	}
	sorted := make([]string, len(tags))
	copy(sorted, tags)
	sort.Strings(sorted)
	return sorted
}

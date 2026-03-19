// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/prov"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/registry"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const ResourceTypeLogsIndex = "Datadog::Logs::Index"

func init() {
	registry.Register(ResourceTypeLogsIndex, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &LogsIndex{Client: c}
	})
}

type LogsIndex struct {
	Client *client.Client
}

type logsIndexProps struct {
	Name             string              `json:"name"`
	Filter           logsIndexFilter     `json:"filter"`
	ExclusionFilters []logsExclusion     `json:"exclusionFilters,omitempty"`
	DailyLimit       *int64              `json:"dailyLimit,omitempty"`
	NumRetentionDays *int64              `json:"numRetentionDays,omitempty"`
}

type logsIndexFilter struct {
	Query *string `json:"query,omitempty"`
}

type logsExclusion struct {
	Name      string               `json:"name"`
	IsEnabled *bool                `json:"isEnabled,omitempty"`
	Filter    *logsExclusionFilter `json:"filter,omitempty"`
}

type logsExclusionFilter struct {
	Query      *string `json:"query,omitempty"`
	SampleRate float64 `json:"sampleRate"`
}

func (l *LogsIndex) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props logsIndexProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	body := buildLogsIndex(props)

	api := datadogV1.NewLogsIndexesApi(l.Client.ApiClient)
	resp, httpResp, err := api.CreateLogsIndex(l.Client.Ctx, body)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       mapHTTPError(httpResp, err),
			},
		}, nil
	}

	nativeID := resp.GetName()
	propsJSON := marshalLogsIndexProps(&resp)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (l *LogsIndex) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV1.NewLogsIndexesApi(l.Client.ApiClient)
	resp, httpResp, err := api.GetLogsIndex(l.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	propsJSON := marshalLogsIndexProps(&resp)

	return &resource.ReadResult{
		ResourceType: ResourceTypeLogsIndex,
		Properties:   string(propsJSON),
	}, nil
}

func (l *LogsIndex) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props logsIndexProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	updateBody := datadogV1.LogsIndexUpdateRequest{
		Filter:           buildLogsFilter(props.Filter),
		ExclusionFilters: buildExclusions(props.ExclusionFilters),
	}
	if props.DailyLimit != nil {
		updateBody.DailyLimit = props.DailyLimit
	}
	if props.NumRetentionDays != nil {
		updateBody.NumRetentionDays = props.NumRetentionDays
	}

	api := datadogV1.NewLogsIndexesApi(l.Client.ApiClient)
	resp, httpResp, err := api.UpdateLogsIndex(l.Client.Ctx, request.NativeID, updateBody)
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       mapHTTPError(httpResp, err),
				NativeID:        request.NativeID,
			},
		}, nil
	}

	propsJSON := marshalLogsIndexProps(&resp)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (l *LogsIndex) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV1.NewLogsIndexesApi(l.Client.ApiClient)
	httpResp, err := api.DeleteLogsIndex(l.Client.Ctx, request.NativeID)
	// Datadog returns 403 (not 404) when deleting an already-deleted index.
	// Treat both as idempotent success.
	if err != nil && !isDeleteSuccessError(httpResp) && (httpResp == nil || httpResp.StatusCode != 403) {
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       mapHTTPError(httpResp, err),
				NativeID:        request.NativeID,
			},
		}, nil
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (l *LogsIndex) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (l *LogsIndex) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV1.NewLogsIndexesApi(l.Client.ApiClient)
	resp, httpResp, err := api.ListLogIndexes(l.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list logs indexes: %w (status: %d)", err, httpResp.StatusCode)
	}

	indexes := resp.GetIndexes()
	nativeIDs := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		nativeIDs = append(nativeIDs, idx.GetName())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func buildLogsFilter(f logsIndexFilter) datadogV1.LogsFilter {
	filter := datadogV1.LogsFilter{}
	if f.Query != nil {
		filter.Query = f.Query
	}
	return filter
}

func buildExclusions(exclusions []logsExclusion) []datadogV1.LogsExclusion {
	result := make([]datadogV1.LogsExclusion, 0, len(exclusions))
	for _, e := range exclusions {
		exc := datadogV1.LogsExclusion{
			Name: e.Name,
		}
		if e.IsEnabled != nil {
			exc.IsEnabled = e.IsEnabled
		}
		if e.Filter != nil {
			exc.Filter = &datadogV1.LogsExclusionFilter{
				SampleRate: e.Filter.SampleRate,
			}
			if e.Filter.Query != nil {
				exc.Filter.Query = e.Filter.Query
			}
		}
		result = append(result, exc)
	}
	return result
}

func buildLogsIndex(props logsIndexProps) datadogV1.LogsIndex {
	body := datadogV1.LogsIndex{
		Name:             props.Name,
		Filter:           buildLogsFilter(props.Filter),
		ExclusionFilters: buildExclusions(props.ExclusionFilters),
	}
	if props.DailyLimit != nil {
		body.DailyLimit = props.DailyLimit
	}
	if props.NumRetentionDays != nil {
		body.NumRetentionDays = props.NumRetentionDays
	}
	return body
}

func marshalLogsIndexProps(idx *datadogV1.LogsIndex) json.RawMessage {
	props := logsIndexProps{
		Name: idx.GetName(),
	}

	filter := idx.GetFilter()
	if filter.Query != nil {
		props.Filter.Query = filter.Query
	}

	if idx.DailyLimit != nil {
		props.DailyLimit = idx.DailyLimit
	}
	if idx.NumRetentionDays != nil {
		props.NumRetentionDays = idx.NumRetentionDays
	}

	exclusions := idx.GetExclusionFilters()
	if len(exclusions) > 0 {
		props.ExclusionFilters = make([]logsExclusion, 0, len(exclusions))
		for _, e := range exclusions {
			exc := logsExclusion{
				Name: e.GetName(),
			}
			if e.IsEnabled != nil {
				exc.IsEnabled = e.IsEnabled
			}
			if e.Filter != nil {
				exc.Filter = &logsExclusionFilter{
					SampleRate: e.Filter.GetSampleRate(),
				}
				if e.Filter.Query != nil {
					exc.Filter.Query = e.Filter.Query
				}
			}
			props.ExclusionFilters = append(props.ExclusionFilters, exc)
		}
	}

	d, _ := json.Marshal(props)
	return d
}

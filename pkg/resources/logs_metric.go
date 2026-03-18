// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/prov"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/registry"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const ResourceTypeLogsMetric = "Datadog::Logs::Metric"

func init() {
	registry.Register(ResourceTypeLogsMetric, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &LogsMetric{Client: c}
	})
}

type LogsMetric struct {
	Client *client.Client
}

type logsMetricProps struct {
	MetricId string              `json:"metricId"`
	Compute  logsMetricCompute   `json:"compute"`
	Filter   *logsMetricFilter   `json:"filter,omitempty"`
	GroupBy  []logsMetricGroupBy `json:"groupBy,omitempty"`
}

type logsMetricCompute struct {
	AggregationType    string `json:"aggregationType"`
	Path               *string `json:"path,omitempty"`
	IncludePercentiles *bool   `json:"includePercentiles,omitempty"`
}

type logsMetricFilter struct {
	Query *string `json:"query,omitempty"`
}

type logsMetricGroupBy struct {
	Path    string  `json:"path"`
	TagName *string `json:"tagName,omitempty"`
}

func (m *LogsMetric) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props logsMetricProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	compute := datadogV2.LogsMetricCompute{
		AggregationType: datadogV2.LogsMetricComputeAggregationType(props.Compute.AggregationType),
	}
	if props.Compute.Path != nil {
		compute.Path = props.Compute.Path
	}
	if props.Compute.IncludePercentiles != nil {
		compute.IncludePercentiles = props.Compute.IncludePercentiles
	}

	attrs := datadogV2.LogsMetricCreateAttributes{
		Compute: compute,
	}
	if props.Filter != nil && props.Filter.Query != nil {
		attrs.Filter = &datadogV2.LogsMetricFilter{
			Query: props.Filter.Query,
		}
	}
	if len(props.GroupBy) > 0 {
		groupBy := make([]datadogV2.LogsMetricGroupBy, 0, len(props.GroupBy))
		for _, g := range props.GroupBy {
			gb := datadogV2.LogsMetricGroupBy{Path: g.Path}
			if g.TagName != nil {
				gb.TagName = g.TagName
			}
			groupBy = append(groupBy, gb)
		}
		attrs.GroupBy = groupBy
	}

	body := datadogV2.LogsMetricCreateRequest{
		Data: datadogV2.LogsMetricCreateData{
			Attributes: attrs,
			Id:         props.MetricId,
			Type:       datadogV2.LOGSMETRICTYPE_LOGS_METRICS,
		},
	}

	api := datadogV2.NewLogsMetricsApi(m.Client.ApiClient)
	resp, httpResp, err := api.CreateLogsMetric(m.Client.Ctx, body)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       mapHTTPError(httpResp, err),
			},
		}, nil
	}

	data := resp.GetData()
	nativeID := data.GetId()
	propsJSON := marshalLogsMetricProps(&data)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (m *LogsMetric) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV2.NewLogsMetricsApi(m.Client.ApiClient)
	resp, httpResp, err := api.GetLogsMetric(m.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	data := resp.GetData()
	propsJSON := marshalLogsMetricProps(&data)

	return &resource.ReadResult{
		ResourceType: ResourceTypeLogsMetric,
		Properties:   string(propsJSON),
	}, nil
}

func (m *LogsMetric) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props logsMetricProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	attrs := datadogV2.LogsMetricUpdateAttributes{}

	// Only includePercentiles can be updated on compute
	if props.Compute.IncludePercentiles != nil {
		attrs.Compute = &datadogV2.LogsMetricUpdateCompute{
			IncludePercentiles: props.Compute.IncludePercentiles,
		}
	}

	if props.Filter != nil && props.Filter.Query != nil {
		attrs.Filter = &datadogV2.LogsMetricFilter{
			Query: props.Filter.Query,
		}
	}
	if len(props.GroupBy) > 0 {
		groupBy := make([]datadogV2.LogsMetricGroupBy, 0, len(props.GroupBy))
		for _, g := range props.GroupBy {
			gb := datadogV2.LogsMetricGroupBy{Path: g.Path}
			if g.TagName != nil {
				gb.TagName = g.TagName
			}
			groupBy = append(groupBy, gb)
		}
		attrs.GroupBy = groupBy
	}

	body := datadogV2.LogsMetricUpdateRequest{
		Data: datadogV2.LogsMetricUpdateData{
			Attributes: attrs,
			Type:       datadogV2.LOGSMETRICTYPE_LOGS_METRICS,
		},
	}

	api := datadogV2.NewLogsMetricsApi(m.Client.ApiClient)
	resp, httpResp, err := api.UpdateLogsMetric(m.Client.Ctx, request.NativeID, body)
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

	data := resp.GetData()
	propsJSON := marshalLogsMetricProps(&data)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (m *LogsMetric) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV2.NewLogsMetricsApi(m.Client.ApiClient)
	httpResp, err := api.DeleteLogsMetric(m.Client.Ctx, request.NativeID)
	if err != nil && !isDeleteSuccessError(httpResp) {
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

func (m *LogsMetric) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (m *LogsMetric) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV2.NewLogsMetricsApi(m.Client.ApiClient)
	resp, httpResp, err := api.ListLogsMetrics(m.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list logs metrics: %w (status: %d)", err, httpResp.StatusCode)
	}

	data := resp.GetData()
	nativeIDs := make([]string, 0, len(data))
	for _, metric := range data {
		nativeIDs = append(nativeIDs, metric.GetId())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func marshalLogsMetricProps(data *datadogV2.LogsMetricResponseData) json.RawMessage {
	props := logsMetricProps{
		MetricId: data.GetId(),
	}

	attrs := data.GetAttributes()

	if attrs.Compute != nil {
		props.Compute.AggregationType = string(*attrs.Compute.AggregationType)
		if attrs.Compute.Path != nil {
			props.Compute.Path = attrs.Compute.Path
		}
		if attrs.Compute.IncludePercentiles != nil {
			props.Compute.IncludePercentiles = attrs.Compute.IncludePercentiles
		}
	}

	if attrs.Filter != nil && attrs.Filter.Query != nil {
		props.Filter = &logsMetricFilter{
			Query: attrs.Filter.Query,
		}
	}

	if len(attrs.GroupBy) > 0 {
		props.GroupBy = make([]logsMetricGroupBy, 0, len(attrs.GroupBy))
		for _, g := range attrs.GroupBy {
			gb := logsMetricGroupBy{}
			if g.Path != nil {
				gb.Path = *g.Path
			}
			if g.TagName != nil {
				gb.TagName = g.TagName
			}
			props.GroupBy = append(props.GroupBy, gb)
		}
	}

	d, _ := json.Marshal(props)
	return d
}

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

const ResourceTypeLogsPipeline = "Datadog::Logs::Pipeline"

func init() {
	registry.Register(ResourceTypeLogsPipeline, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &LogsPipeline{Client: c}
	})
}

type LogsPipeline struct {
	Client *client.Client
}

type logsPipelineProps struct {
	Name       string             `json:"name"`
	Filter     *logsPipelineFilter `json:"filter,omitempty"`
	IsEnabled  *bool              `json:"isEnabled,omitempty"`
	Processors *string            `json:"processors,omitempty"`
	Tags       []string           `json:"tags,omitempty"`
}

type logsPipelineFilter struct {
	Query *string `json:"query,omitempty"`
}

func (l *LogsPipeline) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props logsPipelineProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	body := buildLogsPipeline(props)

	api := datadogV1.NewLogsPipelinesApi(l.Client.ApiClient)
	resp, httpResp, err := api.CreateLogsPipeline(l.Client.Ctx, body)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       mapHTTPError(httpResp, err),
			},
		}, nil
	}

	nativeID := resp.GetId()
	propsJSON := marshalLogsPipelineProps(&resp)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (l *LogsPipeline) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV1.NewLogsPipelinesApi(l.Client.ApiClient)
	resp, httpResp, err := api.GetLogsPipeline(l.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	propsJSON := marshalLogsPipelineProps(&resp)

	return &resource.ReadResult{
		ResourceType: ResourceTypeLogsPipeline,
		Properties:   string(propsJSON),
	}, nil
}

func (l *LogsPipeline) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props logsPipelineProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	body := buildLogsPipeline(props)

	api := datadogV1.NewLogsPipelinesApi(l.Client.ApiClient)
	resp, httpResp, err := api.UpdateLogsPipeline(l.Client.Ctx, request.NativeID, body)
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

	propsJSON := marshalLogsPipelineProps(&resp)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (l *LogsPipeline) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV1.NewLogsPipelinesApi(l.Client.ApiClient)
	httpResp, err := api.DeleteLogsPipeline(l.Client.Ctx, request.NativeID)
	if err != nil && !isDeleteSuccessError(httpResp) && (httpResp == nil || httpResp.StatusCode != 400) {
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

func (l *LogsPipeline) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (l *LogsPipeline) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV1.NewLogsPipelinesApi(l.Client.ApiClient)
	resp, httpResp, err := api.ListLogsPipelines(l.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list logs pipelines: %w (status: %d)", err, httpResp.StatusCode)
	}

	nativeIDs := make([]string, 0, len(resp))
	for _, pipeline := range resp {
		nativeIDs = append(nativeIDs, pipeline.GetId())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func buildLogsPipeline(props logsPipelineProps) datadogV1.LogsPipeline {
	body := *datadogV1.NewLogsPipeline(props.Name)

	if props.Filter != nil && props.Filter.Query != nil {
		body.Filter = &datadogV1.LogsFilter{
			Query: props.Filter.Query,
		}
	}
	if props.IsEnabled != nil {
		body.IsEnabled = props.IsEnabled
	}
	if len(props.Tags) > 0 {
		body.Tags = props.Tags
	}

	// Deserialize processors from raw JSON
	if props.Processors != nil && *props.Processors != "" {
		var processors []datadogV1.LogsProcessor
		if err := json.Unmarshal([]byte(*props.Processors), &processors); err == nil {
			body.Processors = processors
		}
	}

	return body
}

func marshalLogsPipelineProps(pipeline *datadogV1.LogsPipeline) json.RawMessage {
	props := logsPipelineProps{
		Name:      pipeline.GetName(),
		IsEnabled: pipeline.IsEnabled,
	}

	filter := pipeline.GetFilter()
	if filter.Query != nil {
		props.Filter = &logsPipelineFilter{
			Query: filter.Query,
		}
	}

	if len(pipeline.Tags) > 0 {
		props.Tags = sortedTags(pipeline.Tags)
	}

	// Serialize processors as raw JSON
	processors := pipeline.GetProcessors()
	if len(processors) > 0 {
		processorsJSON, err := json.Marshal(processors)
		if err == nil {
			s := string(processorsJSON)
			props.Processors = &s
		}
	}

	d, _ := json.Marshal(props)
	return d
}

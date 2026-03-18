// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/prov"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/registry"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const ResourceTypeSLO = "Datadog::Monitoring::SLO"

func init() {
	registry.Register(ResourceTypeSLO, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &SLO{Client: c}
	})
}

type SLO struct {
	Client *client.Client
}

type sloProps struct {
	Id          *string          `json:"id,omitempty"`
	Name        string           `json:"name"`
	SloType     string           `json:"sloType"`
	Thresholds  []sloThreshold   `json:"thresholds"`
	Description *string          `json:"description,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Query       *sloQueryProps   `json:"query,omitempty"`
	MonitorIds  []int64          `json:"monitorIds,omitempty"`
}

type sloThreshold struct {
	Target    float64  `json:"target"`
	Timeframe string   `json:"timeframe"`
	Warning   *float64 `json:"warning,omitempty"`
}

type sloQueryProps struct {
	Numerator   string `json:"numerator"`
	Denominator string `json:"denominator"`
}

func (s *SLO) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props sloProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	thresholds := make([]datadogV1.SLOThreshold, 0, len(props.Thresholds))
	for _, t := range props.Thresholds {
		threshold := datadogV1.SLOThreshold{
			Target:    t.Target,
			Timeframe: datadogV1.SLOTimeframe(t.Timeframe),
		}
		if t.Warning != nil {
			threshold.SetWarning(*t.Warning)
		}
		thresholds = append(thresholds, threshold)
	}

	body := datadogV1.ServiceLevelObjectiveRequest{
		Name:       props.Name,
		Type:       datadogV1.SLOType(props.SloType),
		Thresholds: thresholds,
	}
	if props.Description != nil {
		body.Description = *datadog.NewNullableString(props.Description)
	}
	if len(props.Tags) > 0 {
		body.Tags = props.Tags
	}
	if props.Query != nil {
		body.Query = &datadogV1.ServiceLevelObjectiveQuery{
			Numerator:   props.Query.Numerator,
			Denominator: props.Query.Denominator,
		}
	}
	if len(props.MonitorIds) > 0 {
		body.MonitorIds = props.MonitorIds
	}

	api := datadogV1.NewServiceLevelObjectivesApi(s.Client.ApiClient)
	resp, httpResp, err := api.CreateSLO(s.Client.Ctx, body)
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
	if len(data) == 0 {
		return nil, fmt.Errorf("SLO create returned empty response")
	}

	slo := data[0]
	nativeID := slo.GetId()
	propsJSON := marshalSloProps(&slo)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (s *SLO) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV1.NewServiceLevelObjectivesApi(s.Client.ApiClient)
	resp, httpResp, err := api.GetSLO(s.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	data := resp.GetData()
	propsJSON := marshalSloResponseData(&data)

	return &resource.ReadResult{
		ResourceType: ResourceTypeSLO,
		Properties:   string(propsJSON),
	}, nil
}

func (s *SLO) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props sloProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	thresholds := make([]datadogV1.SLOThreshold, 0, len(props.Thresholds))
	for _, t := range props.Thresholds {
		threshold := datadogV1.SLOThreshold{
			Target:    t.Target,
			Timeframe: datadogV1.SLOTimeframe(t.Timeframe),
		}
		if t.Warning != nil {
			threshold.SetWarning(*t.Warning)
		}
		thresholds = append(thresholds, threshold)
	}

	body := datadogV1.ServiceLevelObjective{
		Name:       props.Name,
		Type:       datadogV1.SLOType(props.SloType),
		Thresholds: thresholds,
	}
	if props.Description != nil {
		body.Description = *datadog.NewNullableString(props.Description)
	}
	if props.Tags != nil {
		body.Tags = props.Tags
	}
	if props.Query != nil {
		body.Query = &datadogV1.ServiceLevelObjectiveQuery{
			Numerator:   props.Query.Numerator,
			Denominator: props.Query.Denominator,
		}
	}
	if len(props.MonitorIds) > 0 {
		body.MonitorIds = props.MonitorIds
	}

	api := datadogV1.NewServiceLevelObjectivesApi(s.Client.ApiClient)
	resp, httpResp, err := api.UpdateSLO(s.Client.Ctx, request.NativeID, body)
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
	if len(data) == 0 {
		return nil, fmt.Errorf("SLO update returned empty response")
	}

	slo := data[0]
	propsJSON := marshalSloProps(&slo)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (s *SLO) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV1.NewServiceLevelObjectivesApi(s.Client.ApiClient)
	_, httpResp, err := api.DeleteSLO(s.Client.Ctx, request.NativeID)
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

func (s *SLO) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	// SLO operations are synchronous — no async polling needed.
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (s *SLO) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV1.NewServiceLevelObjectivesApi(s.Client.ApiClient)
	resp, httpResp, err := api.ListSLOs(s.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list SLOs: %w (status: %d)", err, httpResp.StatusCode)
	}

	data := resp.GetData()
	nativeIDs := make([]string, 0, len(data))
	for _, slo := range data {
		nativeIDs = append(nativeIDs, slo.GetId())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

// marshalSloProps converts a Datadog ServiceLevelObjective response to JSON properties.
func marshalSloProps(slo *datadogV1.ServiceLevelObjective) json.RawMessage {
	props := sloProps{
		Name:    slo.GetName(),
		SloType: string(slo.GetType()),
	}

	id := slo.GetId()
	props.Id = &id

	if slo.Description.IsSet() && slo.Description.Get() != nil {
		props.Description = slo.Description.Get()
	}
	if len(slo.Tags) > 0 {
		props.Tags = slo.Tags
	}
	if slo.Query != nil {
		props.Query = &sloQueryProps{
			Numerator:   slo.Query.GetNumerator(),
			Denominator: slo.Query.GetDenominator(),
		}
	}
	if len(slo.MonitorIds) > 0 {
		props.MonitorIds = slo.MonitorIds
	}

	thresholds := make([]sloThreshold, 0, len(slo.Thresholds))
	for _, t := range slo.Thresholds {
		st := sloThreshold{
			Target:    t.GetTarget(),
			Timeframe: string(t.GetTimeframe()),
		}
		if t.Warning != nil {
			w := *t.Warning
			st.Warning = &w
		}
		thresholds = append(thresholds, st)
	}
	props.Thresholds = thresholds

	data, _ := json.Marshal(props)
	return data
}

// marshalSloResponseData converts a SLOResponseData to JSON properties.
func marshalSloResponseData(data *datadogV1.SLOResponseData) json.RawMessage {
	props := sloProps{
		Name: data.GetName(),
	}

	if data.Id != nil {
		props.Id = data.Id
	}
	if data.Type != nil {
		props.SloType = string(*data.Type)
	}
	if data.Description.IsSet() && data.Description.Get() != nil {
		props.Description = data.Description.Get()
	}
	if len(data.Tags) > 0 {
		props.Tags = data.Tags
	}
	if data.Query != nil {
		props.Query = &sloQueryProps{
			Numerator:   data.Query.GetNumerator(),
			Denominator: data.Query.GetDenominator(),
		}
	}
	if len(data.MonitorIds) > 0 {
		props.MonitorIds = data.MonitorIds
	}

	thresholds := make([]sloThreshold, 0, len(data.Thresholds))
	for _, t := range data.Thresholds {
		st := sloThreshold{
			Target:    t.GetTarget(),
			Timeframe: string(t.GetTimeframe()),
		}
		if t.Warning != nil {
			w := *t.Warning
			st.Warning = &w
		}
		thresholds = append(thresholds, st)
	}
	props.Thresholds = thresholds

	d, _ := json.Marshal(props)
	return d
}

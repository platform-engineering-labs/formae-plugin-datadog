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

const ResourceTypeMonitor = "Datadog::Monitoring::Monitor"

func init() {
	registry.Register(ResourceTypeMonitor, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &Monitor{Client: c}
	})
}

type Monitor struct {
	Client *client.Client
}

type monitorProps struct {
	Id       *int64               `json:"id,omitempty"`
	Name     string               `json:"name"`
	Type     string               `json:"monitorType"`
	Query    string               `json:"query"`
	Message  *string              `json:"message,omitempty"`
	Priority *int64               `json:"priority,omitempty"`
	Tags     []string             `json:"tags,omitempty"`
	Options  *monitorOptionsProps `json:"options,omitempty"`
}

type monitorOptionsProps struct {
	Thresholds        *monitorThresholdsProps `json:"thresholds,omitempty"`
	NotifyNoData      *bool                   `json:"notifyNoData,omitempty"`
	TimeoutH          *int64                  `json:"timeoutH,omitempty"`
	RenotifyInterval  *int64                  `json:"renotifyInterval,omitempty"`
	EscalationMessage *string                 `json:"escalationMessage,omitempty"`
	IncludeTags       *bool                   `json:"includeTags,omitempty"`
}

type monitorThresholdsProps struct {
	Critical         *float64 `json:"critical,omitempty"`
	Warning          *float64 `json:"warning,omitempty"`
	Ok               *float64 `json:"ok,omitempty"`
	CriticalRecovery *float64 `json:"criticalRecovery,omitempty"`
	WarningRecovery  *float64 `json:"warningRecovery,omitempty"`
}

func (m *Monitor) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props monitorProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	body := datadogV1.Monitor{
		Query: props.Query,
		Type:  datadogV1.MonitorType(props.Type),
	}
	if props.Name != "" {
		body.Name = &props.Name
	}
	if props.Message != nil {
		body.Message = props.Message
	}
	if props.Priority != nil {
		body.SetPriority(*props.Priority)
	}
	if len(props.Tags) > 0 {
		body.Tags = props.Tags
	}
	if props.Options != nil {
		body.Options = buildMonitorOptions(props.Options)
	}

	api := datadogV1.NewMonitorsApi(m.Client.ApiClient)
	resp, httpResp, err := api.CreateMonitor(m.Client.Ctx, body)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       mapHTTPError(httpResp, err),
			},
		}, nil
	}

	nativeID := int64ToNativeID(resp.GetId())
	propsJSON := marshalMonitorProps(&resp)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (m *Monitor) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	monitorID, err := nativeIDToInt64(request.NativeID)
	if err != nil {
		return nil, fmt.Errorf("invalid native ID %q: %w", request.NativeID, err)
	}

	api := datadogV1.NewMonitorsApi(m.Client.ApiClient)
	resp, httpResp, err := api.GetMonitor(m.Client.Ctx, monitorID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	propsJSON := marshalMonitorProps(&resp)

	return &resource.ReadResult{
		ResourceType: ResourceTypeMonitor,
		Properties:   string(propsJSON),
	}, nil
}

func (m *Monitor) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	monitorID, err := nativeIDToInt64(request.NativeID)
	if err != nil {
		return nil, fmt.Errorf("invalid native ID %q: %w", request.NativeID, err)
	}

	var props monitorProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	updateReq := datadogV1.MonitorUpdateRequest{}
	if props.Name != "" {
		updateReq.Name = &props.Name
	}
	updateReq.Query = &props.Query
	updateReq.Type = (*datadogV1.MonitorType)(&props.Type)
	if props.Message != nil {
		updateReq.Message = props.Message
	}
	if props.Priority != nil {
		updateReq.SetPriority(*props.Priority)
	}
	if props.Tags != nil {
		updateReq.Tags = props.Tags
	}
	if props.Options != nil {
		updateReq.Options = buildMonitorOptions(props.Options)
	}

	api := datadogV1.NewMonitorsApi(m.Client.ApiClient)
	resp, httpResp, err := api.UpdateMonitor(m.Client.Ctx, monitorID, updateReq)
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

	propsJSON := marshalMonitorProps(&resp)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (m *Monitor) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	monitorID, err := nativeIDToInt64(request.NativeID)
	if err != nil {
		return nil, fmt.Errorf("invalid native ID %q: %w", request.NativeID, err)
	}

	api := datadogV1.NewMonitorsApi(m.Client.ApiClient)
	_, httpResp, err := api.DeleteMonitor(m.Client.Ctx, monitorID)
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

func (m *Monitor) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	// Monitor operations are synchronous — no async polling needed.
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (m *Monitor) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV1.NewMonitorsApi(m.Client.ApiClient)
	monitors, httpResp, err := api.ListMonitors(m.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list monitors: %w (status: %d)", err, httpResp.StatusCode)
	}

	nativeIDs := make([]string, 0, len(monitors))
	for _, mon := range monitors {
		nativeIDs = append(nativeIDs, int64ToNativeID(mon.GetId()))
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

// buildMonitorOptions converts props to the SDK MonitorOptions type.
func buildMonitorOptions(opts *monitorOptionsProps) *datadogV1.MonitorOptions {
	o := &datadogV1.MonitorOptions{}

	if opts.Thresholds != nil {
		t := &datadogV1.MonitorThresholds{}
		if opts.Thresholds.Critical != nil {
			t.Critical = opts.Thresholds.Critical
		}
		if opts.Thresholds.Warning != nil {
			t.SetWarning(*opts.Thresholds.Warning)
		}
		if opts.Thresholds.Ok != nil {
			t.SetOk(*opts.Thresholds.Ok)
		}
		if opts.Thresholds.CriticalRecovery != nil {
			t.SetCriticalRecovery(*opts.Thresholds.CriticalRecovery)
		}
		if opts.Thresholds.WarningRecovery != nil {
			t.SetWarningRecovery(*opts.Thresholds.WarningRecovery)
		}
		o.Thresholds = t
	}
	if opts.NotifyNoData != nil {
		o.NotifyNoData = opts.NotifyNoData
	}
	if opts.TimeoutH != nil {
		o.SetTimeoutH(*opts.TimeoutH)
	}
	if opts.RenotifyInterval != nil {
		o.SetRenotifyInterval(*opts.RenotifyInterval)
	}
	if opts.EscalationMessage != nil {
		o.EscalationMessage = opts.EscalationMessage
	}
	if opts.IncludeTags != nil {
		o.IncludeTags = opts.IncludeTags
	}

	return o
}

// marshalMonitorProps converts a Datadog Monitor response to JSON properties.
func marshalMonitorProps(mon *datadogV1.Monitor) json.RawMessage {
	props := monitorProps{
		Name:  mon.GetName(),
		Type:  string(mon.GetType()),
		Query: mon.GetQuery(),
	}

	id := mon.GetId()
	props.Id = &id

	if mon.Message != nil {
		props.Message = mon.Message
	}
	if mon.Priority.IsSet() && mon.Priority.Get() != nil {
		props.Priority = mon.Priority.Get()
	}
	if len(mon.Tags) > 0 {
		props.Tags = mon.Tags
	}

	if mon.Options != nil {
		opts := &monitorOptionsProps{}
		hasOpts := false

		if mon.Options.Thresholds != nil {
			t := &monitorThresholdsProps{}
			hasThresholds := false

			if mon.Options.Thresholds.Critical != nil {
				t.Critical = mon.Options.Thresholds.Critical
				hasThresholds = true
			}
			if mon.Options.Thresholds.Warning.IsSet() && mon.Options.Thresholds.Warning.Get() != nil {
				t.Warning = mon.Options.Thresholds.Warning.Get()
				hasThresholds = true
			}
			if mon.Options.Thresholds.Ok.IsSet() && mon.Options.Thresholds.Ok.Get() != nil {
				t.Ok = mon.Options.Thresholds.Ok.Get()
				hasThresholds = true
			}
			if mon.Options.Thresholds.CriticalRecovery.IsSet() && mon.Options.Thresholds.CriticalRecovery.Get() != nil {
				t.CriticalRecovery = mon.Options.Thresholds.CriticalRecovery.Get()
				hasThresholds = true
			}
			if mon.Options.Thresholds.WarningRecovery.IsSet() && mon.Options.Thresholds.WarningRecovery.Get() != nil {
				t.WarningRecovery = mon.Options.Thresholds.WarningRecovery.Get()
				hasThresholds = true
			}

			if hasThresholds {
				opts.Thresholds = t
				hasOpts = true
			}
		}

		if mon.Options.NotifyNoData != nil {
			opts.NotifyNoData = mon.Options.NotifyNoData
			hasOpts = true
		}
		if mon.Options.TimeoutH.IsSet() && mon.Options.TimeoutH.Get() != nil {
			opts.TimeoutH = mon.Options.TimeoutH.Get()
			hasOpts = true
		}
		if mon.Options.RenotifyInterval.IsSet() && mon.Options.RenotifyInterval.Get() != nil {
			opts.RenotifyInterval = mon.Options.RenotifyInterval.Get()
			hasOpts = true
		}
		if mon.Options.EscalationMessage != nil {
			opts.EscalationMessage = mon.Options.EscalationMessage
			hasOpts = true
		}
		if mon.Options.IncludeTags != nil {
			opts.IncludeTags = mon.Options.IncludeTags
			hasOpts = true
		}

		if hasOpts {
			props.Options = opts
		}
	}

	data, _ := json.Marshal(props)
	return data
}

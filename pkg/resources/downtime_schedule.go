// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/client"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/config"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/prov"
	"github.com/platform-engineering-labs/formae-plugin-datadog/pkg/registry"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const ResourceTypeDowntimeSchedule = "Datadog::Monitoring::DowntimeSchedule"

func init() {
	registry.Register(ResourceTypeDowntimeSchedule, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &DowntimeSchedule{Client: c}
	})
}

type DowntimeSchedule struct {
	Client *client.Client
}

type downtimeProps struct {
	Id                            *string               `json:"id,omitempty"`
	Scope                         string                `json:"scope"`
	MonitorTags                   []string              `json:"monitorTags"`
	Message                       *string               `json:"message,omitempty"`
	MuteFirstRecoveryNotification *bool                 `json:"muteFirstRecoveryNotification,omitempty"`
	DisplayTimezone               *string               `json:"displayTimezone,omitempty"`
	OneTimeSchedule               *oneTimeScheduleProps `json:"oneTimeSchedule,omitempty"`
	RecurringSchedule             *recurringSchedProps  `json:"recurringSchedule,omitempty"`
}

type oneTimeScheduleProps struct {
	Start *string `json:"start,omitempty"`
	End   *string `json:"end,omitempty"`
}

type recurringSchedProps struct {
	Recurrences []recurrenceProps `json:"recurrences"`
	Timezone    *string           `json:"timezone,omitempty"`
}

type recurrenceProps struct {
	Duration string  `json:"duration"`
	Rrule    string  `json:"rrule"`
	Start    *string `json:"start,omitempty"`
}

func (d *DowntimeSchedule) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props downtimeProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	monitorIdent := datadogV2.DowntimeMonitorIdentifierTagsAsDowntimeMonitorIdentifier(
		datadogV2.NewDowntimeMonitorIdentifierTags(props.MonitorTags),
	)

	attrs := datadogV2.DowntimeCreateRequestAttributes{
		MonitorIdentifier: monitorIdent,
		Scope:             props.Scope,
	}
	if props.Message != nil {
		attrs.Message = *datadog.NewNullableString(props.Message)
	}
	if props.MuteFirstRecoveryNotification != nil {
		attrs.MuteFirstRecoveryNotification = props.MuteFirstRecoveryNotification
	}
	if props.DisplayTimezone != nil {
		attrs.DisplayTimezone = *datadog.NewNullableString(props.DisplayTimezone)
	}
	if props.OneTimeSchedule != nil {
		attrs.Schedule = buildOneTimeScheduleCreate(props.OneTimeSchedule)
	} else if props.RecurringSchedule != nil {
		attrs.Schedule = buildRecurringScheduleCreate(props.RecurringSchedule)
		// Datadog requires display_timezone to match the schedule timezone.
		if props.RecurringSchedule.Timezone != nil && props.DisplayTimezone == nil {
			attrs.DisplayTimezone = *datadog.NewNullableString(props.RecurringSchedule.Timezone)
		}
	}

	body := datadogV2.DowntimeCreateRequest{
		Data: datadogV2.DowntimeCreateRequestData{
			Attributes: attrs,
			Type:       datadogV2.DOWNTIMERESOURCETYPE_DOWNTIME,
		},
	}

	api := datadogV2.NewDowntimesApi(d.Client.ApiClient)
	resp, httpResp, err := api.CreateDowntime(d.Client.Ctx, body)
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
	propsJSON := marshalDowntimeProps(&data)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (d *DowntimeSchedule) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV2.NewDowntimesApi(d.Client.ApiClient)
	resp, httpResp, err := api.GetDowntime(d.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	data := resp.GetData()

	// Datadog doesn't hard-delete cancelled downtimes; it marks them as
	// canceled and keeps returning them via GET. Treat as NotFound so sync
	// correctly tombstones resources deleted out-of-band.
	if attrs, ok := data.GetAttributesOk(); ok && attrs != nil {
		if attrs.GetStatus() == datadogV2.DOWNTIMESTATUS_CANCELED {
			return &resource.ReadResult{
				ErrorCode: resource.OperationErrorCodeNotFound,
			}, nil
		}
	}

	propsJSON := marshalDowntimeProps(&data)

	return &resource.ReadResult{
		ResourceType: ResourceTypeDowntimeSchedule,
		Properties:   string(propsJSON),
	}, nil
}

func (d *DowntimeSchedule) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props downtimeProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	monitorIdent := datadogV2.DowntimeMonitorIdentifierTagsAsDowntimeMonitorIdentifier(
		datadogV2.NewDowntimeMonitorIdentifierTags(props.MonitorTags),
	)

	attrs := datadogV2.DowntimeUpdateRequestAttributes{}
	attrs.MonitorIdentifier = &monitorIdent
	attrs.Scope = &props.Scope
	if props.Message != nil {
		attrs.Message = *datadog.NewNullableString(props.Message)
	}
	if props.MuteFirstRecoveryNotification != nil {
		attrs.MuteFirstRecoveryNotification = props.MuteFirstRecoveryNotification
	}
	if props.DisplayTimezone != nil {
		attrs.DisplayTimezone = *datadog.NewNullableString(props.DisplayTimezone)
	}
	if props.OneTimeSchedule != nil {
		attrs.Schedule = buildOneTimeScheduleUpdate(props.OneTimeSchedule)
	} else if props.RecurringSchedule != nil {
		attrs.Schedule = buildRecurringScheduleUpdate(props.RecurringSchedule)
		if props.RecurringSchedule.Timezone != nil && props.DisplayTimezone == nil {
			attrs.DisplayTimezone = *datadog.NewNullableString(props.RecurringSchedule.Timezone)
		}
	}

	body := datadogV2.DowntimeUpdateRequest{
		Data: datadogV2.DowntimeUpdateRequestData{
			Attributes: attrs,
			Id:         request.NativeID,
			Type:       datadogV2.DOWNTIMERESOURCETYPE_DOWNTIME,
		},
	}

	api := datadogV2.NewDowntimesApi(d.Client.ApiClient)
	resp, httpResp, err := api.UpdateDowntime(d.Client.Ctx, request.NativeID, body)
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
	propsJSON := marshalDowntimeProps(&data)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (d *DowntimeSchedule) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV2.NewDowntimesApi(d.Client.ApiClient)
	httpResp, err := api.CancelDowntime(d.Client.Ctx, request.NativeID)
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

func (d *DowntimeSchedule) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	// Downtime operations are synchronous — no async polling needed.
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (d *DowntimeSchedule) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV2.NewDowntimesApi(d.Client.ApiClient)
	var nativeIDs []string
	items, cancel := api.ListDowntimesWithPagination(d.Client.Ctx)
	defer cancel()
	for item := range items {
		if item.Error != nil {
			return nil, fmt.Errorf("failed to list downtimes: %w", item.Error)
		}
		// Skip cancelled downtimes — Datadog retains them in list results
		// indefinitely, but they shouldn't appear in inventory.
		if attrs, ok := item.Item.GetAttributesOk(); ok && attrs != nil {
			if attrs.GetStatus() == datadogV2.DOWNTIMESTATUS_CANCELED {
				continue
			}
		}
		nativeIDs = append(nativeIDs, item.Item.GetId())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func buildOneTimeScheduleCreate(ots *oneTimeScheduleProps) *datadogV2.DowntimeScheduleCreateRequest {
	sched := datadogV2.NewDowntimeScheduleOneTimeCreateUpdateRequest()
	if ots.Start != nil {
		t, err := parseISO8601(*ots.Start)
		if err == nil {
			sched.Start = *datadog.NewNullableTime(&t)
		}
	}
	if ots.End != nil {
		t, err := parseISO8601(*ots.End)
		if err == nil {
			sched.End = *datadog.NewNullableTime(&t)
		}
	}
	result := datadogV2.DowntimeScheduleOneTimeCreateUpdateRequestAsDowntimeScheduleCreateRequest(sched)
	return &result
}

func buildRecurringScheduleCreate(rs *recurringSchedProps) *datadogV2.DowntimeScheduleCreateRequest {
	recurrences := make([]datadogV2.DowntimeScheduleRecurrenceCreateUpdateRequest, 0, len(rs.Recurrences))
	for _, r := range rs.Recurrences {
		rec := datadogV2.DowntimeScheduleRecurrenceCreateUpdateRequest{
			Duration: r.Duration,
			Rrule:    r.Rrule,
		}
		if r.Start != nil {
			rec.Start = *datadog.NewNullableString(r.Start)
		}
		recurrences = append(recurrences, rec)
	}
	sched := datadogV2.NewDowntimeScheduleRecurrencesCreateRequest(recurrences)
	if rs.Timezone != nil {
		sched.Timezone = rs.Timezone
	}
	result := datadogV2.DowntimeScheduleRecurrencesCreateRequestAsDowntimeScheduleCreateRequest(sched)
	return &result
}

func buildOneTimeScheduleUpdate(ots *oneTimeScheduleProps) *datadogV2.DowntimeScheduleUpdateRequest {
	sched := datadogV2.NewDowntimeScheduleOneTimeCreateUpdateRequest()
	if ots.Start != nil {
		t, err := parseISO8601(*ots.Start)
		if err == nil {
			sched.Start = *datadog.NewNullableTime(&t)
		}
	}
	if ots.End != nil {
		t, err := parseISO8601(*ots.End)
		if err == nil {
			sched.End = *datadog.NewNullableTime(&t)
		}
	}
	result := datadogV2.DowntimeScheduleOneTimeCreateUpdateRequestAsDowntimeScheduleUpdateRequest(sched)
	return &result
}

func buildRecurringScheduleUpdate(rs *recurringSchedProps) *datadogV2.DowntimeScheduleUpdateRequest {
	recurrences := make([]datadogV2.DowntimeScheduleRecurrenceCreateUpdateRequest, 0, len(rs.Recurrences))
	for _, r := range rs.Recurrences {
		rec := datadogV2.DowntimeScheduleRecurrenceCreateUpdateRequest{
			Duration: r.Duration,
			Rrule:    r.Rrule,
		}
		if r.Start != nil {
			rec.Start = *datadog.NewNullableString(r.Start)
		}
		recurrences = append(recurrences, rec)
	}
	sched := &datadogV2.DowntimeScheduleRecurrencesUpdateRequest{
		Recurrences: recurrences,
	}
	if rs.Timezone != nil {
		sched.Timezone = rs.Timezone
	}
	result := datadogV2.DowntimeScheduleRecurrencesUpdateRequestAsDowntimeScheduleUpdateRequest(sched)
	return &result
}

// marshalDowntimeProps converts a Datadog DowntimeResponseData to JSON properties.
func marshalDowntimeProps(data *datadogV2.DowntimeResponseData) json.RawMessage {
	props := downtimeProps{}

	id := data.GetId()
	props.Id = &id

	attrs := data.GetAttributes()

	if attrs.Scope != nil {
		props.Scope = *attrs.Scope
	}
	if attrs.MonitorIdentifier != nil {
		if attrs.MonitorIdentifier.DowntimeMonitorIdentifierTags != nil {
			props.MonitorTags = attrs.MonitorIdentifier.DowntimeMonitorIdentifierTags.GetMonitorTags()
		}
	}
	if attrs.Message.IsSet() && attrs.Message.Get() != nil {
		props.Message = attrs.Message.Get()
	}
	if attrs.MuteFirstRecoveryNotification != nil {
		props.MuteFirstRecoveryNotification = attrs.MuteFirstRecoveryNotification
	}
	if attrs.DisplayTimezone.IsSet() && attrs.DisplayTimezone.Get() != nil {
		props.DisplayTimezone = attrs.DisplayTimezone.Get()
	}

	if attrs.Schedule != nil {
		if attrs.Schedule.DowntimeScheduleOneTimeResponse != nil {
			ots := &oneTimeScheduleProps{}
			r := attrs.Schedule.DowntimeScheduleOneTimeResponse
			startStr := r.Start.Format("2006-01-02T15:04:05Z")
			ots.Start = &startStr
			if r.End.IsSet() && r.End.Get() != nil {
				endStr := r.End.Get().Format("2006-01-02T15:04:05Z")
				ots.End = &endStr
			}
			props.OneTimeSchedule = ots
		} else if attrs.Schedule.DowntimeScheduleRecurrencesResponse != nil {
			rs := &recurringSchedProps{}
			r := attrs.Schedule.DowntimeScheduleRecurrencesResponse
			if r.Timezone != nil {
				rs.Timezone = r.Timezone
			}
			for _, rec := range r.Recurrences {
				rp := recurrenceProps{
					Duration: rec.GetDuration(),
					Rrule:    rec.GetRrule(),
				}
				// Note: rec.Start is omitted because Datadog auto-fills it
				// when not provided, and including it would cause drift
				// between desired and actual state.
				rs.Recurrences = append(rs.Recurrences, rp)
			}
			props.RecurringSchedule = rs
		}
	}

	d, _ := json.Marshal(props)
	return d
}

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

const ResourceTypeDashboard = "Datadog::Dashboard::Dashboard"

func init() {
	registry.Register(ResourceTypeDashboard, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &Dashboard{Client: c}
	})
}

type Dashboard struct {
	Client *client.Client
}

type dashboardProps struct {
	Title       string   `json:"title"`
	LayoutType  string   `json:"layoutType"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Widgets     *string  `json:"widgets,omitempty"`
}

func (d *Dashboard) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props dashboardProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	body := datadogV1.Dashboard{
		Title:      props.Title,
		LayoutType: datadogV1.DashboardLayoutType(props.LayoutType),
		Widgets:    buildDashboardWidgets(props.Widgets),
	}
	if props.Description != nil {
		body.SetDescription(*props.Description)
	}
	if len(props.Tags) > 0 {
		body.SetTags(props.Tags)
	}

	api := datadogV1.NewDashboardsApi(d.Client.ApiClient)
	resp, httpResp, err := api.CreateDashboard(d.Client.Ctx, body)
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
	propsJSON := marshalDashboardProps(&resp)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (d *Dashboard) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV1.NewDashboardsApi(d.Client.ApiClient)
	resp, httpResp, err := api.GetDashboard(d.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	propsJSON := marshalDashboardProps(&resp)

	return &resource.ReadResult{
		ResourceType: ResourceTypeDashboard,
		Properties:   string(propsJSON),
	}, nil
}

func (d *Dashboard) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props dashboardProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	body := datadogV1.Dashboard{
		Title:      props.Title,
		LayoutType: datadogV1.DashboardLayoutType(props.LayoutType),
		Widgets:    buildDashboardWidgets(props.Widgets),
	}
	if props.Description != nil {
		body.SetDescription(*props.Description)
	}
	if len(props.Tags) > 0 {
		body.SetTags(props.Tags)
	}

	api := datadogV1.NewDashboardsApi(d.Client.ApiClient)
	resp, httpResp, err := api.UpdateDashboard(d.Client.Ctx, request.NativeID, body)
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

	propsJSON := marshalDashboardProps(&resp)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (d *Dashboard) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV1.NewDashboardsApi(d.Client.ApiClient)
	_, httpResp, err := api.DeleteDashboard(d.Client.Ctx, request.NativeID)
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

func (d *Dashboard) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (d *Dashboard) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV1.NewDashboardsApi(d.Client.ApiClient)
	resp, httpResp, err := api.ListDashboards(d.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list dashboards: %w (status: %d)", err, httpResp.StatusCode)
	}

	dashboards := resp.GetDashboards()
	nativeIDs := make([]string, 0, len(dashboards))
	for _, db := range dashboards {
		nativeIDs = append(nativeIDs, db.GetId())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

// buildDashboardWidgets deserializes raw JSON widgets into SDK Widget structs.
// If widgets is nil or empty, returns an empty widget list (Datadog requires non-null).
func buildDashboardWidgets(widgetsJSON *string) []datadogV1.Widget {
	if widgetsJSON == nil || *widgetsJSON == "" {
		return []datadogV1.Widget{}
	}
	var widgets []datadogV1.Widget
	if err := json.Unmarshal([]byte(*widgetsJSON), &widgets); err != nil {
		return []datadogV1.Widget{}
	}
	return widgets
}

func marshalDashboardProps(db *datadogV1.Dashboard) json.RawMessage {
	props := dashboardProps{
		Title:      db.GetTitle(),
		LayoutType: string(db.GetLayoutType()),
	}

	if desc, ok := db.GetDescriptionOk(); ok && desc != nil {
		props.Description = desc
	}

	if tags, ok := db.GetTagsOk(); ok && tags != nil {
		props.Tags = *tags
	}

	// Serialize widgets back to raw JSON
	widgets := db.GetWidgets()
	if len(widgets) > 0 {
		widgetsJSON, err := json.Marshal(widgets)
		if err == nil {
			s := string(widgetsJSON)
			props.Widgets = &s
		}
	}

	d, _ := json.Marshal(props)
	return d
}

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

const ResourceTypeTeam = "Datadog::IAM::Team"

func init() {
	registry.Register(ResourceTypeTeam, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &Team{Client: c}
	})
}

type Team struct {
	Client *client.Client
}

type teamProps struct {
	Name        string  `json:"name"`
	Handle      string  `json:"handle"`
	Description *string `json:"description,omitempty"`
}

func (t *Team) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props teamProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	attrs := datadogV2.TeamCreateAttributes{
		Name:   props.Name,
		Handle: props.Handle,
	}
	if props.Description != nil {
		attrs.Description = props.Description
	}

	body := datadogV2.TeamCreateRequest{
		Data: datadogV2.TeamCreate{
			Attributes: attrs,
			Type:       datadogV2.TEAMTYPE_TEAM,
		},
	}

	api := datadogV2.NewTeamsApi(t.Client.ApiClient)
	resp, httpResp, err := api.CreateTeam(t.Client.Ctx, body)
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
	propsJSON := marshalTeamProps(&data)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (t *Team) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV2.NewTeamsApi(t.Client.ApiClient)
	resp, httpResp, err := api.GetTeam(t.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	data := resp.GetData()
	propsJSON := marshalTeamProps(&data)

	return &resource.ReadResult{
		ResourceType: ResourceTypeTeam,
		Properties:   string(propsJSON),
	}, nil
}

func (t *Team) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props teamProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	attrs := datadogV2.TeamUpdateAttributes{
		Name:   props.Name,
		Handle: props.Handle,
	}
	if props.Description != nil {
		attrs.Description = props.Description
	}

	body := datadogV2.TeamUpdateRequest{
		Data: datadogV2.TeamUpdate{
			Attributes: attrs,
			Type:       datadogV2.TEAMTYPE_TEAM,
		},
	}

	api := datadogV2.NewTeamsApi(t.Client.ApiClient)
	resp, httpResp, err := api.UpdateTeam(t.Client.Ctx, request.NativeID, body)
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
	propsJSON := marshalTeamProps(&data)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (t *Team) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV2.NewTeamsApi(t.Client.ApiClient)
	httpResp, err := api.DeleteTeam(t.Client.Ctx, request.NativeID)
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

func (t *Team) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (t *Team) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV2.NewTeamsApi(t.Client.ApiClient)
	resp, httpResp, err := api.ListTeams(t.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w (status: %d)", err, httpResp.StatusCode)
	}

	teams := resp.GetData()
	nativeIDs := make([]string, 0, len(teams))
	for _, team := range teams {
		nativeIDs = append(nativeIDs, team.GetId())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func marshalTeamProps(data *datadogV2.Team) json.RawMessage {
	attrs := data.GetAttributes()
	props := teamProps{
		Name:   attrs.GetName(),
		Handle: attrs.GetHandle(),
	}

	desc := attrs.GetDescription()
	if desc != "" {
		props.Description = &desc
	}

	d, _ := json.Marshal(props)
	return d
}

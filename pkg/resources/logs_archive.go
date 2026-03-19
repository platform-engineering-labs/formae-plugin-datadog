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

const ResourceTypeLogsArchive = "Datadog::Logs::Archive"

func init() {
	registry.Register(ResourceTypeLogsArchive, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &LogsArchive{Client: c}
	})
}

type LogsArchive struct {
	Client *client.Client
}

type logsArchiveProps struct {
	Id               *string              `json:"id,omitempty"`
	Name             string               `json:"name"`
	Query            string               `json:"query"`
	IncludeTags      *bool                `json:"includeTags,omitempty"`
	RehydrationTags  []string             `json:"rehydrationTags,omitempty"`
	S3Destination    *s3DestinationProps   `json:"s3Destination,omitempty"`
	GCSDestination   *gcsDestinationProps  `json:"gcsDestination,omitempty"`
	AzureDestination *azureDestProps       `json:"azureDestination,omitempty"`
}

type s3DestinationProps struct {
	Bucket      string            `json:"bucket"`
	Integration s3IntegrationProps `json:"integration"`
	Path        *string           `json:"path,omitempty"`
}

type s3IntegrationProps struct {
	AccountId string `json:"accountId"`
	RoleName  string `json:"roleName"`
}

type gcsDestinationProps struct {
	Bucket      string               `json:"bucket"`
	Integration gcsIntegrationProps   `json:"integration"`
	Path        *string              `json:"path,omitempty"`
}

type gcsIntegrationProps struct {
	ClientEmail string  `json:"clientEmail"`
	ProjectId   *string `json:"projectId,omitempty"`
}

type azureDestProps struct {
	Container      string               `json:"container"`
	StorageAccount string               `json:"storageAccount"`
	Integration    azureIntegrationProps `json:"integration"`
	Path           *string              `json:"path,omitempty"`
}

type azureIntegrationProps struct {
	ClientId string `json:"clientId"`
	TenantId string `json:"tenantId"`
}

func (a *LogsArchive) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props logsArchiveProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	attrs := datadogV2.LogsArchiveCreateRequestAttributes{
		Destination: buildArchiveDestinationCreate(props),
		Name:        props.Name,
		Query:       props.Query,
	}
	if props.IncludeTags != nil {
		attrs.IncludeTags = props.IncludeTags
	}
	if len(props.RehydrationTags) > 0 {
		attrs.RehydrationTags = props.RehydrationTags
	}

	body := datadogV2.LogsArchiveCreateRequest{
		Data: &datadogV2.LogsArchiveCreateRequestDefinition{
			Attributes: &attrs,
			Type:       "archives",
		},
	}

	api := datadogV2.NewLogsArchivesApi(a.Client.ApiClient)
	resp, httpResp, err := api.CreateLogsArchive(a.Client.Ctx, body)
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
	propsJSON := marshalLogsArchiveProps(&data)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (a *LogsArchive) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV2.NewLogsArchivesApi(a.Client.ApiClient)
	resp, httpResp, err := api.GetLogsArchive(a.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	data := resp.GetData()
	propsJSON := marshalLogsArchiveProps(&data)

	return &resource.ReadResult{
		ResourceType: ResourceTypeLogsArchive,
		Properties:   string(propsJSON),
	}, nil
}

func (a *LogsArchive) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props logsArchiveProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	attrs := datadogV2.LogsArchiveCreateRequestAttributes{
		Destination: buildArchiveDestinationCreate(props),
		Name:        props.Name,
		Query:       props.Query,
	}
	if props.IncludeTags != nil {
		attrs.IncludeTags = props.IncludeTags
	}
	if len(props.RehydrationTags) > 0 {
		attrs.RehydrationTags = props.RehydrationTags
	}

	// UpdateLogsArchive uses LogsArchiveCreateRequest as body type.
	body := datadogV2.LogsArchiveCreateRequest{
		Data: &datadogV2.LogsArchiveCreateRequestDefinition{
			Attributes: &attrs,
			Type:       "archives",
		},
	}

	api := datadogV2.NewLogsArchivesApi(a.Client.ApiClient)
	resp, httpResp, err := api.UpdateLogsArchive(a.Client.Ctx, request.NativeID, body)
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
	propsJSON := marshalLogsArchiveProps(&data)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (a *LogsArchive) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV2.NewLogsArchivesApi(a.Client.ApiClient)
	httpResp, err := api.DeleteLogsArchive(a.Client.Ctx, request.NativeID)
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

func (a *LogsArchive) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (a *LogsArchive) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV2.NewLogsArchivesApi(a.Client.ApiClient)
	resp, httpResp, err := api.ListLogsArchives(a.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list logs archives: %w (status: %d)", err, httpResp.StatusCode)
	}

	data := resp.GetData()
	nativeIDs := make([]string, 0, len(data))
	for _, archive := range data {
		nativeIDs = append(nativeIDs, archive.GetId())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func buildArchiveDestinationCreate(props logsArchiveProps) datadogV2.LogsArchiveCreateRequestDestination {
	if props.S3Destination != nil {
		dest := datadogV2.LogsArchiveDestinationS3{
			Bucket: props.S3Destination.Bucket,
			Integration: datadogV2.LogsArchiveIntegrationS3{
				AccountId: props.S3Destination.Integration.AccountId,
				RoleName:  props.S3Destination.Integration.RoleName,
			},
			Type: datadogV2.LOGSARCHIVEDESTINATIONS3TYPE_S3,
		}
		if props.S3Destination.Path != nil {
			dest.Path = props.S3Destination.Path
		}
		return datadogV2.LogsArchiveDestinationS3AsLogsArchiveCreateRequestDestination(&dest)
	}
	if props.GCSDestination != nil {
		dest := datadogV2.LogsArchiveDestinationGCS{
			Bucket: props.GCSDestination.Bucket,
			Integration: datadogV2.LogsArchiveIntegrationGCS{
				ClientEmail: props.GCSDestination.Integration.ClientEmail,
			},
			Type: datadogV2.LOGSARCHIVEDESTINATIONGCSTYPE_GCS,
		}
		if props.GCSDestination.Integration.ProjectId != nil {
			dest.Integration.ProjectId = props.GCSDestination.Integration.ProjectId
		}
		if props.GCSDestination.Path != nil {
			dest.Path = props.GCSDestination.Path
		}
		return datadogV2.LogsArchiveDestinationGCSAsLogsArchiveCreateRequestDestination(&dest)
	}
	if props.AzureDestination != nil {
		dest := datadogV2.LogsArchiveDestinationAzure{
			Container:      props.AzureDestination.Container,
			StorageAccount: props.AzureDestination.StorageAccount,
			Integration: datadogV2.LogsArchiveIntegrationAzure{
				ClientId: props.AzureDestination.Integration.ClientId,
				TenantId: props.AzureDestination.Integration.TenantId,
			},
			Type: datadogV2.LOGSARCHIVEDESTINATIONAZURETYPE_AZURE,
		}
		if props.AzureDestination.Path != nil {
			dest.Path = props.AzureDestination.Path
		}
		return datadogV2.LogsArchiveDestinationAzureAsLogsArchiveCreateRequestDestination(&dest)
	}
	// Fallback: shouldn't happen with valid input.
	return datadogV2.LogsArchiveCreateRequestDestination{}
}

func marshalLogsArchiveProps(data *datadogV2.LogsArchiveDefinition) json.RawMessage {
	props := logsArchiveProps{}

	id := data.GetId()
	props.Id = &id

	attrs := data.GetAttributes()
	props.Name = attrs.GetName()
	props.Query = attrs.GetQuery()

	if attrs.IncludeTags != nil {
		props.IncludeTags = attrs.IncludeTags
	}
	if len(attrs.RehydrationTags) > 0 {
		props.RehydrationTags = attrs.RehydrationTags
	}

	dest := attrs.GetDestination()
	{
		if dest.LogsArchiveDestinationS3 != nil {
			s3 := dest.LogsArchiveDestinationS3
			props.S3Destination = &s3DestinationProps{
				Bucket: s3.GetBucket(),
				Integration: s3IntegrationProps{
					AccountId: s3.Integration.GetAccountId(),
					RoleName:  s3.Integration.GetRoleName(),
				},
			}
			if s3.Path != nil {
				props.S3Destination.Path = s3.Path
			}
		} else if dest.LogsArchiveDestinationGCS != nil {
			gcs := dest.LogsArchiveDestinationGCS
			props.GCSDestination = &gcsDestinationProps{
				Bucket: gcs.GetBucket(),
				Integration: gcsIntegrationProps{
					ClientEmail: gcs.Integration.GetClientEmail(),
				},
			}
			if gcs.Integration.ProjectId != nil {
				props.GCSDestination.Integration.ProjectId = gcs.Integration.ProjectId
			}
			if gcs.Path != nil {
				props.GCSDestination.Path = gcs.Path
			}
		} else if dest.LogsArchiveDestinationAzure != nil {
			az := dest.LogsArchiveDestinationAzure
			props.AzureDestination = &azureDestProps{
				Container:      az.GetContainer(),
				StorageAccount: az.GetStorageAccount(),
				Integration: azureIntegrationProps{
					ClientId: az.Integration.GetClientId(),
					TenantId: az.Integration.GetTenantId(),
				},
			}
			if az.Path != nil {
				props.AzureDestination.Path = az.Path
			}
		}
	}

	d, _ := json.Marshal(props)
	return d
}

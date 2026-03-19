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

const ResourceTypeRole = "Datadog::IAM::Role"

func init() {
	registry.Register(ResourceTypeRole, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &Role{Client: c}
	})
}

type Role struct {
	Client *client.Client
}

type roleProps struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions,omitempty"`
}

func (r *Role) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props roleProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	body := datadogV2.RoleCreateRequest{
		Data: datadogV2.RoleCreateData{
			Attributes: datadogV2.RoleCreateAttributes{
				Name: props.Name,
			},
			Type: datadogV2.ROLESTYPE_ROLES.Ptr(),
		},
	}

	api := datadogV2.NewRolesApi(r.Client.ApiClient)
	resp, httpResp, err := api.CreateRole(r.Client.Ctx, body)
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
	name := ""
	if a := data.GetAttributes(); a.Name != "" {
		name = a.Name
	}

	// Grant permissions if specified
	if len(props.Permissions) > 0 {
		if err := r.syncPermissions(nativeID, nil, props.Permissions); err != nil {
			return nil, fmt.Errorf("role created but failed to grant permissions: %w", err)
		}
	}

	propsJSON := r.marshalRolePropsByName(nativeID, name)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (r *Role) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV2.NewRolesApi(r.Client.ApiClient)
	resp, httpResp, err := api.GetRole(r.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	data := resp.GetData()
	attrs := data.GetAttributes()
	propsJSON := r.marshalRolePropsByName(request.NativeID, attrs.GetName())

	return &resource.ReadResult{
		ResourceType: ResourceTypeRole,
		Properties:   string(propsJSON),
	}, nil
}

func (r *Role) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props roleProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	body := datadogV2.RoleUpdateRequest{
		Data: datadogV2.RoleUpdateData{
			Attributes: datadogV2.RoleUpdateAttributes{
				Name: &props.Name,
			},
			Type: datadogV2.ROLESTYPE_ROLES,
		},
	}

	api := datadogV2.NewRolesApi(r.Client.ApiClient)
	resp, httpResp, err := api.UpdateRole(r.Client.Ctx, request.NativeID, body)
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

	// Sync permissions
	currentPerms, err := r.listPermissionIDs(request.NativeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list current permissions: %w", err)
	}
	if err := r.syncPermissions(request.NativeID, currentPerms, props.Permissions); err != nil {
		return nil, fmt.Errorf("failed to sync permissions: %w", err)
	}

	data := resp.GetData()
	updatedName := ""
	if a := data.GetAttributes(); a.Name != nil {
		updatedName = *a.Name
	}
	propsJSON := r.marshalRolePropsByName(request.NativeID, updatedName)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (r *Role) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV2.NewRolesApi(r.Client.ApiClient)
	httpResp, err := api.DeleteRole(r.Client.Ctx, request.NativeID)
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

func (r *Role) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (r *Role) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV2.NewRolesApi(r.Client.ApiClient)
	resp, httpResp, err := api.ListRoles(r.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w (status: %d)", err, httpResp.StatusCode)
	}

	roles := resp.GetData()
	nativeIDs := make([]string, 0, len(roles))
	for _, role := range roles {
		nativeIDs = append(nativeIDs, role.GetId())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

// listPermissionIDs returns the permission IDs currently assigned to the role.
func (r *Role) listPermissionIDs(roleID string) ([]string, error) {
	api := datadogV2.NewRolesApi(r.Client.ApiClient)
	resp, _, err := api.ListRolePermissions(r.Client.Ctx, roleID)
	if err != nil {
		return nil, err
	}

	perms := resp.GetData()
	ids := make([]string, 0, len(perms))
	for _, p := range perms {
		ids = append(ids, p.GetId())
	}
	return ids, nil
}

// syncPermissions diffs current vs desired permissions and grants/revokes as needed.
func (r *Role) syncPermissions(roleID string, current, desired []string) error {
	currentSet := make(map[string]bool, len(current))
	for _, id := range current {
		currentSet[id] = true
	}
	desiredSet := make(map[string]bool, len(desired))
	for _, id := range desired {
		desiredSet[id] = true
	}

	api := datadogV2.NewRolesApi(r.Client.ApiClient)

	// Grant new permissions
	for _, id := range desired {
		if !currentSet[id] {
			body := datadogV2.RelationshipToPermission{
				Data: &datadogV2.RelationshipToPermissionData{
					Id:   &id,
					Type: datadogV2.PERMISSIONSTYPE_PERMISSIONS.Ptr(),
				},
			}
			if _, _, err := api.AddPermissionToRole(r.Client.Ctx, roleID, body); err != nil {
				return fmt.Errorf("failed to grant permission %s: %w", id, err)
			}
		}
	}

	// Revoke removed permissions
	for _, id := range current {
		if !desiredSet[id] {
			body := datadogV2.RelationshipToPermission{
				Data: &datadogV2.RelationshipToPermissionData{
					Id:   &id,
					Type: datadogV2.PERMISSIONSTYPE_PERMISSIONS.Ptr(),
				},
			}
			if _, _, err := api.RemovePermissionFromRole(r.Client.Ctx, roleID, body); err != nil {
				return fmt.Errorf("failed to revoke permission %s: %w", id, err)
			}
		}
	}

	return nil
}

func (r *Role) marshalRolePropsByName(roleID, name string) json.RawMessage {
	props := roleProps{
		Name: name,
	}

	// Fetch permissions
	permIDs, err := r.listPermissionIDs(roleID)
	if err == nil && len(permIDs) > 0 {
		props.Permissions = permIDs
	}

	d, _ := json.Marshal(props)
	return d
}

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

const ResourceTypeSyntheticsTest = "Datadog::Synthetics::Test"

func init() {
	registry.Register(ResourceTypeSyntheticsTest, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &SyntheticsTest{Client: c}
	})
}

type SyntheticsTest struct {
	Client *client.Client
}

type syntheticsTestProps struct {
	Name      string   `json:"name"`
	TestType  string   `json:"testType"`
	Message   string   `json:"message"`
	Status    *string  `json:"status,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Locations []string `json:"locations,omitempty"`
	Config    *string  `json:"config,omitempty"`
	Options   *string  `json:"options,omitempty"`
}

func (s *SyntheticsTest) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props syntheticsTestProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	// Build API test (most common type for IaC)
	body := datadogV1.SyntheticsAPITest{
		Name:      props.Name,
		Message:   props.Message,
		Type:      datadogV1.SYNTHETICSAPITESTTYPE_API,
		Locations: props.Locations,
		Config:    buildSyntheticsConfig(props.Config),
		Options:   buildSyntheticsOptions(props.Options),
	}
	if len(props.Tags) > 0 {
		body.Tags = props.Tags
	}
	if props.Status != nil {
		status := datadogV1.SyntheticsTestPauseStatus(*props.Status)
		body.Status = &status
	}

	api := datadogV1.NewSyntheticsApi(s.Client.ApiClient)
	resp, httpResp, err := api.CreateSyntheticsAPITest(s.Client.Ctx, body)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       mapHTTPError(httpResp, err),
			},
		}, nil
	}

	nativeID := resp.GetPublicId()
	propsJSON := marshalSyntheticsAPITestProps(&resp)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (s *SyntheticsTest) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV1.NewSyntheticsApi(s.Client.ApiClient)
	resp, httpResp, err := api.GetAPITest(s.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	propsJSON := marshalSyntheticsAPITestProps(&resp)

	return &resource.ReadResult{
		ResourceType: ResourceTypeSyntheticsTest,
		Properties:   string(propsJSON),
	}, nil
}

func (s *SyntheticsTest) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props syntheticsTestProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	body := datadogV1.SyntheticsAPITest{
		Name:      props.Name,
		Message:   props.Message,
		Type:      datadogV1.SYNTHETICSAPITESTTYPE_API,
		Locations: props.Locations,
		Config:    buildSyntheticsConfig(props.Config),
		Options:   buildSyntheticsOptions(props.Options),
	}
	if len(props.Tags) > 0 {
		body.Tags = props.Tags
	}
	if props.Status != nil {
		status := datadogV1.SyntheticsTestPauseStatus(*props.Status)
		body.Status = &status
	}

	api := datadogV1.NewSyntheticsApi(s.Client.ApiClient)
	resp, httpResp, err := api.UpdateAPITest(s.Client.Ctx, request.NativeID, body)
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

	propsJSON := marshalSyntheticsAPITestProps(&resp)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (s *SyntheticsTest) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV1.NewSyntheticsApi(s.Client.ApiClient)
	body := datadogV1.SyntheticsDeleteTestsPayload{
		PublicIds: []string{request.NativeID},
	}
	_, httpResp, err := api.DeleteTests(s.Client.Ctx, body)
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

func (s *SyntheticsTest) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (s *SyntheticsTest) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV1.NewSyntheticsApi(s.Client.ApiClient)
	resp, httpResp, err := api.ListTests(s.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list synthetics tests: %w (status: %d)", err, httpResp.StatusCode)
	}

	tests := resp.GetTests()
	nativeIDs := make([]string, 0, len(tests))
	for _, test := range tests {
		nativeIDs = append(nativeIDs, test.GetPublicId())
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func buildSyntheticsConfig(configJSON *string) datadogV1.SyntheticsAPITestConfig {
	if configJSON == nil || *configJSON == "" {
		return datadogV1.SyntheticsAPITestConfig{}
	}
	var cfg datadogV1.SyntheticsAPITestConfig
	if err := json.Unmarshal([]byte(*configJSON), &cfg); err != nil {
		return datadogV1.SyntheticsAPITestConfig{}
	}
	return cfg
}

func buildSyntheticsOptions(optionsJSON *string) datadogV1.SyntheticsTestOptions {
	if optionsJSON == nil || *optionsJSON == "" {
		return datadogV1.SyntheticsTestOptions{}
	}
	var opts datadogV1.SyntheticsTestOptions
	if err := json.Unmarshal([]byte(*optionsJSON), &opts); err != nil {
		return datadogV1.SyntheticsTestOptions{}
	}
	return opts
}

func marshalSyntheticsAPITestProps(test *datadogV1.SyntheticsAPITest) json.RawMessage {
	props := syntheticsTestProps{
		Name:     test.GetName(),
		TestType: string(test.GetType()),
		Message:  test.GetMessage(),
	}

	if test.Status != nil {
		s := string(*test.Status)
		props.Status = &s
	}
	if len(test.Tags) > 0 {
		props.Tags = sortedTags(test.Tags)
	}
	if len(test.Locations) > 0 {
		props.Locations = test.Locations
	}

	// Serialize config and options as raw JSON
	cfgJSON, err := json.Marshal(test.Config)
	if err == nil {
		s := string(cfgJSON)
		props.Config = &s
	}
	optsJSON, err := json.Marshal(test.Options)
	if err == nil {
		s := string(optsJSON)
		props.Options = &s
	}

	d, _ := json.Marshal(props)
	return d
}

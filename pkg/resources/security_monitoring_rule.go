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

const ResourceTypeSecurityMonitoringRule = "Datadog::Security::MonitoringRule"

func init() {
	registry.Register(ResourceTypeSecurityMonitoringRule, func(c *client.Client, cfg *config.Config) prov.Provisioner {
		return &SecurityMonitoringRule{Client: c}
	})
}

type SecurityMonitoringRule struct {
	Client *client.Client
}

type secMonRuleProps struct {
	Name      string            `json:"name"`
	Message   string            `json:"message"`
	Cases     []secMonRuleCase  `json:"cases"`
	Queries   []secMonRuleQuery `json:"queries"`
	Options   *secMonRuleOpts   `json:"options,omitempty"`
	IsEnabled *bool             `json:"isEnabled,omitempty"`
	RuleType  *string           `json:"ruleType,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Filters   []secMonFilter    `json:"filters,omitempty"`
}

type secMonRuleCase struct {
	Status        string   `json:"status"`
	Name          *string  `json:"name,omitempty"`
	Condition     *string  `json:"condition,omitempty"`
	Notifications []string `json:"notifications,omitempty"`
}

type secMonRuleQuery struct {
	Query          string   `json:"query"`
	Name           *string  `json:"name,omitempty"`
	Aggregation    *string  `json:"aggregation,omitempty"`
	GroupByFields  []string `json:"groupByFields,omitempty"`
	DistinctFields []string `json:"distinctFields,omitempty"`
}

type secMonRuleOpts struct {
	DetectionMethod             *string `json:"detectionMethod,omitempty"`
	EvaluationWindow            *int32  `json:"evaluationWindow,omitempty"`
	KeepAlive                   *int32  `json:"keepAlive,omitempty"`
	MaxSignalDuration           *int32  `json:"maxSignalDuration,omitempty"`
	DecreaseCriticalityBasedOnEnv *bool `json:"decreaseCriticalityBasedOnEnv,omitempty"`
}

type secMonFilter struct {
	Query  string `json:"query"`
	Action string `json:"action"`
}

func (s *SecurityMonitoringRule) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props secMonRuleProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	payload := buildSecMonCreatePayload(props)

	api := datadogV2.NewSecurityMonitoringApi(s.Client.ApiClient)
	resp, httpResp, err := api.CreateSecurityMonitoringRule(s.Client.Ctx, payload)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       mapHTTPError(httpResp, err),
			},
		}, nil
	}

	nativeID, propsJSON := marshalSecMonRuleResponse(resp)

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (s *SecurityMonitoringRule) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	api := datadogV2.NewSecurityMonitoringApi(s.Client.ApiClient)
	resp, httpResp, err := api.GetSecurityMonitoringRule(s.Client.Ctx, request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: mapHTTPError(httpResp, err),
		}, nil
	}

	_, propsJSON := marshalSecMonRuleResponse(resp)

	return &resource.ReadResult{
		ResourceType: ResourceTypeSecurityMonitoringRule,
		Properties:   string(propsJSON),
	}, nil
}

func (s *SecurityMonitoringRule) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	var props secMonRuleProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse desired properties: %w", err)
	}

	payload := buildSecMonUpdatePayload(props)

	api := datadogV2.NewSecurityMonitoringApi(s.Client.ApiClient)
	resp, httpResp, err := api.UpdateSecurityMonitoringRule(s.Client.Ctx, request.NativeID, payload)
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

	_, propsJSON := marshalSecMonRuleResponse(resp)

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

func (s *SecurityMonitoringRule) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	api := datadogV2.NewSecurityMonitoringApi(s.Client.ApiClient)
	httpResp, err := api.DeleteSecurityMonitoringRule(s.Client.Ctx, request.NativeID)
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

func (s *SecurityMonitoringRule) Status(_ context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

func (s *SecurityMonitoringRule) List(ctx context.Context, _ *resource.ListRequest) (*resource.ListResult, error) {
	api := datadogV2.NewSecurityMonitoringApi(s.Client.ApiClient)
	resp, httpResp, err := api.ListSecurityMonitoringRules(s.Client.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list security monitoring rules: %w (status: %d)", err, httpResp.StatusCode)
	}

	rules := resp.GetData()
	nativeIDs := make([]string, 0, len(rules))
	for _, rule := range rules {
		if std, ok := rule.GetActualInstance().(*datadogV2.SecurityMonitoringStandardRuleResponse); ok {
			nativeIDs = append(nativeIDs, std.GetId())
		} else if sig, ok := rule.GetActualInstance().(*datadogV2.SecurityMonitoringSignalRuleResponse); ok {
			nativeIDs = append(nativeIDs, sig.GetId())
		}
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func buildSecMonCreatePayload(props secMonRuleProps) datadogV2.SecurityMonitoringRuleCreatePayload {
	cases := make([]datadogV2.SecurityMonitoringRuleCaseCreate, 0, len(props.Cases))
	for _, c := range props.Cases {
		rc := datadogV2.SecurityMonitoringRuleCaseCreate{
			Status: datadogV2.SecurityMonitoringRuleSeverity(c.Status),
		}
		if c.Name != nil {
			rc.Name = c.Name
		}
		if c.Condition != nil {
			rc.Condition = c.Condition
		}
		if len(c.Notifications) > 0 {
			rc.Notifications = c.Notifications
		}
		cases = append(cases, rc)
	}

	queries := make([]datadogV2.SecurityMonitoringStandardRuleQuery, 0, len(props.Queries))
	for _, q := range props.Queries {
		rq := datadogV2.SecurityMonitoringStandardRuleQuery{
			Query: &q.Query,
		}
		if q.Name != nil {
			rq.Name = q.Name
		}
		if q.Aggregation != nil {
			agg := datadogV2.SecurityMonitoringRuleQueryAggregation(*q.Aggregation)
			rq.Aggregation = &agg
		}
		if len(q.GroupByFields) > 0 {
			rq.GroupByFields = q.GroupByFields
		}
		if len(q.DistinctFields) > 0 {
			rq.DistinctFields = q.DistinctFields
		}
		queries = append(queries, rq)
	}

	isEnabled := true
	if props.IsEnabled != nil {
		isEnabled = *props.IsEnabled
	}

	rule := datadogV2.SecurityMonitoringStandardRuleCreatePayload{
		Name:      props.Name,
		Message:   props.Message,
		Cases:     cases,
		Queries:   queries,
		IsEnabled: isEnabled,
		Options:   buildSecMonOptions(props.Options),
	}

	if props.RuleType != nil {
		rt := datadogV2.SecurityMonitoringRuleTypeCreate(*props.RuleType)
		rule.Type = &rt
	}
	if len(props.Tags) > 0 {
		rule.Tags = props.Tags
	}
	if len(props.Filters) > 0 {
		rule.Filters = buildSecMonFilters(props.Filters)
	}

	payload := datadogV2.SecurityMonitoringRuleCreatePayload{}
	payload.SecurityMonitoringStandardRuleCreatePayload = &rule
	return payload
}

func buildSecMonUpdatePayload(props secMonRuleProps) datadogV2.SecurityMonitoringRuleUpdatePayload {
	cases := make([]datadogV2.SecurityMonitoringRuleCase, 0, len(props.Cases))
	for _, c := range props.Cases {
		rc := datadogV2.SecurityMonitoringRuleCase{
			Status: (*datadogV2.SecurityMonitoringRuleSeverity)(&c.Status),
		}
		if c.Name != nil {
			rc.Name = c.Name
		}
		if c.Condition != nil {
			rc.Condition = c.Condition
		}
		if len(c.Notifications) > 0 {
			rc.Notifications = c.Notifications
		}
		cases = append(cases, rc)
	}

	queries := make([]datadogV2.SecurityMonitoringRuleQuery, 0, len(props.Queries))
	for _, q := range props.Queries {
		stdQuery := datadogV2.SecurityMonitoringStandardRuleQuery{
			Query: &q.Query,
		}
		if q.Name != nil {
			stdQuery.Name = q.Name
		}
		if q.Aggregation != nil {
			agg := datadogV2.SecurityMonitoringRuleQueryAggregation(*q.Aggregation)
			stdQuery.Aggregation = &agg
		}
		if len(q.GroupByFields) > 0 {
			stdQuery.GroupByFields = q.GroupByFields
		}
		if len(q.DistinctFields) > 0 {
			stdQuery.DistinctFields = q.DistinctFields
		}
		rq := datadogV2.SecurityMonitoringRuleQuery{}
		rq.SecurityMonitoringStandardRuleQuery = &stdQuery
		queries = append(queries, rq)
	}

	payload := datadogV2.SecurityMonitoringRuleUpdatePayload{
		Name:    &props.Name,
		Message: &props.Message,
		Cases:   cases,
		Queries: queries,
	}

	if props.IsEnabled != nil {
		payload.IsEnabled = props.IsEnabled
	}
	if props.Options != nil {
		opts := buildSecMonOptions(props.Options)
		payload.Options = &opts
	}
	if len(props.Tags) > 0 {
		payload.Tags = props.Tags
	}
	if len(props.Filters) > 0 {
		payload.Filters = buildSecMonFilters(props.Filters)
	}

	return payload
}

func buildSecMonOptions(opts *secMonRuleOpts) datadogV2.SecurityMonitoringRuleOptions {
	o := datadogV2.SecurityMonitoringRuleOptions{}
	if opts == nil {
		return o
	}
	if opts.DetectionMethod != nil {
		dm := datadogV2.SecurityMonitoringRuleDetectionMethod(*opts.DetectionMethod)
		o.DetectionMethod = &dm
	}
	if opts.EvaluationWindow != nil {
		ew := datadogV2.SecurityMonitoringRuleEvaluationWindow(*opts.EvaluationWindow)
		o.EvaluationWindow = &ew
	}
	if opts.KeepAlive != nil {
		ka := datadogV2.SecurityMonitoringRuleKeepAlive(*opts.KeepAlive)
		o.KeepAlive = &ka
	}
	if opts.MaxSignalDuration != nil {
		msd := datadogV2.SecurityMonitoringRuleMaxSignalDuration(*opts.MaxSignalDuration)
		o.MaxSignalDuration = &msd
	}
	if opts.DecreaseCriticalityBasedOnEnv != nil {
		o.DecreaseCriticalityBasedOnEnv = opts.DecreaseCriticalityBasedOnEnv
	}
	return o
}

func buildSecMonFilters(filters []secMonFilter) []datadogV2.SecurityMonitoringFilter {
	result := make([]datadogV2.SecurityMonitoringFilter, 0, len(filters))
	for _, f := range filters {
		action := datadogV2.SecurityMonitoringFilterAction(f.Action)
		result = append(result, datadogV2.SecurityMonitoringFilter{
			Query:  &f.Query,
			Action: &action,
		})
	}
	return result
}

func marshalSecMonRuleResponse(resp datadogV2.SecurityMonitoringRuleResponse) (string, json.RawMessage) {
	if std, ok := resp.GetActualInstance().(*datadogV2.SecurityMonitoringStandardRuleResponse); ok {
		return marshalStandardRuleResponse(std)
	}
	if sig, ok := resp.GetActualInstance().(*datadogV2.SecurityMonitoringSignalRuleResponse); ok {
		return marshalSignalRuleResponse(sig)
	}
	// Fallback
	d, _ := json.Marshal(secMonRuleProps{})
	return "", d
}

func marshalStandardRuleResponse(std *datadogV2.SecurityMonitoringStandardRuleResponse) (string, json.RawMessage) {
	props := secMonRuleProps{
		Name:      std.GetName(),
		Message:   std.GetMessage(),
		IsEnabled: std.IsEnabled,
	}

	if std.Type != nil {
		rt := string(*std.Type)
		props.RuleType = &rt
	}

	for _, c := range std.GetCases() {
		rc := secMonRuleCase{}
		if c.Status != nil {
			rc.Status = string(*c.Status)
		}
		rc.Name = c.Name
		rc.Condition = c.Condition
		if len(c.Notifications) > 0 {
			rc.Notifications = c.Notifications
		}
		props.Cases = append(props.Cases, rc)
	}

	for _, q := range std.GetQueries() {
		rq := secMonRuleQuery{}
		if q.Query != nil {
			rq.Query = *q.Query
		}
		rq.Name = q.Name
		if q.Aggregation != nil {
			agg := string(*q.Aggregation)
			rq.Aggregation = &agg
		}
		if len(q.GroupByFields) > 0 {
			rq.GroupByFields = q.GroupByFields
		}
		if len(q.DistinctFields) > 0 {
			rq.DistinctFields = q.DistinctFields
		}
		props.Queries = append(props.Queries, rq)
	}

	if std.Options != nil {
		opts := &secMonRuleOpts{}
		if std.Options.DetectionMethod != nil {
			dm := string(*std.Options.DetectionMethod)
			opts.DetectionMethod = &dm
		}
		if std.Options.EvaluationWindow != nil {
			ew := int32(*std.Options.EvaluationWindow)
			opts.EvaluationWindow = &ew
		}
		if std.Options.KeepAlive != nil {
			ka := int32(*std.Options.KeepAlive)
			opts.KeepAlive = &ka
		}
		if std.Options.MaxSignalDuration != nil {
			msd := int32(*std.Options.MaxSignalDuration)
			opts.MaxSignalDuration = &msd
		}
		opts.DecreaseCriticalityBasedOnEnv = std.Options.DecreaseCriticalityBasedOnEnv
		props.Options = opts
	}

	if len(std.Tags) > 0 {
		props.Tags = sortedTags(std.Tags)
	}

	for _, f := range std.GetFilters() {
		sf := secMonFilter{}
		if f.Query != nil {
			sf.Query = *f.Query
		}
		if f.Action != nil {
			sf.Action = string(*f.Action)
		}
		props.Filters = append(props.Filters, sf)
	}

	d, _ := json.Marshal(props)
	return std.GetId(), d
}

func marshalSignalRuleResponse(sig *datadogV2.SecurityMonitoringSignalRuleResponse) (string, json.RawMessage) {
	props := secMonRuleProps{
		Name:      sig.GetName(),
		Message:   sig.GetMessage(),
		IsEnabled: sig.IsEnabled,
	}

	for _, c := range sig.GetCases() {
		rc := secMonRuleCase{}
		if c.Status != nil {
			rc.Status = string(*c.Status)
		}
		rc.Name = c.Name
		rc.Condition = c.Condition
		props.Cases = append(props.Cases, rc)
	}

	if len(sig.Tags) > 0 {
		props.Tags = sortedTags(sig.Tags)
	}

	d, _ := json.Marshal(props)
	return sig.GetId(), d
}

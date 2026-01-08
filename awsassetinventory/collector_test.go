package awsassetinventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
)

type mockConfigClient struct {
	listDiscoveredResourcesFunc     func(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error)
	batchGetResourceConfigFunc      func(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error)
	getDiscoveredResourceCountsFunc func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error)
}

func (m *mockConfigClient) ListDiscoveredResources(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error) {
	if m.listDiscoveredResourcesFunc != nil {
		return m.listDiscoveredResourcesFunc(ctx, params, optFns...)
	}
	return &configservice.ListDiscoveredResourcesOutput{}, nil
}

func (m *mockConfigClient) BatchGetResourceConfig(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error) {
	if m.batchGetResourceConfigFunc != nil {
		return m.batchGetResourceConfigFunc(ctx, params, optFns...)
	}
	return &configservice.BatchGetResourceConfigOutput{}, nil
}

func (m *mockConfigClient) GetDiscoveredResourceCounts(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
	if m.getDiscoveredResourceCountsFunc != nil {
		return m.getDiscoveredResourceCountsFunc(ctx, params, optFns...)
	}
	return &configservice.GetDiscoveredResourceCountsOutput{}, nil
}

func TestNewCollector(t *testing.T) {
	factory := func(r Region) ConfigClient {
		return &mockConfigClient{}
	}
	c := NewCollector("test-profile", factory)

	if c.profile != "test-profile" {
		t.Errorf("NewCollector().profile = %v, want %v", c.profile, "test-profile")
	}
	if c.clientFactory == nil {
		t.Error("NewCollector().clientFactory should not be nil")
	}
}

func TestCollector_Collect_EmptyRegions(t *testing.T) {
	factory := func(r Region) ConfigClient {
		return &mockConfigClient{}
	}
	c := NewCollector("test", factory)

	inv, err := c.Collect(context.Background(), []Region{})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(inv.Resources) != 0 {
		t.Errorf("Collect() resources = %v, want empty", len(inv.Resources))
	}
}

func TestCollector_Collect_SingleRegion(t *testing.T) {
	mock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return &configservice.GetDiscoveredResourceCountsOutput{
				ResourceCounts: []types.ResourceCount{
					{ResourceType: "AWS::EC2::Instance", Count: 2},
				},
			}, nil
		},
		listDiscoveredResourcesFunc: func(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error) {
			return &configservice.ListDiscoveredResourcesOutput{
				ResourceIdentifiers: []types.ResourceIdentifier{
					{ResourceId: aws.String("i-12345"), ResourceName: aws.String("instance-1")},
					{ResourceId: aws.String("i-67890"), ResourceName: aws.String("instance-2")},
				},
			}, nil
		},
		batchGetResourceConfigFunc: func(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error) {
			return &configservice.BatchGetResourceConfigOutput{
				BaseConfigurationItems: []types.BaseConfigurationItem{
					{
						ResourceType: "AWS::EC2::Instance",
						ResourceId:   aws.String("i-12345"),
						ResourceName: aws.String("instance-1"),
						AccountId:    aws.String("123456789012"),
						Arn:          aws.String("arn:aws:ec2:us-east-1:123456789012:instance/i-12345"),
					},
					{
						ResourceType: "AWS::EC2::Instance",
						ResourceId:   aws.String("i-67890"),
						ResourceName: aws.String("instance-2"),
						AccountId:    aws.String("123456789012"),
						Arn:          aws.String("arn:aws:ec2:us-east-1:123456789012:instance/i-67890"),
					},
				},
			}, nil
		},
	}

	factory := func(r Region) ConfigClient { return mock }
	c := NewCollector("test", factory)

	inv, err := c.Collect(context.Background(), []Region{"us-east-1"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(inv.Resources) != 2 {
		t.Errorf("Collect() resources = %v, want 2", len(inv.Resources))
	}
	if inv.Resources[0].AccountID != "123456789012" {
		t.Errorf("Collect() resource AccountID = %v, want 123456789012", inv.Resources[0].AccountID)
	}
}

func TestCollector_Collect_MultipleRegions(t *testing.T) {
	mock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return &configservice.GetDiscoveredResourceCountsOutput{
				ResourceCounts: []types.ResourceCount{
					{ResourceType: "AWS::S3::Bucket", Count: 1},
				},
			}, nil
		},
		listDiscoveredResourcesFunc: func(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error) {
			return &configservice.ListDiscoveredResourcesOutput{
				ResourceIdentifiers: []types.ResourceIdentifier{
					{ResourceId: aws.String("bucket-1"), ResourceName: aws.String("my-bucket")},
				},
			}, nil
		},
		batchGetResourceConfigFunc: func(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error) {
			return &configservice.BatchGetResourceConfigOutput{
				BaseConfigurationItems: []types.BaseConfigurationItem{
					{
						ResourceType: "AWS::S3::Bucket",
						ResourceId:   aws.String("bucket-1"),
						ResourceName: aws.String("my-bucket"),
						AccountId:    aws.String("123456789012"),
					},
				},
			}, nil
		},
	}

	factory := func(r Region) ConfigClient { return mock }
	c := NewCollector("test", factory)

	regions := []Region{"us-east-1", "us-west-2"}
	inv, err := c.Collect(context.Background(), regions)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(inv.Regions) != 2 {
		t.Errorf("Collect() regions = %v, want 2", len(inv.Regions))
	}
	if len(inv.Resources) != 2 {
		t.Errorf("Collect() resources = %v, want 2 (one per region)", len(inv.Resources))
	}
}

func TestCollector_Collect_ErrorHandling(t *testing.T) {
	expectedErr := errors.New("API error")
	mock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return nil, expectedErr
		},
	}

	factory := func(r Region) ConfigClient { return mock }
	c := NewCollector("test", factory)

	inv, err := c.Collect(context.Background(), []Region{"us-east-1"})
	if err == nil {
		t.Fatal("Collect() expected error, got nil")
	}
	if inv == nil {
		t.Fatal("Collect() should return partial inventory even on error")
	}

	var collectErrs CollectErrors
	if !errors.As(err, &collectErrs) {
		t.Fatal("Collect() error should be CollectErrors type")
	}
	if len(collectErrs.Errors) != 1 {
		t.Errorf("CollectErrors should have 1 error, got %d", len(collectErrs.Errors))
	}
	if collectErrs.Errors[0].Region != "us-east-1" {
		t.Errorf("CollectErrors region = %v, want us-east-1", collectErrs.Errors[0].Region)
	}
}

func TestCollector_Collect_MultipleRegionErrors(t *testing.T) {
	mock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return nil, errors.New("access denied")
		},
	}

	factory := func(r Region) ConfigClient { return mock }
	c := NewCollector("test", factory)

	regions := []Region{"us-east-1", "us-west-2", "eu-west-1"}
	inv, err := c.Collect(context.Background(), regions)
	if err == nil {
		t.Fatal("Collect() expected error, got nil")
	}
	if inv == nil {
		t.Fatal("Collect() should return partial inventory even on error")
	}

	var collectErrs CollectErrors
	if !errors.As(err, &collectErrs) {
		t.Fatal("Collect() error should be CollectErrors type")
	}
	if len(collectErrs.Errors) != 3 {
		t.Errorf("CollectErrors should have 3 errors, got %d", len(collectErrs.Errors))
	}

	failedRegions := collectErrs.Regions()
	if len(failedRegions) != 3 {
		t.Errorf("CollectErrors.Regions() should return 3 regions, got %d", len(failedRegions))
	}
}

func TestCollector_Collect_PartialSuccess(t *testing.T) {
	successMock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return &configservice.GetDiscoveredResourceCountsOutput{
				ResourceCounts: []types.ResourceCount{
					{ResourceType: "AWS::S3::Bucket", Count: 1},
				},
			}, nil
		},
		listDiscoveredResourcesFunc: func(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error) {
			return &configservice.ListDiscoveredResourcesOutput{
				ResourceIdentifiers: []types.ResourceIdentifier{
					{ResourceId: aws.String("bucket-1")},
				},
			}, nil
		},
		batchGetResourceConfigFunc: func(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error) {
			return &configservice.BatchGetResourceConfigOutput{
				BaseConfigurationItems: []types.BaseConfigurationItem{
					{
						ResourceType: "AWS::S3::Bucket",
						ResourceId:   aws.String("bucket-1"),
						AccountId:    aws.String("123456789012"),
					},
				},
			}, nil
		},
	}

	failMock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return nil, errors.New("access denied")
		},
	}

	factory := func(r Region) ConfigClient {
		if r == "us-east-1" {
			return successMock
		}
		return failMock
	}
	c := NewCollector("test", factory)

	regions := []Region{"us-east-1", "us-west-2"}
	inv, err := c.Collect(context.Background(), regions)

	if err == nil {
		t.Fatal("Collect() expected error for partial failure")
	}

	var collectErrs CollectErrors
	if !errors.As(err, &collectErrs) {
		t.Fatal("Collect() error should be CollectErrors type")
	}
	if len(collectErrs.Errors) != 1 {
		t.Errorf("CollectErrors should have 1 error (us-west-2), got %d", len(collectErrs.Errors))
	}
	if collectErrs.Errors[0].Region != "us-west-2" {
		t.Errorf("Failed region = %v, want us-west-2", collectErrs.Errors[0].Region)
	}

	if len(inv.Resources) != 1 {
		t.Errorf("Inventory should have 1 resource from successful region, got %d", len(inv.Resources))
	}
}

func TestCollector_Collect_NilClient(t *testing.T) {
	factory := func(r Region) ConfigClient { return nil }
	c := NewCollector("test", factory)

	inv, err := c.Collect(context.Background(), []Region{"us-east-1"})
	if err == nil {
		t.Fatal("Collect() expected error for nil client, got nil")
	}
	if !strings.Contains(err.Error(), "us-east-1") {
		t.Errorf("Collect() error = %v, want region context", err)
	}
	if inv == nil {
		t.Fatal("Collect() should return partial inventory even on error")
	}
}

func TestCollector_Collect_Pagination(t *testing.T) {
	callCount := 0
	mock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			callCount++
			if callCount == 1 {
				return &configservice.GetDiscoveredResourceCountsOutput{
					ResourceCounts: []types.ResourceCount{
						{ResourceType: "AWS::EC2::Instance", Count: 1},
					},
					NextToken: aws.String("token1"),
				}, nil
			}
			return &configservice.GetDiscoveredResourceCountsOutput{
				ResourceCounts: []types.ResourceCount{
					{ResourceType: "AWS::S3::Bucket", Count: 1},
				},
			}, nil
		},
		listDiscoveredResourcesFunc: func(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error) {
			return &configservice.ListDiscoveredResourcesOutput{
				ResourceIdentifiers: []types.ResourceIdentifier{
					{ResourceId: aws.String("resource-1")},
				},
			}, nil
		},
		batchGetResourceConfigFunc: func(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error) {
			return &configservice.BatchGetResourceConfigOutput{
				BaseConfigurationItems: []types.BaseConfigurationItem{
					{
						ResourceType: params.ResourceKeys[0].ResourceType,
						ResourceId:   aws.String("resource-1"),
						AccountId:    aws.String("123456789012"),
					},
				},
			}, nil
		},
	}

	factory := func(r Region) ConfigClient { return mock }
	c := NewCollector("test", factory)

	inv, err := c.Collect(context.Background(), []Region{"us-east-1"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if callCount != 2 {
		t.Errorf("GetDiscoveredResourceCounts called %v times, want 2", callCount)
	}
	if len(inv.Resources) != 2 {
		t.Errorf("Collect() resources = %v, want 2", len(inv.Resources))
	}
}

func TestCollector_Collect_BatchGetFallback(t *testing.T) {
	mock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return &configservice.GetDiscoveredResourceCountsOutput{
				ResourceCounts: []types.ResourceCount{
					{ResourceType: "AWS::EC2::Instance", Count: 1},
				},
			}, nil
		},
		listDiscoveredResourcesFunc: func(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error) {
			return &configservice.ListDiscoveredResourcesOutput{
				ResourceIdentifiers: []types.ResourceIdentifier{
					{ResourceId: aws.String("i-12345"), ResourceName: aws.String("instance-1")},
				},
			}, nil
		},
		batchGetResourceConfigFunc: func(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error) {
			return nil, errors.New("batch get failed")
		},
	}

	factory := func(r Region) ConfigClient { return mock }
	c := NewCollector("test", factory)

	inv, err := c.Collect(context.Background(), []Region{"us-east-1"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(inv.Resources) != 1 {
		t.Errorf("Collect() resources = %v, want 1 (fallback)", len(inv.Resources))
	}
	if inv.Resources[0].ResourceName != "instance-1" {
		t.Errorf("Collect() fallback resource name = %v, want instance-1", inv.Resources[0].ResourceName)
	}
}

func TestCollector_Collect_NoResources(t *testing.T) {
	mock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return &configservice.GetDiscoveredResourceCountsOutput{
				ResourceCounts: []types.ResourceCount{},
			}, nil
		},
	}

	factory := func(r Region) ConfigClient { return mock }
	c := NewCollector("test", factory)

	inv, err := c.Collect(context.Background(), []Region{"us-east-1"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(inv.Resources) != 0 {
		t.Errorf("Collect() resources = %v, want 0", len(inv.Resources))
	}
}

func TestCollector_Collect_WithLogger(t *testing.T) {
	mock := &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return &configservice.GetDiscoveredResourceCountsOutput{
				ResourceCounts: []types.ResourceCount{
					{ResourceType: "AWS::EC2::Instance", Count: 1},
				},
			}, nil
		},
		listDiscoveredResourcesFunc: func(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error) {
			return &configservice.ListDiscoveredResourcesOutput{
				ResourceIdentifiers: []types.ResourceIdentifier{
					{ResourceId: aws.String("i-12345"), ResourceName: aws.String("instance-1")},
				},
			}, nil
		},
		batchGetResourceConfigFunc: func(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error) {
			return &configservice.BatchGetResourceConfigOutput{
				BaseConfigurationItems: []types.BaseConfigurationItem{
					{
						ResourceType: "AWS::EC2::Instance",
						ResourceId:   aws.String("i-12345"),
						ResourceName: aws.String("instance-1"),
						AccountId:    aws.String("123456789012"),
					},
				},
			}, nil
		},
	}

	var logs []string
	factory := func(r Region) ConfigClient { return mock }
	c := NewCollector("test", factory)
	c.Logger = func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}

	_, err := c.Collect(context.Background(), []Region{"us-east-1"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(logs) == 0 {
		t.Error("Logger should have been called")
	}

	// Verify expected log messages
	hasStarting := false
	hasCompleted := false
	for _, log := range logs {
		if strings.Contains(log, "Starting collection") {
			hasStarting = true
		}
		if strings.Contains(log, "Completed with") {
			hasCompleted = true
		}
	}

	if !hasStarting {
		t.Error("Logger should have logged 'Starting collection'")
	}
	if !hasCompleted {
		t.Error("Logger should have logged 'Completed with'")
	}
}

package awsassetinventory

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
)

func instanceMock(selectFunc func(ctx context.Context, params *configservice.SelectResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.SelectResourceConfigOutput, error)) *mockConfigClient {
	return &mockConfigClient{
		getDiscoveredResourceCountsFunc: func(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return &configservice.GetDiscoveredResourceCountsOutput{
				ResourceCounts: []types.ResourceCount{{ResourceType: "AWS::EC2::Instance", Count: 1}},
			}, nil
		},
		listDiscoveredResourcesFunc: func(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error) {
			return &configservice.ListDiscoveredResourcesOutput{
				ResourceIdentifiers: []types.ResourceIdentifier{{ResourceId: aws.String("i-12345")}},
			}, nil
		},
		batchGetResourceConfigFunc: func(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error) {
			return &configservice.BatchGetResourceConfigOutput{
				BaseConfigurationItems: []types.BaseConfigurationItem{
					{ResourceType: "AWS::EC2::Instance", ResourceId: aws.String("i-12345")},
				},
			}, nil
		},
		selectResourceConfigFunc: selectFunc,
	}
}

func TestCollector_Collect_Relationships(t *testing.T) {
	var pages int
	mock := instanceMock(func(ctx context.Context, params *configservice.SelectResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.SelectResourceConfigOutput, error) {
		pages++
		if aws.ToString(params.Expression) != metadataQuery {
			t.Errorf("SelectResourceConfig expression = %q, want %q", aws.ToString(params.Expression), metadataQuery)
		}
		if params.NextToken == nil {
			return &configservice.SelectResourceConfigOutput{
				Results: []string{
					`{"resourceId":"vpc-1","resourceType":"AWS::EC2::VPC"}`,
					`not json`,
				},
				NextToken: aws.String("page2"),
			}, nil
		}
		return &configservice.SelectResourceConfigOutput{
			Results: []string{
				`{"resourceId":"i-12345","resourceType":"AWS::EC2::Instance",
				  "relationships":[
				    {"resourceId":"vpc-1","resourceType":"AWS::EC2::VPC","relationshipName":"Is contained in Vpc"},
				    {"resourceId":"vol-1","resourceType":"AWS::EC2::Volume","name":"Is attached to Volume"}],
				  "tags":[{"key":"Name","value":"web"}]}`,
			},
		}, nil
	})

	c := NewCollector("test", func(r Region) ConfigClient { return mock })
	inv, err := c.Collect(context.Background(), []Region{"us-east-1"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if pages != 2 {
		t.Errorf("SelectResourceConfig called %d times, want 2", pages)
	}
	if len(inv.Resources) != 1 {
		t.Fatalf("Collect() resources = %d, want 1", len(inv.Resources))
	}

	r := inv.Resources[0]
	want := []Relationship{
		{ResourceType: "AWS::EC2::VPC", ResourceID: "vpc-1", Name: "Is contained in Vpc"},
		{ResourceType: "AWS::EC2::Volume", ResourceID: "vol-1", Name: "Is attached to Volume"},
	}
	if len(r.Relationships) != len(want) {
		t.Fatalf("Relationships = %+v, want %+v", r.Relationships, want)
	}
	for i := range want {
		if r.Relationships[i] != want[i] {
			t.Errorf("Relationships[%d] = %+v, want %+v", i, r.Relationships[i], want[i])
		}
	}
	if r.Tags["Name"] != "web" {
		t.Errorf("Tags[Name] = %q, want web", r.Tags["Name"])
	}
}

func TestCollector_Collect_RelationshipsErrorIsWarning(t *testing.T) {
	mock := instanceMock(func(ctx context.Context, params *configservice.SelectResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.SelectResourceConfigOutput, error) {
		return nil, errors.New("AccessDeniedException")
	})

	var warnings []string
	c := NewCollector("test", func(r Region) ConfigClient { return mock })
	c.WarnLogger = func(format string, args ...any) {
		warnings = append(warnings, format)
	}

	inv, err := c.Collect(context.Background(), []Region{"us-east-1"})
	if err != nil {
		t.Fatalf("Collect() error = %v, want nil when only relationships fail", err)
	}
	if len(inv.Resources) != 1 {
		t.Errorf("Collect() resources = %d, want 1", len(inv.Resources))
	}
	if len(warnings) != 1 {
		t.Errorf("WarnLogger called %d times, want 1", len(warnings))
	}
}

func TestApplyMetadata_KeepsExistingTags(t *testing.T) {
	resources := []Resource{
		{ResourceType: "AWS::S3::Bucket", ResourceID: "b", Tags: map[string]string{"env": "prod"}},
		{ResourceType: "AWS::S3::Bucket", ResourceID: "other"},
	}
	applyMetadata(resources, map[string]resourceMetadata{
		resourceKey("AWS::S3::Bucket", "b"): {Tags: map[string]string{"env": "dev"}},
	})
	if resources[0].Tags["env"] != "prod" {
		t.Errorf("Tags[env] = %q, want prod", resources[0].Tags["env"])
	}
	if resources[1].Tags != nil || resources[1].Relationships != nil {
		t.Errorf("resource without metadata changed: %+v", resources[1])
	}
}

package awsassetinventory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
)

// ConfigClient defines the interface for AWS Config operations.
type ConfigClient interface {
	ListDiscoveredResources(ctx context.Context, params *configservice.ListDiscoveredResourcesInput, optFns ...func(*configservice.Options)) (*configservice.ListDiscoveredResourcesOutput, error)
	BatchGetResourceConfig(ctx context.Context, params *configservice.BatchGetResourceConfigInput, optFns ...func(*configservice.Options)) (*configservice.BatchGetResourceConfigOutput, error)
	GetDiscoveredResourceCounts(ctx context.Context, params *configservice.GetDiscoveredResourceCountsInput, optFns ...func(*configservice.Options)) (*configservice.GetDiscoveredResourceCountsOutput, error)
}

// ConfigClientFactory creates ConfigClient instances for specific regions.
type ConfigClientFactory func(region Region) ConfigClient

// Logger is a callback for progress messages.
type Logger func(format string, args ...any)

// Collector gathers AWS resources from AWS Config across regions.
type Collector struct {
	profile        string
	clientFactory  ConfigClientFactory
	Logger         Logger
	MaxConcurrency int // 0 means use default (5)
	MaxRetries     int // 0 means use default (3)
}

func (c *Collector) maxConcurrency() int {
	if c.MaxConcurrency > 0 {
		return c.MaxConcurrency
	}
	return DefaultMaxConcurrency
}

func (c *Collector) maxRetries() int {
	if c.MaxRetries > 0 {
		return c.MaxRetries
	}
	return DefaultMaxRetries
}

// NewCollector creates a new Collector with the given AWS config and profile name.
func NewCollector(profile string, clientFactory ConfigClientFactory) *Collector {
	return &Collector{
		profile:       profile,
		clientFactory: clientFactory,
	}
}

// CollectResult holds the result of collecting resources from a single region.
type CollectResult struct {
	Region    Region
	Resources []Resource
	Err       error
}

// Collect gathers all resources from AWS Config across the specified regions.
func (c *Collector) Collect(ctx context.Context, regions []Region) (*Inventory, error) {
	inv := NewInventory(c.profile, regions)

	resultCh := make(chan CollectResult, len(regions))
	sem := make(chan struct{}, c.maxConcurrency())
	var wg sync.WaitGroup

	for _, region := range regions {
		wg.Add(1)
		go func(r Region) {
			defer wg.Done()
			sem <- struct{}{}        // acquire semaphore
			defer func() { <-sem }() // release semaphore
			resources, err := c.collectRegion(ctx, r)
			resultCh <- CollectResult{Region: r, Resources: resources, Err: err}
		}(region)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var regionErrors []RegionError
	for result := range resultCh {
		if result.Err != nil {
			regionErrors = append(regionErrors, RegionError{
				Region: result.Region,
				Err:    result.Err,
			})
			continue
		}
		for _, r := range result.Resources {
			inv.AddResource(r)
		}
	}

	if len(regionErrors) > 0 {
		return inv, CollectErrors{Errors: regionErrors}
	}

	return inv, nil
}

func (c *Collector) collectRegion(ctx context.Context, region Region) ([]Resource, error) {
	if c.Logger != nil {
		c.Logger("[%s] Starting collection", region)
	}

	client := c.clientFactory(region)
	if client == nil {
		return nil, fmt.Errorf("nil AWS Config client for region %s", region)
	}

	resourceTypes, err := c.discoverResourceTypes(ctx, client)
	if err != nil {
		return nil, err
	}

	if c.Logger != nil {
		c.Logger("[%s] Found %d resource types", region, len(resourceTypes))
	}

	var resources []Resource
	for _, rt := range resourceTypes {
		rtResources, err := c.collectResourceType(ctx, client, region, rt)
		if err != nil {
			return resources, err
		}
		if c.Logger != nil && len(rtResources) > 0 {
			c.Logger("[%s] Collected %d %s", region, len(rtResources), rt)
		}
		resources = append(resources, rtResources...)
	}

	if c.Logger != nil {
		c.Logger("[%s] Completed with %d resources", region, len(resources))
	}

	return resources, nil
}

func (c *Collector) discoverResourceTypes(ctx context.Context, client ConfigClient) ([]types.ResourceType, error) {
	var resourceTypes []types.ResourceType
	var nextToken *string

	for {
		input := &configservice.GetDiscoveredResourceCountsInput{
			NextToken: nextToken,
		}

		output, err := retry(ctx, c.maxRetries(), func() (*configservice.GetDiscoveredResourceCountsOutput, error) {
			return client.GetDiscoveredResourceCounts(ctx, input)
		})
		if err != nil {
			return nil, err
		}

		for _, count := range output.ResourceCounts {
			if count.ResourceType != "" {
				resourceTypes = append(resourceTypes, count.ResourceType)
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return resourceTypes, nil
}

func (c *Collector) collectResourceType(ctx context.Context, client ConfigClient, region Region, resourceType types.ResourceType) ([]Resource, error) {
	var resources []Resource
	var nextToken *string

	for {
		input := &configservice.ListDiscoveredResourcesInput{
			ResourceType: resourceType,
			NextToken:    nextToken,
		}

		output, err := retry(ctx, c.maxRetries(), func() (*configservice.ListDiscoveredResourcesOutput, error) {
			return client.ListDiscoveredResources(ctx, input)
		})
		if err != nil {
			return nil, err
		}

		resourceKeys := make([]types.ResourceKey, 0, len(output.ResourceIdentifiers))
		for _, ri := range output.ResourceIdentifiers {
			resourceKeys = append(resourceKeys, types.ResourceKey{
				ResourceType: resourceType,
				ResourceId:   ri.ResourceId,
			})
		}

		if len(resourceKeys) > 0 {
			detailed, err := c.batchGetResources(ctx, client, region, resourceKeys)
			if err != nil {
				for _, ri := range output.ResourceIdentifiers {
					r := Resource{
						ResourceType: ResourceType(resourceType),
						ResourceID:   aws.ToString(ri.ResourceId),
						ResourceName: aws.ToString(ri.ResourceName),
						Region:       region,
					}
					resources = append(resources, r)
				}
			} else {
				resources = append(resources, detailed...)
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return resources, nil
}

func (c *Collector) batchGetResources(ctx context.Context, client ConfigClient, region Region, keys []types.ResourceKey) ([]Resource, error) {
	var resources []Resource

	for i := 0; i < len(keys); i += 100 {
		end := i + 100
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		input := &configservice.BatchGetResourceConfigInput{
			ResourceKeys: batch,
		}

		output, err := retry(ctx, c.maxRetries(), func() (*configservice.BatchGetResourceConfigOutput, error) {
			return client.BatchGetResourceConfig(ctx, input)
		})
		if err != nil {
			return nil, err
		}

		for _, item := range output.BaseConfigurationItems {
			var config json.RawMessage
			if item.Configuration != nil {
				config = json.RawMessage(*item.Configuration)
			}

			r := Resource{
				ResourceType:     ResourceType(item.ResourceType),
				ResourceID:       aws.ToString(item.ResourceId),
				ResourceName:     aws.ToString(item.ResourceName),
				Region:           region,
				AvailabilityZone: aws.ToString(item.AvailabilityZone),
				AccountID:        aws.ToString(item.AccountId),
				ARN:              aws.ToString(item.Arn),
				Configuration:    config,
			}

			resources = append(resources, r)
		}
	}

	return resources, nil
}

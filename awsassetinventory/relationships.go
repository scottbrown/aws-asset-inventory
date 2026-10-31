package awsassetinventory

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
)

// metadataQuery is the AWS Config advanced query that returns the relationships
// and tags of every resource in a region. BatchGetResourceConfig does not return them.
const metadataQuery = "SELECT resourceId, resourceType, relationships, tags"

// resourceMetadata holds the data that metadataQuery returns for one resource.
type resourceMetadata struct {
	Relationships []Relationship
	Tags          map[string]string
}

// selectResult is one row of the metadataQuery results.
type selectResult struct {
	ResourceID    string `json:"resourceId"`
	ResourceType  string `json:"resourceType"`
	Relationships []struct {
		ResourceID       string `json:"resourceId"`
		ResourceType     string `json:"resourceType"`
		ResourceName     string `json:"resourceName"`
		RelationshipName string `json:"relationshipName"`
		Name             string `json:"name"`
	} `json:"relationships"`
	Tags []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"tags"`
}

// resourceKey returns a key that identifies a resource within one region.
func resourceKey(rt ResourceType, id string) string {
	return string(rt) + "|" + id
}

// collectMetadata runs metadataQuery and returns the results keyed by resourceKey.
func (c *Collector) collectMetadata(ctx context.Context, client ConfigClient) (map[string]resourceMetadata, error) {
	metadata := make(map[string]resourceMetadata)
	var nextToken *string

	for {
		input := &configservice.SelectResourceConfigInput{
			Expression: aws.String(metadataQuery),
			Limit:      100,
			NextToken:  nextToken,
		}

		output, err := retry(ctx, c.maxRetries(), func() (*configservice.SelectResourceConfigOutput, error) {
			return client.SelectResourceConfig(ctx, input)
		})
		if err != nil {
			return nil, err
		}

		for _, raw := range output.Results {
			var row selectResult
			if err := json.Unmarshal([]byte(raw), &row); err != nil || row.ResourceID == "" {
				continue
			}

			var m resourceMetadata
			for _, rel := range row.Relationships {
				name := rel.RelationshipName
				if name == "" {
					name = rel.Name
				}
				m.Relationships = append(m.Relationships, Relationship{
					ResourceType: ResourceType(rel.ResourceType),
					ResourceID:   rel.ResourceID,
					ResourceName: rel.ResourceName,
					Name:         name,
				})
			}
			if len(row.Tags) > 0 {
				m.Tags = make(map[string]string, len(row.Tags))
				for _, tag := range row.Tags {
					m.Tags[tag.Key] = tag.Value
				}
			}

			metadata[resourceKey(ResourceType(row.ResourceType), row.ResourceID)] = m
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return metadata, nil
}

// applyMetadata copies relationships and tags into the matching resources.
func applyMetadata(resources []Resource, metadata map[string]resourceMetadata) {
	for i := range resources {
		m, ok := metadata[resourceKey(resources[i].ResourceType, resources[i].ResourceID)]
		if !ok {
			continue
		}
		resources[i].Relationships = m.Relationships
		if len(resources[i].Tags) == 0 && len(m.Tags) > 0 {
			resources[i].Tags = m.Tags
		}
	}
}

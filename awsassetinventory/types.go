package awsassetinventory

import (
	"encoding/json"
	"regexp"
	"time"
)

var regionPattern = regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d+$`)

// Region represents an AWS region identifier.
type Region string

// String returns the string representation of the region.
func (r Region) String() string {
	return string(r)
}

// IsValid checks if the region follows the AWS region naming pattern.
func (r Region) IsValid() bool {
	return regionPattern.MatchString(string(r))
}

// ResourceType represents an AWS Config resource type (e.g., AWS::EC2::Instance).
type ResourceType string

// String returns the string representation of the resource type.
func (rt ResourceType) String() string {
	return string(rt)
}

// Resource represents an AWS resource discovered by AWS Config.
type Resource struct {
	ResourceType     ResourceType      `json:"resourceType"`
	ResourceID       string            `json:"resourceId"`
	ResourceName     string            `json:"resourceName,omitempty"`
	Region           Region            `json:"awsRegion"`
	AvailabilityZone string            `json:"availabilityZone,omitempty"`
	AccountID        string            `json:"accountId"`
	ARN              string            `json:"arn,omitempty"`
	Configuration    json.RawMessage   `json:"configuration,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// Inventory holds the collection of AWS resources discovered across regions.
type Inventory struct {
	CollectedAt time.Time  `json:"collectedAt"`
	Profile     string     `json:"profile"`
	Regions     []Region   `json:"regions"`
	Resources   []Resource `json:"resources"`
}

// NewInventory creates a new Inventory with the given profile and regions.
func NewInventory(profile string, regions []Region) *Inventory {
	return &Inventory{
		CollectedAt: time.Now().UTC(),
		Profile:     profile,
		Regions:     regions,
		Resources:   make([]Resource, 0),
	}
}

// AddResource appends a resource to the inventory.
func (inv *Inventory) AddResource(r Resource) {
	inv.Resources = append(inv.Resources, r)
}

// ResourceCount returns the total number of resources in the inventory.
func (inv *Inventory) ResourceCount() int {
	return len(inv.Resources)
}

// ResourceCountByType returns a map of resource type to count.
func (inv *Inventory) ResourceCountByType() map[ResourceType]int {
	counts := make(map[ResourceType]int)
	for _, r := range inv.Resources {
		counts[r.ResourceType]++
	}
	return counts
}

// ResourceCountByRegion returns a map of region to count.
func (inv *Inventory) ResourceCountByRegion() map[Region]int {
	counts := make(map[Region]int)
	for _, r := range inv.Resources {
		counts[r.Region]++
	}
	return counts
}

// ResourceCountByTypeAndRegion returns a nested map of region to resource type to count.
func (inv *Inventory) ResourceCountByTypeAndRegion() map[Region]map[ResourceType]int {
	counts := make(map[Region]map[ResourceType]int)
	for _, r := range inv.Resources {
		if counts[r.Region] == nil {
			counts[r.Region] = make(map[ResourceType]int)
		}
		counts[r.Region][r.ResourceType]++
	}
	return counts
}

// ResourcesByType returns resources grouped by type.
func (inv *Inventory) ResourcesByType() map[ResourceType][]Resource {
	grouped := make(map[ResourceType][]Resource)
	for _, r := range inv.Resources {
		grouped[r.ResourceType] = append(grouped[r.ResourceType], r)
	}
	return grouped
}

// ToJSON serializes the inventory to JSON.
func (inv *Inventory) ToJSON() ([]byte, error) {
	return json.MarshalIndent(inv, "", "  ")
}

// LoadFromJSON deserializes an inventory from JSON data.
func LoadFromJSON(data []byte) (*Inventory, error) {
	var inv Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

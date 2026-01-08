package awsassetinventory

import (
	"encoding/json"
	"testing"
)

func TestRegion_String(t *testing.T) {
	r := Region("us-east-1")
	if got := r.String(); got != "us-east-1" {
		t.Errorf("Region.String() = %v, want %v", got, "us-east-1")
	}
}

func TestRegion_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		region Region
		want   bool
	}{
		{"valid us-east-1", Region("us-east-1"), true},
		{"valid eu-west-1", Region("eu-west-1"), true},
		{"valid ap-southeast-2", Region("ap-southeast-2"), true},
		{"too short", Region("us"), false},
		{"empty", Region(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.region.IsValid(); got != tt.want {
				t.Errorf("Region.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceType_String(t *testing.T) {
	rt := ResourceType("AWS::EC2::Instance")
	if got := rt.String(); got != "AWS::EC2::Instance" {
		t.Errorf("ResourceType.String() = %v, want %v", got, "AWS::EC2::Instance")
	}
}

func TestNewInventory(t *testing.T) {
	regions := []Region{"us-east-1", "us-west-2"}
	inv := NewInventory("test-profile", regions)

	if inv.Profile != "test-profile" {
		t.Errorf("NewInventory().Profile = %v, want %v", inv.Profile, "test-profile")
	}
	if len(inv.Regions) != 2 {
		t.Errorf("NewInventory().Regions length = %v, want %v", len(inv.Regions), 2)
	}
	if inv.CollectedAt.IsZero() {
		t.Error("NewInventory().CollectedAt should not be zero")
	}
	if inv.Resources == nil {
		t.Error("NewInventory().Resources should not be nil")
	}
}

func TestInventory_AddResource(t *testing.T) {
	inv := NewInventory("test", []Region{"us-east-1"})
	r := Resource{
		ResourceType: "AWS::EC2::Instance",
		ResourceID:   "i-12345",
		Region:       "us-east-1",
		AccountID:    "123456789012",
	}

	inv.AddResource(r)

	if len(inv.Resources) != 1 {
		t.Errorf("AddResource() resources length = %v, want %v", len(inv.Resources), 1)
	}
	if inv.Resources[0].ResourceID != "i-12345" {
		t.Errorf("AddResource() resource ID = %v, want %v", inv.Resources[0].ResourceID, "i-12345")
	}
}

func TestInventory_ResourceCount(t *testing.T) {
	inv := NewInventory("test", []Region{"us-east-1"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-1"})
	inv.AddResource(Resource{ResourceType: "AWS::S3::Bucket", ResourceID: "bucket-1"})

	if got := inv.ResourceCount(); got != 2 {
		t.Errorf("ResourceCount() = %v, want %v", got, 2)
	}
}

func TestInventory_ResourceCountByType(t *testing.T) {
	inv := NewInventory("test", []Region{"us-east-1"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-1"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-2"})
	inv.AddResource(Resource{ResourceType: "AWS::S3::Bucket", ResourceID: "bucket-1"})

	counts := inv.ResourceCountByType()

	if counts["AWS::EC2::Instance"] != 2 {
		t.Errorf("ResourceCountByType()[EC2] = %v, want %v", counts["AWS::EC2::Instance"], 2)
	}
	if counts["AWS::S3::Bucket"] != 1 {
		t.Errorf("ResourceCountByType()[S3] = %v, want %v", counts["AWS::S3::Bucket"], 1)
	}
}

func TestInventory_ResourceCountByRegion(t *testing.T) {
	inv := NewInventory("test", []Region{"us-east-1", "us-west-2"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-1", Region: "us-east-1"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-2", Region: "us-east-1"})
	inv.AddResource(Resource{ResourceType: "AWS::S3::Bucket", ResourceID: "bucket-1", Region: "us-west-2"})

	counts := inv.ResourceCountByRegion()

	if counts["us-east-1"] != 2 {
		t.Errorf("ResourceCountByRegion()[us-east-1] = %v, want %v", counts["us-east-1"], 2)
	}
	if counts["us-west-2"] != 1 {
		t.Errorf("ResourceCountByRegion()[us-west-2] = %v, want %v", counts["us-west-2"], 1)
	}
}

func TestInventory_ResourceCountByTypeAndRegion(t *testing.T) {
	inv := NewInventory("test", []Region{"us-east-1", "us-west-2"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-1", Region: "us-east-1"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-2", Region: "us-east-1"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-3", Region: "us-west-2"})
	inv.AddResource(Resource{ResourceType: "AWS::S3::Bucket", ResourceID: "bucket-1", Region: "us-west-2"})

	counts := inv.ResourceCountByTypeAndRegion()

	if counts["us-east-1"]["AWS::EC2::Instance"] != 2 {
		t.Errorf("ResourceCountByTypeAndRegion()[us-east-1][EC2] = %v, want %v", counts["us-east-1"]["AWS::EC2::Instance"], 2)
	}
	if counts["us-west-2"]["AWS::EC2::Instance"] != 1 {
		t.Errorf("ResourceCountByTypeAndRegion()[us-west-2][EC2] = %v, want %v", counts["us-west-2"]["AWS::EC2::Instance"], 1)
	}
	if counts["us-west-2"]["AWS::S3::Bucket"] != 1 {
		t.Errorf("ResourceCountByTypeAndRegion()[us-west-2][S3] = %v, want %v", counts["us-west-2"]["AWS::S3::Bucket"], 1)
	}
}

func TestInventory_ResourcesByType(t *testing.T) {
	inv := NewInventory("test", []Region{"us-east-1"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-1"})
	inv.AddResource(Resource{ResourceType: "AWS::EC2::Instance", ResourceID: "i-2"})
	inv.AddResource(Resource{ResourceType: "AWS::S3::Bucket", ResourceID: "bucket-1"})

	grouped := inv.ResourcesByType()

	if len(grouped["AWS::EC2::Instance"]) != 2 {
		t.Errorf("ResourcesByType()[EC2] length = %v, want %v", len(grouped["AWS::EC2::Instance"]), 2)
	}
	if len(grouped["AWS::S3::Bucket"]) != 1 {
		t.Errorf("ResourcesByType()[S3] length = %v, want %v", len(grouped["AWS::S3::Bucket"]), 1)
	}
}

func TestInventory_ToJSON(t *testing.T) {
	inv := NewInventory("test", []Region{"us-east-1"})
	inv.AddResource(Resource{
		ResourceType: "AWS::EC2::Instance",
		ResourceID:   "i-12345",
		Region:       "us-east-1",
		AccountID:    "123456789012",
	})

	data, err := inv.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var parsed Inventory
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("ToJSON() produced invalid JSON: %v", err)
	}

	if parsed.Profile != "test" {
		t.Errorf("ToJSON() parsed profile = %v, want %v", parsed.Profile, "test")
	}
	if len(parsed.Resources) != 1 {
		t.Errorf("ToJSON() parsed resources length = %v, want %v", len(parsed.Resources), 1)
	}
}

func TestResource_JSONMarshalling(t *testing.T) {
	r := Resource{
		ResourceType: "AWS::EC2::Instance",
		ResourceID:   "i-12345",
		ResourceName: "my-instance",
		Region:       "us-east-1",
		AccountID:    "123456789012",
		ARN:          "arn:aws:ec2:us-east-1:123456789012:instance/i-12345",
		Tags: map[string]string{
			"Name": "my-instance",
			"Env":  "prod",
		},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed Resource
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if parsed.ResourceID != r.ResourceID {
		t.Errorf("parsed.ResourceID = %v, want %v", parsed.ResourceID, r.ResourceID)
	}
	if parsed.Tags["Name"] != "my-instance" {
		t.Errorf("parsed.Tags[Name] = %v, want %v", parsed.Tags["Name"], "my-instance")
	}
}

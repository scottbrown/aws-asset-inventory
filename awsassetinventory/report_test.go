package awsassetinventory

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewReportGenerator(t *testing.T) {
	inv := NewInventory("test", []Region{"us-east-1"})
	rg := NewReportGenerator(inv)

	if rg.inventory != inv {
		t.Error("NewReportGenerator() should store inventory reference")
	}
}

func TestReportGenerator_Generate_EmptyInventory(t *testing.T) {
	inv := &Inventory{
		CollectedAt: time.Date(2026, 1, 7, 15, 30, 0, 0, time.UTC),
		Profile:     "test-profile",
		Regions:     []Region{"us-east-1"},
		Resources:   []Resource{},
	}
	rg := NewReportGenerator(inv)

	var buf bytes.Buffer
	err := rg.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "# AWS Asset Inventory Report") {
		t.Error("Generate() should include header")
	}
	if !strings.Contains(output, "**Profile:** test-profile") {
		t.Error("Generate() should include profile")
	}
	if !strings.Contains(output, "**Regions:** us-east-1") {
		t.Error("Generate() should include regions")
	}
	if !strings.Contains(output, "**Total Resources:** 0") {
		t.Error("Generate() should include total resources count")
	}
	if !strings.Contains(output, "No resources found.") {
		t.Error("Generate() should indicate no resources")
	}
}

func TestReportGenerator_Generate_WithResources(t *testing.T) {
	inv := &Inventory{
		CollectedAt: time.Date(2026, 1, 7, 15, 30, 0, 0, time.UTC),
		Profile:     "prod",
		Regions:     []Region{"us-east-1", "us-west-2"},
		Resources: []Resource{
			{
				ResourceType: "AWS::EC2::Instance",
				ResourceID:   "i-12345",
				ResourceName: "web-server-1",
				Region:       "us-east-1",
				ARN:          "arn:aws:ec2:us-east-1:123456789012:instance/i-12345",
			},
			{
				ResourceType: "AWS::EC2::Instance",
				ResourceID:   "i-67890",
				ResourceName: "web-server-2",
				Region:       "us-west-2",
				ARN:          "arn:aws:ec2:us-west-2:123456789012:instance/i-67890",
			},
			{
				ResourceType: "AWS::S3::Bucket",
				ResourceID:   "my-bucket",
				ResourceName: "my-bucket",
				Region:       "us-east-1",
				ARN:          "arn:aws:s3:::my-bucket",
			},
		},
	}
	rg := NewReportGenerator(inv)

	var buf bytes.Buffer
	err := rg.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "**Total Resources:** 3") {
		t.Error("Generate() should show correct total")
	}
	if !strings.Contains(output, "## Summary") {
		t.Error("Generate() should include summary section")
	}
	if !strings.Contains(output, "AWS::EC2::Instance") {
		t.Error("Generate() should include EC2 instances in summary")
	}
	if !strings.Contains(output, "AWS::S3::Bucket") {
		t.Error("Generate() should include S3 buckets in summary")
	}
	if !strings.Contains(output, "## By Region") {
		t.Error("Generate() should include by region section")
	}
	if !strings.Contains(output, "### us-east-1") {
		t.Error("Generate() should include us-east-1 section")
	}
	if !strings.Contains(output, "### us-west-2") {
		t.Error("Generate() should include us-west-2 section")
	}
	if !strings.Contains(output, "## Resource Details") {
		t.Error("Generate() should include resource details section")
	}
	if !strings.Contains(output, "web-server-1") {
		t.Error("Generate() should include resource names")
	}
	if !strings.Contains(output, "i-12345") {
		t.Error("Generate() should include resource IDs")
	}
}

func TestReportGenerator_Generate_MultipleRegions(t *testing.T) {
	inv := &Inventory{
		CollectedAt: time.Date(2026, 1, 7, 15, 30, 0, 0, time.UTC),
		Profile:     "test",
		Regions:     []Region{"us-east-1", "eu-west-1", "ap-southeast-2"},
		Resources:   []Resource{},
	}
	rg := NewReportGenerator(inv)

	var buf bytes.Buffer
	err := rg.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "**Regions:** us-east-1, eu-west-1, ap-southeast-2") {
		t.Errorf("Generate() regions = %v, want comma-separated list", output)
	}
}

func TestReportGenerator_Generate_ResourceWithoutName(t *testing.T) {
	inv := &Inventory{
		CollectedAt: time.Date(2026, 1, 7, 15, 30, 0, 0, time.UTC),
		Profile:     "test",
		Regions:     []Region{"us-east-1"},
		Resources: []Resource{
			{
				ResourceType: "AWS::EC2::Instance",
				ResourceID:   "i-12345",
				ResourceName: "",
				Region:       "us-east-1",
				ARN:          "",
			},
		},
	}
	rg := NewReportGenerator(inv)

	var buf bytes.Buffer
	err := rg.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "| - |") {
		t.Error("Generate() should show '-' for missing name")
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no special chars", "hello", "hello"},
		{"pipe char", "a|b", "a\\|b"},
		{"newline", "a\nb", "a b"},
		{"multiple pipes", "a|b|c", "a\\|b\\|c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeMarkdown(tt.input); got != tt.want {
				t.Errorf("escapeMarkdown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruncateARN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"short ARN", "arn:aws:s3:::bucket", "arn:aws:s3:::bucket"},
		{"exact length", strings.Repeat("a", 60), strings.Repeat("a", 60)},
		{"long ARN", strings.Repeat("a", 70), strings.Repeat("a", 57) + "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateARN(tt.input); got != tt.want {
				t.Errorf("truncateARN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortedResourceTypes(t *testing.T) {
	counts := map[ResourceType]int{
		"AWS::S3::Bucket":     1,
		"AWS::EC2::Instance":  2,
		"AWS::Lambda::Function": 1,
	}

	sorted := sortedResourceTypes(counts)

	if len(sorted) != 3 {
		t.Fatalf("sortedResourceTypes() length = %v, want 3", len(sorted))
	}
	if sorted[0] != "AWS::EC2::Instance" {
		t.Errorf("sortedResourceTypes()[0] = %v, want AWS::EC2::Instance", sorted[0])
	}
	if sorted[1] != "AWS::Lambda::Function" {
		t.Errorf("sortedResourceTypes()[1] = %v, want AWS::Lambda::Function", sorted[1])
	}
	if sorted[2] != "AWS::S3::Bucket" {
		t.Errorf("sortedResourceTypes()[2] = %v, want AWS::S3::Bucket", sorted[2])
	}
}

func TestSortedRegions(t *testing.T) {
	countsByRegion := map[Region]map[ResourceType]int{
		"us-west-2":      {"AWS::EC2::Instance": 1},
		"eu-west-1":      {"AWS::EC2::Instance": 1},
		"ap-southeast-2": {"AWS::EC2::Instance": 1},
	}

	sorted := sortedRegions(countsByRegion)

	if len(sorted) != 3 {
		t.Fatalf("sortedRegions() length = %v, want 3", len(sorted))
	}
	if sorted[0] != "ap-southeast-2" {
		t.Errorf("sortedRegions()[0] = %v, want ap-southeast-2", sorted[0])
	}
	if sorted[1] != "eu-west-1" {
		t.Errorf("sortedRegions()[1] = %v, want eu-west-1", sorted[1])
	}
	if sorted[2] != "us-west-2" {
		t.Errorf("sortedRegions()[2] = %v, want us-west-2", sorted[2])
	}
}

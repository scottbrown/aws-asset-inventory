package main

import (
	"testing"

	"github.com/scottbrown/aws-asset-inventory/awsassetinventory"
)

func TestParseRegions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []awsassetinventory.Region
	}{
		{
			name:  "single region",
			input: "us-east-1",
			want:  []awsassetinventory.Region{"us-east-1"},
		},
		{
			name:  "multiple regions",
			input: "us-east-1,us-west-2,eu-west-1",
			want:  []awsassetinventory.Region{"us-east-1", "us-west-2", "eu-west-1"},
		},
		{
			name:  "regions with spaces",
			input: "us-east-1, us-west-2 , eu-west-1",
			want:  []awsassetinventory.Region{"us-east-1", "us-west-2", "eu-west-1"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []awsassetinventory.Region{},
		},
		{
			name:  "only commas",
			input: ",,",
			want:  []awsassetinventory.Region{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRegions(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseRegions() returned %d regions, want %d", len(got), len(tt.want))
				return
			}
			for i, r := range got {
				if r != tt.want[i] {
					t.Errorf("parseRegions()[%d] = %s, want %s", i, r, tt.want[i])
				}
			}
		})
	}
}

func TestRegionStrings(t *testing.T) {
	regions := []awsassetinventory.Region{"us-east-1", "us-west-2"}
	got := regionStrings(regions)

	if len(got) != 2 {
		t.Fatalf("regionStrings() returned %d strings, want 2", len(got))
	}
	if got[0] != "us-east-1" {
		t.Errorf("regionStrings()[0] = %s, want us-east-1", got[0])
	}
	if got[1] != "us-west-2" {
		t.Errorf("regionStrings()[1] = %s, want us-west-2", got[1])
	}
}

func TestCollectRequiresRegions(t *testing.T) {
	// Save original values
	origRegions := collectRegions
	t.Cleanup(func() {
		collectRegions = origRegions
	})

	collectRegions = ""

	err := runCollect(nil, nil)
	if err == nil {
		t.Error("runCollect should return error when regions is empty")
	}
}

func TestCollectValidatesRegions(t *testing.T) {
	// Save original values
	origRegions := collectRegions
	t.Cleanup(func() {
		collectRegions = origRegions
	})

	collectRegions = "invalid-region"

	err := runCollect(nil, nil)
	if err == nil {
		t.Error("runCollect should return error for invalid region")
	}
}

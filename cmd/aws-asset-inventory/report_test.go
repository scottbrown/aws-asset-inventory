package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scottbrown/aws-asset-inventory/awsassetinventory"
)

func TestReportRequiresInput(t *testing.T) {
	// Save original values
	origInput := reportInput
	t.Cleanup(func() {
		reportInput = origInput
	})

	reportInput = ""

	err := runReport(nil, nil)
	if err == nil {
		t.Error("runReport should return error when input is empty")
	}
}

func TestReportWithNonexistentFile(t *testing.T) {
	// Save original values
	origInput := reportInput
	t.Cleanup(func() {
		reportInput = origInput
	})

	reportInput = "/nonexistent/path/to/file.json"

	err := runReport(nil, nil)
	if err == nil {
		t.Error("runReport should return error for nonexistent file")
	}
}

func TestReportWithInvalidJSON(t *testing.T) {
	// Create temp file with invalid JSON
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(tmpFile, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Save original values
	origInput := reportInput
	t.Cleanup(func() {
		reportInput = origInput
	})

	reportInput = tmpFile

	err := runReport(nil, nil)
	if err == nil {
		t.Error("runReport should return error for invalid JSON")
	}
}

func TestReportWithValidInput(t *testing.T) {
	// Create a valid inventory JSON
	inv := awsassetinventory.NewInventory("test-profile", []awsassetinventory.Region{"us-east-1"})
	inv.AddResource(awsassetinventory.Resource{
		ResourceType: "AWS::EC2::Instance",
		ResourceID:   "i-12345",
		Region:       "us-east-1",
		AccountID:    "123456789012",
	})

	data, err := inv.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize inventory: %v", err)
	}

	// Create temp input file
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "inventory.json")
	if err := os.WriteFile(inputFile, data, 0644); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	// Create temp output file path
	outputFile := filepath.Join(tmpDir, "report.md")

	// Save original values
	origInput := reportInput
	origOutput := reportOutput
	origDetails := reportIncludeDetails
	t.Cleanup(func() {
		reportInput = origInput
		reportOutput = origOutput
		reportIncludeDetails = origDetails
	})

	reportInput = inputFile
	reportOutput = outputFile
	reportIncludeDetails = false

	err = runReport(nil, nil)
	if err != nil {
		t.Fatalf("runReport failed: %v", err)
	}

	// Verify output file exists and has content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("report output should not be empty")
	}

	// Verify report contains expected sections
	reportStr := string(content)
	if !strings.Contains(reportStr, "AWS Asset Inventory Report") {
		t.Error("report should contain header")
	}
	if !strings.Contains(reportStr, "AWS::EC2::Instance") {
		t.Error("report should contain resource type")
	}
}

func TestReportWithDetails(t *testing.T) {
	// Create a valid inventory JSON
	inv := awsassetinventory.NewInventory("test-profile", []awsassetinventory.Region{"us-east-1"})
	inv.AddResource(awsassetinventory.Resource{
		ResourceType: "AWS::EC2::Instance",
		ResourceID:   "i-12345",
		ResourceName: "my-instance",
		Region:       "us-east-1",
		AccountID:    "123456789012",
	})

	data, err := inv.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize inventory: %v", err)
	}

	// Create temp files
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "inventory.json")
	if err := os.WriteFile(inputFile, data, 0644); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "report.md")

	// Save original values
	origInput := reportInput
	origOutput := reportOutput
	origDetails := reportIncludeDetails
	t.Cleanup(func() {
		reportInput = origInput
		reportOutput = origOutput
		reportIncludeDetails = origDetails
	})

	reportInput = inputFile
	reportOutput = outputFile
	reportIncludeDetails = true

	err = runReport(nil, nil)
	if err != nil {
		t.Fatalf("runReport failed: %v", err)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	reportStr := string(content)
	if !strings.Contains(reportStr, "Resource Details") {
		t.Error("report with --include-details should contain Resource Details section")
	}
	if !strings.Contains(reportStr, "i-12345") {
		t.Error("report with --include-details should contain resource ID")
	}
}

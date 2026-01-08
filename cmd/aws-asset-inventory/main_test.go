package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = orig
	out := <-done
	_ = r.Close()
	return out
}

func TestPermissionsFlagOutputsList(t *testing.T) {
	prevPermissionsOnly := permissionsOnly
	permissionsOnly = true
	t.Cleanup(func() {
		permissionsOnly = prevPermissionsOnly
	})

	output := captureStdout(t, func() {
		if err := run(nil, nil); err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	expected := strings.Join(requiredPermissions(), "\n") + "\n"
	if output != expected {
		t.Fatalf("unexpected output:\n%q\nwant:\n%q", output, expected)
	}
}

func TestRunRequiresRegions(t *testing.T) {
	prevProfile := profile
	prevRegions := regions
	t.Cleanup(func() {
		profile = prevProfile
		regions = prevRegions
	})

	profile = ""
	regions = ""

	err := run(nil, nil)
	if err == nil {
		t.Fatal("expected error for missing regions")
	}
	if !strings.Contains(err.Error(), "--regions is required") {
		t.Fatalf("expected regions required error, got: %v", err)
	}
}

func TestRunDoesNotRequireProfile(t *testing.T) {
	prevProfile := profile
	prevRegions := regions
	t.Cleanup(func() {
		profile = prevProfile
		regions = prevRegions
	})

	profile = ""
	regions = "us-east-1"

	// The run will fail when trying to use AWS credentials (expected),
	// but it should NOT fail with "--profile is required" error
	err := run(nil, nil)
	if err != nil && strings.Contains(err.Error(), "--profile is required") {
		t.Fatalf("profile should be optional, got: %v", err)
	}
	// Any other error (like credential failure) is acceptable for this test
}

func TestOutputStdoutConflict(t *testing.T) {
	prevRegions := regions
	prevOutputFile := outputFile
	prevReportFile := reportFile
	prevNoReport := noReport
	t.Cleanup(func() {
		regions = prevRegions
		outputFile = prevOutputFile
		reportFile = prevReportFile
		noReport = prevNoReport
	})

	regions = "us-east-1"
	outputFile = "-"
	reportFile = ""
	noReport = false

	err := run(nil, nil)
	if err == nil {
		t.Fatal("expected error for stdout conflict")
	}
	if !strings.Contains(err.Error(), "cannot write both JSON and report to stdout") {
		t.Fatalf("expected stdout conflict error, got: %v", err)
	}
}

func TestOutputStdoutWithExplicitReportDash(t *testing.T) {
	prevRegions := regions
	prevOutputFile := outputFile
	prevReportFile := reportFile
	prevNoReport := noReport
	t.Cleanup(func() {
		regions = prevRegions
		outputFile = prevOutputFile
		reportFile = prevReportFile
		noReport = prevNoReport
	})

	regions = "us-east-1"
	outputFile = "-"
	reportFile = "-"
	noReport = false

	err := run(nil, nil)
	if err == nil {
		t.Fatal("expected error for stdout conflict")
	}
	if !strings.Contains(err.Error(), "cannot write both JSON and report to stdout") {
		t.Fatalf("expected stdout conflict error, got: %v", err)
	}
}

func TestNoReportFlagAllowsOutputStdout(t *testing.T) {
	prevRegions := regions
	prevOutputFile := outputFile
	prevReportFile := reportFile
	prevNoReport := noReport
	t.Cleanup(func() {
		regions = prevRegions
		outputFile = prevOutputFile
		reportFile = prevReportFile
		noReport = prevNoReport
	})

	regions = "us-east-1"
	outputFile = "-"
	reportFile = ""
	noReport = true

	// Should not fail with stdout conflict error
	err := run(nil, nil)
	if err != nil && strings.Contains(err.Error(), "cannot write both JSON and report to stdout") {
		t.Fatalf("--no-report should allow --output -, got: %v", err)
	}
	// Other errors (like AWS credential issues) are acceptable
}

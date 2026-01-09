package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	// Save original values
	origBranch := gitBranch
	origSHA := gitSHA
	t.Cleanup(func() {
		gitBranch = origBranch
		gitSHA = origSHA
	})

	gitBranch = "test-branch"
	gitSHA = "abc123"

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test-branch") {
		t.Errorf("version output should contain branch, got: %s", output)
	}
	if !strings.Contains(output, "abc123") {
		t.Errorf("version output should contain SHA, got: %s", output)
	}
}

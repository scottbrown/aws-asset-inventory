package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPermissionsCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"permissions"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("permissions command failed: %v", err)
	}

	output := buf.String()

	expectedPerms := requiredPermissions()
	for _, perm := range expectedPerms {
		if !strings.Contains(output, perm) {
			t.Errorf("permissions output should contain %s, got: %s", perm, output)
		}
	}
}

func TestRequiredPermissions(t *testing.T) {
	perms := requiredPermissions()

	if len(perms) == 0 {
		t.Error("requiredPermissions should return at least one permission")
	}

	expected := []string{
		"config:GetDiscoveredResourceCounts",
		"config:ListDiscoveredResources",
		"config:BatchGetResourceConfig",
	}

	if len(perms) != len(expected) {
		t.Errorf("requiredPermissions() returned %d permissions, want %d", len(perms), len(expected))
	}

	for i, perm := range expected {
		if perms[i] != perm {
			t.Errorf("requiredPermissions()[%d] = %s, want %s", i, perms[i], perm)
		}
	}
}

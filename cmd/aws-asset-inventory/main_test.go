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

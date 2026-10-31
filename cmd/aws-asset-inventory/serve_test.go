package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestServeWithNonexistentFile(t *testing.T) {
	origInput := serveInput
	t.Cleanup(func() {
		serveInput = origInput
	})

	serveInput = "/nonexistent/path/to/file.json"

	if err := runServe(nil, nil); err == nil {
		t.Error("runServe should return error for nonexistent file")
	}
}

func TestServeWithInvalidJSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(tmpFile, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	origInput := serveInput
	t.Cleanup(func() {
		serveInput = origInput
	})

	serveInput = tmpFile

	if err := runServe(nil, nil); err == nil {
		t.Error("runServe should return error for invalid JSON")
	}
}

func TestServeWithBadListenAddress(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "inventory.json")
	if err := os.WriteFile(tmpFile, []byte(`{"resources":[]}`), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	origInput, origListen := serveInput, serveListen
	t.Cleanup(func() {
		serveInput, serveListen = origInput, origListen
	})

	serveInput = tmpFile
	serveListen = "not-an-address"

	if err := runServe(nil, nil); err == nil {
		t.Error("runServe should return error for an invalid listen address")
	}
}

func TestIsLoopbackAddr(t *testing.T) {
	tests := map[string]bool{
		"127.0.0.1:8080": true,
		"localhost:8080": true,
		"[::1]:8080":     true,
		":8080":          false,
		"0.0.0.0:8080":   false,
		"10.0.0.5:8080":  false,
		"bad":            false,
	}
	for addr, want := range tests {
		if got := isLoopbackAddr(addr); got != want {
			t.Errorf("isLoopbackAddr(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestServerURL(t *testing.T) {
	tests := []struct {
		addr *net.TCPAddr
		want string
	}{
		{&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}, "http://127.0.0.1:8080"},
		{&net.TCPAddr{IP: net.IPv4zero, Port: 9000}, "http://localhost:9000"},
	}
	for _, tt := range tests {
		if got := serverURL(tt.addr); got != tt.want {
			t.Errorf("serverURL(%v) = %q, want %q", tt.addr, got, tt.want)
		}
	}
}

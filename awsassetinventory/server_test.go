package awsassetinventory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testServer() *Server {
	inv := NewInventory("test", []Region{"us-east-1"})
	inv.AddResource(Resource{
		ResourceType:  "AWS::EC2::Instance",
		ResourceID:    "i-1",
		Region:        "us-east-1",
		Configuration: json.RawMessage(`{"instanceType":"t3.micro"}`),
		Relationships: []Relationship{{ResourceType: "AWS::EC2::VPC", ResourceID: "vpc-1"}},
	})
	return NewServer(inv)
}

func get(t *testing.T, h http.Handler, host, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = host
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestServer_Static(t *testing.T) {
	h := testServer().Handler()
	for _, path := range []string{"/", "/app.js", "/style.css", "/vendor/3d-force-graph.min.js"} {
		rec := get(t, h, "localhost:8080", path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
		if rec.Header().Get("Content-Security-Policy") == "" {
			t.Errorf("GET %s has no Content-Security-Policy header", path)
		}
	}
}

func TestServer_Graph(t *testing.T) {
	rec := get(t, testServer().Handler(), "localhost", "/api/graph")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/graph = %d, want 200", rec.Code)
	}
	var g Graph
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(g.Nodes) != 2 || len(g.Links) != 1 {
		t.Errorf("graph = %d nodes, %d links, want 2 and 1", len(g.Nodes), len(g.Links))
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("API responses should not be cached")
	}
}

func TestServer_Summary(t *testing.T) {
	rec := get(t, testServer().Handler(), "localhost", "/api/summary")
	var s Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if s.Resources != 1 || s.Nodes != 2 || s.Links != 1 || s.Profile != "test" {
		t.Errorf("summary = %+v", s)
	}
}

func TestServer_Resource(t *testing.T) {
	h := testServer().Handler()

	rec := get(t, h, "localhost", "/api/resource?id="+url.QueryEscape("us-east-1|AWS::EC2::Instance|i-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET resource = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "t3.micro") {
		t.Errorf("resource response should include configuration: %s", rec.Body.String())
	}

	rec = get(t, h, "localhost", "/api/resource?id="+url.QueryEscape("us-east-1|AWS::EC2::VPC|vpc-1"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET stub resource = %d, want 404", rec.Code)
	}
}

func TestServer_LoopbackOnly(t *testing.T) {
	srv := testServer()
	srv.LoopbackOnly = true
	h := srv.Handler()

	tests := map[string]int{
		"localhost:8080":    http.StatusOK,
		"127.0.0.1:8080":    http.StatusOK,
		"[::1]:8080":        http.StatusOK,
		"LOCALHOST":         http.StatusOK,
		"evil.example:8080": http.StatusForbidden,
		"10.0.0.5:8080":     http.StatusForbidden,
	}
	for host, want := range tests {
		if rec := get(t, h, host, "/api/summary"); rec.Code != want {
			t.Errorf("Host %s = %d, want %d", host, rec.Code, want)
		}
	}

	srv.LoopbackOnly = false
	if rec := get(t, srv.Handler(), "evil.example", "/api/summary"); rec.Code != http.StatusOK {
		t.Errorf("without LoopbackOnly any host should work, got %d", rec.Code)
	}
}

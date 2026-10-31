package awsassetinventory

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

// Server serves the 3D inventory viewer and the JSON API it uses.
type Server struct {
	inventory *Inventory
	graph     *Graph
	resources map[string]int // node ID -> position in inventory.Resources

	// LoopbackOnly rejects requests whose Host header is not a loopback name or address.
	// Set it when the server listens on a loopback address, to stop DNS rebinding attacks.
	LoopbackOnly bool
}

// Summary describes the inventory that the server shows.
type Summary struct {
	CollectedAt time.Time `json:"collectedAt"`
	Profile     string    `json:"profile"`
	Regions     []Region  `json:"regions"`
	Resources   int       `json:"resources"`
	Nodes       int       `json:"nodes"`
	Links       int       `json:"links"`
}

// NewServer creates a Server for the given inventory.
func NewServer(inv *Inventory) *Server {
	resources := make(map[string]int, len(inv.Resources))
	for i, r := range inv.Resources {
		if _, dup := resources[NodeID(r)]; !dup {
			resources[NodeID(r)] = i
		}
	}
	return &Server{
		inventory: inv,
		graph:     BuildGraph(inv),
		resources: resources,
	}
}

// Handler returns the HTTP handler for the viewer and the API.
func (s *Server) Handler() http.Handler {
	static, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err) // the embedded directory always exists
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(static))
	mux.HandleFunc("GET /api/summary", s.handleSummary)
	mux.HandleFunc("GET /api/graph", s.handleGraph)
	mux.HandleFunc("GET /api/resource", s.handleResource)
	return s.protect(mux)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Summary{
		CollectedAt: s.inventory.CollectedAt,
		Profile:     s.inventory.Profile,
		Regions:     s.inventory.Regions,
		Resources:   s.inventory.ResourceCount(),
		Nodes:       len(s.graph.Nodes),
		Links:       len(s.graph.Links),
	})
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.graph)
}

func (s *Server) handleResource(w http.ResponseWriter, r *http.Request) {
	i, ok := s.resources[r.URL.Query().Get("id")]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "resource not found"})
		return
	}
	writeJSON(w, http.StatusOK, s.inventory.Resources[i])
}

// protect adds security headers and, when LoopbackOnly is set, checks the Host header.
func (s *Server) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.LoopbackOnly && !isLoopbackHost(r.Host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func isLoopbackHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/scottbrown/aws-asset-inventory/awsassetinventory"
	"github.com/spf13/cobra"
)

var (
	serveInput  string
	serveListen string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Explore inventory JSON in a 3D map in the browser",
	Long: `Start a local web server that shows a previously collected inventory as a 3D map.
Each resource is a node and each AWS Config relationship is a link. Fly through the
map, filter by region and service, and click a resource to see its details.`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringVarP(&serveInput, "input", "i", "", "Input JSON inventory file (required)")
	serveCmd.Flags().StringVarP(&serveListen, "listen", "l", "127.0.0.1:8080", "Address to listen on")

	_ = serveCmd.MarkFlagRequired("input")
}

func runServe(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(serveInput)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	inventory, err := awsassetinventory.LoadFromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse inventory JSON: %w", err)
	}

	srv := awsassetinventory.NewServer(inventory)
	srv.LoopbackOnly = isLoopbackAddr(serveListen)

	ln, err := net.Listen("tcp", serveListen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", serveListen, err)
	}

	if !srv.LoopbackOnly {
		fmt.Fprintln(os.Stderr, "Warning: listening on a non-loopback address. Anyone who can reach it can read the inventory.")
	}
	fmt.Fprintf(os.Stderr, "Serving %d resources at %s (press Ctrl+C to stop)\n", inventory.ResourceCount(), serverURL(ln.Addr()))

	httpServer := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// isLoopbackAddr reports whether a listen address only accepts local connections.
func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// serverURL returns the URL to open in a browser for a listening address.
func serverURL(addr net.Addr) string {
	tcp, ok := addr.(*net.TCPAddr)
	if !ok || tcp.IP.IsUnspecified() {
		_, port, _ := net.SplitHostPort(addr.String())
		return "http://localhost:" + port
	}
	return "http://" + tcp.String()
}

package viewer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/raqolbi/qolauncher/internal/config"
)

// Server serves the HTTP log viewer.
type Server struct {
	dir    string
	server *http.Server
}

// New creates a log viewer Server from configuration.
func New(cfg *config.Config) *Server {
	s := &Server{dir: cfg.LogDir}

	inner := http.NewServeMux()
	inner.HandleFunc("GET /", s.handleRoot)
	inner.HandleFunc("GET /logs", s.handleLogsListHTML)
	inner.HandleFunc("GET /logs/{filename}/download", s.handleLogDownload)
	inner.HandleFunc("GET /logs/{filename}", s.handleLogView)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.Handle("/", basicAuth(cfg.LogUsername, cfg.LogPassword, inner))

	s.server = &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", cfg.LogPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

// Handler returns the HTTP handler (for tests).
func (s *Server) Handler() http.Handler {
	return s.server.Handler
}

// Start listens in a background goroutine.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return err
	}
	go func() {
		_ = s.server.Serve(ln)
	}()
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

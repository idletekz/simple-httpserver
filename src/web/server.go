package web

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server add timeout for shutdown
type Server struct {
	*http.Server
	Timeout time.Duration
}

// ListenAndServe add graceful shutdown
// ListenAndServe always returns a non-nil error
func (s *Server) ListenAndServe() error {
	server := s.Server
	serverErr := make(chan error)
	go func() {
		serverErr <- server.ListenAndServe()
	}()
	// Listen for an interrupt signal from the OS. Use a buffered
	// channel because of how the signal package is implemented.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case <-osSignals:
		server.SetKeepAlivesEnabled(false)
		ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
		defer cancel()
		// Attempt the graceful shutdown by closing the listener and
		// completing all inflight requests.
		var sdErr error
		if err := server.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout
			sdErr = err
			if err := server.Close(); err != nil {
				sdErr = err
			}
		}
		if sdErr != nil {
			return sdErr
		}
		return <-serverErr
	}
}

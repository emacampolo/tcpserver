package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/emacampolo/tcpserver"
)

func main() {
	server := tcpserver.New(tcpserver.Config{Address: "127.0.0.1:8080"})

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve()
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		slog.Error("server error", "error", err)
	case <-done:
		slog.Info("shutting down")
		server.Shutdown()
	}
}

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// Config defines the HTTP server startup options.
type Config struct {
	Host        string
	Port        int
	OpenBrowser bool
}

// Start boots the embedded HTTP web fuzzy finder server and blocks until SIGINT/SIGTERM.
func Start(cfg Config) error {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port <= 0 {
		cfg.Port = 8080
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind to %s: %w", addr, err)
	}

	handler := RegisterRoutes()
	httpServer := &http.Server{
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	url := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	if cfg.Host == "0.0.0.0" {
		url = fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)
	}

	fmt.Println()
	fmt.Println("  ╭─────────────────────────────────────────────────────────────╮")
	fmt.Println("  │                                                             │")
	fmt.Println("  │   📦 KB Web Fuzzy Finder & Live Preview                     │")
	fmt.Printf("  │   🌐 Running at: %-42s │\n", url)
	fmt.Println("  │   ⚡ Press Ctrl+C to stop the server                        │")
	fmt.Println("  │                                                             │")
	fmt.Println("  ╰─────────────────────────────────────────────────────────────╯")
	fmt.Println()

	if cfg.OpenBrowser {
		go func() {
			time.Sleep(150 * time.Millisecond)
			_ = OpenInBrowser(url)
		}()
	}

	// Server error channel
	serverErr := make(chan error, 1)
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-stopSig:
		fmt.Println("\nShutting down KB Web Finder gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return httpServer.Shutdown(ctx)
	}
}

// OpenInBrowser attempts to open the specified URL in the user's default browser.
func OpenInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

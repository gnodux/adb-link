// Command adb-link runs the unified API + MCP server, the API only,
// or the MCP stdio server depending on subcommand.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gnodux/adb-link/internal/api"
	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/mcp"
	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
)

const version = "1.0.15"

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: adb-link <command>")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  run-all      Start API + MCP HTTP transport on a single server")
	fmt.Fprintln(os.Stderr, "  run-api      Start only the HTTP API")
	fmt.Fprintln(os.Stderr, "  run-mcp      Start MCP server over stdio")
	fmt.Fprintln(os.Stderr, "  update       Update to the latest release")
	fmt.Fprintln(os.Stderr, "  version      Print version")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	switch cmd {
	case "run-all":
		runAll()
	case "run-api":
		runAPI()
	case "run-mcp":
		runMCP()
	case "update":
		runUpdate(version)
	case "version":
		fmt.Println("adb-link", version)
	case "-h", "--help", "help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
}

// driver registration imports (database/sql drivers register themselves on import).
// These are referenced via blank imports in drivers.go.

func runAll() {
	fs := flag.NewFlagSet("run-all", flag.ExitOnError)
	host := fs.String("host", "", "API bind host (overrides env)")
	port := fs.Int("port", 0, "API bind port (overrides env)")
	_ = fs.Parse(os.Args[2:])

	settings := config.DefaultSettings()
	if *host != "" {
		settings.APIHost = *host
	}
	if *port != 0 {
		settings.APIPort = *port
	}

	container := services.NewContainer(settings)
	container.Start()
	defer container.Stop()

	mcpServer := mcp.NewServer("adb-link", version)
	mcp.RegisterCoreTools(mcpServer, container)
	mcp.RegisterDynamicTools(mcpServer, container)

	// Notify MCP clients when configs change (hot-reload).
	container.ConfigService.AddReloadCallback(func() {
		mcpServer.NotifyToolListChanged()
	})

	router := api.NewRouterWithMCP(container, mcpServer.HTTPHandler())
	addr := fmt.Sprintf("%s:%d", settings.APIHost, settings.APIPort)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", addr, "mcp", "/mcp")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	waitForSignal()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func runAPI() {
	fs := flag.NewFlagSet("run-api", flag.ExitOnError)
	host := fs.String("host", "", "API bind host")
	port := fs.Int("port", 0, "API bind port")
	_ = fs.Parse(os.Args[2:])

	settings := config.DefaultSettings()
	if *host != "" {
		settings.APIHost = *host
	}
	if *port != 0 {
		settings.APIPort = *port
	}

	container := services.NewContainer(settings)
	container.Start()
	defer container.Stop()

	router := api.NewRouter(container)
	addr := fmt.Sprintf("%s:%d", settings.APIHost, settings.APIPort)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		slog.Info("API server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	waitForSignal()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func runMCP() {
	settings := config.DefaultSettings()
	container := services.NewContainer(settings)
	container.Start()
	defer container.Stop()

	mcpServer := mcp.NewServer("adb-link", version)
	mcp.RegisterCoreTools(mcpServer, container)
	mcp.RegisterDynamicTools(mcpServer, container)

	// Notify MCP clients when configs change (hot-reload).
	container.ConfigService.AddReloadCallback(func() {
		mcpServer.NotifyToolListChanged()
	})

	ctx, cancel := signalContext()
	defer cancel()
	// Inject a named user for stdio transport so permission checks apply.
	ctx = models.WithAuthUser(ctx, &models.AuthUser{Name: "mcp_stdio_user"})
	if err := mcpServer.ServeStdio(ctx, os.Stdin, os.Stdout); err != nil && err != context.Canceled {
		slog.Error("mcp stdio error", "err", err)
		os.Exit(1)
	}
}

func waitForSignal() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	slog.Info("shutdown signal received, stopping...")
	// Restore default signal handling so a second Ctrl+C kills the process
	// immediately instead of being silently swallowed.
	signal.Stop(c)
	go func() {
		<-c
		slog.Warn("second signal received, forcing exit")
		os.Exit(1)
	}()
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		// Restore default signal handling so a second Ctrl+C kills the process
		// immediately instead of being silently swallowed.
		signal.Stop(c)
		go func() {
			<-c
			slog.Warn("second signal received, forcing exit")
			os.Exit(1)
		}()
	}()
	return ctx, cancel
}

// Command controlcenter is the console: a web UI that shows the Katalogs,
// CRDs and custom resources of one or more running runtimes, and lets
// developers submit intents through the gateway's generated forms. It reads
// everything over the runtime and gateway HTTP APIs and has no Kubernetes
// access of its own.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cc "github.com/orkspace/orkestra-cc/cc"
	v "github.com/orkspace/orkestra-cc/version"
)

var (
	version   = v.Version
	commit    = v.Commit
	buildDate = v.Date
)

func printVersion() {
	fmt.Printf("orkcc %s (commit %s, built %s)\n",
		version, commit, buildDate)
}

func main() {
	kfg := cc.NewControlCenterKonfig()

	// Parse flags
	var (
		orkestraURLs  = flag.String("u", strings.Join(kfg.URLs, ","), "Comma-separated URLs of Orkestra runtime instances")
		ignoreDefault = flag.Bool("ignore-default", kfg.IgnoreDefault, "Do not add the default localhost:8080 URL; start with no instances")
		port          = flag.String("p", kfg.Port, "Port to serve the control center on")
		refresh       = flag.Duration("refresh", kfg.RefreshInterval, "Refresh interval for fetching Katalogs")
		logLevel      = flag.String("log-level", kfg.LogLevel, "Log level (debug, info, warn, error)")
		showVersion   = flag.Bool("version", false, "Show version information")
	)

	flag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	// Parse and deduplicate URLs
	urls := parseAndDedupeURLs(*orkestraURLs)
	if len(urls) == 0 && !*ignoreDefault {
		log.Fatal("ERROR: at least one Orkestra URL is required. Use -u flag or --ignore-default.")
	}

	if len(urls) == 0 {
		urls = append(urls, "http://localhost:8080")
	}

	// Create control center
	cc := cc.New(urls, cc.Config{
		RefreshInterval:      *refresh,
		LogLevel:             *logLevel,
		Version:              version,
		EnableRuntimeManager: kfg.EnableRuntimeManager,
		NoLogin:              kfg.NoLogin,
		GatewayToken:         kfg.GatewayToken,
	})

	// Setup routes
	mux := setupRoutes(cc, version, commit, buildDate)

	// Setup HTTP server
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("🚀 Orkestra Control Center starting on :%s", *port)
		log.Printf("Watching instances: %v", urls)
		log.Printf("Refresh interval: %v", *refresh)
		log.Printf("Control Center URL: http://localhost:%s/controlcenter", *port)
		log.Printf("Health endpoint: http://localhost:%s/controlcenter/health", *port)
		log.Printf("Ready endpoint: http://localhost:%s/controlcenter/ready", *port)
		log.Printf("Version endpoint: http://localhost:%s/controlcenter/version", *port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("🛑 Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ Shutdown error: %v", err)
	}

	log.Println("✅ Control Center stopped")
}

// parseAndDedupeURLs parses a comma-separated URL string and removes duplicates
func parseAndDedupeURLs(input string) []string {
	parts := strings.Split(input, ",")
	seen := make(map[string]bool)
	urls := make([]string, 0, len(parts))

	for _, p := range parts {
		url := strings.TrimSpace(p)
		if url == "" {
			continue
		}

		// Add scheme if missing
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + url
		}

		// Remove trailing slash
		url = strings.TrimSuffix(url, "/")

		// Deduplicate
		if !seen[url] {
			seen[url] = true
			urls = append(urls, url)
		}
	}

	return urls
}

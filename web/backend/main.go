// PicoClaw Web Console - Web-based chat and management interface
//
// Provides a web UI for chatting with PicoClaw via the Pico Channel WebSocket,
// with configuration management and gateway process control.
//
// Usage:
//
//	go build -o picoclaw-web ./web/backend/
//	./picoclaw-web [config.json]
//	./picoclaw-web -public config.json

package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sipeed/picoclaw/web/backend/api"
	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
	"github.com/sipeed/picoclaw/web/backend/middleware"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

func main() {
	port := flag.String("port", "18800", "Port to listen on")
	public := flag.Bool("public", false, "Listen on all interfaces (0.0.0.0) instead of localhost only")
	noBrowser := flag.Bool("no-browser", false, "Do not auto-open browser on startup")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "PicoClaw Launcher - A web-based configuration editor\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [config.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  config.json    Path to the configuration file (default: ~/.picoclaw/config.json)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                          Use default config path\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./config.json             Specify a config file\n", os.Args[0])
		fmt.Fprintf(
			os.Stderr,
			"  %s -public ./config.json     Allow access from other devices on the network\n",
			os.Args[0],
		)
	}
	flag.Parse()

	// Resolve config path
	configPath := utils.GetDefaultConfigPath()
	if flag.NArg() > 0 {
		configPath = flag.Arg(0)
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		log.Fatalf("Failed to resolve config path: %v", err)
	}
	err = utils.EnsureOnboarded(absPath)
	if err != nil {
		log.Printf("Warning: Failed to initialize PicoClaw config automatically: %v", err)
	}

	var explicitPort bool
	var explicitPublic bool
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "port":
			explicitPort = true
		case "public":
			explicitPublic = true
		}
	})

	launcherPath := launcherconfig.PathForAppConfig(absPath)
	launcherCfg, err := launcherconfig.Load(launcherPath, launcherconfig.Default())
	if err != nil {
		log.Printf("Warning: Failed to load %s: %v", launcherPath, err)
		launcherCfg = launcherconfig.Default()
	}

	// Bootstrap auth credentials if needed when running in public mode
	if *public || launcherCfg.Public {
		if pw, changed := launcherconfig.EnsureAuthBootstrapped(&launcherCfg); changed {
			if err := launcherconfig.Save(launcherPath, launcherCfg); err != nil {
				log.Printf("Warning: Failed to save bootstrapped auth config: %v", err)
			}
			if pw != "" {
				fmt.Println()
				fmt.Println("  ============================================")
				fmt.Println("  Authentication has been enabled automatically")
				fmt.Printf("  Username: %s\n", launcherCfg.AuthUsername)
				fmt.Printf("  Password: %s\n", pw)
				fmt.Println("  ============================================")
				fmt.Println()
			}
		}
	}

	effectivePort := *port
	effectivePublic := *public
	if !explicitPort {
		effectivePort = strconv.Itoa(launcherCfg.Port)
	}
	if !explicitPublic {
		effectivePublic = launcherCfg.Public
	}

	portNum, err := strconv.Atoi(effectivePort)
	if err != nil || portNum < 1 || portNum > 65535 {
		if err == nil {
			err = errors.New("must be in range 1-65535")
		}
		log.Fatalf("Invalid port %q: %v", effectivePort, err)
	}

	// Determine listen address
	var addr string
	if effectivePublic {
		addr = "0.0.0.0:" + effectivePort
	} else {
		addr = "127.0.0.1:" + effectivePort
	}

	// Initialize Server components
	mux := http.NewServeMux()

	// API Routes (e.g. /api/status)
	apiHandler := api.NewHandler(absPath)
	apiHandler.SetServerOptions(portNum, effectivePublic, explicitPublic, launcherCfg.AllowedCIDRs)

	// Set up auth cookie secret
	if launcherCfg.AuthCookieSecret != "" {
		secret, err := hex.DecodeString(launcherCfg.AuthCookieSecret)
		if err != nil {
			log.Printf("Warning: Invalid cookie secret: %v", err)
		} else {
			apiHandler.SetCookieSecret(secret)
		}
	}

	apiHandler.RegisterRoutes(mux)

	// Frontend Embedded Assets
	registerEmbedRoutes(mux)

	accessControlledMux, err := middleware.IPAllowlist(launcherCfg.AllowedCIDRs, mux)
	if err != nil {
		log.Fatalf("Invalid allowed CIDR configuration: %v", err)
	}

	// Optionally wrap with auth middleware
	var authedMux http.Handler = accessControlledMux
	if launcherCfg.AuthEnabled && launcherCfg.AuthCookieSecret != "" {
		secret, err := hex.DecodeString(launcherCfg.AuthCookieSecret)
		if err != nil {
			log.Fatalf("Invalid auth cookie secret: %v", err)
		}
		authedMux = middleware.SessionAuth(secret, accessControlledMux)
	}

	// Apply middleware stack
	handler := middleware.Recoverer(
		middleware.Logger(
			middleware.JSONContentType(authedMux),
		),
	)

	// Print startup banner
	fmt.Print(utils.Banner)
	fmt.Println()
	fmt.Println("  Open the following URL in your browser:")
	fmt.Println()
	fmt.Printf("    >> http://localhost:%s <<\n", effectivePort)
	if effectivePublic {
		if ip := utils.GetLocalIP(); ip != "" {
			fmt.Printf("    >> http://%s:%s <<\n", ip, effectivePort)
		}
	}
	fmt.Println()

	// Auto-open browser
	if !*noBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			url := "http://localhost:" + effectivePort
			if err := utils.OpenBrowser(url); err != nil {
				log.Printf("Warning: Failed to auto-open browser: %v", err)
			}
		}()
	}

	// Auto-start gateway after backend starts listening.
	go func() {
		time.Sleep(1 * time.Second)
		apiHandler.TryAutoStartGateway()
	}()

	// Start the Server
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

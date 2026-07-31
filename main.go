package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	serverFlag := flag.Bool("server", false, "Run in Headless Web Server mode without GUI")
	portFlag := flag.Int("port", 8080, "Port for Headless Web Server")
	hostFlag := flag.String("host", "0.0.0.0", "Host/IP to bind Headless Web Server")
	apiKeyFlag := flag.String("api-key", "", "API key for authenticating HTTP requests in server mode")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *versionFlag {
		fmt.Println("Awd DriveRouter v1.2.1 (Dual-Mode: Desktop GUI & Headless Web Server)")
		os.Exit(0)
	}

	serverMode := *serverFlag || os.Getenv("SERVER_MODE") == "true" || os.Getenv("SERVER_MODE") == "1"
	port := *portFlag
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}
	host := *hostFlag
	if envHost := os.Getenv("HOST"); envHost != "" {
		host = envHost
	}
	apiKey := *apiKeyFlag
	if envKey := os.Getenv("API_KEY"); envKey != "" {
		apiKey = envKey
	}

	app := NewApp()

	if serverMode {
		hs := NewHeadlessServer(app, host, port, apiKey)
		if err := hs.Start(); err != nil {
			log.Fatalf("Headless server failed: %v", err)
		}
		return
	}

	// Create application with options (Desktop GUI mode)
	err := wails.Run(&options.App{
		Title:  "Awd DriveRouter",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.BeforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatalf("Application failed to start: %v", err)
	}
}


package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to config file")
	
	var runCmd bool
	flag.BoolVar(&runCmd, "run", false, "Run command (ignored, for compatibility)")
	
	flag.Parse()

	// Handle positional args for compatibility with: xray-v2root.exe run -config config.json
	args := flag.Args()
	if len(args) >= 2 && args[0] == "run" {
		for i, arg := range args {
			if arg == "-config" && i+1 < len(args) {
				configPath = args[i+1]
				break
			}
		}
	}

	if configPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: xray-tun -config <path> or xray-tun run -config <path>")
		os.Exit(1)
	}

	// Load config
	configFiles := cmdarg.Arg{configPath}
	config, err := core.LoadConfig("json", configFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create server
	server, err := core.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Start server
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Xray server started with TUN support")
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for interrupt
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("\nStopping server...")
	server.Close()
}

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bawadev/spa-server/internal/config"
	"github.com/bawadev/spa-server/internal/server"
	"github.com/urfave/cli/v2"
)

var version = "dev"

func main() {
	app := &cli.App{
		Name:    "spa-server",
		Usage:   "Lightning-fast static file server for Single Page Applications",
		Version: version,
		Flags:   config.Flags,
		Action: func(c *cli.Context) error {
			cfg, err := config.FromContext(c)
			if err != nil {
				return fmt.Errorf("failed to parse config: %w", err)
			}

			srv := server.New(cfg)

			// Handle graceful shutdown
			done := make(chan os.Signal, 1)
			signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-done
				fmt.Println("\nShutting down gracefully...")
				if err := srv.Shutdown(); err != nil {
					fmt.Printf("Error during shutdown: %v\n", err)
				}
			}()

			// Pre-compress files in background
			go srv.CompressFiles()

			// Start server (blocking)
			return srv.Listen()
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

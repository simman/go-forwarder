package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/simman/go-forwarder/internal/config"
	"github.com/simman/go-forwarder/internal/server"
	"github.com/simman/go-forwarder/pkg/logger"
)

var (
	configPath = flag.String("config", "configs/config.yaml", "Path to configuration file")
	version    = flag.Bool("version", false, "Print version information")
)

const (
	appVersion = "1.0.0"
	appName    = "go-forwarder"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log.Info().
		Str("version", appVersion).
		Str("config", *configPath).
		Msg("starting go-forwarder")

	// Create server
	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
	}

	// Start server
	if err := srv.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}

	// Setup config watcher for hot-reload
	watcher, err := config.NewWatcher(*configPath, func(newCfg *config.Config) error {
		log.Info().Msg("config changed, reloading")
		
		// Reinitialize logger if logging config changed
		if cfg.Logging != newCfg.Logging {
			if err := logger.InitLogger(newCfg.Logging.Level, newCfg.Logging.Format, newCfg.Logging.Output); err != nil {
				return fmt.Errorf("failed to reinitialize logger: %w", err)
			}
		}
		
		// Reload server configuration
		if err := srv.Reload(newCfg); err != nil {
			return fmt.Errorf("failed to reload server: %w", err)
		}
		
		cfg = newCfg
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create config watcher")
	}

	if err := watcher.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start config watcher")
	}
	defer watcher.Stop()

	log.Info().Msg("go-forwarder is ready")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	sig := <-sigCh
	log.Info().Str("signal", sig.String()).Msg("received shutdown signal")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Error().Err(err).Msg("error during shutdown")
		os.Exit(1)
	}

	log.Info().Msg("go-forwarder stopped gracefully")
}

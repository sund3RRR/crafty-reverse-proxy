// Package app implements the main application logic.
package app

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/sund3RRR/crafty-reverse-proxy/config"
	"github.com/sund3RRR/crafty-reverse-proxy/internal/adapters/crafty"
	"github.com/sund3RRR/crafty-reverse-proxy/internal/modules/connector"
	"github.com/sund3RRR/crafty-reverse-proxy/internal/modules/mc_operator"
	"github.com/sund3RRR/crafty-reverse-proxy/internal/modules/proxy"
	"github.com/sund3RRR/crafty-reverse-proxy/pkg/logger"
)

const (
	// startUpTimeout is the maximum time the application will wait for the Minecraft server to start.
	startUpTimeout = 2 * time.Minute
	// dialTimeout is the timeout for establishing connections to the Minecraft server.
	dialTimeout = 3 * time.Minute
)

// App represents the main application, which handles the setup of multiple proxy servers.
type App struct {
	cfg    config.Config  // Configuration for the application.
	logger *logger.Logger // Logger used to log application events.
	crafty *crafty.Crafty // Crafty instance for interacting with the Minecraft server.
}

// New creates and returns a new instance of the App.
func New(cfg config.Config, logger *logger.Logger, crafty *crafty.Crafty) *App {
	return &App{
		cfg:    cfg,
		logger: logger,
		crafty: crafty,
	}
}

// Run starts the application and begins proxying Minecraft server traffic.
//
// The app will start multiple proxy servers based on the provided configuration,
// with each server handling connections from clients and proxying them to the Minecraft server.
func (app *App) Run(ctx context.Context) {
	var wg sync.WaitGroup

	// Disable TLS verification for the HTTP client used to communicate with Crafty (for insecure environments).
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint

	// For each address in the configuration, create and start a new proxy server.
	for _, address := range app.cfg.Addresses {
		wg.Add(1)
		go func(serverConfig config.ServerType) {
			defer wg.Done()

			// Create a new Minecraft operator with the given server configuration.
			mcOperator := mc_operator.New(
				serverConfig,
				startUpTimeout,
				app.cfg.Timeout,
				app.logger,
				app.crafty,
			)

			// Create a new connector responsible for managing connections to the Minecraft server.
			connector := connector.New(app.logger, app.cfg.AutoShutdown, mcOperator, dialTimeout)

			// Create a new proxy server and start it.
			server := proxy.New(serverConfig, app.logger, connector)
			if err := server.ListenAndProxy(ctx); err != nil {
				// If an error occurs while starting the proxy server, log and terminate.
				log.Fatal(err)
			}
		}(address)
	}

	// Wait for all proxy servers to finish before exiting the app.
	wg.Wait()
}

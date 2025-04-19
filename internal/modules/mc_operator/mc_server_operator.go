// Package mc_operator provides the main logic for managing the lifecycle of a Minecraft server instance.
package mc_operator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/sund3RRR/crafty-reverse-proxy/config"
)

const dialTimeout = 1 * time.Second

var (
	// ErrTimeoutReached is returned when the server fails to start within the given timeout.
	ErrTimeoutReached = errors.New("timeout reached")
)

// Logger defines the logging interface used by ServerOperator.
type Logger interface {
	Debug(format string, args ...any)
	Warn(format string, args ...any)
	Info(format string, args ...any)
	Error(format string, args ...any)
}

// Crafty defines the interface for controlling Minecraft servers via the Crafty API.
type Crafty interface {
	StartMcServer(port int) error
	StopMcServer(port int) error
}

// ServerOperator manages the lifecycle of a Minecraft server instance.
type ServerOperator struct {
	targetPort      int
	targetAddress   string
	protocol        string
	startUpTimeout  time.Duration
	shutDownTimeout time.Duration

	logger        Logger
	crafty        Crafty
	shutDownTimer *time.Timer
}

// New creates and returns a new ServerOperator instance based on the provided configuration.
func New(cfg config.ServerType, startUpTimeout, shutDownTimeout time.Duration, logger Logger, crafty Crafty) *ServerOperator {
	return &ServerOperator{
		targetPort:      cfg.CraftyHost.Port,
		targetAddress:   fmt.Sprintf("%s:%d", cfg.CraftyHost.Addr, cfg.CraftyHost.Port),
		protocol:        cfg.Protocol,
		startUpTimeout:  startUpTimeout,
		shutDownTimeout: shutDownTimeout,
		logger:          logger,
		crafty:          crafty,
		shutDownTimer:   nil,
	}
}

// StartMinecraftServer starts the Minecraft server if it's not already running.
func (so *ServerOperator) StartMinecraftServer() error {
	so.logger.Info("Server is not running. Starting server with port %d", so.targetPort)
	return so.crafty.StartMcServer(so.targetPort)
}

// IsServerRunning checks whether the Minecraft server is currently accepting connections.
func (so *ServerOperator) IsServerRunning() bool {
	serverConnection, err := net.DialTimeout(so.protocol, so.targetAddress, dialTimeout)
	if err != nil {
		return false
	}
	serverConnection.Close()
	return true
}

// ConnectToServer attempts to establish a network connection to the server.
func (so *ServerOperator) ConnectToServer() (net.Conn, error) {
	return net.DialTimeout(so.protocol, so.targetAddress, dialTimeout)
}

// AwaitForServerStart waits for the server to start up and accept connections within a timeout.
func (so *ServerOperator) AwaitForServerStart(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, so.startUpTimeout)
	defer cancel()

	const cooldown = time.Second
	ticker := time.NewTicker(cooldown)
	defer ticker.Stop()

	attempt := 1
	so.logger.Info("Waiting for server :%d to start...", so.targetPort)

	for {
		select {
		case <-ctx.Done():
			return ErrTimeoutReached
		case <-ticker.C:
			so.logger.Debug("Attempt %d: connecting to %s (%s)", attempt, so.targetAddress, so.protocol)
			conn, err := net.DialTimeout(so.protocol, so.targetAddress, dialTimeout)
			if err != nil {
				so.logger.Warn("Connection attempt %d failed: %v", attempt, err)
				attempt++
				continue
			}
			conn.Close()
			so.logger.Info("Server %s is up! Connected on attempt %d", so.targetAddress, attempt)
			return nil
		}
	}
}

// ScheduleShutdown sets a timer to shut down the server after a period of inactivity.
func (so *ServerOperator) ScheduleShutdown() {
	so.logger.Info("No players left, scheduling MC server shutdown with port %d and timeout %s", so.targetPort, so.shutDownTimeout.String())
	so.shutDownTimer = time.AfterFunc(so.shutDownTimeout, func() {
		so.logger.Info("No players left, shutting down MC server with port %d", so.targetPort)
		if err := so.crafty.StopMcServer(so.targetPort); err != nil {
			so.logger.Error("Failed to stop MC server: %v", err)
		}
	})
}

// StopShuttingDown cancels a scheduled shutdown if the server becomes active again.
func (so *ServerOperator) StopShuttingDown() {
	if so.shutDownTimer != nil {
		so.shutDownTimer.Stop()
		so.shutDownTimer = nil
	}
}

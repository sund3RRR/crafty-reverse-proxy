package proxy

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sund3RRR/crafty-reverse-proxy/config"
)

type ServerOperator struct {
	targetPort      int
	targetAddress   string
	protocol        string
	shutDownTimeout time.Duration

	logger        Logger
	crafty        Crafty
	shutDownTimer *time.Timer
}

func NewServerOperator(cfg config.ServerType, shutdownTimeout time.Duration, logger Logger, crafty Crafty) *ServerOperator {
	return &ServerOperator{
		targetPort:      cfg.CraftyHost.Port,
		targetAddress:   fmt.Sprintf("%s:%d", cfg.CraftyHost.Addr, cfg.CraftyHost.Port),
		protocol:        cfg.Protocol,
		shutDownTimeout: shutdownTimeout,
		logger:          logger,
		crafty:          crafty,
		shutDownTimer:   nil,
	}
}

func (so *ServerOperator) StartMinecraftServer() error {
	so.logger.Info("Server is not running. Starting server with port %d", so.targetPort)
	err := so.crafty.StartMcServer(so.targetPort)
	if err != nil {
		return err
	}

	return nil
}

func (so *ServerOperator) IsServerRunning() bool {
	serverConnection, err := net.DialTimeout(so.protocol, so.targetAddress, dialTimeout)
	if err != nil {
		return false
	}
	serverConnection.Close()
	return true
}

func (so *ServerOperator) ConnectToServer() (net.Conn, error) {
	return net.DialTimeout(so.protocol, so.targetAddress, dialTimeout)
}

func (so *ServerOperator) AwaitForServerStart(ctx context.Context, timeout, cooldown time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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

func (so *ServerOperator) ScheduleShutdown() {
	so.logger.Info("No players left, scheduling MC server shutdown with port %d and timeout %s", so.targetPort, so.shutDownTimeout.String())
	so.shutDownTimer = time.AfterFunc(so.shutDownTimeout, func() {
		so.logger.Info("No players left, shutting down MC server with port %d", so.targetPort)
		if err := so.crafty.StopMcServer(so.targetPort); err != nil {
			so.logger.Error("Failed to stop MC server: %v", err)
			return
		}
	})
}

func (so *ServerOperator) StopShuttingDown() {
	if so.shutDownTimer != nil {
		so.shutDownTimer.Stop()
		so.shutDownTimer = nil
	}
}

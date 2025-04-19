// Package proxy provides the main logic for handling proxying traffic between Minecraft clients and servers.
package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/sund3RRR/crafty-reverse-proxy/config"
)

var (
	// ErrStartingServer is returned when the proxy server fails to start.
	ErrStartingServer = errors.New("error starting server")
)

// Logger defines the logging interface used by ProxyServer.
type Logger interface {
	Debug(format string, args ...any)
	Warn(format string, args ...any)
	Info(format string, args ...any)
	Error(format string, args ...any)
}

// Connector defines the interface for managing Minecraft server connections.
type Connector interface {
	StartLoop(ctx context.Context)
	GetConnection(ctx context.Context) (net.Conn, error)
	PutConnection(ctx context.Context, conn net.Conn) error
}

// Server handles proxying traffic between Minecraft clients and servers.
type Server struct {
	listenAddr string
	targetAddr string
	protocol   string

	logger    Logger
	connector Connector
}

// New creates and returns a new ProxyServer instance based on the provided configuration.
func New(proxyCfg config.ServerType, logger Logger, connector Connector) *Server {
	ps := &Server{
		protocol:   proxyCfg.Protocol,
		listenAddr: fmt.Sprintf("%s:%d", proxyCfg.Listener.Addr, proxyCfg.Listener.Port),
		targetAddr: fmt.Sprintf("%s:%d", proxyCfg.CraftyHost.Addr, proxyCfg.CraftyHost.Port),
		logger:     logger,
		connector:  connector,
	}
	return ps
}

// ListenAndProxy starts the proxy server, listens for incoming client connections,
// and forwards traffic to and from the Minecraft server.
func (ps *Server) ListenAndProxy(ctx context.Context) error {
	ps.connector.StartLoop(ctx)

	listener, err := net.Listen(ps.protocol, ps.listenAddr)
	if err != nil {
		return fmt.Errorf("%w with protocol %s, err: %w", ErrStartingServer, ps.protocol, err)
	}
	defer func() {
		listener.Close()
		ps.logger.Info("Listener closed for external address: %s", ps.targetAddr)
	}()

	ps.logger.Info("%s: reverse proxy running on %s, forwarding to %s", ps.protocol, ps.listenAddr, ps.targetAddr)

	for {
		client, err := listener.Accept()
		if err != nil {
			ps.logger.Error("Failed to accept connection: %v", err)
			continue
		}

		go func() {
			if err := ps.handleClient(ctx, client); err != nil {
				ps.logger.Error("Failed to handle client: %v", err)
			}
		}()
	}
}

// handleClient proxies data between the connected Minecraft client and server.
func (ps *Server) handleClient(ctx context.Context, client net.Conn) error {
	defer client.Close()

	serverConnection, err := ps.connector.GetConnection(ctx)
	defer func() {
		err := ps.connector.PutConnection(ctx, serverConnection)
		if err != nil {
			ps.logger.Error("Failed to put connection: %v", err)
		}
	}()
	if err != nil {
		return err
	}

	ps.logger.Info("Starting proxy from %s to %s", client.RemoteAddr(), serverConnection.RemoteAddr())

	completed := make(chan struct{})
	go func() {
		defer func() {
			completed <- struct{}{}
			close(completed)
		}()
		_, err := io.Copy(client, serverConnection)
		if err != nil {
			ps.logger.Warn("An error occurred copying from server to client: %v", err)
		}
		ps.logger.Info("Proxying from %s to %s completed", client.RemoteAddr(), serverConnection.RemoteAddr())
	}()

	_, err = io.Copy(serverConnection, client)
	if err != nil {
		ps.logger.Error("Error copying from client to server: %s", err)
	}

	<-completed

	return nil
}
